package client

import (
	"bytes"
	"io"
	"testing"

	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

func newUnitTestSession(writer io.Writer) *Session {
	return &Session{
		writer:      writer,
		hasSkyLight: true,
		world:       newWorldCache(),
		entities:    make(map[int32]*trackedEntity),
		playerInfo:  make(map[string]int16),
		events:      make(chan Event, 8),
		done:        make(chan struct{}),
	}
}

func TestHandleKeepAliveWritesResponse(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)

	if err := s.handlePacket(&protocol.Packet0KeepAlive{RandomID: 42}); err != nil {
		t.Fatalf("handlePacket keepalive failed: %v", err)
	}

	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read keepalive response: %v", err)
	}
	reply, ok := packet.(*protocol.Packet0KeepAlive)
	if !ok {
		t.Fatalf("expected Packet0KeepAlive, got %T", packet)
	}
	if reply.RandomID != 42 {
		t.Fatalf("keepalive id mismatch: got=%d want=42", reply.RandomID)
	}
}

func TestWorldCacheApplyMapChunkAndBlockChange(t *testing.T) {
	world := newWorldCache()
	if world.chunkRevision(0, 0) != 0 {
		t.Fatal("expected initial chunk revision to be zero")
	}
	ch := chunk.NewChunk(nil, 0, 0)
	if ok := ch.SetBlockIDWithMetadata(1, 4, 1, 1, 0); !ok {
		t.Fatal("failed to seed source chunk block")
	}

	packet, err := protocol.NewPacket51MapChunk(ch, true, 65535, false)
	if err != nil {
		t.Fatalf("NewPacket51MapChunk failed: %v", err)
	}
	if err := world.applyMapChunk(packet, true); err != nil {
		t.Fatalf("applyMapChunk failed: %v", err)
	}
	afterMapChunkRev := world.chunkRevision(0, 0)
	if afterMapChunkRev == 0 {
		t.Fatal("expected chunk revision to advance after map chunk")
	}

	id, meta, ok := world.blockAt(1, 4, 1)
	if !ok {
		t.Fatal("expected cached block")
	}
	if id != 1 || meta != 0 {
		t.Fatalf("cached block mismatch: got=(%d,%d) want=(1,0)", id, meta)
	}

	world.applyBlockChange(&protocol.Packet53BlockChange{
		XPosition: 1,
		YPosition: 4,
		ZPosition: 1,
		Type:      0,
		Metadata:  0,
	})
	if world.chunkRevision(0, 0) <= afterMapChunkRev {
		t.Fatal("expected chunk revision to advance after block change")
	}

	id, meta, ok = world.blockAt(1, 4, 1)
	if !ok {
		t.Fatal("expected cached block after block change")
	}
	if id != 0 || meta != 0 {
		t.Fatalf("block change mismatch: got=(%d,%d) want=(0,0)", id, meta)
	}
}

func TestWorldCacheBlockChangeAtChunkEdgeBumpsNeighborRevision(t *testing.T) {
	world := newWorldCache()
	world.applyBlockChange(&protocol.Packet53BlockChange{
		XPosition: 0,
		YPosition: 64,
		ZPosition: 0,
		Type:      1,
		Metadata:  0,
	})

	if world.chunkRevision(0, 0) == 0 {
		t.Fatal("expected source chunk revision bump")
	}
	if world.chunkRevision(-1, 0) == 0 {
		t.Fatal("expected west neighbor chunk revision bump")
	}
	if world.chunkRevision(0, -1) == 0 {
		t.Fatal("expected north neighbor chunk revision bump")
	}
}

func TestHandleEntityLifecyclePackets(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet20NamedEntitySpawn{
		EntityID:  5,
		Name:      "other",
		XPosition: 320,
		YPosition: 160,
		ZPosition: 320,
		Rotation:  10,
		Pitch:     20,
	}); err != nil {
		t.Fatalf("spawn handle failed: %v", err)
	}

	move := &protocol.Packet31RelEntityMove{}
	move.EntityID = 5
	move.XPosition = 2
	move.YPosition = -1
	move.ZPosition = 3
	if err := s.handlePacket(move); err != nil {
		t.Fatalf("rel move handle failed: %v", err)
	}

	if err := s.handlePacket(&protocol.Packet35EntityHeadRotation{
		EntityID:        5,
		HeadRotationYaw: 33,
	}); err != nil {
		t.Fatalf("head rotation handle failed: %v", err)
	}

	s.stateMu.RLock()
	ent, ok := s.entities[5]
	s.stateMu.RUnlock()
	if !ok {
		t.Fatal("expected tracked entity")
	}
	if ent.XPosition != 322 || ent.YPosition != 159 || ent.ZPosition != 323 {
		t.Fatalf("entity position mismatch: got=(%d,%d,%d) want=(322,159,323)", ent.XPosition, ent.YPosition, ent.ZPosition)
	}
	if ent.HeadYaw != 33 {
		t.Fatalf("entity head yaw mismatch: got=%d want=33", ent.HeadYaw)
	}

	if err := s.handlePacket(&protocol.Packet29DestroyEntity{
		EntityIDs: []int32{5},
	}); err != nil {
		t.Fatalf("destroy handle failed: %v", err)
	}

	s.stateMu.RLock()
	_, ok = s.entities[5]
	s.stateMu.RUnlock()
	if ok {
		t.Fatal("expected entity to be removed")
	}
}

func TestHandleMobSpawnPacketTracksEntity(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet24MobSpawn{
		EntityID:  42,
		Type:      90,
		XPosition: 64,
		YPosition: 160,
		ZPosition: -32,
		Yaw:       12,
		Pitch:     -4,
		HeadYaw:   15,
		VelocityX: 80,
		VelocityY: 0,
		VelocityZ: -160,
		Metadata: []protocol.WatchableObject{
			{
				ObjectType:  0,
				DataValueID: 0,
				Value:       int8(0),
			},
		},
	}); err != nil {
		t.Fatalf("mob spawn handle failed: %v", err)
	}

	s.stateMu.RLock()
	ent, ok := s.entities[42]
	s.stateMu.RUnlock()
	if !ok {
		t.Fatal("expected tracked mob entity")
	}
	if ent.Type != 90 {
		t.Fatalf("mob type mismatch: got=%d want=90", ent.Type)
	}
	if ent.Name != "mob:90" {
		t.Fatalf("mob name mismatch: got=%q want=%q", ent.Name, "mob:90")
	}
	if ent.XPosition != 64 || ent.YPosition != 160 || ent.ZPosition != -32 {
		t.Fatalf("mob position mismatch: got=(%d,%d,%d)", ent.XPosition, ent.YPosition, ent.ZPosition)
	}
}

func TestHandleMobSpawnPacketReadsSheepMetadata(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet24MobSpawn{
		EntityID:  77,
		Type:      91,
		XPosition: 0,
		YPosition: 0,
		ZPosition: 0,
		Metadata: []protocol.WatchableObject{
			{
				ObjectType:  0,
				DataValueID: 16,
				Value:       int8(0x15), // color=5 + sheared
			},
		},
	}); err != nil {
		t.Fatalf("mob spawn handle failed: %v", err)
	}

	out := s.EntitiesSnapshot()
	if len(out) != 1 {
		t.Fatalf("expected 1 entity snapshot, got %d", len(out))
	}
	if out[0].Type != 91 {
		t.Fatalf("entity type mismatch: got=%d want=91", out[0].Type)
	}
	if out[0].SheepColor != 5 || !out[0].SheepSheared {
		t.Fatalf("sheep metadata mismatch: %#v", out[0])
	}
}

func TestHandleVehicleSpawnAndVelocityPackets(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet23VehicleSpawn{
		EntityID:        23,
		Type:            60,
		XPosition:       320,
		YPosition:       160,
		ZPosition:       -64,
		Yaw:             15,
		Pitch:           -5,
		ThrowerEntityID: 1,
		SpeedX:          800,
		SpeedY:          -400,
		SpeedZ:          0,
	}); err != nil {
		t.Fatalf("spawn handle failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet28EntityVelocity{
		EntityID: 23,
		MotionX:  1600,
		MotionY:  -2000,
		MotionZ:  3200,
	}); err != nil {
		t.Fatalf("velocity handle failed: %v", err)
	}

	s.stateMu.RLock()
	ent, ok := s.entities[23]
	s.stateMu.RUnlock()
	if !ok {
		t.Fatal("expected tracked object entity")
	}
	if ent.Type != 60 {
		t.Fatalf("entity type mismatch: got=%d want=60", ent.Type)
	}
	if ent.Name != "obj:60" {
		t.Fatalf("entity name mismatch: got=%q want=%q", ent.Name, "obj:60")
	}
	if ent.MotionX != 0.2 || ent.MotionY != -0.25 || ent.MotionZ != 0.4 {
		t.Fatalf("velocity mismatch: got=(%f,%f,%f)", ent.MotionX, ent.MotionY, ent.MotionZ)
	}

	out := s.EntitiesSnapshot()
	if len(out) != 1 {
		t.Fatalf("expected 1 entity snapshot, got %d", len(out))
	}
	if out[0].Type != 60 || out[0].X != 10.0 || out[0].Y != 5.0 || out[0].Z != -2.0 {
		t.Fatalf("snapshot mismatch: %#v", out[0])
	}
}

func TestHandlePacket40EntityMetadataUpdatesDroppedItemFields(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet23VehicleSpawn{
		EntityID:        99,
		Type:            2,
		XPosition:       0,
		YPosition:       0,
		ZPosition:       0,
		ThrowerEntityID: 1,
	}); err != nil {
		t.Fatalf("spawn handle failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet40EntityMetadata{
		EntityID: 99,
		Metadata: []protocol.WatchableObject{
			{
				ObjectType:  5,
				DataValueID: 10,
				Value: &protocol.ItemStack{
					ItemID:     4,
					StackSize:  3,
					ItemDamage: 2,
				},
			},
		},
	}); err != nil {
		t.Fatalf("metadata handle failed: %v", err)
	}

	out := s.EntitiesSnapshot()
	if len(out) != 1 {
		t.Fatalf("expected 1 entity snapshot, got %d", len(out))
	}
	if out[0].Type != 2 {
		t.Fatalf("entity type mismatch: got=%d want=2", out[0].Type)
	}
	if out[0].DroppedItemID != 4 || out[0].DroppedItemCount != 3 || out[0].DroppedItemDamage != 2 {
		t.Fatalf("dropped item metadata mismatch: %#v", out[0])
	}
}

func TestHandlePacket22CollectRemovesEntityAndEmitsSoundEvent(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet23VehicleSpawn{
		EntityID:        123,
		Type:            2,
		XPosition:       0,
		YPosition:       0,
		ZPosition:       0,
		ThrowerEntityID: 1,
	}); err != nil {
		t.Fatalf("spawn handle failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet22Collect{
		CollectedEntityID: 123,
		CollectorEntityID: 1,
	}); err != nil {
		t.Fatalf("collect handle failed: %v", err)
	}

	s.stateMu.RLock()
	_, ok := s.entities[123]
	s.stateMu.RUnlock()
	if ok {
		t.Fatal("expected collected entity to be removed")
	}

	select {
	case ev := <-s.events:
		if ev.Type != EventSound {
			t.Fatalf("event type mismatch: got=%s want=%s", ev.Type, EventSound)
		}
		if ev.SoundName != "random.pop" {
			t.Fatalf("sound mismatch: got=%q want=%q", ev.SoundName, "random.pop")
		}
	default:
		t.Fatal("expected collect sound event")
	}
}

func TestDecodeChunkPacketDataShortBuffer(t *testing.T) {
	_, err := decodeChunkPacketData(0, 0, 1, 0, []byte{1, 2, 3}, true, true)
	if err == nil {
		t.Fatal("expected decodeChunkPacketData to fail on short buffer")
	}
}

func TestHandleInventoryPacketsAndSnapshotHeldItem(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet16BlockItemSwitch{ID: 2}); err != nil {
		t.Fatalf("handle Packet16 failed: %v", err)
	}
	windowItems := make([]*protocol.ItemStack, 39)
	windowItems[38] = &protocol.ItemStack{ItemID: 1, StackSize: 10, ItemDamage: 0}
	if err := s.handlePacket(&protocol.Packet104WindowItems{
		WindowID:   0,
		ItemStacks: windowItems,
	}); err != nil {
		t.Fatalf("handle Packet104 failed: %v", err)
	}

	snap := s.Snapshot()
	if snap.HeldSlot != 2 {
		t.Fatalf("held slot mismatch: got=%d want=2", snap.HeldSlot)
	}
	if snap.HeldItemID != 1 || snap.HeldCount != 10 {
		t.Fatalf("held item mismatch: id=%d count=%d", snap.HeldItemID, snap.HeldCount)
	}

	if err := s.handlePacket(&protocol.Packet103SetSlot{
		WindowID: 0,
		ItemSlot: 38,
		ItemStack: &protocol.ItemStack{
			ItemID:     4,
			StackSize:  7,
			ItemDamage: 0,
		},
	}); err != nil {
		t.Fatalf("handle Packet103 failed: %v", err)
	}
	snap = s.Snapshot()
	if snap.HeldItemID != 4 || snap.HeldCount != 7 {
		t.Fatalf("held item slot update mismatch: id=%d count=%d", snap.HeldItemID, snap.HeldCount)
	}
}

func TestSelectHotbarWritesPacket16(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)

	if err := s.SelectHotbar(4); err != nil {
		t.Fatalf("SelectHotbar failed: %v", err)
	}

	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet16: %v", err)
	}
	switchPkt, ok := packet.(*protocol.Packet16BlockItemSwitch)
	if !ok {
		t.Fatalf("expected Packet16BlockItemSwitch, got %T", packet)
	}
	if switchPkt.ID != 4 {
		t.Fatalf("slot mismatch: got=%d want=4", switchPkt.ID)
	}
}

func TestDropHeldItemWritesPacket14Status(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)

	if err := s.DropHeldItem(false); err != nil {
		t.Fatalf("DropHeldItem(single) failed: %v", err)
	}
	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet14(single): %v", err)
	}
	dig, ok := packet.(*protocol.Packet14BlockDig)
	if !ok {
		t.Fatalf("expected Packet14BlockDig, got %T", packet)
	}
	if dig.Status != 4 {
		t.Fatalf("drop single status mismatch: got=%d want=4", dig.Status)
	}

	if err := s.DropHeldItem(true); err != nil {
		t.Fatalf("DropHeldItem(full) failed: %v", err)
	}
	packet, err = protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet14(full): %v", err)
	}
	dig, ok = packet.(*protocol.Packet14BlockDig)
	if !ok {
		t.Fatalf("expected Packet14BlockDig, got %T", packet)
	}
	if dig.Status != 3 {
		t.Fatalf("drop full status mismatch: got=%d want=3", dig.Status)
	}
}

func TestSetCreativeHotbarSlotWritesPacket107(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)

	if err := s.SetCreativeHotbarSlot(2, 1, 5, 1); err != nil {
		t.Fatalf("SetCreativeHotbarSlot failed: %v", err)
	}
	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet107: %v", err)
	}
	setSlot, ok := packet.(*protocol.Packet107CreativeSetSlot)
	if !ok {
		t.Fatalf("expected Packet107CreativeSetSlot, got %T", packet)
	}
	if setSlot.Slot != 38 {
		t.Fatalf("creative slot mismatch: got=%d want=38", setSlot.Slot)
	}
	if setSlot.ItemStack == nil || setSlot.ItemStack.ItemID != 1 || setSlot.ItemStack.ItemDamage != 5 || setSlot.ItemStack.StackSize != 1 {
		t.Fatalf("creative stack mismatch: %#v", setSlot.ItemStack)
	}
}

func TestClickWindowSlotWritesPacket102(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)
	s.inventory[36] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  5,
		ItemDamage: 0,
	}

	if err := s.ClickWindowSlot(36, false, false); err != nil {
		t.Fatalf("ClickWindowSlot failed: %v", err)
	}

	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet102: %v", err)
	}
	click, ok := packet.(*protocol.Packet102WindowClick)
	if !ok {
		t.Fatalf("expected Packet102WindowClick, got %T", packet)
	}
	if click.WindowID != 0 || click.InventorySlot != 36 || click.MouseClick != 0 || click.ActionNumber != 1 || click.HoldingShift {
		t.Fatalf("click header mismatch: %#v", click)
	}
	if click.ItemStack == nil || click.ItemStack.ItemID != 1 || click.ItemStack.StackSize != 5 {
		t.Fatalf("click stack mismatch: %#v", click.ItemStack)
	}
}

func TestHandleAbilitiesPacketUpdatesSnapshot(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet1Login{
		ClientEntityID: 1,
		GameType:       1,
	}); err != nil {
		t.Fatalf("handle Packet1Login failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet202PlayerAbilities{
		DisableDamage: true,
		IsFlying:      false,
		AllowFlying:   true,
		IsCreative:    true,
		FlySpeed:      0.05,
		WalkSpeed:     0.1,
	}); err != nil {
		t.Fatalf("handle Packet202 failed: %v", err)
	}

	snap := s.Snapshot()
	if snap.GameType != 1 || !snap.IsCreative || !snap.CanFly || !snap.Invulnerable {
		t.Fatalf("snapshot abilities mismatch: %#v", snap)
	}
}

func TestUseEntityWritesPacket7(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)
	s.entityID = 9

	if err := s.UseEntity(42, true); err != nil {
		t.Fatalf("UseEntity failed: %v", err)
	}

	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet7: %v", err)
	}
	use, ok := packet.(*protocol.Packet7UseEntity)
	if !ok {
		t.Fatalf("expected Packet7UseEntity, got %T", packet)
	}
	if use.PlayerEntityID != 9 || use.TargetEntityID != 42 || use.Action != 1 {
		t.Fatalf("packet mismatch: %#v", use)
	}
}

func TestSwingArmWritesPacket18(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)
	s.entityID = 33

	if err := s.SwingArm(); err != nil {
		t.Fatalf("SwingArm failed: %v", err)
	}

	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet18: %v", err)
	}
	anim, ok := packet.(*protocol.Packet18Animation)
	if !ok {
		t.Fatalf("expected Packet18Animation, got %T", packet)
	}
	if anim.EntityID != 33 || anim.AnimateID != 1 {
		t.Fatalf("packet mismatch: %#v", anim)
	}
}

func TestSetSneakingWritesPacket19(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)
	s.entityID = 88

	if err := s.SetSneaking(true); err != nil {
		t.Fatalf("SetSneaking failed: %v", err)
	}

	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet19: %v", err)
	}
	action, ok := packet.(*protocol.Packet19EntityAction)
	if !ok {
		t.Fatalf("expected Packet19EntityAction, got %T", packet)
	}
	if action.EntityID != 88 || action.Action != 1 || action.AuxData != 0 {
		t.Fatalf("packet mismatch: %#v", action)
	}
}

func TestSetFlyingWritesPacket202(t *testing.T) {
	var out bytes.Buffer
	s := newUnitTestSession(&out)
	s.canFly = true
	s.isCreative = true
	s.invulnerable = true

	if err := s.SetFlying(true); err != nil {
		t.Fatalf("SetFlying failed: %v", err)
	}

	packet, err := protocol.ReadPacket(&out, protocol.DirectionServerbound)
	if err != nil {
		t.Fatalf("failed to read Packet202: %v", err)
	}
	abilities, ok := packet.(*protocol.Packet202PlayerAbilities)
	if !ok {
		t.Fatalf("expected Packet202PlayerAbilities, got %T", packet)
	}
	if !abilities.IsFlying || !abilities.AllowFlying || !abilities.IsCreative || !abilities.DisableDamage {
		t.Fatalf("ability packet mismatch: %#v", abilities)
	}
}

func TestEntitiesSnapshotContainsTrackedEntities(t *testing.T) {
	s := newUnitTestSession(io.Discard)
	s.entities[3] = &trackedEntity{
		EntityID:  3,
		Name:      "Alex",
		XPosition: 64,
		YPosition: 160,
		ZPosition: -32,
		Yaw:       10,
		Pitch:     5,
		HeadYaw:   12,
		Sneaking:  true,
		Sprinting: false,
		UsingItem: true,
	}

	out := s.EntitiesSnapshot()
	if len(out) != 1 {
		t.Fatalf("expected 1 entity snapshot, got %d", len(out))
	}
	if out[0].EntityID != 3 || out[0].Name != "Alex" {
		t.Fatalf("snapshot identity mismatch: %#v", out[0])
	}
	if out[0].X != 2.0 || out[0].Y != 5.0 || out[0].Z != -1.0 {
		t.Fatalf("snapshot position mismatch: %#v", out[0])
	}
	if !out[0].Sneaking || out[0].Sprinting || !out[0].UsingItem {
		t.Fatalf("snapshot movement-state mismatch: %#v", out[0])
	}
}

func TestPlayerListSnapshotTracksPacket201(t *testing.T) {
	s := newUnitTestSession(io.Discard)
	s.username = "Steve"

	if err := s.handlePacket(&protocol.Packet201PlayerInfo{
		PlayerName:  "Alex",
		IsConnected: true,
		Ping:        42,
	}); err != nil {
		t.Fatalf("handle Packet201 connect failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet201PlayerInfo{
		PlayerName:  "Alex",
		IsConnected: false,
		Ping:        0,
	}); err != nil {
		t.Fatalf("handle Packet201 disconnect failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet201PlayerInfo{
		PlayerName:  "Bob",
		IsConnected: true,
		Ping:        7,
	}); err != nil {
		t.Fatalf("handle Packet201 connect Bob failed: %v", err)
	}

	list := s.PlayerListSnapshot()
	if len(list) != 2 {
		t.Fatalf("player list size mismatch: got=%d want=2", len(list))
	}
	if list[0].Name != "Bob" || list[0].Ping != 7 {
		t.Fatalf("player list first mismatch: %#v", list[0])
	}
	if list[1].Name != "Steve" {
		t.Fatalf("self player should be present: %#v", list)
	}
}

func TestHandlePacket40EntityMetadataUpdatesTrackedFlags(t *testing.T) {
	s := newUnitTestSession(io.Discard)
	s.entities[8] = &trackedEntity{EntityID: 8, Name: "other"}

	if err := s.handlePacket(&protocol.Packet40EntityMetadata{
		EntityID: 8,
		Metadata: []protocol.WatchableObject{
			{ObjectType: 0, DataValueID: 0, Value: int8((1 << 1) | (1 << 3) | (1 << 4))},
		},
	}); err != nil {
		t.Fatalf("handle Packet40 failed: %v", err)
	}

	out := s.EntitiesSnapshot()
	if len(out) != 1 {
		t.Fatalf("expected 1 entity snapshot, got %d", len(out))
	}
	if !out[0].Sneaking || !out[0].Sprinting || !out[0].UsingItem {
		t.Fatalf("entity flags mismatch: %#v", out[0])
	}
}

func TestHandlePacket40EntityMetadataUpdatesMobSpecificFields(t *testing.T) {
	s := newUnitTestSession(io.Discard)
	s.entities[50] = &trackedEntity{EntityID: 50, Type: 50, Name: "mob:50"}
	s.entities[51] = &trackedEntity{EntityID: 51, Type: 55, Name: "mob:55"}
	s.entities[52] = &trackedEntity{EntityID: 52, Type: 91, Name: "mob:91"}
	s.entities[53] = &trackedEntity{EntityID: 53, Type: 52, Name: "mob:52"}
	s.entities[54] = &trackedEntity{EntityID: 54, Type: 51, Name: "mob:51"}
	s.entities[55] = &trackedEntity{EntityID: 55, Type: 54, Name: "mob:54"}

	if err := s.handlePacket(&protocol.Packet40EntityMetadata{
		EntityID: 50,
		Metadata: []protocol.WatchableObject{
			{ObjectType: 0, DataValueID: 16, Value: int8(1)},
			{ObjectType: 0, DataValueID: 17, Value: int8(1)},
		},
	}); err != nil {
		t.Fatalf("handle creeper metadata failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet40EntityMetadata{
		EntityID: 51,
		Metadata: []protocol.WatchableObject{
			{ObjectType: 0, DataValueID: 16, Value: int8(4)},
		},
	}); err != nil {
		t.Fatalf("handle slime metadata failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet40EntityMetadata{
		EntityID: 52,
		Metadata: []protocol.WatchableObject{
			{ObjectType: 0, DataValueID: 16, Value: int8(0x1A)}, // color=10 + sheared
		},
	}); err != nil {
		t.Fatalf("handle sheep metadata failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet40EntityMetadata{
		EntityID: 53,
		Metadata: []protocol.WatchableObject{
			{ObjectType: 0, DataValueID: 16, Value: int8(1)},
		},
	}); err != nil {
		t.Fatalf("handle spider metadata failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet40EntityMetadata{
		EntityID: 54,
		Metadata: []protocol.WatchableObject{
			{ObjectType: 0, DataValueID: 13, Value: int8(1)},
		},
	}); err != nil {
		t.Fatalf("handle skeleton metadata failed: %v", err)
	}
	if err := s.handlePacket(&protocol.Packet40EntityMetadata{
		EntityID: 55,
		Metadata: []protocol.WatchableObject{
			{ObjectType: 0, DataValueID: 12, Value: int8(1)},
			{ObjectType: 0, DataValueID: 13, Value: int8(1)},
		},
	}); err != nil {
		t.Fatalf("handle zombie metadata failed: %v", err)
	}

	s.stateMu.RLock()
	creeper := s.entities[50]
	slime := s.entities[51]
	sheep := s.entities[52]
	spider := s.entities[53]
	skeleton := s.entities[54]
	zombie := s.entities[55]
	s.stateMu.RUnlock()

	if creeper.CreeperState != 1 || !creeper.CreeperPowered {
		t.Fatalf("creeper metadata mismatch: %#v", creeper)
	}
	if slime.SlimeSize != 4 {
		t.Fatalf("slime metadata mismatch: %#v", slime)
	}
	if sheep.SheepColor != 10 || !sheep.SheepSheared {
		t.Fatalf("sheep metadata mismatch: %#v", sheep)
	}
	if !spider.SpiderClimbing {
		t.Fatalf("spider metadata mismatch: %#v", spider)
	}
	if skeleton.SkeletonType != 1 {
		t.Fatalf("skeleton metadata mismatch: %#v", skeleton)
	}
	if !zombie.ZombieChild || !zombie.ZombieVillager {
		t.Fatalf("zombie metadata mismatch: %#v", zombie)
	}
}

func TestHandlePacket20SpawnReadsMetadataFlags(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet20NamedEntitySpawn{
		EntityID:  12,
		Name:      "spawned",
		XPosition: 0,
		YPosition: 0,
		ZPosition: 0,
		Metadata: []protocol.WatchableObject{
			{ObjectType: 0, DataValueID: 0, Value: int8((1 << 1) | (1 << 3) | (1 << 4))},
		},
	}); err != nil {
		t.Fatalf("handle Packet20 failed: %v", err)
	}

	out := s.EntitiesSnapshot()
	if len(out) != 1 {
		t.Fatalf("expected 1 entity snapshot, got %d", len(out))
	}
	if !out[0].Sneaking || !out[0].Sprinting || !out[0].UsingItem {
		t.Fatalf("spawn metadata flags mismatch: %#v", out[0])
	}
}

func TestHandlePacket38EntityStatusEmitsEvent(t *testing.T) {
	s := newUnitTestSession(io.Discard)

	if err := s.handlePacket(&protocol.Packet38EntityStatus{
		EntityID:     77,
		EntityStatus: 2,
	}); err != nil {
		t.Fatalf("handle Packet38 failed: %v", err)
	}

	select {
	case ev := <-s.events:
		if ev.Type != EventSystem {
			t.Fatalf("event type mismatch: got=%s want=%s", ev.Type, EventSystem)
		}
		if ev.Message == "" {
			t.Fatal("expected non-empty event message")
		}
	default:
		t.Fatal("expected entity status event")
	}
}

func TestHandlePacket18AnimationEmitsEvent(t *testing.T) {
	s := newUnitTestSession(io.Discard)
	s.entities[77] = &trackedEntity{EntityID: 77, Name: "other"}

	if err := s.handlePacket(&protocol.Packet18Animation{
		EntityID:  77,
		AnimateID: 5,
	}); err != nil {
		t.Fatalf("handle Packet18 failed: %v", err)
	}

	select {
	case ev := <-s.events:
		if ev.Type != EventSystem {
			t.Fatalf("event type mismatch: got=%s want=%s", ev.Type, EventSystem)
		}
		if ev.Message == "" {
			t.Fatal("expected non-empty event message")
		}
	default:
		t.Fatal("expected animation event")
	}

	if err := s.handlePacket(&protocol.Packet18Animation{
		EntityID:  77,
		AnimateID: 1,
	}); err != nil {
		t.Fatalf("handle Packet18 swing failed: %v", err)
	}
	out := s.EntitiesSnapshot()
	if len(out) != 1 {
		t.Fatalf("expected 1 entity snapshot, got %d", len(out))
	}
	if out[0].SwingProgress <= 0 {
		t.Fatalf("expected positive swing progress, got %.3f", out[0].SwingProgress)
	}
}
