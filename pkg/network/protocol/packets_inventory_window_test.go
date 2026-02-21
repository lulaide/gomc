package protocol

import (
	"bytes"
	"testing"
)

func TestPacket103SetSlotRoundTrip(t *testing.T) {
	in := &Packet103SetSlot{
		WindowID: 0,
		ItemSlot: 36,
		ItemStack: &ItemStack{
			ItemID:     1,
			StackSize:  64,
			ItemDamage: 2,
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
	out, ok := packet.(*Packet103SetSlot)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.WindowID != 0 || out.ItemSlot != 36 {
		t.Fatalf("slot header mismatch: %#v", out)
	}
	if out.ItemStack == nil || out.ItemStack.ItemID != 1 || out.ItemStack.StackSize != 64 || out.ItemStack.ItemDamage != 2 {
		t.Fatalf("slot stack mismatch: %#v", out.ItemStack)
	}
}

func TestPacket104WindowItemsRoundTrip(t *testing.T) {
	in := &Packet104WindowItems{
		WindowID: 0,
		ItemStacks: []*ItemStack{
			nil,
			{
				ItemID:     4,
				StackSize:  32,
				ItemDamage: 0,
			},
			{
				ItemID:     98,
				StackSize:  16,
				ItemDamage: 3,
			},
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
	out, ok := packet.(*Packet104WindowItems)
	if !ok {
		t.Fatalf("unexpected packet type: %T", packet)
	}
	if out.WindowID != 0 {
		t.Fatalf("window mismatch: got=%d want=0", out.WindowID)
	}
	if len(out.ItemStacks) != 3 {
		t.Fatalf("stack length mismatch: got=%d want=3", len(out.ItemStacks))
	}
	if out.ItemStacks[0] != nil {
		t.Fatalf("expected nil stack at index 0, got=%#v", out.ItemStacks[0])
	}
	if out.ItemStacks[1] == nil || out.ItemStacks[1].ItemID != 4 || out.ItemStacks[1].StackSize != 32 {
		t.Fatalf("stack[1] mismatch: %#v", out.ItemStacks[1])
	}
	if out.ItemStacks[2] == nil || out.ItemStacks[2].ItemID != 98 || out.ItemStacks[2].ItemDamage != 3 {
		t.Fatalf("stack[2] mismatch: %#v", out.ItemStacks[2])
	}
}
