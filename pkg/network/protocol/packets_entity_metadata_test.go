package protocol

import (
	"bytes"
	"testing"
)

func TestPacket40EntityMetadataRoundTrip(t *testing.T) {
	in := &Packet40EntityMetadata{
		EntityID: 55,
		Metadata: []WatchableObject{
			{ObjectType: 0, DataValueID: 0, Value: int8(0x0A)},
			{ObjectType: 3, DataValueID: 6, Value: float32(19.5)},
		},
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet40EntityMetadata)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.EntityID != 55 {
		t.Fatalf("entity id mismatch: got=%d want=55", out.EntityID)
	}
	if len(out.Metadata) != 2 {
		t.Fatalf("metadata length mismatch: got=%d want=2", len(out.Metadata))
	}
	v0, ok := out.Metadata[0].Value.(int8)
	if !ok || v0 != 0x0A {
		t.Fatalf("metadata[0] mismatch: %#v", out.Metadata[0])
	}
	v1, ok := out.Metadata[1].Value.(float32)
	if !ok || v1 != 19.5 {
		t.Fatalf("metadata[1] mismatch: %#v", out.Metadata[1])
	}
}
