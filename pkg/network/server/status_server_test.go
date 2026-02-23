package server

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lulaide/gomc/pkg/network/protocol"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

func TestBuildServerPingResponseLegacy(t *testing.T) {
	ping := &protocol.Packet254ServerPing{ReadSuccessfully: 0}
	got := BuildServerPingResponse(ping, "GoMC", "1.6.4", 1, 20)
	want := "GoMC\u00a71\u00a720"
	if got != want {
		t.Fatalf("legacy ping response mismatch: got=%q want=%q", got, want)
	}
}

func TestBuildServerPingResponseModern(t *testing.T) {
	ping := &protocol.Packet254ServerPing{ReadSuccessfully: 78}
	got := BuildServerPingResponse(ping, "GoMC", "1.6.4", 1, 20)
	parts := strings.Split(got, "\x00")
	if len(parts) != 6 {
		t.Fatalf("modern ping parts mismatch: got=%d want=6", len(parts))
	}
	if parts[0] != "\u00a71" || parts[1] != "78" || parts[2] != "1.6.4" || parts[3] != "GoMC" || parts[4] != "1" || parts[5] != "20" {
		t.Fatalf("modern ping response mismatch: %q", got)
	}
}

func TestTickPlayerInfoBroadcastsIndexedPlayerPing(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	var bufA bytes.Buffer
	var bufB bytes.Buffer
	a := newInteractionTestSession(srv, &bufA)
	b := newInteractionTestSession(srv, &bufB)
	a.latencyMS.Store(42)
	b.latencyMS.Store(99)

	srv.activeMu.Lock()
	srv.activePlayers[a] = "A"
	srv.activePlayers[b] = "B"
	srv.activeOrder = []*loginSession{a, b}
	srv.playerPingIdx = 0 // Tick increments to 1, then picks index 1.
	srv.activeMu.Unlock()

	srv.TickPlayerInfo()

	packetA, err := protocol.ReadPacket(&bufA, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("reader A failed: %v", err)
	}
	infoA, ok := packetA.(*protocol.Packet201PlayerInfo)
	if !ok {
		t.Fatalf("packet type A mismatch: %T", packetA)
	}
	if infoA.PlayerName != "B" || infoA.Ping != 99 {
		t.Fatalf("packet A mismatch: %#v", infoA)
	}

	packetB, err := protocol.ReadPacket(&bufB, protocol.DirectionClientbound)
	if err != nil {
		t.Fatalf("reader B failed: %v", err)
	}
	infoB, ok := packetB.(*protocol.Packet201PlayerInfo)
	if !ok {
		t.Fatalf("packet type B mismatch: %T", packetB)
	}
	if infoB.PlayerName != "B" || infoB.Ping != 99 {
		t.Fatalf("packet B mismatch: %#v", infoB)
	}
}

func TestStatusServerSaveWorldDirtyPersistsBlocks(t *testing.T) {
	worldDir := t.TempDir()
	srv := NewStatusServer(StatusConfig{
		PersistWorld: true,
		WorldDir:     worldDir,
	})

	if !srv.world.setBlock(3, 5, 3, 1, 2) {
		t.Fatal("setBlock returned false")
	}
	if err := srv.SaveWorldDirty(); err != nil {
		t.Fatalf("SaveWorldDirty failed: %v", err)
	}
	if err := srv.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	srv2 := NewStatusServer(StatusConfig{
		PersistWorld: true,
		WorldDir:     worldDir,
	})
	defer func() {
		_ = srv2.Close()
	}()

	blockID, meta := srv2.world.getBlock(3, 5, 3)
	if blockID != 1 || meta != 2 {
		t.Fatalf("persisted block mismatch: got=(%d,%d) want=(1,2)", blockID, meta)
	}
}

func TestStatusServerSaveWorldDirtySkippedWhenAutoSaveDisabled(t *testing.T) {
	worldDir := t.TempDir()
	srv := NewStatusServer(StatusConfig{
		PersistWorld: true,
		WorldDir:     worldDir,
	})
	srv.setAutoSaveEnabled(false)

	if !srv.world.setBlock(4, 5, 4, 1, 1) {
		t.Fatal("setBlock returned false")
	}
	if err := srv.SaveWorldDirty(); err != nil {
		t.Fatalf("SaveWorldDirty failed: %v", err)
	}
	if err := srv.world.closeStorage(); err != nil {
		t.Fatalf("closeStorage failed: %v", err)
	}

	srv2 := NewStatusServer(StatusConfig{
		PersistWorld: true,
		WorldDir:     worldDir,
	})
	defer func() {
		_ = srv2.Close()
	}()

	blockID, meta := srv2.world.getBlock(4, 5, 4)
	if blockID == 1 && meta == 1 {
		t.Fatalf("block should not persist while auto-save disabled: got=(%d,%d)", blockID, meta)
	}
}

func TestFindChunksForSpawningSendsMobSpawnPacket(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})
	srv.mobRand.SetSeed(1)

	var watcherBuf bytes.Buffer
	watcher := newInteractionTestSession(srv, &watcherBuf)
	watcher.playerRegistered = true
	watcher.entityID = 101

	watcher.loadedChunks = make(map[chunk.CoordIntPair]struct{})
	for cx := -8; cx <= 8; cx++ {
		for cz := -8; cz <= 8; cz++ {
			watcher.loadedChunks[chunk.NewCoordIntPair(int32(cx), int32(cz))] = struct{}{}
		}
	}

	srv.activeMu.Lock()
	srv.activePlayers[watcher] = "watcher"
	srv.activeOrder = []*loginSession{watcher}
	srv.activeMu.Unlock()

	spawned := 0
	for i := 0; i < 200 && spawned == 0; i++ {
		spawned += srv.findChunksForSpawning(true, true, true)
	}
	if spawned == 0 {
		t.Fatal("expected at least one mob spawn")
	}

	foundMobSpawn := false
	for {
		packet, err := protocol.ReadPacket(&watcherBuf, protocol.DirectionClientbound)
		if err != nil {
			break
		}
		if _, ok := packet.(*protocol.Packet24MobSpawn); ok {
			foundMobSpawn = true
			break
		}
	}

	if !foundMobSpawn {
		t.Fatal("expected watcher to receive Packet24MobSpawn")
	}
}

func TestTickChunkInhabitedTimeCountsChunksWithPlayersOnce(t *testing.T) {
	srv := NewStatusServer(StatusConfig{})

	a := newInteractionTestSession(srv, &bytes.Buffer{})
	a.playerRegistered = true
	a.playerDead = false
	a.entityID = 301
	a.playerX = 0.5
	a.playerY = 5.0
	a.playerZ = 0.5

	b := newInteractionTestSession(srv, &bytes.Buffer{})
	b.playerRegistered = true
	b.playerDead = false
	b.entityID = 302
	b.playerX = 1.5
	b.playerY = 5.0
	b.playerZ = 1.5

	srv.activeMu.Lock()
	srv.activePlayers[a] = "a"
	srv.activePlayers[b] = "b"
	srv.activeOrder = []*loginSession{a, b}
	srv.activeMu.Unlock()

	ch00 := srv.world.getChunk(0, 0)
	start00 := ch00.InhabitedTime
	srv.TickChunkInhabitedTime()
	if got := ch00.InhabitedTime; got != start00+1 {
		t.Fatalf("inhabited time should increment once for shared chunk: got=%d want=%d", got, start00+1)
	}

	b.playerX = 16.5
	b.playerZ = 1.5
	ch10 := srv.world.getChunk(1, 0)
	start10 := ch10.InhabitedTime

	srv.TickChunkInhabitedTime()
	if got := ch00.InhabitedTime; got != start00+2 {
		t.Fatalf("chunk (0,0) inhabited time mismatch: got=%d want=%d", got, start00+2)
	}
	if got := ch10.InhabitedTime; got != start10+1 {
		t.Fatalf("chunk (1,0) inhabited time mismatch: got=%d want=%d", got, start10+1)
	}
}
