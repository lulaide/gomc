package protocol

import (
	"bytes"
	"testing"
)

func TestPacket22CollectRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet22Collect{
		CollectedEntityID: 123,
		CollectorEntityID: 456,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := packet.(*Packet22Collect)
	if !ok {
		t.Fatalf("type mismatch: %T", packet)
	}
	if *in != *out {
		t.Fatalf("decoded mismatch: got=%#v want=%#v", in, out)
	}
}
