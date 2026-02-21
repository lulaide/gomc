package tick

import (
	"testing"

	"github.com/lulaide/gomc/pkg/world/block"
)

type mirrorBlock struct {
	id    int
	other int
}

func (m *mirrorBlock) ID() int { return m.id }
func (m *mirrorBlock) IsAssociatedBlockID(otherID int) bool {
	return otherID == m.id || otherID == m.other
}

func TestScheduler_DeduplicateByAssociatedBlock(t *testing.T) {
	block.ResetRegistry()
	block.Register(&mirrorBlock{id: 10, other: 11})
	block.Register(&mirrorBlock{id: 11, other: 10})
	ResetEntryIDsForTest()

	s := NewScheduler()
	a := NewNextTickListEntry(1, 2, 3, 10).SetScheduledTime(100)
	b := NewNextTickListEntry(1, 2, 3, 11).SetScheduledTime(120)

	if !s.Schedule(a) {
		t.Fatal("first schedule should succeed")
	}
	if s.Schedule(b) {
		t.Fatal("associated block at same coords should deduplicate")
	}
}

func TestScheduler_DrainDueOrder(t *testing.T) {
	block.ResetRegistry()
	block.Register(block.NewBaseDefinition(1))
	ResetEntryIDsForTest()

	s := NewScheduler()

	e1 := NewNextTickListEntry(0, 0, 0, 1)
	e1.SetScheduledTime(10)
	e1.SetPriority(1)

	e2 := NewNextTickListEntry(0, 0, 1, 1)
	e2.SetScheduledTime(10)
	e2.SetPriority(0)

	e3 := NewNextTickListEntry(0, 0, 2, 1)
	e3.SetScheduledTime(20)
	e3.SetPriority(0)

	s.Schedule(e1)
	s.Schedule(e2)
	s.Schedule(e3)

	drained := s.DrainDue(10, false, 1000)
	if len(drained) != 2 {
		t.Fatalf("unexpected drained count: got=%d want=2", len(drained))
	}
	if drained[0] != e2 || drained[1] != e1 {
		t.Fatalf("drain order mismatch")
	}

	if !s.HasPending() {
		t.Fatal("scheduler should still contain future tick")
	}

	rest := s.DrainDue(10, true, 1000)
	if len(rest) != 1 || rest[0] != e3 {
		t.Fatalf("force drain mismatch")
	}
}
