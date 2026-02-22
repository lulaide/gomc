package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lulaide/gomc/pkg/network/crypt"
	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

const playerEyeHeight = 1.6200000047683716
const (
	playerWindowSlotCount = 45
	hotbarBaseWindowSlot  = 36
	hotbarCount           = 9
)

// EventType classifies session-side notifications for UI/CLI layers.
type EventType string

const (
	EventChat       EventType = "chat"
	EventKick       EventType = "kick"
	EventDisconnect EventType = "disconnect"
	EventSystem     EventType = "system"
	EventSound      EventType = "sound"
)

// Event is emitted by Session.Events().
type Event struct {
	Type        EventType
	Message     string
	SoundName   string
	SoundVolume float32
	SoundPitch  float32
}

type trackedEntity struct {
	EntityID int32
	Name     string
	Type     int8

	XPosition int32
	YPosition int32
	ZPosition int32

	Yaw     int8
	Pitch   int8
	HeadYaw int8

	MotionX float64
	MotionY float64
	MotionZ float64

	Sneaking  bool
	Sprinting bool
	UsingItem bool

	CreeperState    int8
	CreeperPowered  bool
	SlimeSize       int8
	SheepColor      int8
	SheepSheared    bool
	SpiderClimbing  bool
	SkeletonType    int8
	ZombieChild     bool
	ZombieVillager  bool
	SwingStartNanos int64

	DroppedItemID     int16
	DroppedItemCount  int8
	DroppedItemDamage int16
}

// EntitySnapshot is a read-only view of a currently tracked remote entity.
type EntitySnapshot struct {
	EntityID int32
	Name     string
	Type     int8

	X float64
	Y float64
	Z float64

	Yaw     int8
	Pitch   int8
	HeadYaw int8

	Sneaking  bool
	Sprinting bool
	UsingItem bool

	CreeperState   int8
	CreeperPowered bool
	SlimeSize      int8
	SheepColor     int8
	SheepSheared   bool
	SpiderClimbing bool
	SkeletonType   int8
	ZombieChild    bool
	ZombieVillager bool
	SwingProgress  float32

	DroppedItemID     int16
	DroppedItemCount  int8
	DroppedItemDamage int16
}

// HotbarSlotSnapshot represents one player hotbar slot (0..8).
type HotbarSlotSnapshot struct {
	ItemID     int16
	StackSize  int8
	ItemDamage int16
}

// InventorySlotSnapshot represents one player inventory slot in window 0.
type InventorySlotSnapshot struct {
	ItemID     int16
	StackSize  int8
	ItemDamage int16
}

// PlayerInfoSnapshot mirrors one tab-list entry from Packet201.
type PlayerInfoSnapshot struct {
	Name string
	Ping int16
}

// StateSnapshot mirrors live client session state for external rendering/UI loops.
type StateSnapshot struct {
	Username string
	EntityID int32

	PlayerX      float64
	PlayerY      float64
	PlayerZ      float64
	PlayerStance float64
	PlayerYaw    float32
	PlayerPitch  float32
	OnGround     bool
	HeldSlot     int16
	HeldItemID   int16
	HeldCount    int8
	HeldDamage   int16
	Hotbar       [hotbarCount]HotbarSlotSnapshot
	GameType     int8
	CanFly       bool
	IsFlying     bool
	IsCreative   bool
	Invulnerable bool

	SpawnX int32
	SpawnY int32
	SpawnZ int32

	Health         float32
	Food           int16
	FoodSaturation float32
	ExperienceBar  float32
	ExperienceLvl  int16
	ExperienceTot  int16

	WorldAge  int64
	WorldTime int64

	LoadedChunks    int
	TrackedEntities int
}

// Session translates core client play-network behavior from NetClientHandler.
type Session struct {
	conn   net.Conn
	reader io.Reader
	writer io.Writer

	username string

	writeMu sync.Mutex
	stateMu sync.RWMutex

	entityID int32

	playerX      float64
	playerY      float64
	playerZ      float64
	playerStance float64
	playerYaw    float32
	playerPitch  float32
	onGround     bool
	heldItemSlot int16
	gameType     int8
	canFly       bool
	isFlying     bool
	isCreative   bool
	invulnerable bool

	spawnX int32
	spawnY int32
	spawnZ int32

	health         float32
	food           int16
	foodSaturation float32
	expBar         float32
	expLevel       int16
	expTotal       int16
	worldAge       int64
	worldTime      int64
	hasSkyLight    bool

	world      *worldCache
	entities   map[int32]*trackedEntity
	playerInfo map[string]int16
	inventory  [playerWindowSlotCount]*protocol.ItemStack
	nextAction int16

	events chan Event
	done   chan struct{}

	closing   atomic.Bool
	closeOnce sync.Once

	errMu   sync.Mutex
	loopErr error
}

// DialAndLogin performs 1.6.4 login flow and starts packet read loop.
//
// Translation target:
// - net.minecraft.src.NetClientHandler
// - net.minecraft.src.TcpConnection (AES/CFB8 stream switch timing)
func DialAndLogin(addr, username string) (*Session, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("invalid addr %q: %w", addr, err)
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("invalid port %q: %w", port, err)
	}

	reader := io.Reader(conn)
	writer := io.Writer(conn)

	if err := protocol.WritePacket(writer, &protocol.Packet2ClientProtocol{
		ProtocolVersion: protocol.ProtocolVersion,
		Username:        username,
		ServerHost:      host,
		ServerPort:      int32(portNum),
	}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	first, err := protocol.ReadPacket(reader, protocol.DirectionClientbound)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	auth, ok := first.(*protocol.Packet253ServerAuthData)
	if !ok {
		if kick, isKick := first.(*protocol.Packet255KickDisconnect); isKick {
			_ = conn.Close()
			return nil, fmt.Errorf("kicked: %s", kick.Reason)
		}
		_ = conn.Close()
		return nil, fmt.Errorf("unexpected first packet type: %T", first)
	}

	pub, err := crypt.DecodePublicKey(auth.PublicKey)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	sharedKey, err := crypt.CreateNewSharedKey()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	encKey, err := crypt.EncryptData(pub, sharedKey)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	encToken, err := crypt.EncryptData(pub, auth.VerifyToken)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := protocol.WritePacket(writer, &protocol.Packet252SharedKey{
		SharedSecret: encKey,
		VerifyToken:  encToken,
	}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	encryptedWriter, err := crypt.EncryptOutputStream(sharedKey, conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	ack, err := protocol.ReadPacket(reader, protocol.DirectionClientbound)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, ok := ack.(*protocol.Packet252SharedKey); !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("unexpected packet after shared key: %T", ack)
	}

	decryptedReader, err := crypt.DecryptInputStream(sharedKey, conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	reader = decryptedReader
	writer = encryptedWriter

	if err := protocol.WritePacket(writer, &protocol.Packet205ClientCommand{ForceRespawn: 0}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	s := &Session{
		conn:        conn,
		reader:      reader,
		writer:      writer,
		username:    username,
		hasSkyLight: true,
		world:       newWorldCache(),
		entities:    make(map[int32]*trackedEntity),
		playerInfo:  make(map[string]int16),
		events:      make(chan Event, 256),
		done:        make(chan struct{}),
	}
	go s.readLoop()
	return s, nil
}

func (s *Session) readLoop() {
	for {
		packet, err := protocol.ReadPacket(s.reader, protocol.DirectionClientbound)
		if err != nil {
			s.finish(err)
			return
		}
		if err := s.handlePacket(packet); err != nil {
			s.finish(err)
			return
		}
	}
}

func (s *Session) finish(err error) {
	s.closeOnce.Do(func() {
		_ = s.conn.Close()

		if isCloseErr(err) && s.closing.Load() {
			err = nil
		}

		s.errMu.Lock()
		s.loopErr = err
		s.errMu.Unlock()

		if err != nil {
			s.emitEvent(Event{
				Type:    EventDisconnect,
				Message: err.Error(),
			})
		}

		close(s.done)
		close(s.events)
	})
}

func isCloseErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "closed network connection")
}

func (s *Session) emitEvent(ev Event) {
	select {
	case s.events <- ev:
	default:
	}
}

func cloneItemStack(stack *protocol.ItemStack) *protocol.ItemStack {
	if stack == nil {
		return nil
	}
	out := *stack
	return &out
}

func watchableByte(m protocol.WatchableObject) (int8, bool) {
	if m.ObjectType != 0 {
		return 0, false
	}
	v, ok := m.Value.(int8)
	if !ok {
		return 0, false
	}
	return v, true
}

func watchableItemStack(m protocol.WatchableObject) (*protocol.ItemStack, bool) {
	if m.ObjectType != 5 {
		return nil, false
	}
	v, ok := m.Value.(*protocol.ItemStack)
	if !ok {
		return nil, false
	}
	return v, true
}

func applyEntityMetadata(ent *trackedEntity, metadata []protocol.WatchableObject) {
	if ent == nil {
		return
	}
	for _, m := range metadata {
		if ent.Type == 2 && m.DataValueID == 10 {
			if stack, ok := watchableItemStack(m); ok && stack != nil {
				ent.DroppedItemID = stack.ItemID
				ent.DroppedItemCount = stack.StackSize
				ent.DroppedItemDamage = stack.ItemDamage
			} else if m.ObjectType == 5 {
				ent.DroppedItemID = 0
				ent.DroppedItemCount = 0
				ent.DroppedItemDamage = 0
			}
			continue
		}

		value, ok := watchableByte(m)
		if !ok {
			continue
		}

		if m.DataValueID == 0 {
			ent.Sneaking = (value & (1 << 1)) != 0
			ent.Sprinting = (value & (1 << 3)) != 0
			ent.UsingItem = (value & (1 << 4)) != 0
			continue
		}

		switch ent.Type {
		case 50: // creeper
			switch m.DataValueID {
			case 16:
				ent.CreeperState = value
			case 17:
				ent.CreeperPowered = value == 1
			}
		case 52: // spider
			if m.DataValueID == 16 {
				ent.SpiderClimbing = (value & 1) != 0
			}
		case 51: // skeleton
			if m.DataValueID == 13 {
				ent.SkeletonType = value
			}
		case 54: // zombie
			switch m.DataValueID {
			case 12:
				ent.ZombieChild = value == 1
			case 13:
				ent.ZombieVillager = value == 1
			}
		case 55: // slime
			if m.DataValueID == 16 {
				if value <= 0 {
					ent.SlimeSize = 1
				} else {
					ent.SlimeSize = value
				}
			}
		case 91: // sheep
			if m.DataValueID == 16 {
				ent.SheepColor = value & 15
				ent.SheepSheared = (value & 16) != 0
			}
		}
	}
}

func swingProgressFromStart(startNanos, nowNanos int64) float32 {
	if startNanos <= 0 {
		return 0
	}
	const swingDurationNanos = int64(300 * time.Millisecond)
	if nowNanos <= startNanos {
		return float32(1.0 / float64(swingDurationNanos))
	}
	elapsed := nowNanos - startNanos
	if elapsed >= swingDurationNanos {
		return 0
	}
	return float32(float64(elapsed) / float64(swingDurationNanos))
}

func (s *Session) handlePacket(packet protocol.Packet) error {
	switch p := packet.(type) {
	case *protocol.Packet0KeepAlive:
		return s.sendPacket(&protocol.Packet0KeepAlive{RandomID: p.RandomID})
	case *protocol.Packet1Login:
		s.stateMu.Lock()
		s.entityID = p.ClientEntityID
		s.gameType = p.GameType
		s.stateMu.Unlock()
	case *protocol.Packet3Chat:
		s.emitEvent(Event{
			Type:    EventChat,
			Message: p.Message,
		})
	case *protocol.Packet4UpdateTime:
		s.stateMu.Lock()
		s.worldAge = p.WorldAge
		s.worldTime = p.Time
		s.stateMu.Unlock()
	case *protocol.Packet6SpawnPosition:
		s.stateMu.Lock()
		s.spawnX = p.XPosition
		s.spawnY = p.YPosition
		s.spawnZ = p.ZPosition
		s.stateMu.Unlock()
	case *protocol.Packet8UpdateHealth:
		s.stateMu.Lock()
		s.health = p.HealthMP
		s.food = p.Food
		s.foodSaturation = p.FoodSaturation
		s.stateMu.Unlock()
	case *protocol.Packet9Respawn:
		s.stateMu.Lock()
		s.entities = make(map[int32]*trackedEntity)
		s.gameType = p.GameType
		s.stateMu.Unlock()
	case *protocol.Packet13PlayerLookMove:
		// Server->client Packet13 uses the same wire order; when correction packets are sent,
		// this rewrite follows NetServerHandler#setPlayerLocation argument ordering.
		feetY := p.YPosition
		stanceY := p.Stance
		if p.YPosition > p.Stance {
			feetY = p.Stance
			stanceY = p.YPosition
		}
		s.stateMu.Lock()
		s.playerX = p.XPosition
		s.playerY = feetY
		s.playerZ = p.ZPosition
		s.playerStance = stanceY
		s.playerYaw = p.Yaw
		s.playerPitch = p.Pitch
		s.onGround = p.OnGround
		s.stateMu.Unlock()
	case *protocol.Packet16BlockItemSwitch:
		s.stateMu.Lock()
		s.heldItemSlot = p.ID
		if s.heldItemSlot < 0 || s.heldItemSlot >= hotbarCount {
			s.heldItemSlot = 0
		}
		s.stateMu.Unlock()
	case *protocol.Packet18Animation:
		switch p.AnimateID {
		case 1:
			s.stateMu.Lock()
			if ent, ok := s.entities[p.EntityID]; ok && ent != nil {
				ent.SwingStartNanos = time.Now().UnixNano()
			}
			s.stateMu.Unlock()
			s.emitEvent(Event{
				Type:    EventSystem,
				Message: fmt.Sprintf("entity %d swing", p.EntityID),
			})
		case 5:
			s.emitEvent(Event{
				Type:    EventSystem,
				Message: fmt.Sprintf("entity %d eat", p.EntityID),
			})
		}
	case *protocol.Packet202PlayerAbilities:
		s.stateMu.Lock()
		s.invulnerable = p.DisableDamage
		s.isFlying = p.IsFlying
		s.canFly = p.AllowFlying
		s.isCreative = p.IsCreative
		s.stateMu.Unlock()
	case *protocol.Packet20NamedEntitySpawn:
		s.stateMu.Lock()
		ent := &trackedEntity{
			EntityID:  p.EntityID,
			Name:      p.Name,
			Type:      0,
			XPosition: p.XPosition,
			YPosition: p.YPosition,
			ZPosition: p.ZPosition,
			Yaw:       p.Rotation,
			Pitch:     p.Pitch,
			HeadYaw:   p.Rotation,
		}
		applyEntityMetadata(ent, p.Metadata)
		s.entities[p.EntityID] = ent
		s.stateMu.Unlock()
	case *protocol.Packet23VehicleSpawn:
		s.stateMu.Lock()
		s.entities[p.EntityID] = &trackedEntity{
			EntityID:  p.EntityID,
			Name:      fmt.Sprintf("obj:%d", p.Type),
			Type:      p.Type,
			XPosition: p.XPosition,
			YPosition: p.YPosition,
			ZPosition: p.ZPosition,
			Yaw:       p.Yaw,
			Pitch:     p.Pitch,
			HeadYaw:   p.Yaw,
			MotionX:   float64(p.SpeedX) / 8000.0,
			MotionY:   float64(p.SpeedY) / 8000.0,
			MotionZ:   float64(p.SpeedZ) / 8000.0,
		}
		s.stateMu.Unlock()
	case *protocol.Packet24MobSpawn:
		s.stateMu.Lock()
		ent := &trackedEntity{
			EntityID:  p.EntityID,
			Name:      fmt.Sprintf("mob:%d", p.Type),
			Type:      p.Type,
			XPosition: p.XPosition,
			YPosition: p.YPosition,
			ZPosition: p.ZPosition,
			Yaw:       p.Yaw,
			Pitch:     p.Pitch,
			HeadYaw:   p.HeadYaw,
			MotionX:   float64(p.VelocityX) / 8000.0,
			MotionY:   float64(p.VelocityY) / 8000.0,
			MotionZ:   float64(p.VelocityZ) / 8000.0,
		}
		applyEntityMetadata(ent, p.Metadata)
		s.entities[p.EntityID] = ent
		s.stateMu.Unlock()
	case *protocol.Packet28EntityVelocity:
		s.stateMu.Lock()
		if ent, ok := s.entities[p.EntityID]; ok {
			ent.MotionX = float64(p.MotionX) / 8000.0
			ent.MotionY = float64(p.MotionY) / 8000.0
			ent.MotionZ = float64(p.MotionZ) / 8000.0
		}
		s.stateMu.Unlock()
	case *protocol.Packet29DestroyEntity:
		s.stateMu.Lock()
		for _, id := range p.EntityIDs {
			delete(s.entities, id)
		}
		s.stateMu.Unlock()
	case *protocol.Packet31RelEntityMove:
		s.stateMu.Lock()
		if ent, ok := s.entities[p.EntityID]; ok {
			ent.XPosition += int32(p.XPosition)
			ent.YPosition += int32(p.YPosition)
			ent.ZPosition += int32(p.ZPosition)
		}
		s.stateMu.Unlock()
	case *protocol.Packet32EntityLook:
		s.stateMu.Lock()
		if ent, ok := s.entities[p.EntityID]; ok {
			ent.Yaw = p.Yaw
			ent.Pitch = p.Pitch
		}
		s.stateMu.Unlock()
	case *protocol.Packet33RelEntityMoveLook:
		s.stateMu.Lock()
		if ent, ok := s.entities[p.EntityID]; ok {
			ent.XPosition += int32(p.XPosition)
			ent.YPosition += int32(p.YPosition)
			ent.ZPosition += int32(p.ZPosition)
			ent.Yaw = p.Yaw
			ent.Pitch = p.Pitch
		}
		s.stateMu.Unlock()
	case *protocol.Packet34EntityTeleport:
		s.stateMu.Lock()
		if ent, ok := s.entities[p.EntityID]; ok {
			ent.XPosition = p.XPosition
			ent.YPosition = p.YPosition
			ent.ZPosition = p.ZPosition
			ent.Yaw = p.Yaw
			ent.Pitch = p.Pitch
		}
		s.stateMu.Unlock()
	case *protocol.Packet35EntityHeadRotation:
		s.stateMu.Lock()
		if ent, ok := s.entities[p.EntityID]; ok {
			ent.HeadYaw = p.HeadRotationYaw
		}
		s.stateMu.Unlock()
	case *protocol.Packet38EntityStatus:
		switch p.EntityStatus {
		case 2:
			s.emitEvent(Event{
				Type:    EventSystem,
				Message: fmt.Sprintf("entity %d hurt", p.EntityID),
			})
		case 3:
			s.emitEvent(Event{
				Type:    EventSystem,
				Message: fmt.Sprintf("entity %d died", p.EntityID),
			})
		case 9:
			// Translation reference:
			// - net.minecraft.src.EntityPlayerMP#onItemUseFinish()
			s.emitEvent(Event{
				Type:        EventSound,
				SoundName:   "random.burp",
				SoundVolume: 0.5,
				SoundPitch:  0.9,
			})
		}
	case *protocol.Packet40EntityMetadata:
		s.stateMu.Lock()
		if ent, ok := s.entities[p.EntityID]; ok {
			applyEntityMetadata(ent, p.Metadata)
		}
		s.stateMu.Unlock()
	case *protocol.Packet22Collect:
		s.stateMu.Lock()
		delete(s.entities, p.CollectedEntityID)
		s.stateMu.Unlock()
		s.emitEvent(Event{
			Type:        EventSound,
			SoundName:   "random.pop",
			SoundVolume: 0.2,
			SoundPitch:  1.0,
		})
	case *protocol.Packet43Experience:
		s.stateMu.Lock()
		s.expBar = p.Experience
		s.expLevel = p.ExperienceLevel
		s.expTotal = p.ExperienceTotal
		s.stateMu.Unlock()
	case *protocol.Packet103SetSlot:
		if p.WindowID == 0 && p.ItemSlot >= 0 && int(p.ItemSlot) < playerWindowSlotCount {
			s.stateMu.Lock()
			s.inventory[int(p.ItemSlot)] = cloneItemStack(p.ItemStack)
			s.stateMu.Unlock()
		}
	case *protocol.Packet104WindowItems:
		if p.WindowID == 0 {
			s.stateMu.Lock()
			for i := 0; i < playerWindowSlotCount; i++ {
				s.inventory[i] = nil
			}
			limit := len(p.ItemStacks)
			if limit > playerWindowSlotCount {
				limit = playerWindowSlotCount
			}
			for i := 0; i < limit; i++ {
				s.inventory[i] = cloneItemStack(p.ItemStacks[i])
			}
			s.stateMu.Unlock()
		}
	case *protocol.Packet51MapChunk:
		if err := s.world.applyMapChunk(p, s.hasSkyLight); err != nil {
			if retryErr := s.world.applyMapChunk(p, !s.hasSkyLight); retryErr != nil {
				return err
			}
			s.stateMu.Lock()
			s.hasSkyLight = !s.hasSkyLight
			s.stateMu.Unlock()
		}
	case *protocol.Packet53BlockChange:
		s.world.applyBlockChange(p)
	case *protocol.Packet56MapChunks:
		if err := s.world.applyMapChunks(p); err != nil {
			return err
		}
		s.stateMu.Lock()
		s.hasSkyLight = p.SkyLightSent
		s.stateMu.Unlock()
	case *protocol.Packet62LevelSound:
		s.emitEvent(Event{
			Type:        EventSound,
			SoundName:   p.SoundName,
			SoundVolume: p.Volume,
			SoundPitch:  p.PitchFloat(),
		})
	case *protocol.Packet201PlayerInfo:
		s.stateMu.Lock()
		if s.playerInfo == nil {
			s.playerInfo = make(map[string]int16)
		}
		if p.PlayerName != "" {
			if p.IsConnected {
				s.playerInfo[p.PlayerName] = p.Ping
			} else {
				delete(s.playerInfo, p.PlayerName)
			}
		}
		s.stateMu.Unlock()
		if p.PlayerName != "" {
			state := "joined"
			if !p.IsConnected {
				state = "left"
			}
			s.emitEvent(Event{
				Type:    EventSystem,
				Message: fmt.Sprintf("%s %s (ping=%d)", p.PlayerName, state, p.Ping),
			})
		}
	case *protocol.Packet255KickDisconnect:
		s.emitEvent(Event{
			Type:    EventKick,
			Message: p.Reason,
		})
		return fmt.Errorf("kicked: %s", p.Reason)
	}
	return nil
}

func (s *Session) sendPacket(packet protocol.Packet) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return protocol.WritePacket(s.writer, packet)
}

func (s *Session) Events() <-chan Event {
	return s.events
}

func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) Wait() error {
	<-s.done
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.loopErr
}

func (s *Session) Close() error {
	s.closing.Store(true)
	_ = s.conn.Close()
	<-s.done
	return nil
}

func (s *Session) Snapshot() StateSnapshot {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	heldSlot := s.heldItemSlot
	if heldSlot < 0 || heldSlot >= hotbarCount {
		heldSlot = 0
	}
	heldWindowSlot := int(hotbarBaseWindowSlot + heldSlot)
	var heldItemID int16
	var heldCount int8
	var heldDamage int16
	var hotbar [hotbarCount]HotbarSlotSnapshot
	for i := 0; i < hotbarCount; i++ {
		slot := int(hotbarBaseWindowSlot) + i
		if slot < 0 || slot >= playerWindowSlotCount {
			continue
		}
		if stack := s.inventory[slot]; stack != nil {
			hotbar[i] = HotbarSlotSnapshot{
				ItemID:     stack.ItemID,
				StackSize:  stack.StackSize,
				ItemDamage: stack.ItemDamage,
			}
		}
	}
	if heldWindowSlot >= 0 && heldWindowSlot < playerWindowSlotCount {
		if stack := s.inventory[heldWindowSlot]; stack != nil {
			heldItemID = stack.ItemID
			heldCount = stack.StackSize
			heldDamage = stack.ItemDamage
		}
	}

	return StateSnapshot{
		Username: s.username,
		EntityID: s.entityID,

		PlayerX:      s.playerX,
		PlayerY:      s.playerY,
		PlayerZ:      s.playerZ,
		PlayerStance: s.playerStance,
		PlayerYaw:    s.playerYaw,
		PlayerPitch:  s.playerPitch,
		OnGround:     s.onGround,
		HeldSlot:     heldSlot,
		HeldItemID:   heldItemID,
		HeldCount:    heldCount,
		HeldDamage:   heldDamage,
		Hotbar:       hotbar,
		GameType:     s.gameType,
		CanFly:       s.canFly,
		IsFlying:     s.isFlying,
		IsCreative:   s.isCreative,
		Invulnerable: s.invulnerable,

		SpawnX: s.spawnX,
		SpawnY: s.spawnY,
		SpawnZ: s.spawnZ,

		Health:         s.health,
		Food:           s.food,
		FoodSaturation: s.foodSaturation,
		ExperienceBar:  s.expBar,
		ExperienceLvl:  s.expLevel,
		ExperienceTot:  s.expTotal,

		WorldAge:  s.worldAge,
		WorldTime: s.worldTime,

		LoadedChunks:    s.world.chunkCount(),
		TrackedEntities: len(s.entities),
	}
}

// InventorySnapshot returns all player inventory slots for window 0 (0..44).
func (s *Session) InventorySnapshot() [playerWindowSlotCount]InventorySlotSnapshot {
	var out [playerWindowSlotCount]InventorySlotSnapshot
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	for i := 0; i < playerWindowSlotCount; i++ {
		stack := s.inventory[i]
		if stack == nil {
			continue
		}
		out[i] = InventorySlotSnapshot{
			ItemID:     stack.ItemID,
			StackSize:  stack.StackSize,
			ItemDamage: stack.ItemDamage,
		}
	}
	return out
}

// PlayerListSnapshot returns the current tab-list state sorted by case-insensitive name.
func (s *Session) PlayerListSnapshot() []PlayerInfoSnapshot {
	s.stateMu.RLock()
	out := make([]PlayerInfoSnapshot, 0, len(s.playerInfo)+1)
	for name, ping := range s.playerInfo {
		if strings.TrimSpace(name) == "" {
			continue
		}
		out = append(out, PlayerInfoSnapshot{Name: name, Ping: ping})
	}
	seenSelf := false
	for i := range out {
		if out[i].Name == s.username {
			seenSelf = true
			break
		}
	}
	if !seenSelf && strings.TrimSpace(s.username) != "" {
		out = append(out, PlayerInfoSnapshot{Name: s.username, Ping: 0})
	}
	s.stateMu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		li := strings.ToLower(out[i].Name)
		lj := strings.ToLower(out[j].Name)
		if li == lj {
			return out[i].Name < out[j].Name
		}
		return li < lj
	})
	return out
}

// EntitiesSnapshot returns a point-in-time list of tracked remote entities.
func (s *Session) EntitiesSnapshot() []EntitySnapshot {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	nowNanos := time.Now().UnixNano()
	out := make([]EntitySnapshot, 0, len(s.entities))
	for _, ent := range s.entities {
		if ent == nil {
			continue
		}
		out = append(out, EntitySnapshot{
			EntityID:  ent.EntityID,
			Name:      ent.Name,
			Type:      ent.Type,
			X:         float64(ent.XPosition) / 32.0,
			Y:         float64(ent.YPosition) / 32.0,
			Z:         float64(ent.ZPosition) / 32.0,
			Yaw:       ent.Yaw,
			Pitch:     ent.Pitch,
			HeadYaw:   ent.HeadYaw,
			Sneaking:  ent.Sneaking,
			Sprinting: ent.Sprinting,
			UsingItem: ent.UsingItem,

			CreeperState:   ent.CreeperState,
			CreeperPowered: ent.CreeperPowered,
			SlimeSize:      ent.SlimeSize,
			SheepColor:     ent.SheepColor,
			SheepSheared:   ent.SheepSheared,
			SpiderClimbing: ent.SpiderClimbing,
			SkeletonType:   ent.SkeletonType,
			ZombieChild:    ent.ZombieChild,
			ZombieVillager: ent.ZombieVillager,
			SwingProgress:  swingProgressFromStart(ent.SwingStartNanos, nowNanos),

			DroppedItemID:     ent.DroppedItemID,
			DroppedItemCount:  ent.DroppedItemCount,
			DroppedItemDamage: ent.DroppedItemDamage,
		})
	}
	return out
}

// BlockAt returns a cached block id/metadata from loaded chunk packets.
func (s *Session) BlockAt(x, y, z int) (int, int, bool) {
	return s.world.blockAt(x, y, z)
}

// ChunkRevision returns an incrementing value when a chunk cache entry changes.
func (s *Session) ChunkRevision(chunkX, chunkZ int32) uint64 {
	return s.world.chunkRevision(chunkX, chunkZ)
}

// BiomeAt returns a cached biome id from loaded chunk packet biome arrays.
func (s *Session) BiomeAt(x, z int) (int, bool) {
	return s.world.biomeAt(x, z)
}

// SendOnGround writes Packet10Flying heartbeat state.
func (s *Session) SendOnGround(onGround bool) error {
	s.stateMu.Lock()
	s.onGround = onGround
	s.stateMu.Unlock()
	return s.sendPacket(&protocol.Packet10Flying{OnGround: onGround})
}

// SetLocalOnGround updates local state used for subsequent move/look packets.
func (s *Session) SetLocalOnGround(onGround bool) {
	s.stateMu.Lock()
	s.onGround = onGround
	s.stateMu.Unlock()
}

// SelectHotbar sends Packet16 to switch currently held slot (0..8).
func (s *Session) SelectHotbar(slot int16) error {
	if slot < 0 || slot >= hotbarCount {
		return fmt.Errorf("hotbar slot out of range: %d", slot)
	}
	s.stateMu.Lock()
	s.heldItemSlot = slot
	s.stateMu.Unlock()
	return s.sendPacket(&protocol.Packet16BlockItemSwitch{ID: slot})
}

func (s *Session) nextActionNumber() int16 {
	s.stateMu.Lock()
	s.nextAction++
	v := s.nextAction
	s.stateMu.Unlock()
	return v
}

// ClickWindowSlot sends Packet102 for player inventory (window 0). Use slot -999 for outside click.
func (s *Session) ClickWindowSlot(slot int16, rightClick bool, shift bool) error {
	if slot < -999 || slot >= playerWindowSlotCount {
		return fmt.Errorf("window slot out of range: %d", slot)
	}
	mouse := int8(0)
	if rightClick {
		mouse = 1
	}
	var stack *protocol.ItemStack
	if slot >= 0 && int(slot) < playerWindowSlotCount {
		s.stateMu.RLock()
		stack = cloneItemStack(s.inventory[int(slot)])
		s.stateMu.RUnlock()
	}
	return s.sendPacket(&protocol.Packet102WindowClick{
		WindowID:      0,
		InventorySlot: slot,
		MouseClick:    mouse,
		ActionNumber:  s.nextActionNumber(),
		HoldingShift:  shift,
		ItemStack:     stack,
	})
}

// CloseInventoryWindow sends Packet101 for window 0.
func (s *Session) CloseInventoryWindow() error {
	return s.sendPacket(&protocol.Packet101CloseWindow{WindowID: 0})
}

// Look updates yaw/pitch and sends Packet12.
func (s *Session) Look(yaw, pitch float32) error {
	s.stateMu.Lock()
	s.playerYaw = yaw
	s.playerPitch = pitch
	onGround := s.onGround
	s.stateMu.Unlock()

	packet := protocol.NewPacket12PlayerLook()
	packet.Yaw = yaw
	packet.Pitch = pitch
	packet.OnGround = onGround
	return s.sendPacket(packet)
}

// MoveRelative applies a local delta and sends Packet13.
func (s *Session) MoveRelative(dx, dy, dz float64) error {
	s.stateMu.Lock()
	x := s.playerX + dx
	y := s.playerY + dy
	z := s.playerZ + dz
	yaw := s.playerYaw
	pitch := s.playerPitch
	onGround := s.onGround

	s.playerX = x
	s.playerY = y
	s.playerZ = z
	s.playerStance = y + playerEyeHeight
	s.stateMu.Unlock()

	packet := protocol.NewPacket13PlayerLookMove()
	packet.XPosition = x
	packet.YPosition = y
	packet.Stance = y + playerEyeHeight
	packet.ZPosition = z
	packet.Yaw = yaw
	packet.Pitch = pitch
	packet.OnGround = onGround
	return s.sendPacket(packet)
}

// SendChat writes Packet3Chat to server.
func (s *Session) SendChat(message string) error {
	return s.sendPacket(protocol.NewPacket3Chat(message, false))
}

// DigBlock sends finish-dig status packet.
func (s *Session) DigBlock(x, y, z, face int32) error {
	return s.sendPacket(&protocol.Packet14BlockDig{
		Status:    2,
		XPosition: x,
		YPosition: y,
		ZPosition: z,
		Face:      face,
	})
}

// DropHeldItem sends Packet14 status 4(single) or 3(full stack) like EntityClientPlayerMP.dropOneItem.
func (s *Session) DropHeldItem(fullStack bool) error {
	status := int32(4)
	if fullStack {
		status = 3
	}
	return s.sendPacket(&protocol.Packet14BlockDig{
		Status:    status,
		XPosition: 0,
		YPosition: 0,
		ZPosition: 0,
		Face:      0,
	})
}

// SetCreativeHotbarSlot writes Packet107 for one hotbar slot and updates local cache immediately.
func (s *Session) SetCreativeHotbarSlot(slot int16, itemID, itemDamage int16, count int8) error {
	if slot < 0 || slot >= hotbarCount {
		return fmt.Errorf("hotbar slot out of range: %d", slot)
	}
	windowSlot := int16(hotbarBaseWindowSlot + slot)

	var stack *protocol.ItemStack
	if itemID > 0 && count > 0 {
		stack = &protocol.ItemStack{
			ItemID:     itemID,
			StackSize:  count,
			ItemDamage: itemDamage,
		}
	}

	s.stateMu.Lock()
	s.inventory[int(windowSlot)] = cloneItemStack(stack)
	s.stateMu.Unlock()

	return s.sendPacket(&protocol.Packet107CreativeSetSlot{
		Slot:      windowSlot,
		ItemStack: cloneItemStack(stack),
	})
}

// PlaceBlock sends Packet15 with a single block item stack.
func (s *Session) PlaceBlock(x, y, z, direction int32, itemID, itemDamage int16) error {
	return s.sendPacket(&protocol.Packet15Place{
		XPosition: x,
		YPosition: y,
		ZPosition: z,
		Direction: direction,
		ItemStack: &protocol.ItemStack{
			ItemID:     itemID,
			StackSize:  1,
			ItemDamage: itemDamage,
		},
		XOffset: 0.5,
		YOffset: 0.5,
		ZOffset: 0.5,
	})
}

// PlaceHeldBlock sends Packet15 using no explicit ItemStack, relying on selected server-side slot.
func (s *Session) PlaceHeldBlock(x, y, z, direction int32) error {
	return s.sendPacket(&protocol.Packet15Place{
		XPosition: x,
		YPosition: y,
		ZPosition: z,
		Direction: direction,
		ItemStack: nil,
		XOffset:   0.5,
		YOffset:   0.5,
		ZOffset:   0.5,
	})
}

// UseHeldItemInAir sends Packet15 right-click-air action, equivalent to
// PlayerControllerMP#sendUseItem in 1.6.4.
func (s *Session) UseHeldItemInAir() error {
	s.stateMu.RLock()
	heldSlot := s.heldItemSlot
	if heldSlot < 0 || heldSlot >= hotbarCount {
		heldSlot = 0
	}
	windowSlot := int(hotbarBaseWindowSlot + heldSlot)
	var stack *protocol.ItemStack
	if windowSlot >= 0 && windowSlot < playerWindowSlotCount {
		stack = cloneItemStack(s.inventory[windowSlot])
	}
	s.stateMu.RUnlock()

	return s.sendPacket(&protocol.Packet15Place{
		XPosition: -1,
		YPosition: -1,
		ZPosition: -1,
		Direction: 255,
		ItemStack: stack,
		XOffset:   0.0,
		YOffset:   0.0,
		ZOffset:   0.0,
	})
}

// ReleaseUseItem sends Packet14 status=5, equivalent to
// PlayerControllerMP#onStoppedUsingItem in 1.6.4.
func (s *Session) ReleaseUseItem() error {
	return s.sendPacket(&protocol.Packet14BlockDig{
		Status:    5,
		XPosition: 0,
		YPosition: 0,
		ZPosition: 0,
		Face:      255,
	})
}

// SwingArm sends Packet18 animation id=1 (arm swing).
func (s *Session) SwingArm() error {
	s.stateMu.RLock()
	entityID := s.entityID
	s.stateMu.RUnlock()
	return s.sendPacket(&protocol.Packet18Animation{
		EntityID:  entityID,
		AnimateID: 1,
	})
}

// UseEntity sends Packet7 with action 0(interact) or 1(attack).
func (s *Session) UseEntity(targetEntityID int32, attack bool) error {
	if targetEntityID == 0 {
		return fmt.Errorf("target entity id cannot be 0")
	}
	action := int8(0)
	if attack {
		action = 1
	}

	s.stateMu.RLock()
	playerEntityID := s.entityID
	s.stateMu.RUnlock()
	return s.sendPacket(&protocol.Packet7UseEntity{
		PlayerEntityID: playerEntityID,
		TargetEntityID: targetEntityID,
		Action:         action,
	})
}

// SetSneaking sends Packet19 actions 1(start) / 2(stop).
func (s *Session) SetSneaking(enabled bool) error {
	action := int8(2)
	if enabled {
		action = 1
	}

	s.stateMu.RLock()
	entityID := s.entityID
	s.stateMu.RUnlock()
	return s.sendPacket(&protocol.Packet19EntityAction{
		EntityID: entityID,
		Action:   action,
		AuxData:  0,
	})
}

// SetSprinting sends Packet19 actions 4(start) / 5(stop).
func (s *Session) SetSprinting(enabled bool) error {
	action := int8(5)
	if enabled {
		action = 4
	}

	s.stateMu.RLock()
	entityID := s.entityID
	s.stateMu.RUnlock()
	return s.sendPacket(&protocol.Packet19EntityAction{
		EntityID: entityID,
		Action:   action,
		AuxData:  0,
	})
}

// SetFlying sends Packet202 ability update to toggle creative flight.
func (s *Session) SetFlying(enabled bool) error {
	s.stateMu.Lock()
	allowFlying := s.canFly || s.isCreative
	if !allowFlying {
		enabled = false
	}
	s.isFlying = enabled
	packet := &protocol.Packet202PlayerAbilities{
		DisableDamage: s.invulnerable,
		IsFlying:      enabled,
		AllowFlying:   allowFlying,
		IsCreative:    s.isCreative,
		FlySpeed:      0.05,
		WalkSpeed:     0.1,
	}
	s.stateMu.Unlock()

	return s.sendPacket(packet)
}

type worldCache struct {
	mu        sync.RWMutex
	chunks    map[chunk.CoordIntPair]*chunk.Chunk
	revisions map[chunk.CoordIntPair]uint64
}

func newWorldCache() *worldCache {
	return &worldCache{
		chunks:    make(map[chunk.CoordIntPair]*chunk.Chunk),
		revisions: make(map[chunk.CoordIntPair]uint64),
	}
}

func (w *worldCache) chunkCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.chunks)
}

func (w *worldCache) chunkRevision(chunkX, chunkZ int32) uint64 {
	key := chunk.NewCoordIntPair(chunkX, chunkZ)
	w.mu.RLock()
	rev := w.revisions[key]
	w.mu.RUnlock()
	return rev
}

func (w *worldCache) bumpRevisionLocked(chunkX, chunkZ int32) {
	key := chunk.NewCoordIntPair(chunkX, chunkZ)
	w.revisions[key] = w.revisions[key] + 1
}

func (w *worldCache) blockAt(x, y, z int) (int, int, bool) {
	if y < 0 || y >= 256 {
		return 0, 0, false
	}

	chunkX := int32(x >> 4)
	chunkZ := int32(z >> 4)
	localX := x & 15
	localZ := z & 15
	key := chunk.NewCoordIntPair(chunkX, chunkZ)

	w.mu.RLock()
	ch, ok := w.chunks[key]
	w.mu.RUnlock()
	if !ok || ch == nil {
		return 0, 0, false
	}
	return ch.GetBlockID(localX, y, localZ), ch.GetBlockMetadata(localX, y, localZ), true
}

func (w *worldCache) biomeAt(x, z int) (int, bool) {
	chunkX := int32(x >> 4)
	chunkZ := int32(z >> 4)
	localX := x & 15
	localZ := z & 15
	key := chunk.NewCoordIntPair(chunkX, chunkZ)

	w.mu.RLock()
	ch, ok := w.chunks[key]
	w.mu.RUnlock()
	if !ok || ch == nil {
		return 0, false
	}

	biomes := ch.GetBiomeArray()
	if len(biomes) < 256 {
		return 0, false
	}
	id := int(biomes[(localZ<<4)|localX])
	if id == 0xFF {
		return 0, false
	}
	return id, true
}

func (w *worldCache) applyBlockChange(packet *protocol.Packet53BlockChange) {
	if packet == nil {
		return
	}
	chunkX := int32(packet.XPosition >> 4)
	chunkZ := int32(packet.ZPosition >> 4)
	localX := int(packet.XPosition & 15)
	localZ := int(packet.ZPosition & 15)
	localY := int(packet.YPosition)
	key := chunk.NewCoordIntPair(chunkX, chunkZ)

	w.mu.Lock()
	ch, ok := w.chunks[key]
	if !ok || ch == nil {
		ch = chunk.NewChunk(nil, chunkX, chunkZ)
		w.chunks[key] = ch
	}
	_ = ch.SetBlockIDWithMetadata(localX, localY, localZ, int(packet.Type), int(packet.Metadata))
	w.bumpRevisionLocked(chunkX, chunkZ)
	// Neighbor chunk faces can change visibility when edge blocks update.
	if localX == 0 {
		w.bumpRevisionLocked(chunkX-1, chunkZ)
	}
	if localX == 15 {
		w.bumpRevisionLocked(chunkX+1, chunkZ)
	}
	if localZ == 0 {
		w.bumpRevisionLocked(chunkX, chunkZ-1)
	}
	if localZ == 15 {
		w.bumpRevisionLocked(chunkX, chunkZ+1)
	}
	w.mu.Unlock()
}

func (w *worldCache) applyMapChunk(packet *protocol.Packet51MapChunk, hasSkyLight bool) error {
	if packet == nil {
		return nil
	}
	ch, err := decodeChunkPacketData(
		packet.XCh,
		packet.ZCh,
		packet.YChMin,
		packet.YChMax,
		packet.GetCompressedChunkData(),
		packet.IncludeInitialize,
		hasSkyLight,
	)
	if err != nil {
		return err
	}

	key := chunk.NewCoordIntPair(packet.XCh, packet.ZCh)
	w.mu.Lock()
	w.chunks[key] = ch
	w.bumpRevisionLocked(packet.XCh, packet.ZCh)
	w.mu.Unlock()
	return nil
}

func (w *worldCache) applyMapChunks(packet *protocol.Packet56MapChunks) error {
	if packet == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	for i := 0; i < packet.GetNumberOfChunkInPacket(); i++ {
		ch, err := decodeChunkPacketData(
			packet.GetChunkPosX(i),
			packet.GetChunkPosZ(i),
			packet.Field73590A[i],
			packet.Field73588B[i],
			packet.GetChunkCompressedData(i),
			true,
			packet.SkyLightSent,
		)
		if err != nil {
			return err
		}
		key := chunk.NewCoordIntPair(packet.GetChunkPosX(i), packet.GetChunkPosZ(i))
		w.chunks[key] = ch
		w.bumpRevisionLocked(packet.GetChunkPosX(i), packet.GetChunkPosZ(i))
	}
	return nil
}

func decodeChunkPacketData(chunkX, chunkZ int32, existMask, addMask int32, data []byte, includeBiomes bool, hasSkyLight bool) (*chunk.Chunk, error) {
	offset := 0
	take := func(size int) ([]byte, error) {
		if size < 0 || offset+size > len(data) {
			return nil, fmt.Errorf("chunk data buffer too short: need=%d have=%d", offset+size, len(data))
		}
		out := make([]byte, size)
		copy(out, data[offset:offset+size])
		offset += size
		return out, nil
	}

	var lsbData [16][]byte
	var metaData [16][]byte
	var blockLightData [16][]byte
	var skyLightData [16][]byte
	var addData [16][]byte

	for sec := 0; sec < 16; sec++ {
		if (existMask & (1 << sec)) == 0 {
			continue
		}
		lsb, err := take(4096)
		if err != nil {
			return nil, err
		}
		lsbData[sec] = lsb
	}
	for sec := 0; sec < 16; sec++ {
		if (existMask & (1 << sec)) == 0 {
			continue
		}
		meta, err := take(2048)
		if err != nil {
			return nil, err
		}
		metaData[sec] = meta
	}
	for sec := 0; sec < 16; sec++ {
		if (existMask & (1 << sec)) == 0 {
			continue
		}
		light, err := take(2048)
		if err != nil {
			return nil, err
		}
		blockLightData[sec] = light
	}
	if hasSkyLight {
		for sec := 0; sec < 16; sec++ {
			if (existMask & (1 << sec)) == 0 {
				continue
			}
			sky, err := take(2048)
			if err != nil {
				return nil, err
			}
			skyLightData[sec] = sky
		}
	}
	for sec := 0; sec < 16; sec++ {
		if (addMask & (1 << sec)) == 0 {
			continue
		}
		add, err := take(2048)
		if err != nil {
			return nil, err
		}
		addData[sec] = add
	}

	var biomes []byte
	if includeBiomes {
		b, err := take(256)
		if err != nil {
			return nil, err
		}
		biomes = b
	}

	ch := chunk.NewChunk(nil, chunkX, chunkZ)
	storage := make([]*chunk.ExtendedBlockStorage, 16)
	for sec := 0; sec < 16; sec++ {
		if (existMask & (1 << sec)) == 0 {
			continue
		}

		ext := chunk.NewExtendedBlockStorage(sec<<4, hasSkyLight)
		ext.SetBlockLSBArray(lsbData[sec])
		ext.SetBlockMetadataArray(chunk.NewNibbleArrayFromData(metaData[sec], 4))
		ext.SetBlocklightArray(chunk.NewNibbleArrayFromData(blockLightData[sec], 4))
		if hasSkyLight {
			ext.SetSkylightArray(chunk.NewNibbleArrayFromData(skyLightData[sec], 4))
		}
		if len(addData[sec]) > 0 {
			ext.SetBlockMSBArray(chunk.NewNibbleArrayFromData(addData[sec], 4))
		}
		storage[sec] = ext
	}
	ch.SetBlockStorageArray(storage)
	if len(biomes) > 0 {
		ch.SetBiomeArray(biomes)
	}
	return ch, nil
}
