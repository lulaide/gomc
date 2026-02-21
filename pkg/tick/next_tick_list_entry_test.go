package tick

import (
	"testing"

	"github.com/lulaide/gomc/pkg/world/block"
)

type associatedBlock struct {
	id int
}

func (a *associatedBlock) ID() int {
	return a.id
}

func (a *associatedBlock) IsAssociatedBlockID(otherID int) bool {
	return (a.id == 5 && otherID == 6) || (a.id == 6 && otherID == 5) || a.id == otherID
}

func TestNextTickListEntryEqualsAndHash(t *testing.T) {
	block.ResetRegistry()
	block.Register(&associatedBlock{id: 5})
	block.Register(&associatedBlock{id: 6})
	ResetEntryIDsForTest()

	a := NewNextTickListEntry(1, 2, 3, 5)
	b := NewNextTickListEntry(1, 2, 3, 6)
	c := NewNextTickListEntry(1, 2, 4, 6)

	if !a.Equals(b) {
		t.Fatal("entries with associated block ids should be equal")
	}
	if a.Equals(c) {
		t.Fatal("entries with different coordinates should not be equal")
	}

	wantHash := int(int32((1*1024*1024 + 3*1024 + 2) * 256))
	if got := a.HashCode(); got != wantHash {
		t.Fatalf("hash mismatch: got=%d want=%d", got, wantHash)
	}
}

func TestNextTickListEntryCompareOrder(t *testing.T) {
	block.ResetRegistry()
	block.Register(block.NewBaseDefinition(1))
	ResetEntryIDsForTest()

	first := NewNextTickListEntry(0, 0, 0, 1)
	second := NewNextTickListEntry(0, 0, 0, 1)

	first.SetScheduledTime(100)
	second.SetScheduledTime(100)
	first.SetPriority(0)
	second.SetPriority(1)

	if got := first.Compare(second); got >= 0 {
		t.Fatalf("priority compare mismatch: got=%d want<0", got)
	}

	second.SetPriority(0)
	if got := first.Compare(second); got >= 0 {
		t.Fatalf("id compare mismatch: got=%d want<0", got)
	}
}
