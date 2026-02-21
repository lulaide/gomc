package storage

import (
	"errors"
	"testing"

	"github.com/lulaide/gomc/pkg/nbt"
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

type loaderBlock struct {
	id int
}

func (b *loaderBlock) ID() int {
	return b.id
}

func (b *loaderBlock) IsAssociatedBlockID(otherID int) bool {
	return b.id == otherID
}

type testWorldBridge struct{}

func (testWorldBridge) HasNoSky() bool                                         { return false }
func (testWorldBridge) IsRemote() bool                                         { return false }
func (testWorldBridge) MarkBlockForRenderUpdate(x, y, z int)                   {}
func (testWorldBridge) MarkBlocksDirtyVertical(x, z, yFrom, yTo int)           {}
func (testWorldBridge) DoChunksNearChunkExist(x, y, z, radius int) bool        { return false }
func (testWorldBridge) UpdateLightByType(kind chunk.EnumSkyBlock, x, y, z int) {}
func (testWorldBridge) GetChunkHeightMapMinimum(x, z int) int                  { return 0 }
func (testWorldBridge) GetHeightValue(x, z int) int                            { return 0 }
func (testWorldBridge) OnBlockPreDestroy(blockID, x, y, z, metadata int)       {}
func (testWorldBridge) BreakBlock(blockID, x, y, z, oldBlockID, oldMetadata int) {
}
func (testWorldBridge) OnBlockAdded(blockID, x, y, z int)          {}
func (testWorldBridge) RemoveBlockTileEntity(x, y, z int)          {}
func (testWorldBridge) EnsureBlockTileEntity(blockID, x, y, z int) {}
func (testWorldBridge) UpdateBlockTileEntityInfo(x, y, z int)      {}

func TestAnvilChunkLoaderLoadFromPendingBeforeDisk(t *testing.T) {
	worldDir := t.TempDir()
	defer func() { _ = ClearRegionFileReferences() }()
	block.ResetRegistry()
	block.Register(&loaderBlock{id: 1})

	loader := NewAnvilChunkLoader(worldDir)
	col := chunk.NewColumn(2, 3)
	col.InhabitedTime = 111
	sec := chunk.NewExtendedBlockStorage(0, true)
	sec.SetExtBlockID(0, 0, 0, 1)
	arr := make([]*chunk.ExtendedBlockStorage, 16)
	arr[0] = sec
	col.SetStorageArrays(arr)

	if err := loader.SaveChunk(col, 1000, false, nil); err != nil {
		t.Fatalf("SaveChunk failed: %v", err)
	}
	if loader.PendingCount() != 1 {
		t.Fatalf("pending count mismatch: got=%d want=1", loader.PendingCount())
	}

	loaded, ticks, err := loader.LoadChunk(false, 2, 3)
	if err != nil {
		t.Fatalf("LoadChunk failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded chunk is nil")
	}
	if len(ticks) != 0 {
		t.Fatalf("unexpected ticks loaded: %d", len(ticks))
	}
	if loaded.InhabitedTime != 111 {
		t.Fatalf("inhabited time mismatch: got=%d want=111", loaded.InhabitedTime)
	}
}

func TestAnvilChunkLoaderSaveReplaceAndWriteNextIO(t *testing.T) {
	worldDir := t.TempDir()
	defer func() { _ = ClearRegionFileReferences() }()
	block.ResetRegistry()
	block.Register(&loaderBlock{id: 1})

	loader := NewAnvilChunkLoader(worldDir)
	col1 := chunk.NewColumn(1, 1)
	col1.InhabitedTime = 10
	col2 := chunk.NewColumn(1, 1)
	col2.InhabitedTime = 20

	if err := loader.SaveChunk(col1, 1000, false, nil); err != nil {
		t.Fatalf("SaveChunk #1 failed: %v", err)
	}
	if err := loader.SaveChunk(col2, 1000, false, nil); err != nil {
		t.Fatalf("SaveChunk #2 failed: %v", err)
	}

	if loader.PendingCount() != 1 {
		t.Fatalf("duplicate coord should replace pending entry: got=%d", loader.PendingCount())
	}

	if !loader.WriteNextIO() {
		t.Fatal("WriteNextIO should process one pending entry")
	}
	if loader.WriteNextIO() {
		t.Fatal("WriteNextIO should be empty on second call")
	}

	loaded, _, err := loader.LoadChunk(false, 1, 1)
	if err != nil {
		t.Fatalf("LoadChunk failed after write: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded chunk is nil")
	}
	if loaded.InhabitedTime != 20 {
		t.Fatalf("replacement chunk was not written: got=%d want=20", loaded.InhabitedTime)
	}
}

func TestAnvilChunkLoaderMissingLevelValidation(t *testing.T) {
	worldDir := t.TempDir()
	defer func() { _ = ClearRegionFileReferences() }()
	loader := NewAnvilChunkLoader(worldDir)

	root := nbt.NewCompoundTag("")
	out, err := GetChunkOutputStream(worldDir, 0, 0)
	if err != nil {
		t.Fatalf("GetChunkOutputStream failed: %v", err)
	}
	if out == nil {
		t.Fatal("output stream is nil")
	}
	if err := nbt.Write(root, out); err != nil {
		t.Fatalf("nbt.Write failed: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	_, _, loadErr := loader.LoadChunk(false, 0, 0)
	if loadErr == nil {
		t.Fatal("expected missing level error")
	}
	if !errors.Is(loadErr, ErrMissingLevelData) {
		t.Fatalf("unexpected error: %v", loadErr)
	}
}

func TestAnvilChunkLoaderRuntimeChunkRoundTrip(t *testing.T) {
	worldDir := t.TempDir()
	defer func() { _ = ClearRegionFileReferences() }()
	block.ResetRegistry()
	block.Register(&loaderBlock{id: 1})
	block.SetLightOpacity(1, 255)

	loader := NewAnvilChunkLoader(worldDir)
	runtimeChunk := chunk.NewChunk(testWorldBridge{}, 5, -4)
	if ok := runtimeChunk.SetBlockIDWithMetadata(0, 0, 0, 1, 2); !ok {
		t.Fatal("runtime block placement failed")
	}
	runtimeChunk.InhabitedTime = 77

	if err := loader.SaveRuntimeChunk(runtimeChunk, 2000, false, nil); err != nil {
		t.Fatalf("SaveRuntimeChunk failed: %v", err)
	}
	if !loader.WriteNextIO() {
		t.Fatal("expected pending runtime chunk write")
	}

	loadedRuntime, ticks, err := loader.LoadRuntimeChunk(testWorldBridge{}, false, 5, -4)
	if err != nil {
		t.Fatalf("LoadRuntimeChunk failed: %v", err)
	}
	if loadedRuntime == nil {
		t.Fatal("loaded runtime chunk is nil")
	}
	if len(ticks) != 0 {
		t.Fatalf("unexpected tile ticks: %d", len(ticks))
	}
	if got := loadedRuntime.GetBlockID(0, 0, 0); got != 1 {
		t.Fatalf("runtime block id mismatch: got=%d want=1", got)
	}
	if got := loadedRuntime.GetBlockMetadata(0, 0, 0); got != 2 {
		t.Fatalf("runtime block metadata mismatch: got=%d want=2", got)
	}
	if loadedRuntime.InhabitedTime != 77 {
		t.Fatalf("runtime inhabited time mismatch: got=%d want=77", loadedRuntime.InhabitedTime)
	}
}
