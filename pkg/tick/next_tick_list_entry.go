package tick

import (
	"fmt"
	"sync/atomic"

	"github.com/lulaide/gomc/pkg/world/block"
)

var nextTickEntryID int64

// NextTickListEntry translates net.minecraft.src.NextTickListEntry.
type NextTickListEntry struct {
	XCoord int
	YCoord int
	ZCoord int

	BlockID int

	ScheduledTime int64
	Priority      int

	tickEntryID int64
}

func NewNextTickListEntry(x, y, z, blockID int) *NextTickListEntry {
	id := atomic.AddInt64(&nextTickEntryID, 1) - 1
	return &NextTickListEntry{
		XCoord:      x,
		YCoord:      y,
		ZCoord:      z,
		BlockID:     blockID,
		tickEntryID: id,
	}
}

func (e *NextTickListEntry) Equals(other *NextTickListEntry) bool {
	if other == nil {
		return false
	}
	return e.XCoord == other.XCoord &&
		e.YCoord == other.YCoord &&
		e.ZCoord == other.ZCoord &&
		block.IsAssociatedBlockID(e.BlockID, other.BlockID)
}

func (e *NextTickListEntry) HashCode() int {
	// Translation target: NextTickListEntry#hashCode()
	h := int32((e.XCoord*1024*1024 + e.ZCoord*1024 + e.YCoord) * 256)
	return int(h)
}

func (e *NextTickListEntry) SetScheduledTime(t int64) *NextTickListEntry {
	e.ScheduledTime = t
	return e
}

func (e *NextTickListEntry) SetPriority(p int) {
	e.Priority = p
}

// Compare matches NextTickListEntry#comparer.
func (e *NextTickListEntry) Compare(other *NextTickListEntry) int {
	if e.ScheduledTime < other.ScheduledTime {
		return -1
	}
	if e.ScheduledTime > other.ScheduledTime {
		return 1
	}
	if e.Priority != other.Priority {
		return e.Priority - other.Priority
	}
	if e.tickEntryID < other.tickEntryID {
		return -1
	}
	if e.tickEntryID > other.tickEntryID {
		return 1
	}
	return 0
}

func (e *NextTickListEntry) String() string {
	return fmt.Sprintf(
		"%d: (%d, %d, %d), %d, %d, %d",
		e.BlockID, e.XCoord, e.YCoord, e.ZCoord, e.ScheduledTime, e.Priority, e.tickEntryID,
	)
}

func ResetEntryIDsForTest() {
	atomic.StoreInt64(&nextTickEntryID, 0)
}
