package protocol

import (
	"bytes"
	"testing"
)

func TestPacket14BlockDigRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet14BlockDig{
		Status:    2,
		XPosition: 10,
		YPosition: 64,
		ZPosition: -20,
		Face:      1,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet14BlockDig)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.Status != out.Status || in.XPosition != out.XPosition || in.YPosition != out.YPosition || in.ZPosition != out.ZPosition || in.Face != out.Face {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}

func TestPacket15PlaceRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet15Place{
		XPosition: 15,
		YPosition: 70,
		ZPosition: 3,
		Direction: 5,
		ItemStack: &ItemStack{
			ItemID:     1,
			StackSize:  1,
			ItemDamage: 0,
		},
		XOffset: 0.25,
		YOffset: 0.5,
		ZOffset: 0.75,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet15Place)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.Direction != out.Direction || in.XPosition != out.XPosition || in.YPosition != out.YPosition || in.ZPosition != out.ZPosition {
		t.Fatalf("decoded mismatch: %#v", in)
	}
	if in.ItemStack == nil || in.ItemStack.ItemID != out.ItemStack.ItemID {
		t.Fatalf("item stack mismatch: %#v", in.ItemStack)
	}
}

func TestPacket53BlockChangeRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet53BlockChange{
		XPosition: -2,
		YPosition: 4,
		ZPosition: 8,
		Type:      2,
		Metadata:  1,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet53BlockChange)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if *in != *out {
		t.Fatalf("decoded mismatch: %#v", in)
	}
}
