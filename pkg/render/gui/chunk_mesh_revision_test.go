//go:build cgo

package gui

import "testing"

func TestChunkMeshRevisionIncludesNeighbors(t *testing.T) {
	revs := map[[2]int32]uint64{
		{0, 0}: 10,
	}
	revAt := func(x, z int32) uint64 {
		return revs[[2]int32{x, z}]
	}

	base := chunkMeshRevision(0, 0, revAt)
	if base == 0 {
		t.Fatal("base mesh revision should not be zero")
	}

	revs[[2]int32{1, 0}] = 25
	changed := chunkMeshRevision(0, 0, revAt)
	if changed == base {
		t.Fatal("mesh revision should change when neighbor revision changes")
	}

	revs[[2]int32{1, 0}] = 0
	back := chunkMeshRevision(0, 0, revAt)
	if back == changed {
		t.Fatal("mesh revision should change when neighbor unloads")
	}
}

func TestChunkMeshRevisionZeroWhenBaseMissing(t *testing.T) {
	revAt := func(x, z int32) uint64 {
		return 0
	}
	if got := chunkMeshRevision(0, 0, revAt); got != 0 {
		t.Fatalf("expected zero revision for missing base chunk, got %d", got)
	}
}
