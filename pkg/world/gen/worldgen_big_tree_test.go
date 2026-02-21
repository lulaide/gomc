package gen

import (
	"bytes"
	"testing"

	"github.com/lulaide/gomc/pkg/util"
)

func flatGrassChunkForTreeTests() []byte {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDGrass)
		}
	}
	return blocks
}

func countBlockID(blocks []byte, targetChunkX, targetChunkZ int, id byte) int {
	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < worldHeightLegacyY; y++ {
				b, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z)
				if ok && b == id {
					count++
				}
			}
		}
	}
	return count
}

func TestGenerateBigTreeAtWorldPlacesLogsAndLeaves(t *testing.T) {
	blocks := flatGrassChunkForTreeTests()
	rng := util.NewJavaRandom(171717)

	if !generateBigTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
		t.Fatal("expected big tree generation to succeed on flat grass")
	}

	base, ok := blockAtWorldInTargetChunk(blocks, 0, 0, 8, 63, 8)
	if !ok || (base != blockIDGrass && base != blockIDDirt) {
		t.Fatalf("expected trunk base to remain grass/dirt for big tree, got=%d ok=%t", base, ok)
	}

	logCount := countBlockID(blocks, 0, 0, blockIDLog)
	leafCount := countBlockID(blocks, 0, 0, blockIDLeaves)
	if logCount == 0 || leafCount == 0 {
		t.Fatalf("expected big tree logs/leaves, got logs=%d leaves=%d", logCount, leafCount)
	}
}

func TestForestTreeSelectorUsesBigTreeBranch(t *testing.T) {
	// Translation reference:
	// - net.minecraft.src.BiomeGenForest.getRandomWorldGenForTrees(...)
	//   nextInt(5)==0 ? WorldGenForest : (nextInt(10)==0 ? WorldGenBigTree : WorldGenTrees)
	found := false

	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(5) == 0 {
			continue
		}
		if selector.NextInt(10) != 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(ForestBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		// Keep RNG state aligned with generateBiomeTreeAtWorld branch checks.
		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(5)
		_ = rngDirect.NextInt(10)
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateBigTreeAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: forest branch result mismatch, biome=%t directBigTree=%t", seed, okBiome, okDirect)
		}
		if !okBiome {
			continue
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: forest branch blocks mismatch vs direct big tree generation", seed)
		}

		found = true
		break
	}

	if !found {
		t.Fatal("failed to find deterministic seed where forest big tree branch succeeds")
	}
}
