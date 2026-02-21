package protocol

import (
	"bytes"
	"testing"
)

func TestPacket250CustomPayloadRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out, err := NewPacket250CustomPayload("MC|Brand", []byte("gomc"))
	if err != nil {
		t.Fatalf("NewPacket250CustomPayload failed: %v", err)
	}

	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet250CustomPayload)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.Channel != "MC|Brand" {
		t.Fatalf("channel mismatch: got=%q", in.Channel)
	}
	if string(in.Data) != "gomc" {
		t.Fatalf("payload mismatch: got=%q", string(in.Data))
	}
}

func TestPacket250RejectsLargePayload(t *testing.T) {
	tooLarge := make([]byte, 32768)
	if _, err := NewPacket250CustomPayload("MC|Test", tooLarge); err == nil {
		t.Fatal("expected payload size validation error")
	}
}
