package protocol

import (
	"bytes"
	"testing"
)

func TestPacket20NamedEntitySpawnRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet20NamedEntitySpawn{
		EntityID:    7,
		Name:        "Alex",
		XPosition:   32,
		YPosition:   160,
		ZPosition:   -96,
		Rotation:    64,
		Pitch:       32,
		CurrentItem: 0,
		Metadata: []WatchableObject{
			{ObjectType: 0, DataValueID: 0, Value: int8(1)},
			{ObjectType: 2, DataValueID: 1, Value: int32(123)},
			{ObjectType: 4, DataValueID: 2, Value: "abc"},
			{ObjectType: 6, DataValueID: 3, Value: [3]int32{1, 2, 3}},
		},
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet20NamedEntitySpawn)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if in.EntityID != out.EntityID || in.Name != out.Name || in.XPosition != out.XPosition || in.YPosition != out.YPosition || in.ZPosition != out.ZPosition {
		t.Fatalf("base fields mismatch: %#v", in)
	}
	if len(in.Metadata) != len(out.Metadata) {
		t.Fatalf("metadata len mismatch: got=%d want=%d", len(in.Metadata), len(out.Metadata))
	}
}

func TestPacket23VehicleSpawnRoundTripWithThrowerVelocity(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet23VehicleSpawn{
		EntityID:        13,
		Type:            60,
		XPosition:       320,
		YPosition:       224,
		ZPosition:       -128,
		Pitch:           12,
		Yaw:             34,
		ThrowerEntityID: 7,
		SpeedX:          1234,
		SpeedY:          -4321,
		SpeedZ:          2222,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet23VehicleSpawn)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if *in != *out {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}

func TestPacket23VehicleSpawnRoundTripWithoutThrowerVelocity(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet23VehicleSpawn{
		EntityID:        14,
		Type:            60,
		XPosition:       10,
		YPosition:       20,
		ZPosition:       30,
		Pitch:           1,
		Yaw:             2,
		ThrowerEntityID: 0,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet23VehicleSpawn)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if *in != *out {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}

func TestPacket28EntityVelocityRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := NewPacket28EntityVelocity(99, 4.5, -5.0, 0.25)
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet28EntityVelocity)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if in.EntityID != 99 {
		t.Fatalf("entity id mismatch: got=%d want=99", in.EntityID)
	}
	// values are clamped to +/-3.9 then encoded at 8000 scale.
	if in.MotionX != int16(3.9*8000) || in.MotionY != int16(-3.9*8000) || in.MotionZ != int16(0.25*8000) {
		t.Fatalf("motion mismatch: %#v", in)
	}
}

func TestPacket29DestroyEntityRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet29DestroyEntity{EntityIDs: []int32{1, 2, 3}}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet29DestroyEntity)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if len(in.EntityIDs) != 3 || in.EntityIDs[2] != 3 {
		t.Fatalf("decoded mismatch: %#v", in.EntityIDs)
	}
}

func TestPacket31RelEntityMoveRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet31RelEntityMove{
		Packet30Entity: Packet30Entity{
			EntityID:  10,
			XPosition: 1,
			YPosition: -2,
			ZPosition: 3,
		},
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet31RelEntityMove)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if in.EntityID != out.EntityID || in.XPosition != out.XPosition || in.YPosition != out.YPosition || in.ZPosition != out.ZPosition {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}

func TestPacket34EntityTeleportRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet34EntityTeleport{
		EntityID:  99,
		XPosition: 100,
		YPosition: 200,
		ZPosition: -300,
		Yaw:       12,
		Pitch:     34,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet34EntityTeleport)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if *in != *out {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}

func TestPacket35EntityHeadRotationRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet35EntityHeadRotation{
		EntityID:        11,
		HeadRotationYaw: 47,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet35EntityHeadRotation)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if *in != *out {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}
