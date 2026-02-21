package protocol

import (
	"bytes"
	"testing"
)

func TestPacket10FlyingRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet10Flying{OnGround: true}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet10Flying)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if !in.OnGround {
		t.Fatal("expected onGround=true")
	}
	if in.Moving || in.Rotating {
		t.Fatal("base Packet10Flying should not set moving/rotating flags")
	}
}

func TestPacket11PlayerPositionRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := NewPacket11PlayerPosition()
	out.XPosition = 1.25
	out.YPosition = 64
	out.Stance = 65.62
	out.ZPosition = -10.5
	out.OnGround = true

	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet11PlayerPosition)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if !in.Moving {
		t.Fatal("Packet11 must set moving=true")
	}
	if in.XPosition != 1.25 || in.Stance != 65.62 {
		t.Fatalf("position mismatch: %#v", in)
	}
}

func TestPacket12And13FlagsAfterDecode(t *testing.T) {
	{
		var buf bytes.Buffer
		out := NewPacket12PlayerLook()
		out.Yaw = 90
		out.Pitch = 30
		out.OnGround = false
		if err := WritePacket(&buf, out); err != nil {
			t.Fatalf("WritePacket packet12 failed: %v", err)
		}
		decoded, err := ReadPacket(&buf, DirectionServerbound)
		if err != nil {
			t.Fatalf("ReadPacket packet12 failed: %v", err)
		}
		in, ok := decoded.(*Packet12PlayerLook)
		if !ok {
			t.Fatalf("type mismatch packet12: %T", decoded)
		}
		if !in.Rotating || in.Moving {
			t.Fatal("Packet12 should set rotating=true and moving=false")
		}
	}

	{
		var buf bytes.Buffer
		out := NewPacket13PlayerLookMove()
		out.XPosition = 3
		out.YPosition = 70
		out.Stance = 71.62
		out.ZPosition = 2
		out.Yaw = 45
		out.Pitch = 10
		out.OnGround = true
		if err := WritePacket(&buf, out); err != nil {
			t.Fatalf("WritePacket packet13 failed: %v", err)
		}
		decoded, err := ReadPacket(&buf, DirectionServerbound)
		if err != nil {
			t.Fatalf("ReadPacket packet13 failed: %v", err)
		}
		in, ok := decoded.(*Packet13PlayerLookMove)
		if !ok {
			t.Fatalf("type mismatch packet13: %T", decoded)
		}
		if !in.Rotating || !in.Moving {
			t.Fatal("Packet13 should set rotating=true and moving=true")
		}
	}
}

func TestPacket4UpdateTimeConstructorBehavior(t *testing.T) {
	p := NewPacket4UpdateTime(100, 0, false)
	if p.Time != -1 {
		t.Fatalf("expected time=-1 when daylight cycle off and input time 0, got=%d", p.Time)
	}

	p2 := NewPacket4UpdateTime(100, 123, false)
	if p2.Time != -123 {
		t.Fatalf("expected negated time, got=%d", p2.Time)
	}
}

func TestPacket9RespawnFallbackWorldType(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.WriteUint8(9); err != nil {
		t.Fatalf("write id failed: %v", err)
	}
	if err := w.WriteInt32(0); err != nil {
		t.Fatalf("write dim failed: %v", err)
	}
	if err := w.WriteInt8(2); err != nil {
		t.Fatalf("write difficulty failed: %v", err)
	}
	if err := w.WriteInt8(1); err != nil {
		t.Fatalf("write gametype failed: %v", err)
	}
	if err := w.WriteInt16(256); err != nil {
		t.Fatalf("write height failed: %v", err)
	}
	if err := w.WriteString("not-real"); err != nil {
		t.Fatalf("write terrain failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	p, ok := decoded.(*Packet9Respawn)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if p.TerrainType != "default" {
		t.Fatalf("fallback terrain mismatch: got=%q", p.TerrainType)
	}
}

func TestPacket202PlayerAbilitiesBitPacking(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet202PlayerAbilities{
		DisableDamage: true,
		IsFlying:      false,
		AllowFlying:   true,
		IsCreative:    true,
		FlySpeed:      0.05,
		WalkSpeed:     0.1,
	}

	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet202PlayerAbilities)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if !in.DisableDamage || in.IsFlying || !in.AllowFlying || !in.IsCreative {
		t.Fatalf("flags mismatch: %#v", in)
	}
	if in.PacketSize() != 2 {
		t.Fatalf("packet size mismatch: got=%d want=2", in.PacketSize())
	}
}
