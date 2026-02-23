package server

import (
	"math"

	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/util"
)

const (
	entityTypeArrow int8 = 60

	arrowBaseDamage       = 2.0
	arrowMaxLifetimeTicks = 1200
	arrowGravityPerTick   = 0.05
	arrowDragPerTick      = 0.99
	arrowHitRadius        = 0.9

	arrowPickupNone         = 0
	arrowPickupSurvival     = 1
	arrowPickupCreativeOnly = 2
)

type trackedProjectile struct {
	EntityID        int32
	Type            int8
	ShooterEntityID int32

	X float64
	Y float64
	Z float64

	MotionX float64
	MotionY float64
	MotionZ float64

	Yaw   float32
	Pitch float32

	Damage   float64
	Critical bool
	AgeTicks int
	InGround bool

	InTileX int32
	InTileY int32
	InTileZ int32

	ArrowShake    int
	CanBePickedUp int8
	SeenBy        map[*loginSession]struct{}

	rand *util.JavaRandom
}

func newArrowProjectile(entityID, shooterEntityID int32, shooterX, shooterY, shooterZ float64, shooterYaw, shooterPitch float32, bowVelocity float64, critical bool, canBePickedUp int8) *trackedProjectile {
	// Deterministic per-entity seed for stable behavior/testing while keeping JavaRandom math paths.
	seed := int64(entityID)*341873128712 + int64(shooterEntityID)*132897987541
	rng := util.NewJavaRandom(seed)

	arrow := &trackedProjectile{
		EntityID:        entityID,
		Type:            entityTypeArrow,
		ShooterEntityID: shooterEntityID,
		X:               shooterX,
		Y:               shooterY + defaultPlayerEyeY,
		Z:               shooterZ,
		Yaw:             shooterYaw,
		Pitch:           shooterPitch,
		Damage:          arrowBaseDamage,
		Critical:        critical,
		CanBePickedUp:   canBePickedUp,
		SeenBy:          make(map[*loginSession]struct{}),
		rand:            rng,
	}

	// Translated from EntityArrow(World, EntityLivingBase, float):
	// initial position offsets + heading + setThrowableHeading.
	yawRad := float64(shooterYaw) * math.Pi / 180.0
	pitchRad := float64(shooterPitch) * math.Pi / 180.0
	arrow.X -= math.Cos(yawRad) * 0.16
	arrow.Y -= 0.10000000149011612
	arrow.Z -= math.Sin(yawRad) * 0.16

	arrow.MotionX = -math.Sin(yawRad) * math.Cos(pitchRad)
	arrow.MotionZ = math.Cos(yawRad) * math.Cos(pitchRad)
	arrow.MotionY = -math.Sin(pitchRad)
	arrow.setThrowableHeading(arrow.MotionX, arrow.MotionY, arrow.MotionZ, bowVelocity*1.5, 1.0)
	return arrow
}

func (p *trackedProjectile) setThrowableHeading(x, y, z, velocity, inaccuracy float64) {
	// Translated from EntityArrow#setThrowableHeading.
	norm := math.Sqrt(x*x + y*y + z*z)
	if norm <= 1.0e-9 {
		return
	}

	x /= norm
	y /= norm
	z /= norm

	sign := func() float64 {
		if p.rand.NextBoolean() {
			return -1
		}
		return 1
	}
	x += p.rand.NextGaussian() * sign() * 0.007499999832361937 * inaccuracy
	y += p.rand.NextGaussian() * sign() * 0.007499999832361937 * inaccuracy
	z += p.rand.NextGaussian() * sign() * 0.007499999832361937 * inaccuracy

	x *= velocity
	y *= velocity
	z *= velocity

	p.MotionX = x
	p.MotionY = y
	p.MotionZ = z

	horiz := math.Sqrt(x*x + z*z)
	p.Yaw = float32(math.Atan2(x, z) * 180.0 / math.Pi)
	p.Pitch = float32(math.Atan2(y, horiz) * 180.0 / math.Pi)
}

func (p *trackedProjectile) impactDamage() float32 {
	speed := math.Sqrt(p.MotionX*p.MotionX + p.MotionY*p.MotionY + p.MotionZ*p.MotionZ)
	damage := int(math.Ceil(speed * p.Damage))
	if p.Critical && damage > 0 {
		damage += int(p.rand.NextInt(damage/2 + 2))
	}
	if damage < 0 {
		damage = 0
	}
	return float32(damage)
}

func (p *trackedProjectile) spawnPacket() *protocol.Packet23VehicleSpawn {
	vel := protocol.NewPacket28EntityVelocity(p.EntityID, p.MotionX, p.MotionY, p.MotionZ)
	return &protocol.Packet23VehicleSpawn{
		EntityID:        p.EntityID,
		Type:            p.Type,
		XPosition:       toPacketPosition(p.X),
		YPosition:       toPacketPosition(p.Y),
		ZPosition:       toPacketPosition(p.Z),
		Pitch:           toPacketAngle(p.Pitch),
		Yaw:             toPacketAngle(p.Yaw),
		ThrowerEntityID: p.ShooterEntityID,
		SpeedX:          vel.MotionX,
		SpeedY:          vel.MotionY,
		SpeedZ:          vel.MotionZ,
	}
}

func (p *trackedProjectile) teleportPacket() *protocol.Packet34EntityTeleport {
	return &protocol.Packet34EntityTeleport{
		EntityID:  p.EntityID,
		XPosition: toPacketPosition(p.X),
		YPosition: toPacketPosition(p.Y),
		ZPosition: toPacketPosition(p.Z),
		Yaw:       toPacketAngle(p.Yaw),
		Pitch:     toPacketAngle(p.Pitch),
	}
}

func (s *StatusServer) spawnArrowFromPlayer(shooter *loginSession, bowVelocity float64, critical bool) {
	if shooter == nil || bowVelocity <= 0 {
		return
	}

	var (
		shooterEntityID int32
		shooterX        float64
		shooterY        float64
		shooterZ        float64
		shooterYaw      float32
		shooterPitch    float32
		canBePickedUp   int8
	)
	shooter.stateMu.Lock()
	if shooter.playerDead || !shooter.playerRegistered || shooter.entityID == 0 {
		shooter.stateMu.Unlock()
		return
	}
	shooterEntityID = shooter.entityID
	shooterX = shooter.playerX
	shooterY = shooter.playerY
	shooterZ = shooter.playerZ
	shooterYaw = shooter.playerYaw
	shooterPitch = shooter.playerPitch
	if shooter.gameType == 1 {
		canBePickedUp = arrowPickupCreativeOnly
	} else {
		canBePickedUp = arrowPickupSurvival
	}
	shooter.stateMu.Unlock()

	entityID := s.nextEntityID.Add(1)
	arrow := newArrowProjectile(entityID, shooterEntityID, shooterX, shooterY, shooterZ, shooterYaw, shooterPitch, bowVelocity, critical, canBePickedUp)

	s.projectileMu.Lock()
	s.projectiles[entityID] = arrow
	s.projectileMu.Unlock()

	s.updateProjectileVisibility(arrow, nil)
}

func (s *StatusServer) spawnArrowFromMob(shooter *trackedMob, target *loginSession, power float64) {
	if shooter == nil || target == nil {
		return
	}
	power = clampFloat64(power, 0.1, 1.0)

	var (
		shooterEntityID int32
		shooterX        float64
		shooterY        float64
		shooterZ        float64
	)
	s.mobMu.Lock()
	live, ok := s.mobs[shooter.EntityID]
	if !ok || live != shooter {
		s.mobMu.Unlock()
		return
	}
	shooterEntityID = shooter.EntityID
	shooterX = shooter.X
	shooterY = shooter.Y
	shooterZ = shooter.Z
	s.mobMu.Unlock()

	target.stateMu.Lock()
	alive := target.playerRegistered && !target.playerDead && target.entityID != 0
	targetX := target.playerX
	targetY := target.playerY
	targetZ := target.playerZ
	target.stateMu.Unlock()
	if !alive {
		return
	}

	dx := targetX - shooterX
	dz := targetZ - shooterZ
	horiz := math.Sqrt(dx*dx + dz*dz)
	if horiz <= 1.0e-9 {
		return
	}
	dy := (targetY + defaultPlayerEyeY) - (shooterY + defaultPlayerEyeY)
	yaw := float32(math.Atan2(dx, dz) * 180.0 / math.Pi)
	pitch := float32(-math.Atan2(dy, horiz) * 180.0 / math.Pi)

	entityID := s.nextEntityID.Add(1)
	arrow := newArrowProjectile(entityID, shooterEntityID, shooterX, shooterY, shooterZ, yaw, pitch, 1.6, false, arrowPickupNone)
	arrow.Damage = power*2.0 + arrow.rand.NextGaussian()*0.25 + float64(s.currentDifficulty())*0.11
	if arrow.Damage < 0 {
		arrow.Damage = 0
	}

	s.projectileMu.Lock()
	s.projectiles[entityID] = arrow
	s.projectileMu.Unlock()

	s.updateProjectileVisibility(arrow, nil)
}

func (s *StatusServer) TickProjectiles() {
	s.projectileMu.Lock()
	if len(s.projectiles) == 0 {
		s.projectileMu.Unlock()
		return
	}
	projectiles := make([]*trackedProjectile, 0, len(s.projectiles))
	for _, projectile := range s.projectiles {
		if projectile != nil {
			projectiles = append(projectiles, projectile)
		}
	}
	s.projectileMu.Unlock()

	for _, projectile := range projectiles {
		s.tickProjectile(projectile)
	}
}

func (s *StatusServer) tickProjectile(projectile *trackedProjectile) {
	if projectile == nil {
		return
	}

	projectile.AgeTicks++
	if projectile.AgeTicks > arrowMaxLifetimeTicks {
		s.destroyProjectile(projectile)
		return
	}
	if projectile.ArrowShake > 0 {
		projectile.ArrowShake--
	}

	if projectile.InGround {
		s.updateProjectileVisibility(projectile, nil)
		if s.tryPickupArrow(projectile) {
			return
		}
		return
	}

	oldX := projectile.X
	oldY := projectile.Y
	oldZ := projectile.Z
	nextX := oldX + projectile.MotionX
	nextY := oldY + projectile.MotionY
	nextZ := oldZ + projectile.MotionZ

	blockID, _ := s.world.getBlock(int(math.Floor(nextX)), int(math.Floor(nextY)), int(math.Floor(nextZ)))
	if blockID != 0 {
		projectile.X = nextX
		projectile.Y = nextY
		projectile.Z = nextZ
		projectile.InGround = true
		projectile.InTileX = int32(math.Floor(nextX))
		projectile.InTileY = int32(math.Floor(nextY))
		projectile.InTileZ = int32(math.Floor(nextZ))
		projectile.MotionX = 0
		projectile.MotionY = 0
		projectile.MotionZ = 0
		projectile.Critical = false
		projectile.ArrowShake = 7
		s.updateProjectileVisibility(projectile, projectile.teleportPacket())
		if s.tryPickupArrow(projectile) {
			return
		}
		return
	}

	var hitTarget *loginSession
	var hitTargetDistSq float64
	hitTarget, hitTargetDistSq = s.findProjectileHitTarget(projectile, oldX, oldY, oldZ, nextX, nextY, nextZ)

	var hitMob *trackedMob
	var hitMobDistSq float64
	hitMob, hitMobDistSq = s.findProjectileHitMob(projectile, oldX, oldY, oldZ, nextX, nextY, nextZ)

	if hitTarget != nil || hitMob != nil {
		damage := projectile.impactDamage()
		shouldHitMob := false
		if hitMob != nil {
			if hitTarget == nil || hitMobDistSq <= hitTargetDistSq {
				shouldHitMob = true
			}
		}
		if shouldHitMob {
			if damage > 0 {
				mob, damaged, died, hurtStatus := s.applyDamageToMob(hitMob.EntityID, damage)
				if mob != nil && damaged && hurtStatus {
					s.broadcastMobEntityStatus(mob, 2)
				}
				if mob != nil && died {
					s.broadcastMobEntityStatus(mob, 3)
					s.killMob(mob, projectile.CanBePickedUp != arrowPickupNone)
				}
			}
		} else if hitTarget != nil {
			if damage > 0 {
				damaged, died, hurtStatus := hitTarget.applyIncomingDamage(damage, true)
				if damaged && hurtStatus {
					hitTarget.broadcastEntityStatusToSelfAndWatchers(2)
				}
				if died {
					hitTarget.broadcastEntityStatusToSelfAndWatchers(3)
					hitTarget.sendSystemChat("You died.")
				}
			}
		}
		s.destroyProjectile(projectile)
		return
	}

	projectile.X = nextX
	projectile.Y = nextY
	projectile.Z = nextZ
	projectile.MotionX *= arrowDragPerTick
	projectile.MotionY *= arrowDragPerTick
	projectile.MotionZ *= arrowDragPerTick
	projectile.MotionY -= arrowGravityPerTick

	horiz := math.Sqrt(projectile.MotionX*projectile.MotionX + projectile.MotionZ*projectile.MotionZ)
	projectile.Yaw = float32(math.Atan2(projectile.MotionX, projectile.MotionZ) * 180.0 / math.Pi)
	projectile.Pitch = float32(math.Atan2(projectile.MotionY, horiz) * 180.0 / math.Pi)

	s.updateProjectileVisibility(projectile, projectile.teleportPacket())
}

func (s *StatusServer) destroyProjectile(projectile *trackedProjectile) {
	if projectile == nil {
		return
	}

	s.projectileMu.Lock()
	seenTargets := make([]*loginSession, 0, len(projectile.SeenBy))
	for session := range projectile.SeenBy {
		seenTargets = append(seenTargets, session)
	}
	projectile.SeenBy = make(map[*loginSession]struct{})
	if _, exists := s.projectiles[projectile.EntityID]; exists {
		delete(s.projectiles, projectile.EntityID)
	}
	s.projectileMu.Unlock()
	destroy := &protocol.Packet29DestroyEntity{EntityIDs: []int32{projectile.EntityID}}
	for _, target := range seenTargets {
		_ = target.sendPacket(destroy)
	}
}

func (s *StatusServer) updateProjectileVisibility(projectile *trackedProjectile, movementPacket protocol.Packet) {
	if projectile == nil {
		return
	}
	s.projectileMu.Lock()
	if projectile.SeenBy == nil {
		projectile.SeenBy = make(map[*loginSession]struct{})
	}
	s.projectileMu.Unlock()

	chunkX := chunkCoordFromPos(projectile.X)
	chunkZ := chunkCoordFromPos(projectile.Z)
	targets := s.activeSessionsExcept(nil)
	spawnPacket := projectile.spawnPacket()
	destroyPacket := &protocol.Packet29DestroyEntity{EntityIDs: []int32{projectile.EntityID}}

	for _, target := range targets {
		if target == nil {
			continue
		}
		shouldSee := target.isWatchingChunk(chunkX, chunkZ)

		s.projectileMu.Lock()
		_, wasSeen := projectile.SeenBy[target]
		s.projectileMu.Unlock()

		switch {
		case shouldSee && !wasSeen:
			if target.sendPacket(spawnPacket) {
				s.projectileMu.Lock()
				projectile.SeenBy[target] = struct{}{}
				s.projectileMu.Unlock()
			}
		case !shouldSee && wasSeen:
			if target.sendPacket(destroyPacket) {
				s.projectileMu.Lock()
				delete(projectile.SeenBy, target)
				s.projectileMu.Unlock()
			}
		case shouldSee && wasSeen:
			if movementPacket != nil {
				_ = target.sendPacket(movementPacket)
			}
		}
	}
}

func (s *StatusServer) tryPickupArrow(projectile *trackedProjectile) bool {
	if projectile == nil || !projectile.InGround || projectile.ArrowShake > 0 {
		return false
	}
	targets := s.activeSessionsExcept(nil)
	for _, target := range targets {
		if target == nil {
			continue
		}
		target.stateMu.Lock()
		targetDead := target.playerDead || target.playerHealth <= 0
		targetX := target.playerX
		targetY := target.playerY
		targetZ := target.playerZ
		targetCreative := target.gameType == 1
		target.stateMu.Unlock()
		if targetDead {
			continue
		}

		dx := targetX - projectile.X
		dy := (targetY + 0.9) - projectile.Y
		dz := targetZ - projectile.Z
		if dx*dx+dy*dy+dz*dz > 1.5*1.5 {
			continue
		}

		canPickup := false
		switch projectile.CanBePickedUp {
		case arrowPickupSurvival:
			left := target.addInventoryItem(itemIDArrow, 1, 0)
			canPickup = left == 0
		case arrowPickupCreativeOnly:
			canPickup = targetCreative
		}
		if !canPickup {
			continue
		}

		s.destroyProjectile(projectile)
		return true
	}
	return false
}

func (s *StatusServer) findProjectileHitTarget(projectile *trackedProjectile, x0, y0, z0, x1, y1, z1 float64) (*loginSession, float64) {
	targets := s.activeSessionsExcept(nil)

	var (
		closestTarget *loginSession
		closestDistSq = math.MaxFloat64
	)
	for _, target := range targets {
		if target == nil {
			continue
		}

		target.stateMu.Lock()
		targetEntityID := target.entityID
		targetDead := target.playerDead || target.playerHealth <= 0
		targetX := target.playerX
		targetY := target.playerY
		targetZ := target.playerZ
		target.stateMu.Unlock()

		if targetDead || targetEntityID == 0 {
			continue
		}
		if targetEntityID == projectile.ShooterEntityID && projectile.AgeTicks < 5 {
			continue
		}

		// Approximate with segment-to-player-center distance.
		centerX := targetX
		centerY := targetY + 0.9
		centerZ := targetZ
		distSq := segmentPointDistanceSquared(x0, y0, z0, x1, y1, z1, centerX, centerY, centerZ)
		if distSq > arrowHitRadius*arrowHitRadius {
			continue
		}
		if distSq < closestDistSq {
			closestDistSq = distSq
			closestTarget = target
		}
	}
	return closestTarget, closestDistSq
}

func (s *StatusServer) findProjectileHitMob(projectile *trackedProjectile, x0, y0, z0, x1, y1, z1 float64) (*trackedMob, float64) {
	s.mobMu.Lock()
	mobs := make([]*trackedMob, 0, len(s.mobs))
	for _, mob := range s.mobs {
		if mob != nil {
			mobs = append(mobs, mob)
		}
	}
	s.mobMu.Unlock()

	var (
		closestMob    *trackedMob
		closestDistSq = math.MaxFloat64
	)
	for _, mob := range mobs {
		if mob == nil || mob.Health <= 0 {
			continue
		}
		if mob.EntityID == projectile.ShooterEntityID && projectile.AgeTicks < 5 {
			continue
		}

		height, width := mobCollisionSize(mob)
		centerX := mob.X
		centerY := mob.Y + height*0.5
		centerZ := mob.Z
		radius := arrowHitRadius + width*0.5
		distSq := segmentPointDistanceSquared(x0, y0, z0, x1, y1, z1, centerX, centerY, centerZ)
		if distSq > radius*radius {
			continue
		}
		if distSq < closestDistSq {
			closestDistSq = distSq
			closestMob = mob
		}
	}
	return closestMob, closestDistSq
}

func mobCollisionSize(mob *trackedMob) (height, width float64) {
	if mob == nil {
		return 1.8, 0.6
	}
	switch mob.EntityType {
	case entityTypeZombie:
		if mob.zombieChild {
			return 0.9, 0.3
		}
		return 1.8, 0.6
	case entityTypeSpider:
		return 0.9, 1.4
	case entityTypeSlime:
		size := float64(mob.slimeSize)
		if size < 1.0 {
			size = 1.0
		}
		side := 0.6 * size
		return side, side
	case entityTypeEnderman:
		return 2.9, 0.6
	case entityTypeBat:
		return 0.9, 0.5
	case entityTypePig:
		return 0.9, 0.9
	case entityTypeSheep, entityTypeCow:
		return 1.3, 0.9
	case entityTypeChicken:
		return 0.7, 0.3
	case entityTypeSquid:
		return 0.95, 0.95
	case entityTypeHorse:
		return 1.6, 1.4
	default:
		return 1.8, 0.6
	}
}

func segmentPointDistanceSquared(x0, y0, z0, x1, y1, z1, px, py, pz float64) float64 {
	vx := x1 - x0
	vy := y1 - y0
	vz := z1 - z0
	wx := px - x0
	wy := py - y0
	wz := pz - z0

	segLenSq := vx*vx + vy*vy + vz*vz
	if segLenSq <= 1.0e-12 {
		dx := px - x0
		dy := py - y0
		dz := pz - z0
		return dx*dx + dy*dy + dz*dz
	}

	t := (wx*vx + wy*vy + wz*vz) / segLenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	cx := x0 + vx*t
	cy := y0 + vy*t
	cz := z0 + vz*t
	dx := px - cx
	dy := py - cy
	dz := pz - cz
	return dx*dx + dy*dy + dz*dz
}
