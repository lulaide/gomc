package nbt

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestNBT_ReadWriteRoundTrip(t *testing.T) {
	root := NewCompoundTag("Level")
	root.SetInteger("Health", 20)
	root.SetLong("Time", 123456789)
	root.SetFloat("Saturation", 5.0)
	root.SetBoolean("Hardcore", false)
	root.SetString("PlayerName", "A\x00B𐐷")
	root.SetByteArray("Colors", []byte{1, 2, 3, 4})
	root.SetIntArray("Heights", []int32{10, 64, 255})

	pos := NewListTag("Pos")
	pos.AppendTag(NewDoubleTag("", 1.25))
	pos.AppendTag(NewDoubleTag("", 65.0))
	pos.AppendTag(NewDoubleTag("", -10.5))
	root.SetTag("Pos", pos)

	inventory := NewCompoundTag("Inventory")
	inventory.SetInteger("SelectedSlot", 2)
	root.SetCompoundTag("Inventory", inventory)

	var buf bytes.Buffer
	if err := Write(root, &buf); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	loaded, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if !equalTags(root, loaded) {
		t.Fatalf("roundtrip mismatch\nwant=%s\ngot=%s", root, loaded)
	}
}

func TestNBT_CompressDecompressRoundTrip(t *testing.T) {
	root := NewCompoundTag("Root")
	root.SetString("Msg", "hello")
	root.SetInteger("X", 42)

	compressed, err := Compress(root)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	loaded, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !equalTags(root, loaded) {
		t.Fatalf("compressed roundtrip mismatch")
	}
}

func TestNBT_SafeWriteAndReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "level.dat")

	root := NewCompoundTag("Level")
	root.SetLong("Seed", 12345)

	if err := SafeWriteFile(root, path); err != nil {
		t.Fatalf("SafeWriteFile failed: %v", err)
	}

	loaded, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if loaded == nil {
		t.Fatalf("ReadFile returned nil root")
	}

	if !equalTags(root, loaded) {
		t.Fatalf("safe write/read mismatch")
	}
}

func TestNBT_EmptyListDefaultType(t *testing.T) {
	root := NewCompoundTag("Root")
	list := NewListTag("Empty")
	root.SetTag("Empty", list)

	var buf bytes.Buffer
	if err := Write(root, &buf); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	loaded, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	tag := loaded.GetTag("Empty")
	lt, ok := tag.(*ListTag)
	if !ok {
		t.Fatalf("expected list tag, got %T", tag)
	}
	if lt.TagType != TagByteID {
		t.Fatalf("empty list tagType mismatch: got=%d want=%d", lt.TagType, TagByteID)
	}
}

func equalTags(a, b Tag) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.ID() != b.ID() || a.Name() != b.Name() {
		return false
	}

	switch ta := a.(type) {
	case *EndTag:
		_, ok := b.(*EndTag)
		return ok
	case *ByteTag:
		tb, ok := b.(*ByteTag)
		return ok && ta.Data == tb.Data
	case *ShortTag:
		tb, ok := b.(*ShortTag)
		return ok && ta.Data == tb.Data
	case *IntTag:
		tb, ok := b.(*IntTag)
		return ok && ta.Data == tb.Data
	case *LongTag:
		tb, ok := b.(*LongTag)
		return ok && ta.Data == tb.Data
	case *FloatTag:
		tb, ok := b.(*FloatTag)
		return ok && ta.Data == tb.Data
	case *DoubleTag:
		tb, ok := b.(*DoubleTag)
		return ok && ta.Data == tb.Data
	case *StringTag:
		tb, ok := b.(*StringTag)
		return ok && ta.Data == tb.Data
	case *ByteArrayTag:
		tb, ok := b.(*ByteArrayTag)
		if !ok || len(ta.Bytes) != len(tb.Bytes) {
			return false
		}
		for i := range ta.Bytes {
			if ta.Bytes[i] != tb.Bytes[i] {
				return false
			}
		}
		return true
	case *IntArrayTag:
		tb, ok := b.(*IntArrayTag)
		if !ok || len(ta.Ints) != len(tb.Ints) {
			return false
		}
		for i := range ta.Ints {
			if ta.Ints[i] != tb.Ints[i] {
				return false
			}
		}
		return true
	case *ListTag:
		tb, ok := b.(*ListTag)
		if !ok || ta.TagType != tb.TagType || len(ta.TagList) != len(tb.TagList) {
			return false
		}
		for i := range ta.TagList {
			if !equalTags(ta.TagList[i], tb.TagList[i]) {
				return false
			}
		}
		return true
	case *CompoundTag:
		tb, ok := b.(*CompoundTag)
		if !ok || len(ta.TagMap) != len(tb.TagMap) {
			return false
		}
		for k, va := range ta.TagMap {
			vb, exists := tb.TagMap[k]
			if !exists {
				return false
			}
			if !equalTags(va, vb) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
