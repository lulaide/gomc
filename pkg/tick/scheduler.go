package tick

import (
	"container/heap"
	"errors"
)

type coordKey struct {
	x int
	y int
	z int
}

type entryHeap []*NextTickListEntry

func (h entryHeap) Len() int { return len(h) }
func (h entryHeap) Less(i, j int) bool {
	return h[i].Compare(h[j]) < 0
}
func (h entryHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *entryHeap) Push(v any) {
	*h = append(*h, v.(*NextTickListEntry))
}
func (h *entryHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// Scheduler translates the pending tick collections in WorldServer:
// - pendingTickListEntriesHashSet
// - pendingTickListEntriesTreeSet
type Scheduler struct {
	entriesByCoord map[coordKey][]*NextTickListEntry
	ordered        entryHeap
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		entriesByCoord: make(map[coordKey][]*NextTickListEntry),
		ordered:        make(entryHeap, 0),
	}
	heap.Init(&s.ordered)
	return s
}

func (s *Scheduler) Size() int {
	return s.ordered.Len()
}

func (s *Scheduler) HasPending() bool {
	return s.Size() > 0
}

func (s *Scheduler) Schedule(entry *NextTickListEntry) bool {
	if entry == nil {
		return false
	}

	key := coordKey{x: entry.XCoord, y: entry.YCoord, z: entry.ZCoord}
	list := s.entriesByCoord[key]
	for _, existing := range list {
		if existing.Equals(entry) {
			return false
		}
	}

	s.entriesByCoord[key] = append(list, entry)
	heap.Push(&s.ordered, entry)
	return true
}

func (s *Scheduler) Peek() (*NextTickListEntry, error) {
	if s.ordered.Len() == 0 {
		return nil, errors.New("scheduler is empty")
	}
	return s.ordered[0], nil
}

// DrainDue translates WorldServer#tickUpdates cleaning phase.
// limit corresponds to the 1000 hard-cap in the Java loop.
func (s *Scheduler) DrainDue(currentWorldTime int64, force bool, limit int) []*NextTickListEntry {
	if limit <= 0 {
		return nil
	}

	if limit > s.ordered.Len() {
		limit = s.ordered.Len()
	}

	drained := make([]*NextTickListEntry, 0, limit)
	for i := 0; i < limit; i++ {
		first := s.ordered[0]
		if !force && first.ScheduledTime > currentWorldTime {
			break
		}

		popped := heap.Pop(&s.ordered).(*NextTickListEntry)
		s.removeFromSet(popped)
		drained = append(drained, popped)
	}

	return drained
}

func (s *Scheduler) removeFromSet(entry *NextTickListEntry) {
	key := coordKey{x: entry.XCoord, y: entry.YCoord, z: entry.ZCoord}
	list := s.entriesByCoord[key]
	if len(list) == 0 {
		return
	}

	for i, existing := range list {
		if existing == entry {
			list = append(list[:i], list[i+1:]...)
			if len(list) == 0 {
				delete(s.entriesByCoord, key)
			} else {
				s.entriesByCoord[key] = list
			}
			return
		}
	}
}
