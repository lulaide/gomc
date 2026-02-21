package gen

import (
	"testing"

	"github.com/lulaide/gomc/pkg/util"
)

func TestGenerateLakeAtWorldCarvesAndFills(t *testing.T) {
	provider := NewChunkProviderGenerate(1, NewFixedBiomeSource(PlainsBiome))
	blocks := make([]byte, 32768)
	for i := range blocks {
		blocks[i] = blockIDStone
	}

	rng := util.NewJavaRandom(12345)
	if !provider.generateLakeAtWorld(blocks, rng, 0, 0, 8, 64, 8, blockIDWater) {
		t.Fatal("expected lake generation to succeed")
	}

	waterCount := 0
	airCount := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < worldHeightLegacyY; y++ {
				id := blocks[chunkBlockIndex(x, z, y)]
				if id == blockIDWater {
					waterCount++
				}
				if id == blockIDAir {
					airCount++
				}
			}
		}
	}

	if waterCount == 0 {
		t.Fatal("expected water blocks from generated lake")
	}
	if airCount == 0 {
		t.Fatal("expected carved air cavity from generated lake")
	}
}

func TestGenerateDungeonAtWorldPlacesSpawner(t *testing.T) {
	blocks := make([]byte, 32768)
	for i := range blocks {
		blocks[i] = blockIDStone
	}

	const (
		centerX = 8
		centerY = 40
		centerZ = 8
		seed    = 4242
	)

	probe := util.NewJavaRandom(seed)
	radiusX := int(probe.NextInt(2)) + 2
	_ = int(probe.NextInt(2)) + 2
	minX := centerX - radiusX - 1
	// Single doorway so opening count stays within [1,5].
	_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, minX, centerY, centerZ, blockIDAir)
	_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, minX, centerY+1, centerZ, blockIDAir)

	rng := util.NewJavaRandom(seed)
	if !generateDungeonAtWorld(blocks, rng, 0, 0, centerX, centerY, centerZ) {
		t.Fatal("expected dungeon generation to succeed")
	}

	id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, centerX, centerY, centerZ)
	if !ok || id != blockIDMobSpawner {
		t.Fatalf("expected mob spawner at center, got id=%d ok=%t", id, ok)
	}
}
