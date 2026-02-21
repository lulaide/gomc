package protocol

import (
	"bytes"
	"testing"
)

func TestPacket43ExperienceRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet43Experience{
		Experience:      0.5,
		ExperienceLevel: 3,
		ExperienceTotal: 42,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet43Experience)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if *in != *out {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}
