package chunk

import (
	"testing"

	"github.com/lulaide/gomc/pkg/world/block"
)

type tickBlock struct {
	id   int
	tick bool
}

func (b *tickBlock) ID() int {
	return b.id
}

func (b *tickBlock) IsAssociatedBlockID(otherID int) bool {
	return b.id == otherID
}

func (b *tickBlock) GetTickRandomly() bool {
	return b.tick
}

func TestExtendedBlockStorageSetGetAndCounts(t *testing.T) {
	block.ResetRegistry()
	block.Register(&tickBlock{id: 1, tick: false})
	block.Register(&tickBlock{id: 2, tick: true})

	section := NewExtendedBlockStorage(0, true)
	if !section.IsEmpty() {
		t.Fatal("new section must be empty")
	}

	section.SetExtBlockID(0, 0, 0, 1)
	section.SetExtBlockID(1, 0, 0, 2)

	if got := section.GetExtBlockID(0, 0, 0); got != 1 {
		t.Fatalf("id mismatch for slot 0: got=%d want=1", got)
	}
	if got := section.GetExtBlockID(1, 0, 0); got != 2 {
		t.Fatalf("id mismatch for slot 1: got=%d want=2", got)
	}
	if section.IsEmpty() {
		t.Fatal("section should not be empty after placing blocks")
	}
	if !section.GetNeedsRandomTick() {
		t.Fatal("section should require random ticking when tick-random block exists")
	}

	section.SetExtBlockID(1, 0, 0, 0)
	if section.GetNeedsRandomTick() {
		t.Fatal("section should not require random ticking after removing tick-random block")
	}
}

func TestExtendedBlockStorageMSBAndRemoveInvalid(t *testing.T) {
	block.ResetRegistry()
	block.Register(&tickBlock{id: 400, tick: false})

	section := NewExtendedBlockStorage(0, false)
	section.SetExtBlockID(2, 3, 4, 400)
	if got := section.GetExtBlockID(2, 3, 4); got != 400 {
		t.Fatalf("msb id mismatch: got=%d want=400", got)
	}

	// Remove registration to force invalid cleanup.
	block.ResetRegistry()
	section.RemoveInvalidBlocks()
	if got := section.GetExtBlockID(2, 3, 4); got != 0 {
		t.Fatalf("invalid block should be cleared, got=%d", got)
	}
}
