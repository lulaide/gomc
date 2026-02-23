package server

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lulaide/gomc/pkg/nbt"
	"github.com/lulaide/gomc/pkg/network/crypt"
	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

const (
	buildLimit         = 256
	maxWorldCoordinate = 3.2e7
	maxPlaceableBlock  = 255
	playerWindowSlots  = 45
	hotbarBaseSlot     = 36
	hotbarSlotCount    = 9
	defaultSpawnX      = 0.5
	defaultSpawnY      = 5.0
	defaultSpawnZ      = 0.5
	defaultPlayerEyeY  = 1.6200000047683716
	basePlayerDamage   = 1.0
	maxHurtResistant   = 20
	maxHurtTime        = 10
	maxPlayerHealth    = 20.0
	bowMaxDurability   = 384

	// Translated baseline from default 1.6.4 server state:
	// naturalRegeneration gamerule defaults true.
	naturalRegenerationGamerule = true
)

const (
	itemIDIronSword    int16 = 267
	itemIDWoodSword    int16 = 268
	itemIDStoneSword   int16 = 272
	itemIDDiamondSword int16 = 276
	itemIDGoldSword    int16 = 283

	itemIDIronShovel    int16 = 256
	itemIDIronPickaxe   int16 = 257
	itemIDIronAxe       int16 = 258
	itemIDWoodShovel    int16 = 269
	itemIDWoodPickaxe   int16 = 270
	itemIDWoodAxe       int16 = 271
	itemIDStoneShovel   int16 = 273
	itemIDStonePickaxe  int16 = 274
	itemIDStoneAxe      int16 = 275
	itemIDDiamondShovel int16 = 277
	itemIDDiamondPick   int16 = 278
	itemIDDiamondAxe    int16 = 279
	itemIDGoldShovel    int16 = 284
	itemIDGoldPickaxe   int16 = 285
	itemIDGoldAxe       int16 = 286

	itemIDBow          int16 = 261
	itemIDArrow        int16 = 262
	itemIDIronIngot    int16 = 265
	itemIDBucketEmpty  int16 = 325
	itemIDBucketMilk   int16 = 335
	itemIDSaddle       int16 = 329
	itemIDString       int16 = 287
	itemIDFeather      int16 = 288
	itemIDGunpowder    int16 = 289
	itemIDGlassBottle  int16 = 374
	itemIDPotion       int16 = 373
	itemIDShears       int16 = 359
	itemIDLeather      int16 = 334
	itemIDSlimeBall    int16 = 341
	itemIDBone         int16 = 352
	itemIDCoal         int16 = 263
	itemIDEnderPearl   int16 = 368
	itemIDBowlEmpty    int16 = 281
	itemIDMushroomStew int16 = 282

	itemIDAppleRed      int16 = 260
	itemIDBread         int16 = 297
	itemIDPorkRaw       int16 = 319
	itemIDPorkCooked    int16 = 320
	itemIDAppleGold     int16 = 322
	itemIDFishRaw       int16 = 349
	itemIDFishCooked    int16 = 350
	itemIDCookie        int16 = 357
	itemIDMelon         int16 = 360
	itemIDBeefRaw       int16 = 363
	itemIDBeefCooked    int16 = 364
	itemIDChickenRaw    int16 = 365
	itemIDChickenCooked int16 = 366
	itemIDRottenFlesh   int16 = 367
	itemIDSpiderEye     int16 = 375
	itemIDSkull         int16 = 397
	itemIDDyePowder     int16 = 351
	itemIDCarrot        int16 = 391
	itemIDPotato        int16 = 392
	itemIDBakedPotato   int16 = 393
	itemIDPoisonPotato  int16 = 394
	itemIDGoldenCarrot  int16 = 396
	itemIDPumpkinPie    int16 = 400
)

const (
	blockIDWool         int16 = 35
	shearsMaxDurability int16 = 238
)

const (
	// Translation reference:
	// - net.minecraft.src.Enchantment#looting (effectId=21)
	enchantIDLooting int = 21
)

type meleeItemProfile struct {
	AttackModifier   float32
	HitDamageCost    int16
	MaxDurability    int16
	MaxUseDuration   int
	SupportsBlocking bool
}

type useItemKind int

const (
	useItemKindNone useItemKind = iota
	useItemKindBlock
	useItemKindBow
	useItemKindFood
	useItemKindPotionDrink
	useItemKindMilkDrink
)

type useItemProfile struct {
	Kind           useItemKind
	MaxUseDuration int
	AlwaysEdible   bool
	FoodHealAmount int16
	FoodSaturation float32
}

var meleeItemProfiles = map[int16]meleeItemProfile{
	// Translated from ItemSword/ItemTool + EnumToolMaterial constructors in 1.6.4.
	itemIDIronSword:    {AttackModifier: 6, HitDamageCost: 1, MaxDurability: 250, MaxUseDuration: 72000, SupportsBlocking: true},
	itemIDWoodSword:    {AttackModifier: 4, HitDamageCost: 1, MaxDurability: 59, MaxUseDuration: 72000, SupportsBlocking: true},
	itemIDStoneSword:   {AttackModifier: 5, HitDamageCost: 1, MaxDurability: 131, MaxUseDuration: 72000, SupportsBlocking: true},
	itemIDDiamondSword: {AttackModifier: 7, HitDamageCost: 1, MaxDurability: 1561, MaxUseDuration: 72000, SupportsBlocking: true},
	itemIDGoldSword:    {AttackModifier: 4, HitDamageCost: 1, MaxDurability: 32, MaxUseDuration: 72000, SupportsBlocking: true},

	itemIDIronShovel:    {AttackModifier: 3, HitDamageCost: 2, MaxDurability: 250},
	itemIDIronPickaxe:   {AttackModifier: 4, HitDamageCost: 2, MaxDurability: 250},
	itemIDIronAxe:       {AttackModifier: 5, HitDamageCost: 2, MaxDurability: 250},
	itemIDWoodShovel:    {AttackModifier: 1, HitDamageCost: 2, MaxDurability: 59},
	itemIDWoodPickaxe:   {AttackModifier: 2, HitDamageCost: 2, MaxDurability: 59},
	itemIDWoodAxe:       {AttackModifier: 3, HitDamageCost: 2, MaxDurability: 59},
	itemIDStoneShovel:   {AttackModifier: 2, HitDamageCost: 2, MaxDurability: 131},
	itemIDStonePickaxe:  {AttackModifier: 3, HitDamageCost: 2, MaxDurability: 131},
	itemIDStoneAxe:      {AttackModifier: 4, HitDamageCost: 2, MaxDurability: 131},
	itemIDDiamondShovel: {AttackModifier: 4, HitDamageCost: 2, MaxDurability: 1561},
	itemIDDiamondPick:   {AttackModifier: 5, HitDamageCost: 2, MaxDurability: 1561},
	itemIDDiamondAxe:    {AttackModifier: 6, HitDamageCost: 2, MaxDurability: 1561},
	itemIDGoldShovel:    {AttackModifier: 1, HitDamageCost: 2, MaxDurability: 32},
	itemIDGoldPickaxe:   {AttackModifier: 2, HitDamageCost: 2, MaxDurability: 32},
	itemIDGoldAxe:       {AttackModifier: 3, HitDamageCost: 2, MaxDurability: 32},
}

var foodItemProfiles = map[int16]useItemProfile{
	// Translated from Item.java ItemFood/ItemSoup/ItemSeedFood constructor arguments.
	itemIDAppleRed:      {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 4, FoodSaturation: 0.3},
	itemIDMushroomStew:  {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 6, FoodSaturation: 0.6},
	itemIDBread:         {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 5, FoodSaturation: 0.6},
	itemIDPorkRaw:       {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 3, FoodSaturation: 0.3},
	itemIDPorkCooked:    {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 8, FoodSaturation: 0.8},
	itemIDAppleGold:     {Kind: useItemKindFood, MaxUseDuration: 32, AlwaysEdible: true, FoodHealAmount: 4, FoodSaturation: 1.2},
	itemIDFishRaw:       {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 2, FoodSaturation: 0.3},
	itemIDFishCooked:    {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 5, FoodSaturation: 0.6},
	itemIDCookie:        {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 2, FoodSaturation: 0.1},
	itemIDMelon:         {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 2, FoodSaturation: 0.3},
	itemIDBeefRaw:       {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 3, FoodSaturation: 0.3},
	itemIDBeefCooked:    {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 8, FoodSaturation: 0.8},
	itemIDChickenRaw:    {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 2, FoodSaturation: 0.3},
	itemIDChickenCooked: {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 6, FoodSaturation: 0.6},
	itemIDRottenFlesh:   {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 4, FoodSaturation: 0.1},
	itemIDSpiderEye:     {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 2, FoodSaturation: 0.8},
	itemIDCarrot:        {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 4, FoodSaturation: 0.6},
	itemIDPotato:        {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 1, FoodSaturation: 0.3},
	itemIDBakedPotato:   {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 6, FoodSaturation: 0.6},
	itemIDPoisonPotato:  {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 2, FoodSaturation: 0.3},
	itemIDGoldenCarrot:  {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 6, FoodSaturation: 1.2},
	itemIDPumpkinPie:    {Kind: useItemKindFood, MaxUseDuration: 32, FoodHealAmount: 8, FoodSaturation: 0.3},
}

var (
	patternControlCode = regexp.MustCompile("(?i)" + "\u00A7" + "[0-9A-FK-OR]")
	xzDirectionsConst  = [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}
)

func stripControlCodes(v string) string {
	return patternControlCode.ReplaceAllString(v, "")
}

type loginSession struct {
	server *StatusServer
	conn   net.Conn

	reader  io.Reader
	writer  io.Writer
	writeMu sync.Mutex

	stateMu sync.Mutex

	playerX            float64
	playerY            float64
	playerZ            float64
	playerStance       float64
	playerYaw          float32
	playerPitch        float32
	playerOnGround     bool
	playerHealth       float32
	playerFood         int16
	playerSat          float32
	playerFoodExhaust  float32
	playerFoodTimer    int
	playerPrevFood     int16
	playerExperience   float32
	playerExpLevel     int32
	playerExpTotal     int32
	playerFallDistance float32
	playerDead         bool
	playerSneaking     bool
	playerSprinting    bool
	playerInputStrafe  float32
	playerInputForward float32
	playerInputJump    bool
	playerInputSneak   bool
	ridingEntityID     int32
	playerUsingItem    bool
	playerIsFlying     bool
	playerItemUseCount int
	hurtTime           int
	hurtResistantTime  int
	lastDamage         float32
	managedChunkX      int32
	managedChunkZ      int32
	loadedChunks       map[chunk.CoordIntPair]struct{}
	pendingChunks      []chunk.CoordIntPair
	seenBy             map[*loginSession]struct{}

	chatSpamThresholdCount int
	// Translation reference:
	// - net.minecraft.src.NetServerHandler.creativeItemCreationSpamThresholdTally
	creativeItemCreationSpamThresholdTally int
	clientRenderDistance                   int8
	clientLocale                           string
	heldItemSlot                           int16
	gameType                               int8
	playerSpawnSet                         bool
	playerSpawnForced                      bool
	playerSpawnX                           int32
	playerSpawnY                           int32
	playerSpawnZ                           int32
	inventory                              [playerWindowSlots]*protocol.ItemStack
	cursorItem                             *protocol.ItemStack

	entityID        int32
	lastEntityPosX  int32
	lastEntityPosY  int32
	lastEntityPosZ  int32
	lastEntityYaw   int8
	lastEntityPitch int8
	lastHeadYaw     int8

	clientUsername    string
	verifyToken       []byte
	loginServerID     string
	sharedKey         []byte
	inputDecrypted    bool
	outputEncrypted   bool
	loginCommandSeen  bool
	playerReady       bool
	playerInitialized bool
	playerCounted     bool
	playerRegistered  bool

	keepAliveExpectedID int32
	keepAlivePending    atomic.Bool
	keepAliveSentAtMS   int64
	latencyMS           atomic.Int32
	stopLoops           chan struct{}
}

func newLoginSession(server *StatusServer, conn net.Conn) *loginSession {
	return &loginSession{
		server:       server,
		conn:         conn,
		reader:       conn,
		writer:       conn,
		stopLoops:    make(chan struct{}),
		loadedChunks: make(map[chunk.CoordIntPair]struct{}),
		seenBy:       make(map[*loginSession]struct{}),
	}
}

// run translates the packet flow in net.minecraft.src.NetLoginHandler.
func (s *loginSession) run() {
	defer close(s.stopLoops)
	defer func() {
		if s.playerInitialized && s.clientUsername != "" {
			_ = s.server.savePlayerState(s.clientUsername, s.snapshotPersistedState())
		}
		if s.playerRegistered {
			if username, ok := s.server.unregisterActivePlayer(s); ok {
				s.server.broadcastPacket(&protocol.Packet201PlayerInfo{
					PlayerName:  username,
					IsConnected: false,
					Ping:        0,
				})
				if s.entityID != 0 {
					viewers := s.snapshotSeenBy()
					for _, viewer := range viewers {
						_ = viewer.sendPacket(&protocol.Packet29DestroyEntity{
							EntityIDs: []int32{s.entityID},
						})
					}
				}
			}
		}
		if s.playerCounted {
			s.server.currentPlayers.Add(-1)
		}
	}()

	for {
		packet, err := protocol.ReadPacket(s.reader, protocol.DirectionServerbound)
		if err != nil {
			return
		}

		switch p := packet.(type) {
		case *protocol.Packet254ServerPing:
			s.sendPacket(&protocol.Packet255KickDisconnect{
				Reason: BuildServerPingResponse(p, s.server.cfg.MOTD, s.server.cfg.VersionName, s.server.CurrentPlayers(), s.server.cfg.MaxPlayers),
			})
			return
		case *protocol.Packet2ClientProtocol:
			if !s.handleClientProtocol(p) {
				return
			}
		case *protocol.Packet252SharedKey:
			if !s.handleSharedKey(p) {
				return
			}
		case *protocol.Packet205ClientCommand:
			if !s.handleClientCommand(p) {
				return
			}
		case *protocol.Packet1Login:
			// Translation target: NetLoginHandler#handleLogin is intentionally empty.
		case *protocol.Packet0KeepAlive:
			s.handleKeepAliveResponse(p)
		case *protocol.Packet3Chat:
			if !s.handleChat(p) {
				return
			}
		case *protocol.Packet10Flying:
			if !s.handleFlying(p) {
				return
			}
		case *protocol.Packet11PlayerPosition:
			if !s.handleFlying(&p.Packet10Flying) {
				return
			}
		case *protocol.Packet12PlayerLook:
			if !s.handleFlying(&p.Packet10Flying) {
				return
			}
		case *protocol.Packet13PlayerLookMove:
			if !s.handleFlying(&p.Packet10Flying) {
				return
			}
		case *protocol.Packet14BlockDig:
			if !s.handleBlockDig(p) {
				return
			}
		case *protocol.Packet15Place:
			if !s.handlePlace(p) {
				return
			}
		case *protocol.Packet7UseEntity:
			if !s.handleUseEntity(p) {
				return
			}
		case *protocol.Packet18Animation:
			s.handleAnimation(p)
		case *protocol.Packet19EntityAction:
			s.handleEntityAction(p)
		case *protocol.Packet27PlayerInput:
			s.handlePlayerInput(p)
		case *protocol.Packet16BlockItemSwitch:
			s.handleHeldItemSwitch(p)
		case *protocol.Packet101CloseWindow:
			s.handleCloseWindow(p)
		case *protocol.Packet102WindowClick:
			if !s.handleWindowClick(p) {
				return
			}
		case *protocol.Packet107CreativeSetSlot:
			s.handleCreativeSetSlot(p)
		case *protocol.Packet106Transaction:
			// Translation target: server consumes client transaction confirm packet.
		case *protocol.Packet202PlayerAbilities:
			s.handlePlayerAbilities(p)
		case *protocol.Packet204ClientInfo:
			s.handleClientInfo(p)
		default:
			s.disconnect("Protocol error")
			return
		}

		if s.playerReady && !s.playerInitialized {
			if !s.initializePlayerConnection() {
				return
			}
			s.playerInitialized = true
			go s.keepAliveLoop()
			go s.playLoop()
		}
	}
}

func (s *loginSession) snapshotPersistedState() persistedPlayerState {
	state := defaultPersistedPlayerState()

	s.stateMu.Lock()
	state.X = s.playerX
	state.Y = s.playerY
	state.Z = s.playerZ
	state.Yaw = s.playerYaw
	state.Pitch = s.playerPitch
	state.OnGround = s.playerOnGround
	state.Health = s.playerHealth
	state.Food = s.playerFood
	state.Sat = s.playerSat
	state.FoodExhaust = s.playerFoodExhaust
	state.FoodTickTimer = s.playerFoodTimer
	state.Experience = s.playerExperience
	state.ExperienceLvl = s.playerExpLevel
	state.ExperienceTot = s.playerExpTotal
	state.GameType = s.gameType
	state.HeldSlot = s.heldItemSlot
	state.HasSpawn = s.playerSpawnSet
	state.SpawnForced = s.playerSpawnForced
	state.SpawnX = s.playerSpawnX
	state.SpawnY = s.playerSpawnY
	state.SpawnZ = s.playerSpawnZ
	state.Inventory = cloneInventoryArray(s.inventory)
	s.stateMu.Unlock()

	if state.Health < 0 {
		state.Health = 0
	}
	if state.Food < 0 {
		state.Food = 0
	}
	if state.Food > 20 {
		state.Food = 20
	}
	if state.Sat < 0 {
		state.Sat = 0
	}
	if state.Sat > float32(state.Food) {
		state.Sat = float32(state.Food)
	}
	if state.FoodExhaust < 0 {
		state.FoodExhaust = 0
	}
	if state.FoodExhaust > 40 {
		state.FoodExhaust = 40
	}
	if state.FoodTickTimer < 0 {
		state.FoodTickTimer = 0
	}
	if state.Experience < 0 {
		state.Experience = 0
	}
	if state.Experience > 1 {
		state.Experience = 1
	}
	if state.ExperienceLvl < 0 {
		state.ExperienceLvl = 0
	}
	if state.ExperienceTot < 0 {
		state.ExperienceTot = 0
	}
	if state.HeldSlot < 0 || state.HeldSlot >= hotbarSlotCount {
		state.HeldSlot = 0
	}
	state.GameType = normalizeGameTypeValue(state.GameType)
	return state
}

func (s *loginSession) sendPacket(packet protocol.Packet) bool {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := protocol.WritePacket(s.writer, packet); err != nil {
		return false
	}
	return true
}

func (s *loginSession) disconnect(reason string) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = protocol.WritePacket(s.writer, &protocol.Packet255KickDisconnect{Reason: reason})
}

func (s *loginSession) randomServerID() string {
	var b [8]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "0"
	}
	n := int64(binary.BigEndian.Uint64(b[:]))
	return strconv.FormatInt(n, 16)
}

func (s *loginSession) handleClientProtocol(packet *protocol.Packet2ClientProtocol) bool {
	if s.clientUsername != "" {
		s.disconnect("Quit repeating yourself!")
		return false
	}

	s.clientUsername = packet.Username
	if s.clientUsername != stripControlCodes(s.clientUsername) {
		s.disconnect("Invalid username!")
		return false
	}

	if int(packet.ProtocolVersion) != protocol.ProtocolVersion {
		if int(packet.ProtocolVersion) > protocol.ProtocolVersion {
			s.disconnect("Outdated server!")
		} else {
			s.disconnect("Outdated client!")
		}
		return false
	}

	if s.server.cfg.OnlineMode {
		s.loginServerID = s.randomServerID()
	} else {
		s.loginServerID = "-"
	}

	s.verifyToken = make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, s.verifyToken); err != nil {
		s.disconnect("Protocol error")
		return false
	}

	if !s.sendPacket(&protocol.Packet253ServerAuthData{
		ServerID:    s.loginServerID,
		PublicKey:   s.server.publicKeyDER,
		VerifyToken: s.verifyToken,
	}) {
		return false
	}
	return true
}

func (s *loginSession) handleSharedKey(packet *protocol.Packet252SharedKey) bool {
	sharedKey, err := crypt.DecryptSharedKey(s.server.privateKey, packet.SharedSecret)
	if err != nil {
		s.disconnect("Invalid client reply")
		return false
	}
	verifyToken, err := crypt.DecryptData(s.server.privateKey, packet.VerifyToken)
	if err != nil {
		s.disconnect("Invalid client reply")
		return false
	}

	if !bytes.Equal(s.verifyToken, verifyToken) {
		s.disconnect("Invalid client reply")
		return false
	}

	s.sharedKey = sharedKey
	if !s.inputDecrypted {
		decryptedReader, err := crypt.DecryptInputStream(s.sharedKey, s.conn)
		if err != nil {
			s.disconnect("Protocol error")
			return false
		}
		s.reader = decryptedReader
		s.inputDecrypted = true
	}

	if !s.sendPacket(&protocol.Packet252SharedKey{}) {
		return false
	}

	if !s.outputEncrypted {
		encryptedWriter, err := crypt.EncryptOutputStream(s.sharedKey, s.conn)
		if err != nil {
			return false
		}
		s.writer = encryptedWriter
		s.outputEncrypted = true
	}

	return true
}

func (s *loginSession) handleClientCommand(packet *protocol.Packet205ClientCommand) bool {
	if s.playerInitialized {
		if packet.ForceRespawn != 1 {
			return true
		}
		return s.handleRespawnCommand()
	}

	if packet.ForceRespawn != 0 {
		return true
	}

	if s.loginCommandSeen {
		s.disconnect("Duplicate login")
		return false
	}
	s.loginCommandSeen = true

	if s.server.cfg.OnlineMode {
		s.disconnect("Online mode login verification not implemented")
		return false
	}

	s.playerReady = true
	return true
}

func (s *loginSession) sendHealthState() bool {
	s.stateMu.Lock()
	health := s.playerHealth
	food := s.playerFood
	sat := s.playerSat
	s.stateMu.Unlock()
	return s.sendPacket(&protocol.Packet8UpdateHealth{
		HealthMP:       health,
		Food:           food,
		FoodSaturation: sat,
	})
}

func clampInt16FromInt32(v int32) int16 {
	if v < -32768 {
		return -32768
	}
	if v > 32767 {
		return 32767
	}
	return int16(v)
}

func (s *loginSession) sendExperienceState() bool {
	s.stateMu.Lock()
	exp := s.playerExperience
	level := s.playerExpLevel
	total := s.playerExpTotal
	s.stateMu.Unlock()

	if exp < 0 {
		exp = 0
	}
	if exp > 1 {
		exp = 1
	}
	if level < 0 {
		level = 0
	}
	if total < 0 {
		total = 0
	}
	return s.sendPacket(&protocol.Packet43Experience{
		Experience:      exp,
		ExperienceLevel: clampInt16FromInt32(level),
		ExperienceTotal: clampInt16FromInt32(total),
	})
}

// Translation reference:
// - net.minecraft.src.EntityPlayer.xpBarCap()
func xpBarCapForLevel(level int32) int32 {
	if level >= 30 {
		return 62 + (level-30)*7
	}
	if level >= 15 {
		return 17 + (level-15)*3
	}
	return 17
}

// Translation reference:
// - net.minecraft.src.EntityPlayer.addExperienceLevel(int)
func (s *loginSession) addExperienceLevel(delta int32) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.playerExpLevel += delta
	if s.playerExpLevel < 0 {
		s.playerExpLevel = 0
		s.playerExperience = 0
		s.playerExpTotal = 0
	}
}

// Translation reference:
// - net.minecraft.src.EntityPlayer.addExperience(int)
func (s *loginSession) addExperience(amount int32) {
	if amount <= 0 {
		return
	}

	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	const maxTotalXP = int32(2147483647)
	remaining := maxTotalXP - s.playerExpTotal
	if amount > remaining {
		amount = remaining
	}
	if amount <= 0 {
		return
	}

	capNow := xpBarCapForLevel(s.playerExpLevel)
	if capNow <= 0 {
		capNow = 1
	}
	s.playerExperience += float32(amount) / float32(capNow)

	for s.playerExpTotal += amount; s.playerExperience >= 1.0; {
		capBefore := xpBarCapForLevel(s.playerExpLevel)
		if capBefore <= 0 {
			capBefore = 1
		}
		s.playerExperience = (s.playerExperience - 1.0) * float32(capBefore)
		s.playerExpLevel++

		capAfter := xpBarCapForLevel(s.playerExpLevel)
		if capAfter <= 0 {
			capAfter = 1
		}
		s.playerExperience /= float32(capAfter)
	}
}

func (s *loginSession) handleRespawnCommand() bool {
	s.stateMu.Lock()
	dead := s.playerDead
	gameType := s.gameType
	s.stateMu.Unlock()
	if !dead {
		return true
	}

	difficulty := s.server.currentDifficulty()
	if !s.sendPacket(&protocol.Packet9Respawn{
		RespawnDimension: 0,
		Difficulty:       difficulty,
		WorldHeight:      0,
		GameType:         gameType,
		TerrainType:      "default",
	}) {
		return false
	}

	spawnX, spawnY, spawnZ := s.respawnPosition()
	if !s.setPlayerLocation(spawnX, spawnY, spawnZ, 0, 0, true) {
		return false
	}

	s.stateMu.Lock()
	s.playerHealth = maxPlayerHealth
	s.playerFood = 20
	s.playerSat = 5.0
	s.playerFoodExhaust = 0
	s.playerFoodTimer = 0
	s.playerPrevFood = s.playerFood
	s.playerFallDistance = 0
	s.playerDead = false
	s.playerUsingItem = false
	s.playerIsFlying = false
	s.playerItemUseCount = 0
	s.hurtTime = 0
	s.hurtResistantTime = 0
	s.lastDamage = 0
	s.stateMu.Unlock()

	if !s.sendHealthState() {
		return false
	}
	if !s.sendExperienceState() {
		return false
	}
	worldAge, worldTime := s.server.CurrentWorldTime()
	return s.sendPacket(protocol.NewPacket4UpdateTime(worldAge, worldTime, true))
}

func (s *loginSession) respawnPosition() (float64, float64, float64) {
	worldX, worldY, worldZ := s.server.world.safePlayerSpawnPosition()

	s.stateMu.Lock()
	hasSpawn := s.playerSpawnSet
	spawnForced := s.playerSpawnForced
	spawnX := s.playerSpawnX
	spawnY := s.playerSpawnY
	spawnZ := s.playerSpawnZ
	s.stateMu.Unlock()
	if !hasSpawn {
		return worldX, worldY, worldZ
	}

	x := float64(spawnX) + 0.5
	y := float64(spawnY) + 0.1
	z := float64(spawnZ) + 0.5
	if s.isValidRespawnPosition(x, y, z, spawnForced) {
		return x, y, z
	}
	return worldX, worldY, worldZ
}

func (s *loginSession) isValidRespawnPosition(x, y, z float64, spawnForced bool) bool {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) {
		return false
	}
	if math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
		return false
	}

	blockX := int(math.Floor(x))
	blockY := int(math.Floor(y))
	blockZ := int(math.Floor(z))
	if blockY < 0 || blockY > buildLimit {
		return false
	}

	feetID, _ := s.server.world.getBlock(blockX, blockY, blockZ)
	headID, _ := s.server.world.getBlock(blockX, blockY+1, blockZ)
	if block.BlocksMovement(feetID) || block.IsLiquid(feetID) {
		return false
	}
	if block.BlocksMovement(headID) || block.IsLiquid(headID) {
		return false
	}
	if spawnForced {
		return true
	}
	if blockY <= 0 {
		return false
	}
	belowID, _ := s.server.world.getBlock(blockX, blockY-1, blockZ)
	if !block.BlocksMovement(belowID) {
		return false
	}
	return true
}

func (s *loginSession) killPlayer() bool {
	s.stateMu.Lock()
	s.playerHealth = 0
	s.playerDead = true
	s.playerUsingItem = false
	s.playerIsFlying = false
	s.playerItemUseCount = 0
	s.playerFoodTimer = 0
	s.playerFallDistance = 0
	s.stateMu.Unlock()
	return s.sendHealthState()
}

func (s *loginSession) initializePlayerConnection() bool {
	loadedState, loadedFromDisk := s.server.loadPlayerState(s.clientUsername)
	if !loadedFromDisk {
		spawnX, spawnY, spawnZ := s.server.world.safePlayerSpawnPosition()
		loadedState.X = spawnX
		loadedState.Y = spawnY
		loadedState.Z = spawnZ
		loadedState.Yaw = 0
		loadedState.Pitch = 0
		loadedState.OnGround = false
		loadedState.GameType = int8(s.server.defaultGameType.Load())
	}
	loadedState.GameType = normalizeGameTypeValue(loadedState.GameType)
	if loadedState.HeldSlot < 0 || loadedState.HeldSlot >= hotbarSlotCount {
		loadedState.HeldSlot = 0
	}
	if loadedState.Health <= 0 {
		loadedState.Health = maxPlayerHealth
	}
	if loadedState.Food < 0 {
		loadedState.Food = 20
	}
	if loadedState.Food > 20 {
		loadedState.Food = 20
	}
	if loadedState.Sat < 0 {
		loadedState.Sat = 0
	}
	if loadedState.Sat > float32(loadedState.Food) {
		loadedState.Sat = float32(loadedState.Food)
	}
	if loadedState.FoodExhaust < 0 {
		loadedState.FoodExhaust = 0
	}
	if loadedState.FoodExhaust > 40 {
		loadedState.FoodExhaust = 40
	}
	if loadedState.FoodTickTimer < 0 {
		loadedState.FoodTickTimer = 0
	}
	if loadedState.Experience < 0 {
		loadedState.Experience = 0
	}
	if loadedState.Experience > 1 {
		loadedState.Experience = 1
	}
	if loadedState.ExperienceLvl < 0 {
		loadedState.ExperienceLvl = 0
	}
	if loadedState.ExperienceTot < 0 {
		loadedState.ExperienceTot = 0
	}
	loadedState = s.sanitizeLoadedPlayerPosition(loadedState)

	entityID := s.server.nextEntityID.Add(1)
	s.entityID = entityID
	s.server.currentPlayers.Add(1)
	s.playerCounted = true
	gameType := loadedState.GameType
	s.gameType = gameType

	if brand, err := protocol.NewPacket250CustomPayload("MC|Brand", []byte("gomc")); err == nil {
		if !s.sendPacket(brand) {
			return false
		}
	}

	if !s.sendPacket(&protocol.Packet1Login{
		ClientEntityID:    entityID,
		TerrainType:       "default",
		HardcoreMode:      false,
		GameType:          gameType,
		Dimension:         0,
		DifficultySetting: s.server.currentDifficulty(),
		WorldHeight:       0,
		MaxPlayers:        int8(s.server.cfg.MaxPlayers),
	}) {
		return false
	}

	spawnBlockX, spawnBlockY, spawnBlockZ := s.server.world.spawnBlockPosition()
	if !s.sendPacket(&protocol.Packet6SpawnPosition{
		XPosition: spawnBlockX,
		YPosition: spawnBlockY,
		ZPosition: spawnBlockZ,
	}) {
		return false
	}

	if !s.sendPacket(s.currentAbilitiesPacket()) {
		return false
	}
	if !s.sendPacket(&protocol.Packet8UpdateHealth{
		HealthMP:       loadedState.Health,
		Food:           loadedState.Food,
		FoodSaturation: loadedState.Sat,
	}) {
		return false
	}
	if !s.sendPacket(&protocol.Packet43Experience{
		Experience:      loadedState.Experience,
		ExperienceLevel: clampInt16FromInt32(loadedState.ExperienceLvl),
		ExperienceTotal: clampInt16FromInt32(loadedState.ExperienceTot),
	}) {
		return false
	}

	if !s.sendPacket(&protocol.Packet16BlockItemSwitch{ID: loadedState.HeldSlot}) {
		return false
	}
	if !s.sendPacket(&protocol.Packet104WindowItems{
		WindowID:   0,
		ItemStacks: toInventorySlice(cloneInventoryArray(loadedState.Inventory)),
	}) {
		return false
	}

	move := protocol.NewPacket13PlayerLookMove()
	move.XPosition = loadedState.X
	move.YPosition = loadedState.Y + defaultPlayerEyeY
	move.Stance = loadedState.Y
	move.ZPosition = loadedState.Z
	move.Yaw = loadedState.Yaw
	move.Pitch = loadedState.Pitch
	move.OnGround = loadedState.OnGround
	if !s.sendPacket(move) {
		return false
	}

	spawnChunk := s.server.world.getChunk(chunkCoordFromPos(loadedState.X), chunkCoordFromPos(loadedState.Z))
	mapChunkPacket, err := protocol.NewPacket51MapChunk(spawnChunk, true, 65535, false)
	if err == nil {
		if !s.sendPacket(mapChunkPacket) {
			return false
		}
	}

	worldAge, worldTime := s.server.CurrentWorldTime()
	if !s.sendPacket(protocol.NewPacket4UpdateTime(worldAge, worldTime, true)) {
		return false
	}

	s.stateMu.Lock()
	s.playerX = loadedState.X
	s.playerY = loadedState.Y
	s.playerZ = loadedState.Z
	s.playerStance = loadedState.Y + defaultPlayerEyeY
	s.playerYaw = loadedState.Yaw
	s.playerPitch = loadedState.Pitch
	s.playerOnGround = loadedState.OnGround
	s.playerHealth = loadedState.Health
	s.playerFood = loadedState.Food
	s.playerSat = loadedState.Sat
	s.playerFoodExhaust = loadedState.FoodExhaust
	s.playerFoodTimer = loadedState.FoodTickTimer
	s.playerPrevFood = loadedState.Food
	s.playerExperience = loadedState.Experience
	s.playerExpLevel = loadedState.ExperienceLvl
	s.playerExpTotal = loadedState.ExperienceTot
	s.playerFallDistance = 0
	s.playerDead = false
	s.playerUsingItem = false
	s.playerIsFlying = false
	s.playerItemUseCount = 0
	s.hurtTime = 0
	s.hurtResistantTime = 0
	s.lastDamage = 0
	s.heldItemSlot = loadedState.HeldSlot
	s.playerSpawnSet = loadedState.HasSpawn
	s.playerSpawnForced = loadedState.SpawnForced
	s.playerSpawnX = loadedState.SpawnX
	s.playerSpawnY = loadedState.SpawnY
	s.playerSpawnZ = loadedState.SpawnZ
	s.inventory = cloneInventoryArray(loadedState.Inventory)
	s.managedChunkX = chunkCoordFromPos(loadedState.X)
	s.managedChunkZ = chunkCoordFromPos(loadedState.Z)
	s.loadedChunks = make(map[chunk.CoordIntPair]struct{})
	s.loadedChunks[chunk.NewCoordIntPair(s.managedChunkX, s.managedChunkZ)] = struct{}{}
	s.pendingChunks = nil
	s.lastEntityPosX = toPacketPosition(loadedState.X)
	s.lastEntityPosY = toPacketPosition(loadedState.Y)
	s.lastEntityPosZ = toPacketPosition(loadedState.Z)
	s.lastEntityYaw = toPacketAngle(loadedState.Yaw)
	s.lastEntityPitch = toPacketAngle(loadedState.Pitch)
	s.lastHeadYaw = toPacketAngle(loadedState.Yaw)
	s.rebuildChunkQueuesLocked(s.managedChunkX, s.managedChunkZ, s.server.cfg.ViewDistance)
	s.stateMu.Unlock()

	existing := s.server.registerActivePlayer(s, s.clientUsername)
	s.playerRegistered = true
	for _, ref := range existing {
		if !s.sendPacket(&protocol.Packet201PlayerInfo{
			PlayerName:  ref.Username,
			IsConnected: true,
			Ping:        int16(ref.Session.currentPing()),
		}) {
			return false
		}
		if spawn := ref.Session.buildNamedEntitySpawnPacket(); spawn != nil {
			attach := ref.Session.buildAttachEntityPacket()
			existingChunkX, existingChunkZ := ref.Session.currentChunkCoords()
			if s.isWatchingChunk(existingChunkX, existingChunkZ) {
				if !s.sendPacket(spawn) {
					return false
				}
				if attach != nil && !s.sendPacket(attach) {
					return false
				}
				ref.Session.markSeenBy(s, true)
				s.markSeenBy(ref.Session, true)
			}
		}
	}

	s.server.broadcastPacket(&protocol.Packet201PlayerInfo{
		PlayerName:  s.clientUsername,
		IsConnected: true,
		Ping:        1000,
	})
	if spawn := s.buildNamedEntitySpawnPacket(); spawn != nil {
		attach := s.buildAttachEntityPacket()
		targets := s.server.activeSessionsExcept(s)
		for _, target := range targets {
			if !target.isWatchingChunk(s.managedChunkX, s.managedChunkZ) {
				continue
			}
			if !target.sendPacket(spawn) {
				continue
			}
			if attach != nil {
				_ = target.sendPacket(attach)
			}
			s.markSeenBy(target, true)
		}
	}
	return true
}

func (s *loginSession) sanitizeLoadedPlayerPosition(state persistedPlayerState) persistedPlayerState {
	spawnX, spawnY, spawnZ := s.server.world.safePlayerSpawnPosition()
	resetToSpawn := func() persistedPlayerState {
		state.X = spawnX
		state.Y = spawnY
		state.Z = spawnZ
		state.Yaw = 0
		state.Pitch = 0
		state.OnGround = false
		return state
	}

	invalid := math.IsNaN(state.X) || math.IsNaN(state.Y) || math.IsNaN(state.Z) ||
		math.IsInf(state.X, 0) || math.IsInf(state.Y, 0) || math.IsInf(state.Z, 0)
	if invalid || state.Y < 0 || state.Y > 255 {
		return resetToSpawn()
	}

	blockX := int(math.Floor(state.X))
	blockY := int(math.Floor(state.Y))
	blockZ := int(math.Floor(state.Z))
	if blockY < 0 || blockY >= 255 {
		return resetToSpawn()
	}

	feetBlockID, _ := s.server.world.getBlock(blockX, blockY, blockZ)
	headBlockID, _ := s.server.world.getBlock(blockX, blockY+1, blockZ)
	belowBlockID, _ := s.server.world.getBlock(blockX, blockY-1, blockZ)
	if block.BlocksMovement(feetBlockID) || block.BlocksMovement(headBlockID) || !block.BlocksMovement(belowBlockID) {
		return resetToSpawn()
	}
	return state
}

func (s *loginSession) randomInt32() int32 {
	var b [4]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return 0
	}
	return int32(binary.BigEndian.Uint32(b[:]))
}

func (s *loginSession) keepAliveLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopLoops:
			return
		case <-ticker.C:
			id := s.randomInt32()
			s.keepAliveExpectedID = id
			s.keepAliveSentAtMS = time.Now().UnixMilli()
			s.keepAlivePending.Store(true)
			if !s.sendPacket(&protocol.Packet0KeepAlive{RandomID: id}) {
				return
			}
		}
	}
}

func (s *loginSession) playLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	ticks := 0

	for {
		select {
		case <-s.stopLoops:
			return
		case <-ticker.C:
			ticks++
			s.stateMu.Lock()
			if s.chatSpamThresholdCount > 0 {
				s.chatSpamThresholdCount--
			}
			if s.creativeItemCreationSpamThresholdTally > 0 {
				s.creativeItemCreationSpamThresholdTally--
			}
			if s.hurtTime > 0 {
				s.hurtTime--
			}
			if s.hurtResistantTime > 0 {
				s.hurtResistantTime--
			}
			s.stateMu.Unlock()
			s.tickHeldItemUse()
			s.tickFoodStats()

			if ticks%20 == 0 {
				worldAge, worldTime := s.server.CurrentWorldTime()
				if !s.sendPacket(protocol.NewPacket4UpdateTime(worldAge, worldTime, true)) {
					return
				}
			}

			coords := s.drainPendingChunks(12)
			if len(coords) == 0 {
				continue
			}

			chunks := s.server.world.getChunks(coords)
			packet, err := protocol.NewPacket56MapChunks(chunks, false)
			if err != nil {
				continue
			}
			if !s.sendPacket(packet) {
				return
			}
			s.refreshVisibleEntitiesForSelf()
		}
	}
}

func (s *loginSession) handleKeepAliveResponse(packet *protocol.Packet0KeepAlive) {
	if packet == nil {
		return
	}
	if s.keepAlivePending.Load() && packet.RandomID == s.keepAliveExpectedID {
		s.keepAlivePending.Store(false)
		rtt := int32(time.Now().UnixMilli() - s.keepAliveSentAtMS)
		if rtt < 0 {
			rtt = 0
		}
		prev := s.latencyMS.Load()
		s.latencyMS.Store((prev*3 + rtt) / 4)
	}
}

func (s *loginSession) handleChat(packet *protocol.Packet3Chat) bool {
	if packet == nil {
		return true
	}

	message := normalizeSpace(packet.Message)
	if len(message) > 100 {
		s.disconnect("Chat message too long")
		return false
	}

	for _, r := range message {
		if !isAllowedChatCharacter(r) {
			s.disconnect("Illegal characters in chat")
			return false
		}
	}

	if strings.HasPrefix(message, "/") {
		return s.handleSlashCommand(message)
	}

	chatText := "<" + s.clientUsername + "> " + message
	s.server.broadcastPacket(protocol.NewPacket3Chat(chatText, true))

	s.stateMu.Lock()
	s.chatSpamThresholdCount += 20
	spam := s.chatSpamThresholdCount
	s.stateMu.Unlock()

	if spam > 200 {
		s.disconnect("disconnect.spam")
		return false
	}
	return true
}

func (s *loginSession) handleSlashCommand(command string) bool {
	// Translation target:
	// - NetServerHandler#handleSlashCommand delegates to CommandManager.
	//   Command tree translation is pending; unknown commands are reported as server chat text.
	args := strings.Fields(command)
	if len(args) == 0 {
		return true
	}

	if strings.EqualFold(args[0], "/help") || args[0] == "/?" {
		// Translation target:
		// - net.minecraft.src.CommandHelp#processCommand
		helpCommands := []string{
			"clear",
			"difficulty",
			"gamemode",
			"give",
			"help",
			"kill",
			"list",
			"me",
			"save-all",
			"save-off",
			"save-on",
			"say",
			"seed",
			"spawnpoint",
			"setblock",
			"tell",
			"time",
			"tp",
			"xp",
		}
		commandUsage := map[string]string{
			"clear":      "/clear <player> [item] [data]",
			"difficulty": "/difficulty <new difficulty>",
			"gamemode":   "/gamemode <mode> [player]",
			"give":       "/give <player> <item> [amount] [data]",
			"help":       "/help [page|command name]",
			"kill":       "/kill",
			"list":       "/list",
			"me":         "/me <action ...>",
			"save-all":   "/save-all",
			"save-off":   "/save-off",
			"save-on":    "/save-on",
			"say":        "/say <message ...>",
			"seed":       "/seed",
			"spawnpoint": "/spawnpoint OR /spawnpoint <player> OR /spawnpoint <player> <x> <y> <z>",
			"setblock":   "/setblock <x> <y> <z> <id> [meta]",
			"tell":       "/tell <player> <private message ...>",
			"time":       "/time <set|add> <value>",
			"tp":         "/tp [target player] <destination player> OR /tp [target player] <x> <y> <z>",
			"xp":         "/xp <amount> [player] OR /xp <amount>L [player]",
		}
		pageSize := 7
		totalPages := (len(helpCommands) + pageSize - 1) / pageSize
		pageIndex := 0
		if len(args) >= 2 {
			query := strings.TrimSpace(args[1])
			if p, err := strconv.Atoi(query); err == nil {
				if p < 1 || p > totalPages {
					s.sendSystemChat("Usage: /help [page|command name]")
					return true
				}
				pageIndex = p - 1
			} else {
				query = strings.TrimPrefix(strings.ToLower(query), "/")
				usage, ok := commandUsage[query]
				if !ok {
					s.sendSystemChat("Unknown command.")
					return true
				}
				s.sendSystemChat(usage)
				return true
			}
		}

		s.sendSystemChat(fmt.Sprintf("--- Showing help page %d of %d (/help <page>) ---", pageIndex+1, totalPages))
		start := pageIndex * pageSize
		end := start + pageSize
		if end > len(helpCommands) {
			end = len(helpCommands)
		}
		for i := start; i < end; i++ {
			s.sendSystemChat(commandUsage[helpCommands[i]])
		}
		if pageIndex == 0 {
			s.sendSystemChat("Tip: Use the <tab> key while typing a command to auto-complete the command or its arguments")
		}
		return true
	}

	if strings.EqualFold(args[0], "/list") {
		names := s.server.activePlayerNames()
		if len(names) == 0 {
			s.sendSystemChat("Players online (0)")
		} else {
			s.sendSystemChat("Players online (" + strconv.Itoa(len(names)) + "): " + strings.Join(names, ", "))
		}
		return true
	}

	if strings.EqualFold(args[0], "/seed") {
		// Translation target:
		// - net.minecraft.src.CommandShowSeed#processCommand
		s.sendSystemChat("Seed: " + strconv.FormatInt(s.server.world.seed, 10))
		return true
	}

	if strings.EqualFold(args[0], "/spawnpoint") {
		// Translation target:
		// - net.minecraft.src.CommandSetSpawnpoint#processCommand
		usage := "Usage: /spawnpoint OR /spawnpoint <player> OR /spawnpoint <player> <x> <y> <z>"
		if len(args) != 1 && len(args) != 2 && len(args) != 5 {
			s.sendSystemChat(usage)
			return true
		}

		target := s
		if len(args) >= 2 {
			found := s.server.activeSessionByUsername(args[1])
			if found == nil {
				s.sendSystemChat("Player not found.")
				return true
			}
			target = found
		}

		var (
			spawnX int32
			spawnY int32
			spawnZ int32
		)
		if len(args) == 5 {
			const coordinateBound = 30000000
			x, err := strconv.Atoi(args[2])
			if err != nil || x < -coordinateBound || x > coordinateBound {
				s.sendSystemChat(usage)
				return true
			}
			y, err := strconv.Atoi(args[3])
			if err != nil || y < 0 || y > buildLimit {
				s.sendSystemChat(usage)
				return true
			}
			z, err := strconv.Atoi(args[4])
			if err != nil || z < -coordinateBound || z > coordinateBound {
				s.sendSystemChat(usage)
				return true
			}
			spawnX = int32(x)
			spawnY = int32(y)
			spawnZ = int32(z)
		} else {
			posX, posY, posZ, _, _ := target.positionRotationSnapshot()
			spawnX = int32(math.Floor(posX))
			spawnY = int32(math.Floor(posY + 0.5))
			spawnZ = int32(math.Floor(posZ))
		}

		target.stateMu.Lock()
		target.playerSpawnSet = true
		target.playerSpawnForced = true
		target.playerSpawnX = spawnX
		target.playerSpawnY = spawnY
		target.playerSpawnZ = spawnZ
		target.stateMu.Unlock()

		targetName := strings.TrimSpace(target.clientUsername)
		if targetName == "" {
			targetName = "player"
		}
		s.sendSystemChat(fmt.Sprintf("Set %s's spawn point to (%d, %d, %d)", targetName, spawnX, spawnY, spawnZ))
		return true
	}

	if strings.EqualFold(args[0], "/save-off") {
		// Translation target:
		// - net.minecraft.src.CommandServerSaveOff#processCommand
		if !s.server.isAutoSaveEnabled() {
			s.sendSystemChat("Saving is already turned off.")
			return true
		}
		s.server.setAutoSaveEnabled(false)
		s.sendSystemChat("Turned off world auto-saving")
		return true
	}

	if strings.EqualFold(args[0], "/save-on") {
		// Translation target:
		// - net.minecraft.src.CommandServerSaveOn#processCommand
		if s.server.isAutoSaveEnabled() {
			s.sendSystemChat("Saving is already turned on.")
			return true
		}
		s.server.setAutoSaveEnabled(true)
		s.sendSystemChat("Turned on world auto-saving")
		return true
	}

	if strings.EqualFold(args[0], "/save-all") {
		// Translation target:
		// - net.minecraft.src.CommandServerSaveAll#processCommand
		s.sendSystemChat("Saving...")

		if s.server.cfg.PersistWorld {
			for _, session := range s.server.activeSessions() {
				if session == nil || !session.playerInitialized {
					continue
				}
				username := strings.TrimSpace(session.clientUsername)
				if username == "" {
					continue
				}
				_ = s.server.savePlayerState(username, session.snapshotPersistedState())
			}
		}

		if err := s.server.SaveWorldAll(); err != nil {
			s.sendSystemChat("Saving failed: " + err.Error())
			return true
		}
		s.sendSystemChat("Saved the world")
		return true
	}

	if strings.EqualFold(args[0], "/say") {
		// Translation target:
		// - net.minecraft.src.CommandServerSay#processCommand
		if len(args) < 2 {
			s.sendSystemChat("Usage: /say <message ...>")
			return true
		}
		message := strings.TrimSpace(strings.Join(args[1:], " "))
		if message == "" {
			s.sendSystemChat("Usage: /say <message ...>")
			return true
		}
		senderName := strings.TrimSpace(s.clientUsername)
		if senderName == "" {
			senderName = "Server"
		}
		s.server.broadcastPacket(protocol.NewPacket3Chat("["+senderName+"] "+message, true))
		return true
	}

	if strings.EqualFold(args[0], "/me") {
		// Translation target:
		// - net.minecraft.src.CommandServerEmote#processCommand
		if len(args) < 2 {
			s.sendSystemChat("Usage: /me <action ...>")
			return true
		}
		action := strings.TrimSpace(strings.Join(args[1:], " "))
		if action == "" {
			s.sendSystemChat("Usage: /me <action ...>")
			return true
		}
		senderName := strings.TrimSpace(s.clientUsername)
		if senderName == "" {
			senderName = "Player"
		}
		s.server.broadcastPacket(protocol.NewPacket3Chat("* "+senderName+" "+action, true))
		return true
	}

	if strings.EqualFold(args[0], "/tell") || strings.EqualFold(args[0], "/w") || strings.EqualFold(args[0], "/msg") {
		// Translation target:
		// - net.minecraft.src.CommandServerMessage#processCommand
		if len(args) < 3 {
			s.sendSystemChat("Usage: /tell <player> <private message ...>")
			return true
		}
		target := s.server.activeSessionByUsername(args[1])
		if target == nil {
			s.sendSystemChat("Player not found.")
			return true
		}
		if target == s {
			s.sendSystemChat("You can't send a private message to yourself!")
			return true
		}
		message := strings.TrimSpace(strings.Join(args[2:], " "))
		if message == "" {
			s.sendSystemChat("Usage: /tell <player> <private message ...>")
			return true
		}
		senderName := strings.TrimSpace(s.clientUsername)
		if senderName == "" {
			senderName = "Player"
		}
		targetName := strings.TrimSpace(target.clientUsername)
		if targetName == "" {
			targetName = "player"
		}
		if !target.sendPacket(protocol.NewPacket3Chat(senderName+" whispers to you: "+message, true)) {
			return false
		}
		s.sendSystemChat("You whisper to " + targetName + ": " + message)
		return true
	}

	if strings.EqualFold(args[0], "/tp") {
		params := args[1:]
		if len(params) < 1 {
			s.sendSystemChat("Usage: /tp <x> <y> <z>")
			return true
		}

		target := s
		if len(params) == 2 || len(params) == 4 {
			found := s.server.activeSessionByUsername(params[0])
			if found == nil {
				s.sendSystemChat("Player not found.")
				return true
			}
			target = found
		}

		targetName := strings.TrimSpace(target.clientUsername)
		if targetName == "" {
			targetName = "player"
		}

		if len(params) == 1 || len(params) == 2 {
			dest := s.server.activeSessionByUsername(params[len(params)-1])
			if dest == nil {
				s.sendSystemChat("Player not found.")
				return true
			}
			destX, destY, destZ, destYaw, destPitch := dest.positionRotationSnapshot()
			if !target.setPlayerLocation(destX, destY, destZ, destYaw, destPitch, true) {
				return false
			}
			destName := strings.TrimSpace(dest.clientUsername)
			if destName == "" {
				destName = "player"
			}
			s.sendSystemChat("Teleported " + targetName + " to " + destName)
			return true
		}

		if len(params) == 3 || len(params) == 4 {
			start := len(params) - 3
			curX, curY, curZ, yaw, pitch := target.positionRotationSnapshot()
			x, err := parseTeleportCoordinate(params[start], curX)
			if err != nil {
				s.sendSystemChat("Invalid X coordinate")
				return true
			}
			y, err := parseTeleportCoordinate(params[start+1], curY)
			if err != nil {
				s.sendSystemChat("Invalid Y coordinate")
				return true
			}
			z, err := parseTeleportCoordinate(params[start+2], curZ)
			if err != nil {
				s.sendSystemChat("Invalid Z coordinate")
				return true
			}
			if y < 0 || y >= buildLimit {
				s.sendSystemChat("Y must be between 0 and 255")
				return true
			}
			if math.Abs(x) > maxWorldCoordinate || math.Abs(z) > maxWorldCoordinate {
				s.sendSystemChat("Coordinates out of world bounds")
				return true
			}
			if !target.setPlayerLocation(x, y, z, yaw, pitch, true) {
				return false
			}
			s.sendSystemChat("Teleported " + targetName + " to " + strconv.FormatFloat(x, 'f', 3, 64) + " " + strconv.FormatFloat(y, 'f', 3, 64) + " " + strconv.FormatFloat(z, 'f', 3, 64))
			return true
		}

		s.sendSystemChat("Usage: /tp <x> <y> <z>")
		return true
	}

	if strings.EqualFold(args[0], "/give") {
		// Translation target:
		// - net.minecraft.src.CommandGive#processCommand
		if len(args) < 3 || len(args) > 5 {
			s.sendSystemChat("Usage: /give <player> <itemId> [count] [damage]")
			return true
		}

		target := s.server.activeSessionByUsername(args[1])
		if target == nil {
			s.sendSystemChat("Player not found.")
			return true
		}

		itemID, err := strconv.Atoi(args[2])
		if err != nil || itemID <= 0 || itemID > 32767 {
			s.sendSystemChat("Invalid item id")
			return true
		}

		count := 1
		if len(args) >= 4 {
			count, err = strconv.Atoi(args[3])
			if err != nil || count <= 0 || count > 64 {
				s.sendSystemChat("Invalid count (1-64)")
				return true
			}
		}

		damage := 0
		if len(args) >= 5 {
			damage, err = strconv.Atoi(args[4])
			if err != nil || damage < math.MinInt16 || damage > math.MaxInt16 {
				s.sendSystemChat("Invalid damage")
				return true
			}
		}

		dropped := s.server.spawnDroppedItemFromPlayer(target, &protocol.ItemStack{
			ItemID:     int16(itemID),
			StackSize:  int8(count),
			ItemDamage: int16(damage),
		}, false, false)
		if dropped == nil {
			s.sendSystemChat("Failed to give item")
			return true
		}
		s.server.droppedItemMu.Lock()
		dropped.DelayBeforeCanPick = 0
		s.server.droppedItemMu.Unlock()

		targetName := strings.TrimSpace(target.clientUsername)
		if targetName == "" {
			targetName = "player"
		}
		s.sendSystemChat("Given " + strconv.Itoa(count) + " of item " + strconv.Itoa(itemID) + " to " + targetName)
		return true
	}

	if strings.EqualFold(args[0], "/time") {
		if len(args) < 3 {
			s.sendSystemChat("Usage: /time <set|add> <value|day|night>")
			return true
		}

		action := strings.ToLower(args[1])
		valueText := strings.ToLower(args[2])
		parseValue := func(v string) (int64, error) {
			switch v {
			case "day":
				return 1000, nil
			case "night":
				return 13000, nil
			default:
				return strconv.ParseInt(v, 10, 64)
			}
		}

		value, err := parseValue(valueText)
		if err != nil {
			s.sendSystemChat("Invalid time value")
			return true
		}

		switch action {
		case "set":
			s.server.SetWorldTime(value)
		case "add":
			s.server.AdvanceWorldTime(value)
		default:
			s.sendSystemChat("Usage: /time <set|add> <value|day|night>")
			return true
		}

		worldAge, worldTime := s.server.CurrentWorldTime()
		s.server.broadcastPacket(protocol.NewPacket4UpdateTime(worldAge, worldTime, true))
		s.sendSystemChat("Set time to " + strconv.FormatInt(worldTime, 10))
		return true
	}

	if strings.EqualFold(args[0], "/difficulty") {
		// Translation target:
		// - net.minecraft.src.CommandDifficulty#processCommand
		if len(args) < 2 {
			s.sendSystemChat("Usage: /difficulty <new difficulty>")
			return true
		}

		modeText := strings.ToLower(strings.TrimSpace(args[1]))
		var difficulty int8
		switch modeText {
		case "peaceful", "p":
			difficulty = 0
		case "easy", "e":
			difficulty = 1
		case "normal", "n":
			difficulty = 2
		case "hard", "h":
			difficulty = 3
		default:
			v, err := strconv.Atoi(modeText)
			if err != nil || v < 0 || v > 3 {
				s.sendSystemChat("Usage: /difficulty <new difficulty>")
				return true
			}
			difficulty = int8(v)
		}

		difficulty = s.server.setDifficulty(difficulty)
		name := "Easy"
		switch difficulty {
		case 0:
			name = "Peaceful"
		case 1:
			name = "Easy"
		case 2:
			name = "Normal"
		case 3:
			name = "Hard"
		}
		s.sendSystemChat("Set game difficulty to " + name)
		return true
	}

	if strings.EqualFold(args[0], "/defaultgamemode") {
		if len(args) < 2 {
			s.sendSystemChat("Usage: /defaultgamemode <mode>")
			return true
		}

		modeText := strings.ToLower(args[1])
		var (
			mode     int8
			modeName string
		)
		switch modeText {
		case "0", "s", "survival":
			mode = 0
			modeName = "Survival Mode"
		case "1", "c", "creative":
			mode = 1
			modeName = "Creative Mode"
		case "2", "a", "adventure":
			mode = 2
			modeName = "Adventure Mode"
		default:
			s.sendSystemChat("Usage: /defaultgamemode <mode>")
			return true
		}

		s.server.defaultGameType.Store(int32(mode))
		s.sendSystemChat("The world's default game mode is now " + modeName)
		return true
	}

	if strings.EqualFold(args[0], "/gamemode") {
		if len(args) < 2 {
			s.sendSystemChat("Usage: /gamemode <0|1|2|survival|creative|adventure>")
			return true
		}

		modeText := strings.ToLower(args[1])
		var mode int8
		switch modeText {
		case "0", "s", "survival":
			mode = 0
		case "1", "c", "creative":
			mode = 1
		case "2", "a", "adventure":
			mode = 2
		default:
			s.sendSystemChat("Usage: /gamemode <0|1|2|survival|creative|adventure>")
			return true
		}

		target := s
		targetName := strings.TrimSpace(s.clientUsername)
		if len(args) >= 3 {
			found := s.server.activeSessionByUsername(args[2])
			if found == nil {
				s.sendSystemChat("Player not found.")
				return true
			}
			target = found
			if name := strings.TrimSpace(found.clientUsername); name != "" {
				targetName = name
			}
		}
		if targetName == "" {
			targetName = "player"
		}

		target.stateMu.Lock()
		target.gameType = mode
		target.playerFallDistance = 0
		if mode != 1 {
			target.playerIsFlying = false
		}
		target.stateMu.Unlock()
		if !target.sendPacket(target.currentAbilitiesPacket()) {
			return false
		}
		modeName := "Survival Mode"
		switch mode {
		case 1:
			modeName = "Creative Mode"
		case 2:
			modeName = "Adventure Mode"
		}
		if target == s {
			s.sendSystemChat("Set own game mode to " + modeName)
		} else {
			s.sendSystemChat("Set " + targetName + " game mode to " + modeName)
		}
		return true
	}

	if strings.EqualFold(args[0], "/xp") {
		if len(args) < 2 || len(args) > 3 {
			s.sendSystemChat("Usage: /xp <amount>[L] [player]")
			return true
		}

		amountText := strings.TrimSpace(args[1])
		if amountText == "" {
			s.sendSystemChat("Usage: /xp <amount>[L] [player]")
			return true
		}

		levelMode := strings.HasSuffix(amountText, "l") || strings.HasSuffix(amountText, "L")
		if levelMode && len(amountText) > 1 {
			amountText = amountText[:len(amountText)-1]
		}
		amount, err := strconv.Atoi(amountText)
		if err != nil {
			s.sendSystemChat("Usage: /xp <amount>[L] [player]")
			return true
		}

		negative := amount < 0
		if negative {
			amount *= -1
		}
		if amount < 0 {
			amount = 0
		}

		target := s
		targetName := s.clientUsername
		if len(args) == 3 {
			found := s.server.activeSessionByUsername(args[2])
			if found == nil {
				s.sendSystemChat("Player not found.")
				return true
			}
			target = found
			if name := strings.TrimSpace(found.clientUsername); name != "" {
				targetName = name
			}
		}
		if targetName == "" {
			targetName = "player"
		}

		if levelMode {
			if negative {
				target.addExperienceLevel(-int32(amount))
				s.sendSystemChat("Removed " + strconv.Itoa(amount) + " levels from " + targetName)
			} else {
				target.addExperienceLevel(int32(amount))
				s.sendSystemChat("Given " + strconv.Itoa(amount) + " levels to " + targetName)
			}
			return target.sendExperienceState()
		}

		if negative {
			s.sendSystemChat("Cannot withdraw experience points.")
			return true
		}

		target.addExperience(int32(amount))
		s.sendSystemChat("Given " + strconv.Itoa(amount) + " experience to " + targetName)
		return target.sendExperienceState()
	}

	if strings.EqualFold(args[0], "/clear") {
		if len(args) > 4 {
			s.sendSystemChat("Usage: /clear [player] [itemId] [damage]")
			return true
		}

		target := s
		targetName := strings.TrimSpace(s.clientUsername)
		if len(args) >= 2 {
			found := s.server.activeSessionByUsername(args[1])
			if found == nil {
				s.sendSystemChat("Player not found.")
				return true
			}
			target = found
			if name := strings.TrimSpace(found.clientUsername); name != "" {
				targetName = name
			}
		}
		if targetName == "" {
			targetName = "player"
		}

		itemID := int16(-1)
		if len(args) >= 3 {
			item, err := strconv.Atoi(args[2])
			if err != nil || item < 1 || item > 32767 {
				s.sendSystemChat("Invalid item id")
				return true
			}
			itemID = int16(item)
		}

		damage := int16(-1)
		if len(args) >= 4 {
			meta, err := strconv.Atoi(args[3])
			if err != nil || meta < 0 || meta > 32767 {
				s.sendSystemChat("Invalid damage")
				return true
			}
			damage = int16(meta)
		}

		removed, changedSlots, cursorChanged := target.clearInventoryFiltered(itemID, damage)
		if removed <= 0 {
			s.sendSystemChat("No items were found to clear from " + targetName)
			return true
		}
		for slot := 0; slot < playerWindowSlots; slot++ {
			if !changedSlots[slot] {
				continue
			}
			if !target.sendInventorySetSlot(slot) {
				return false
			}
		}
		if cursorChanged {
			if !target.sendCursorSetSlot(nil) {
				return false
			}
		}
		s.sendSystemChat("Cleared " + strconv.Itoa(removed) + " items from " + targetName)
		return true
	}

	if strings.EqualFold(args[0], "/kill") {
		if !s.killPlayer() {
			return false
		}
		s.sendSystemChat("Ouch. You died.")
		return true
	}

	if strings.EqualFold(args[0], "/setblock") {
		if len(args) < 5 || len(args) > 6 {
			s.sendSystemChat("Usage: /setblock <x> <y> <z> <id> [meta]")
			return true
		}

		x, err := strconv.Atoi(args[1])
		if err != nil {
			s.sendSystemChat("Invalid X coordinate")
			return true
		}
		y, err := strconv.Atoi(args[2])
		if err != nil {
			s.sendSystemChat("Invalid Y coordinate")
			return true
		}
		z, err := strconv.Atoi(args[3])
		if err != nil {
			s.sendSystemChat("Invalid Z coordinate")
			return true
		}
		if y < 0 || y >= buildLimit {
			s.sendSystemChat("Y must be between 0 and 255")
			return true
		}
		if math.Abs(float64(x)) > maxWorldCoordinate || math.Abs(float64(z)) > maxWorldCoordinate {
			s.sendSystemChat("Coordinates out of world bounds")
			return true
		}

		blockID, err := strconv.Atoi(args[4])
		if err != nil || blockID < 0 || blockID > maxPlaceableBlock {
			s.sendSystemChat("Invalid block id")
			return true
		}

		meta := 0
		if len(args) == 6 {
			meta, err = strconv.Atoi(args[5])
			if err != nil || meta < 0 || meta > 15 {
				s.sendSystemChat("Invalid block metadata")
				return true
			}
		}

		if s.server.world.setBlock(x, y, z, blockID, meta) {
			s.server.broadcastBlockChange(int32(x), int32(y), int32(z), int32(blockID), int32(meta))
		}
		s.sendSystemChat("Set block at " + strconv.Itoa(x) + " " + strconv.Itoa(y) + " " + strconv.Itoa(z))
		return true
	}

	s.sendSystemChat("Unknown command.")
	return true
}

func (s *loginSession) sendSystemChat(text string) {
	_ = s.sendPacket(protocol.NewPacket3Chat(text, true))
}

func parseTeleportCoordinate(token string, current float64) (float64, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, fmt.Errorf("empty coordinate")
	}
	if strings.HasPrefix(token, "~") {
		if len(token) == 1 {
			return current, nil
		}
		off, err := strconv.ParseFloat(token[1:], 64)
		if err != nil {
			return 0, err
		}
		return current + off, nil
	}
	return strconv.ParseFloat(token, 64)
}

func normalizeGameTypeValue(v int8) int8 {
	switch v {
	case 1, 2:
		return v
	default:
		return 0
	}
}

func (s *loginSession) currentAbilitiesPacket() *protocol.Packet202PlayerAbilities {
	s.stateMu.Lock()
	gameType := normalizeGameTypeValue(s.gameType)
	s.gameType = gameType
	if gameType != 1 {
		s.playerIsFlying = false
	}
	isFlying := s.playerIsFlying
	s.stateMu.Unlock()

	isCreative := gameType == 1
	return &protocol.Packet202PlayerAbilities{
		DisableDamage: isCreative,
		IsFlying:      isFlying && isCreative,
		AllowFlying:   isCreative,
		IsCreative:    isCreative,
		FlySpeed:      0.05,
		WalkSpeed:     0.1,
	}
}

func cloneItemStack(stack *protocol.ItemStack) *protocol.ItemStack {
	if stack == nil {
		return nil
	}
	out := *stack
	return &out
}

func intFromNBTNumericTag(tag nbt.Tag) (int, bool) {
	switch t := tag.(type) {
	case *nbt.ByteTag:
		return int(t.Data), true
	case *nbt.ShortTag:
		return int(t.Data), true
	case *nbt.IntTag:
		return int(t.Data), true
	case *nbt.LongTag:
		return int(t.Data), true
	default:
		return 0, false
	}
}

func lootingLevelFromItemStack(stack *protocol.ItemStack) int {
	if stack == nil || stack.Tag == nil {
		return 0
	}
	listTag, ok := stack.Tag.GetTag("ench").(*nbt.ListTag)
	if !ok || listTag == nil {
		return 0
	}
	maxLevel := 0
	for i := 0; i < listTag.TagCount(); i++ {
		entry, ok := listTag.TagAt(i).(*nbt.CompoundTag)
		if !ok || entry == nil {
			continue
		}
		enchID, okID := intFromNBTNumericTag(entry.GetTag("id"))
		if !okID || enchID != enchantIDLooting {
			continue
		}
		level, okLevel := intFromNBTNumericTag(entry.GetTag("lvl"))
		if !okLevel || level <= 0 {
			continue
		}
		if level > maxLevel {
			maxLevel = level
		}
	}
	return maxLevel
}

func toInventorySlice(arr [playerWindowSlots]*protocol.ItemStack) []*protocol.ItemStack {
	out := make([]*protocol.ItemStack, playerWindowSlots)
	for i := 0; i < playerWindowSlots; i++ {
		out[i] = cloneItemStack(arr[i])
	}
	return out
}

func (s *loginSession) snapshotInventoryWindow() []*protocol.ItemStack {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	out := make([]*protocol.ItemStack, playerWindowSlots)
	for i := 0; i < playerWindowSlots; i++ {
		out[i] = cloneItemStack(s.inventory[i])
	}
	return out
}

func (s *loginSession) heldWindowSlotLocked() int {
	slot := int(s.heldItemSlot)
	if slot < 0 || slot >= hotbarSlotCount {
		slot = 0
		s.heldItemSlot = 0
	}
	return hotbarBaseSlot + slot
}

func (s *loginSession) sendInventorySetSlot(windowSlot int) bool {
	if windowSlot < 0 || windowSlot >= playerWindowSlots {
		return true
	}
	s.stateMu.Lock()
	stack := cloneItemStack(s.inventory[windowSlot])
	s.stateMu.Unlock()
	return s.sendPacket(&protocol.Packet103SetSlot{
		WindowID:  0,
		ItemSlot:  int16(windowSlot),
		ItemStack: stack,
	})
}

func (s *loginSession) sendCursorSetSlot(stack *protocol.ItemStack) bool {
	return s.sendPacket(&protocol.Packet103SetSlot{
		WindowID:  -1,
		ItemSlot:  -1,
		ItemStack: cloneItemStack(stack),
	})
}

func (s *loginSession) clearInventoryFiltered(itemID, damage int16) (int, [playerWindowSlots]bool, bool) {
	var changedSlots [playerWindowSlots]bool
	removed := 0
	cursorChanged := false

	s.stateMu.Lock()
	for i := 0; i < playerWindowSlots; i++ {
		stack := s.inventory[i]
		if stack == nil || stack.StackSize <= 0 {
			continue
		}
		if itemID >= 0 && stack.ItemID != itemID {
			continue
		}
		if damage >= 0 && stack.ItemDamage != damage {
			continue
		}
		removed += int(stack.StackSize)
		s.inventory[i] = nil
		changedSlots[i] = true
	}
	if s.cursorItem != nil && s.cursorItem.StackSize > 0 {
		if (itemID < 0 || s.cursorItem.ItemID == itemID) && (damage < 0 || s.cursorItem.ItemDamage == damage) {
			removed += int(s.cursorItem.StackSize)
			s.cursorItem = nil
			cursorChanged = true
		}
	}
	s.stateMu.Unlock()
	return removed, changedSlots, cursorChanged
}

func (s *loginSession) addInventoryItem(itemID int16, count int, damage int16) int {
	if count <= 0 {
		return 0
	}
	remaining := count
	changed := make(map[int]struct{})
	insertOrder := make([]int, 0, hotbarSlotCount+27)
	for i := 0; i < hotbarSlotCount; i++ {
		insertOrder = append(insertOrder, hotbarBaseSlot+i)
	}
	for i := 9; i <= 35; i++ {
		insertOrder = append(insertOrder, i)
	}

	s.stateMu.Lock()
	for _, i := range insertOrder {
		if remaining <= 0 {
			break
		}
		stack := s.inventory[i]
		if stack == nil {
			continue
		}
		if stack.ItemID != itemID || stack.ItemDamage != damage || stack.StackSize <= 0 || stack.StackSize >= 64 {
			continue
		}
		space := 64 - int(stack.StackSize)
		add := space
		if add > remaining {
			add = remaining
		}
		stack.StackSize += int8(add)
		remaining -= add
		changed[i] = struct{}{}
	}

	for _, i := range insertOrder {
		if remaining <= 0 {
			break
		}
		if s.inventory[i] != nil {
			continue
		}
		add := 64
		if add > remaining {
			add = remaining
		}
		s.inventory[i] = &protocol.ItemStack{
			ItemID:     itemID,
			StackSize:  int8(add),
			ItemDamage: damage,
		}
		remaining -= add
		changed[i] = struct{}{}
	}
	s.stateMu.Unlock()

	for i := range changed {
		if !s.sendInventorySetSlot(i) {
			return remaining
		}
	}
	return remaining
}

func (s *loginSession) hasInventoryItemLocked(itemID int16) bool {
	for i := 0; i < playerWindowSlots; i++ {
		stack := s.inventory[i]
		if stack == nil || stack.ItemID != itemID || stack.StackSize <= 0 {
			continue
		}
		return true
	}
	return false
}

func (s *loginSession) consumeInventoryItemLocked(itemID int16, count int) []int {
	if count <= 0 {
		return nil
	}
	remaining := count
	changed := make([]int, 0, 2)

	for i := 0; i < playerWindowSlots && remaining > 0; i++ {
		stack := s.inventory[i]
		if stack == nil || stack.ItemID != itemID || stack.StackSize <= 0 {
			continue
		}
		take := int(stack.StackSize)
		if take > remaining {
			take = remaining
		}
		stack.StackSize -= int8(take)
		remaining -= take
		if stack.StackSize <= 0 {
			s.inventory[i] = nil
		}
		changed = append(changed, i)
	}
	return changed
}

func sameItemType(a, b *protocol.ItemStack) bool {
	if a == nil || b == nil {
		return false
	}
	return a.ItemID == b.ItemID && a.ItemDamage == b.ItemDamage
}

func (s *loginSession) moveStackToRangesLocked(stack *protocol.ItemStack, ranges [][2]int, changedSlots map[int]struct{}) *protocol.ItemStack {
	if stack == nil || stack.StackSize <= 0 {
		return nil
	}
	remaining := cloneItemStack(stack)

	for _, r := range ranges {
		start := r[0]
		end := r[1]
		if start < 0 {
			start = 0
		}
		if end >= playerWindowSlots {
			end = playerWindowSlots - 1
		}
		for i := start; i <= end && remaining.StackSize > 0; i++ {
			target := s.inventory[i]
			if target == nil || !sameItemType(target, remaining) || target.StackSize >= 64 {
				continue
			}

			space := 64 - int(target.StackSize)
			move := int(remaining.StackSize)
			if move > space {
				move = space
			}
			target.StackSize += int8(move)
			remaining.StackSize -= int8(move)
			changedSlots[i] = struct{}{}
		}
	}

	for _, r := range ranges {
		start := r[0]
		end := r[1]
		if start < 0 {
			start = 0
		}
		if end >= playerWindowSlots {
			end = playerWindowSlots - 1
		}
		for i := start; i <= end && remaining.StackSize > 0; i++ {
			if s.inventory[i] != nil {
				continue
			}
			move := int(remaining.StackSize)
			if move > 64 {
				move = 64
			}
			s.inventory[i] = cloneItemStack(remaining)
			s.inventory[i].StackSize = int8(move)
			remaining.StackSize -= int8(move)
			changedSlots[i] = struct{}{}
		}
	}

	if remaining.StackSize <= 0 {
		return nil
	}
	return remaining
}

func (s *loginSession) applyWindowClickLocked(packet *protocol.Packet102WindowClick, changedSlots map[int]struct{}) (accepted bool, dropped []*protocol.ItemStack) {
	if packet == nil || packet.WindowID != 0 {
		return false, nil
	}

	mode := packet.Mode
	if mode == 0 && packet.HoldingShift {
		mode = 1
	}

	// Translation reference:
	// - net.minecraft.src.Container#slotClick(..., clickMode=4) drop-from-slot path.
	if mode == 4 {
		slot := int(packet.InventorySlot)
		if slot < 0 || slot >= playerWindowSlots {
			return false, nil
		}
		source := s.inventory[slot]
		if source == nil || source.StackSize <= 0 {
			return true, nil
		}
		if packet.MouseClick != 0 && packet.MouseClick != 1 {
			return false, nil
		}

		remove := int8(1)
		if packet.MouseClick == 1 {
			remove = source.StackSize
		}
		if remove <= 0 {
			remove = 1
		}
		if remove > source.StackSize {
			remove = source.StackSize
		}

		drop := cloneItemStack(source)
		drop.StackSize = remove
		if source.StackSize <= remove {
			s.inventory[slot] = nil
		} else {
			source.StackSize -= remove
		}
		changedSlots[slot] = struct{}{}
		return true, []*protocol.ItemStack{drop}
	}

	if mode == 1 {
		slot := int(packet.InventorySlot)
		if slot < 0 || slot >= playerWindowSlots {
			return false, nil
		}

		source := s.inventory[slot]
		if source == nil || source.StackSize <= 0 {
			return true, nil
		}

		var ranges [][2]int
		switch {
		case slot >= 9 && slot <= 35:
			ranges = [][2]int{{36, 44}}
		case slot >= 36 && slot <= 44:
			ranges = [][2]int{{9, 35}}
		default:
			ranges = [][2]int{{9, 44}}
		}

		original := source.StackSize
		remaining := s.moveStackToRangesLocked(source, ranges, changedSlots)
		if remaining == nil {
			s.inventory[slot] = nil
			changedSlots[slot] = struct{}{}
			return true, nil
		}
		if remaining.StackSize != original {
			s.inventory[slot].StackSize = remaining.StackSize
			changedSlots[slot] = struct{}{}
		}
		return true, nil
	}

	if mode != 0 {
		return false, nil
	}

	if packet.MouseClick != 0 && packet.MouseClick != 1 {
		return false, nil
	}

	if packet.InventorySlot == -999 {
		if s.cursorItem == nil {
			return true, nil
		}
		if packet.MouseClick == 0 {
			s.cursorItem = nil
			return true, nil
		}
		s.cursorItem.StackSize--
		if s.cursorItem.StackSize <= 0 {
			s.cursorItem = nil
		}
		return true, nil
	}

	slot := int(packet.InventorySlot)
	if slot < 0 || slot >= playerWindowSlots {
		return false, nil
	}

	slotStack := s.inventory[slot]
	cursor := s.cursorItem

	if packet.MouseClick == 0 {
		switch {
		case cursor == nil:
			s.cursorItem = cloneItemStack(slotStack)
			s.inventory[slot] = nil
		case slotStack == nil:
			s.inventory[slot] = cloneItemStack(cursor)
			s.cursorItem = nil
		case sameItemType(cursor, slotStack) && slotStack.StackSize < 64:
			space := 64 - int(slotStack.StackSize)
			move := int(cursor.StackSize)
			if move > space {
				move = space
			}
			slotStack.StackSize += int8(move)
			cursor.StackSize -= int8(move)
			if cursor.StackSize <= 0 {
				s.cursorItem = nil
			}
		default:
			s.inventory[slot] = cloneItemStack(cursor)
			s.cursorItem = cloneItemStack(slotStack)
		}
		changedSlots[slot] = struct{}{}
		return true, nil
	}

	switch {
	case cursor == nil:
		if slotStack == nil {
			return true, nil
		}
		take := int(slotStack.StackSize+1) / 2
		s.cursorItem = cloneItemStack(slotStack)
		s.cursorItem.StackSize = int8(take)
		slotStack.StackSize -= int8(take)
		if slotStack.StackSize <= 0 {
			s.inventory[slot] = nil
		}
	case slotStack == nil:
		s.inventory[slot] = cloneItemStack(cursor)
		s.inventory[slot].StackSize = 1
		cursor.StackSize--
		if cursor.StackSize <= 0 {
			s.cursorItem = nil
		}
	case sameItemType(cursor, slotStack) && slotStack.StackSize < 64:
		slotStack.StackSize++
		cursor.StackSize--
		if cursor.StackSize <= 0 {
			s.cursorItem = nil
		}
	default:
		s.inventory[slot] = cloneItemStack(cursor)
		s.cursorItem = cloneItemStack(slotStack)
	}
	changedSlots[slot] = struct{}{}
	return true, nil
}

func (s *loginSession) handleFlying(packet *protocol.Packet10Flying) bool {
	if packet == nil {
		return true
	}
	s.stateMu.Lock()
	if s.playerDead {
		s.stateMu.Unlock()
		return true
	}
	mounted := s.ridingEntityID != 0
	s.stateMu.Unlock()

	if mounted {
		// Translation reference:
		// - net.minecraft.src.NetServerHandler#handleFlying(Packet10Flying)
		// ridingEntity != null branch:
		//   skip normal movement validation and preserve rider world position.
		var movementPacket protocol.Packet
		var headRotationPacket protocol.Packet
		var movementChunkX int32
		var movementChunkZ int32

		s.stateMu.Lock()
		if packet.Rotating {
			s.playerYaw = packet.Yaw
			s.playerPitch = packet.Pitch
		}
		s.playerOnGround = packet.OnGround
		movementChunkX = chunkCoordFromPos(s.playerX)
		movementChunkZ = chunkCoordFromPos(s.playerZ)

		if s.playerRegistered {
			yaw := toPacketAngle(s.playerYaw)
			pitch := toPacketAngle(s.playerPitch)
			rotChanged := yaw != s.lastEntityYaw || pitch != s.lastEntityPitch
			if rotChanged {
				pkt := protocol.NewPacket32EntityLook()
				pkt.EntityID = s.entityID
				pkt.Yaw = yaw
				pkt.Pitch = pitch
				movementPacket = pkt
				s.lastEntityYaw = yaw
				s.lastEntityPitch = pitch
			}
			if absInt(int(yaw)-int(s.lastHeadYaw)) >= 4 {
				headRotationPacket = &protocol.Packet35EntityHeadRotation{
					EntityID:        s.entityID,
					HeadRotationYaw: yaw,
				}
				s.lastHeadYaw = yaw
			}
		}
		s.stateMu.Unlock()

		if movementPacket != nil || headRotationPacket != nil {
			s.updateTrackedViewers(movementChunkX, movementChunkZ, movementPacket, headRotationPacket)
		}
		return true
	}

	if packet.Moving {
		stanceDelta := packet.Stance - packet.YPosition
		if stanceDelta > 1.65 || stanceDelta < 0.1 {
			s.disconnect("Illegal stance")
			return false
		}
		if math.Abs(packet.XPosition) > maxWorldCoordinate || math.Abs(packet.ZPosition) > maxWorldCoordinate {
			s.disconnect("Illegal position")
			return false
		}

		s.stateMu.Lock()
		prevX := s.playerX
		prevY := s.playerY
		prevZ := s.playerZ
		prevYaw := s.playerYaw
		prevPitch := s.playerPitch
		s.stateMu.Unlock()

		deltaX := packet.XPosition - prevX
		deltaY := packet.YPosition - prevY
		deltaZ := packet.ZPosition - prevZ
		if deltaX*deltaX+deltaY*deltaY+deltaZ*deltaZ > 100.0 {
			return s.setPlayerLocation(prevX, prevY, prevZ, prevYaw, prevPitch, false)
		}
	}

	var unload []chunk.CoordIntPair
	var movementPacket protocol.Packet
	var headRotationPacket protocol.Packet
	var movementChunkX int32
	var movementChunkZ int32
	viewWindowChanged := false
	fallDamage := 0
	outOfWorldDamage := false

	s.stateMu.Lock()
	prevOnGround := s.playerOnGround
	prevX := s.playerX
	prevY := s.playerY
	prevZ := s.playerZ
	gameType := s.gameType
	if packet.Moving {
		s.playerX = packet.XPosition
		s.playerY = packet.YPosition
		s.playerZ = packet.ZPosition
		s.playerStance = packet.Stance
	}
	if packet.Rotating {
		s.playerYaw = packet.Yaw
		s.playerPitch = packet.Pitch
	}
	s.playerOnGround = packet.OnGround
	if packet.OnGround {
		if !prevOnGround {
			damage := int(math.Ceil(float64(s.playerFallDistance - 3.0)))
			if gameType != 1 && damage > 0 {
				fallDamage = damage
			}
		}
		s.playerFallDistance = 0
	} else if packet.Moving {
		deltaY := s.playerY - prevY
		if deltaY < 0 {
			s.playerFallDistance += float32(-deltaY)
		}
		if prevOnGround && deltaY > 0 {
			// Translated from EntityPlayer#jump exhaustion costs (walk jump 0.2F, sprint jump 0.8F).
			if s.playerSprinting {
				s.addFoodExhaustionLocked(0.8)
			} else {
				s.addFoodExhaustionLocked(0.2)
			}
		}
	}
	if packet.Moving && packet.OnGround {
		// Translated subset from EntityPlayer#addMovementStat on-ground branch.
		deltaX := s.playerX - prevX
		deltaZ := s.playerZ - prevZ
		distanceCM := int(math.Round(math.Sqrt(deltaX*deltaX+deltaZ*deltaZ) * 100.0))
		if distanceCM > 0 {
			if s.playerSprinting {
				s.addFoodExhaustionLocked(0.099999994 * float32(distanceCM) * 0.01)
			} else {
				s.addFoodExhaustionLocked(0.01 * float32(distanceCM) * 0.01)
			}
		}
	}
	if gameType != 1 && s.playerY < -64.0 {
		// Translated from Entity#onEntityUpdate: position below -64 triggers out-of-world damage.
		outOfWorldDamage = true
	}

	newChunkX := chunkCoordFromPos(s.playerX)
	newChunkZ := chunkCoordFromPos(s.playerZ)
	if newChunkX != s.managedChunkX || newChunkZ != s.managedChunkZ {
		unload = s.rebuildChunkQueuesLocked(newChunkX, newChunkZ, s.server.cfg.ViewDistance)
		viewWindowChanged = true
	}
	movementChunkX = newChunkX
	movementChunkZ = newChunkZ

	if s.playerRegistered {
		posX := toPacketPosition(s.playerX)
		posY := toPacketPosition(s.playerY)
		posZ := toPacketPosition(s.playerZ)
		yaw := toPacketAngle(s.playerYaw)
		pitch := toPacketAngle(s.playerPitch)

		deltaX := posX - s.lastEntityPosX
		deltaY := posY - s.lastEntityPosY
		deltaZ := posZ - s.lastEntityPosZ
		moveChanged := deltaX != 0 || deltaY != 0 || deltaZ != 0
		rotChanged := yaw != s.lastEntityYaw || pitch != s.lastEntityPitch

		if moveChanged || rotChanged {
			if deltaX >= -128 && deltaX <= 127 && deltaY >= -128 && deltaY <= 127 && deltaZ >= -128 && deltaZ <= 127 {
				switch {
				case moveChanged && rotChanged:
					pkt := protocol.NewPacket33RelEntityMoveLook()
					pkt.EntityID = s.entityID
					pkt.XPosition = int8(deltaX)
					pkt.YPosition = int8(deltaY)
					pkt.ZPosition = int8(deltaZ)
					pkt.Yaw = yaw
					pkt.Pitch = pitch
					movementPacket = pkt
				case moveChanged:
					pkt := &protocol.Packet31RelEntityMove{}
					pkt.EntityID = s.entityID
					pkt.XPosition = int8(deltaX)
					pkt.YPosition = int8(deltaY)
					pkt.ZPosition = int8(deltaZ)
					movementPacket = pkt
				case rotChanged:
					pkt := protocol.NewPacket32EntityLook()
					pkt.EntityID = s.entityID
					pkt.Yaw = yaw
					pkt.Pitch = pitch
					movementPacket = pkt
				}
			} else {
				movementPacket = &protocol.Packet34EntityTeleport{
					EntityID:  s.entityID,
					XPosition: posX,
					YPosition: posY,
					ZPosition: posZ,
					Yaw:       yaw,
					Pitch:     pitch,
				}
			}

			s.lastEntityPosX = posX
			s.lastEntityPosY = posY
			s.lastEntityPosZ = posZ
			s.lastEntityYaw = yaw
			s.lastEntityPitch = pitch
		}
		if absInt(int(yaw)-int(s.lastHeadYaw)) >= 4 {
			headRotationPacket = &protocol.Packet35EntityHeadRotation{
				EntityID:        s.entityID,
				HeadRotationYaw: yaw,
			}
			s.lastHeadYaw = yaw
		}
	}
	s.stateMu.Unlock()

	for _, pos := range unload {
		ch := s.server.world.getChunk(pos.ChunkXPos, pos.ChunkZPos)
		pkt, err := protocol.NewPacket51MapChunk(ch, true, 0, false)
		if err != nil {
			continue
		}
		if !s.sendPacket(pkt) {
			return false
		}
	}
	if viewWindowChanged {
		s.refreshVisibleEntitiesForSelf()
	}
	if movementPacket != nil || headRotationPacket != nil {
		s.updateTrackedViewers(movementChunkX, movementChunkZ, movementPacket, headRotationPacket)
	}
	if fallDamage > 0 {
		s.applyNonPlayerDamage(float32(fallDamage))
	}
	if outOfWorldDamage {
		s.applyNonPlayerDamage(4.0)
	}
	return true
}

func (s *loginSession) setPlayerLocation(x, y, z float64, yaw, pitch float32, broadcast bool) bool {
	const eyeHeight = 1.6200000047683716

	var movementChunkX int32
	var movementChunkZ int32
	var teleportPacket protocol.Packet
	var headPacket protocol.Packet

	s.stateMu.Lock()
	s.playerX = x
	s.playerY = y
	s.playerZ = z
	s.playerStance = y + eyeHeight
	s.playerYaw = yaw
	s.playerPitch = pitch
	s.playerFallDistance = 0
	newChunkX := chunkCoordFromPos(x)
	newChunkZ := chunkCoordFromPos(z)
	if newChunkX != s.managedChunkX || newChunkZ != s.managedChunkZ {
		_ = s.rebuildChunkQueuesLocked(newChunkX, newChunkZ, s.server.cfg.ViewDistance)
	}
	movementChunkX = newChunkX
	movementChunkZ = newChunkZ
	if s.playerRegistered && broadcast {
		posX := toPacketPosition(x)
		posY := toPacketPosition(y)
		posZ := toPacketPosition(z)
		yawByte := toPacketAngle(yaw)
		pitchByte := toPacketAngle(pitch)
		teleportPacket = &protocol.Packet34EntityTeleport{
			EntityID:  s.entityID,
			XPosition: posX,
			YPosition: posY,
			ZPosition: posZ,
			Yaw:       yawByte,
			Pitch:     pitchByte,
		}
		if absInt(int(yawByte)-int(s.lastHeadYaw)) >= 4 {
			headPacket = &protocol.Packet35EntityHeadRotation{
				EntityID:        s.entityID,
				HeadRotationYaw: yawByte,
			}
		}
		s.lastEntityPosX = posX
		s.lastEntityPosY = posY
		s.lastEntityPosZ = posZ
		s.lastEntityYaw = yawByte
		s.lastEntityPitch = pitchByte
		s.lastHeadYaw = yawByte
	}
	s.stateMu.Unlock()

	move := protocol.NewPacket13PlayerLookMove()
	move.XPosition = x
	move.YPosition = y + eyeHeight
	move.Stance = y
	move.ZPosition = z
	move.Yaw = yaw
	move.Pitch = pitch
	move.OnGround = false
	if !s.sendPacket(move) {
		return false
	}
	if teleportPacket != nil || headPacket != nil {
		s.updateTrackedViewers(movementChunkX, movementChunkZ, teleportPacket, headPacket)
	}
	return true
}

func (s *loginSession) handleHeldItemSwitch(packet *protocol.Packet16BlockItemSwitch) {
	if packet == nil {
		return
	}
	if packet.ID < 0 || packet.ID >= hotbarSlotCount {
		return
	}
	changed := false
	s.stateMu.Lock()
	if s.playerUsingItem {
		s.playerUsingItem = false
		s.playerItemUseCount = 0
		changed = true
	}
	s.heldItemSlot = packet.ID
	s.stateMu.Unlock()
	if changed {
		s.broadcastOwnEntityMetadata()
	}
}

func (s *loginSession) handleCreativeSetSlot(packet *protocol.Packet107CreativeSetSlot) {
	if packet == nil {
		return
	}

	s.stateMu.Lock()
	isCreative := s.gameType == 1
	s.stateMu.Unlock()
	if !isCreative {
		return
	}

	var stack *protocol.ItemStack
	if packet.ItemStack != nil {
		stack = cloneItemStack(packet.ItemStack)
	}

	if packet.Slot < 0 {
		// Translation references:
		// - net.minecraft.src.NetServerHandler#handleCreativeSetSlot(Packet107CreativeSetSlot)
		//   creative drop path (slot < 0) with spam throttle and creative despawn age.
		if stack == nil || stack.ItemID <= 0 || stack.ItemDamage < 0 || stack.StackSize <= 0 || stack.StackSize > 64 {
			return
		}

		allowDrop := false
		s.stateMu.Lock()
		if s.creativeItemCreationSpamThresholdTally < 200 {
			s.creativeItemCreationSpamThresholdTally += 20
			allowDrop = true
		}
		s.stateMu.Unlock()
		if !allowDrop {
			return
		}
		s.server.spawnDroppedItemFromPlayer(s, stack, false, true)
		return
	}
	slot := int(packet.Slot)
	if slot < 0 || slot >= playerWindowSlots {
		return
	}

	if stack != nil {
		if stack.StackSize < 1 {
			stack = nil
		} else if stack.StackSize > 64 {
			stack.StackSize = 64
		}
	}

	s.stateMu.Lock()
	s.inventory[slot] = stack
	s.stateMu.Unlock()
	_ = s.sendInventorySetSlot(slot)
}

func (s *loginSession) handleClientInfo(packet *protocol.Packet204ClientInfo) {
	if packet == nil {
		return
	}
	s.stateMu.Lock()
	s.clientLocale = packet.Language
	s.clientRenderDistance = packet.RenderDistance
	s.stateMu.Unlock()
}

func (s *loginSession) handlePlayerAbilities(packet *protocol.Packet202PlayerAbilities) {
	if packet == nil {
		return
	}

	// Translated from NetServerHandler#handlePlayerAbilities:
	// player.capabilities.isFlying = packet.getFlying() && player.capabilities.allowFlying.
	s.stateMu.Lock()
	allowFlying := normalizeGameTypeValue(s.gameType) == 1
	s.playerIsFlying = packet.IsFlying && allowFlying
	s.stateMu.Unlock()
}

func (s *loginSession) handleCloseWindow(packet *protocol.Packet101CloseWindow) {
	if packet == nil {
		return
	}
	if packet.WindowID != 0 {
		return
	}

	s.stateMu.Lock()
	hadCursor := s.cursorItem != nil
	s.cursorItem = nil
	s.stateMu.Unlock()
	if hadCursor {
		_ = s.sendCursorSetSlot(nil)
	}
}

func (s *loginSession) handleWindowClick(packet *protocol.Packet102WindowClick) bool {
	if packet == nil {
		return true
	}

	accepted := false
	changed := make(map[int]struct{})
	var droppedStacks []*protocol.ItemStack
	var cursorSnapshot *protocol.ItemStack
	var inventorySnapshot []*protocol.ItemStack

	s.stateMu.Lock()
	accepted, droppedStacks = s.applyWindowClickLocked(packet, changed)
	cursorSnapshot = cloneItemStack(s.cursorItem)
	if !accepted && packet.WindowID == 0 {
		inventorySnapshot = make([]*protocol.ItemStack, playerWindowSlots)
		for i := 0; i < playerWindowSlots; i++ {
			inventorySnapshot[i] = cloneItemStack(s.inventory[i])
		}
	}
	s.stateMu.Unlock()

	if !s.sendPacket(&protocol.Packet106Transaction{
		WindowID:     packet.WindowID,
		ActionNumber: packet.ActionNumber,
		Accepted:     accepted,
	}) {
		return false
	}

	if accepted {
		for slot := range changed {
			if !s.sendInventorySetSlot(slot) {
				return false
			}
		}
		if !s.sendCursorSetSlot(cursorSnapshot) {
			return false
		}
		for _, dropped := range droppedStacks {
			if dropped == nil || dropped.ItemID <= 0 || dropped.StackSize <= 0 {
				continue
			}
			s.server.spawnDroppedItemFromPlayer(s, dropped, false, false)
		}
		return true
	}

	if packet.WindowID == 0 {
		if !s.sendPacket(&protocol.Packet104WindowItems{
			WindowID:   0,
			ItemStacks: inventorySnapshot,
		}) {
			return false
		}
		return s.sendCursorSetSlot(cursorSnapshot)
	}

	return true
}

func (s *loginSession) handleBlockDig(packet *protocol.Packet14BlockDig) bool {
	if packet == nil {
		return true
	}

	if packet.Status == 4 || packet.Status == 3 {
		// Translation reference:
		// - net.minecraft.src.NetServerHandler.handleBlockDig(Packet14BlockDig)
		//   status 4 -> dropOneItem(false), status 3 -> dropOneItem(true)
		slot, dropped, changed := s.dropHeldItemStack(packet.Status == 3)
		if changed {
			_ = s.sendInventorySetSlot(slot)
		}
		if dropped != nil {
			s.server.spawnDroppedItemFromPlayer(s, dropped, false, false)
		}
		return true
	}

	if packet.Status == 5 {
		s.stopUsingHeldItem(true)
		return true
	}

	if packet.Status != 0 && packet.Status != 1 && packet.Status != 2 {
		return true
	}

	if packet.YPosition >= buildLimit {
		return true
	}
	if !s.isWithinReach(packet.XPosition, packet.YPosition, packet.ZPosition, 36.0) {
		return true
	}

	if packet.Status == 2 {
		s.stateMu.Lock()
		adventureMode := s.gameType == 2
		s.stateMu.Unlock()
		if adventureMode {
			return s.sendBlockChange(packet.XPosition, packet.YPosition, packet.ZPosition)
		}

		blockID, _ := s.server.world.getBlock(int(packet.XPosition), int(packet.YPosition), int(packet.ZPosition))
		if blockID != 0 {
			if s.server.world.setBlock(int(packet.XPosition), int(packet.YPosition), int(packet.ZPosition), 0, 0) {
				s.server.broadcastBlockChange(packet.XPosition, packet.YPosition, packet.ZPosition, 0, 0)
			}
		}
	}

	return s.sendBlockChange(packet.XPosition, packet.YPosition, packet.ZPosition)
}

func (s *loginSession) dropHeldItemStack(dropAll bool) (windowSlot int, dropped *protocol.ItemStack, changed bool) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	windowSlot = s.heldWindowSlotLocked()
	stack := s.inventory[windowSlot]
	if stack == nil || stack.StackSize <= 0 {
		return windowSlot, nil, false
	}

	remove := int8(1)
	if dropAll {
		remove = stack.StackSize
	}
	if remove <= 0 {
		remove = 1
	}

	dropped = cloneItemStack(stack)
	dropped.StackSize = remove

	if stack.StackSize <= remove {
		s.inventory[windowSlot] = nil
	} else {
		stack.StackSize -= remove
	}
	return windowSlot, dropped, true
}

func (s *loginSession) handlePlace(packet *protocol.Packet15Place) bool {
	if packet == nil {
		return true
	}

	x := packet.XPosition
	y := packet.YPosition
	z := packet.ZPosition
	direction := packet.Direction
	shouldResync := false

	if direction == 255 {
		s.tryUseHeldItemInAir(packet)
		return true
	}

	if y >= buildLimit-1 && (direction == 1 || y >= buildLimit) {
		s.sendSystemChat("Height limit for building is 256 blocks")
		shouldResync = true
	} else {
		if s.isWithinReach(x, y, z, 64.0) {
			s.stateMu.Lock()
			adventureMode := s.gameType == 2
			s.stateMu.Unlock()
			if !adventureMode {
				s.tryPlaceBlockFromItem(packet)
			}
		}
		shouldResync = true
	}

	if !shouldResync {
		return true
	}

	if !s.sendBlockChange(x, y, z) {
		return false
	}

	adjX, adjY, adjZ := offsetByDirection(x, y, z, direction)
	return s.sendBlockChange(adjX, adjY, adjZ)
}

func meleeProfileForItemID(itemID int16) (meleeItemProfile, bool) {
	profile, ok := meleeItemProfiles[itemID]
	return profile, ok
}

func resolveUseItemProfile(itemID int16, itemDamage int16) (useItemProfile, bool) {
	if melee, ok := meleeProfileForItemID(itemID); ok && melee.SupportsBlocking && melee.MaxUseDuration > 0 {
		return useItemProfile{
			Kind:           useItemKindBlock,
			MaxUseDuration: melee.MaxUseDuration,
		}, true
	}
	if itemID == itemIDBow {
		return useItemProfile{
			Kind:           useItemKindBow,
			MaxUseDuration: 72000,
		}, true
	}
	if itemID == itemIDBucketMilk {
		return useItemProfile{
			Kind:           useItemKindMilkDrink,
			MaxUseDuration: 32,
		}, true
	}
	if itemID == itemIDPotion {
		if (itemDamage & 16384) != 0 {
			// Splash potions are thrown immediately in vanilla (non-duration use path).
			return useItemProfile{}, false
		}
		return useItemProfile{
			Kind:           useItemKindPotionDrink,
			MaxUseDuration: 32,
		}, true
	}
	if food, ok := foodItemProfiles[itemID]; ok {
		return food, true
	}
	return useItemProfile{}, false
}

func (s *loginSession) tryUseHeldItemInAir(packet *protocol.Packet15Place) {
	if packet == nil {
		return
	}

	var (
		heldSlot  int
		stack     *protocol.ItemStack
		gameType  int8
		foodLevel int16
	)
	s.stateMu.Lock()
	heldSlot = s.heldWindowSlotLocked()
	stack = cloneItemStack(s.inventory[heldSlot])
	gameType = s.gameType
	foodLevel = s.playerFood
	s.stateMu.Unlock()
	if stack == nil {
		return
	}

	if packet.ItemStack != nil && (packet.ItemStack.ItemID != stack.ItemID || packet.ItemStack.ItemDamage != stack.ItemDamage) {
		_ = s.sendInventorySetSlot(heldSlot)
		return
	}

	profile, ok := resolveUseItemProfile(stack.ItemID, stack.ItemDamage)
	if !ok || profile.MaxUseDuration <= 0 {
		return
	}

	if profile.Kind == useItemKindBow && gameType != 1 {
		s.stateMu.Lock()
		hasArrow := s.hasInventoryItemLocked(itemIDArrow)
		s.stateMu.Unlock()
		if !hasArrow {
			return
		}
	}
	if profile.Kind == useItemKindFood && gameType == 1 {
		// Translated from EntityPlayer#canEat: creative (disableDamage) cannot consume food.
		return
	}
	if profile.Kind == useItemKindFood && !profile.AlwaysEdible && foodLevel >= 20 {
		return
	}

	changed := false
	s.stateMu.Lock()
	if !s.playerUsingItem || s.playerItemUseCount != profile.MaxUseDuration {
		s.playerUsingItem = true
		s.playerItemUseCount = profile.MaxUseDuration
		changed = true
	}
	s.stateMu.Unlock()
	if changed {
		s.broadcastOwnEntityMetadata()
		if profile.Kind == useItemKindFood && s.entityID != 0 && s.playerRegistered {
			// Translated from EntityPlayerMP#setItemInUse: send eating animation start (id=5) to watchers.
			chunkX, chunkZ := s.currentChunkCoords()
			s.server.broadcastEntityPacketToWatchers(&protocol.Packet18Animation{
				EntityID:  s.entityID,
				AnimateID: 5,
			}, chunkX, chunkZ, s)
		}
	}
}

func (s *loginSession) stopUsingHeldItem(releaseAction bool) {
	var (
		metaChanged      bool
		selfChanged      = make(map[int]struct{})
		spawnBowVelocity float64
		spawnBowCritical bool
	)

	s.stateMu.Lock()
	if !s.playerUsingItem {
		s.stateMu.Unlock()
		return
	}

	heldSlot := s.heldWindowSlotLocked()
	stack := s.inventory[heldSlot]
	useCount := s.playerItemUseCount
	isCreative := s.gameType == 1

	if releaseAction && stack != nil {
		if profile, ok := resolveUseItemProfile(stack.ItemID, stack.ItemDamage); ok && profile.Kind == useItemKindBow {
			// Translated subset from ItemBow#onPlayerStoppedUsing:
			// draw strength gate + arrow requirement + bow durability/arrow consumption.
			hasArrow := isCreative || s.hasInventoryItemLocked(itemIDArrow)
			if hasArrow {
				useTicks := profile.MaxUseDuration - useCount
				draw := float64(useTicks) / 20.0
				draw = (draw*draw + draw*2.0) / 3.0
				if draw >= 0.1 {
					if draw > 1.0 {
						draw = 1.0
					}
					spawnBowVelocity = draw * 2.0
					spawnBowCritical = draw == 1.0
					if !isCreative {
						for _, slot := range s.consumeInventoryItemLocked(itemIDArrow, 1) {
							selfChanged[slot] = struct{}{}
						}
					}
					stack.ItemDamage++
					selfChanged[heldSlot] = struct{}{}
					if stack.ItemDamage >= bowMaxDurability {
						stack.StackSize--
						if stack.StackSize <= 0 {
							s.inventory[heldSlot] = nil
						} else {
							stack.ItemDamage = 0
						}
					}
				}
			}
		}
	}

	s.playerUsingItem = false
	s.playerItemUseCount = 0
	metaChanged = true
	s.stateMu.Unlock()

	if spawnBowVelocity > 0 {
		s.server.spawnArrowFromPlayer(s, spawnBowVelocity, spawnBowCritical)
	}
	if metaChanged {
		s.broadcastOwnEntityMetadata()
	}
	for slot := range selfChanged {
		_ = s.sendInventorySetSlot(slot)
	}
}

func (s *loginSession) tickHeldItemUse() {
	var (
		finished      bool
		profile       useItemProfile
		heldSlot      int
		slotChanged   bool
		healthChanged bool
		metaChanged   bool
		finishStatus  bool
		entityID      int32
	)

	s.stateMu.Lock()
	if !s.playerUsingItem {
		s.stateMu.Unlock()
		return
	}

	heldSlot = s.heldWindowSlotLocked()
	stack := s.inventory[heldSlot]
	if stack == nil || stack.StackSize <= 0 {
		s.playerUsingItem = false
		s.playerItemUseCount = 0
		metaChanged = true
		s.stateMu.Unlock()
		if metaChanged {
			s.broadcastOwnEntityMetadata()
		}
		return
	}

	var ok bool
	profile, ok = resolveUseItemProfile(stack.ItemID, stack.ItemDamage)
	if !ok || profile.MaxUseDuration <= 0 {
		s.playerUsingItem = false
		s.playerItemUseCount = 0
		metaChanged = true
		s.stateMu.Unlock()
		if metaChanged {
			s.broadcastOwnEntityMetadata()
		}
		return
	}

	if s.playerItemUseCount <= 0 {
		s.playerItemUseCount = profile.MaxUseDuration
	}
	s.playerItemUseCount--
	if s.playerItemUseCount > 0 {
		s.stateMu.Unlock()
		return
	}

	finished = true
	s.playerUsingItem = false
	s.playerItemUseCount = 0
	metaChanged = true
	entityID = s.entityID
	finishStatus = entityID != 0

	isCreative := s.gameType == 1
	switch profile.Kind {
	case useItemKindFood:
		if !isCreative {
			stack.StackSize--
			slotChanged = true
		}
		s.applyFoodStatsLocked(profile.FoodHealAmount, profile.FoodSaturation)
		healthChanged = true
		if stack.StackSize <= 0 {
			if stack.ItemID == itemIDMushroomStew {
				s.inventory[heldSlot] = &protocol.ItemStack{
					ItemID:     itemIDBowlEmpty,
					StackSize:  1,
					ItemDamage: 0,
				}
			} else {
				s.inventory[heldSlot] = nil
			}
			slotChanged = true
		}
	case useItemKindPotionDrink:
		if !isCreative {
			stack.StackSize--
			slotChanged = true
			if stack.StackSize <= 0 {
				s.inventory[heldSlot] = &protocol.ItemStack{
					ItemID:     itemIDGlassBottle,
					StackSize:  1,
					ItemDamage: 0,
				}
			}
		}
	case useItemKindMilkDrink:
		if !isCreative {
			stack.StackSize--
			slotChanged = true
			if stack.StackSize <= 0 {
				s.inventory[heldSlot] = &protocol.ItemStack{
					ItemID:     itemIDBucketEmpty,
					StackSize:  1,
					ItemDamage: 0,
				}
			}
		}
	}
	s.stateMu.Unlock()

	if !finished {
		return
	}
	if finishStatus {
		// Translated from EntityPlayerMP#onItemUseFinish: status id=9 to self.
		_ = s.sendPacket(&protocol.Packet38EntityStatus{
			EntityID:     entityID,
			EntityStatus: 9,
		})
	}
	if metaChanged {
		s.broadcastOwnEntityMetadata()
	}
	if slotChanged {
		_ = s.sendInventorySetSlot(heldSlot)
	}
	if healthChanged {
		_ = s.sendHealthState()
	}
}

func (s *loginSession) applyFoodStatsLocked(healAmount int16, saturationModifier float32) {
	if healAmount <= 0 {
		return
	}
	newFood := s.playerFood + healAmount
	if newFood > 20 {
		newFood = 20
	}
	s.playerFood = newFood

	s.playerSat = s.playerSat + float32(healAmount)*saturationModifier*2.0
	if s.playerSat > float32(s.playerFood) {
		s.playerSat = float32(s.playerFood)
	}
	if s.playerSat < 0 {
		s.playerSat = 0
	}
}

func (s *loginSession) addFoodExhaustionLocked(amount float32) {
	if amount <= 0 {
		return
	}
	if s.gameType == 1 {
		// Translated from EntityPlayer#addExhaustion: creative skips FoodStats exhaustion updates.
		return
	}
	s.playerFoodExhaust += amount
	if s.playerFoodExhaust > 40 {
		s.playerFoodExhaust = 40
	}
}

func (s *loginSession) tickFoodStats() {
	var (
		sendHealth    bool
		starveAttempt bool
	)
	difficulty := s.server.currentDifficulty()

	s.stateMu.Lock()
	if s.playerDead || s.playerHealth <= 0 {
		s.stateMu.Unlock()
		return
	}

	prevHealth := s.playerHealth
	prevFood := s.playerFood
	prevSat := s.playerSat

	// Translation target:
	// - net.minecraft.src.FoodStats#onUpdate(EntityPlayer)
	s.playerPrevFood = s.playerFood
	if s.playerFoodExhaust > 4.0 {
		s.playerFoodExhaust -= 4.0
		if s.playerSat > 0.0 {
			s.playerSat = float32(math.Max(float64(s.playerSat-1.0), 0.0))
		} else if difficulty > 0 {
			if s.playerFood > 0 {
				s.playerFood--
			}
		}
	}

	shouldHeal := s.playerHealth > 0 && s.playerHealth < maxPlayerHealth
	if naturalRegenerationGamerule && s.playerFood >= 18 && shouldHeal {
		s.playerFoodTimer++
		if s.playerFoodTimer >= 80 {
			s.playerHealth += 1.0
			if s.playerHealth > maxPlayerHealth {
				s.playerHealth = maxPlayerHealth
			}
			s.addFoodExhaustionLocked(3.0)
			s.playerFoodTimer = 0
		}
	} else if s.playerFood <= 0 {
		s.playerFoodTimer++
		if s.playerFoodTimer >= 80 {
			if s.playerHealth > 10.0 || difficulty >= 3 || (s.playerHealth > 1.0 && difficulty >= 2) {
				starveAttempt = true
			}
			s.playerFoodTimer = 0
		}
	} else {
		s.playerFoodTimer = 0
	}

	sendHealth = s.playerHealth != prevHealth || s.playerFood != prevFood || s.playerSat != prevSat
	s.stateMu.Unlock()

	if starveAttempt {
		s.applyNonPlayerDamage(1.0)
		return
	}
	if sendHealth {
		_ = s.sendHealthState()
	}
}

func (s *loginSession) applyNonPlayerDamage(amount float32) {
	damaged, died, hurtStatus := s.applyIncomingDamage(amount, false)
	if damaged && hurtStatus {
		s.broadcastEntityStatusToSelfAndWatchers(2)
	}
	if died {
		s.broadcastEntityStatusToSelfAndWatchers(3)
		s.sendSystemChat("You died.")
	}
}

func (s *loginSession) positionSnapshot() (float64, float64, float64) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.playerX, s.playerY, s.playerZ
}

func (s *loginSession) positionRotationSnapshot() (float64, float64, float64, float32, float32) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.playerX, s.playerY, s.playerZ, s.playerYaw, s.playerPitch
}

func (s *loginSession) handleAnimation(packet *protocol.Packet18Animation) {
	if packet == nil || packet.AnimateID != 1 {
		return
	}
	if s.entityID == 0 || !s.playerRegistered {
		return
	}

	// Translated from: net.minecraft.src.NetServerHandler#handleAnimation(Packet18Animation)
	// EntityPlayer.swingItem() side effects are reduced to watcher animation broadcast in this baseline.
	chunkX, chunkZ := s.currentChunkCoords()
	s.server.broadcastEntityPacketToWatchers(&protocol.Packet18Animation{
		EntityID:  s.entityID,
		AnimateID: 1,
	}, chunkX, chunkZ, s)
}

func (s *loginSession) handleEntityAction(packet *protocol.Packet19EntityAction) {
	if packet == nil {
		return
	}

	// Translated from: net.minecraft.src.NetServerHandler#handleEntityAction(Packet19EntityAction)
	changed := false
	shouldDismount := false
	s.stateMu.Lock()
	switch packet.Action {
	case 1:
		if !s.playerSneaking {
			s.playerSneaking = true
			changed = true
		}
		if s.ridingEntityID != 0 {
			shouldDismount = true
		}
	case 2:
		if s.playerSneaking {
			s.playerSneaking = false
			changed = true
		}
	case 4:
		if !s.playerSprinting {
			s.playerSprinting = true
			changed = true
		}
	case 5:
		if s.playerSprinting {
			s.playerSprinting = false
			changed = true
		}
	}
	s.stateMu.Unlock()
	if shouldDismount {
		s.dismountRidingEntity()
	}
	if changed {
		s.broadcastOwnEntityMetadata()
	}
}

func (s *loginSession) handlePlayerInput(packet *protocol.Packet27PlayerInput) {
	if packet == nil {
		return
	}

	// Translation reference:
	// - net.minecraft.src.NetServerHandler#func_110774_a(Packet27PlayerInput)
	// - net.minecraft.src.EntityPlayerMP#setEntityActionState(float,float,boolean,boolean)
	s.stateMu.Lock()
	if packet.MoveStrafing >= -1.0 && packet.MoveStrafing <= 1.0 {
		s.playerInputStrafe = packet.MoveStrafing
	}
	if packet.MoveForward >= -1.0 && packet.MoveForward <= 1.0 {
		s.playerInputForward = packet.MoveForward
	}
	s.playerInputJump = packet.Jump
	s.playerInputSneak = packet.Sneak
	s.stateMu.Unlock()
}

func (s *loginSession) handleUseEntity(packet *protocol.Packet7UseEntity) bool {
	if packet == nil {
		return true
	}

	s.stateMu.Lock()
	if s.playerDead {
		s.stateMu.Unlock()
		return true
	}
	s.stateMu.Unlock()

	// Translated from: net.minecraft.src.NetServerHandler#handleUseEntity(Packet7UseEntity)
	selfX, selfY, selfZ := s.positionSnapshot()
	targetPlayer := s.server.activeSessionByEntityID(packet.TargetEntityID)
	var targetMob *trackedMob
	if targetPlayer == nil {
		targetMob = s.server.mobByEntityID(packet.TargetEntityID)
		if targetMob == nil {
			if packet.Action == 1 && s.isInvalidAttackTargetEntity(packet.TargetEntityID) {
				s.disconnect("Attempting to attack an invalid entity")
				return false
			}
			return true
		}
	}

	var targetX, targetY, targetZ float64
	if targetPlayer != nil {
		targetX, targetY, targetZ = targetPlayer.positionSnapshot()
	} else {
		targetX, targetY, targetZ = targetMob.X, targetMob.Y, targetMob.Z
	}

	dx := selfX - targetX
	dy := selfY - targetY
	dz := selfZ - targetZ
	targetAimY := defaultPlayerEyeY
	if targetMob != nil {
		height, _ := mobCollisionSize(targetMob)
		targetAimY = height * 0.5
	}
	maxDistanceSq := 36.0
	if !s.server.hasLineOfSight(selfX, selfY+defaultPlayerEyeY, selfZ, targetX, targetY+targetAimY, targetZ) {
		maxDistanceSq = 9.0
	}
	if dx*dx+dy*dy+dz*dz >= maxDistanceSq {
		return true
	}

	switch packet.Action {
	case 0:
		if targetMob != nil {
			s.interactWithMob(targetMob)
		}
		return true
	case 1:
		if targetPlayer == s {
			s.disconnect("Attempting to attack an invalid entity")
			return false
		}
		if targetPlayer != nil {
			// Translated subset from:
			// - net.minecraft.src.EntityPlayer#attackTargetEntityWithCurrentItem(Entity)
			// - net.minecraft.src.EntityLivingBase#attackEntityFrom(DamageSource,float)
			s.attackTargetPlayerWithCurrentItem(targetPlayer)
			return true
		}
		s.attackTargetMobWithCurrentItem(targetMob)
		return true
	default:
		return true
	}
}

func (s *loginSession) isInvalidAttackTargetEntity(entityID int32) bool {
	if entityID == 0 {
		return false
	}

	s.stateMu.Lock()
	selfEntityID := s.entityID
	s.stateMu.Unlock()
	if entityID == selfEntityID {
		return true
	}

	s.server.droppedItemMu.Lock()
	_, isDropped := s.server.droppedItems[entityID]
	s.server.droppedItemMu.Unlock()
	if isDropped {
		return true
	}

	s.server.projectileMu.Lock()
	projectile := s.server.projectiles[entityID]
	s.server.projectileMu.Unlock()
	if projectile != nil && projectile.Type == entityTypeArrow {
		return true
	}

	return false
}

func (s *loginSession) attackTargetMobWithCurrentItem(target *trackedMob) {
	if target == nil {
		return
	}

	s.stateMu.Lock()
	if s.playerDead {
		s.stateMu.Unlock()
		return
	}
	damage := float32(basePlayerDamage)
	lootingLevel := 0
	knockbackLevel := 0
	attackerYaw := s.playerYaw
	// Translated subset from EntityPlayer#attackTargetEntityWithCurrentItem critical gate:
	// fallDistance > 0 && !onGround && ridingEntity == null
	critical := s.playerFallDistance > 0 && !s.playerOnGround && s.ridingEntityID == 0
	heldSlot := s.heldWindowSlotLocked()
	heldStack := cloneItemStack(s.inventory[heldSlot])
	if heldStack != nil {
		if profile, ok := meleeProfileForItemID(heldStack.ItemID); ok {
			damage += profile.AttackModifier
		}
		lootingLevel = lootingLevelFromItemStack(heldStack)
	}
	if critical && damage > 0 {
		damage *= 1.5
	}
	if s.playerSprinting {
		knockbackLevel++
	}
	s.stateMu.Unlock()

	mob, damaged, died, hurtStatus := s.server.applyDamageToMob(target.EntityID, damage)
	if !damaged || mob == nil {
		return
	}
	if hurtStatus {
		s.server.broadcastMobEntityStatus(mob, 2)
	}
	if knockbackLevel > 0 {
		// Translation reference:
		// - net.minecraft.src.EntityPlayer#attackTargetEntityWithCurrentItem(Entity)
		// target.addVelocity(-sin(yaw)*k*0.5, 0.1, cos(yaw)*k*0.5)
		s.server.applyKnockbackToMob(mob, attackerYaw, knockbackLevel)
		changed := false
		s.stateMu.Lock()
		if s.playerSprinting {
			s.playerSprinting = false
			changed = true
		}
		s.stateMu.Unlock()
		if changed {
			s.broadcastOwnEntityMetadata()
		}
	}

	s.applyHeldMeleeDurabilityAfterHit()
	s.stateMu.Lock()
	s.addFoodExhaustionLocked(0.3)
	s.stateMu.Unlock()

	if died {
		s.server.broadcastMobEntityStatus(mob, 3)
		s.server.killMob(mob, true, lootingLevel)
	}
}

func (s *loginSession) interactWithMob(target *trackedMob) bool {
	if target == nil {
		return false
	}

	handled := false
	switch target.EntityType {
	case entityTypePig:
		handled = s.interactWithPig(target)
	case entityTypeCow:
		handled = s.interactWithCow(target)
	case entityTypeSheep:
		handled = s.interactWithSheep(target)
	}
	if handled {
		return true
	}
	return s.interactItemWithMob(target)
}

func (s *loginSession) interactWithPig(target *trackedMob) bool {
	if target == nil {
		return false
	}

	// Translation reference:
	// - net.minecraft.src.EntityPig#interact(EntityPlayer)
	// Minimal riding attach parity: saddled pig interaction emits Packet39 attach.
	var (
		pigEntityID int32
		playerID    int32
		chunkX      int32
		chunkZ      int32
	)
	s.server.mobMu.Lock()
	live := s.server.mobs[target.EntityID]
	saddled := live != nil && live.EntityType == entityTypePig && live.pigSaddled
	if saddled {
		pigEntityID = live.EntityID
		chunkX, chunkZ = live.chunkCoords()
	}
	s.server.mobMu.Unlock()
	if !saddled {
		return false
	}

	s.stateMu.Lock()
	playerID = s.entityID
	alreadyMounted := s.ridingEntityID == pigEntityID
	s.ridingEntityID = pigEntityID
	s.stateMu.Unlock()
	if alreadyMounted {
		return true
	}

	attach := &protocol.Packet39AttachEntity{
		RidingEntityID:  playerID,
		VehicleEntityID: pigEntityID,
		AttachState:     0,
	}
	_ = s.sendPacket(attach)
	s.server.broadcastEntityPacketToWatchers(attach, chunkX, chunkZ, s)
	return true
}

func (s *loginSession) dismountRidingEntity() {
	var (
		vehicleID int32
		playerID  int32
		chunkX    int32
		chunkZ    int32
	)

	s.stateMu.Lock()
	vehicleID = s.ridingEntityID
	playerID = s.entityID
	s.ridingEntityID = 0
	s.stateMu.Unlock()

	if vehicleID == 0 || playerID == 0 {
		return
	}
	chunkX, chunkZ = s.currentChunkCoords()
	if mob := s.server.mobByEntityID(vehicleID); mob != nil {
		chunkX, chunkZ = mob.chunkCoords()
	}

	detach := &protocol.Packet39AttachEntity{
		RidingEntityID:  playerID,
		VehicleEntityID: -1,
		AttachState:     0,
	}
	_ = s.sendPacket(detach)
	s.server.broadcastEntityPacketToWatchers(detach, chunkX, chunkZ, s)
}

func (s *loginSession) interactWithCow(target *trackedMob) bool {
	// Translation reference:
	// - net.minecraft.src.EntityCow#interact(EntityPlayer)
	var (
		heldSlot   int
		consumeOne bool
		isCreative bool
	)
	s.stateMu.Lock()
	heldSlot = s.heldWindowSlotLocked()
	stack := s.inventory[heldSlot]
	if stack == nil || stack.StackSize <= 0 || stack.ItemID != itemIDBucketEmpty {
		s.stateMu.Unlock()
		return false
	}
	isCreative = s.gameType == 1
	if isCreative {
		s.stateMu.Unlock()
		return false
	}
	if stack.StackSize == 1 {
		s.inventory[heldSlot] = &protocol.ItemStack{
			ItemID:     itemIDBucketMilk,
			StackSize:  1,
			ItemDamage: 0,
		}
	} else {
		stack.StackSize--
		consumeOne = true
	}
	s.stateMu.Unlock()

	_ = s.sendInventorySetSlot(heldSlot)
	if !consumeOne {
		return true
	}

	remaining := s.addInventoryItem(itemIDBucketMilk, 1, 0)
	if remaining > 0 {
		s.server.spawnDroppedItemFromPlayer(s, &protocol.ItemStack{
			ItemID:     itemIDBucketMilk,
			StackSize:  int8(remaining),
			ItemDamage: 0,
		}, false, false)
	}
	return true
}

func (s *loginSession) interactWithSheep(target *trackedMob) bool {
	// Translation reference:
	// - net.minecraft.src.EntitySheep#interact(EntityPlayer)
	var (
		heldSlot   int
		isCreative bool
	)
	s.stateMu.Lock()
	heldSlot = s.heldWindowSlotLocked()
	stack := s.inventory[heldSlot]
	if stack == nil || stack.StackSize <= 0 || stack.ItemID != itemIDShears {
		s.stateMu.Unlock()
		return false
	}
	isCreative = s.gameType == 1
	s.stateMu.Unlock()

	var (
		live       *trackedMob
		woolColor  int16
		dropCount  int
		x          float64
		y          float64
		z          float64
		dropMotion [][3]float64
	)
	s.server.mobMu.Lock()
	live = s.server.mobs[target.EntityID]
	if live == nil || live.EntityType != entityTypeSheep || live.sheepSheared {
		s.server.mobMu.Unlock()
		return false
	}
	live.sheepSheared = true
	x = live.X
	y = live.Y
	z = live.Z
	woolColor = int16(live.sheepColor & 15)
	dropCount = 1 + int(s.server.mobRand.NextInt(3))
	dropMotion = make([][3]float64, dropCount)
	for i := 0; i < dropCount; i++ {
		dropMotion[i][1] = float64(s.server.mobRand.NextFloat()) * 0.05
		dropMotion[i][0] = float64(s.server.mobRand.NextFloat()-s.server.mobRand.NextFloat()) * 0.1
		dropMotion[i][2] = float64(s.server.mobRand.NextFloat()-s.server.mobRand.NextFloat()) * 0.1
	}
	s.server.mobMu.Unlock()

	slotChanged := false
	if !isCreative {
		s.stateMu.Lock()
		stack := s.inventory[heldSlot]
		if stack != nil && stack.StackSize > 0 && stack.ItemID == itemIDShears {
			stack.ItemDamage++
			slotChanged = true
			if stack.ItemDamage >= shearsMaxDurability {
				stack.StackSize--
				if stack.StackSize <= 0 {
					s.inventory[heldSlot] = nil
				} else {
					stack.ItemDamage = 0
				}
			}
		}
		s.stateMu.Unlock()
	}
	if slotChanged {
		_ = s.sendInventorySetSlot(heldSlot)
	}

	s.server.broadcastMobMetadata(live)
	s.server.broadcastEntityPacketToWatchers(&protocol.Packet62LevelSound{
		SoundName: "mob.sheep.shear",
		EffectX:   int32(math.Floor(x * 8.0)),
		EffectY:   int32(math.Floor(y * 8.0)),
		EffectZ:   int32(math.Floor(z * 8.0)),
		Volume:    1.0,
		Pitch:     63,
	}, chunkCoordFromPos(x), chunkCoordFromPos(z), nil)

	for i := 0; i < dropCount; i++ {
		m := dropMotion[i]
		s.server.spawnDroppedItemAt(&protocol.ItemStack{
			ItemID:     blockIDWool,
			StackSize:  1,
			ItemDamage: woolColor,
		}, x, y+1.0, z, m[0], m[1], m[2], entityDropPickupDelayTicks)
	}
	return true
}

func (s *loginSession) interactItemWithMob(target *trackedMob) bool {
	if target == nil {
		return false
	}

	// Translation reference:
	// - net.minecraft.src.EntityPlayer#interactWith(Entity)
	// - net.minecraft.src.ItemDye#itemInteractionForEntity
	// - net.minecraft.src.ItemSaddle#itemInteractionForEntity
	var (
		heldSlot   int
		isCreative bool
		itemID     int16
		itemMeta   int16
	)
	s.stateMu.Lock()
	heldSlot = s.heldWindowSlotLocked()
	stack := s.inventory[heldSlot]
	if stack == nil || stack.StackSize <= 0 {
		s.stateMu.Unlock()
		return false
	}
	itemID = stack.ItemID
	itemMeta = stack.ItemDamage
	isCreative = s.gameType == 1
	s.stateMu.Unlock()

	if itemID == itemIDSaddle && target.EntityType == entityTypePig {
		changed := false
		s.server.mobMu.Lock()
		live := s.server.mobs[target.EntityID]
		if live != nil && live.EntityType == entityTypePig && !live.pigSaddled {
			live.pigSaddled = true
			changed = true
		}
		s.server.mobMu.Unlock()
		if !changed {
			return false
		}

		if !isCreative {
			slotChanged := false
			s.stateMu.Lock()
			stack := s.inventory[heldSlot]
			if stack != nil && stack.StackSize > 0 && stack.ItemID == itemIDSaddle {
				stack.StackSize--
				slotChanged = true
				if stack.StackSize <= 0 {
					s.inventory[heldSlot] = nil
				}
			}
			s.stateMu.Unlock()
			if slotChanged {
				_ = s.sendInventorySetSlot(heldSlot)
			}
		}
		s.server.broadcastMobMetadata(target)
		return true
	}

	if itemID != itemIDDyePowder || target.EntityType != entityTypeSheep {
		return false
	}
	dyeMeta := int16(itemMeta & 15)
	targetColor := int8(^dyeMeta & 15)
	changed := false
	s.server.mobMu.Lock()
	live := s.server.mobs[target.EntityID]
	if live != nil && live.EntityType == entityTypeSheep {
		if !live.sheepSheared && live.sheepColor != targetColor {
			live.sheepColor = targetColor
			changed = true
		}
	}
	s.server.mobMu.Unlock()

	if !changed {
		// ItemDye returns true for sheep targets even when no state change.
		return true
	}

	if !isCreative {
		slotChanged := false
		s.stateMu.Lock()
		stack := s.inventory[heldSlot]
		if stack != nil && stack.StackSize > 0 && stack.ItemID == itemIDDyePowder && int16(stack.ItemDamage&15) == dyeMeta {
			stack.StackSize--
			slotChanged = true
			if stack.StackSize <= 0 {
				s.inventory[heldSlot] = nil
			}
		}
		s.stateMu.Unlock()
		if slotChanged {
			_ = s.sendInventorySetSlot(heldSlot)
		}
	}

	s.server.broadcastMobMetadata(target)
	return true
}

func (s *loginSession) attackTargetPlayerWithCurrentItem(target *loginSession) {
	if target == nil {
		return
	}

	s.stateMu.Lock()
	if s.playerDead {
		s.stateMu.Unlock()
		return
	}
	damage := float32(basePlayerDamage)
	knockbackLevel := 0
	attackerYaw := s.playerYaw
	// Translated subset from EntityPlayer#attackTargetEntityWithCurrentItem critical gate:
	// fallDistance > 0 && !onGround && ridingEntity == null
	critical := s.playerFallDistance > 0 && !s.playerOnGround && s.ridingEntityID == 0
	heldSlot := s.heldWindowSlotLocked()
	heldStack := cloneItemStack(s.inventory[heldSlot])
	if heldStack != nil {
		if profile, ok := meleeProfileForItemID(heldStack.ItemID); ok {
			damage += profile.AttackModifier
		}
	}
	if critical && damage > 0 {
		// Translated subset from EntityPlayer#attackTargetEntityWithCurrentItem critical gate.
		// World checks (ladder/water/blindness) are pending until movement status parity is implemented.
		damage *= 1.5
	}
	if s.playerSprinting {
		knockbackLevel++
	}
	s.stateMu.Unlock()

	damaged, died, hurtStatus := target.applyIncomingPlayerDamage(damage)
	if damaged {
		if hurtStatus {
			target.broadcastEntityStatusToSelfAndWatchers(2)
		}
		if knockbackLevel > 0 {
			target.applyKnockbackFromYaw(attackerYaw, knockbackLevel)
			changed := false
			s.stateMu.Lock()
			if s.playerSprinting {
				s.playerSprinting = false
				changed = true
			}
			s.stateMu.Unlock()
			if changed {
				s.broadcastOwnEntityMetadata()
			}
		}
	}
	s.applyHeldMeleeDurabilityAfterHit()
	s.stateMu.Lock()
	s.addFoodExhaustionLocked(0.3)
	s.stateMu.Unlock()
	if !damaged {
		return
	}
	if died {
		target.broadcastEntityStatusToSelfAndWatchers(3)
		target.sendSystemChat("You died.")
	}
}

func (s *loginSession) applyIncomingPlayerDamage(incomingDamage float32) (damaged bool, died bool, hurtStatus bool) {
	return s.applyIncomingDamage(incomingDamage, true)
}

func (s *loginSession) applyIncomingDamage(incomingDamage float32, allowBlocking bool) (damaged bool, died bool, hurtStatus bool) {
	if incomingDamage <= 0 {
		return false, false, false
	}

	justDied := false
	didDamage := false
	sendHurtStatus := true

	s.stateMu.Lock()
	if s.playerDead || s.playerHealth <= 0 {
		s.stateMu.Unlock()
		return false, false, false
	}

	applyDamage := incomingDamage
	if float32(s.hurtResistantTime) > float32(maxHurtResistant)/2.0 {
		if incomingDamage <= s.lastDamage {
			s.stateMu.Unlock()
			return false, false, false
		}
		applyDamage = incomingDamage - s.lastDamage
		s.lastDamage = incomingDamage
		sendHurtStatus = false
	} else {
		s.lastDamage = incomingDamage
		s.hurtResistantTime = maxHurtResistant
		s.hurtTime = maxHurtTime
	}
	if allowBlocking && s.isBlockingWithHeldItemLocked() && applyDamage > 0 {
		// Translated from EntityPlayer#damageEntity: blocking halves incoming damage.
		applyDamage = (1.0 + applyDamage) * 0.5
	}

	if applyDamage > 0 {
		s.playerHealth -= applyDamage
		if s.playerHealth <= 0 {
			s.playerHealth = 0
			s.playerDead = true
			s.playerUsingItem = false
			s.playerItemUseCount = 0
			s.hurtTime = 0
			s.hurtResistantTime = 0
			justDied = true
		}
		didDamage = true
	}
	s.stateMu.Unlock()

	if didDamage {
		_ = s.sendHealthState()
	}
	return didDamage, justDied, sendHurtStatus
}

func (s *loginSession) isBlockingWithHeldItemLocked() bool {
	if !s.playerUsingItem {
		return false
	}
	heldSlot := s.heldWindowSlotLocked()
	stack := s.inventory[heldSlot]
	if stack == nil {
		return false
	}
	profile, ok := meleeProfileForItemID(stack.ItemID)
	return ok && profile.SupportsBlocking
}

func (s *loginSession) applyHeldMeleeDurabilityAfterHit() {
	changed := false
	metaChanged := false
	slot := -1

	s.stateMu.Lock()
	if s.gameType != 1 {
		slot = s.heldWindowSlotLocked()
		stack := s.inventory[slot]
		if stack != nil {
			if profile, ok := meleeProfileForItemID(stack.ItemID); ok && profile.HitDamageCost > 0 && profile.MaxDurability > 0 {
				stack.ItemDamage += profile.HitDamageCost
				changed = true
				if stack.ItemDamage >= profile.MaxDurability {
					stack.StackSize--
					if stack.StackSize <= 0 {
						s.inventory[slot] = nil
					} else {
						stack.ItemDamage = 0
					}
				}
			}
		}
	}
	if s.playerUsingItem {
		heldSlot := s.heldWindowSlotLocked()
		if heldSlot >= 0 && heldSlot < playerWindowSlots && s.inventory[heldSlot] == nil {
			s.playerUsingItem = false
			metaChanged = true
		}
	}
	s.stateMu.Unlock()

	if changed && slot >= 0 {
		_ = s.sendInventorySetSlot(slot)
	}
	if metaChanged {
		s.broadcastOwnEntityMetadata()
	}
}

func (s *loginSession) broadcastEntityStatusToSelfAndWatchers(status int8) {
	if s.entityID == 0 || status == 0 {
		return
	}

	packet := &protocol.Packet38EntityStatus{
		EntityID:     s.entityID,
		EntityStatus: status,
	}
	_ = s.sendPacket(packet)
	chunkX, chunkZ := s.currentChunkCoords()
	s.server.broadcastEntityPacketToWatchers(packet, chunkX, chunkZ, s)
}

func (s *loginSession) applyKnockbackFromYaw(attackerYaw float32, knockbackLevel int) {
	if knockbackLevel <= 0 {
		return
	}

	s.stateMu.Lock()
	if s.playerDead {
		s.stateMu.Unlock()
		return
	}
	posX := s.playerX
	posY := s.playerY
	posZ := s.playerZ
	yaw := s.playerYaw
	pitch := s.playerPitch
	s.stateMu.Unlock()

	// Translated direction from EntityPlayer#attackTargetEntityWithCurrentItem:
	// target.addVelocity(-sin(yaw)*k*0.5, 0.1, cos(yaw)*k*0.5)
	yawRad := float64(attackerYaw) * math.Pi / 180.0
	dx := -math.Sin(yawRad) * float64(knockbackLevel) * 0.5
	dz := math.Cos(yawRad) * float64(knockbackLevel) * 0.5
	_ = s.setPlayerLocation(posX+dx, posY, posZ+dz, yaw, pitch, true)
}

func (s *loginSession) currentEntityFlagsByte() int8 {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	flags := int8(0)
	if s.playerSneaking {
		flags |= 1 << 1
	}
	if s.playerSprinting {
		flags |= 1 << 3
	}
	if s.playerUsingItem {
		flags |= 1 << 4
	}
	if s.playerUsingItem {
		flags |= 1 << 4
	}
	return flags
}

func (s *loginSession) broadcastOwnEntityMetadata() {
	if !s.playerRegistered || s.entityID == 0 {
		return
	}

	packet := &protocol.Packet40EntityMetadata{
		EntityID: s.entityID,
		Metadata: []protocol.WatchableObject{
			{
				ObjectType:  0,
				DataValueID: 0,
				Value:       s.currentEntityFlagsByte(),
			},
		},
	}
	chunkX, chunkZ := s.currentChunkCoords()
	s.server.broadcastEntityPacketToWatchers(packet, chunkX, chunkZ, s)
}

func (s *loginSession) sendBlockChange(x, y, z int32) bool {
	if y < 0 || y >= 256 {
		return true
	}
	blockID, metadata := s.server.world.getBlock(int(x), int(y), int(z))
	return s.sendPacket(&protocol.Packet53BlockChange{
		XPosition: x,
		YPosition: y,
		ZPosition: z,
		Type:      int32(blockID),
		Metadata:  int32(metadata),
	})
}

func (s *loginSession) tryPlaceBlockFromItem(packet *protocol.Packet15Place) {
	if packet == nil {
		return
	}

	var (
		heldSlot       int
		stack          *protocol.ItemStack
		isCreative     bool
		usingHeldStack bool
	)
	s.stateMu.Lock()
	heldSlot = s.heldWindowSlotLocked()
	stack = cloneItemStack(s.inventory[heldSlot])
	isCreative = s.gameType == 1
	s.stateMu.Unlock()
	usingHeldStack = stack != nil

	if stack == nil {
		if !isCreative {
			_ = s.sendInventorySetSlot(heldSlot)
			return
		}
		stack = cloneItemStack(packet.ItemStack)
	}
	if stack == nil || stack.StackSize <= 0 {
		return
	}

	if packet.ItemStack != nil && (packet.ItemStack.ItemID != stack.ItemID || packet.ItemStack.ItemDamage != stack.ItemDamage) {
		_ = s.sendInventorySetSlot(heldSlot)
		return
	}

	blockID := int(stack.ItemID)
	if blockID <= 0 || blockID > maxPlaceableBlock {
		return
	}

	direction := packet.Direction
	clickedBlockID, clickedMeta := s.server.world.getBlock(int(packet.XPosition), int(packet.YPosition), int(packet.ZPosition))
	targetX, targetY, targetZ := packet.XPosition, packet.YPosition, packet.ZPosition

	if clickedBlockID == 78 && (clickedMeta&7) < 1 {
		direction = 1
	} else if !isReplaceablePlacementBlock(clickedBlockID) {
		targetX, targetY, targetZ = offsetByDirection(packet.XPosition, packet.YPosition, packet.ZPosition, direction)
	}

	if targetY < 0 || targetY >= buildLimit {
		return
	}

	targetBlockID, _ := s.server.world.getBlock(int(targetX), int(targetY), int(targetZ))
	if targetBlockID != 0 && !isReplaceablePlacementBlock(targetBlockID) {
		return
	}
	if targetY == 255 && block.BlocksMovement(blockID) {
		return
	}

	metadata := int(stack.ItemDamage)
	if s.server.world.setBlock(int(targetX), int(targetY), int(targetZ), blockID, metadata) {
		s.server.broadcastBlockChange(targetX, targetY, targetZ, int32(blockID), int32(metadata))

		if !isCreative && usingHeldStack {
			s.stateMu.Lock()
			if inv := s.inventory[heldSlot]; inv != nil && inv.ItemID == stack.ItemID && inv.ItemDamage == stack.ItemDamage && inv.StackSize > 0 {
				inv.StackSize--
				if inv.StackSize <= 0 {
					s.inventory[heldSlot] = nil
				}
			}
			s.stateMu.Unlock()
			_ = s.sendInventorySetSlot(heldSlot)
		}
	}
}

func (s *loginSession) drainPendingChunks(max int) []chunk.CoordIntPair {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if max <= 0 || len(s.pendingChunks) == 0 {
		return nil
	}

	count := max
	if len(s.pendingChunks) < count {
		count = len(s.pendingChunks)
	}

	out := make([]chunk.CoordIntPair, count)
	copy(out, s.pendingChunks[:count])
	s.pendingChunks = s.pendingChunks[count:]
	for _, pos := range out {
		s.loadedChunks[pos] = struct{}{}
	}
	return out
}

func (s *loginSession) rebuildChunkQueuesLocked(centerChunkX, centerChunkZ int32, radius int) []chunk.CoordIntPair {
	desiredOrder := buildSpiralChunkOrder(centerChunkX, centerChunkZ, radius)
	desiredSet := make(map[chunk.CoordIntPair]struct{}, len(desiredOrder))
	for _, pos := range desiredOrder {
		desiredSet[pos] = struct{}{}
	}

	unload := make([]chunk.CoordIntPair, 0)
	for pos := range s.loadedChunks {
		if _, ok := desiredSet[pos]; !ok {
			unload = append(unload, pos)
			delete(s.loadedChunks, pos)
		}
	}

	s.pendingChunks = s.pendingChunks[:0]
	for _, pos := range desiredOrder {
		if _, ok := s.loadedChunks[pos]; ok {
			continue
		}
		s.pendingChunks = append(s.pendingChunks, pos)
	}

	s.managedChunkX = centerChunkX
	s.managedChunkZ = centerChunkZ
	return unload
}

func (s *loginSession) isWatchingChunk(chunkX, chunkZ int32) bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	_, ok := s.loadedChunks[chunk.NewCoordIntPair(chunkX, chunkZ)]
	return ok
}

func (s *loginSession) currentPing() int32 {
	return s.latencyMS.Load()
}

func (s *loginSession) markSeenBy(other *loginSession, seen bool) {
	if other == nil {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if seen {
		s.seenBy[other] = struct{}{}
	} else {
		delete(s.seenBy, other)
	}
}

func (s *loginSession) hasSeenBy(other *loginSession) bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	_, ok := s.seenBy[other]
	return ok
}

func (s *loginSession) snapshotSeenBy() []*loginSession {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	out := make([]*loginSession, 0, len(s.seenBy))
	for viewer := range s.seenBy {
		out = append(out, viewer)
	}
	return out
}

func (s *loginSession) currentChunkCoords() (int32, int32) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return chunkCoordFromPos(s.playerX), chunkCoordFromPos(s.playerZ)
}

func (s *loginSession) buildNamedEntitySpawnPacket() *protocol.Packet20NamedEntitySpawn {
	if s.entityID == 0 || s.clientUsername == "" {
		return nil
	}

	s.stateMu.Lock()
	x := s.playerX
	y := s.playerY
	z := s.playerZ
	yaw := s.playerYaw
	pitch := s.playerPitch
	flags := int8(0)
	if s.playerSneaking {
		flags |= 1 << 1
	}
	if s.playerSprinting {
		flags |= 1 << 3
	}
	currentItem := int16(0)
	heldWindowSlot := s.heldWindowSlotLocked()
	if heldWindowSlot >= 0 && heldWindowSlot < playerWindowSlots {
		if stack := s.inventory[heldWindowSlot]; stack != nil {
			currentItem = stack.ItemID
		}
	}
	s.stateMu.Unlock()

	return &protocol.Packet20NamedEntitySpawn{
		EntityID:    s.entityID,
		Name:        s.clientUsername,
		XPosition:   toPacketPosition(x),
		YPosition:   toPacketPosition(y),
		ZPosition:   toPacketPosition(z),
		Rotation:    toPacketAngle(yaw),
		Pitch:       toPacketAngle(pitch),
		CurrentItem: currentItem,
		Metadata: []protocol.WatchableObject{
			{
				ObjectType:  0,
				DataValueID: 0,
				Value:       flags,
			},
		},
	}
}

func (s *loginSession) buildAttachEntityPacket() *protocol.Packet39AttachEntity {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.entityID == 0 || s.ridingEntityID == 0 {
		return nil
	}
	return &protocol.Packet39AttachEntity{
		RidingEntityID:  s.entityID,
		VehicleEntityID: s.ridingEntityID,
		AttachState:     0,
	}
}

func (s *loginSession) updateTrackedViewers(entityChunkX, entityChunkZ int32, movementPacket protocol.Packet, headRotationPacket protocol.Packet) {
	if !s.playerRegistered || s.entityID == 0 {
		return
	}

	targets := s.server.activeSessionsExcept(s)
	spawnPacket := s.buildNamedEntitySpawnPacket()
	attachPacket := s.buildAttachEntityPacket()
	destroyPacket := &protocol.Packet29DestroyEntity{EntityIDs: []int32{s.entityID}}

	for _, target := range targets {
		shouldSee := target.isWatchingChunk(entityChunkX, entityChunkZ)
		wasSeen := s.hasSeenBy(target)

		switch {
		case shouldSee && !wasSeen:
			if spawnPacket != nil && target.sendPacket(spawnPacket) {
				if attachPacket != nil {
					_ = target.sendPacket(attachPacket)
				}
				s.markSeenBy(target, true)
			}
		case !shouldSee && wasSeen:
			if target.sendPacket(destroyPacket) {
				s.markSeenBy(target, false)
			}
		case shouldSee && wasSeen:
			if movementPacket != nil {
				_ = target.sendPacket(movementPacket)
			}
			if headRotationPacket != nil {
				_ = target.sendPacket(headRotationPacket)
			}
		}
	}
}

func (s *loginSession) refreshVisibleEntitiesForSelf() {
	if !s.playerRegistered {
		return
	}

	targets := s.server.activeSessionsExcept(s)
	for _, other := range targets {
		otherChunkX, otherChunkZ := other.currentChunkCoords()
		shouldSee := s.isWatchingChunk(otherChunkX, otherChunkZ)
		wasSeen := other.hasSeenBy(s)

		switch {
		case shouldSee && !wasSeen:
			if spawn := other.buildNamedEntitySpawnPacket(); spawn != nil {
				if s.sendPacket(spawn) {
					if attach := other.buildAttachEntityPacket(); attach != nil {
						_ = s.sendPacket(attach)
					}
					other.markSeenBy(s, true)
				}
			}
		case !shouldSee && wasSeen:
			if s.sendPacket(&protocol.Packet29DestroyEntity{
				EntityIDs: []int32{other.entityID},
			}) {
				other.markSeenBy(s, false)
			}
		}
	}
}

func buildSpiralChunkOrder(centerX, centerZ int32, radius int) []chunk.CoordIntPair {
	if radius < 0 {
		radius = 0
	}
	total := (radius*2 + 1) * (radius*2 + 1)
	out := make([]chunk.CoordIntPair, 0, total)
	out = append(out, chunk.NewCoordIntPair(centerX, centerZ))

	dirIndex := 0
	offsetX := 0
	offsetZ := 0

	for stepLen := 1; stepLen <= radius*2; stepLen++ {
		for rep := 0; rep < 2; rep++ {
			delta := xzDirectionsConst[dirIndex%4]
			dirIndex++
			for step := 0; step < stepLen; step++ {
				offsetX += delta[0]
				offsetZ += delta[1]
				if absInt(offsetX) <= radius && absInt(offsetZ) <= radius {
					out = append(out, chunk.NewCoordIntPair(centerX+int32(offsetX), centerZ+int32(offsetZ)))
				}
			}
		}
	}

	dirIndex %= 4
	for step := 0; step < radius*2; step++ {
		delta := xzDirectionsConst[dirIndex]
		offsetX += delta[0]
		offsetZ += delta[1]
		if absInt(offsetX) <= radius && absInt(offsetZ) <= radius {
			out = append(out, chunk.NewCoordIntPair(centerX+int32(offsetX), centerZ+int32(offsetZ)))
		}
	}

	return out
}

func chunkCoordFromPos(pos float64) int32 {
	// Translated from: net.minecraft.src.MathHelper.floor_double(pos) >> 4
	// Java cast truncates toward zero, but chunk mapping requires floor for negatives.
	return int32(int(math.Floor(pos)) >> 4)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func toPacketPosition(pos float64) int32 {
	return int32(math.Floor(pos * 32.0))
}

func toPacketAngle(angle float32) int8 {
	return int8(int(angle * 256.0 / 360.0))
}

func (s *loginSession) isWithinReach(blockX, blockY, blockZ int32, maxDistanceSq float64) bool {
	s.stateMu.Lock()
	px := s.playerX
	py := s.playerY
	pz := s.playerZ
	s.stateMu.Unlock()

	dx := px - (float64(blockX) + 0.5)
	dy := py - (float64(blockY) + 0.5) + 1.5
	dz := pz - (float64(blockZ) + 0.5)
	return dx*dx+dy*dy+dz*dz <= maxDistanceSq
}

func normalizeSpace(v string) string {
	return strings.Join(strings.Fields(v), " ")
}

func isAllowedChatCharacter(r rune) bool {
	return r != 167 && r >= 32 && r != 127
}

func offsetByDirection(x, y, z, direction int32) (int32, int32, int32) {
	switch direction {
	case 0:
		y--
	case 1:
		y++
	case 2:
		z--
	case 3:
		z++
	case 4:
		x--
	case 5:
		x++
	}
	return x, y, z
}

func isReplaceablePlacementBlock(blockID int) bool {
	switch blockID {
	case 31, 32, 78, 106:
		return true
	default:
		return false
	}
}
