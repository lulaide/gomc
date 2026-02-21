package protocol

import (
	"bytes"
	"testing"
)

func TestPacket107CreativeSetSlotRoundTrip(t *testing.T) {
	in := &Packet107CreativeSetSlot{
		Slot: 38,
		ItemStack: &ItemStack{
			ItemID:     1,
			StackSize:  64,
			ItemDamage: 0,
		},
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	packet, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet107CreativeSetSlot)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.Slot != 38 {
		t.Fatalf("slot mismatch: got=%d want=38", out.Slot)
	}
	if out.ItemStack == nil || out.ItemStack.ItemID != 1 || out.ItemStack.StackSize != 64 {
		t.Fatalf("stack mismatch: %#v", out.ItemStack)
	}
}
