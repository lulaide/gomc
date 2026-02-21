package server

import (
	"math"
	"time"

	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

const (
	mobSpawnRadiusChunks = 8
	mobSpawnPackAttempts = 3
	mobSpawnRetries      = 4
	mobSpawnMaxOffset    = 6
	mobSpawnMinDistSq    = 24.0 * 24.0
	mobSpawnFromWorldSq  = 576.0
)

const (
	mobWanderMinTicks   = 30
	mobWanderExtraTicks = 100
	mobWanderPauseMin   = 10
	mobWanderPauseExtra = 60
	mobStuckRetarget    = 10
)

const (
	creeperFuseTimeDefault        = 30
	creeperExplosionRadiusNormal  = 3
	creeperExplosionRadiusCharged = 6
	mobGriefingDefault            = true
)

const (
	skeletonArrowMinInterval = 20
	skeletonArrowMaxInterval = 60
	skeletonArrowRange       = 15.0
	skeletonArrowRangeSq     = skeletonArrowRange * skeletonArrowRange
	skeletonStopMoveSeeTicks = 20
)

const (
	spiderLeapMinDist = 2.0
	spiderLeapMaxDist = 6.0
)

const undeadSunDamagePerTick = 1.0

const (
	entityTypeCreeper  int8 = 50
	entityTypeSkeleton int8 = 51
	entityTypeSpider   int8 = 52
	entityTypeZombie   int8 = 54
	entityTypeSlime    int8 = 55
	entityTypeEnderman int8 = 58
	entityTypeBat      int8 = 65

	entityTypePig     int8 = 90
	entityTypeSheep   int8 = 91
	entityTypeCow     int8 = 92
	entityTypeChicken int8 = 93
	entityTypeSquid   int8 = 94
	entityTypeHorse   int8 = 100
)

const (
	mobItemIDPumpkin      int16 = 86
	mobItemIDJackOLantern int16 = 91

	mobItemIDBow int16 = 261

	mobItemIDIronShovel int16 = 256
	mobItemIDIronSword  int16 = 267

	mobItemIDHelmetLeather int16 = 298
	mobItemIDHelmetChain   int16 = 302
	mobItemIDHelmetIron    int16 = 306
	mobItemIDHelmetDiamond int16 = 310
	mobItemIDHelmetGold    int16 = 314

	mobItemIDChestLeather int16 = 299
	mobItemIDChestChain   int16 = 303
	mobItemIDChestIron    int16 = 307
	mobItemIDChestDiamond int16 = 311
	mobItemIDChestGold    int16 = 315

	mobItemIDLegsLeather int16 = 300
	mobItemIDLegsChain   int16 = 304
	mobItemIDLegsIron    int16 = 308
	mobItemIDLegsDiamond int16 = 312
	mobItemIDLegsGold    int16 = 316

	mobItemIDBootsLeather int16 = 301
	mobItemIDBootsChain   int16 = 305
	mobItemIDBootsIron    int16 = 309
	mobItemIDBootsDiamond int16 = 313
	mobItemIDBootsGold    int16 = 317
)

type zombieSpawnGroupData struct {
	child    bool
	villager bool
}

type spawnMaterial int

const (
	spawnMaterialAir spawnMaterial = iota
	spawnMaterialWater
)

type mobCreatureType int

const (
	creatureTypeMonster mobCreatureType = iota
	creatureTypeCreature
	creatureTypeAmbient
	creatureTypeWater
)

type creatureTypeConfig struct {
	creatureType mobCreatureType
	maxNumber    int
	material     spawnMaterial
	peaceful     bool
	animal       bool
}

var creatureTypeOrder = []creatureTypeConfig{
	{creatureType: creatureTypeMonster, maxNumber: 70, material: spawnMaterialAir, peaceful: false, animal: false},
	{creatureType: creatureTypeCreature, maxNumber: 10, material: spawnMaterialAir, peaceful: true, animal: true},
	{creatureType: creatureTypeAmbient, maxNumber: 15, material: spawnMaterialAir, peaceful: true, animal: false},
	{creatureType: creatureTypeWater, maxNumber: 5, material: spawnMaterialWater, peaceful: true, animal: false},
}

type spawnListEntry struct {
	entityType   int8
	weight       int
	minGroupSize int
	maxGroupSize int
	maxPerChunk  int
	fixedSlime   int
}

var defaultCreatureSpawns = []spawnListEntry{
	{entityType: entityTypeSheep, weight: 12, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
	{entityType: entityTypePig, weight: 10, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
	{entityType: entityTypeChicken, weight: 10, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
	{entityType: entityTypeCow, weight: 8, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
}

var plainsCreatureSpawns = append(append([]spawnListEntry{}, defaultCreatureSpawns...),
	spawnListEntry{entityType: entityTypeHorse, weight: 5, minGroupSize: 2, maxGroupSize: 6, maxPerChunk: 6},
)

var defaultMonsterSpawns = []spawnListEntry{
	{entityType: entityTypeSpider, weight: 10, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
	{entityType: entityTypeZombie, weight: 10, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
	{entityType: entityTypeSkeleton, weight: 10, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
	{entityType: entityTypeCreeper, weight: 10, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
	{entityType: entityTypeSlime, weight: 10, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
	{entityType: entityTypeEnderman, weight: 1, minGroupSize: 1, maxGroupSize: 4, maxPerChunk: 4},
}

var defaultAmbientSpawns = []spawnListEntry{
	{entityType: entityTypeBat, weight: 10, minGroupSize: 8, maxGroupSize: 8, maxPerChunk: 4},
}

var defaultWaterSpawns = []spawnListEntry{
	{entityType: entityTypeSquid, weight: 10, minGroupSize: 4, maxGroupSize: 4, maxPerChunk: 4},
}

type trackedMob struct {
	EntityID       int32
	EntityType     int8
	CreatureType   mobCreatureType
	X              float64
	Y              float64
	Z              float64
	Yaw            float32
	Pitch          float32
	HeadYaw        float32
	MotionX        float64
	MotionY        float64
	MotionZ        float64
	Health         float32
	LastDamage     float32
	HurtResistant  int
	SeenBy         map[*loginSession]struct{}
	stationaryAge  int
	attackCooldown int
	wanderYaw      float64
	wanderPitch    float64
	wanderTicks    int
	wanderPause    int
	creeperState   int8
	creeperIgnited int
	creeperFuse    int
	creeperRadius  int
	creeperPowered bool
	rangedCooldown int
	seeTime        int
	slimeSize      int
	slimeJumpDelay int
	spiderLeapCD   int
	spiderClimb    bool
	endermanStare  int
	sheepColor     int8
	sheepSheared   bool
	skeletonType   int8
	zombieChild    bool
	zombieVillager bool
	canPickUpLoot  bool
	heldItemID     int16
	chestItemID    int16
	legsItemID     int16
	bootsItemID    int16
	helmetItemID   int16
	helmetDamage   int16
}

func (m *trackedMob) chunkCoords() (int32, int32) {
	return chunkCoordFromPos(m.X), chunkCoordFromPos(m.Z)
}

func (m *trackedMob) updateCreeperState(target *loginSession, s *StatusServer) {
	if m == nil || s == nil || m.EntityType != entityTypeCreeper {
		return
	}

	state := int8(-1)
	if target != nil {
		tx, ty, tz := target.positionSnapshot()
		dx := tx - m.X
		dy := ty - m.Y
		dz := tz - m.Z
		if dx*dx+dy*dy+dz*dz <= 49.0 && s.hasLineOfSight(m.X, m.Y+1.0, m.Z, tx, ty+1.0, tz) {
			state = 1
		}
	}

	m.creeperState = state
	m.creeperIgnited += int(state)
	if m.creeperIgnited < 0 {
		m.creeperIgnited = 0
	}
}

func (m *trackedMob) spawnPacket() *protocol.Packet24MobSpawn {
	vel := protocol.NewPacket28EntityVelocity(m.EntityID, m.MotionX, m.MotionY, m.MotionZ)
	return &protocol.Packet24MobSpawn{
		EntityID:  m.EntityID,
		Type:      m.EntityType,
		XPosition: toPacketPosition(m.X),
		YPosition: toPacketPosition(m.Y),
		ZPosition: toPacketPosition(m.Z),
		Yaw:       toPacketAngle(m.Yaw),
		Pitch:     toPacketAngle(m.Pitch),
		HeadYaw:   toPacketAngle(m.HeadYaw),
		VelocityX: vel.MotionX,
		VelocityY: vel.MotionY,
		VelocityZ: vel.MotionZ,
		Metadata:  m.metadataWatchers(),
	}
}

func (m *trackedMob) metadataPacket() *protocol.Packet40EntityMetadata {
	return &protocol.Packet40EntityMetadata{
		EntityID: m.EntityID,
		Metadata: m.metadataWatchers(),
	}
}

func (m *trackedMob) metadataWatchers() []protocol.WatchableObject {
	if m == nil {
		return nil
	}

	metadata := []protocol.WatchableObject{
		{
			ObjectType:  0,
			DataValueID: 0,
			Value:       int8(0),
		},
	}

	switch m.EntityType {
	case entityTypeCreeper:
		powered := int8(0)
		if m.creeperPowered {
			powered = 1
		}
		metadata = append(metadata,
			protocol.WatchableObject{
				ObjectType:  0,
				DataValueID: 16,
				Value:       m.creeperState,
			},
			protocol.WatchableObject{
				ObjectType:  0,
				DataValueID: 17,
				Value:       powered,
			},
		)
	case entityTypeSlime:
		size := m.slimeSize
		if size <= 0 {
			size = 1
		}
		if size > 127 {
			size = 127
		}
		metadata = append(metadata, protocol.WatchableObject{
			ObjectType:  0,
			DataValueID: 16,
			Value:       int8(size),
		})
	case entityTypeSheep:
		value := m.sheepColor & 15
		if m.sheepSheared {
			value |= 16
		}
		metadata = append(metadata, protocol.WatchableObject{
			ObjectType:  0,
			DataValueID: 16,
			Value:       value,
		})
	case entityTypeSpider:
		value := int8(0)
		if m.spiderClimb {
			value |= 1
		}
		metadata = append(metadata, protocol.WatchableObject{
			ObjectType:  0,
			DataValueID: 16,
			Value:       value,
		})
	case entityTypeSkeleton:
		metadata = append(metadata, protocol.WatchableObject{
			ObjectType:  0,
			DataValueID: 13,
			Value:       m.skeletonType,
		})
	case entityTypeZombie:
		child := int8(0)
		if m.zombieChild {
			child = 1
		}
		villager := int8(0)
		if m.zombieVillager {
			villager = 1
		}
		metadata = append(metadata,
			protocol.WatchableObject{
				ObjectType:  0,
				DataValueID: 12,
				Value:       child,
			},
			protocol.WatchableObject{
				ObjectType:  0,
				DataValueID: 13,
				Value:       villager,
			},
		)
	}

	return metadata
}

func (m *trackedMob) teleportPacket() *protocol.Packet34EntityTeleport {
	return &protocol.Packet34EntityTeleport{
		EntityID:  m.EntityID,
		XPosition: toPacketPosition(m.X),
		YPosition: toPacketPosition(m.Y),
		ZPosition: toPacketPosition(m.Z),
		Yaw:       toPacketAngle(m.Yaw),
		Pitch:     toPacketAngle(m.Pitch),
	}
}

type spawnPlayerSnapshot struct {
	session *loginSession
	x       float64
	y       float64
	z       float64
	chunkX  int32
	chunkZ  int32
}

// TickMobSpawning translates baseline flow from WorldServer#tick mobSpawner section
// and SpawnerAnimals#findChunksForSpawning.
func (s *StatusServer) TickMobSpawning() {
	s.tickMobsMotionAndVisibility()

	_, worldTime := s.CurrentWorldTime()
	spawnAnimals := worldTime%400 == 0
	s.findChunksForSpawning(true, true, spawnAnimals)
}

func (s *StatusServer) tickMobsMotionAndVisibility() {
	s.mobMu.Lock()
	if len(s.mobs) == 0 {
		s.mobMu.Unlock()
		return
	}
	mobs := make([]*trackedMob, 0, len(s.mobs))
	for _, mob := range s.mobs {
		if mob != nil {
			mobs = append(mobs, mob)
		}
	}
	s.mobMu.Unlock()

	for _, mob := range mobs {
		s.tickSingleMob(mob)
	}
}

func (s *StatusServer) tickSingleMob(mob *trackedMob) {
	if mob == nil {
		return
	}

	var target *loginSession
	var movementPacket protocol.Packet
	shouldAttack := false
	shouldExplode := false
	shouldShootArrow := false
	arrowPower := 0.0
	holdRangedPosition := false
	metadataDirty := false
	sunDamaged := false
	sunDied := false
	sunHurtStatus := false
	explodeX, explodeY, explodeZ, explodeRadius := 0.0, 0.0, 0.0, 0.0

	s.mobMu.Lock()
	if mob.Health <= 0 {
		s.mobMu.Unlock()
		s.killMob(mob)
		return
	}
	if mob.HurtResistant > 0 {
		mob.HurtResistant--
	}
	if s.shouldUndeadBurnInDaylightLocked(mob) {
		sunDamaged, sunDied, sunHurtStatus = s.applyDamageToMobLocked(mob, undeadSunDamagePerTick)
	}
	if sunDied {
		s.mobMu.Unlock()
		if sunHurtStatus {
			s.broadcastMobEntityStatus(mob, 2)
		}
		s.broadcastMobEntityStatus(mob, 3)
		s.killMob(mob)
		return
	}
	if mob.attackCooldown > 0 {
		mob.attackCooldown--
	}

	if mob.CreatureType == creatureTypeMonster {
		target = s.nearestTargetForMob(mob, s.mobFollowRange(mob))
		if target != nil {
			tx, ty, tz := target.positionSnapshot()
			if !s.hasLineOfSight(mob.X, mob.Y+1.0, mob.Z, tx, ty+1.0, tz) {
				dx := tx - mob.X
				dy := ty - mob.Y
				dz := tz - mob.Z
				if dx*dx+dy*dy+dz*dz > 9.0 {
					target = nil
				}
			}
		}
	}
	if mob.EntityType == entityTypeSpider && s.isSpiderPassiveByDay(mob) {
		target = nil
	}

	if mob.EntityType == entityTypeSkeleton {
		distSq := math.MaxFloat64
		canSee := false
		if target != nil {
			tx, ty, tz := target.positionSnapshot()
			dx := tx - mob.X
			dy := ty - mob.Y
			dz := tz - mob.Z
			distSq = dx*dx + dy*dy + dz*dz
			canSee = s.hasLineOfSight(mob.X, mob.Y+1.0, mob.Z, tx, ty+1.0, tz)
		}

		if canSee {
			mob.seeTime++
		} else {
			mob.seeTime = 0
		}
		if canSee && distSq <= skeletonArrowRangeSq && mob.seeTime >= skeletonStopMoveSeeTicks {
			holdRangedPosition = true
		}

		mob.rangedCooldown--
		if mob.rangedCooldown == 0 {
			if target != nil && canSee && distSq <= skeletonArrowRangeSq {
				arrowPower = clampFloat64(math.Sqrt(distSq)/skeletonArrowRange, 0.1, 1.0)
				shouldShootArrow = true
				mob.rangedCooldown = s.skeletonArrowCooldownFromDistSq(distSq)
			}
		} else if mob.rangedCooldown < 0 {
			if target != nil && distSq < math.MaxFloat64 {
				mob.rangedCooldown = s.skeletonArrowCooldownFromDistSq(distSq)
			}
		}
	}

	if mob.EntityType == entityTypeCreeper {
		prevCreeperState := mob.creeperState
		mob.updateCreeperState(target, s)
		if mob.creeperState != prevCreeperState {
			metadataDirty = true
		}
		fuse := mob.creeperFuse
		if fuse <= 0 {
			fuse = creeperFuseTimeDefault
			mob.creeperFuse = fuse
		}
		if mob.creeperIgnited >= fuse {
			mob.creeperIgnited = fuse
			radius := mob.creeperRadius
			if radius <= 0 {
				radius = creeperExplosionRadiusNormal
			}
			if mob.creeperPowered {
				radius *= 2
			}
			shouldExplode = true
			explodeX, explodeY, explodeZ = mob.X, mob.Y, mob.Z
			explodeRadius = float64(radius)
		}
	}
	if shouldExplode {
		s.mobMu.Unlock()
		s.mobExplode(mob, explodeX, explodeY, explodeZ, explodeRadius)
		return
	}

	desiredX, desiredY, desiredZ := s.mobDesiredVelocityLocked(mob, target)
	if mob.EntityType == entityTypeSpider && target != nil {
		if mob.spiderLeapCD > 0 {
			mob.spiderLeapCD--
		}
		tx, _, tz := target.positionSnapshot()
		dx := tx - mob.X
		dz := tz - mob.Z
		distHoriz := math.Sqrt(dx*dx + dz*dz)
		if mob.spiderLeapCD <= 0 &&
			distHoriz > spiderLeapMinDist &&
			distHoriz < spiderLeapMaxDist &&
			s.mobIsOnGround(mob) &&
			s.mobRand.NextInt(10) == 0 {
			if distHoriz > 1.0e-9 {
				desiredX = dx/distHoriz*0.32 + desiredX*0.2
				desiredZ = dz/distHoriz*0.32 + desiredZ*0.2
				mob.spiderLeapCD = 20
			}
		}
	}
	if mob.EntityType == entityTypeSlime && s.mobIsOnGround(mob) {
		if mob.slimeJumpDelay <= 0 {
			mob.slimeJumpDelay = s.nextSlimeJumpDelay(target != nil)
		}
		mob.slimeJumpDelay--
		if mob.slimeJumpDelay <= 0 {
			mob.slimeJumpDelay = s.nextSlimeJumpDelay(target != nil)
			desiredX *= 1.25
			desiredZ *= 1.25
		} else {
			desiredX, desiredY, desiredZ = 0, 0, 0
		}
	}
	if mob.EntityType == entityTypeCreeper && mob.creeperState > 0 {
		// Translation reference:
		// - net.minecraft.src.EntityAICreeperSwell#startExecuting()
		// Creeper swells in place while igniting.
		desiredX, desiredY, desiredZ = 0, 0, 0
	}
	if holdRangedPosition {
		desiredX, desiredY, desiredZ = 0, 0, 0
	}
	oldX, oldY, oldZ := mob.X, mob.Y, mob.Z

	moved := false
	switch mob.CreatureType {
	case creatureTypeAmbient:
		moved = s.tryMoveFlyingMob(mob, desiredX, desiredY, desiredZ)
	case creatureTypeWater:
		moved = s.tryMoveWaterMob(mob, desiredX, desiredY, desiredZ)
	default:
		moved = s.tryMoveLandMob(mob, desiredX, desiredZ)
	}

	if moved {
		mob.MotionX = mob.X - oldX
		mob.MotionY = mob.Y - oldY
		mob.MotionZ = mob.Z - oldZ
		horizontal := math.Abs(mob.MotionX) + math.Abs(mob.MotionZ)
		if horizontal > 1.0e-6 {
			mob.Yaw = float32(math.Atan2(mob.MotionX, mob.MotionZ) * 180.0 / math.Pi)
			mob.HeadYaw = mob.Yaw
		}
		mob.stationaryAge = 0
		movementPacket = mob.teleportPacket()
	} else {
		mob.MotionX *= 0.4
		mob.MotionY *= 0.4
		mob.MotionZ *= 0.4
		mob.stationaryAge++
	}
	if mob.EntityType == entityTypeSpider {
		shouldClimb := !moved && (math.Abs(desiredX)+math.Abs(desiredZ) > 1.0e-6)
		if shouldClimb != mob.spiderClimb {
			mob.spiderClimb = shouldClimb
			metadataDirty = true
		}
	}

	if mob.CreatureType == creatureTypeMonster && mob.EntityType != entityTypeCreeper && mob.EntityType != entityTypeSkeleton && target != nil && mob.attackCooldown <= 0 {
		tx, ty, tz := target.positionSnapshot()
		dx := tx - mob.X
		dy := ty - mob.Y
		dz := tz - mob.Z
		if dx*dx+dy*dy+dz*dz <= s.mobAttackRangeSq(mob) && s.hasLineOfSight(mob.X, mob.Y+1.0, mob.Z, tx, ty+1.0, tz) {
			shouldAttack = true
			mob.attackCooldown = 20
		}
	}
	s.mobMu.Unlock()

	if shouldAttack {
		s.mobMeleeAttack(mob, target)
	}
	if shouldShootArrow {
		s.spawnArrowFromMob(mob, target, arrowPower)
	}
	if sunDamaged && sunHurtStatus {
		s.broadcastMobEntityStatus(mob, 2)
	}

	s.updateMobVisibility(mob, movementPacket)
	if metadataDirty {
		s.broadcastMobMetadata(mob)
	}
}

func (s *StatusServer) mobDesiredVelocityLocked(mob *trackedMob, target *loginSession) (float64, float64, float64) {
	if mob == nil {
		return 0, 0, 0
	}
	speed := s.mobMoveSpeed(mob)
	if speed <= 0 {
		return 0, 0, 0
	}

	if mob.CreatureType == creatureTypeMonster && target != nil {
		tx, ty, tz := target.positionSnapshot()
		dx := tx - mob.X
		dy := ty - mob.Y
		dz := tz - mob.Z
		distSq := dx*dx + dy*dy + dz*dz
		if distSq > 1.0e-6 {
			dist := math.Sqrt(distSq)
			return dx / dist * speed, clampFloat64(dy/dist*speed, -speed*0.4, speed*0.4), dz / dist * speed
		}
	}

	return s.mobWanderVelocityLocked(mob, speed)
}

func (s *StatusServer) mobWanderVelocityLocked(mob *trackedMob, speed float64) (float64, float64, float64) {
	if mob == nil || speed <= 0 {
		return 0, 0, 0
	}
	if mob.wanderPause > 0 {
		mob.wanderPause--
		return 0, 0, 0
	}

	if mob.wanderTicks <= 0 || mob.stationaryAge >= mobStuckRetarget || s.mobRand.NextInt(40) == 0 {
		s.randomizeWanderLocked(mob)
	}

	mob.wanderTicks--
	if mob.wanderTicks <= 0 {
		mob.wanderPause = mobWanderPauseMin + int(s.mobRand.NextInt(mobWanderPauseExtra))
	}

	if s.mobRand.NextInt(25) == 0 {
		mob.wanderYaw += (s.mobRand.NextDouble() - 0.5) * 1.2
		if mob.CreatureType == creatureTypeAmbient || mob.CreatureType == creatureTypeWater {
			mob.wanderPitch += (s.mobRand.NextDouble() - 0.5) * 0.4
			mob.wanderPitch = clampFloat64(mob.wanderPitch, -0.7, 0.7)
		}
	}

	vx := -math.Sin(mob.wanderYaw) * speed
	vz := math.Cos(mob.wanderYaw) * speed
	vy := 0.0

	switch mob.CreatureType {
	case creatureTypeAmbient:
		vy = math.Sin(mob.wanderPitch) * speed * 0.5
		if mob.Y < 3 {
			vy = math.Abs(vy)
		}
		if mob.Y > 96 {
			vy = -math.Abs(vy)
		}
	case creatureTypeWater:
		vy = math.Sin(mob.wanderPitch) * speed * 0.35
		if mob.Y < 46 {
			vy = math.Abs(vy)
		}
		if mob.Y > 62 {
			vy = -math.Abs(vy)
		}
	}
	return vx, vy, vz
}

func (s *StatusServer) randomizeWanderLocked(mob *trackedMob) {
	if mob == nil {
		return
	}
	mob.wanderYaw = s.mobRand.NextDouble() * math.Pi * 2.0
	mob.wanderPitch = (s.mobRand.NextDouble() - 0.5) * 1.0
	mob.wanderTicks = mobWanderMinTicks + int(s.mobRand.NextInt(mobWanderExtraTicks))
}

func (s *StatusServer) tryMoveLandMob(mob *trackedMob, dx, dz float64) bool {
	if mob == nil {
		return false
	}
	if math.Abs(dx)+math.Abs(dz) <= 1.0e-6 {
		return false
	}

	startX, startY, startZ := mob.X, mob.Y, mob.Z
	try := func(nx, ny, nz float64) bool {
		if s.canMobStandAt(nx, ny, nz) {
			mob.X = nx
			mob.Y = ny
			mob.Z = nz
			return true
		}
		return false
	}
	tryAllHeights := func(nx, nz float64) bool {
		return try(nx, startY, nz) || try(nx, startY+1, nz) || try(nx, startY-1, nz)
	}

	if tryAllHeights(startX+dx, startZ+dz) {
		return true
	}
	if tryAllHeights(startX+dx, startZ) {
		return true
	}
	if tryAllHeights(startX, startZ+dz) {
		return true
	}

	halfX, halfZ := dx*0.5, dz*0.5
	if math.Abs(halfX)+math.Abs(halfZ) > 1.0e-6 && tryAllHeights(startX+halfX, startZ+halfZ) {
		return true
	}
	return false
}

func (s *StatusServer) tryMoveFlyingMob(mob *trackedMob, dx, dy, dz float64) bool {
	if mob == nil {
		return false
	}
	if math.Abs(dx)+math.Abs(dy)+math.Abs(dz) <= 1.0e-6 {
		return false
	}
	startX, startY, startZ := mob.X, mob.Y, mob.Z
	try := func(nx, ny, nz float64) bool {
		if s.canMobFlyAt(nx, ny, nz) {
			mob.X = nx
			mob.Y = ny
			mob.Z = nz
			return true
		}
		return false
	}
	if try(startX+dx, startY+dy, startZ+dz) {
		return true
	}
	if try(startX+dx, startY, startZ+dz) {
		return true
	}
	return try(startX+dx*0.5, startY+dy*0.5, startZ+dz*0.5)
}

func (s *StatusServer) tryMoveWaterMob(mob *trackedMob, dx, dy, dz float64) bool {
	if mob == nil {
		return false
	}
	if math.Abs(dx)+math.Abs(dy)+math.Abs(dz) <= 1.0e-6 {
		return false
	}
	startX, startY, startZ := mob.X, mob.Y, mob.Z
	try := func(nx, ny, nz float64) bool {
		if s.canMobSwimAt(nx, ny, nz) {
			mob.X = nx
			mob.Y = ny
			mob.Z = nz
			return true
		}
		return false
	}
	if try(startX+dx, startY+dy, startZ+dz) {
		return true
	}
	if try(startX+dx, startY, startZ+dz) {
		return true
	}
	if dy != 0 {
		if try(startX+dx, startY+dy*0.5, startZ+dz) {
			return true
		}
	}
	return false
}

func (s *StatusServer) canMobFlyAt(x, y, z float64) bool {
	blockX := int(math.Floor(x))
	blockY := int(math.Floor(y))
	blockZ := int(math.Floor(z))
	if blockY <= 1 || blockY >= 254 {
		return false
	}

	feetID, _ := s.world.getBlock(blockX, blockY, blockZ)
	headID, _ := s.world.getBlock(blockX, blockY+1, blockZ)
	if block.BlocksMovement(feetID) || block.IsLiquid(feetID) {
		return false
	}
	if block.BlocksMovement(headID) || block.IsLiquid(headID) {
		return false
	}
	return true
}

func (s *StatusServer) canMobSwimAt(x, y, z float64) bool {
	blockX := int(math.Floor(x))
	blockY := int(math.Floor(y))
	blockZ := int(math.Floor(z))
	if blockY <= 1 || blockY >= 254 {
		return false
	}

	feetID, _ := s.world.getBlock(blockX, blockY, blockZ)
	headID, _ := s.world.getBlock(blockX, blockY+1, blockZ)
	return block.IsLiquid(feetID) && block.IsLiquid(headID)
}

func (s *StatusServer) mobMoveSpeed(mob *trackedMob) float64 {
	if mob == nil {
		return 0
	}
	switch mob.EntityType {
	case entityTypeChicken:
		return 0.045
	case entityTypePig, entityTypeSheep, entityTypeCow:
		return 0.05
	case entityTypeHorse:
		return 0.09
	case entityTypeSpider:
		return 0.11
	case entityTypeZombie:
		speed := 0.095
		if mob.zombieChild {
			// Translation reference:
			// - net.minecraft.src.EntityZombie#setChild(boolean)
			//   babySpeedBoostModifier amount=0.5 (operation 1).
			speed *= 1.5
		}
		return speed
	case entityTypeSkeleton, entityTypeCreeper:
		return 0.095
	case entityTypeSlime:
		return 0.08
	case entityTypeEnderman:
		return 0.12
	case entityTypeBat:
		return 0.07
	case entityTypeSquid:
		return 0.06
	default:
		switch mob.CreatureType {
		case creatureTypeMonster:
			return 0.09
		case creatureTypeCreature:
			return 0.05
		case creatureTypeAmbient:
			return 0.07
		case creatureTypeWater:
			return 0.06
		default:
			return 0.05
		}
	}
}

func (s *StatusServer) mobFollowRange(mob *trackedMob) float64 {
	if mob == nil || mob.CreatureType != creatureTypeMonster {
		return 0
	}
	switch mob.EntityType {
	case entityTypeZombie:
		// Translation reference:
		// - net.minecraft.src.EntityZombie#applyEntityAttributes (followRange=40.0D)
		return 40.0
	case entityTypeEnderman:
		// Translation reference:
		// - net.minecraft.src.EntityEnderman#findPlayerToAttack() uses 64.0D search
		return 64.0
	case entityTypeSpider:
		// Translation reference:
		// - net.minecraft.src.EntitySpider#findPlayerToAttack() uses 16.0D search
		return 16.0
	case entityTypeSlime:
		return 16.0
	default:
		// Translation reference:
		// - net.minecraft.src.SharedMonsterAttributes.followRange default = 32.0D
		return 32.0
	}
}

func (s *StatusServer) mobAttackRangeSq(mob *trackedMob) float64 {
	if mob == nil {
		return 4.0
	}
	switch mob.EntityType {
	case entityTypeSpider:
		return 2.4 * 2.4
	case entityTypeEnderman:
		return 2.8 * 2.8
	case entityTypeSlime:
		size := mob.slimeSize
		if size <= 0 {
			size = 1
		}
		r := 0.6 * float64(size)
		return r * r
	default:
		return 2.0 * 2.0
	}
}

func (s *StatusServer) hasLineOfSight(x0, y0, z0, x1, y1, z1 float64) bool {
	dx := x1 - x0
	dy := y1 - y0
	dz := z1 - z0
	maxAxis := math.Max(math.Abs(dx), math.Max(math.Abs(dy), math.Abs(dz)))
	steps := int(maxAxis * 8.0)
	if steps < 1 {
		return true
	}
	for i := 1; i < steps; i++ {
		t := float64(i) / float64(steps)
		bx := int(math.Floor(x0 + dx*t))
		by := int(math.Floor(y0 + dy*t))
		bz := int(math.Floor(z0 + dz*t))
		id, _ := s.world.getBlock(bx, by, bz)
		if block.BlocksMovement(id) {
			return false
		}
	}
	return true
}

func clampFloat64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (s *StatusServer) skeletonArrowCooldownFromDistSq(distSq float64) int {
	if distSq < 0 {
		distSq = 0
	}
	f := clampFloat64(math.Sqrt(distSq)/skeletonArrowRange, 0.1, 1.0)
	return int(math.Floor(f*float64(skeletonArrowMaxInterval-skeletonArrowMinInterval) + float64(skeletonArrowMinInterval)))
}

func (s *StatusServer) mobIsOnGround(mob *trackedMob) bool {
	if mob == nil {
		return false
	}
	return s.canMobStandAt(mob.X, mob.Y, mob.Z)
}

func (s *StatusServer) nextSlimeJumpDelay(hasTarget bool) int {
	delay := int(s.mobRand.NextInt(20)) + 10
	if hasTarget {
		delay /= 3
		if delay < 1 {
			delay = 1
		}
	}
	return delay
}

func (s *StatusServer) isSpiderPassiveByDay(mob *trackedMob) bool {
	if mob == nil {
		return false
	}
	_, worldTime := s.CurrentWorldTime()
	tickOfDay := worldTime % 24000
	if tickOfDay < 0 {
		tickOfDay += 24000
	}
	if tickOfDay >= 12000 {
		return false
	}
	x := int(math.Floor(mob.X))
	y := int(math.Floor(mob.Y))
	z := int(math.Floor(mob.Z))
	return s.fullBlockLightValue(x, y, z) > 8
}

func (s *StatusServer) nearestTargetForMob(mob *trackedMob, maxDistance float64) *loginSession {
	if mob == nil {
		return nil
	}
	targets := s.activeSessionsExcept(nil)
	bestDistSq := maxDistance * maxDistance

	if mob.EntityType == entityTypeEnderman {
		var bestLooked *loginSession
		for _, target := range targets {
			if target == nil {
				continue
			}
			target.stateMu.Lock()
			alive := target.playerRegistered && !target.playerDead && target.entityID != 0
			tx := target.playerX
			ty := target.playerY
			tz := target.playerZ
			yaw := target.playerYaw
			pitch := target.playerPitch
			helmet := int16(0)
			if head := target.inventory[5]; head != nil {
				helmet = head.ItemID
			}
			target.stateMu.Unlock()
			if !alive {
				continue
			}

			dx := tx - mob.X
			dy := ty - mob.Y
			dz := tz - mob.Z
			distSq := dx*dx + dy*dy + dz*dz
			if distSq >= bestDistSq {
				continue
			}
			if !s.shouldEndermanAttackPlayer(mob, tx, ty, tz, yaw, pitch, helmet) {
				continue
			}
			bestDistSq = distSq
			bestLooked = target
		}
		if bestLooked == nil {
			mob.endermanStare = 0
			return nil
		}
		mob.endermanStare++
		if mob.endermanStare < 5 {
			return nil
		}
		return bestLooked
	}

	var best *loginSession
	for _, target := range targets {
		if target == nil {
			continue
		}
		target.stateMu.Lock()
		alive := target.playerRegistered && !target.playerDead && target.entityID != 0
		tx := target.playerX
		ty := target.playerY
		tz := target.playerZ
		target.stateMu.Unlock()
		if !alive {
			continue
		}

		dx := tx - mob.X
		dy := ty - mob.Y
		dz := tz - mob.Z
		distSq := dx*dx + dy*dy + dz*dz
		if distSq < bestDistSq {
			bestDistSq = distSq
			best = target
		}
	}
	return best
}

func (s *StatusServer) shouldEndermanAttackPlayer(mob *trackedMob, playerX, playerY, playerZ float64, playerYaw, playerPitch float32, helmetItemID int16) bool {
	if mob == nil {
		return false
	}
	// Translation reference:
	// - net.minecraft.src.EntityEnderman#shouldAttackPlayer(EntityPlayer)
	if helmetItemID == 86 { // pumpkin block ID
		return false
	}

	lookX, lookY, lookZ := lookVectorFromYawPitch(playerYaw, playerPitch)
	height, _ := mobCollisionSize(mob)
	toX := mob.X - playerX
	toY := (mob.Y + height*0.5) - (playerY + defaultPlayerEyeY)
	toZ := mob.Z - playerZ
	dist := math.Sqrt(toX*toX + toY*toY + toZ*toZ)
	if dist <= 1.0e-9 {
		return true
	}
	toX /= dist
	toY /= dist
	toZ /= dist
	dot := lookX*toX + lookY*toY + lookZ*toZ
	if dot <= 1.0-0.025/dist {
		return false
	}
	return s.hasLineOfSight(playerX, playerY+defaultPlayerEyeY, playerZ, mob.X, mob.Y+height*0.5, mob.Z)
}

func lookVectorFromYawPitch(yaw, pitch float32) (x, y, z float64) {
	yawRad := float64(yaw) * math.Pi / 180.0
	pitchRad := float64(pitch) * math.Pi / 180.0
	x = -math.Sin(yawRad) * math.Cos(pitchRad)
	y = -math.Sin(pitchRad)
	z = math.Cos(yawRad) * math.Cos(pitchRad)
	return x, y, z
}

func (s *StatusServer) mobMeleeAttack(mob *trackedMob, target *loginSession) {
	if mob == nil || target == nil {
		return
	}
	damaged, died, hurtStatus := target.applyIncomingDamage(s.mobAttackDamage(mob), true)
	if damaged && hurtStatus {
		target.broadcastEntityStatusToSelfAndWatchers(2)
	}
	if died {
		target.broadcastEntityStatusToSelfAndWatchers(3)
		target.sendSystemChat("You died.")
	}
}

func (s *StatusServer) mobAttackDamage(mob *trackedMob) float32 {
	if mob == nil {
		return 2.0
	}
	switch mob.EntityType {
	case entityTypeZombie, entityTypeSkeleton:
		return 3.0
	case entityTypeSpider:
		return 2.0
	case entityTypeSlime:
		if mob.slimeSize <= 1 {
			return 0
		}
		return float32(mob.slimeSize)
	case entityTypeCreeper:
		return 4.0
	case entityTypeEnderman:
		// Translation reference:
		// - net.minecraft.src.EntityEnderman#applyEntityAttributes (attackDamage=7.0D)
		return 7.0
	default:
		return 2.0
	}
}

func (s *StatusServer) mobExplode(mob *trackedMob, x, y, z, size float64) {
	if mob == nil || size <= 0 {
		return
	}

	if mobGriefingDefault {
		s.destroyBlocksInExplosion(x, y, z, size)
	}
	s.damagePlayersInExplosion(x, y, z, size)
	s.removeMob(mob)
}

func (s *StatusServer) damagePlayersInExplosion(x, y, z, size float64) {
	if size <= 0 {
		return
	}
	effectRadius := size * 2.0
	if effectRadius <= 0 {
		return
	}

	targets := s.activeSessionsExcept(nil)
	for _, target := range targets {
		if target == nil {
			continue
		}
		target.stateMu.Lock()
		alive := target.playerRegistered && !target.playerDead && target.entityID != 0
		tx := target.playerX
		ty := target.playerY + defaultPlayerEyeY
		tz := target.playerZ
		target.stateMu.Unlock()
		if !alive {
			continue
		}

		dx := tx - x
		dy := ty - y
		dz := tz - z
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if dist > effectRadius {
			continue
		}
		impact := 1.0 - dist/effectRadius
		if impact <= 0 {
			continue
		}
		damage := float32(int(((impact*impact+impact)/2.0)*8.0*size + 1.0))
		if damage <= 0 {
			continue
		}

		damaged, died, hurtStatus := target.applyIncomingDamage(damage, true)
		if damaged && hurtStatus {
			target.broadcastEntityStatusToSelfAndWatchers(2)
		}
		if died {
			target.broadcastEntityStatusToSelfAndWatchers(3)
			target.sendSystemChat("You died.")
		}
	}
}

func (s *StatusServer) destroyBlocksInExplosion(x, y, z, size float64) {
	if size <= 0 {
		return
	}

	minX := int(math.Floor(x - size))
	maxX := int(math.Floor(x + size))
	minY := int(math.Floor(y - size))
	maxY := int(math.Floor(y + size))
	minZ := int(math.Floor(z - size))
	maxZ := int(math.Floor(z + size))

	radiusSq := size * size
	for bx := minX; bx <= maxX; bx++ {
		for by := minY; by <= maxY; by++ {
			if by <= 0 || by >= 255 {
				continue
			}
			for bz := minZ; bz <= maxZ; bz++ {
				cx := float64(bx) + 0.5
				cy := float64(by) + 0.5
				cz := float64(bz) + 0.5
				dx := cx - x
				dy := cy - y
				dz := cz - z
				if dx*dx+dy*dy+dz*dz > radiusSq {
					continue
				}

				blockID, _ := s.world.getBlock(bx, by, bz)
				if blockID == 0 || blockID == 7 {
					continue
				}
				if block.IsLiquid(blockID) {
					continue
				}
				if !s.world.setBlock(bx, by, bz, 0, 0) {
					continue
				}
				s.broadcastBlockChange(int32(bx), int32(by), int32(bz), 0, 0)
			}
		}
	}
}

func (s *StatusServer) removeMob(mob *trackedMob) {
	if mob == nil {
		return
	}

	s.mobMu.Lock()
	if _, exists := s.mobs[mob.EntityID]; !exists {
		s.mobMu.Unlock()
		return
	}
	delete(s.mobs, mob.EntityID)
	seenBy := make([]*loginSession, 0, len(mob.SeenBy))
	for session := range mob.SeenBy {
		seenBy = append(seenBy, session)
	}
	mob.SeenBy = make(map[*loginSession]struct{})
	s.mobMu.Unlock()

	destroy := &protocol.Packet29DestroyEntity{EntityIDs: []int32{mob.EntityID}}
	for _, session := range seenBy {
		if session == nil {
			continue
		}
		_ = session.sendPacket(destroy)
	}
}

func (s *StatusServer) mobByEntityID(entityID int32) *trackedMob {
	if entityID == 0 {
		return nil
	}
	s.mobMu.Lock()
	defer s.mobMu.Unlock()
	return s.mobs[entityID]
}

func (s *StatusServer) mobMaxHealth(mob *trackedMob) float32 {
	if mob == nil {
		return 20.0
	}
	switch mob.EntityType {
	case entityTypeSpider:
		return 16.0
	case entityTypeEnderman:
		return 40.0
	case entityTypeSlime:
		size := mob.slimeSize
		if size <= 0 {
			size = 1
		}
		return float32(size * size)
	case entityTypePig, entityTypeCow, entityTypeSquid:
		return 10.0
	case entityTypeSheep:
		return 8.0
	case entityTypeChicken:
		return 4.0
	case entityTypeBat:
		return 6.0
	case entityTypeHorse:
		return 53.0
	default:
		// Translation reference:
		// - SharedMonsterAttributes.maxHealth default value (20.0D)
		return 20.0
	}
}

func (s *StatusServer) shouldUndeadBurnInDaylightLocked(mob *trackedMob) bool {
	if mob == nil {
		return false
	}
	if mob.EntityType != entityTypeZombie && mob.EntityType != entityTypeSkeleton {
		return false
	}
	if mob.EntityType == entityTypeZombie && mob.zombieChild {
		// Translation reference:
		// - net.minecraft.src.EntityZombie#onLivingUpdate()
		return false
	}
	_, worldTime := s.CurrentWorldTime()
	timeOfDay := worldTime % 24000
	if timeOfDay < 0 {
		timeOfDay += 24000
	}
	if timeOfDay >= 12000 {
		return false
	}

	x := int(math.Floor(mob.X))
	y := int(math.Floor(mob.Y))
	z := int(math.Floor(mob.Z))
	brightness := float64(s.blockLightValue(x, y, z)) / 15.0
	if brightness <= 0.5 {
		return false
	}
	if s.mobRand.NextFloat()*30.0 >= float32((brightness-0.4)*2.0) {
		return false
	}
	if !s.canMobSeeSky(x, y, z) {
		return false
	}

	if mob.helmetItemID != 0 {
		// Translation reference:
		// - net.minecraft.src.EntityZombie#onLivingUpdate()
		// - net.minecraft.src.EntitySkeleton#onLivingUpdate()
		// Wearing helmet prevents burning; damageable helmets lose 0..1 durability.
		if maxDamage, ok := mobHelmetMaxDurability(mob.helmetItemID); ok {
			mob.helmetDamage += int16(s.mobRand.NextInt(2))
			if mob.helmetDamage >= maxDamage {
				mob.helmetItemID = 0
				mob.helmetDamage = 0
			}
		}
		return false
	}
	return true
}

func (s *StatusServer) canMobSeeSky(x, y, z int) bool {
	if y < 0 || y >= 256 {
		return false
	}
	if s.savedLightValue(chunk.EnumSkyBlockSky, x, y, z) < 15 {
		return false
	}
	for yy := y + 1; yy < 256; yy++ {
		id, _ := s.world.getBlock(x, yy, z)
		if block.BlocksMovement(id) {
			return false
		}
	}
	return true
}

func mobHelmetMaxDurability(itemID int16) (int16, bool) {
	// Translation reference:
	// - net.minecraft.src.ItemArmor#maxDamageArray[0] (helmet base = 11)
	// - net.minecraft.src.EnumArmorMaterial#getDurability(int)
	switch itemID {
	case mobItemIDHelmetLeather: // leather helmet: 11 * 5
		return 55, true
	case mobItemIDHelmetChain: // chain helmet: 11 * 15
		return 165, true
	case mobItemIDHelmetIron: // iron helmet: 11 * 15
		return 165, true
	case mobItemIDHelmetDiamond: // diamond helmet: 11 * 33
		return 363, true
	case mobItemIDHelmetGold: // gold helmet: 11 * 7
		return 77, true
	default:
		return 0, false
	}
}

func (s *StatusServer) applyDamageToMobLocked(mob *trackedMob, incomingDamage float32) (damaged bool, died bool, hurtStatus bool) {
	if mob == nil || incomingDamage <= 0 || mob.Health <= 0 {
		return false, false, false
	}

	applyDamage := incomingDamage
	hurtStatus = true
	if float32(mob.HurtResistant) > float32(maxHurtResistant)/2.0 {
		if incomingDamage <= mob.LastDamage {
			return false, false, false
		}
		applyDamage = incomingDamage - mob.LastDamage
		mob.LastDamage = incomingDamage
		hurtStatus = false
	} else {
		mob.LastDamage = incomingDamage
		mob.HurtResistant = maxHurtResistant
	}

	if applyDamage <= 0 {
		return false, false, false
	}
	mob.Health -= applyDamage
	if mob.Health <= 0 {
		mob.Health = 0
		died = true
	}
	return true, died, hurtStatus
}

func (s *StatusServer) applyDamageToMob(entityID int32, incomingDamage float32) (mob *trackedMob, damaged bool, died bool, hurtStatus bool) {
	if incomingDamage <= 0 || entityID == 0 {
		return nil, false, false, false
	}

	s.mobMu.Lock()
	mob = s.mobs[entityID]
	if mob == nil || mob.Health <= 0 {
		s.mobMu.Unlock()
		return nil, false, false, false
	}
	damaged, died, hurtStatus = s.applyDamageToMobLocked(mob, incomingDamage)
	s.mobMu.Unlock()
	return mob, damaged, died, hurtStatus
}

func (s *StatusServer) broadcastMobEntityStatus(mob *trackedMob, status int8) {
	if mob == nil || status == 0 {
		return
	}
	packet := &protocol.Packet38EntityStatus{
		EntityID:     mob.EntityID,
		EntityStatus: status,
	}
	chunkX, chunkZ := mob.chunkCoords()
	s.broadcastEntityPacketToWatchers(packet, chunkX, chunkZ, nil)
}

func (s *StatusServer) killMob(mob *trackedMob) {
	if mob == nil {
		return
	}

	shouldSplit := mob.EntityType == entityTypeSlime && mob.slimeSize > 1
	splitSize := mob.slimeSize / 2
	splitX, splitY, splitZ := mob.X, mob.Y, mob.Z

	s.removeMob(mob)

	if !shouldSplit || splitSize <= 0 {
		return
	}

	// Translation reference:
	// - net.minecraft.src.EntitySlime#setDead()
	splitCount := 2 + int(s.mobRand.NextInt(3))
	for i := 0; i < splitCount; i++ {
		offX := (float64(i%2) - 0.5) * float64(mob.slimeSize) / 4.0
		offZ := (float64(i/2) - 0.5) * float64(mob.slimeSize) / 4.0
		yaw := s.mobRand.NextFloat() * 360.0
		entry := &spawnListEntry{entityType: entityTypeSlime, fixedSlime: splitSize}
		_ = s.spawnMob(entry, splitX+offX, splitY+0.5, splitZ+offZ, yaw)
	}
}

func (s *StatusServer) applyKnockbackToMob(mob *trackedMob, attackerYaw float32, knockbackLevel int) bool {
	if mob == nil || knockbackLevel <= 0 {
		return false
	}

	s.mobMu.Lock()
	live := s.mobs[mob.EntityID]
	if live == nil {
		s.mobMu.Unlock()
		return false
	}

	yawRad := float64(attackerYaw) * math.Pi / 180.0
	dx := -math.Sin(yawRad) * float64(knockbackLevel) * 0.5
	dz := math.Cos(yawRad) * float64(knockbackLevel) * 0.5
	oldX, oldY, oldZ := live.X, live.Y, live.Z

	moved := false
	switch live.CreatureType {
	case creatureTypeAmbient:
		moved = s.tryMoveFlyingMob(live, dx, 0.1, dz)
	case creatureTypeWater:
		moved = s.tryMoveWaterMob(live, dx, 0.1, dz)
	default:
		moved = s.tryMoveLandMob(live, dx, dz)
	}

	var movementPacket protocol.Packet
	if moved {
		live.MotionX = live.X - oldX
		live.MotionY = live.Y - oldY
		live.MotionZ = live.Z - oldZ
		movementPacket = live.teleportPacket()
	}
	s.mobMu.Unlock()

	if moved {
		s.updateMobVisibility(live, movementPacket)
	}
	return moved
}

func (s *StatusServer) canMobStandAt(x, y, z float64) bool {
	blockX := int(math.Floor(x))
	blockY := int(math.Floor(y))
	blockZ := int(math.Floor(z))
	if blockY <= 0 || blockY >= 255 {
		return false
	}

	belowID, _ := s.world.getBlock(blockX, blockY-1, blockZ)
	feetID, _ := s.world.getBlock(blockX, blockY, blockZ)
	headID, _ := s.world.getBlock(blockX, blockY+1, blockZ)
	if !block.BlocksMovement(belowID) {
		return false
	}
	if block.BlocksMovement(feetID) || block.IsLiquid(feetID) {
		return false
	}
	if block.BlocksMovement(headID) || block.IsLiquid(headID) {
		return false
	}
	return true
}

func (s *StatusServer) findChunksForSpawning(spawnHostile, spawnPeaceful, spawnAnimals bool) int {
	if !spawnHostile && !spawnPeaceful {
		return 0
	}

	players := s.activeSpawnPlayers()
	if len(players) == 0 {
		return 0
	}

	eligible := make(map[chunk.CoordIntPair]bool)
	for _, pl := range players {
		for dx := -mobSpawnRadiusChunks; dx <= mobSpawnRadiusChunks; dx++ {
			for dz := -mobSpawnRadiusChunks; dz <= mobSpawnRadiusChunks; dz++ {
				border := dx == -mobSpawnRadiusChunks || dx == mobSpawnRadiusChunks || dz == -mobSpawnRadiusChunks || dz == mobSpawnRadiusChunks
				key := chunk.NewCoordIntPair(pl.chunkX+int32(dx), pl.chunkZ+int32(dz))
				if !border {
					eligible[key] = false
				} else {
					if _, ok := eligible[key]; !ok {
						eligible[key] = true
					}
				}
			}
		}
	}

	spawnX, spawnY, spawnZ := s.world.spawnBlockPosition()
	spawnedTotal := 0

	for _, creatureCfg := range creatureTypeOrder {
		if (!creatureCfg.peaceful || spawnPeaceful) &&
			(creatureCfg.peaceful || spawnHostile) &&
			(!creatureCfg.animal || spawnAnimals) {
			capLimit := creatureCfg.maxNumber * len(eligible) / 256
			if s.countMobsByCreatureType(creatureCfg.creatureType) > capLimit {
				continue
			}

			for pos, border := range eligible {
				if border {
					continue
				}

				baseX, baseY, baseZ := s.getRandomSpawningPointInChunk(pos.ChunkXPos, pos.ChunkZPos)
				if s.isBlockNormalCube(baseX, baseY, baseZ) || s.blockMaterialAt(baseX, baseY, baseZ) != creatureCfg.material {
					continue
				}

				groupSpawnedInChunk := 0
				stopChunk := false

				for pack := 0; pack < mobSpawnPackAttempts && !stopChunk; pack++ {
					x := baseX
					y := baseY
					z := baseZ
					var selected *spawnListEntry
					var spawnGroupData any

					for retry := 0; retry < mobSpawnRetries; retry++ {
						x += int(s.mobRand.NextInt(mobSpawnMaxOffset)) - int(s.mobRand.NextInt(mobSpawnMaxOffset))
						y += int(s.mobRand.NextInt(1)) - int(s.mobRand.NextInt(1))
						z += int(s.mobRand.NextInt(mobSpawnMaxOffset)) - int(s.mobRand.NextInt(mobSpawnMaxOffset))

						if !s.canCreatureTypeSpawnAtLocation(creatureCfg, x, y, z) {
							continue
						}

						spawnFX := float64(x) + 0.5
						spawnFY := float64(y)
						spawnFZ := float64(z) + 0.5
						if s.closestPlayerDistanceSq(players, spawnFX, spawnFY, spawnFZ) < mobSpawnMinDistSq {
							continue
						}
						dx := spawnFX - float64(spawnX)
						dy := spawnFY - float64(spawnY)
						dz := spawnFZ - float64(spawnZ)
						if dx*dx+dy*dy+dz*dz < mobSpawnFromWorldSq {
							continue
						}

						if selected == nil {
							selected = s.spawnRandomCreature(creatureCfg.creatureType, x, y, z)
							if selected == nil {
								break
							}
						}

						if !s.entityCanSpawnHere(selected, creatureCfg, x, y, z) {
							continue
						}

						yaw := s.mobRand.NextFloat() * 360.0
						mob, nextGroupData := s.spawnMobWithGroupData(selected, spawnFX, spawnFY, spawnFZ, yaw, spawnGroupData)
						spawnGroupData = nextGroupData
						if mob == nil {
							continue
						}
						groupSpawnedInChunk++
						spawnedTotal++

						if groupSpawnedInChunk >= selected.maxPerChunk {
							stopChunk = true
							break
						}
					}
				}
			}
		}
	}

	return spawnedTotal
}

func (s *StatusServer) activeSpawnPlayers() []spawnPlayerSnapshot {
	s.activeMu.RLock()
	sessions := make([]*loginSession, 0, len(s.activePlayers))
	for session := range s.activePlayers {
		sessions = append(sessions, session)
	}
	s.activeMu.RUnlock()

	out := make([]spawnPlayerSnapshot, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		session.stateMu.Lock()
		alive := session.playerRegistered && !session.playerDead && session.entityID != 0
		x := session.playerX
		y := session.playerY
		z := session.playerZ
		session.stateMu.Unlock()
		if !alive {
			continue
		}
		out = append(out, spawnPlayerSnapshot{
			session: session,
			x:       x,
			y:       y,
			z:       z,
			chunkX:  chunkCoordFromPos(x),
			chunkZ:  chunkCoordFromPos(z),
		})
	}
	return out
}

func (s *StatusServer) closestPlayerDistanceSq(players []spawnPlayerSnapshot, x, y, z float64) float64 {
	best := math.MaxFloat64
	for _, pl := range players {
		dx := pl.x - x
		dy := pl.y - y
		dz := pl.z - z
		dist := dx*dx + dy*dy + dz*dz
		if dist < best {
			best = dist
		}
	}
	return best
}

func (s *StatusServer) getRandomSpawningPointInChunk(chunkX, chunkZ int32) (int, int, int) {
	ch := s.world.getChunk(chunkX, chunkZ)
	maxY := 256
	if ch != nil {
		maxY = ch.GetTopFilledSegment() + 16 - 1
	}
	if maxY <= 0 {
		maxY = 1
	}

	x := int(chunkX)*16 + int(s.mobRand.NextInt(16))
	z := int(chunkZ)*16 + int(s.mobRand.NextInt(16))
	y := int(s.mobRand.NextInt(maxY))
	return x, y, z
}

func (s *StatusServer) spawnRandomCreature(creatureType mobCreatureType, x, y, z int) *spawnListEntry {
	biomeID := s.biomeIDAt(x, z)
	entries := spawnEntriesForBiomeAndType(biomeID, creatureType)
	if len(entries) == 0 {
		return nil
	}

	total := 0
	for _, e := range entries {
		total += e.weight
	}
	if total <= 0 {
		return nil
	}

	pick := int(s.mobRand.NextInt(total))
	for i := range entries {
		pick -= entries[i].weight
		if pick < 0 {
			entry := entries[i]
			return &entry
		}
	}
	return nil
}

func (s *StatusServer) biomeIDAt(x, z int) byte {
	ch := s.world.getChunk(int32(x>>4), int32(z>>4))
	if ch == nil {
		return 1
	}
	localX := x & 15
	localZ := z & 15
	biomes := ch.GetBiomeArray()
	if len(biomes) != 256 {
		return 1
	}
	id := biomes[localZ<<4|localX]
	if id == 0xFF {
		return 1
	}
	return id
}

func spawnEntriesForBiomeAndType(biomeID byte, creatureType mobCreatureType) []spawnListEntry {
	switch creatureType {
	case creatureTypeMonster:
		return defaultMonsterSpawns
	case creatureTypeCreature:
		if biomeID == 1 {
			return plainsCreatureSpawns
		}
		return defaultCreatureSpawns
	case creatureTypeAmbient:
		return defaultAmbientSpawns
	case creatureTypeWater:
		return defaultWaterSpawns
	default:
		return nil
	}
}

func (s *StatusServer) spawnMob(entry *spawnListEntry, x, y, z float64, yaw float32) *trackedMob {
	mob, _ := s.spawnMobWithGroupData(entry, x, y, z, yaw, nil)
	return mob
}

func (s *StatusServer) spawnMobWithGroupData(entry *spawnListEntry, x, y, z float64, yaw float32, groupData any) (*trackedMob, any) {
	if entry == nil {
		return nil, groupData
	}

	entityID := s.nextEntityID.Add(1)
	mob := &trackedMob{
		EntityID:       entityID,
		EntityType:     entry.entityType,
		CreatureType:   creatureTypeFromEntity(entry.entityType),
		X:              x,
		Y:              y,
		Z:              z,
		Yaw:            yaw,
		HeadYaw:        yaw,
		SeenBy:         make(map[*loginSession]struct{}),
		creeperState:   -1,
		creeperFuse:    creeperFuseTimeDefault,
		rangedCooldown: -1,
	}
	if entry.entityType == entityTypeCreeper {
		mob.creeperRadius = creeperExplosionRadiusNormal
	}
	if entry.entityType == entityTypeSlime {
		if entry.fixedSlime > 0 {
			mob.slimeSize = entry.fixedSlime
		} else {
			mob.slimeSize = 1 << uint(s.mobRand.NextInt(3))
		}
		mob.slimeJumpDelay = int(s.mobRand.NextInt(20)) + 10
	}
	if entry.entityType == entityTypeSheep {
		mob.sheepColor = s.randomSheepFleeceColor()
		mob.sheepSheared = false
	}
	groupData = s.onMobSpawnWithEgg(mob, groupData)
	mob.Health = s.mobMaxHealth(mob)
	mob.wanderYaw = float64(yaw) * math.Pi / 180.0
	mob.wanderTicks = mobWanderMinTicks + int(s.mobRand.NextInt(mobWanderExtraTicks))
	mob.wanderPause = int(s.mobRand.NextInt(mobWanderPauseExtra))
	mob.wanderPitch = (s.mobRand.NextDouble() - 0.5) * 0.5

	s.mobMu.Lock()
	s.mobs[entityID] = mob
	s.mobMu.Unlock()
	s.updateMobVisibility(mob, nil)
	return mob, groupData
}

func (s *StatusServer) onMobSpawnWithEgg(mob *trackedMob, groupData any) any {
	if mob == nil {
		return groupData
	}

	switch mob.EntityType {
	case entityTypeSkeleton:
		// Translation reference:
		// - net.minecraft.src.EntitySkeleton#onSpawnWithEgg(EntityLivingData)
		tension := s.locationTensionFactor(mob.X, mob.Y, mob.Z)
		mob.canPickUpLoot = s.mobRand.NextFloat() < float32(0.55*tension)
		s.addRandomArmorToMob(mob, tension)
		mob.heldItemID = mobItemIDBow
		s.tryEquipHalloweenHead(mob)
		return groupData
	case entityTypeZombie:
		// Translation reference:
		// - net.minecraft.src.EntityZombie#onSpawnWithEgg(EntityLivingData)
		tension := s.locationTensionFactor(mob.X, mob.Y, mob.Z)
		mob.canPickUpLoot = s.mobRand.NextFloat() < float32(0.55*tension)

		zombieData, ok := groupData.(*zombieSpawnGroupData)
		if !ok || zombieData == nil {
			zombieData = &zombieSpawnGroupData{
				child:    s.mobRand.NextFloat() < 0.05,
				villager: s.mobRand.NextFloat() < 0.05,
			}
		}
		mob.zombieChild = zombieData.child
		mob.zombieVillager = zombieData.villager

		s.addRandomArmorToMob(mob, tension)
		s.addRandomZombieWeapon(mob)
		s.tryEquipHalloweenHead(mob)
		return zombieData
	default:
		return groupData
	}
}

func (s *StatusServer) locationTensionFactor(x, y, z float64) float64 {
	// Translation reference:
	// - net.minecraft.src.World#getLocationTensionFactor(double,double,double)
	// - net.minecraft.src.World#getTensionFactorForBlock(int,int,int)
	difficulty := serverDifficulty
	isHard := difficulty == 3
	value := 0.0

	blockX := int(math.Floor(x))
	blockY := int(math.Floor(y))
	blockZ := int(math.Floor(z))
	if blockY >= 0 && blockY < 256 {
		ch := s.world.getChunk(int32(blockX>>4), int32(blockZ>>4))
		if ch != nil {
			inhabited := clampFloat64(float64(ch.InhabitedTime)/3600000.0, 0.0, 1.0)
			if isHard {
				value += inhabited
			} else {
				value += inhabited * 0.75
			}
		}
		value += float64(s.currentMoonPhaseFactor()) * 0.25
	}

	if difficulty < 2 {
		value *= float64(difficulty) / 2.0
	}

	maxValue := 1.0
	if isHard {
		maxValue = 1.5
	}
	return clampFloat64(value, 0.0, maxValue)
}

func (s *StatusServer) currentMoonPhaseFactor() float32 {
	// Translation reference:
	// - net.minecraft.src.World#getCurrentMoonPhaseFactor()
	// - net.minecraft.src.WorldProvider#getMoonPhase(long)
	_, worldTime := s.CurrentWorldTime()
	moonPhase := int((worldTime / 24000) % 8)
	if moonPhase < 0 {
		moonPhase += 8
	}
	phases := [...]float32{1.0, 0.75, 0.5, 0.25, 0.0, 0.25, 0.5, 0.75}
	return phases[moonPhase]
}

func (s *StatusServer) addRandomArmorToMob(mob *trackedMob, tension float64) {
	if mob == nil {
		return
	}
	if s.mobRand.NextFloat() >= float32(0.15*tension) {
		return
	}

	// Translation reference:
	// - net.minecraft.src.EntityLiving#addRandomArmor()
	armorTier := int(s.mobRand.NextInt(2))
	stopChance := 0.25
	if serverDifficulty == 3 {
		stopChance = 0.1
	}
	for i := 0; i < 3; i++ {
		if s.mobRand.NextFloat() < 0.095 {
			armorTier++
		}
	}

	for slot := 3; slot >= 0; slot-- {
		armorSlot := slot + 1
		if slot < 3 && s.mobRand.NextFloat() < float32(stopChance) {
			break
		}
		if s.mobArmorSlotItemID(mob, armorSlot) != 0 {
			continue
		}
		itemID := armorItemForSlot(armorSlot, armorTier)
		if itemID != 0 {
			s.setMobArmorSlotItemID(mob, armorSlot, itemID)
		}
	}
}

func (s *StatusServer) addRandomZombieWeapon(mob *trackedMob) {
	if mob == nil || mob.EntityType != entityTypeZombie {
		return
	}
	// Translation reference:
	// - net.minecraft.src.EntityZombie#addRandomArmor()
	chance := float32(0.01)
	if serverDifficulty == 3 {
		chance = 0.05
	}
	if s.mobRand.NextFloat() >= chance {
		return
	}
	if s.mobRand.NextInt(3) == 0 {
		mob.heldItemID = mobItemIDIronSword
		return
	}
	mob.heldItemID = mobItemIDIronShovel
}

func (s *StatusServer) tryEquipHalloweenHead(mob *trackedMob) {
	if mob == nil || mob.helmetItemID != 0 || s.now == nil {
		return
	}
	// Translation reference:
	// - net.minecraft.src.EntitySkeleton#onSpawnWithEgg(EntityLivingData)
	// - net.minecraft.src.EntityZombie#onSpawnWithEgg(EntityLivingData)
	now := s.now()
	if now.Month() != time.October || now.Day() != 31 {
		return
	}
	if s.mobRand.NextFloat() >= 0.25 {
		return
	}
	if s.mobRand.NextFloat() < 0.1 {
		mob.helmetItemID = mobItemIDJackOLantern
	} else {
		mob.helmetItemID = mobItemIDPumpkin
	}
	mob.helmetDamage = 0
}

func (s *StatusServer) mobArmorSlotItemID(mob *trackedMob, slot int) int16 {
	if mob == nil {
		return 0
	}
	switch slot {
	case 1:
		return mob.bootsItemID
	case 2:
		return mob.legsItemID
	case 3:
		return mob.chestItemID
	case 4:
		return mob.helmetItemID
	default:
		return 0
	}
}

func (s *StatusServer) setMobArmorSlotItemID(mob *trackedMob, slot int, itemID int16) {
	if mob == nil {
		return
	}
	switch slot {
	case 1:
		mob.bootsItemID = itemID
	case 2:
		mob.legsItemID = itemID
	case 3:
		mob.chestItemID = itemID
	case 4:
		mob.helmetItemID = itemID
		mob.helmetDamage = 0
	}
}

func armorItemForSlot(slot, tier int) int16 {
	switch slot {
	case 4: // helmet
		switch tier {
		case 0:
			return mobItemIDHelmetLeather
		case 1:
			return mobItemIDHelmetGold
		case 2:
			return mobItemIDHelmetChain
		case 3:
			return mobItemIDHelmetIron
		case 4:
			return mobItemIDHelmetDiamond
		}
	case 3: // chestplate
		switch tier {
		case 0:
			return mobItemIDChestLeather
		case 1:
			return mobItemIDChestGold
		case 2:
			return mobItemIDChestChain
		case 3:
			return mobItemIDChestIron
		case 4:
			return mobItemIDChestDiamond
		}
	case 2: // leggings
		switch tier {
		case 0:
			return mobItemIDLegsLeather
		case 1:
			return mobItemIDLegsGold
		case 2:
			return mobItemIDLegsChain
		case 3:
			return mobItemIDLegsIron
		case 4:
			return mobItemIDLegsDiamond
		}
	case 1: // boots
		switch tier {
		case 0:
			return mobItemIDBootsLeather
		case 1:
			return mobItemIDBootsGold
		case 2:
			return mobItemIDBootsChain
		case 3:
			return mobItemIDBootsIron
		case 4:
			return mobItemIDBootsDiamond
		}
	}
	return 0
}

func (s *StatusServer) randomSheepFleeceColor() int8 {
	// Translation reference:
	// - net.minecraft.src.EntitySheep#getRandomFleeceColor(Random)
	roll := int(s.mobRand.NextInt(100))
	switch {
	case roll < 5:
		return 15
	case roll < 10:
		return 7
	case roll < 15:
		return 8
	case roll < 18:
		return 12
	case s.mobRand.NextInt(500) == 0:
		return 6
	default:
		return 0
	}
}

func creatureTypeFromEntity(entityType int8) mobCreatureType {
	switch entityType {
	case entityTypeCreeper, entityTypeSkeleton, entityTypeSpider, entityTypeZombie, entityTypeSlime, entityTypeEnderman:
		return creatureTypeMonster
	case entityTypePig, entityTypeSheep, entityTypeCow, entityTypeChicken, entityTypeHorse:
		return creatureTypeCreature
	case entityTypeBat:
		return creatureTypeAmbient
	case entityTypeSquid:
		return creatureTypeWater
	default:
		return creatureTypeMonster
	}
}

func (s *StatusServer) countMobsByCreatureType(creatureType mobCreatureType) int {
	s.mobMu.Lock()
	defer s.mobMu.Unlock()
	total := 0
	for _, mob := range s.mobs {
		if mob != nil && mob.CreatureType == creatureType {
			total++
		}
	}
	return total
}

func (s *StatusServer) updateMobVisibility(mob *trackedMob, movementPacket protocol.Packet) {
	if mob == nil {
		return
	}

	chunkX, chunkZ := mob.chunkCoords()
	targets := s.activeSessionsExcept(nil)
	spawnPacket := mob.spawnPacket()
	destroyPacket := &protocol.Packet29DestroyEntity{EntityIDs: []int32{mob.EntityID}}

	active := make(map[*loginSession]struct{}, len(targets))
	for _, target := range targets {
		active[target] = struct{}{}
	}

	s.mobMu.Lock()
	if mob.SeenBy == nil {
		mob.SeenBy = make(map[*loginSession]struct{})
	}
	for viewer := range mob.SeenBy {
		if _, ok := active[viewer]; !ok {
			delete(mob.SeenBy, viewer)
		}
	}
	s.mobMu.Unlock()

	for _, target := range targets {
		if target == nil {
			continue
		}
		shouldSee := target.isWatchingChunk(chunkX, chunkZ)

		s.mobMu.Lock()
		_, wasSeen := mob.SeenBy[target]
		s.mobMu.Unlock()

		switch {
		case shouldSee && !wasSeen:
			if target.sendPacket(spawnPacket) {
				s.mobMu.Lock()
				mob.SeenBy[target] = struct{}{}
				s.mobMu.Unlock()
			}
		case !shouldSee && wasSeen:
			if target.sendPacket(destroyPacket) {
				s.mobMu.Lock()
				delete(mob.SeenBy, target)
				s.mobMu.Unlock()
			}
		case shouldSee && wasSeen:
			if movementPacket != nil {
				_ = target.sendPacket(movementPacket)
			}
		}
	}
}

func (s *StatusServer) broadcastMobMetadata(mob *trackedMob) {
	if mob == nil {
		return
	}
	packet := mob.metadataPacket()
	chunkX, chunkZ := mob.chunkCoords()
	s.broadcastEntityPacketToWatchers(packet, chunkX, chunkZ, nil)
}

func (s *StatusServer) isBlockNormalCube(x, y, z int) bool {
	id, _ := s.world.getBlock(x, y, z)
	return block.BlocksMovement(id)
}

func (s *StatusServer) blockMaterialAt(x, y, z int) spawnMaterial {
	id, _ := s.world.getBlock(x, y, z)
	if block.IsLiquid(id) {
		return spawnMaterialWater
	}
	return spawnMaterialAir
}

func (s *StatusServer) canCreatureTypeSpawnAtLocation(creatureCfg creatureTypeConfig, x, y, z int) bool {
	if y <= 0 || y >= 255 {
		return false
	}

	if creatureCfg.material == spawnMaterialWater {
		return s.blockMaterialAt(x, y, z) == spawnMaterialWater &&
			s.blockMaterialAt(x, y-1, z) == spawnMaterialWater &&
			!s.isBlockNormalCube(x, y+1, z)
	}

	if !s.isBlockNormalCube(x, y-1, z) {
		return false
	}

	belowID, _ := s.world.getBlock(x, y-1, z)
	if belowID == 7 { // bedrock
		return false
	}

	id, _ := s.world.getBlock(x, y, z)
	return !s.isBlockNormalCube(x, y, z) &&
		!block.IsLiquid(id) &&
		!s.isBlockNormalCube(x, y+1, z)
}

func (s *StatusServer) entityCanSpawnHere(entry *spawnListEntry, creatureCfg creatureTypeConfig, x, y, z int) bool {
	if entry == nil {
		return false
	}
	switch entry.entityType {
	case entityTypePig, entityTypeSheep, entityTypeCow, entityTypeChicken, entityTypeHorse:
		// Translation reference:
		// - net.minecraft.src.EntityAnimal#getCanSpawnHere()
		belowID, _ := s.world.getBlock(x, y-1, z)
		if belowID != 2 { // grass
			return false
		}
		return s.fullBlockLightValue(x, y, z) > 8
	case entityTypeSquid:
		// Translation reference:
		// - net.minecraft.src.EntitySquid#getCanSpawnHere()
		return y > 45 && y < 63
	case entityTypeBat:
		// Translation reference:
		// - net.minecraft.src.EntityBat#getCanSpawnHere()
		if y >= 63 {
			return false
		}
		if s.mobRand.NextBoolean() {
			return false
		}
		return s.blockLightValue(x, y, z) <= int(s.mobRand.NextInt(4))
	default:
		if creatureCfg.creatureType == creatureTypeMonster {
			// Translation reference:
			// - net.minecraft.src.EntityMob#isValidLightLevel()
			// - net.minecraft.src.EntityMob#getCanSpawnHere()
			return s.isValidMonsterLightLevel(x, y, z)
		}
		return true
	}
}

func (s *StatusServer) isValidMonsterLightLevel(x, y, z int) bool {
	if s.savedLightValue(chunk.EnumSkyBlockSky, x, y, z) > int(s.mobRand.NextInt(32)) {
		return false
	}
	return s.blockLightValue(x, y, z) <= int(s.mobRand.NextInt(8))
}

func (s *StatusServer) fullBlockLightValue(x, y, z int) int {
	// Baseline equivalent of World#getFullBlockLightValue for spawn checks.
	return s.blockLightValue(x, y, z)
}

func (s *StatusServer) blockLightValue(x, y, z int) int {
	if y < 0 {
		return 0
	}
	if y >= 256 {
		y = 255
	}

	ch := s.world.getChunk(int32(x>>4), int32(z>>4))
	localX := x & 15
	localZ := z & 15

	sky := 0
	if !s.world.hasNoSky {
		sky = ch.GetSavedLightValue(chunk.EnumSkyBlockSky, localX, y, localZ)
	}
	sky -= s.skylightSubtracted()
	if sky < 0 {
		sky = 0
	}

	blockLight := ch.GetSavedLightValue(chunk.EnumSkyBlockBlock, localX, y, localZ)
	if blockLight > sky {
		return blockLight
	}
	return sky
}

func (s *StatusServer) savedLightValue(kind chunk.EnumSkyBlock, x, y, z int) int {
	if y < 0 {
		return 0
	}
	if y >= 256 {
		y = 255
	}
	ch := s.world.getChunk(int32(x>>4), int32(z>>4))
	return ch.GetSavedLightValue(kind, x&15, y, z&15)
}

func (s *StatusServer) skylightSubtracted() int {
	_, worldTime := s.CurrentWorldTime()
	timeOfDay := int(worldTime % 24000)
	celestial := (float64(timeOfDay)+1.0)/24000.0 - 0.25
	if celestial < 0.0 {
		celestial++
	}
	if celestial > 1.0 {
		celestial--
	}
	base := celestial
	celestial = 1.0 - (math.Cos(celestial*math.Pi)+1.0)/2.0
	celestial = base + (celestial-base)/3.0

	v := 1.0 - (math.Cos(celestial*math.Pi*2.0)*2.0 + 0.5)
	if v < 0.0 {
		v = 0.0
	}
	if v > 1.0 {
		v = 1.0
	}
	return int(v * 11.0)
}
