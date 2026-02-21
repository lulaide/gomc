package protocol

import (
	"bytes"
	"testing"
)

func TestPacket38EntityStatusRoundTrip(t *testing.T) {
	in := &Packet38EntityStatus{
		EntityID:     81,
		EntityStatus: 2,
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet38EntityStatus)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.EntityID != 81 || out.EntityStatus != 2 {
		t.Fatalf("packet mismatch: %#v", out)
	}
}
