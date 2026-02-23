package server

import (
	"math"

	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/util"
)

const (
	entityTypeDroppedItem int8 = 2

	droppedItemPickupDelayTicks = 40
	entityDropPickupDelayTicks  = 10
	droppedItemLifetimeTicks    = 6000
	droppedItemCreativeAgeTicks = 4800

	droppedItemGravityPerTick = 0.03999999910593033
	droppedItemAirDrag        = 0.9800000190734863
	droppedItemGroundDrag     = 0.58800006
	droppedItemGroundBounce   = -0.5
	droppedItemPickupRadiusSq = 1.5 * 1.5
)

// trackedDroppedItem translates server-side subset behavior for EntityItem.
//
// Translation references:
// - net.minecraft.src.EntityItem
// - net.minecraft.src.EntityTrackerEntry#getPacketForThisEntity (Packet23 type=2)
type trackedDroppedItem struct {
	EntityID int32

	ItemID     int16
	ItemDamage int16
	StackSize  int8

	X float64
	Y float64
	Z float64

	MotionX float64
	MotionY float64
	MotionZ float64

	Yaw float32

	AgeTicks            int
	DelayBeforeCanPick  int
	SeenBy              map[*loginSession]struct{}
	lastSentX           int32
	lastSentY           int32
	lastSentZ           int32
	lastSentInitialized bool
}

func newTrackedDroppedItem(entityID int32, stack *protocol.ItemStack, playerX, playerY, playerZ float64, playerYaw, playerPitch float32, randomChoice bool) *trackedDroppedItem {
	if stack == nil || stack.ItemID <= 0 || stack.StackSize <= 0 {
		return nil
	}
	stack = cloneItemStack(stack)
	if stack.StackSize <= 0 {
		return nil
	}

	seed := int64(entityID)*341873128712 + int64(stack.ItemID)*132897987541 + int64(stack.ItemDamage)*177
	rng := util.NewJavaRandom(seed)

	item := &trackedDroppedItem{
		EntityID:           entityID,
		ItemID:             stack.ItemID,
		ItemDamage:         stack.ItemDamage,
		StackSize:          stack.StackSize,
		X:                  playerX,
		Y:                  playerY - 0.30000001192092896 + defaultPlayerEyeY,
		Z:                  playerZ,
		Yaw:                float32(rng.NextFloat()) * 360.0,
		DelayBeforeCanPick: droppedItemPickupDelayTicks,
		SeenBy:             make(map[*loginSession]struct{}),
	}

	// Translated from:
	// - net.minecraft.src.EntityPlayer#dropPlayerItemWithRandomChoice(ItemStack,boolean)
	if randomChoice {
		speed := float64(rng.NextFloat()) * 0.5
		angle := float64(rng.NextFloat()) * math.Pi * 2.0
		item.MotionX = -math.Sin(angle) * speed
		item.MotionZ = math.Cos(angle) * speed
		item.MotionY = 0.20000000298023224
	} else {
		speed := 0.3
		yawRad := float64(playerYaw) * math.Pi / 180.0
		pitchRad := float64(playerPitch) * math.Pi / 180.0
		item.MotionX = -math.Sin(yawRad) * math.Cos(pitchRad) * speed
		item.MotionZ = math.Cos(yawRad) * math.Cos(pitchRad) * speed
		item.MotionY = -math.Sin(pitchRad)*speed + 0.1

		speed = 0.02
		randAngle := float64(rng.NextFloat()) * math.Pi * 2.0
		speed *= float64(rng.NextFloat())
		item.MotionX += math.Cos(randAngle) * speed
		item.MotionY += float64(rng.NextFloat()-rng.NextFloat()) * 0.1
		item.MotionZ += math.Sin(randAngle) * speed
	}

	item.lastSentX = toPacketPosition(item.X)
	item.lastSentY = toPacketPosition(item.Y)
	item.lastSentZ = toPacketPosition(item.Z)
	item.lastSentInitialized = true
	return item
}

func (i *trackedDroppedItem) spawnPacket() *protocol.Packet23VehicleSpawn {
	vel := protocol.NewPacket28EntityVelocity(i.EntityID, i.MotionX, i.MotionY, i.MotionZ)
	return &protocol.Packet23VehicleSpawn{
		EntityID:        i.EntityID,
		Type:            entityTypeDroppedItem,
		XPosition:       toPacketPosition(i.X),
		YPosition:       toPacketPosition(i.Y),
		ZPosition:       toPacketPosition(i.Z),
		Pitch:           0,
		Yaw:             toPacketAngle(i.Yaw),
		ThrowerEntityID: 1,
		SpeedX:          vel.MotionX,
		SpeedY:          vel.MotionY,
		SpeedZ:          vel.MotionZ,
	}
}

func (i *trackedDroppedItem) metadataPacket() *protocol.Packet40EntityMetadata {
	return &protocol.Packet40EntityMetadata{
		EntityID: i.EntityID,
		Metadata: []protocol.WatchableObject{
			{
				ObjectType:  5,
				DataValueID: 10,
				Value: &protocol.ItemStack{
					ItemID:     i.ItemID,
					StackSize:  i.StackSize,
					ItemDamage: i.ItemDamage,
				},
			},
		},
	}
}

func (i *trackedDroppedItem) teleportPacket() *protocol.Packet34EntityTeleport {
	return &protocol.Packet34EntityTeleport{
		EntityID:  i.EntityID,
		XPosition: toPacketPosition(i.X),
		YPosition: toPacketPosition(i.Y),
		ZPosition: toPacketPosition(i.Z),
		Yaw:       toPacketAngle(i.Yaw),
		Pitch:     0,
	}
}

func (i *trackedDroppedItem) chunkCoords() (int32, int32) {
	return chunkCoordFromPos(i.X), chunkCoordFromPos(i.Z)
}

func (s *StatusServer) spawnDroppedItemFromPlayer(player *loginSession, stack *protocol.ItemStack, randomChoice bool, creativeDespawn bool) *trackedDroppedItem {
	if player == nil || stack == nil || stack.ItemID <= 0 || stack.StackSize <= 0 {
		return nil
	}

	var (
		playerX     float64
		playerY     float64
		playerZ     float64
		playerYaw   float32
		playerPitch float32
	)
	player.stateMu.Lock()
	playerX = player.playerX
	playerY = player.playerY
	playerZ = player.playerZ
	playerYaw = player.playerYaw
	playerPitch = player.playerPitch
	player.stateMu.Unlock()

	entityID := s.nextEntityID.Add(1)
	item := newTrackedDroppedItem(entityID, stack, playerX, playerY, playerZ, playerYaw, playerPitch, randomChoice)
	if item == nil {
		return nil
	}
	if creativeDespawn {
		item.AgeTicks = droppedItemCreativeAgeTicks
	}

	s.droppedItemMu.Lock()
	s.droppedItems[entityID] = item
	s.droppedItemMu.Unlock()

	s.updateDroppedItemVisibility(item, nil, true)
	return item
}

func (s *StatusServer) spawnDroppedItemAt(stack *protocol.ItemStack, x, y, z, motionX, motionY, motionZ float64, pickupDelay int) *trackedDroppedItem {
	if stack == nil || stack.ItemID <= 0 || stack.StackSize <= 0 {
		return nil
	}
	if pickupDelay < 0 {
		pickupDelay = 0
	}

	entityID := s.nextEntityID.Add(1)
	item := &trackedDroppedItem{
		EntityID:           entityID,
		ItemID:             stack.ItemID,
		ItemDamage:         stack.ItemDamage,
		StackSize:          stack.StackSize,
		X:                  x,
		Y:                  y,
		Z:                  z,
		MotionX:            motionX,
		MotionY:            motionY,
		MotionZ:            motionZ,
		DelayBeforeCanPick: pickupDelay,
		SeenBy:             make(map[*loginSession]struct{}),
	}
	item.lastSentX = toPacketPosition(item.X)
	item.lastSentY = toPacketPosition(item.Y)
	item.lastSentZ = toPacketPosition(item.Z)
	item.lastSentInitialized = true

	s.droppedItemMu.Lock()
	s.droppedItems[entityID] = item
	s.droppedItemMu.Unlock()

	s.updateDroppedItemVisibility(item, nil, true)
	return item
}

func (s *StatusServer) TickDroppedItems() {
	s.droppedItemMu.Lock()
	if len(s.droppedItems) == 0 {
		s.droppedItemMu.Unlock()
		return
	}
	items := make([]*trackedDroppedItem, 0, len(s.droppedItems))
	for _, item := range s.droppedItems {
		if item != nil {
			items = append(items, item)
		}
	}
	s.droppedItemMu.Unlock()

	for _, item := range items {
		s.tickDroppedItem(item)
	}
}

func (s *StatusServer) tickDroppedItem(item *trackedDroppedItem) {
	if item == nil {
		return
	}

	if item.DelayBeforeCanPick > 0 {
		item.DelayBeforeCanPick--
	}

	// Translated baseline from EntityItem#onUpdate motion path.
	item.MotionY -= droppedItemGravityPerTick
	item.X += item.MotionX
	item.Y += item.MotionY
	item.Z += item.MotionZ

	onGround := false
	belowX := int(math.Floor(item.X))
	belowY := int(math.Floor(item.Y)) - 1
	belowZ := int(math.Floor(item.Z))
	if belowY >= 0 && belowY < 256 {
		belowID, _ := s.world.getBlock(belowX, belowY, belowZ)
		if belowID != 0 {
			onGround = true
			floorY := float64(belowY + 1)
			if item.Y < floorY {
				item.Y = floorY
			}
		}
	}

	drag := droppedItemAirDrag
	if onGround {
		drag = droppedItemGroundDrag
	}
	item.MotionX *= drag
	item.MotionY *= droppedItemAirDrag
	item.MotionZ *= drag
	if onGround {
		item.MotionY *= droppedItemGroundBounce
	}

	item.AgeTicks++
	if item.AgeTicks >= droppedItemLifetimeTicks {
		s.destroyDroppedItem(item)
		return
	}

	picked, collectorEntityID, changedCount := s.tryPickupDroppedItem(item)
	if picked {
		if collectorEntityID != 0 {
			s.broadcastDroppedItemCollect(item, collectorEntityID)
		}
		s.destroyDroppedItem(item)
		return
	}
	if changedCount {
		s.broadcastDroppedItemMetadata(item)
	}

	currX := toPacketPosition(item.X)
	currY := toPacketPosition(item.Y)
	currZ := toPacketPosition(item.Z)
	var movePacket protocol.Packet
	if !item.lastSentInitialized || currX != item.lastSentX || currY != item.lastSentY || currZ != item.lastSentZ {
		movePacket = item.teleportPacket()
		item.lastSentX = currX
		item.lastSentY = currY
		item.lastSentZ = currZ
		item.lastSentInitialized = true
	}
	s.updateDroppedItemVisibility(item, movePacket, false)
}

func (s *StatusServer) tryPickupDroppedItem(item *trackedDroppedItem) (picked bool, collectorEntityID int32, changedCount bool) {
	if item == nil || item.DelayBeforeCanPick > 0 || item.StackSize <= 0 {
		return false, 0, false
	}

	targets := s.activeSessionsExcept(nil)
	for _, target := range targets {
		if target == nil {
			continue
		}

		target.stateMu.Lock()
		dead := target.playerDead || target.playerHealth <= 0
		targetX := target.playerX
		targetY := target.playerY
		targetZ := target.playerZ
		targetEntityID := target.entityID
		target.stateMu.Unlock()
		if dead || targetEntityID == 0 {
			continue
		}

		dx := targetX - item.X
		dy := (targetY + 0.9) - item.Y
		dz := targetZ - item.Z
		if dx*dx+dy*dy+dz*dz > droppedItemPickupRadiusSq {
			continue
		}

		before := int(item.StackSize)
		remaining := target.addInventoryItem(item.ItemID, before, item.ItemDamage)
		if remaining >= before {
			continue
		}

		if remaining > 0 {
			item.StackSize = int8(remaining)
			return false, 0, true
		}
		return true, targetEntityID, false
	}
	return false, 0, false
}

func (s *StatusServer) updateDroppedItemVisibility(item *trackedDroppedItem, movementPacket protocol.Packet, includeMetadataOnSpawn bool) {
	if item == nil {
		return
	}

	s.droppedItemMu.Lock()
	if item.SeenBy == nil {
		item.SeenBy = make(map[*loginSession]struct{})
	}
	s.droppedItemMu.Unlock()

	chunkX, chunkZ := item.chunkCoords()
	targets := s.activeSessionsExcept(nil)
	spawnPacket := item.spawnPacket()
	metaPacket := item.metadataPacket()
	destroyPacket := &protocol.Packet29DestroyEntity{EntityIDs: []int32{item.EntityID}}

	for _, target := range targets {
		if target == nil {
			continue
		}

		shouldSee := target.isWatchingChunk(chunkX, chunkZ)

		s.droppedItemMu.Lock()
		_, wasSeen := item.SeenBy[target]
		s.droppedItemMu.Unlock()

		switch {
		case shouldSee && !wasSeen:
			if !target.sendPacket(spawnPacket) {
				continue
			}
			if includeMetadataOnSpawn {
				_ = target.sendPacket(metaPacket)
			}
			s.droppedItemMu.Lock()
			item.SeenBy[target] = struct{}{}
			s.droppedItemMu.Unlock()
		case !shouldSee && wasSeen:
			if target.sendPacket(destroyPacket) {
				s.droppedItemMu.Lock()
				delete(item.SeenBy, target)
				s.droppedItemMu.Unlock()
			}
		case shouldSee && wasSeen:
			if movementPacket != nil {
				_ = target.sendPacket(movementPacket)
			}
		}
	}
}

func (s *StatusServer) broadcastDroppedItemMetadata(item *trackedDroppedItem) {
	if item == nil {
		return
	}
	meta := item.metadataPacket()

	s.droppedItemMu.Lock()
	targets := make([]*loginSession, 0, len(item.SeenBy))
	for target := range item.SeenBy {
		targets = append(targets, target)
	}
	s.droppedItemMu.Unlock()

	for _, target := range targets {
		_ = target.sendPacket(meta)
	}
}

func (s *StatusServer) broadcastDroppedItemCollect(item *trackedDroppedItem, collectorEntityID int32) {
	if item == nil || collectorEntityID == 0 {
		return
	}
	collect := &protocol.Packet22Collect{
		CollectedEntityID: item.EntityID,
		CollectorEntityID: collectorEntityID,
	}

	s.droppedItemMu.Lock()
	targets := make([]*loginSession, 0, len(item.SeenBy))
	for target := range item.SeenBy {
		targets = append(targets, target)
	}
	s.droppedItemMu.Unlock()

	collectorSession := s.activeSessionByEntityID(collectorEntityID)
	collectorSeen := false
	for _, target := range targets {
		if target == collectorSession {
			collectorSeen = true
		}
		_ = target.sendPacket(collect)
	}
	if collectorSession != nil && !collectorSeen {
		_ = collectorSession.sendPacket(collect)
	}
}

func (s *StatusServer) destroyDroppedItem(item *trackedDroppedItem) {
	if item == nil {
		return
	}

	s.droppedItemMu.Lock()
	seenTargets := make([]*loginSession, 0, len(item.SeenBy))
	for session := range item.SeenBy {
		seenTargets = append(seenTargets, session)
	}
	item.SeenBy = make(map[*loginSession]struct{})
	delete(s.droppedItems, item.EntityID)
	s.droppedItemMu.Unlock()

	destroy := &protocol.Packet29DestroyEntity{EntityIDs: []int32{item.EntityID}}
	for _, target := range seenTargets {
		_ = target.sendPacket(destroy)
	}
}
