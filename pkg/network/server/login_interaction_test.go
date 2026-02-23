package server

import (
	"bytes"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lulaide/gomc/pkg/nbt"
	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

func newInteractionTestSession(srv *StatusServer, writer io.Writer) *loginSession {
	return &loginSession{
		server: srv,
		writer: writer,
		loadedChunks: map[chunk.CoordIntPair]struct{}{
			chunk.NewCoordIntPair(0, 0): {},
		},
		seenBy:  make(map[*loginSession]struct{}),
		playerX: 0.5,
		playerY: 5.0,
		playerZ: 0.5,
	}
}

func TestHandleBlockDigStatus2RemovesBlock(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)

	before, _ := srv.world.getBlock(0, 4, 0)
	if before == 0 {
		t.Fatalf("precondition failed: expected non-air block at spawn layer")
	}

	ok := session.handleBlockDig(&protocol.Packet14BlockDig{
		Status:    2,
		XPosition: 0,
		YPosition: 4,
		ZPosition: 0,
		Face:      1,
	})
	if !ok {
		t.Fatal("handleBlockDig returned false")
	}

	after, _ := srv.world.getBlock(0, 4, 0)
	if after != 0 {
		t.Fatalf("block was not removed: got=%d want=0", after)
	}
}

func TestHandleBlockDigStatus4DropsOneFromHeldStack(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  5,
		ItemDamage: 0,
	}

	ok := session.handleBlockDig(&protocol.Packet14BlockDig{Status: 4})
	if !ok {
		t.Fatal("handleBlockDig returned false")
	}
	if session.inventory[36] == nil || session.inventory[36].StackSize != 4 {
		t.Fatalf("drop-one stack mismatch: %#v", session.inventory[36])
	}
}

func TestHandleBlockDigStatus3DropsWholeHeldStack(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  5,
		ItemDamage: 0,
	}

	ok := session.handleBlockDig(&protocol.Packet14BlockDig{Status: 3})
	if !ok {
		t.Fatal("handleBlockDig returned false")
	}
	if session.inventory[36] != nil {
		t.Fatalf("drop-all should clear held stack: %#v", session.inventory[36])
	}
}

func TestHandleBlockDigStatus4SpawnsDroppedItemPacketsToWatcher(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	actor := newInteractionTestSession(srv, io.Discard)
	actor.entityID = 300
	actor.playerRegistered = true
	actor.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  2,
		ItemDamage: 0,
	}

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 301
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[actor] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{actor, watcher}
	srv.activeMu.Unlock()

	if !actor.handleBlockDig(&protocol.Packet14BlockDig{Status: 4}) {
		t.Fatal("handleBlockDig returned false")
	}
	if actor.inventory[36] == nil || actor.inventory[36].StackSize != 1 {
		t.Fatalf("drop-one stack mismatch: %#v", actor.inventory[36])
	}

	first, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher spawn packet: %v", err)
	}
	spawn, ok := first.(*protocol.Packet23VehicleSpawn)
	if !ok {
		t.Fatalf("expected Packet23VehicleSpawn, got %T", first)
	}
	if spawn.Type != entityTypeDroppedItem {
		t.Fatalf("spawn entity type mismatch: got=%d want=%d", spawn.Type, entityTypeDroppedItem)
	}

	second, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher metadata packet: %v", err)
	}
	meta, ok := second.(*protocol.Packet40EntityMetadata)
	if !ok {
		t.Fatalf("expected Packet40EntityMetadata, got %T", second)
	}
	if len(meta.Metadata) != 1 {
		t.Fatalf("metadata entry count mismatch: %#v", meta.Metadata)
	}
	stack, ok := meta.Metadata[0].Value.(*protocol.ItemStack)
	if !ok || stack == nil {
		t.Fatalf("metadata stack type mismatch: %#v", meta.Metadata[0].Value)
	}
	if stack.ItemID != 1 || stack.StackSize != 1 || stack.ItemDamage != 0 {
		t.Fatalf("dropped item metadata mismatch: %#v", stack)
	}
}

func TestTickDroppedItemsPickupBroadcastsCollectAndDestroy(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	dropper := newInteractionTestSession(srv, io.Discard)
	dropper.entityID = 320
	dropper.playerRegistered = true

	var pickerBuf bytes.Buffer
	picker := newInteractionTestSession(srv, &pickerBuf)
	picker.entityID = 321
	picker.playerRegistered = true
	picker.playerHealth = 20

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 322
	watcher.playerRegistered = true
	watcher.playerHealth = 20

	srv.activeMu.Lock()
	srv.activePlayers[dropper] = "dropper"
	srv.activePlayers[picker] = "picker"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{dropper, picker, watcher}
	srv.activeMu.Unlock()

	item := srv.spawnDroppedItemFromPlayer(dropper, &protocol.ItemStack{
		ItemID:     4,
		StackSize:  1,
		ItemDamage: 0,
	}, false, false)
	if item == nil {
		t.Fatal("spawnDroppedItemFromPlayer returned nil")
	}

	// Force immediate pickup range to make this deterministic.
	srv.droppedItemMu.Lock()
	item.DelayBeforeCanPick = 0
	item.X = picker.playerX
	item.Y = picker.playerY
	item.Z = picker.playerZ
	item.MotionX = 0
	item.MotionY = 0
	item.MotionZ = 0
	srv.droppedItemMu.Unlock()

	pickerBuf.Reset()
	watcherBuf.Reset()

	srv.TickDroppedItems()

	srv.droppedItemMu.Lock()
	_, exists := srv.droppedItems[item.EntityID]
	srv.droppedItemMu.Unlock()
	if exists {
		t.Fatal("expected dropped item to be removed after pickup")
	}
	if picker.inventory[36] == nil || picker.inventory[36].ItemID != 4 || picker.inventory[36].StackSize != 1 {
		t.Fatalf("picker inventory mismatch after pickup: %#v", picker.inventory[36])
	}

	sawCollect := false
	sawDestroy := false
	for {
		packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		switch p := packet.(type) {
		case *protocol.Packet22Collect:
			if p.CollectedEntityID == item.EntityID && p.CollectorEntityID == picker.entityID {
				sawCollect = true
			}
		case *protocol.Packet29DestroyEntity:
			if len(p.EntityIDs) == 1 && p.EntityIDs[0] == item.EntityID {
				sawDestroy = true
			}
		}
	}
	if !sawCollect {
		t.Fatal("expected watcher to receive Packet22Collect")
	}
	if !sawDestroy {
		t.Fatal("expected watcher to receive Packet29DestroyEntity for picked item")
	}
}

func TestHandlePlacePlacesBlockFromPacketItem(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  64,
		ItemDamage: 2,
	}

	ok := session.handlePlace(&protocol.Packet15Place{
		XPosition: 0,
		YPosition: 4,
		ZPosition: 0,
		Direction: 1,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  64,
			ItemDamage: 2,
		},
	})
	if !ok {
		t.Fatal("handlePlace returned false")
	}

	blockID, metadata := srv.world.getBlock(0, 5, 0)
	if blockID != 1 || metadata != 2 {
		t.Fatalf("placed block mismatch: got=(%d,%d) want=(1,2)", blockID, metadata)
	}
}

func TestHandlePlaceBroadcastsBlockChangeToWatchingPlayers(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	placer := newInteractionTestSession(srv, io.Discard)
	placer.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  64,
		ItemDamage: 0,
	}

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)

	srv.activeMu.Lock()
	srv.activePlayers[placer] = "placer"
	srv.activePlayers[watcher] = "watcher"
	srv.activeMu.Unlock()

	ok := placer.handlePlace(&protocol.Packet15Place{
		XPosition: 0,
		YPosition: 4,
		ZPosition: 0,
		Direction: 1,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  64,
			ItemDamage: 0,
		},
	})
	if !ok {
		t.Fatal("handlePlace returned false")
	}

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("watcher did not receive block change packet: %v", err)
	}
	change, ok := packet.(*protocol.Packet53BlockChange)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if change.XPosition != 0 || change.YPosition != 5 || change.ZPosition != 0 || change.Type != 1 {
		t.Fatalf("block change mismatch: %#v", change)
	}
}

func TestHandlePlaceReplacesTallGrassWithoutOffset(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  1,
		ItemDamage: 0,
	}
	if !srv.world.setBlock(1, 5, 1, 31, 0) {
		t.Fatal("failed to seed replaceable block")
	}

	ok := session.handlePlace(&protocol.Packet15Place{
		XPosition: 1,
		YPosition: 5,
		ZPosition: 1,
		Direction: 5,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  1,
			ItemDamage: 0,
		},
	})
	if !ok {
		t.Fatal("handlePlace returned false")
	}

	blockID, _ := srv.world.getBlock(1, 5, 1)
	if blockID != 1 {
		t.Fatalf("replaceable block was not replaced in-place: got=%d want=1", blockID)
	}
	blockID, _ = srv.world.getBlock(2, 5, 1)
	if blockID != 0 {
		t.Fatalf("unexpected offset placement: got=%d want=0", blockID)
	}
}

func TestHandleSlashCommandGiveSetsHotbarSlot(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)

	if !session.handleSlashCommand("/give 1 1") {
		t.Fatal("handleSlashCommand returned false")
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read set slot packet: %v", err)
	}
	setSlot, ok := packet.(*protocol.Packet103SetSlot)
	if !ok {
		t.Fatalf("expected Packet103SetSlot, got %T", packet)
	}
	if setSlot.WindowID != 0 || setSlot.ItemSlot != 36 {
		t.Fatalf("set slot header mismatch: %#v", setSlot)
	}
	if setSlot.ItemStack == nil || setSlot.ItemStack.ItemID != 1 || setSlot.ItemStack.StackSize != 1 {
		t.Fatalf("set slot stack mismatch: %#v", setSlot.ItemStack)
	}
}

func TestHandlePlaceConsumesHeldInventoryItem(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  1,
		ItemDamage: 0,
	}
	session.heldItemSlot = 0

	ok := session.handlePlace(&protocol.Packet15Place{
		XPosition: 0,
		YPosition: 4,
		ZPosition: 0,
		Direction: 1,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  1,
			ItemDamage: 0,
		},
	})
	if !ok {
		t.Fatal("handlePlace returned false")
	}

	blockID, _ := srv.world.getBlock(0, 5, 0)
	if blockID != 1 {
		t.Fatalf("placed block mismatch: got=%d want=1", blockID)
	}
	if session.inventory[36] != nil {
		t.Fatalf("expected held slot to be consumed, got=%#v", session.inventory[36])
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read inventory sync packet: %v", err)
	}
	setSlot, ok := packet.(*protocol.Packet103SetSlot)
	if !ok {
		t.Fatalf("expected Packet103SetSlot, got %T", packet)
	}
	if setSlot.ItemSlot != 36 || setSlot.ItemStack != nil {
		t.Fatalf("set slot mismatch after consume: %#v", setSlot)
	}
}

func TestHandlePlaceCreativeDoesNotConsumeHeldInventoryItem(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.gameType = 1
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  1,
		ItemDamage: 0,
	}

	ok := session.handlePlace(&protocol.Packet15Place{
		XPosition: 0,
		YPosition: 4,
		ZPosition: 0,
		Direction: 1,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  1,
			ItemDamage: 0,
		},
	})
	if !ok {
		t.Fatal("handlePlace returned false")
	}

	blockID, _ := srv.world.getBlock(0, 5, 0)
	if blockID != 1 {
		t.Fatalf("placed block mismatch: got=%d want=1", blockID)
	}
	if session.inventory[36] == nil || session.inventory[36].StackSize != 1 {
		t.Fatalf("creative should not consume item stack, got=%#v", session.inventory[36])
	}
}

func TestHandlePlaceSurvivalWithoutHeldItemDoesNothing(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)

	ok := session.handlePlace(&protocol.Packet15Place{
		XPosition: 0,
		YPosition: 4,
		ZPosition: 0,
		Direction: 1,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  1,
			ItemDamage: 0,
		},
	})
	if !ok {
		t.Fatal("handlePlace returned false")
	}

	blockID, _ := srv.world.getBlock(0, 5, 0)
	if blockID != 0 {
		t.Fatalf("survival place without held item should fail: got=%d want=0", blockID)
	}
}

func TestHandleWindowClickLeftPickupSendsTransactionAndSetSlots(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  5,
		ItemDamage: 0,
	}

	if !session.handleWindowClick(&protocol.Packet102WindowClick{
		WindowID:      0,
		InventorySlot: 36,
		MouseClick:    0,
		ActionNumber:  7,
		HoldingShift:  false,
	}) {
		t.Fatal("handleWindowClick returned false")
	}

	if session.inventory[36] != nil {
		t.Fatalf("slot 36 should be empty after pickup, got=%#v", session.inventory[36])
	}
	if session.cursorItem == nil || session.cursorItem.ItemID != 1 || session.cursorItem.StackSize != 5 {
		t.Fatalf("cursor mismatch: %#v", session.cursorItem)
	}

	first, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read transaction packet: %v", err)
	}
	tx, ok := first.(*protocol.Packet106Transaction)
	if !ok {
		t.Fatalf("expected Packet106Transaction, got %T", first)
	}
	if tx.WindowID != 0 || tx.ActionNumber != 7 || !tx.Accepted {
		t.Fatalf("transaction mismatch: %#v", tx)
	}

	second, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read slot packet: %v", err)
	}
	slot, ok := second.(*protocol.Packet103SetSlot)
	if !ok {
		t.Fatalf("expected Packet103SetSlot, got %T", second)
	}
	if slot.ItemSlot != 36 || slot.ItemStack != nil {
		t.Fatalf("slot update mismatch: %#v", slot)
	}

	third, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read cursor packet: %v", err)
	}
	cursor, ok := third.(*protocol.Packet103SetSlot)
	if !ok {
		t.Fatalf("expected Packet103SetSlot cursor packet, got %T", third)
	}
	if cursor.WindowID != -1 || cursor.ItemSlot != -1 || cursor.ItemStack == nil || cursor.ItemStack.ItemID != 1 || cursor.ItemStack.StackSize != 5 {
		t.Fatalf("cursor update mismatch: %#v", cursor)
	}
}

func TestHandleWindowClickShiftMainToHotbar(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.inventory[9] = &protocol.ItemStack{
		ItemID:     4,
		StackSize:  3,
		ItemDamage: 0,
	}
	session.inventory[36] = nil

	if !session.handleWindowClick(&protocol.Packet102WindowClick{
		WindowID:      0,
		InventorySlot: 9,
		MouseClick:    0,
		ActionNumber:  9,
		HoldingShift:  true,
	}) {
		t.Fatal("handleWindowClick returned false")
	}

	first, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read transaction packet: %v", err)
	}
	tx, ok := first.(*protocol.Packet106Transaction)
	if !ok {
		t.Fatalf("expected Packet106Transaction, got %T", first)
	}
	if !tx.Accepted {
		t.Fatalf("expected accepted transaction, got %#v", tx)
	}

	if session.inventory[9] != nil {
		t.Fatalf("source slot should be empty after shift move, got=%#v", session.inventory[9])
	}
	if session.inventory[36] == nil || session.inventory[36].ItemID != 4 || session.inventory[36].StackSize != 3 {
		t.Fatalf("target hotbar slot mismatch: %#v", session.inventory[36])
	}

	second, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read first slot update: %v", err)
	}
	slotA, ok := second.(*protocol.Packet103SetSlot)
	if !ok {
		t.Fatalf("expected Packet103SetSlot, got %T", second)
	}
	third, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read second slot update: %v", err)
	}
	slotB, ok := third.(*protocol.Packet103SetSlot)
	if !ok {
		t.Fatalf("expected Packet103SetSlot, got %T", third)
	}
	if (slotA.ItemSlot != 9 && slotA.ItemSlot != 36) || (slotB.ItemSlot != 9 && slotB.ItemSlot != 36) || slotA.ItemSlot == slotB.ItemSlot {
		t.Fatalf("unexpected slot updates: %#v %#v", slotA, slotB)
	}
}

func TestHandleWindowClickShiftHotbarToMain(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  2,
		ItemDamage: 0,
	}
	session.inventory[9] = nil

	if !session.handleWindowClick(&protocol.Packet102WindowClick{
		WindowID:      0,
		InventorySlot: 36,
		MouseClick:    0,
		ActionNumber:  10,
		HoldingShift:  true,
	}) {
		t.Fatal("handleWindowClick returned false")
	}

	if session.inventory[36] != nil {
		t.Fatalf("hotbar source slot should be empty, got=%#v", session.inventory[36])
	}
	if session.inventory[9] == nil || session.inventory[9].ItemID != 1 || session.inventory[9].StackSize != 2 {
		t.Fatalf("main inventory target mismatch: %#v", session.inventory[9])
	}
}

func TestHandleWindowClickRejectsUnknownWindowAndResyncs(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     4,
		StackSize:  3,
		ItemDamage: 0,
	}

	if !session.handleWindowClick(&protocol.Packet102WindowClick{
		WindowID:      1,
		InventorySlot: 36,
		MouseClick:    0,
		ActionNumber:  11,
		HoldingShift:  true,
	}) {
		t.Fatal("handleWindowClick returned false")
	}

	first, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read transaction packet: %v", err)
	}
	tx, ok := first.(*protocol.Packet106Transaction)
	if !ok {
		t.Fatalf("expected Packet106Transaction, got %T", first)
	}
	if tx.Accepted {
		t.Fatalf("expected rejected transaction, got accepted=true: %#v", tx)
	}
}

func TestHandleSlashCommandTimeSetBroadcastsUpdate(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	var selfBuf bytes.Buffer
	session := newInteractionTestSession(srv, &selfBuf)
	session.clientUsername = "self"

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.clientUsername = "watcher"

	srv.activeMu.Lock()
	srv.activePlayers[session] = "self"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{session, watcher}
	srv.activeMu.Unlock()

	if !session.handleSlashCommand("/time set 2000") {
		t.Fatal("handleSlashCommand returned false")
	}

	packetSelf, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read self packet: %v", err)
	}
	updateSelf, ok := packetSelf.(*protocol.Packet4UpdateTime)
	if !ok {
		t.Fatalf("expected Packet4UpdateTime, got %T", packetSelf)
	}
	if updateSelf.Time != 2000 {
		t.Fatalf("self time mismatch: got=%d want=2000", updateSelf.Time)
	}

	packetWatcher, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher packet: %v", err)
	}
	updateWatcher, ok := packetWatcher.(*protocol.Packet4UpdateTime)
	if !ok {
		t.Fatalf("expected Packet4UpdateTime, got %T", packetWatcher)
	}
	if updateWatcher.Time != 2000 {
		t.Fatalf("watcher time mismatch: got=%d want=2000", updateWatcher.Time)
	}

	_, worldTime := srv.CurrentWorldTime()
	if worldTime != 2000 {
		t.Fatalf("server world time mismatch: got=%d want=2000", worldTime)
	}
}

func TestHandleSlashCommandGamemodeCreativeSendsAbilities(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)

	if !session.handleSlashCommand("/gamemode creative") {
		t.Fatal("handleSlashCommand returned false")
	}

	first, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read abilities packet: %v", err)
	}
	abilities, ok := first.(*protocol.Packet202PlayerAbilities)
	if !ok {
		t.Fatalf("expected Packet202PlayerAbilities, got %T", first)
	}
	if !abilities.IsCreative || !abilities.AllowFlying || !abilities.DisableDamage {
		t.Fatalf("abilities mismatch for creative mode: %#v", abilities)
	}
	if abilities.WalkSpeed != 0.1 || abilities.FlySpeed != 0.05 {
		t.Fatalf("abilities speed mismatch: %#v", abilities)
	}
}

func TestHandlePlayerAbilitiesFlyingRequiresAllowFlying(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)

	session.gameType = 1
	session.handlePlayerAbilities(&protocol.Packet202PlayerAbilities{IsFlying: true})
	if !session.playerIsFlying {
		t.Fatal("expected creative player to enter flying state")
	}
	session.handlePlayerAbilities(&protocol.Packet202PlayerAbilities{IsFlying: false})
	if session.playerIsFlying {
		t.Fatal("expected flying state to clear when packet disables it")
	}

	session.gameType = 0
	session.handlePlayerAbilities(&protocol.Packet202PlayerAbilities{IsFlying: true})
	if session.playerIsFlying {
		t.Fatal("survival player must not be allowed to set flying")
	}
}

func TestHandleSlashCommandGamemodeSurvivalClearsFlyingState(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.gameType = 1
	session.playerIsFlying = true

	if !session.handleSlashCommand("/gamemode survival") {
		t.Fatal("handleSlashCommand returned false")
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read abilities packet: %v", err)
	}
	abilities, ok := packet.(*protocol.Packet202PlayerAbilities)
	if !ok {
		t.Fatalf("expected Packet202PlayerAbilities, got %T", packet)
	}
	if abilities.IsCreative || abilities.AllowFlying || abilities.IsFlying || abilities.DisableDamage {
		t.Fatalf("survival abilities mismatch: %#v", abilities)
	}
	if session.playerIsFlying {
		t.Fatal("server flying state should be cleared in survival mode")
	}
}

func TestHandleSlashCommandKillSendsZeroHealth(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.playerHealth = 20
	session.playerFood = 20
	session.playerSat = 5

	if !session.handleSlashCommand("/kill") {
		t.Fatal("handleSlashCommand returned false")
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read health packet: %v", err)
	}
	health, ok := packet.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected Packet8UpdateHealth, got %T", packet)
	}
	if health.HealthMP != 0 {
		t.Fatalf("health mismatch: got=%f want=0", health.HealthMP)
	}

	if !session.playerDead {
		t.Fatal("session should be marked dead")
	}
}

func TestHandleClientCommandRespawnAfterDeath(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.playerInitialized = true
	session.playerDead = true
	session.playerHealth = 0
	session.playerFood = 5
	session.playerSat = 0

	if !session.handleClientCommand(&protocol.Packet205ClientCommand{ForceRespawn: 1}) {
		t.Fatal("handleClientCommand returned false")
	}

	first, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read respawn packet: %v", err)
	}
	respawn, ok := first.(*protocol.Packet9Respawn)
	if !ok {
		t.Fatalf("expected Packet9Respawn, got %T", first)
	}
	if respawn.GameType != 0 || respawn.RespawnDimension != 0 {
		t.Fatalf("respawn payload mismatch: %#v", respawn)
	}

	second, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read location packet: %v", err)
	}
	if _, ok := second.(*protocol.Packet13PlayerLookMove); !ok {
		t.Fatalf("expected Packet13PlayerLookMove, got %T", second)
	}

	third, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read health packet: %v", err)
	}
	health, ok := third.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected Packet8UpdateHealth, got %T", third)
	}
	if health.HealthMP != 20 || health.Food != 20 {
		t.Fatalf("respawn health mismatch: %#v", health)
	}

	if session.playerDead {
		t.Fatal("player should be alive after respawn")
	}
}

func TestHandleCreativeSetSlotUpdatesInventoryInCreative(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.gameType = 1

	session.handleCreativeSetSlot(&protocol.Packet107CreativeSetSlot{
		Slot: 37,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  64,
			ItemDamage: 0,
		},
	})

	if session.inventory[37] == nil || session.inventory[37].ItemID != 1 || session.inventory[37].StackSize != 64 {
		t.Fatalf("creative set slot mismatch: %#v", session.inventory[37])
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read slot sync packet: %v", err)
	}
	setSlot, ok := packet.(*protocol.Packet103SetSlot)
	if !ok {
		t.Fatalf("expected Packet103SetSlot, got %T", packet)
	}
	if setSlot.ItemSlot != 37 || setSlot.ItemStack == nil || setSlot.ItemStack.ItemID != 1 {
		t.Fatalf("slot sync mismatch: %#v", setSlot)
	}
}

func TestHandleCreativeSetSlotIgnoredInSurvival(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.gameType = 0

	session.handleCreativeSetSlot(&protocol.Packet107CreativeSetSlot{
		Slot: 37,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  64,
			ItemDamage: 0,
		},
	})

	if session.inventory[37] != nil {
		t.Fatalf("slot should not be updated in survival: %#v", session.inventory[37])
	}
	if _, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound); err == nil {
		t.Fatal("unexpected packet emitted for survival creative-set-slot")
	}
}

func TestHandleCreativeSetSlotNegativeDropsAndRespectsSpamThrottle(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.gameType = 1
	session.entityID = 330
	session.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[session] = "creative"
	srv.activeOrder = []*loginSession{session}
	srv.activeMu.Unlock()

	dropPacket := &protocol.Packet107CreativeSetSlot{
		Slot: -1,
		ItemStack: &protocol.ItemStack{
			ItemID:     1,
			StackSize:  1,
			ItemDamage: 0,
		},
	}

	for i := 0; i < 21; i++ {
		session.handleCreativeSetSlot(dropPacket)
	}

	if session.creativeItemCreationSpamThresholdTally != 200 {
		t.Fatalf("creative spam tally mismatch: got=%d want=200", session.creativeItemCreationSpamThresholdTally)
	}

	srv.droppedItemMu.Lock()
	count := len(srv.droppedItems)
	var sampled *trackedDroppedItem
	for _, item := range srv.droppedItems {
		sampled = item
		break
	}
	srv.droppedItemMu.Unlock()

	// First 10 calls are accepted (20 -> 200), later calls are throttled.
	if count != 10 {
		t.Fatalf("creative drop count mismatch: got=%d want=10", count)
	}
	if sampled == nil {
		t.Fatal("expected at least one dropped item")
	}
	if sampled.AgeTicks != droppedItemCreativeAgeTicks {
		t.Fatalf("creative dropped-item age mismatch: got=%d want=%d", sampled.AgeTicks, droppedItemCreativeAgeTicks)
	}
}

func TestSanitizeLoadedPlayerPositionResetUsesSafeSpawn(t *testing.T) {
	worldDir := t.TempDir()
	if err := writeTestLevelDat(worldDir, func(data *nbt.CompoundTag) {
		data.SetString("generatorName", "default")
		data.SetInteger("generatorVersion", 1)
		data.SetLong("RandomSeed", 12345)
		data.SetInteger("SpawnX", 0)
		data.SetInteger("SpawnY", 1)
		data.SetInteger("SpawnZ", 0)
	}); err != nil {
		t.Fatalf("write level.dat failed: %v", err)
	}

	srv := NewStatusServer(StatusConfig{
		PersistWorld: true,
		WorldDir:     worldDir,
	})
	defer func() {
		_ = srv.Close()
	}()
	session := newInteractionTestSession(srv, io.Discard)
	state := defaultPersistedPlayerState()
	state.Y = -1

	got := session.sanitizeLoadedPlayerPosition(state)
	blockX := int(math.Floor(got.X))
	blockY := int(math.Floor(got.Y))
	blockZ := int(math.Floor(got.Z))

	feetID, _ := srv.world.getBlock(blockX, blockY, blockZ)
	headID, _ := srv.world.getBlock(blockX, blockY+1, blockZ)
	belowID, _ := srv.world.getBlock(blockX, blockY-1, blockZ)
	if block.BlocksMovement(feetID) || block.BlocksMovement(headID) || !block.BlocksMovement(belowID) {
		t.Fatalf("reset spawn is not standable: pos=(%.1f,%.1f,%.1f) feet=%d head=%d below=%d", got.X, got.Y, got.Z, feetID, headID, belowID)
	}
}

func TestHandleFlyingBroadcastsRelativeMove(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	mover := newInteractionTestSession(srv, io.Discard)
	mover.clientUsername = "mover"
	mover.entityID = 123
	mover.playerRegistered = true
	mover.lastEntityPosX = toPacketPosition(mover.playerX)
	mover.lastEntityPosY = toPacketPosition(mover.playerY)
	mover.lastEntityPosZ = toPacketPosition(mover.playerZ)
	mover.lastEntityYaw = toPacketAngle(mover.playerYaw)
	mover.lastEntityPitch = toPacketAngle(mover.playerPitch)

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.clientUsername = "watcher"
	watcher.entityID = 456
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[mover] = "mover"
	srv.activePlayers[watcher] = "watcher"
	srv.activeMu.Unlock()
	mover.markSeenBy(watcher, true)

	move := protocol.NewPacket11PlayerPosition()
	move.XPosition = 1.5
	move.YPosition = 5.0
	move.Stance = 6.6200000047683716
	move.ZPosition = 0.5
	move.OnGround = true
	if !mover.handleFlying(&move.Packet10Flying) {
		t.Fatal("handleFlying returned false")
	}

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("watcher did not receive movement packet: %v", err)
	}
	relMove, ok := packet.(*protocol.Packet31RelEntityMove)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if relMove.EntityID != 123 || relMove.XPosition != 32 {
		t.Fatalf("relative move mismatch: %#v", relMove)
	}
}

func TestHandleFlyingLandingAppliesFallDamage(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var selfBuf bytes.Buffer
	session := newInteractionTestSession(srv, &selfBuf)
	session.entityID = 330
	session.playerRegistered = true
	session.playerHealth = 20
	session.playerFood = 20
	session.playerSat = 5
	session.playerOnGround = false
	session.playerFallDistance = 3.2

	move := protocol.NewPacket11PlayerPosition()
	move.XPosition = session.playerX
	move.YPosition = session.playerY
	move.Stance = session.playerY + 1.6200000047683716
	move.ZPosition = session.playerZ
	move.OnGround = true

	if !session.handleFlying(&move.Packet10Flying) {
		t.Fatal("handleFlying returned false")
	}

	first, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected first packet, got read err: %v", err)
	}
	health, ok := first.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected first packet Packet8UpdateHealth, got %T", first)
	}
	if health.HealthMP != 19 {
		t.Fatalf("fall damage health mismatch: got=%f want=19", health.HealthMP)
	}

	second, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected second packet, got read err: %v", err)
	}
	status, ok := second.(*protocol.Packet38EntityStatus)
	if !ok {
		t.Fatalf("expected second packet Packet38EntityStatus, got %T", second)
	}
	if status.EntityID != session.entityID || status.EntityStatus != 2 {
		t.Fatalf("unexpected status packet: %#v", status)
	}
}

func TestHandleFlyingCreativeLandingSkipsFallDamage(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var selfBuf bytes.Buffer
	session := newInteractionTestSession(srv, &selfBuf)
	session.entityID = 331
	session.playerRegistered = true
	session.gameType = 1
	session.playerHealth = 20
	session.playerOnGround = false
	session.playerFallDistance = 20

	move := protocol.NewPacket11PlayerPosition()
	move.XPosition = session.playerX
	move.YPosition = session.playerY
	move.Stance = session.playerY + 1.6200000047683716
	move.ZPosition = session.playerZ
	move.OnGround = true

	if !session.handleFlying(&move.Packet10Flying) {
		t.Fatal("handleFlying returned false")
	}
	if session.playerHealth != 20 {
		t.Fatalf("creative fall should not damage: got=%f want=20", session.playerHealth)
	}
	if _, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound); err == nil {
		t.Fatal("unexpected packet emitted for creative no-damage fall")
	}
}

func TestHandleFlyingBelowVoidAppliesOutOfWorldDamage(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var selfBuf bytes.Buffer
	session := newInteractionTestSession(srv, &selfBuf)
	session.entityID = 332
	session.playerRegistered = true
	session.playerHealth = 20
	session.playerFood = 20
	session.playerSat = 5
	session.playerY = -70.0
	session.playerStance = -68.37999999523163

	move := protocol.NewPacket11PlayerPosition()
	move.XPosition = session.playerX
	move.YPosition = -70.0
	move.Stance = -68.37999999523163
	move.ZPosition = session.playerZ
	move.OnGround = false

	if !session.handleFlying(&move.Packet10Flying) {
		t.Fatal("handleFlying returned false")
	}

	first, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected first packet, got read err: %v", err)
	}
	health, ok := first.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected first packet Packet8UpdateHealth, got %T", first)
	}
	if health.HealthMP != 16 {
		t.Fatalf("out-of-world health mismatch: got=%f want=16", health.HealthMP)
	}

	second, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected second packet, got read err: %v", err)
	}
	status, ok := second.(*protocol.Packet38EntityStatus)
	if !ok {
		t.Fatalf("expected second packet Packet38EntityStatus, got %T", second)
	}
	if status.EntityID != session.entityID || status.EntityStatus != 2 {
		t.Fatalf("unexpected status packet: %#v", status)
	}
}

func TestHandleFlyingGroundMovementAddsExhaustion(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	walker := newInteractionTestSession(srv, io.Discard)
	walker.playerOnGround = true
	moveWalk := protocol.NewPacket11PlayerPosition()
	moveWalk.XPosition = walker.playerX + 1.0
	moveWalk.YPosition = walker.playerY
	moveWalk.Stance = walker.playerY + 1.6200000047683716
	moveWalk.ZPosition = walker.playerZ
	moveWalk.OnGround = true
	if !walker.handleFlying(&moveWalk.Packet10Flying) {
		t.Fatal("walker handleFlying returned false")
	}
	if math.Abs(float64(walker.playerFoodExhaust-0.01)) > 1e-6 {
		t.Fatalf("walking exhaustion mismatch: got=%f want=0.01", walker.playerFoodExhaust)
	}

	sprinter := newInteractionTestSession(srv, io.Discard)
	sprinter.playerOnGround = true
	sprinter.playerSprinting = true
	moveSprint := protocol.NewPacket11PlayerPosition()
	moveSprint.XPosition = sprinter.playerX + 1.0
	moveSprint.YPosition = sprinter.playerY
	moveSprint.Stance = sprinter.playerY + 1.6200000047683716
	moveSprint.ZPosition = sprinter.playerZ
	moveSprint.OnGround = true
	if !sprinter.handleFlying(&moveSprint.Packet10Flying) {
		t.Fatal("sprinter handleFlying returned false")
	}
	if math.Abs(float64(sprinter.playerFoodExhaust-0.099999994)) > 1e-6 {
		t.Fatalf("sprinting exhaustion mismatch: got=%f want=0.099999994", sprinter.playerFoodExhaust)
	}

	creative := newInteractionTestSession(srv, io.Discard)
	creative.gameType = 1
	creative.playerOnGround = true
	creative.playerSprinting = true
	moveCreative := protocol.NewPacket11PlayerPosition()
	moveCreative.XPosition = creative.playerX + 1.0
	moveCreative.YPosition = creative.playerY
	moveCreative.Stance = creative.playerY + 1.6200000047683716
	moveCreative.ZPosition = creative.playerZ
	moveCreative.OnGround = true
	if !creative.handleFlying(&moveCreative.Packet10Flying) {
		t.Fatal("creative handleFlying returned false")
	}
	if creative.playerFoodExhaust != 0 {
		t.Fatalf("creative should not gain exhaustion: got=%f want=0", creative.playerFoodExhaust)
	}

	jumper := newInteractionTestSession(srv, io.Discard)
	jumper.playerOnGround = true
	moveJump := protocol.NewPacket11PlayerPosition()
	moveJump.XPosition = jumper.playerX
	moveJump.YPosition = jumper.playerY + 0.42
	moveJump.Stance = moveJump.YPosition + 1.6200000047683716
	moveJump.ZPosition = jumper.playerZ
	moveJump.OnGround = false
	if !jumper.handleFlying(&moveJump.Packet10Flying) {
		t.Fatal("jumper handleFlying returned false")
	}
	if math.Abs(float64(jumper.playerFoodExhaust-0.2)) > 1e-6 {
		t.Fatalf("jump exhaustion mismatch: got=%f want=0.2", jumper.playerFoodExhaust)
	}

	sprintJumper := newInteractionTestSession(srv, io.Discard)
	sprintJumper.playerOnGround = true
	sprintJumper.playerSprinting = true
	moveSprintJump := protocol.NewPacket11PlayerPosition()
	moveSprintJump.XPosition = sprintJumper.playerX
	moveSprintJump.YPosition = sprintJumper.playerY + 0.42
	moveSprintJump.Stance = moveSprintJump.YPosition + 1.6200000047683716
	moveSprintJump.ZPosition = sprintJumper.playerZ
	moveSprintJump.OnGround = false
	if !sprintJumper.handleFlying(&moveSprintJump.Packet10Flying) {
		t.Fatal("sprintJumper handleFlying returned false")
	}
	if math.Abs(float64(sprintJumper.playerFoodExhaust-0.8)) > 1e-6 {
		t.Fatalf("sprint jump exhaustion mismatch: got=%f want=0.8", sprintJumper.playerFoodExhaust)
	}
}

func TestHandleKeepAliveResponseUpdatesSmoothedPing(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.keepAliveExpectedID = 123
	session.keepAlivePending.Store(true)
	session.keepAliveSentAtMS = time.Now().UnixMilli() - 200
	session.latencyMS.Store(100)

	session.handleKeepAliveResponse(&protocol.Packet0KeepAlive{RandomID: 123})

	got := session.currentPing()
	if got < 124 || got > 126 {
		t.Fatalf("smoothed ping out of expected range: got=%d", got)
	}
	if session.keepAlivePending.Load() {
		t.Fatal("keepAlivePending should be cleared")
	}
}

func TestHandleFlyingBroadcastsLookAndHeadRotation(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	mover := newInteractionTestSession(srv, io.Discard)
	mover.clientUsername = "mover"
	mover.entityID = 123
	mover.playerRegistered = true
	mover.lastEntityPosX = toPacketPosition(mover.playerX)
	mover.lastEntityPosY = toPacketPosition(mover.playerY)
	mover.lastEntityPosZ = toPacketPosition(mover.playerZ)
	mover.lastEntityYaw = toPacketAngle(mover.playerYaw)
	mover.lastEntityPitch = toPacketAngle(mover.playerPitch)
	mover.lastHeadYaw = toPacketAngle(mover.playerYaw)

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.clientUsername = "watcher"
	watcher.entityID = 456
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[mover] = "mover"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{mover, watcher}
	srv.activeMu.Unlock()
	mover.markSeenBy(watcher, true)

	look := protocol.NewPacket12PlayerLook()
	look.Yaw = 45.0
	look.Pitch = 10.0
	look.OnGround = true
	if !mover.handleFlying(&look.Packet10Flying) {
		t.Fatal("handleFlying returned false")
	}

	first, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read first packet: %v", err)
	}
	if _, ok := first.(*protocol.Packet32EntityLook); !ok {
		t.Fatalf("expected Packet32EntityLook, got %T", first)
	}

	second, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read second packet: %v", err)
	}
	head, ok := second.(*protocol.Packet35EntityHeadRotation)
	if !ok {
		t.Fatalf("expected Packet35EntityHeadRotation, got %T", second)
	}
	if head.EntityID != 123 {
		t.Fatalf("unexpected head rotation entity id: got=%d want=123", head.EntityID)
	}
}

func TestHandleFlyingTooFastSendsPositionCorrection(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.playerYaw = 10
	session.playerPitch = 5

	move := protocol.NewPacket11PlayerPosition()
	move.XPosition = 20.5
	move.YPosition = 5.0
	move.Stance = 6.6200000047683716
	move.ZPosition = 0.5
	move.OnGround = true

	if !session.handleFlying(&move.Packet10Flying) {
		t.Fatal("handleFlying returned false")
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read correction packet: %v", err)
	}
	correction, ok := packet.(*protocol.Packet13PlayerLookMove)
	if !ok {
		t.Fatalf("expected Packet13PlayerLookMove, got %T", packet)
	}
	if correction.XPosition != 0.5 || correction.ZPosition != 0.5 || correction.Stance != 5.0 {
		t.Fatalf("correction packet mismatch: %#v", correction)
	}
}

func TestHandleSlashCommandTpTeleportsAndBroadcasts(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	var selfBuf bytes.Buffer
	session := newInteractionTestSession(srv, &selfBuf)
	session.playerRegistered = true
	session.clientUsername = "tp_user"
	session.entityID = 99
	session.lastEntityPosX = toPacketPosition(session.playerX)
	session.lastEntityPosY = toPacketPosition(session.playerY)
	session.lastEntityPosZ = toPacketPosition(session.playerZ)
	session.lastEntityYaw = toPacketAngle(session.playerYaw)
	session.lastEntityPitch = toPacketAngle(session.playerPitch)
	session.lastHeadYaw = toPacketAngle(session.playerYaw)

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.playerRegistered = true
	watcher.clientUsername = "watcher"
	watcher.entityID = 100

	srv.activeMu.Lock()
	srv.activePlayers[session] = "tp_user"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{session, watcher}
	srv.activeMu.Unlock()
	session.markSeenBy(watcher, true)

	if !session.handleSlashCommand("/tp 10 6 10") {
		t.Fatal("handleSlashCommand returned false")
	}

	packetSelf, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read self packet: %v", err)
	}
	if _, ok := packetSelf.(*protocol.Packet13PlayerLookMove); !ok {
		t.Fatalf("expected Packet13PlayerLookMove, got %T", packetSelf)
	}

	packetWatcher, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher packet: %v", err)
	}
	teleport, ok := packetWatcher.(*protocol.Packet34EntityTeleport)
	if !ok {
		t.Fatalf("expected Packet34EntityTeleport, got %T", packetWatcher)
	}
	if teleport.EntityID != 99 || teleport.XPosition != toPacketPosition(10) || teleport.YPosition != toPacketPosition(6) || teleport.ZPosition != toPacketPosition(10) {
		t.Fatalf("teleport packet mismatch: %#v", teleport)
	}
}

func TestHandleSlashCommandSetBlockUpdatesWorldAndBroadcasts(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)

	srv.activeMu.Lock()
	srv.activePlayers[session] = "caller"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{session, watcher}
	srv.activeMu.Unlock()

	if !session.handleSlashCommand("/setblock 1 5 1 1 3") {
		t.Fatal("handleSlashCommand returned false")
	}

	blockID, meta := srv.world.getBlock(1, 5, 1)
	if blockID != 1 || meta != 3 {
		t.Fatalf("world block mismatch: got=(%d,%d) want=(1,3)", blockID, meta)
	}

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher block change: %v", err)
	}
	change, ok := packet.(*protocol.Packet53BlockChange)
	if !ok {
		t.Fatalf("expected Packet53BlockChange, got %T", packet)
	}
	if change.XPosition != 1 || change.YPosition != 5 || change.ZPosition != 1 || change.Type != 1 || change.Metadata != 3 {
		t.Fatalf("block change mismatch: %#v", change)
	}
}

func TestHandleSlashCommandListIncludesPlayerNames(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.clientUsername = "caller"

	otherA := newInteractionTestSession(srv, io.Discard)
	otherA.clientUsername = "Alice"
	otherB := newInteractionTestSession(srv, io.Discard)
	otherB.clientUsername = "Bob"

	srv.activeMu.Lock()
	srv.activePlayers[session] = "caller"
	srv.activePlayers[otherA] = "Alice"
	srv.activePlayers[otherB] = "Bob"
	srv.activeOrder = []*loginSession{session, otherA, otherB}
	srv.activeMu.Unlock()

	if !session.handleSlashCommand("/list") {
		t.Fatal("handleSlashCommand returned false")
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read list chat packet: %v", err)
	}
	chat, ok := packet.(*protocol.Packet3Chat)
	if !ok {
		t.Fatalf("expected Packet3Chat, got %T", packet)
	}
	if !strings.Contains(chat.Message, "caller") || !strings.Contains(chat.Message, "Alice") || !strings.Contains(chat.Message, "Bob") {
		t.Fatalf("list output missing names: %q", chat.Message)
	}
}

func TestHandleSlashCommandXpAddsPointsAndSendsExperiencePacket(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.clientUsername = "Steve"

	srv.activeMu.Lock()
	srv.activePlayers[session] = "Steve"
	srv.activeOrder = []*loginSession{session}
	srv.activeMu.Unlock()

	if !session.handleSlashCommand("/xp 17") {
		t.Fatal("handleSlashCommand returned false")
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read feedback chat packet: %v", err)
	}
	chat, ok := packet.(*protocol.Packet3Chat)
	if !ok {
		t.Fatalf("expected Packet3Chat, got %T", packet)
	}
	if !strings.Contains(chat.Message, "Given 17 experience") {
		t.Fatalf("unexpected xp chat feedback: %q", chat.Message)
	}

	packet, err = protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read xp packet: %v", err)
	}
	xp, ok := packet.(*protocol.Packet43Experience)
	if !ok {
		t.Fatalf("expected Packet43Experience, got %T", packet)
	}
	if xp.ExperienceLevel != 1 || xp.ExperienceTotal != 17 || xp.Experience != 0.0 {
		t.Fatalf("xp packet mismatch: %#v", xp)
	}
}

func TestHandleSlashCommandXpLevelsSupportsNegative(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.clientUsername = "Steve"
	session.playerExpLevel = 5
	session.playerExpTotal = 50
	session.playerExperience = 0.3

	srv.activeMu.Lock()
	srv.activePlayers[session] = "Steve"
	srv.activeOrder = []*loginSession{session}
	srv.activeMu.Unlock()

	if !session.handleSlashCommand("/xp -2L") {
		t.Fatal("handleSlashCommand returned false")
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read feedback chat packet: %v", err)
	}
	chat, ok := packet.(*protocol.Packet3Chat)
	if !ok {
		t.Fatalf("expected Packet3Chat, got %T", packet)
	}
	if !strings.Contains(chat.Message, "Removed 2 levels") {
		t.Fatalf("unexpected xp level feedback: %q", chat.Message)
	}

	packet, err = protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read xp packet: %v", err)
	}
	xp, ok := packet.(*protocol.Packet43Experience)
	if !ok {
		t.Fatalf("expected Packet43Experience, got %T", packet)
	}
	if xp.ExperienceLevel != 3 || xp.ExperienceTotal != 50 {
		t.Fatalf("xp level packet mismatch: %#v", xp)
	}
}

func TestHandleSlashCommandXpTargetPlayer(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	var callerBuf bytes.Buffer
	caller := newInteractionTestSession(srv, &callerBuf)
	caller.clientUsername = "Caller"

	var targetBuf bytes.Buffer
	target := newInteractionTestSession(srv, &targetBuf)
	target.clientUsername = "Alex"

	srv.activeMu.Lock()
	srv.activePlayers[caller] = "Caller"
	srv.activePlayers[target] = "Alex"
	srv.activeOrder = []*loginSession{caller, target}
	srv.activeMu.Unlock()

	if !caller.handleSlashCommand("/xp 3L Alex") {
		t.Fatal("handleSlashCommand returned false")
	}

	packet, err := protocol.ReadPacket(&callerBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read caller feedback chat: %v", err)
	}
	chat, ok := packet.(*protocol.Packet3Chat)
	if !ok {
		t.Fatalf("expected caller Packet3Chat, got %T", packet)
	}
	if !strings.Contains(chat.Message, "Given 3 levels to Alex") {
		t.Fatalf("unexpected caller feedback: %q", chat.Message)
	}

	packet, err = protocol.ReadPacket(&targetBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read target xp packet: %v", err)
	}
	xp, ok := packet.(*protocol.Packet43Experience)
	if !ok {
		t.Fatalf("expected target Packet43Experience, got %T", packet)
	}
	if xp.ExperienceLevel != 3 {
		t.Fatalf("target level mismatch: %#v", xp)
	}
}

func TestHandleChatBroadcastsLegacyPlainMessage(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	sender := newInteractionTestSession(srv, io.Discard)
	sender.clientUsername = "Steve"

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.clientUsername = "Watcher"

	srv.activeMu.Lock()
	srv.activePlayers[sender] = "Steve"
	srv.activePlayers[watcher] = "Watcher"
	srv.activeOrder = []*loginSession{sender, watcher}
	srv.activeMu.Unlock()

	if !sender.handleChat(protocol.NewPacket3Chat("hello world", false)) {
		t.Fatal("handleChat returned false")
	}

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read chat packet: %v", err)
	}
	chat, ok := packet.(*protocol.Packet3Chat)
	if !ok {
		t.Fatalf("expected Packet3Chat, got %T", packet)
	}
	if chat.Message != "<Steve> hello world" {
		t.Fatalf("chat message mismatch: got=%q", chat.Message)
	}
	if strings.HasPrefix(chat.Message, "{") {
		t.Fatalf("chat message must be plain legacy text, got=%q", chat.Message)
	}
}

func TestSendSystemChatUsesPlainLegacyString(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)

	session.sendSystemChat("Usage: /gamemode <0|1|survival|creative>")

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read system chat packet: %v", err)
	}
	chat, ok := packet.(*protocol.Packet3Chat)
	if !ok {
		t.Fatalf("expected Packet3Chat, got %T", packet)
	}
	if chat.Message != "Usage: /gamemode <0|1|survival|creative>" {
		t.Fatalf("system chat mismatch: got=%q", chat.Message)
	}
	if strings.HasPrefix(chat.Message, "{") {
		t.Fatalf("system chat must be plain legacy text, got=%q", chat.Message)
	}
}

func TestHandleFlyingLeavesWatcherRangeSendsDestroy(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	mover := newInteractionTestSession(srv, io.Discard)
	mover.clientUsername = "mover"
	mover.entityID = 7
	mover.playerRegistered = true
	mover.lastEntityPosX = toPacketPosition(mover.playerX)
	mover.lastEntityPosY = toPacketPosition(mover.playerY)
	mover.lastEntityPosZ = toPacketPosition(mover.playerZ)
	mover.lastEntityYaw = toPacketAngle(mover.playerYaw)
	mover.lastEntityPitch = toPacketAngle(mover.playerPitch)
	mover.lastHeadYaw = toPacketAngle(mover.playerYaw)

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.clientUsername = "watcher"
	watcher.entityID = 8
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[mover] = "mover"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{mover, watcher}
	srv.activeMu.Unlock()
	mover.markSeenBy(watcher, true)

	step1 := protocol.NewPacket11PlayerPosition()
	step1.XPosition = 8.5
	step1.YPosition = 5.0
	step1.Stance = 6.6200000047683716
	step1.ZPosition = 0.5
	step1.OnGround = true
	if !mover.handleFlying(&step1.Packet10Flying) {
		t.Fatal("step1 handleFlying returned false")
	}
	_, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read step1 packet: %v", err)
	}

	step2 := protocol.NewPacket11PlayerPosition()
	step2.XPosition = 16.5
	step2.YPosition = 5.0
	step2.Stance = 6.6200000047683716
	step2.ZPosition = 0.5
	step2.OnGround = true
	if !mover.handleFlying(&step2.Packet10Flying) {
		t.Fatal("step2 handleFlying returned false")
	}

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read destroy packet: %v", err)
	}
	destroy, ok := packet.(*protocol.Packet29DestroyEntity)
	if !ok {
		t.Fatalf("expected Packet29DestroyEntity, got %T", packet)
	}
	if len(destroy.EntityIDs) != 1 || destroy.EntityIDs[0] != 7 {
		t.Fatalf("destroy packet mismatch: %#v", destroy.EntityIDs)
	}
}

func TestWatcherMovementRefreshesVisibleEntities(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	mover := newInteractionTestSession(srv, io.Discard)
	mover.clientUsername = "mover"
	mover.entityID = 21
	mover.playerRegistered = true
	mover.lastEntityPosX = toPacketPosition(mover.playerX)
	mover.lastEntityPosY = toPacketPosition(mover.playerY)
	mover.lastEntityPosZ = toPacketPosition(mover.playerZ)
	mover.lastEntityYaw = toPacketAngle(mover.playerYaw)
	mover.lastEntityPitch = toPacketAngle(mover.playerPitch)
	mover.lastHeadYaw = toPacketAngle(mover.playerYaw)

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.clientUsername = "watcher"
	watcher.entityID = 22
	watcher.playerRegistered = true
	watcher.lastEntityPosX = toPacketPosition(watcher.playerX)
	watcher.lastEntityPosY = toPacketPosition(watcher.playerY)
	watcher.lastEntityPosZ = toPacketPosition(watcher.playerZ)
	watcher.lastEntityYaw = toPacketAngle(watcher.playerYaw)
	watcher.lastEntityPitch = toPacketAngle(watcher.playerPitch)
	watcher.lastHeadYaw = toPacketAngle(watcher.playerYaw)

	srv.activeMu.Lock()
	srv.activePlayers[mover] = "mover"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{mover, watcher}
	srv.activeMu.Unlock()

	// Watcher currently sees mover.
	mover.markSeenBy(watcher, true)

	// Move watcher far enough so chunk (0,0) falls outside default view distance (10).
	positions := []float64{
		10.5, 20.5, 30.5, 40.5, 50.5, 60.5,
		70.5, 80.5, 90.5, 100.5, 110.5, 120.5,
		130.5, 140.5, 150.5, 160.5, 170.5, 176.5,
	}
	for i, x := range positions {
		step := protocol.NewPacket11PlayerPosition()
		step.XPosition = x
		step.YPosition = 5.0
		step.Stance = 6.6200000047683716
		step.ZPosition = 0.5
		step.OnGround = true
		if !watcher.handleFlying(&step.Packet10Flying) {
			t.Fatalf("step %d handleFlying returned false", i+1)
		}
	}

	foundDestroy := false
	for {
		packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		destroy, ok := packet.(*protocol.Packet29DestroyEntity)
		if !ok {
			continue
		}
		if len(destroy.EntityIDs) == 1 && destroy.EntityIDs[0] == 21 {
			foundDestroy = true
			break
		}
	}
	if !foundDestroy {
		t.Fatal("expected destroy packet for mover entity 21, but none was sent")
	}
}

func TestBuildNamedEntitySpawnPacketIncludesCurrentItemAndFlags(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.entityID = 321
	session.clientUsername = "spawn_test"
	session.heldItemSlot = 0
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     267,
		StackSize:  1,
		ItemDamage: 0,
	}
	session.playerSneaking = true
	session.playerSprinting = true

	pkt := session.buildNamedEntitySpawnPacket()
	if pkt == nil {
		t.Fatal("buildNamedEntitySpawnPacket returned nil")
	}
	if pkt.CurrentItem != 267 {
		t.Fatalf("current item mismatch: got=%d want=267", pkt.CurrentItem)
	}
	if len(pkt.Metadata) != 1 {
		t.Fatalf("metadata count mismatch: got=%d want=1", len(pkt.Metadata))
	}
	flags, ok := pkt.Metadata[0].Value.(int8)
	if !ok {
		t.Fatalf("metadata flags type mismatch: %T", pkt.Metadata[0].Value)
	}
	if (flags&(1<<1)) == 0 || (flags&(1<<3)) == 0 {
		t.Fatalf("expected sneaking/sprinting bits set, flags=%08b", uint8(flags))
	}
}

func TestHandleAnimationSwingBroadcastsToWatchers(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	actor := newInteractionTestSession(srv, io.Discard)
	actor.clientUsername = "actor"
	actor.entityID = 40
	actor.playerRegistered = true

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.clientUsername = "watcher"
	watcher.entityID = 41
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[actor] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{actor, watcher}
	srv.activeMu.Unlock()
	actor.markSeenBy(watcher, true)

	actor.handleAnimation(&protocol.Packet18Animation{
		EntityID:  40,
		AnimateID: 1,
	})

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read animation packet: %v", err)
	}
	anim, ok := packet.(*protocol.Packet18Animation)
	if !ok {
		t.Fatalf("expected Packet18Animation, got %T", packet)
	}
	if anim.EntityID != 40 || anim.AnimateID != 1 {
		t.Fatalf("animation packet mismatch: %#v", anim)
	}
}

func TestHandleEntityActionUpdatesSneakSprintFlags(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)

	session.handleEntityAction(&protocol.Packet19EntityAction{Action: 1})
	if !session.playerSneaking {
		t.Fatal("playerSneaking should be true after action=1")
	}
	session.handleEntityAction(&protocol.Packet19EntityAction{Action: 2})
	if session.playerSneaking {
		t.Fatal("playerSneaking should be false after action=2")
	}
	session.handleEntityAction(&protocol.Packet19EntityAction{Action: 4})
	if !session.playerSprinting {
		t.Fatal("playerSprinting should be true after action=4")
	}
	session.handleEntityAction(&protocol.Packet19EntityAction{Action: 5})
	if session.playerSprinting {
		t.Fatal("playerSprinting should be false after action=5")
	}
}

func TestHandleEntityActionBroadcastsMetadataToWatchers(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	actor := newInteractionTestSession(srv, io.Discard)
	actor.entityID = 44
	actor.playerRegistered = true

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 45
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[actor] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{actor, watcher}
	srv.activeMu.Unlock()
	actor.markSeenBy(watcher, true)

	actor.handleEntityAction(&protocol.Packet19EntityAction{Action: 1})

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read metadata packet: %v", err)
	}
	meta, ok := packet.(*protocol.Packet40EntityMetadata)
	if !ok {
		t.Fatalf("expected Packet40EntityMetadata, got %T", packet)
	}
	if meta.EntityID != 44 {
		t.Fatalf("metadata entity mismatch: got=%d want=44", meta.EntityID)
	}
	if len(meta.Metadata) != 1 {
		t.Fatalf("metadata count mismatch: got=%d want=1", len(meta.Metadata))
	}
	flags, ok := meta.Metadata[0].Value.(int8)
	if !ok {
		t.Fatalf("flags value type mismatch: %T", meta.Metadata[0].Value)
	}
	if (flags & (1 << 1)) == 0 {
		t.Fatalf("expected sneaking bit set, flags=%08b", uint8(flags))
	}
}

func TestHandleUseEntityInvalidSelfAttackKicks(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var buf bytes.Buffer
	session := newInteractionTestSession(srv, &buf)
	session.entityID = 72
	session.clientUsername = "self"
	session.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[session] = "self"
	srv.activeOrder = []*loginSession{session}
	srv.activeMu.Unlock()

	ok := session.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: 72,
		TargetEntityID: 72,
		Action:         1,
	})
	if ok {
		t.Fatal("expected handleUseEntity to return false on invalid self-attack")
	}

	packet, err := protocol.ReadPacket(&buf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read kick packet: %v", err)
	}
	kick, ok := packet.(*protocol.Packet255KickDisconnect)
	if !ok {
		t.Fatalf("expected Packet255KickDisconnect, got %T", packet)
	}
	if !strings.Contains(kick.Reason, "invalid entity") {
		t.Fatalf("unexpected kick reason: %q", kick.Reason)
	}
}

func TestHandleUseEntityAttackDamagesTargetAndSendsHealth(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 90
	attacker.clientUsername = "attacker"
	attacker.playerHealth = 20
	attacker.playerRegistered = true

	var targetBuf bytes.Buffer
	target := newInteractionTestSession(srv, &targetBuf)
	target.entityID = 91
	target.clientUsername = "target"
	target.playerHealth = 20
	target.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{attacker, target}
	srv.activeMu.Unlock()

	ok := attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: target.entityID,
		Action:         1,
	})
	if !ok {
		t.Fatal("handleUseEntity returned false")
	}

	if target.playerHealth != 19 {
		t.Fatalf("target health mismatch: got=%f want=19", target.playerHealth)
	}
	if target.playerDead {
		t.Fatal("target should still be alive")
	}

	packet, err := protocol.ReadPacket(&targetBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read target health packet: %v", err)
	}
	health, ok := packet.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected Packet8UpdateHealth, got %T", packet)
	}
	if health.HealthMP != 19 {
		t.Fatalf("health packet mismatch: got=%f want=19", health.HealthMP)
	}
}

func TestHandleUseEntityAttackDamagesMobTarget(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 390
	attacker.playerRegistered = true
	attacker.playerHealth = 20
	attacker.heldItemSlot = 0

	watcher := newInteractionTestSession(srv, io.Discard)
	watcher.entityID = 391
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{attacker, watcher}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 2.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}

	srv.mobMu.Lock()
	startHealth := mob.Health
	srv.mobMu.Unlock()

	ok := attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: mob.EntityID,
		Action:         1,
	})
	if !ok {
		t.Fatal("handleUseEntity returned false")
	}

	srv.mobMu.Lock()
	updated := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if updated == nil {
		t.Fatal("expected mob to remain alive after single baseline hit")
	}
	if !(updated.Health < startHealth) {
		t.Fatalf("expected mob health to drop: start=%.2f now=%.2f", startHealth, updated.Health)
	}
}

func TestHandleUseEntitySprintAttackKnocksBackMobAndStopsSprinting(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 392
	attacker.playerRegistered = true
	attacker.playerHealth = 20
	attacker.playerSprinting = true
	attacker.playerYaw = 0

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activeOrder = []*loginSession{attacker}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 2.5, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	startZ := mob.Z

	ok := attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: mob.EntityID,
		Action:         1,
	})
	if !ok {
		t.Fatal("handleUseEntity returned false")
	}

	srv.mobMu.Lock()
	updated := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if updated == nil {
		t.Fatal("expected mob alive after single sprint hit")
	}
	if !(updated.Z > startZ) {
		t.Fatalf("expected mob knockback along +Z: startZ=%.3f nowZ=%.3f", startZ, updated.Z)
	}
	if attacker.playerSprinting {
		t.Fatal("expected attacker sprinting reset after knockback attack")
	}
}

func TestHandleUseEntityBlockedBySolidBlockDoesNoDamage(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 393
	attacker.playerHealth = 20
	attacker.playerRegistered = true
	attacker.playerX = 0.5
	attacker.playerY = 5.0
	attacker.playerZ = 0.5

	target := newInteractionTestSession(srv, io.Discard)
	target.entityID = 394
	target.playerHealth = 20
	target.playerRegistered = true
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 3.0

	if !srv.world.setBlock(0, 6, 1, 1, 0) {
		t.Fatal("failed to place LOS blocker")
	}
	if !srv.world.setBlock(0, 6, 2, 1, 0) {
		t.Fatal("failed to place LOS blocker")
	}

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{attacker, target}
	srv.activeMu.Unlock()

	ok := attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: target.entityID,
		Action:         1,
	})
	if !ok {
		t.Fatal("handleUseEntity returned false")
	}
	if target.playerHealth != 20 {
		t.Fatalf("expected no damage through wall: hp=%f", target.playerHealth)
	}
}

func TestHandleUseEntityAttackCriticalDamageWhenFalling(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 92
	attacker.clientUsername = "attacker"
	attacker.playerHealth = 20
	attacker.playerRegistered = true
	attacker.playerOnGround = false
	attacker.playerFallDistance = 0.7

	var targetBuf bytes.Buffer
	target := newInteractionTestSession(srv, &targetBuf)
	target.entityID = 93
	target.clientUsername = "target"
	target.playerHealth = 20
	target.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{attacker, target}
	srv.activeMu.Unlock()

	ok := attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: target.entityID,
		Action:         1,
	})
	if !ok {
		t.Fatal("handleUseEntity returned false")
	}

	if target.playerHealth != 18.5 {
		t.Fatalf("critical target health mismatch: got=%f want=18.5", target.playerHealth)
	}

	packet, err := protocol.ReadPacket(&targetBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read target health packet: %v", err)
	}
	health, ok := packet.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected Packet8UpdateHealth, got %T", packet)
	}
	if health.HealthMP != 18.5 {
		t.Fatalf("critical health packet mismatch: got=%f want=18.5", health.HealthMP)
	}
}

func TestHandleUseEntityAttackRespectsHurtResistantCooldown(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 92
	attacker.playerHealth = 20

	var targetBuf bytes.Buffer
	target := newInteractionTestSession(srv, &targetBuf)
	target.entityID = 93
	target.playerHealth = 20

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{attacker, target}
	srv.activeMu.Unlock()

	if !attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: target.entityID,
		Action:         1,
	}) {
		t.Fatal("first attack returned false")
	}
	if !attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: target.entityID,
		Action:         1,
	}) {
		t.Fatal("second attack returned false")
	}

	if target.playerHealth != 19 {
		t.Fatalf("second hit should be blocked by hurt-resistant cooldown: got=%f want=19", target.playerHealth)
	}

	first, err := protocol.ReadPacket(&targetBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read first health packet: %v", err)
	}
	if _, ok := first.(*protocol.Packet8UpdateHealth); !ok {
		t.Fatalf("expected first packet to be Packet8UpdateHealth, got %T", first)
	}
	second, err := protocol.ReadPacket(&targetBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read first hurt-status packet: %v", err)
	}
	status, ok := second.(*protocol.Packet38EntityStatus)
	if !ok {
		t.Fatalf("expected Packet38EntityStatus, got %T", second)
	}
	if status.EntityStatus != 2 {
		t.Fatalf("unexpected entity status byte: got=%d want=2", status.EntityStatus)
	}
	if _, err := protocol.ReadPacket(&targetBuf, protocol.DirectionClientbound); err == nil {
		t.Fatal("expected no extra packets during same-damage cooldown")
	}
}

func TestApplyIncomingPlayerDamageKillsTarget(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var targetBuf bytes.Buffer
	target := newInteractionTestSession(srv, &targetBuf)
	target.playerHealth = 5

	damaged, died, hurtStatus := target.applyIncomingPlayerDamage(10)
	if !damaged || !died || !hurtStatus {
		t.Fatalf("damage flags mismatch: damaged=%t died=%t hurtStatus=%t", damaged, died, hurtStatus)
	}
	if target.playerHealth != 0 || !target.playerDead {
		t.Fatalf("target death state mismatch: hp=%f dead=%t", target.playerHealth, target.playerDead)
	}

	packet, err := protocol.ReadPacket(&targetBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read death health packet: %v", err)
	}
	health, ok := packet.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected Packet8UpdateHealth, got %T", packet)
	}
	if health.HealthMP != 0 {
		t.Fatalf("expected zero health packet on death, got=%f", health.HealthMP)
	}
}

func TestHandleUseEntityAttackWithSwordUsesWeaponDamageAndDurability(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var attackerBuf bytes.Buffer
	attacker := newInteractionTestSession(srv, &attackerBuf)
	attacker.entityID = 94
	attacker.playerHealth = 20
	attacker.playerRegistered = true
	attacker.heldItemSlot = 0
	attacker.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDIronSword,
		StackSize:  1,
		ItemDamage: 0,
	}

	target := newInteractionTestSession(srv, io.Discard)
	target.entityID = 95
	target.playerHealth = 20
	target.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{attacker, target}
	srv.activeMu.Unlock()

	if !attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: target.entityID,
		Action:         1,
	}) {
		t.Fatal("handleUseEntity returned false")
	}

	if target.playerHealth != 13 {
		t.Fatalf("sword damage mismatch: got=%f want=13", target.playerHealth)
	}
	if attacker.inventory[36] == nil || attacker.inventory[36].ItemDamage != 1 {
		t.Fatalf("expected sword durability +1, got=%#v", attacker.inventory[36])
	}

	var slot *protocol.Packet103SetSlot
	for i := 0; i < 4; i++ {
		packet, err := protocol.ReadPacket(&attackerBuf, protocol.DirectionClientbound)
		if err != nil {
			t.Fatalf("failed to read attacker packet %d: %v", i+1, err)
		}
		if pkt, ok := packet.(*protocol.Packet103SetSlot); ok {
			slot = pkt
			break
		}
	}
	if slot == nil {
		t.Fatal("expected attacker Packet103SetSlot sync but none was found")
	}
	if slot.ItemSlot != 36 || slot.ItemStack == nil || slot.ItemStack.ItemDamage != 1 {
		t.Fatalf("attacker slot sync mismatch: %#v", slot)
	}
}

func TestBlockingSwordReducesIncomingDamage(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 96
	attacker.playerHealth = 20
	attacker.playerRegistered = true
	attacker.heldItemSlot = 0
	attacker.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDIronSword,
		StackSize:  1,
		ItemDamage: 0,
	}

	target := newInteractionTestSession(srv, io.Discard)
	target.entityID = 97
	target.playerHealth = 20
	target.playerRegistered = true
	target.heldItemSlot = 0
	target.playerUsingItem = true
	target.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDIronSword,
		StackSize:  1,
		ItemDamage: 0,
	}

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{attacker, target}
	srv.activeMu.Unlock()

	if !attacker.handleUseEntity(&protocol.Packet7UseEntity{
		PlayerEntityID: attacker.entityID,
		TargetEntityID: target.entityID,
		Action:         1,
	}) {
		t.Fatal("handleUseEntity returned false")
	}

	// Iron sword baseline in this rewrite: 1(base)+6(modifier)=7, blocking => (1+7)/2=4.
	if target.playerHealth != 16 {
		t.Fatalf("blocking damage reduction mismatch: got=%f want=16", target.playerHealth)
	}
}

func TestHandlePlaceDirection255StartsUsingSwordAndBroadcastsMetadata(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	actor := newInteractionTestSession(srv, io.Discard)
	actor.entityID = 98
	actor.playerRegistered = true
	actor.heldItemSlot = 0
	actor.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDIronSword,
		StackSize:  1,
		ItemDamage: 0,
	}

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 99
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[actor] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{actor, watcher}
	srv.activeMu.Unlock()
	actor.markSeenBy(watcher, true)

	if !actor.handlePlace(&protocol.Packet15Place{
		Direction: 255,
		ItemStack: &protocol.ItemStack{
			ItemID:     itemIDIronSword,
			StackSize:  1,
			ItemDamage: 0,
		},
	}) {
		t.Fatal("handlePlace returned false")
	}

	if !actor.playerUsingItem {
		t.Fatal("expected playerUsingItem=true after direction=255 sword use")
	}
	if actor.playerItemUseCount != 72000 {
		t.Fatalf("expected sword use count 72000, got=%d", actor.playerItemUseCount)
	}

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher metadata packet: %v", err)
	}
	meta, ok := packet.(*protocol.Packet40EntityMetadata)
	if !ok {
		t.Fatalf("expected Packet40EntityMetadata, got %T", packet)
	}
	flags, ok := meta.Metadata[0].Value.(int8)
	if !ok {
		t.Fatalf("metadata flags type mismatch: %T", meta.Metadata[0].Value)
	}
	if (flags & (1 << 4)) == 0 {
		t.Fatalf("expected using-item bit set, flags=%08b", uint8(flags))
	}
}

func TestHandlePlaceDirection255BowRequiresArrowInSurvival(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	session := newInteractionTestSession(srv, io.Discard)
	session.gameType = 0
	session.heldItemSlot = 0
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDBow,
		StackSize:  1,
		ItemDamage: 0,
	}

	if !session.handlePlace(&protocol.Packet15Place{
		Direction: 255,
		ItemStack: &protocol.ItemStack{
			ItemID:     itemIDBow,
			StackSize:  1,
			ItemDamage: 0,
		},
	}) {
		t.Fatal("handlePlace returned false")
	}
	if session.playerUsingItem {
		t.Fatal("survival bow without arrows should not start using")
	}

	session.inventory[10] = &protocol.ItemStack{
		ItemID:     itemIDArrow,
		StackSize:  1,
		ItemDamage: 0,
	}
	if !session.handlePlace(&protocol.Packet15Place{
		Direction: 255,
		ItemStack: &protocol.ItemStack{
			ItemID:     itemIDBow,
			StackSize:  1,
			ItemDamage: 0,
		},
	}) {
		t.Fatal("handlePlace returned false with arrows")
	}
	if !session.playerUsingItem || session.playerItemUseCount != 72000 {
		t.Fatalf("expected bow use started with arrows, using=%t count=%d", session.playerUsingItem, session.playerItemUseCount)
	}
}

func TestHandleBlockDigStatus5BowReleaseSpawnsArrowEntity(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	actor := newInteractionTestSession(srv, io.Discard)
	actor.entityID = 150
	actor.playerRegistered = true
	actor.gameType = 0
	actor.heldItemSlot = 0
	actor.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDBow,
		StackSize:  1,
		ItemDamage: 0,
	}
	actor.inventory[10] = &protocol.ItemStack{
		ItemID:     itemIDArrow,
		StackSize:  1,
		ItemDamage: 0,
	}
	actor.playerUsingItem = true
	actor.playerItemUseCount = 71980 // full draw (useTicks=20 -> draw=1.0)

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 151
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[actor] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{actor, watcher}
	srv.activeMu.Unlock()
	actor.markSeenBy(watcher, true)

	if !actor.handleBlockDig(&protocol.Packet14BlockDig{Status: 5}) {
		t.Fatal("handleBlockDig returned false")
	}
	if actor.playerUsingItem {
		t.Fatal("expected playerUsingItem=false after bow release")
	}

	var sawVehicleSpawn bool
	for {
		packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		if spawn, ok := packet.(*protocol.Packet23VehicleSpawn); ok {
			if spawn.Type == entityTypeArrow && spawn.ThrowerEntityID == actor.entityID {
				sawVehicleSpawn = true
				break
			}
		}
	}
	if !sawVehicleSpawn {
		t.Fatal("expected watcher to receive Packet23VehicleSpawn for released arrow")
	}

	srv.projectileMu.Lock()
	projectileCount := len(srv.projectiles)
	srv.projectileMu.Unlock()
	if projectileCount != 1 {
		t.Fatalf("projectile count mismatch: got=%d want=1", projectileCount)
	}
}

func TestTickProjectilesArrowHitDamagesTarget(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 160
	attacker.playerRegistered = true
	attacker.playerX = 0.5
	attacker.playerY = 5.0
	attacker.playerZ = 0.5
	attacker.playerYaw = 0
	attacker.playerPitch = 0

	var targetBuf bytes.Buffer
	target := newInteractionTestSession(srv, &targetBuf)
	target.entityID = 161
	target.playerRegistered = true
	target.playerHealth = 20
	target.playerX = 0.5
	target.playerY = 5.0
	target.playerZ = 3.0

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activePlayers[target] = "target"
	srv.activeOrder = []*loginSession{attacker, target}
	srv.activeMu.Unlock()
	attacker.markSeenBy(target, true)
	target.markSeenBy(attacker, true)

	srv.spawnArrowFromPlayer(attacker, 2.0, false)

	for i := 0; i < 40; i++ {
		srv.TickProjectiles()
		if target.playerHealth < 20 {
			break
		}
	}
	if target.playerHealth >= 20 {
		t.Fatalf("expected target to take arrow damage, health=%f", target.playerHealth)
	}

	var sawHealthPacket bool
	for {
		packet, err := protocol.ReadPacket(&targetBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		if health, ok := packet.(*protocol.Packet8UpdateHealth); ok && health.HealthMP < 20 {
			sawHealthPacket = true
		}
	}
	if !sawHealthPacket {
		t.Fatal("expected target to receive damaged Packet8UpdateHealth")
	}
}

func TestTickProjectilesArrowHitDamagesMob(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attacker := newInteractionTestSession(srv, io.Discard)
	attacker.entityID = 162
	attacker.playerRegistered = true
	attacker.playerX = 0.5
	attacker.playerY = 5.0
	attacker.playerZ = 0.5
	attacker.playerYaw = 0
	attacker.playerPitch = 0

	srv.activeMu.Lock()
	srv.activePlayers[attacker] = "attacker"
	srv.activeOrder = []*loginSession{attacker}
	srv.activeMu.Unlock()

	mob := srv.spawnMob(&spawnListEntry{entityType: entityTypeZombie}, 0.5, 5.0, 3.0, 0)
	if mob == nil {
		t.Fatal("spawnMob returned nil")
	}
	srv.mobMu.Lock()
	startHealth := mob.Health
	srv.mobMu.Unlock()

	srv.spawnArrowFromPlayer(attacker, 2.0, false)
	for i := 0; i < 40; i++ {
		srv.TickProjectiles()
		srv.mobMu.Lock()
		live := srv.mobs[mob.EntityID]
		if live != nil && live.Health < startHealth {
			srv.mobMu.Unlock()
			return
		}
		srv.mobMu.Unlock()
	}

	srv.mobMu.Lock()
	live := srv.mobs[mob.EntityID]
	srv.mobMu.Unlock()
	if live == nil {
		// Also valid: arrow dealt lethal damage.
		return
	}
	t.Fatalf("expected arrow to damage mob: start=%.2f now=%.2f", startHealth, live.Health)
}

func TestTickProjectilesInGroundArrowCanBePickedUpBySurvivalPlayer(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	var pickerBuf bytes.Buffer
	picker := newInteractionTestSession(srv, &pickerBuf)
	picker.entityID = 170
	picker.playerRegistered = true
	picker.playerX = 0.5
	picker.playerY = 5.0
	picker.playerZ = 0.5
	picker.playerHealth = 20
	picker.gameType = 0

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 171
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[picker] = "picker"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{picker, watcher}
	srv.activeMu.Unlock()

	arrow := &trackedProjectile{
		EntityID:      999,
		Type:          entityTypeArrow,
		X:             0.5,
		Y:             5.0,
		Z:             0.5,
		InGround:      true,
		ArrowShake:    0,
		CanBePickedUp: arrowPickupSurvival,
	}
	srv.projectileMu.Lock()
	srv.projectiles[arrow.EntityID] = arrow
	srv.projectileMu.Unlock()

	srv.TickProjectiles()

	srv.projectileMu.Lock()
	_, stillExists := srv.projectiles[arrow.EntityID]
	srv.projectileMu.Unlock()
	if stillExists {
		t.Fatal("expected in-ground arrow to be removed after pickup")
	}
	if picker.inventory[36] == nil || picker.inventory[36].ItemID != itemIDArrow || picker.inventory[36].StackSize != 1 {
		t.Fatalf("picker inventory mismatch after arrow pickup: %#v", picker.inventory[36])
	}

	var sawPickerSetSlot bool
	for {
		packet, err := protocol.ReadPacket(&pickerBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		if setSlot, ok := packet.(*protocol.Packet103SetSlot); ok && setSlot.ItemSlot == 36 && setSlot.ItemStack != nil && setSlot.ItemStack.ItemID == itemIDArrow {
			sawPickerSetSlot = true
		}
	}
	if !sawPickerSetSlot {
		t.Fatal("expected picker to receive Packet103SetSlot for picked arrow")
	}

	var sawDestroy bool
	for {
		packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		if destroy, ok := packet.(*protocol.Packet29DestroyEntity); ok && len(destroy.EntityIDs) == 1 && destroy.EntityIDs[0] == arrow.EntityID {
			sawDestroy = true
		}
	}
	if !sawDestroy {
		t.Fatal("expected watcher to receive Packet29DestroyEntity for picked arrow")
	}
}

func TestTickProjectilesCreativeOnlyArrowNotPickedBySurvivalPlayer(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	picker := newInteractionTestSession(srv, io.Discard)
	picker.entityID = 172
	picker.playerRegistered = true
	picker.playerX = 0.5
	picker.playerY = 5.0
	picker.playerZ = 0.5
	picker.playerHealth = 20
	picker.gameType = 0

	srv.activeMu.Lock()
	srv.activePlayers[picker] = "picker"
	srv.activeOrder = []*loginSession{picker}
	srv.activeMu.Unlock()

	arrow := &trackedProjectile{
		EntityID:      1000,
		Type:          entityTypeArrow,
		X:             0.5,
		Y:             5.0,
		Z:             0.5,
		InGround:      true,
		ArrowShake:    0,
		CanBePickedUp: arrowPickupCreativeOnly,
	}
	srv.projectileMu.Lock()
	srv.projectiles[arrow.EntityID] = arrow
	srv.projectileMu.Unlock()

	srv.TickProjectiles()

	srv.projectileMu.Lock()
	_, stillExists := srv.projectiles[arrow.EntityID]
	srv.projectileMu.Unlock()
	if !stillExists {
		t.Fatal("expected creative-only arrow to remain for survival player")
	}
	if picker.inventory[36] != nil {
		t.Fatalf("survival player should not pick creative-only arrow: %#v", picker.inventory[36])
	}
}

func TestTickProjectilesVisibilitySendsSpawnAndDestroyOnViewChange(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 173
	watcher.playerRegistered = true
	watcher.playerHealth = 20
	watcher.loadedChunks = map[chunk.CoordIntPair]struct{}{
		chunk.NewCoordIntPair(10, 10): {},
	}

	srv.activeMu.Lock()
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{watcher}
	srv.activeMu.Unlock()

	arrow := &trackedProjectile{
		EntityID:      1001,
		Type:          entityTypeArrow,
		X:             0.5,
		Y:             5.0,
		Z:             0.5,
		InGround:      true,
		ArrowShake:    0,
		CanBePickedUp: arrowPickupNone,
	}
	srv.projectileMu.Lock()
	srv.projectiles[arrow.EntityID] = arrow
	srv.projectileMu.Unlock()

	// Out of view: no spawn packet.
	srv.TickProjectiles()
	if _, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound); err == nil {
		t.Fatal("unexpected packet while arrow is out of view")
	}

	// Enter view: should receive spawn.
	watcher.stateMu.Lock()
	watcher.loadedChunks = map[chunk.CoordIntPair]struct{}{
		chunk.NewCoordIntPair(0, 0): {},
	}
	watcher.stateMu.Unlock()
	srv.TickProjectiles()

	first, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected spawn packet after entering view: %v", err)
	}
	if _, ok := first.(*protocol.Packet23VehicleSpawn); !ok {
		t.Fatalf("expected Packet23VehicleSpawn after entering view, got %T", first)
	}

	// Leave view: should receive destroy.
	watcher.stateMu.Lock()
	watcher.loadedChunks = map[chunk.CoordIntPair]struct{}{
		chunk.NewCoordIntPair(10, 10): {},
	}
	watcher.stateMu.Unlock()
	srv.TickProjectiles()

	second, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected destroy packet after leaving view: %v", err)
	}
	destroy, ok := second.(*protocol.Packet29DestroyEntity)
	if !ok {
		t.Fatalf("expected Packet29DestroyEntity after leaving view, got %T", second)
	}
	if len(destroy.EntityIDs) != 1 || destroy.EntityIDs[0] != arrow.EntityID {
		t.Fatalf("destroy packet mismatch: %#v", destroy)
	}
}

func TestHandlePlaceDirection255FoodBroadcastsEatAnimationToWatchers(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	actor := newInteractionTestSession(srv, io.Discard)
	actor.entityID = 210
	actor.playerRegistered = true
	actor.gameType = 0
	actor.heldItemSlot = 0
	actor.playerFood = 10
	actor.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDAppleRed,
		StackSize:  1,
		ItemDamage: 0,
	}

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 211
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[actor] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{actor, watcher}
	srv.activeMu.Unlock()
	actor.markSeenBy(watcher, true)

	if !actor.handlePlace(&protocol.Packet15Place{
		Direction: 255,
		ItemStack: &protocol.ItemStack{
			ItemID:     itemIDAppleRed,
			StackSize:  1,
			ItemDamage: 0,
		},
	}) {
		t.Fatal("handlePlace returned false")
	}

	first, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher first packet: %v", err)
	}
	second, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher second packet: %v", err)
	}

	var sawMeta bool
	var sawEatAnim bool
	for _, packet := range []protocol.Packet{first, second} {
		switch p := packet.(type) {
		case *protocol.Packet40EntityMetadata:
			sawMeta = true
			if len(p.Metadata) == 0 {
				t.Fatalf("metadata packet has no entries: %#v", p)
			}
		case *protocol.Packet18Animation:
			if p.EntityID == actor.entityID && p.AnimateID == 5 {
				sawEatAnim = true
			}
		}
	}
	if !sawMeta {
		t.Fatalf("expected watcher to receive Packet40EntityMetadata and Packet18Animation, got %T and %T", first, second)
	}
	if !sawEatAnim {
		t.Fatalf("expected watcher to receive eat animation id=5, got %T and %T", first, second)
	}
}

func TestTickHeldItemUseConsumesFoodAndUpdatesHealthState(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var selfBuf bytes.Buffer
	session := newInteractionTestSession(srv, &selfBuf)
	session.entityID = 110
	session.playerRegistered = true

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 111
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[session] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{session, watcher}
	srv.activeMu.Unlock()
	session.markSeenBy(watcher, true)

	session.heldItemSlot = 0
	session.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDAppleRed,
		StackSize:  2,
		ItemDamage: 0,
	}
	session.playerFood = 16
	session.playerSat = 2
	session.playerUsingItem = true
	session.playerItemUseCount = 1

	session.tickHeldItemUse()

	if session.playerUsingItem {
		t.Fatal("expected playerUsingItem=false after food finish")
	}
	if session.inventory[36] == nil || session.inventory[36].StackSize != 1 {
		t.Fatalf("food stack consume mismatch: %#v", session.inventory[36])
	}
	if session.playerFood != 20 {
		t.Fatalf("food level mismatch: got=%d want=20", session.playerFood)
	}
	if session.playerSat != 4.4 {
		t.Fatalf("food saturation mismatch: got=%f want=4.4", session.playerSat)
	}

	// Watcher receives metadata update (using-item flag cleared).
	watched, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher packet: %v", err)
	}
	if _, ok := watched.(*protocol.Packet40EntityMetadata); !ok {
		t.Fatalf("expected watcher packet Packet40EntityMetadata, got %T", watched)
	}

	// Self receives slot sync + health update.
	first, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read self first packet: %v", err)
	}
	if status, ok := first.(*protocol.Packet38EntityStatus); !ok {
		t.Fatalf("expected self first packet Packet38EntityStatus, got %T", first)
	} else if status.EntityStatus != 9 {
		t.Fatalf("expected status id 9, got %#v", status)
	}
	second, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read self second packet: %v", err)
	}
	if _, ok := second.(*protocol.Packet103SetSlot); !ok {
		t.Fatalf("expected self second packet Packet103SetSlot, got %T", second)
	}
	third, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read self third packet: %v", err)
	}
	health, ok := third.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected self third packet Packet8UpdateHealth, got %T", third)
	}
	if health.Food != 20 {
		t.Fatalf("health packet food mismatch: %#v", health)
	}
}

func TestTickFoodStatsNaturalRegenerationHealsAndAddsExhaustion(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var selfBuf bytes.Buffer
	session := newInteractionTestSession(srv, &selfBuf)
	session.playerHealth = 18
	session.playerFood = 20
	session.playerSat = 5
	session.playerFoodTimer = 79

	session.tickFoodStats()

	if session.playerHealth != 19 {
		t.Fatalf("health mismatch: got=%f want=19", session.playerHealth)
	}
	if session.playerFoodTimer != 0 {
		t.Fatalf("food timer mismatch: got=%d want=0", session.playerFoodTimer)
	}
	if session.playerFoodExhaust != 3.0 {
		t.Fatalf("food exhaustion mismatch: got=%f want=3.0", session.playerFoodExhaust)
	}

	packet, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected health packet, got read err: %v", err)
	}
	health, ok := packet.(*protocol.Packet8UpdateHealth)
	if !ok {
		t.Fatalf("expected Packet8UpdateHealth, got %T", packet)
	}
	if health.HealthMP != 19 || health.Food != 20 || health.FoodSaturation != 5 {
		t.Fatalf("unexpected health packet payload: %#v", health)
	}
}

func TestTickFoodStatsStarvationDamagesAndBroadcastsStatus(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	var selfBuf bytes.Buffer
	session := newInteractionTestSession(srv, &selfBuf)
	session.entityID = 220
	session.playerRegistered = true
	session.playerHealth = 11
	session.playerFood = 0
	session.playerSat = 0
	session.playerFoodTimer = 79

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 221
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[session] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{session, watcher}
	srv.activeMu.Unlock()
	session.markSeenBy(watcher, true)

	session.tickFoodStats()

	if session.playerHealth != 10 {
		t.Fatalf("health mismatch after starvation: got=%f want=10", session.playerHealth)
	}
	if session.playerDead {
		t.Fatal("expected player to remain alive at 10 health")
	}

	selfFirst, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected self health packet, got read err: %v", err)
	}
	if _, ok := selfFirst.(*protocol.Packet8UpdateHealth); !ok {
		t.Fatalf("expected first self packet Packet8UpdateHealth, got %T", selfFirst)
	}
	selfSecond, err := protocol.ReadPacket(&selfBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected self status packet, got read err: %v", err)
	}
	statusSelf, ok := selfSecond.(*protocol.Packet38EntityStatus)
	if !ok {
		t.Fatalf("expected second self packet Packet38EntityStatus, got %T", selfSecond)
	}
	if statusSelf.EntityID != session.entityID || statusSelf.EntityStatus != 2 {
		t.Fatalf("unexpected self status packet: %#v", statusSelf)
	}

	watchPacket, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("expected watcher status packet, got read err: %v", err)
	}
	statusWatcher, ok := watchPacket.(*protocol.Packet38EntityStatus)
	if !ok {
		t.Fatalf("expected watcher packet Packet38EntityStatus, got %T", watchPacket)
	}
	if statusWatcher.EntityID != session.entityID || statusWatcher.EntityStatus != 2 {
		t.Fatalf("unexpected watcher status packet: %#v", statusWatcher)
	}
}

func TestHandleBlockDigStatus5StopsUsingItemAndBroadcastsMetadata(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	actor := newInteractionTestSession(srv, io.Discard)
	actor.entityID = 100
	actor.playerRegistered = true
	actor.playerUsingItem = true
	actor.heldItemSlot = 0
	actor.inventory[36] = &protocol.ItemStack{
		ItemID:     itemIDIronSword,
		StackSize:  1,
		ItemDamage: 0,
	}

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.entityID = 101
	watcher.playerRegistered = true

	srv.activeMu.Lock()
	srv.activePlayers[actor] = "actor"
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{actor, watcher}
	srv.activeMu.Unlock()
	actor.markSeenBy(watcher, true)

	if !actor.handleBlockDig(&protocol.Packet14BlockDig{Status: 5}) {
		t.Fatal("handleBlockDig returned false")
	}
	if actor.playerUsingItem {
		t.Fatal("expected playerUsingItem=false after status=5")
	}

	packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("failed to read watcher metadata packet: %v", err)
	}
	meta, ok := packet.(*protocol.Packet40EntityMetadata)
	if !ok {
		t.Fatalf("expected Packet40EntityMetadata, got %T", packet)
	}
	flags, ok := meta.Metadata[0].Value.(int8)
	if !ok {
		t.Fatalf("metadata flags type mismatch: %T", meta.Metadata[0].Value)
	}
	if (flags & (1 << 4)) != 0 {
		t.Fatalf("expected using-item bit cleared, flags=%08b", uint8(flags))
	}
}

func TestChunkCoordFromPosUsesJavaFloorSemantics(t *testing.T) {
	cases := []struct {
		pos  float64
		want int32
	}{
		{pos: 0.0, want: 0},
		{pos: 15.999, want: 0},
		{pos: 16.0, want: 1},
		{pos: -0.001, want: -1},
		{pos: -1.0, want: -1},
		{pos: -15.999, want: -1},
		{pos: -16.0, want: -1},
		{pos: -16.001, want: -2},
	}

	for _, tc := range cases {
		got := chunkCoordFromPos(tc.pos)
		if got != tc.want {
			t.Fatalf("chunkCoordFromPos(%f)=%d want=%d", tc.pos, got, tc.want)
		}
	}
}
