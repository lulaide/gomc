package server

import (
	"bytes"
	"io"
	"math"
	"testing"
	"time"

	"github.com/lulaide/gomc/pkg/network/protocol"
)

func metadataByteByID(metadata []protocol.WatchableObject, id int8) (int8, bool) {
	for _, m := range metadata {
		if m.ObjectType != 0 || m.DataValueID != id {
			continue
		}
		v, ok := m.Value.(int8)
		if !ok {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

func countDroppedItemStacks(srv *StatusServer, itemID int16) (count int, damages []int16) {
	if srv == nil {
		return 0, nil
	}
	srv.droppedItemMu.Lock()
	defer srv.droppedItemMu.Unlock()
	for _, item := range srv.droppedItems {
		if item == nil || item.ItemID != itemID || item.StackSize <= 0 {
			continue
		}
		count += int(item.StackSize)
		for i := 0; i < int(item.StackSize); i++ {
			damages = append(damages, item.ItemDamage)
		}
	}
	return count, damages
}

func TestSpawnRandomCreaturePeacefulSkipsMonsters(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.setDifficulty(0)
	if got := srv.spawnRandomCreature(creatureTypeMonster, 0, 64, 0); got != nil {
		t.Fatalf("peaceful should skip monster spawns, got=%#v", got)
	}
}

func TestTickSingleMobPeacefulDespawnsMonsterWithoutDrops(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.setDifficulty(0)
	srv.mobRand.SetSeed(81)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	srv.tickSingleMob(mob)

	srv.mobMu.Lock()
	_, alive := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if alive {
		t.Fatalf("expected peaceful monster despawn, entity still alive: %d", mob.EntityID)
	}
	if got, _ := countDroppedItemStacks(srv, itemIDRottenFlesh); got != 0 {
		t.Fatalf("peaceful despawn should not drop items, rotten flesh=%d", got)
	}
}

func TestTickSingleMobMonsterChasesPlayer(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	target := newInteractionTestSession(srv, io.Discard)
	target.playerRegistered = true
	target.playerDead = false
	target.entityID = 101
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 0.5

	srv.activeMu.Lock()
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{target}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 8.5, 180)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	startDistSq := (target.playerX-mob.X)*(target.playerX-mob.X) + (target.playerZ-mob.Z)*(target.playerZ-mob.Z)

	for i := 0; i < 40; i++ {
		srv.tickSingleMob(mob)
	}

	endDistSq := (target.playerX-mob.X)*(target.playerX-mob.X) + (target.playerZ-mob.Z)*(target.playerZ-mob.Z)
	if endDistSq >= startDistSq {
		t.Fatalf("monster did not approach target: start=%.3f end=%.3f mob=(%.2f,%.2f,%.2f)", startDistSq, endDistSq, mob.X, mob.Y, mob.Z)
	}
}

func TestTickSingleMobPassiveWanders(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(7)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSheep}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	startX, startZ := mob.X, mob.Z

	for i := 0; i < 400; i++ {
		srv.tickSingleMob(mob)
	}

	distSq := (mob.X-startX)*(mob.X-startX) + (mob.Z-startZ)*(mob.Z-startZ)
	if distSq < 0.36 {
		t.Fatalf("passive mob moved too little: distSq=%.4f pos=(%.2f,%.2f,%.2f)", distSq, mob.X, mob.Y, mob.Z)
	}
}

func TestTryMoveLandMobCanStepUpOneBlock(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	if !srv.world.setBlock(1, 5, 0, 1, 0) {
		t.Fatal("failed to place obstacle")
	}

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	if moved := srv.tryMoveLandMob(mob, 0.95, 0.0); !moved {
		t.Fatalf("tryMoveLandMob failed: pos=(%.2f,%.2f,%.2f)", mob.X, mob.Y, mob.Z)
	}
	if mob.Y < 5.9 {
		t.Fatalf("mob did not step up: pos=(%.2f,%.2f,%.2f)", mob.X, mob.Y, mob.Z)
	}
	if math.IsNaN(mob.X) || math.IsNaN(mob.Y) || math.IsNaN(mob.Z) {
		t.Fatalf("mob position became NaN: pos=(%.2f,%.2f,%.2f)", mob.X, mob.Y, mob.Z)
	}
}

func TestTickSingleMobCreeperFusesAndExplodes(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.playerRegistered = true
	watcher.playerDead = false
	watcher.entityID = 103
	watcher.playerX = 0.5
	watcher.playerY = 5.0
	watcher.playerZ = 0.5
	watcher.playerHealth = maxPlayerHealth

	srv.activeMu.Lock()
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{watcher}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeCreeper}, 0.5, 5.0, 2.5, 180)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	startHealth := watcher.playerHealth

	for i := 0; i < creeperFuseTimeDefault+4; i++ {
		srv.tickSingleMob(mob)
	}

	srv.mobMu.Lock()
	_, alive := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if alive {
		t.Fatalf("creeper should be removed after explosion: entityID=%d", mob.EntityID)
	}

	if watcher.playerHealth >= startHealth {
		t.Fatalf("creeper explosion should damage nearby player: start=%.2f now=%.2f", startHealth, watcher.playerHealth)
	}

	foundDestroy := false
	for {
		packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		if p, ok := packet.(*protocol.Packet29DestroyEntity); ok {
			for _, id := range p.EntityIDs {
				if id == mob.EntityID {
					foundDestroy = true
					break
				}
			}
		}
	}
	if !foundDestroy {
		t.Fatal("expected watcher to receive destroy packet for exploded creeper")
	}
}

func TestCreeperExplosionDestroysNearbyBlocks(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	target := newInteractionTestSession(srv, io.Discard)
	target.playerRegistered = true
	target.playerDead = false
	target.entityID = 104
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 0.5
	target.playerHealth = maxPlayerHealth

	srv.activeMu.Lock()
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{target}
	srv.activeMu.Unlock()

	if !srv.world.setBlock(1, 5, 1, 1, 0) {
		t.Fatal("failed to place test block")
	}
	blockBefore, _ := srv.world.getBlock(1, 5, 1)
	if blockBefore == 0 {
		t.Fatal("precondition failed: expected non-air block at (1,5,1)")
	}

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeCreeper}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	for i := 0; i < creeperFuseTimeDefault+3; i++ {
		srv.tickSingleMob(mob)
	}

	blockAfter, _ := srv.world.getBlock(1, 5, 1)
	if blockAfter != 0 {
		t.Fatalf("expected explosion to destroy block: got=%d", blockAfter)
	}
}

func TestTickSingleMobSkeletonShootsProjectiles(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	target := newInteractionTestSession(srv, io.Discard)
	target.playerRegistered = true
	target.playerDead = false
	target.entityID = 105
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 10.5
	target.playerHealth = maxPlayerHealth

	srv.activeMu.Lock()
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{target}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSkeleton}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	for i := 0; i < 120; i++ {
		srv.tickSingleMob(mob)
	}

	srv.projectileMu.Lock()
	projectileCount := len(srv.projectiles)
	srv.projectileMu.Unlock()
	if projectileCount == 0 {
		t.Fatal("expected skeleton to fire at least one arrow projectile")
	}
}

func TestTickSingleMobSpiderCanLeap(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.AdvanceWorldTime(14000) // night

	target := newInteractionTestSession(srv, io.Discard)
	target.playerRegistered = true
	target.playerDead = false
	target.entityID = 106
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 4.5

	srv.activeMu.Lock()
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{target}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSpider}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	leaped := false
	for seed := int64(0); seed < 512; seed++ {
		srv.mobRand.SetSeed(seed)
		mob.X, mob.Y, mob.Z = 0.5, 5.0, 0.5
		mob.MotionX, mob.MotionY, mob.MotionZ = 0, 0, 0
		mob.spiderLeapCD = 0

		px, pz := mob.X, mob.Z
		srv.tickSingleMob(mob)
		step := math.Sqrt((mob.X-px)*(mob.X-px) + (mob.Z-pz)*(mob.Z-pz))
		if step > 0.20 {
			leaped = true
			break
		}
	}
	if !leaped {
		t.Fatal("expected at least one seeded tick to trigger spider leap")
	}
}

func TestTickSingleMobSlimeJumpCadence(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	target := newInteractionTestSession(srv, io.Discard)
	target.playerRegistered = true
	target.playerDead = false
	target.entityID = 107
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 6.5

	srv.activeMu.Lock()
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{target}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSlime}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	mob.slimeSize = 2
	mob.slimeJumpDelay = 3
	mob.wanderPause = 0
	mob.wanderTicks = 100
	mob.wanderYaw = 0

	stepTick := func() float64 {
		px, pz := mob.X, mob.Z
		srv.tickSingleMob(mob)
		return math.Sqrt((mob.X-px)*(mob.X-px) + (mob.Z-pz)*(mob.Z-pz))
	}

	step1 := stepTick()
	step2 := stepTick()
	step3 := stepTick()

	if step1 > 0.01 || step2 > 0.01 {
		t.Fatalf("slime moved during wait ticks: step1=%.3f step2=%.3f", step1, step2)
	}
	if step3 <= 0.04 {
		t.Fatalf("slime did not move on jump tick: step3=%.3f", step3)
	}
}

func TestTrackedMobSpawnPacketIncludesMobMetadata(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	creeper := srv.spawnMob(&spawnListEntry{entityType: entityTypeCreeper}, 0.5, 5.0, 0.5, 0)
	if creeper == nil {
		t.Fatal("spawnMob creeper returned nil")
	}
	creeper.creeperState = 1
	creeper.creeperPowered = true
	creeperMeta := creeper.spawnPacket().Metadata
	if v, ok := metadataByteByID(creeperMeta, 16); !ok || v != 1 {
		t.Fatalf("creeper state metadata mismatch: ok=%t v=%d", ok, v)
	}
	if v, ok := metadataByteByID(creeperMeta, 17); !ok || v != 1 {
		t.Fatalf("creeper powered metadata mismatch: ok=%t v=%d", ok, v)
	}

	slime := srv.spawnMob(&spawnListEntry{entityType: entityTypeSlime}, 1.5, 5.0, 0.5, 0)
	if slime == nil {
		t.Fatal("spawnMob slime returned nil")
	}
	slime.slimeSize = 4
	slimeMeta := slime.spawnPacket().Metadata
	if v, ok := metadataByteByID(slimeMeta, 16); !ok || v != 4 {
		t.Fatalf("slime size metadata mismatch: ok=%t v=%d", ok, v)
	}

	sheep := srv.spawnMob(&spawnListEntry{entityType: entityTypeSheep}, 2.5, 5.0, 0.5, 0)
	if sheep == nil {
		t.Fatal("spawnMob sheep returned nil")
	}
	sheep.sheepColor = 10
	sheep.sheepSheared = true
	sheepMeta := sheep.spawnPacket().Metadata
	if v, ok := metadataByteByID(sheepMeta, 16); !ok || v != 0x1A {
		t.Fatalf("sheep metadata mismatch: ok=%t v=%d", ok, v)
	}

	spider := srv.spawnMob(&spawnListEntry{entityType: entityTypeSpider}, 3.5, 5.0, 0.5, 0)
	if spider == nil {
		t.Fatal("spawnMob spider returned nil")
	}
	spider.spiderClimb = true
	spiderMeta := spider.spawnPacket().Metadata
	if v, ok := metadataByteByID(spiderMeta, 16); !ok || (v&1) == 0 {
		t.Fatalf("spider climb metadata mismatch: ok=%t v=%d", ok, v)
	}

	skeleton := srv.spawnMob(&spawnListEntry{entityType: entityTypeSkeleton}, 4.5, 5.0, 0.5, 0)
	if skeleton == nil {
		t.Fatal("spawnMob skeleton returned nil")
	}
	skeleton.skeletonType = 1
	skeletonMeta := skeleton.spawnPacket().Metadata
	if v, ok := metadataByteByID(skeletonMeta, 13); !ok || v != 1 {
		t.Fatalf("skeleton type metadata mismatch: ok=%t v=%d", ok, v)
	}

	zombie := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 5.5, 5.0, 0.5, 0)
	if zombie == nil {
		t.Fatal("spawnMob zombie returned nil")
	}
	zombie.zombieChild = true
	zombie.zombieVillager = true
	zombieMeta := zombie.spawnPacket().Metadata
	if v, ok := metadataByteByID(zombieMeta, 12); !ok || v != 1 {
		t.Fatalf("zombie child metadata mismatch: ok=%t v=%d", ok, v)
	}
	if v, ok := metadataByteByID(zombieMeta, 13); !ok || v != 1 {
		t.Fatalf("zombie villager metadata mismatch: ok=%t v=%d", ok, v)
	}
}

func TestTickSingleMobCreeperStateBroadcastsMetadata(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	var watcherBuf bytes.Buffer
	target := newInteractionTestSession(srv, &watcherBuf)
	target.playerRegistered = true
	target.playerDead = false
	target.entityID = 108
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 0.5

	srv.activeMu.Lock()
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{target}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeCreeper}, 0.5, 5.0, 2.5, 180)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	srv.tickSingleMob(mob)

	foundMeta := false
	for {
		packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		meta, ok := packet.(*protocol.Packet40EntityMetadata)
		if !ok || meta.EntityID != mob.EntityID {
			continue
		}
		if state, ok := metadataByteByID(meta.Metadata, 16); ok && state == 1 {
			foundMeta = true
			break
		}
	}
	if !foundMeta {
		t.Fatal("expected creeper state metadata packet (id=16, value=1)")
	}
}

func TestSpawnMobSheepFleeceColorWithinVanillaSet(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(12345)

	allowed := map[int8]struct{}{
		0:  {},
		6:  {},
		7:  {},
		8:  {},
		12: {},
		15: {},
	}

	for i := 0; i < 256; i++ {
		mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSheep}, float64(i)+0.5, 5.0, 0.5, 0)
		if mob == nil {
			t.Fatalf("spawnMob sheep returned nil at i=%d", i)
		}
		if _, ok := allowed[mob.sheepColor]; !ok {
			t.Fatalf("unexpected sheep color at i=%d: %d", i, mob.sheepColor)
		}
	}
}

func TestKillMobSplitsLargeSlime(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(9)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSlime, fixedSlime: 4}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	srv.killMob(mob, false, 0)

	srv.mobMu.Lock()
	defer srv.mobMu.Unlock()
	if _, ok := srv.mobs[mob.EntityID]; ok {
		t.Fatalf("expected parent slime removed: entity=%d", mob.EntityID)
	}
	childCount := 0
	for _, e := range srv.mobs {
		if e == nil || e.EntityType != entityTypeSlime {
			continue
		}
		if e.slimeSize == 2 {
			childCount++
		}
	}
	if childCount < 2 || childCount > 4 {
		t.Fatalf("slime split child count mismatch: got=%d want=2..4", childCount)
	}
}

func TestKillMobSheepDropsSingleWoolWhenNotSheared(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(11)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSheep}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	srv.mobMu.Lock()
	live := srv.mobs[mob.EntityID]
	if live != nil {
		live.sheepSheared = false
		live.sheepColor = 7
	}
	srv.mobMu.Unlock()

	srv.killMob(mob, true, 0)

	woolCount, woolDamages := countDroppedItemStacks(srv, blockIDWool)
	if woolCount != 1 {
		t.Fatalf("wool drop count mismatch: got=%d want=1", woolCount)
	}
	if len(woolDamages) != 1 || woolDamages[0] != 7 {
		t.Fatalf("wool metadata mismatch: %#v", woolDamages)
	}
}

func TestKillMobCowDropsLeatherAndBeefWithinVanillaRanges(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(21)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeCow}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	srv.killMob(mob, true, 0)

	leatherCount, _ := countDroppedItemStacks(srv, itemIDLeather)
	beefCount, _ := countDroppedItemStacks(srv, itemIDBeefRaw)
	if leatherCount < 0 || leatherCount > 2 {
		t.Fatalf("cow leather drop out of range: got=%d want=0..2", leatherCount)
	}
	if beefCount < 1 || beefCount > 3 {
		t.Fatalf("cow beef drop out of range: got=%d want=1..3", beefCount)
	}
}

func TestKillMobSpiderWithoutPlayerKillDoesNotDropSpiderEye(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(33)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSpider}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	srv.killMob(mob, false, 0)

	eyeCount, _ := countDroppedItemStacks(srv, itemIDSpiderEye)
	if eyeCount != 0 {
		t.Fatalf("spider eye should require player kill flag: got=%d", eyeCount)
	}
}

func TestKillMobPigDropsSaddleOnlyWhenSaddled(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(41)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypePig}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	srv.mobMu.Lock()
	live := srv.mobs[mob.EntityID]
	if live != nil {
		live.pigSaddled = true
	}
	srv.mobMu.Unlock()

	srv.killMob(mob, true, 0)

	saddleCount, _ := countDroppedItemStacks(srv, itemIDSaddle)
	if saddleCount != 1 {
		t.Fatalf("pig saddled drop mismatch: got=%d want=1", saddleCount)
	}
}

func TestKillMobZombieRareDropRequiresPlayerKill(t *testing.T) {
	const forcedRareLooting = 205

	srvNoPlayer := NewStatusServer(StatusConfig{})
	srvNoPlayer.mobRand.SetSeed(52)
	noPlayerZombie := srvNoPlayer.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 0.5, 0)
	if noPlayerZombie == nil {
		t.Fatal("spawnMob returned nil for no-player branch")
	}
	srvNoPlayer.killMob(noPlayerZombie, false, forcedRareLooting)
	noPlayerRare := 0
	for _, itemID := range []int16{itemIDIronIngot, itemIDCarrot, itemIDPotato} {
		count, _ := countDroppedItemStacks(srvNoPlayer, itemID)
		noPlayerRare += count
	}
	if noPlayerRare != 0 {
		t.Fatalf("zombie rare drop should require player kill flag: got=%d", noPlayerRare)
	}

	srvPlayer := NewStatusServer(StatusConfig{})
	srvPlayer.mobRand.SetSeed(52)
	playerZombie := srvPlayer.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 0.5, 0)
	if playerZombie == nil {
		t.Fatal("spawnMob returned nil for player branch")
	}
	srvPlayer.killMob(playerZombie, true, forcedRareLooting)
	playerRare := 0
	for _, itemID := range []int16{itemIDIronIngot, itemIDCarrot, itemIDPotato} {
		count, _ := countDroppedItemStacks(srvPlayer, itemID)
		playerRare += count
	}
	if playerRare != 1 {
		t.Fatalf("zombie rare drop count mismatch: got=%d want=1", playerRare)
	}
}

func TestKillMobWitherSkeletonRareDropSkullWhenRareRollPasses(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(61)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSkeleton}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	srv.mobMu.Lock()
	live := srv.mobs[mob.EntityID]
	if live != nil {
		live.skeletonType = 1
	}
	srv.mobMu.Unlock()

	srv.killMob(mob, true, 205)

	skullCount, skullDamages := countDroppedItemStacks(srv, itemIDSkull)
	if skullCount != 1 {
		t.Fatalf("wither skeleton skull drop mismatch: got=%d want=1", skullCount)
	}
	if len(skullDamages) != 1 || skullDamages[0] != 1 {
		t.Fatalf("wither skeleton skull metadata mismatch: %#v", skullDamages)
	}
}

func TestKillMobChildZombieDropsNothing(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(73)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	srv.mobMu.Lock()
	live := srv.mobs[mob.EntityID]
	if live != nil {
		live.zombieChild = true
	}
	srv.mobMu.Unlock()

	srv.killMob(mob, true, 205)

	rottenCount, _ := countDroppedItemStacks(srv, itemIDRottenFlesh)
	if rottenCount != 0 {
		t.Fatalf("child zombie should not drop rotten flesh: got=%d", rottenCount)
	}
	rareCount := 0
	for _, itemID := range []int16{itemIDIronIngot, itemIDCarrot, itemIDPotato} {
		count, _ := countDroppedItemStacks(srv, itemID)
		rareCount += count
	}
	if rareCount != 0 {
		t.Fatalf("child zombie should not drop rare items: got=%d", rareCount)
	}
}

func TestTickSingleMobUndeadBurnsInDaylight(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.SetWorldTime(1000) // daytime
	srv.mobRand.SetSeed(3)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	mob.wanderPause = 1000
	startHealth := mob.Health

	for i := 0; i < 300; i++ {
		srv.tickSingleMob(mob)
	}

	srv.mobMu.Lock()
	updated := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if updated == nil {
		// Burn damage can kill the mob; this is valid.
		return
	}
	if !(updated.Health < startHealth) {
		t.Fatalf("expected daytime burn damage: start=%.2f now=%.2f", startHealth, updated.Health)
	}
}

func TestTickSingleMobUndeadDoesNotBurnAtNight(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.SetWorldTime(14000) // night
	srv.mobRand.SetSeed(3)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSkeleton}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	mob.wanderPause = 1000
	startHealth := mob.Health

	for i := 0; i < 300; i++ {
		srv.tickSingleMob(mob)
	}

	srv.mobMu.Lock()
	updated := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if updated == nil {
		t.Fatal("night-time undead should not die from sun burn")
	}
	if updated.Health != startHealth {
		t.Fatalf("unexpected night-time burn damage: start=%.2f now=%.2f", startHealth, updated.Health)
	}
}

func TestMobAttackDamageMatchesVanillaBaselines(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	enderman := &trackedMob{EntityType: entityTypeEnderman}
	if got := srv.mobAttackDamage(enderman); got != 7.0 {
		t.Fatalf("enderman attack damage mismatch: got=%.1f want=7.0", got)
	}

	zombie := &trackedMob{EntityType: entityTypeZombie}
	if got := srv.mobAttackDamage(zombie); got != 3.0 {
		t.Fatalf("zombie attack damage mismatch: got=%.1f want=3.0", got)
	}

	spider := &trackedMob{EntityType: entityTypeSpider}
	if got := srv.mobAttackDamage(spider); got != 2.0 {
		t.Fatalf("spider attack damage mismatch: got=%.1f want=2.0", got)
	}
}

func TestTickSingleMobUndeadHelmetPreventsBurnUntilBroken(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.SetWorldTime(1000) // daytime
	srv.mobRand.SetSeed(21)

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	// Gold helmet near break (max 77).
	mob.helmetItemID = 314
	mob.helmetDamage = 76

	startHealth := mob.Health
	damagedBeforeBreak := false
	helmetBroken := false

	for i := 0; i < 2000; i++ {
		srv.mobMu.Lock()
		live := srv.mobs[mob.EntityID]
		if live == nil {
			srv.mobMu.Unlock()
			helmetBroken = true
			break
		}
		if live.helmetItemID == 0 {
			helmetBroken = true
		} else if live.Health < startHealth {
			damagedBeforeBreak = true
		}
		srv.mobMu.Unlock()
		srv.tickSingleMob(mob)
	}

	if damagedBeforeBreak {
		t.Fatal("undead took sun damage before helmet broke")
	}
	if !helmetBroken {
		t.Fatal("expected helmet to break under sunlight wear")
	}

	srv.mobMu.Lock()
	live := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if live != nil && !(live.Health < startHealth) {
		t.Fatalf("expected sun damage after helmet break: start=%.2f now=%.2f", startHealth, live.Health)
	}
}

func TestEndermanRequiresStareToAcquireTarget(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	target := newInteractionTestSession(srv, io.Discard)
	target.playerRegistered = true
	target.playerDead = false
	target.entityID = 201
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 0.5
	target.playerPitch = 0

	srv.activeMu.Lock()
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{target}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeEnderman}, 0.5, 5.0, 6.5, 180)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	// Looking away should not aggro.
	target.playerYaw = 180
	for i := 0; i < 6; i++ {
		got := srv.nearestTargetForMob(mob, srv.mobFollowRange(mob))
		if got != nil {
			t.Fatalf("enderman should not aggro when not stared at (tick=%d)", i)
		}
	}

	// Staring at enderman requires multiple ticks before acquire.
	target.playerYaw = 0
	for i := 0; i < 4; i++ {
		got := srv.nearestTargetForMob(mob, srv.mobFollowRange(mob))
		if got != nil {
			t.Fatalf("enderman acquired target too early while staring (tick=%d)", i)
		}
	}
	got := srv.nearestTargetForMob(mob, srv.mobFollowRange(mob))
	if got != target {
		t.Fatal("expected enderman to acquire stared player after stare delay")
	}
}

func TestSpawnMobWithGroupDataZombieKeepsGroupFlags(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.now = func() time.Time {
		return time.Date(2013, time.January, 1, 12, 0, 0, 0, time.UTC)
	}
	srv.mobRand.SetSeed(1234)

	entry := &spawnListEntry{entityType: entityTypeZombie}
	var groupData any
	mob1, groupData := srv.spawnMobWithGroupData(entry, 0.5, 5.0, 0.5, 0, groupData)
	if mob1 == nil {
		t.Fatal("spawnMobWithGroupData first zombie returned nil")
	}
	mob2, _ := srv.spawnMobWithGroupData(entry, 1.5, 5.0, 0.5, 0, groupData)
	if mob2 == nil {
		t.Fatal("spawnMobWithGroupData second zombie returned nil")
	}

	if mob1.zombieChild != mob2.zombieChild || mob1.zombieVillager != mob2.zombieVillager {
		t.Fatalf("zombie group data mismatch: first(child=%t,villager=%t) second(child=%t,villager=%t)",
			mob1.zombieChild, mob1.zombieVillager, mob2.zombieChild, mob2.zombieVillager)
	}
}

func TestSpawnMobSkeletonGetsBowByDefault(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.now = func() time.Time {
		return time.Date(2013, time.January, 1, 12, 0, 0, 0, time.UTC)
	}

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeSkeleton}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	if mob.heldItemID != mobItemIDBow {
		t.Fatalf("skeleton held item mismatch: got=%d want=%d", mob.heldItemID, mobItemIDBow)
	}
}

func TestTryEquipHalloweenHeadOnOctober31(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.now = func() time.Time {
		return time.Date(2013, time.October, 31, 12, 0, 0, 0, time.UTC)
	}

	mob := &trackedMob{EntityType: entityTypeZombie}
	found := false
	for seed := int64(0); seed < 5000; seed++ {
		srv.mobRand.SetSeed(seed)
		mob.helmetItemID = 0
		mob.helmetDamage = 0
		srv.tryEquipHalloweenHead(mob)
		if mob.helmetItemID == mobItemIDPumpkin || mob.helmetItemID == mobItemIDJackOLantern {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected at least one seeded attempt to equip Halloween head")
	}
}

func TestTickSingleMobChildZombieDoesNotBurnInDaylight(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.SetWorldTime(1000) // daytime
	srv.mobRand.SetSeed(3)
	srv.now = func() time.Time {
		return time.Date(2013, time.January, 1, 12, 0, 0, 0, time.UTC)
	}

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 0.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	mob.zombieChild = true
	mob.wanderPause = 1000
	startHealth := mob.Health

	for i := 0; i < 300; i++ {
		srv.tickSingleMob(mob)
	}

	srv.mobMu.Lock()
	live := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if live == nil {
		t.Fatal("child zombie should not die from sun burn")
	}
	if live.Health != startHealth {
		t.Fatalf("child zombie should not take sun burn: start=%.2f now=%.2f", startHealth, live.Health)
	}
}
