package protocol

import (
	"bytes"
	"testing"
)

func TestPacket101CloseWindowRoundTrip(t *testing.T) {
	in := &Packet101CloseWindow{WindowID: 2}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	packet, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet101CloseWindow)
	if !ok {
		t.Fatalf("unexpected type: %T", packet)
	}
	if out.WindowID != 2 {
		t.Fatalf("window mismatch: got=%d want=2", out.WindowID)
	}
}

func TestPacket102WindowClickRoundTrip(t *testing.T) {
	in := &Packet102WindowClick{
		WindowID:      0,
		InventorySlot: 36,
		MouseClick:    1,
		ActionNumber:  7,
		HoldingShift:  true,
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
	out, ok := packet.(*Packet102WindowClick)
	if !ok {
		t.Fatalf("unexpected type: %T", packet)
	}
	if out.WindowID != 0 || out.InventorySlot != 36 || out.MouseClick != 1 || out.ActionNumber != 7 || !out.HoldingShift {
		t.Fatalf("header mismatch: %#v", out)
	}
	if out.ItemStack == nil || out.ItemStack.ItemID != 1 || out.ItemStack.StackSize != 64 {
		t.Fatalf("stack mismatch: %#v", out.ItemStack)
	}
}

func TestPacket106TransactionRoundTrip(t *testing.T) {
	in := &Packet106Transaction{
		WindowID:     0,
		ActionNumber: 12,
		Accepted:     true,
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, in); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}
	packet, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	out, ok := packet.(*Packet106Transaction)
	if !ok {
		t.Fatalf("unexpected type: %T", packet)
	}
	if out.WindowID != 0 || out.ActionNumber != 12 || !out.Accepted {
		t.Fatalf("transaction mismatch: %#v", out)
	}
}
