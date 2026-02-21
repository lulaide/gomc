package storage

import (
	"testing"

	"github.com/lulaide/gomc/pkg/nbt"
	"github.com/lulaide/gomc/pkg/tick"
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

type codecBlock struct {
	id int
}

func (b *codecBlock) ID() int {
	return b.id
}

func (b *codecBlock) IsAssociatedBlockID(otherID int) bool {
	return b.id == otherID
}

func TestAnvilChunkCodecRoundTrip(t *testing.T) {
	block.ResetRegistry()
	block.Register(&codecBlock{id: 1})
	block.Register(&codecBlock{id: 300})
	tick.ResetEntryIDsForTest()

	col := chunk.NewColumn(2, 3)
	col.HeightMap[0] = 64
	col.IsTerrainPopulated = true
	col.InhabitedTime = 9999

	sec := chunk.NewExtendedBlockStorage(0, true)
	sec.SetExtBlockID(1, 2, 3, 1)
	sec.SetExtBlockID(2, 2, 3, 300)
	sec.SetExtBlockMetadata(2, 2, 3, 7)
	sec.SetExtBlocklightValue(2, 2, 3, 12)
	sec.SetExtSkylightValue(2, 2, 3, 15)

	arr := make([]*chunk.ExtendedBlockStorage, 16)
	arr[0] = sec
	col.SetStorageArrays(arr)

	biomes := make([]byte, 256)
	for i := range biomes {
		biomes[i] = byte(i % 23)
	}
	col.SetBiomeArray(biomes)

	pending := []*tick.NextTickListEntry{
		tick.NewNextTickListEntry(10, 64, 10, 1).SetScheduledTime(1200),
	}
	pending[0].SetPriority(2)

	level := EncodeChunkLevelNBT(col, 1000, false, pending)
	if !level.HasKey("Sections") {
		t.Fatal("encoded level missing Sections")
	}

	decoded, tileTicks := DecodeChunkLevelNBT(level, false)
	if decoded.XPos != 2 || decoded.ZPos != 3 {
		t.Fatalf("decoded chunk pos mismatch: got=(%d,%d)", decoded.XPos, decoded.ZPos)
	}
	if !decoded.IsTerrainPopulated {
		t.Fatal("decoded terrain flag mismatch")
	}
	if decoded.InhabitedTime != 9999 {
		t.Fatalf("decoded inhabited time mismatch: got=%d", decoded.InhabitedTime)
	}

	decodedSec := decoded.GetStorageArrays()[0]
	if decodedSec == nil {
		t.Fatal("decoded section 0 is nil")
	}
	if got := decodedSec.GetExtBlockID(1, 2, 3); got != 1 {
		t.Fatalf("decoded block id mismatch #1: got=%d", got)
	}
	if got := decodedSec.GetExtBlockID(2, 2, 3); got != 300 {
		t.Fatalf("decoded block id mismatch #2: got=%d", got)
	}
	if got := decodedSec.GetExtBlockMetadata(2, 2, 3); got != 7 {
		t.Fatalf("decoded metadata mismatch: got=%d", got)
	}
	if got := decodedSec.GetExtBlocklightValue(2, 2, 3); got != 12 {
		t.Fatalf("decoded blocklight mismatch: got=%d", got)
	}
	if got := decodedSec.GetExtSkylightValue(2, 2, 3); got != 15 {
		t.Fatalf("decoded skylight mismatch: got=%d", got)
	}

	if len(tileTicks) != 1 {
		t.Fatalf("decoded tile tick count mismatch: got=%d", len(tileTicks))
	}
	if tileTicks[0].Delay != 200 || tileTicks[0].Priority != 2 {
		t.Fatalf("decoded tile tick mismatch: got delay=%d priority=%d", tileTicks[0].Delay, tileTicks[0].Priority)
	}
}

func TestAnvilChunkCodecNoSkyWritesZeroSkyLight(t *testing.T) {
	block.ResetRegistry()
	block.Register(&codecBlock{id: 1})

	col := chunk.NewColumn(0, 0)
	sec := chunk.NewExtendedBlockStorage(0, false)
	sec.SetExtBlockID(0, 0, 0, 1)
	sec.SetExtBlocklightValue(0, 0, 0, 9)
	arr := make([]*chunk.ExtendedBlockStorage, 16)
	arr[0] = sec
	col.SetStorageArrays(arr)

	level := EncodeChunkLevelNBT(col, 0, true, nil)
	sections := getList(level, "Sections")
	if sections.TagCount() != 1 {
		t.Fatalf("expected one section, got=%d", sections.TagCount())
	}
	secTag := sections.TagAt(0).(*nbt.CompoundTag)
	sky := getByteArray(secTag, "SkyLight")
	for i, b := range sky {
		if b != 0 {
			t.Fatalf("skylight byte at %d should be zero, got=%d", i, b)
		}
	}
}
