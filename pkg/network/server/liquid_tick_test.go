package server

import (
	"bytes"
	"testing"

	"github.com/lulaide/gomc/pkg/world/chunk"
)

func attachWaterTickWatcher(srv *StatusServer) *loginSession {
	watcher := newInteractionTestSession(srv, &bytes.Buffer{})
	watcher.playerRegistered = true
	watcher.playerDead = false
	watcher.entityID = 9001
	watcher.loadedChunks = map[chunk.CoordIntPair]struct{}{
		chunk.NewCoordIntPair(0, 0): {},
	}

	srv.activeMu.Lock()
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{watcher}
	srv.activeMu.Unlock()
	return watcher
}

func TestTickWaterFlowCreatesFallingColumn(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attachWaterTickWatcher(srv)

	if !srv.world.setBlock(8, 12, 8, blockIDStillWater, 0) {
		t.Fatal("failed to place still water source")
	}

	srv.AdvanceWorldTime(1)

	id, meta := srv.world.getBlock(8, 11, 8)
	if id != blockIDFlowingWater || meta != 8 {
		t.Fatalf("falling water mismatch: got=(%d,%d) want=(%d,8)", id, meta, blockIDFlowingWater)
	}

	sourceID, sourceMeta := srv.world.getBlock(8, 12, 8)
	if sourceID != blockIDStillWater || sourceMeta != 0 {
		t.Fatalf("source should remain still water: got=(%d,%d) want=(%d,0)", sourceID, sourceMeta, blockIDStillWater)
	}
}

func TestTickWaterFlowSpreadsHorizontallyWhenBlockedBelow(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	attachWaterTickWatcher(srv)

	if !srv.world.setBlock(8, 12, 8, blockIDStillWater, 0) {
		t.Fatal("failed to place still water source")
	}
	if !srv.world.setBlock(8, 11, 8, 1, 0) {
		t.Fatal("failed to place solid block below source")
	}

	srv.AdvanceWorldTime(1)

	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		id, meta := srv.world.getBlock(8+d[0], 12, 8+d[1])
		if id != blockIDFlowingWater || meta != 1 {
			t.Fatalf("horizontal spread mismatch at (%d,%d): got=(%d,%d) want=(%d,1)", d[0], d[1], id, meta, blockIDFlowingWater)
		}
	}
}
