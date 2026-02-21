package server

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lulaide/gomc/pkg/util"
	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
	"github.com/lulaide/gomc/pkg/world/gen"
	"github.com/lulaide/gomc/pkg/world/storage"
)

// flatWorld is a baseline chunk source used by the login/play pipeline.
//
// Translation target:
// - WorldServer + ChunkProviderServer responsibilities used by NetServerHandler/PlayerManager paths.
type flatWorld struct {
	mu        sync.RWMutex
	chunks    map[chunk.CoordIntPair]*chunk.Chunk
	dirty     map[chunk.CoordIntPair]struct{}
	loader    *storage.AnvilChunkLoader
	hasNoSky  bool
	generator worldChunkGenerator

	seed          int64
	generatorName string
	spawnBlockX   int32
	spawnBlockY   int32
	spawnBlockZ   int32
}

type worldChunkGenerator interface {
	GenerateChunk(chunkX, chunkZ int32) *chunk.Chunk
}

type flatChunkGenerator struct{}

func (flatChunkGenerator) GenerateChunk(chunkX, chunkZ int32) *chunk.Chunk {
	return buildFlatChunkAt(chunkX, chunkZ)
}

var defaultBlockRegistryOnce sync.Once

const (
	grassBlockID  = 2
	leavesBlockID = 18
)

func ensureDefaultBlockRegistry() {
	defaultBlockRegistryOnce.Do(func() {
		for id := 1; id <= maxPlaceableBlock; id++ {
			if block.Exists(id) {
				continue
			}
			block.Register(block.NewBaseDefinition(id))
		}
	})
}

func newFlatWorld() *flatWorld {
	ensureDefaultBlockRegistry()
	ensureOverworldBlockProperties()

	return &flatWorld{
		chunks:        make(map[chunk.CoordIntPair]*chunk.Chunk),
		dirty:         make(map[chunk.CoordIntPair]struct{}),
		generator:     flatChunkGenerator{},
		seed:          0,
		generatorName: "flat",
		spawnBlockX:   0,
		spawnBlockY:   4,
		spawnBlockZ:   0,
	}
}

func newFlatWorldWithStorage(worldDir string) *flatWorld {
	ensureDefaultBlockRegistry()
	ensureOverworldBlockProperties()

	var loader *storage.AnvilChunkLoader
	if worldDir != "" {
		loader = storage.NewAnvilChunkLoader(worldDir)
	}

	levelData, _, err := loadWorldLevelData(worldDir)
	if err != nil {
		levelData = defaultWorldLevelData()
	}

	w := &flatWorld{
		chunks:      make(map[chunk.CoordIntPair]*chunk.Chunk),
		dirty:       make(map[chunk.CoordIntPair]struct{}),
		loader:      loader,
		seed:        levelData.RandomSeed,
		spawnBlockX: levelData.SpawnX,
		spawnBlockY: levelData.SpawnY,
		spawnBlockZ: levelData.SpawnZ,
	}

	name := strings.ToLower(strings.TrimSpace(levelData.GeneratorName))
	switch name {
	case "flat":
		w.generator = flatChunkGenerator{}
		w.generatorName = "flat"
	default:
		worldType := gen.ParseWorldType(levelData.GeneratorName, levelData.GeneratorVersion)
		w.generator = gen.NewChunkProviderGenerate(levelData.RandomSeed, gen.NewGenLayerBiomeSource(levelData.RandomSeed, worldType))
		w.generatorName = worldType.String()
	}

	if w.spawnBlockY <= 0 || w.spawnBlockY >= 255 {
		if w.generatorName == "flat" {
			w.spawnBlockY = 4
		} else {
			w.spawnBlockY = 64
		}
	}
	return w
}

func (w *flatWorld) getChunk(chunkX, chunkZ int32) *chunk.Chunk {
	key := chunk.NewCoordIntPair(chunkX, chunkZ)

	w.mu.RLock()
	if ch, ok := w.chunks[key]; ok {
		w.mu.RUnlock()
		return ch
	}
	w.mu.RUnlock()

	w.mu.Lock()
	defer w.mu.Unlock()
	if ch, ok := w.chunks[key]; ok {
		return ch
	}
	var ch *chunk.Chunk
	if w.loader != nil {
		loaded, _, err := w.loader.LoadRuntimeChunk(nil, w.hasNoSky, chunkX, chunkZ)
		if err == nil && loaded != nil {
			ch = loaded
		}
	}
	if ch == nil {
		if w.generator != nil {
			ch = w.generator.GenerateChunk(chunkX, chunkZ)
		}
		if ch == nil {
			ch = buildFlatChunkAt(chunkX, chunkZ)
		}
	}
	w.chunks[key] = ch
	return ch
}

func (w *flatWorld) spawnBlockPosition() (int32, int32, int32) {
	return w.spawnBlockX, w.spawnBlockY, w.spawnBlockZ
}

func (w *flatWorld) spawnPosition() (float64, float64, float64) {
	return float64(w.spawnBlockX) + 0.5, float64(w.spawnBlockY), float64(w.spawnBlockZ) + 0.5
}

// Translation reference:
// - net.minecraft.src.WorldProvider#canCoordinateBeSpawn()
// - net.minecraft.src.World#getFirstUncoveredBlock()
func (w *flatWorld) canCoordinateBeSpawn(x, z int) bool {
	return w.getFirstUncoveredBlock(x, z) == grassBlockID
}

// Translation reference:
// - net.minecraft.src.World#getFirstUncoveredBlock()
func (w *flatWorld) getFirstUncoveredBlock(x, z int) int {
	y := 63
	for y < 255 && !w.isAirBlock(x, y+1, z) {
		y++
	}
	id, _ := w.getBlock(x, y, z)
	return id
}

// Translation reference:
// - net.minecraft.src.World#getTopSolidOrLiquidBlock()
func (w *flatWorld) getTopSolidOrLiquidBlock(x, z int) int {
	ch, localX, localZ := w.getChunkForBlock(x, z)
	top := ch.GetTopFilledSegment() + 15
	if top < 1 {
		top = 1
	}
	if top > 255 {
		top = 255
	}

	for y := top; y > 0; y-- {
		id := ch.GetBlockID(localX, y, localZ)
		if id != 0 && block.BlocksMovement(id) && id != leavesBlockID {
			return y + 1
		}
	}
	return -1
}

func (w *flatWorld) isAirBlock(x, y, z int) bool {
	id, _ := w.getBlock(x, y, z)
	return id == 0
}

func (w *flatWorld) findPlayerSpawnColumn() (int, int) {
	x := int(w.spawnBlockX)
	z := int(w.spawnBlockZ)
	if w.canCoordinateBeSpawn(x, z) {
		return x, z
	}

	// Translation reference:
	// - net.minecraft.src.WorldServer#createSpawnPosition()
	rng := util.NewJavaRandom(w.seed)
	for i := 0; i < 1000 && !w.canCoordinateBeSpawn(x, z); i++ {
		x += int(rng.NextInt(64)) - int(rng.NextInt(64))
		z += int(rng.NextInt(64)) - int(rng.NextInt(64))
	}
	return x, z
}

// safePlayerSpawnPosition returns a player feet position that stands above terrain
// and avoids embedding inside solid blocks.
func (w *flatWorld) safePlayerSpawnPosition() (float64, float64, float64) {
	x, z := w.findPlayerSpawnColumn()
	y := w.getTopSolidOrLiquidBlock(x, z)
	if y <= 0 || y >= 255 {
		y = int(w.spawnBlockY)
		if y <= 0 || y >= 255 {
			if w.generatorName == "flat" {
				y = 5
			} else {
				y = 64
			}
		}
	}

	for y < 255 {
		feetID, _ := w.getBlock(x, y, z)
		headID, _ := w.getBlock(x, y+1, z)
		belowID, _ := w.getBlock(x, y-1, z)
		if !block.BlocksMovement(feetID) && !block.BlocksMovement(headID) && block.BlocksMovement(belowID) {
			break
		}
		y++
	}

	if y > 255 {
		y = 255
	}
	return float64(x) + 0.5, float64(y), float64(z) + 0.5
}

func (w *flatWorld) getChunks(coords []chunk.CoordIntPair) []*chunk.Chunk {
	out := make([]*chunk.Chunk, 0, len(coords))
	for _, pos := range coords {
		out = append(out, w.getChunk(pos.ChunkXPos, pos.ChunkZPos))
	}
	return out
}

func (w *flatWorld) getBlock(x, y, z int) (blockID int, metadata int) {
	if y < 0 || y >= 256 {
		return 0, 0
	}
	ch, localX, localZ := w.getChunkForBlock(x, z)
	return ch.GetBlockID(localX, y, localZ), ch.GetBlockMetadata(localX, y, localZ)
}

func (w *flatWorld) setBlock(x, y, z, blockID, metadata int) bool {
	if y < 0 || y >= 256 {
		return false
	}
	chunkX := int32(x >> 4)
	chunkZ := int32(z >> 4)
	key := chunk.NewCoordIntPair(chunkX, chunkZ)
	ch, localX, localZ := w.getChunkForBlock(x, z)
	if ch.SetBlockIDWithMetadata(localX, y, localZ, blockID, metadata) {
		w.mu.Lock()
		w.dirty[key] = struct{}{}
		w.mu.Unlock()
		return true
	}
	return false
}

func (w *flatWorld) getChunkForBlock(x, z int) (*chunk.Chunk, int, int) {
	chunkX := int32(x >> 4)
	chunkZ := int32(z >> 4)
	localX := x & 15
	localZ := z & 15
	return w.getChunk(chunkX, chunkZ), localX, localZ
}

func (w *flatWorld) saveDirty(totalWorldTime int64) error {
	if w.loader == nil {
		return nil
	}

	w.mu.Lock()
	coords := make([]chunk.CoordIntPair, 0, len(w.dirty))
	for c := range w.dirty {
		coords = append(coords, c)
	}
	w.mu.Unlock()

	var firstErr error
	for _, c := range coords {
		w.mu.RLock()
		ch := w.chunks[c]
		w.mu.RUnlock()
		if ch == nil {
			continue
		}
		if err := w.loader.SaveRuntimeChunk(ch, totalWorldTime, w.hasNoSky, nil); err != nil && firstErr == nil {
			firstErr = err
			continue
		}
		w.mu.Lock()
		delete(w.dirty, c)
		w.mu.Unlock()
	}
	w.loader.SaveExtraData()
	return firstErr
}

func (w *flatWorld) saveAll(totalWorldTime int64) error {
	if w.loader == nil {
		return nil
	}

	w.mu.RLock()
	coords := make([]chunk.CoordIntPair, 0, len(w.chunks))
	for c := range w.chunks {
		coords = append(coords, c)
	}
	w.mu.RUnlock()

	var firstErr error
	for _, c := range coords {
		w.mu.RLock()
		ch := w.chunks[c]
		w.mu.RUnlock()
		if ch == nil {
			continue
		}
		if err := w.loader.SaveRuntimeChunk(ch, totalWorldTime, w.hasNoSky, nil); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.loader.SaveExtraData()

	w.mu.Lock()
	w.dirty = make(map[chunk.CoordIntPair]struct{})
	w.mu.Unlock()
	return firstErr
}

func (w *flatWorld) closeStorage() error {
	if w.loader == nil {
		return nil
	}
	if err := storage.ClearRegionFileReferences(); err != nil {
		return fmt.Errorf("close region file cache: %w", err)
	}
	return nil
}
