package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lulaide/gomc/pkg/nbt"
	"github.com/lulaide/gomc/pkg/world/block"
)

func TestFlatWorldGeneratesFlatChunk(t *testing.T) {
	w := newFlatWorld()
	ch := w.getChunk(0, 0)

	if got := ch.GetBlockID(0, 0, 0); got != 7 {
		t.Fatalf("bedrock mismatch: got=%d want=7", got)
	}
	if got := ch.GetBlockID(0, 4, 0); got != 2 {
		t.Fatalf("top block mismatch: got=%d want=2", got)
	}
	if got := ch.GetBlockID(0, 5, 0); got != 0 {
		t.Fatalf("air mismatch: got=%d want=0", got)
	}
}

func TestFlatWorldSetGetBlockNegativeCoord(t *testing.T) {
	w := newFlatWorld()
	if !w.setBlock(-1, 5, -1, 1, 2) {
		t.Fatal("setBlock returned false")
	}

	blockID, metadata := w.getBlock(-1, 5, -1)
	if blockID != 1 || metadata != 2 {
		t.Fatalf("block mismatch: got=(%d,%d) want=(1,2)", blockID, metadata)
	}
}

func TestFlatWorldPersistenceSaveAndLoad(t *testing.T) {
	worldDir := t.TempDir()

	w := newFlatWorldWithStorage(worldDir)
	if !w.setBlock(2, 5, 2, 1, 3) {
		t.Fatal("setBlock returned false")
	}
	if err := w.saveAll(123); err != nil {
		t.Fatalf("saveAll failed: %v", err)
	}
	if err := w.closeStorage(); err != nil {
		t.Fatalf("closeStorage failed: %v", err)
	}

	w2 := newFlatWorldWithStorage(worldDir)
	defer func() {
		_ = w2.closeStorage()
	}()
	blockID, metadata := w2.getBlock(2, 5, 2)
	if blockID != 1 || metadata != 3 {
		t.Fatalf("persisted block mismatch: got=(%d,%d) want=(1,3)", blockID, metadata)
	}
}

func TestWorldWithStorageUsesLevelDataDefaultGenerator(t *testing.T) {
	worldDir := t.TempDir()
	if err := writeTestLevelDat(worldDir, func(data *nbt.CompoundTag) {
		data.SetString("generatorName", "default")
		data.SetInteger("generatorVersion", 1)
		data.SetLong("RandomSeed", 12345)
		data.SetInteger("SpawnX", 11)
		data.SetInteger("SpawnY", 70)
		data.SetInteger("SpawnZ", -7)
	}); err != nil {
		t.Fatalf("write level.dat failed: %v", err)
	}

	w := newFlatWorldWithStorage(worldDir)
	defer func() {
		_ = w.closeStorage()
	}()
	x, y, z := w.spawnBlockPosition()
	if x != 11 || y != 70 || z != -7 {
		t.Fatalf("spawn mismatch: got=(%d,%d,%d) want=(11,70,-7)", x, y, z)
	}

	ch := w.getChunk(0, 0)
	if got := ch.GetBlockID(0, 5, 0); got == 0 {
		t.Fatalf("expected default generator non-air at y=5, got=%d", got)
	}
}

func TestWorldWithStorageUsesLevelDataFlatGenerator(t *testing.T) {
	worldDir := t.TempDir()
	if err := writeTestLevelDat(worldDir, func(data *nbt.CompoundTag) {
		data.SetString("generatorName", "flat")
		data.SetInteger("generatorVersion", 0)
		data.SetLong("RandomSeed", 1)
	}); err != nil {
		t.Fatalf("write level.dat failed: %v", err)
	}

	w := newFlatWorldWithStorage(worldDir)
	defer func() {
		_ = w.closeStorage()
	}()
	ch := w.getChunk(0, 0)
	if got := ch.GetBlockID(0, 4, 0); got != 2 {
		t.Fatalf("flat generator grass mismatch: got=%d want=2", got)
	}
	if got := ch.GetBlockID(0, 5, 0); got != 0 {
		t.Fatalf("flat generator air mismatch: got=%d want=0", got)
	}
}

func TestFlatWorldSafePlayerSpawnPositionIsStandable(t *testing.T) {
	w := newFlatWorld()
	x, y, z := w.safePlayerSpawnPosition()

	bx := int(x)
	by := int(y)
	bz := int(z)
	feetID, _ := w.getBlock(bx, by, bz)
	headID, _ := w.getBlock(bx, by+1, bz)
	belowID, _ := w.getBlock(bx, by-1, bz)

	if block.BlocksMovement(feetID) || block.BlocksMovement(headID) || !block.BlocksMovement(belowID) {
		t.Fatalf("safe spawn is not standable: pos=(%.1f,%.1f,%.1f) feet=%d head=%d below=%d", x, y, z, feetID, headID, belowID)
	}
}

func TestDefaultGeneratorSafePlayerSpawnAvoidsUndergroundSpawnY(t *testing.T) {
	worldDir := t.TempDir()
	if err := writeTestLevelDat(worldDir, func(data *nbt.CompoundTag) {
		data.SetString("generatorName", "default")
		data.SetInteger("generatorVersion", 1)
		data.SetLong("RandomSeed", 12345)
		data.SetInteger("SpawnX", 0)
		data.SetInteger("SpawnY", 1)
		data.SetInteger("SpawnZ", 0)
	}); err != nil {
		t.Fatalf("write level.dat failed: %v", err)
	}

	w := newFlatWorldWithStorage(worldDir)
	defer func() {
		_ = w.closeStorage()
	}()

	x, y, z := w.safePlayerSpawnPosition()
	bx := int(x)
	by := int(y)
	bz := int(z)
	feetID, _ := w.getBlock(bx, by, bz)
	headID, _ := w.getBlock(bx, by+1, bz)
	belowID, _ := w.getBlock(bx, by-1, bz)

	if y <= 1 {
		t.Fatalf("safe spawn y should be above underground input: got=%.1f", y)
	}
	if block.BlocksMovement(feetID) || block.BlocksMovement(headID) || !block.BlocksMovement(belowID) {
		t.Fatalf("safe spawn is not standable: pos=(%.1f,%.1f,%.1f) feet=%d head=%d below=%d", x, y, z, feetID, headID, belowID)
	}
}

func writeTestLevelDat(worldDir string, mutate func(data *nbt.CompoundTag)) error {
	root := nbt.NewCompoundTag("")
	data := nbt.NewCompoundTag("Data")
	data.SetLong("RandomSeed", 0)
	data.SetString("generatorName", "default")
	data.SetInteger("generatorVersion", 1)
	data.SetString("generatorOptions", "")
	data.SetInteger("SpawnX", 0)
	data.SetInteger("SpawnY", 64)
	data.SetInteger("SpawnZ", 0)
	if mutate != nil {
		mutate(data)
	}
	root.SetTag("Data", data)

	path := filepath.Join(worldDir, "level.dat")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return nbt.WriteCompressed(root, f)
}
