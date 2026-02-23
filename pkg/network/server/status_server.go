package server

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lulaide/gomc/pkg/network/crypt"
	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/util"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

// StatusConfig configures a minimal 1.6.4 status/login gateway.
type StatusConfig struct {
	ListenAddress string
	MOTD          string
	MaxPlayers    int
	VersionName   string
	OnlineMode    bool
	ViewDistance  int
	PersistWorld  bool
	WorldDir      string
}

// StatusServer handles status ping + early login protocol checks.
type StatusServer struct {
	cfg            StatusConfig
	currentPlayers atomic.Int32
	nextEntityID   atomic.Int32
	worldAge       atomic.Int64
	worldTime      atomic.Int64

	privateKey   *rsa.PrivateKey
	publicKeyDER []byte

	world *flatWorld
	// Translation reference:
	// - net.minecraft.src.BlockFluid#tickRate(World) => water updates every 5 ticks.
	waterTickAccum int64

	activeMu      sync.RWMutex
	activePlayers map[*loginSession]string
	activeOrder   []*loginSession
	playerPingIdx int

	projectileMu sync.Mutex
	projectiles  map[int32]*trackedProjectile

	droppedItemMu sync.Mutex
	droppedItems  map[int32]*trackedDroppedItem

	mobMu   sync.Mutex
	mobs    map[int32]*trackedMob
	mobRand *util.JavaRandom
	now     func() time.Time

	listenMu sync.Mutex
	listener net.Listener
}

type activePlayerRef struct {
	Session  *loginSession
	Username string
}

func NewStatusServer(cfg StatusConfig) *StatusServer {
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = ":25565"
	}
	if cfg.MOTD == "" {
		cfg.MOTD = "GoMC 1.6.4 rewrite"
	}
	if cfg.MaxPlayers <= 0 {
		cfg.MaxPlayers = 20
	}
	if cfg.VersionName == "" {
		cfg.VersionName = "1.6.4"
	}
	if cfg.ViewDistance < 3 || cfg.ViewDistance > 15 {
		cfg.ViewDistance = 10
	}

	privateKey, err := crypt.CreateNewKeyPair()
	if err != nil {
		panic(err)
	}
	pubDERAny, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		panic(err)
	}

	s := &StatusServer{
		cfg:           cfg,
		privateKey:    privateKey,
		publicKeyDER:  pubDERAny,
		world:         newFlatWorld(),
		activePlayers: make(map[*loginSession]string),
		activeOrder:   make([]*loginSession, 0, cfg.MaxPlayers),
		projectiles:   make(map[int32]*trackedProjectile),
		droppedItems:  make(map[int32]*trackedDroppedItem),
		mobs:          make(map[int32]*trackedMob),
		now:           time.Now,
	}
	if cfg.PersistWorld {
		worldDir := cfg.WorldDir
		if worldDir == "" {
			worldDir = "world"
		}
		s.world = newFlatWorldWithStorage(worldDir)
	}
	s.mobRand = util.NewJavaRandom(s.world.seed ^ 0x4F4D435F4D4F42) // "OMC_MOB"
	s.nextEntityID.Store(1)
	return s
}

// ListenAndServe starts TCP listener and blocks.
func (s *StatusServer) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddress)
	if err != nil {
		return err
	}
	s.listenMu.Lock()
	s.listener = ln
	s.listenMu.Unlock()
	defer func() {
		s.listenMu.Lock()
		if s.listener == ln {
			s.listener = nil
		}
		s.listenMu.Unlock()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if isClosedListenerError(err) {
				return nil
			}
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *StatusServer) CurrentPlayers() int {
	return int(s.currentPlayers.Load())
}

func (s *StatusServer) SetCurrentPlayers(v int) {
	s.currentPlayers.Store(int32(v))
}

func (s *StatusServer) AdvanceWorldTime(ticks int64) {
	if ticks <= 0 {
		return
	}
	s.worldAge.Add(ticks)
	s.worldTime.Add(ticks)
	s.waterTickAccum += ticks
	for s.waterTickAccum >= waterTickRate {
		s.tickWaterFlow()
		s.waterTickAccum -= waterTickRate
	}
}

func (s *StatusServer) CurrentWorldTime() (int64, int64) {
	return s.worldAge.Load(), s.worldTime.Load()
}

func (s *StatusServer) SetWorldTime(timeOfDay int64) {
	s.worldTime.Store(timeOfDay)
}

func (s *StatusServer) TickChunkInhabitedTime() {
	players := s.activeSpawnPlayers()
	if len(players) == 0 {
		return
	}

	seen := make(map[chunk.CoordIntPair]struct{}, len(players))
	for _, pl := range players {
		seen[chunk.NewCoordIntPair(pl.chunkX, pl.chunkZ)] = struct{}{}
	}
	for pos := range seen {
		ch := s.world.getChunk(pos.ChunkXPos, pos.ChunkZPos)
		if ch == nil {
			continue
		}
		ch.InhabitedTime++
	}
}

func (s *StatusServer) SaveWorldDirty() error {
	return s.world.saveDirty(s.worldAge.Load())
}

func (s *StatusServer) SaveWorldAll() error {
	return s.world.saveAll(s.worldAge.Load())
}

func (s *StatusServer) Close() error {
	s.listenMu.Lock()
	ln := s.listener
	s.listener = nil
	s.listenMu.Unlock()
	if ln != nil {
		_ = ln.Close()
	}

	if err := s.SaveWorldAll(); err != nil {
		return err
	}
	return s.world.closeStorage()
}

func isClosedListenerError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}

func (s *StatusServer) handleConn(conn net.Conn) {
	defer conn.Close()
	session := newLoginSession(s, conn)
	session.run()
}

func (s *StatusServer) registerActivePlayer(session *loginSession, username string) []activePlayerRef {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()

	existing := make([]activePlayerRef, 0, len(s.activePlayers))
	for existingSession, name := range s.activePlayers {
		existing = append(existing, activePlayerRef{
			Session:  existingSession,
			Username: name,
		})
	}
	s.activePlayers[session] = username
	s.activeOrder = append(s.activeOrder, session)
	return existing
}

func (s *StatusServer) unregisterActivePlayer(session *loginSession) (string, bool) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()

	name, ok := s.activePlayers[session]
	if ok {
		delete(s.activePlayers, session)
		for otherSession := range s.activePlayers {
			otherSession.markSeenBy(session, false)
		}
		for i, existing := range s.activeOrder {
			if existing == session {
				s.activeOrder = append(s.activeOrder[:i], s.activeOrder[i+1:]...)
				break
			}
		}
		if s.playerPingIdx > len(s.activeOrder) {
			s.playerPingIdx = 0
		}
	}
	return name, ok
}

func (s *StatusServer) broadcastPacket(packet protocol.Packet) {
	s.broadcastPacketExcept(packet, nil)
}

func (s *StatusServer) broadcastPacketExcept(packet protocol.Packet, except *loginSession) {
	s.activeMu.RLock()
	targets := make([]*loginSession, 0, len(s.activePlayers))
	for session := range s.activePlayers {
		if session == except {
			continue
		}
		targets = append(targets, session)
	}
	s.activeMu.RUnlock()

	for _, session := range targets {
		_ = session.sendPacket(packet)
	}
}

func (s *StatusServer) broadcastBlockChange(x, y, z int32, blockID, metadata int32) {
	s.activeMu.RLock()
	targets := make([]*loginSession, 0, len(s.activePlayers))
	for session := range s.activePlayers {
		targets = append(targets, session)
	}
	s.activeMu.RUnlock()

	chunkX := x >> 4
	chunkZ := z >> 4
	for _, session := range targets {
		if !session.isWatchingChunk(chunkX, chunkZ) {
			continue
		}
		_ = session.sendPacket(&protocol.Packet53BlockChange{
			XPosition: x,
			YPosition: y,
			ZPosition: z,
			Type:      blockID,
			Metadata:  metadata,
		})
	}
}

func (s *StatusServer) broadcastEntityPacketToWatchers(packet protocol.Packet, entityChunkX, entityChunkZ int32, except *loginSession) {
	targets := s.activeSessionsExcept(except)

	for _, session := range targets {
		if !session.isWatchingChunk(entityChunkX, entityChunkZ) {
			continue
		}
		_ = session.sendPacket(packet)
	}
}

func (s *StatusServer) activeSessionsExcept(except *loginSession) []*loginSession {
	s.activeMu.RLock()
	targets := make([]*loginSession, 0, len(s.activePlayers))
	for session := range s.activePlayers {
		if session == except {
			continue
		}
		targets = append(targets, session)
	}
	s.activeMu.RUnlock()
	return targets
}

func (s *StatusServer) activePlayerNames() []string {
	s.activeMu.RLock()
	defer s.activeMu.RUnlock()

	out := make([]string, 0, len(s.activePlayers))
	seen := make(map[string]struct{}, len(s.activePlayers))
	for _, session := range s.activeOrder {
		name := s.activePlayers[session]
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, name := range s.activePlayers {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func (s *StatusServer) activeSessionByEntityID(entityID int32) *loginSession {
	if entityID == 0 {
		return nil
	}
	s.activeMu.RLock()
	defer s.activeMu.RUnlock()
	for session := range s.activePlayers {
		if session == nil || session.entityID != entityID {
			continue
		}
		return session
	}
	return nil
}

func (s *StatusServer) activeSessionByUsername(username string) *loginSession {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}
	s.activeMu.RLock()
	defer s.activeMu.RUnlock()
	for session, name := range s.activePlayers {
		if session == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), username) {
			return session
		}
	}
	return nil
}

// TickPlayerInfo mirrors ServerConfigurationManager#sendPlayerInfoToAllPlayers cadence.
func (s *StatusServer) TickPlayerInfo() {
	s.activeMu.Lock()
	s.playerPingIdx++
	if s.playerPingIdx > 600 {
		s.playerPingIdx = 0
	}

	var (
		targetUsername string
		targetPing     int16
		shouldSend     bool
	)
	if s.playerPingIdx < len(s.activeOrder) {
		target := s.activeOrder[s.playerPingIdx]
		targetUsername = s.activePlayers[target]
		targetPing = int16(target.currentPing())
		shouldSend = targetUsername != ""
	}
	s.activeMu.Unlock()

	if shouldSend {
		s.broadcastPacket(&protocol.Packet201PlayerInfo{
			PlayerName:  targetUsername,
			IsConnected: true,
			Ping:        targetPing,
		})
	}
}

// BuildServerPingResponse translates NetLoginHandler#handleServerPing response formatting.
func BuildServerPingResponse(ping *protocol.Packet254ServerPing, motd, versionName string, currentPlayers, maxPlayers int) string {
	if ping != nil && ping.IsLegacyPing() {
		return motd + "\u00a7" + strconv.Itoa(currentPlayers) + "\u00a7" + strconv.Itoa(maxPlayers)
	}

	parts := []string{
		"\u00a71",
		strconv.Itoa(protocol.ProtocolVersion),
		versionName,
		motd,
		strconv.Itoa(currentPlayers),
		strconv.Itoa(maxPlayers),
	}

	out := ""
	for i, part := range parts {
		if i == 0 {
			out = part
		} else {
			out += "\u0000" + part
		}
	}
	return out
}

func (s *StatusServer) String() string {
	return fmt.Sprintf("status-server(addr=%s,motd=%q,max=%d)", s.cfg.ListenAddress, s.cfg.MOTD, s.cfg.MaxPlayers)
}
