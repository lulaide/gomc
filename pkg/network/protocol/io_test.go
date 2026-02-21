package protocol

import (
	"bytes"
	"testing"

	"github.com/lulaide/gomc/pkg/nbt"
)

func TestReadWriteStringUTF16(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	input := "A😀中"
	if err := w.WriteString(input); err != nil {
		t.Fatalf("WriteString failed: %v", err)
	}

	r := NewReader(&buf)
	got, err := r.ReadString(16)
	if err != nil {
		t.Fatalf("ReadString failed: %v", err)
	}
	if got != input {
		t.Fatalf("string mismatch: got=%q want=%q", got, input)
	}
}

func TestReadStringLengthLimit(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.WriteString("abcd"); err != nil {
		t.Fatalf("WriteString failed: %v", err)
	}

	r := NewReader(&buf)
	if _, err := r.ReadString(3); err == nil {
		t.Fatal("expected max length validation error")
	}
}

func TestReadWriteItemStack(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	tag := nbt.NewCompoundTag("")
	tag.SetInteger("DamageBonus", int32(5))
	stack := &ItemStack{
		ItemID:     264,
		StackSize:  3,
		ItemDamage: 0,
		Tag:        tag,
	}

	if err := w.WriteItemStack(stack); err != nil {
		t.Fatalf("WriteItemStack failed: %v", err)
	}

	r := NewReader(&buf)
	got, err := r.ReadItemStack()
	if err != nil {
		t.Fatalf("ReadItemStack failed: %v", err)
	}
	if got == nil {
		t.Fatal("decoded stack is nil")
	}
	if got.ItemID != 264 || got.StackSize != 3 || got.ItemDamage != 0 {
		t.Fatalf("stack fields mismatch: %#v", got)
	}
	if got.Tag == nil || !got.Tag.HasKey("DamageBonus") {
		t.Fatal("decoded stack tag missing DamageBonus")
	}
}

func TestReadWriteNilItemStack(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.WriteItemStack(nil); err != nil {
		t.Fatalf("WriteItemStack(nil) failed: %v", err)
	}

	r := NewReader(&buf)
	got, err := r.ReadItemStack()
	if err != nil {
		t.Fatalf("ReadItemStack failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil stack")
	}
}
