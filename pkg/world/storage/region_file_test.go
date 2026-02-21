package storage

import (
	"testing"

	"github.com/lulaide/gomc/pkg/nbt"
)

func TestRegionFileChunkRoundTrip(t *testing.T) {
	worldDir := t.TempDir()
	defer func() { _ = ClearRegionFileReferences() }()

	out, err := GetChunkOutputStream(worldDir, 0, 0)
	if err != nil {
		t.Fatalf("GetChunkOutputStream failed: %v", err)
	}
	if out == nil {
		t.Fatal("output stream should not be nil")
	}

	root := nbt.NewCompoundTag("Level")
	root.SetInteger("xPos", 0)
	root.SetInteger("zPos", 0)
	root.SetLong("LastUpdate", 12345)
	if err := nbt.Write(root, out); err != nil {
		t.Fatalf("nbt.Write failed: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("output close failed: %v", err)
	}

	in, err := GetChunkInputStream(worldDir, 0, 0)
	if err != nil {
		t.Fatalf("GetChunkInputStream failed: %v", err)
	}
	if in == nil {
		t.Fatal("input stream should not be nil after write")
	}
	defer in.Close()

	loaded, err := nbt.Read(in)
	if err != nil {
		t.Fatalf("nbt.Read failed: %v", err)
	}

	xTag, ok := loaded.GetTag("xPos").(*nbt.IntTag)
	if !ok || xTag.Data != 0 {
		t.Fatalf("xPos mismatch: tag=%T data=%v", loaded.GetTag("xPos"), xTag)
	}
	zTag, ok := loaded.GetTag("zPos").(*nbt.IntTag)
	if !ok || zTag.Data != 0 {
		t.Fatalf("zPos mismatch: tag=%T data=%v", loaded.GetTag("zPos"), zTag)
	}
}

func TestRegionFileCacheReuseAndClear(t *testing.T) {
	worldDir := t.TempDir()
	defer func() { _ = ClearRegionFileReferences() }()

	rf1, err := CreateOrLoadRegionFile(worldDir, 0, 0)
	if err != nil {
		t.Fatalf("create/load #1 failed: %v", err)
	}
	rf2, err := CreateOrLoadRegionFile(worldDir, 1, 1)
	if err != nil {
		t.Fatalf("create/load #2 failed: %v", err)
	}
	if rf1 != rf2 {
		t.Fatal("chunks in same region should reuse same RegionFile instance")
	}

	if err := ClearRegionFileReferences(); err != nil {
		t.Fatalf("ClearRegionFileReferences failed: %v", err)
	}

	rf3, err := CreateOrLoadRegionFile(worldDir, 0, 0)
	if err != nil {
		t.Fatalf("create/load #3 failed: %v", err)
	}
	if rf3 == rf1 {
		t.Fatal("cache clear should create a new RegionFile instance")
	}
}

func TestRegionFileOutOfBoundsAndMissingChunk(t *testing.T) {
	worldDir := t.TempDir()
	defer func() { _ = ClearRegionFileReferences() }()

	rf, err := CreateOrLoadRegionFile(worldDir, 0, 0)
	if err != nil {
		t.Fatalf("CreateOrLoadRegionFile failed: %v", err)
	}

	in, err := rf.GetChunkDataInputStream(32, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in != nil {
		t.Fatal("out-of-bounds input stream should be nil")
	}

	out, err := rf.GetChunkDataOutputStream(32, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Fatal("out-of-bounds output stream should be nil")
	}

	missing, err := GetChunkInputStream(worldDir, 3, 4)
	if err != nil {
		t.Fatalf("missing chunk read should not error: %v", err)
	}
	if missing != nil {
		t.Fatal("missing chunk should return nil stream")
	}
}
