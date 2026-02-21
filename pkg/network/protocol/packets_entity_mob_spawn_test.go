package protocol

import (
	"bytes"
	"testing"
)

func TestPacket24MobSpawnRoundTrip(t *testing.T) {
	out := &Packet24MobSpawn{
		EntityID:  42,
		Type:      90,
		XPosition: 320,
		YPosition: 2080,
		ZPosition: -160,
		Yaw:       64,
		Pitch:     -12,
		HeadYaw:   80,
		VelocityX: 120,
		VelocityY: -80,
		VelocityZ: 0,
		Metadata: []WatchableObject{
			{
				ObjectType:  0,
				DataValueID: 0,
				Value:       int8(0),
			},
		},
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	in, ok := packet.(*Packet24MobSpawn)
	if !ok {
		t.Fatalf("type mismatch: got %T", packet)
	}

	if in.EntityID != out.EntityID || in.Type != out.Type {
		t.Fatalf("identity mismatch: got=(%d,%d) want=(%d,%d)", in.EntityID, in.Type, out.EntityID, out.Type)
	}
	if in.XPosition != out.XPosition || in.YPosition != out.YPosition || in.ZPosition != out.ZPosition {
		t.Fatalf("position mismatch: got=(%d,%d,%d) want=(%d,%d,%d)", in.XPosition, in.YPosition, in.ZPosition, out.XPosition, out.YPosition, out.ZPosition)
	}
	if in.Yaw != out.Yaw || in.Pitch != out.Pitch || in.HeadYaw != out.HeadYaw {
		t.Fatalf("rotation mismatch: got=(%d,%d,%d) want=(%d,%d,%d)", in.Yaw, in.Pitch, in.HeadYaw, out.Yaw, out.Pitch, out.HeadYaw)
	}
	if in.VelocityX != out.VelocityX || in.VelocityY != out.VelocityY || in.VelocityZ != out.VelocityZ {
		t.Fatalf("velocity mismatch: got=(%d,%d,%d) want=(%d,%d,%d)", in.VelocityX, in.VelocityY, in.VelocityZ, out.VelocityX, out.VelocityY, out.VelocityZ)
	}
	if len(in.Metadata) != 1 {
		t.Fatalf("metadata len mismatch: got=%d want=1", len(in.Metadata))
	}
}
