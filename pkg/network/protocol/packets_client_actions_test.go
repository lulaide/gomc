package protocol

import (
	"bytes"
	"testing"
)

func TestPacket7UseEntityRoundTrip(t *testing.T) {
	in := &Packet7UseEntity{
		PlayerEntityID: 5,
		TargetEntityID: 9,
		Action:         1,
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	packet, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet7UseEntity)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.PlayerEntityID != 5 || out.TargetEntityID != 9 || out.Action != 1 {
		t.Fatalf("packet mismatch: %#v", out)
	}
}

func TestPacket18AnimationRoundTrip(t *testing.T) {
	in := &Packet18Animation{
		EntityID:  17,
		AnimateID: 1,
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	packet, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet18Animation)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.EntityID != 17 || out.AnimateID != 1 {
		t.Fatalf("packet mismatch: %#v", out)
	}
}

func TestPacket19EntityActionRoundTrip(t *testing.T) {
	in := &Packet19EntityAction{
		EntityID: 11,
		Action:   4,
		AuxData:  87,
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	packet, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet19EntityAction)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.EntityID != 11 || out.Action != 4 || out.AuxData != 87 {
		t.Fatalf("packet mismatch: %#v", out)
	}
}

func TestPacket27PlayerInputRoundTrip(t *testing.T) {
	in := &Packet27PlayerInput{
		MoveStrafing: -0.75,
		MoveForward:  1.0,
		Jump:         true,
		Sneak:        false,
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	packet, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet27PlayerInput)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.MoveStrafing != in.MoveStrafing || out.MoveForward != in.MoveForward || out.Jump != in.Jump || out.Sneak != in.Sneak {
		t.Fatalf("packet mismatch: %#v", out)
	}
}
