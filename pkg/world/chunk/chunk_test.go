package chunk

import (
	"testing"

	"github.com/lulaide/gomc/pkg/world/block"
)

type worldPos struct {
	x int
	z int
}

type lightUpdate struct {
	kind EnumSkyBlock
	x    int
	y    int
	z    int
}

type blockEvent struct {
	blockID int
	x       int
	y       int
	z       int
	meta    int
}

type mockWorld struct {
	hasNoSky bool
	remote   bool
	near     bool

	renderUpdates int
	lightUpdates  []lightUpdate
	added         []blockEvent
	preDestroy    []blockEvent
	broken        []blockEvent

	chunkHeightMin map[worldPos]int
	heightValues   map[worldPos]int
}

func (w *mockWorld) HasNoSky() bool { return w.hasNoSky }
func (w *mockWorld) IsRemote() bool { return w.remote }

func (w *mockWorld) MarkBlockForRenderUpdate(x, y, z int) { w.renderUpdates++ }
func (w *mockWorld) MarkBlocksDirtyVertical(x, z, yFrom, yTo int) {
}

func (w *mockWorld) DoChunksNearChunkExist(x, y, z, radius int) bool {
	return w.near
}

func (w *mockWorld) UpdateLightByType(kind EnumSkyBlock, x, y, z int) {
	w.lightUpdates = append(w.lightUpdates, lightUpdate{kind: kind, x: x, y: y, z: z})
}

func (w *mockWorld) GetChunkHeightMapMinimum(x, z int) int {
	if w.chunkHeightMin == nil {
		return 0
	}
	if v, ok := w.chunkHeightMin[worldPos{x: x, z: z}]; ok {
		return v
	}
	return 0
}

func (w *mockWorld) GetHeightValue(x, z int) int {
	if w.heightValues == nil {
		return 0
	}
	if v, ok := w.heightValues[worldPos{x: x, z: z}]; ok {
		return v
	}
	return 0
}

func (w *mockWorld) OnBlockPreDestroy(blockID, x, y, z, metadata int) {
	w.preDestroy = append(w.preDestroy, blockEvent{blockID: blockID, x: x, y: y, z: z, meta: metadata})
}

func (w *mockWorld) BreakBlock(blockID, x, y, z, oldBlockID, oldMetadata int) {
	w.broken = append(w.broken, blockEvent{blockID: blockID, x: x, y: y, z: z, meta: oldMetadata})
}

func (w *mockWorld) OnBlockAdded(blockID, x, y, z int) {
	w.added = append(w.added, blockEvent{blockID: blockID, x: x, y: y, z: z})
}

func (w *mockWorld) RemoveBlockTileEntity(x, y, z int)          {}
func (w *mockWorld) EnsureBlockTileEntity(blockID, x, y, z int) {}
func (w *mockWorld) UpdateBlockTileEntityInfo(x, y, z int)      {}

type testBlockDef struct {
	id             int
	opacity        int
	blocksMovement bool
	liquid         bool
}

func (b *testBlockDef) ID() int { return b.id }

func (b *testBlockDef) IsAssociatedBlockID(otherID int) bool {
	return b.id == otherID
}

func (b *testBlockDef) GetLightOpacity() int { return b.opacity }
func (b *testBlockDef) BlocksMovement() bool { return b.blocksMovement }
func (b *testBlockDef) IsLiquid() bool       { return b.liquid }

func TestChunkGenerateHeightMap(t *testing.T) {
	block.ResetRegistry()
	t.Cleanup(block.ResetRegistry)
	block.Register(&testBlockDef{id: 1, opacity: 255, blocksMovement: true})

	w := &mockWorld{near: true}
	c := NewChunk(w, 0, 0)
	sec := NewExtendedBlockStorage(0, true)
	sec.SetExtBlockID(1, 4, 2, 1)
	c.GetBlockStorageArray()[0] = sec

	c.GenerateHeightMap()

	if got := c.GetHeightValue(1, 2); got != 5 {
		t.Fatalf("height mismatch: got=%d want=5", got)
	}

	if got := c.PrecipitationHeightMap[1+(2<<4)]; got != precipitationHeightUnset {
		t.Fatalf("precipitation cache mismatch: got=%d want=%d", got, precipitationHeightUnset)
	}

	if !c.IsModified {
		t.Fatal("chunk should be marked modified")
	}
}

func TestChunkGenerateSkylightMap(t *testing.T) {
	block.ResetRegistry()
	t.Cleanup(block.ResetRegistry)
	block.Register(&testBlockDef{id: 1, opacity: 255, blocksMovement: true})

	w := &mockWorld{near: true}
	c := NewChunk(w, 0, 0)
	sec := NewExtendedBlockStorage(0, true)
	sec.SetExtBlockID(0, 0, 0, 1)
	c.GetBlockStorageArray()[0] = sec

	c.GenerateSkylightMap()

	if got := c.GetHeightValue(0, 0); got != 1 {
		t.Fatalf("height mismatch: got=%d want=1", got)
	}
	if got := c.HeightMapMinimum; got != 1 {
		t.Fatalf("height map minimum mismatch: got=%d want=1", got)
	}
	if got := sec.GetExtSkylightValue(0, 1, 0); got != 15 {
		t.Fatalf("skylight mismatch at y=1: got=%d want=15", got)
	}
	if w.renderUpdates == 0 {
		t.Fatal("expected render updates during skylight generation")
	}
	if !c.updateSkylightColumns[0] {
		t.Fatal("expected skylight column propagation flag to be set")
	}
}

func TestChunkSetBlockIDWithMetadata(t *testing.T) {
	block.ResetRegistry()
	t.Cleanup(block.ResetRegistry)
	block.Register(&testBlockDef{id: 1, opacity: 255, blocksMovement: true})

	w := &mockWorld{near: true}
	c := NewChunk(w, 2, 3)

	if ok := c.SetBlockIDWithMetadata(0, 0, 0, 1, 3); !ok {
		t.Fatal("expected initial block placement to succeed")
	}
	if got := c.GetBlockID(0, 0, 0); got != 1 {
		t.Fatalf("block id mismatch: got=%d want=1", got)
	}
	if got := c.GetBlockMetadata(0, 0, 0); got != 3 {
		t.Fatalf("metadata mismatch: got=%d want=3", got)
	}
	if len(w.added) != 1 {
		t.Fatalf("expected one block-added callback, got=%d", len(w.added))
	}
	if !c.IsModified {
		t.Fatal("chunk should be marked modified")
	}
	if ok := c.SetBlockIDWithMetadata(0, 0, 0, 1, 3); ok {
		t.Fatal("setting same id+metadata should be no-op")
	}
}

func TestChunkGetPrecipitationHeight(t *testing.T) {
	block.ResetRegistry()
	t.Cleanup(block.ResetRegistry)
	block.Register(&testBlockDef{id: 2, opacity: 255, blocksMovement: true})

	w := &mockWorld{near: true}
	c := NewChunk(w, 0, 0)
	sec := NewExtendedBlockStorage(0, true)
	sec.SetExtBlockID(0, 4, 0, 2)
	c.GetBlockStorageArray()[0] = sec

	if got := c.GetPrecipitationHeight(0, 0); got != 5 {
		t.Fatalf("precipitation height mismatch: got=%d want=5", got)
	}
}

func TestChunkSavedLightValueAndSetLightValue(t *testing.T) {
	block.ResetRegistry()
	t.Cleanup(block.ResetRegistry)

	w := &mockWorld{near: true}
	c := NewChunk(w, 0, 0)

	if got := c.GetSavedLightValue(EnumSkyBlockSky, 0, 10, 0); got != 15 {
		t.Fatalf("sky default light mismatch: got=%d want=15", got)
	}
	if got := c.GetSavedLightValue(EnumSkyBlockBlock, 0, 10, 0); got != 0 {
		t.Fatalf("block default light mismatch: got=%d want=0", got)
	}

	c.SetLightValue(EnumSkyBlockSky, 0, 2, 0, 7)
	if got := c.GetSavedLightValue(EnumSkyBlockSky, 0, 2, 0); got != 7 {
		t.Fatalf("saved sky light mismatch: got=%d want=7", got)
	}

	c.SetLightValue(EnumSkyBlockBlock, 0, 2, 0, 9)
	if got := c.GetSavedLightValue(EnumSkyBlockBlock, 0, 2, 0); got != 9 {
		t.Fatalf("saved block light mismatch: got=%d want=9", got)
	}
}

func TestChunkUpdateSkylight(t *testing.T) {
	block.ResetRegistry()
	t.Cleanup(block.ResetRegistry)

	w := &mockWorld{
		near: true,
		heightValues: map[worldPos]int{
			{x: 0, z: 0}: 3,
		},
	}
	c := NewChunk(w, 0, 0)
	c.propagateSkylightOcclusion(0, 0)

	c.UpdateSkylight()

	if len(w.lightUpdates) != 4 {
		t.Fatalf("light updates mismatch: got=%d want=4", len(w.lightUpdates))
	}
	if c.isGapLightingUpdated {
		t.Fatal("gap lighting flag should be cleared after update")
	}
}

func TestChunkColumnRoundTrip(t *testing.T) {
	block.ResetRegistry()
	t.Cleanup(block.ResetRegistry)
	block.Register(&testBlockDef{id: 1, opacity: 255, blocksMovement: true})

	col := NewColumn(4, -2)
	col.HeightMap[0] = 7
	col.IsTerrainPopulated = true
	col.HasEntities = true
	col.InhabitedTime = 123
	sec := NewExtendedBlockStorage(0, true)
	sec.SetExtBlockID(0, 0, 0, 1)
	sec.SetExtBlockMetadata(0, 0, 0, 5)
	col.GetStorageArrays()[0] = sec
	col.GetBiomeArray()[0] = 3

	ch := NewChunkFromColumn(&mockWorld{near: true}, col)
	if got := ch.GetBlockID(0, 0, 0); got != 1 {
		t.Fatalf("chunk block id mismatch: got=%d want=1", got)
	}
	if got := ch.GetBlockMetadata(0, 0, 0); got != 5 {
		t.Fatalf("chunk block meta mismatch: got=%d want=5", got)
	}
	if got := ch.HeightMap[0]; got != 7 {
		t.Fatalf("chunk height map mismatch: got=%d want=7", got)
	}

	back := ch.ToColumn()
	if back.XPos != 4 || back.ZPos != -2 {
		t.Fatalf("column coords mismatch: got=(%d,%d) want=(4,-2)", back.XPos, back.ZPos)
	}
	if back.HeightMap[0] != 7 {
		t.Fatalf("column height map mismatch: got=%d want=7", back.HeightMap[0])
	}
	if back.GetBiomeArray()[0] != 3 {
		t.Fatalf("column biome mismatch: got=%d want=3", back.GetBiomeArray()[0])
	}
	if got := back.GetStorageArrays()[0].GetExtBlockID(0, 0, 0); got != 1 {
		t.Fatalf("column block id mismatch: got=%d want=1", got)
	}
}
