package protocol

import (
	"bytes"
	"testing"
)

func TestPacket201PlayerInfoRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet201PlayerInfo{
		PlayerName:  "Steve",
		IsConnected: true,
		Ping:        42,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet201PlayerInfo)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if *in != *out {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}
