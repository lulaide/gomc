package gen

import (
	"bytes"
	"testing"

	"github.com/lulaide/gomc/pkg/util"
)

func TestGenerateTaiga1TreeAtWorldPlacesLogsAndLeaves(t *testing.T) {
	for seed := int64(0); seed < 10000; seed++ {
		blocks := flatGrassChunkForTreeTests()
		rng := util.NewJavaRandom(seed)
		if !generateTaiga1TreeAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
			continue
		}

		logCount := countBlockID(blocks, 0, 0, blockIDLog)
		leafCount := countBlockID(blocks, 0, 0, blockIDLeaves)
		if logCount == 0 || leafCount == 0 {
			t.Fatalf("seed=%d: expected taiga1 logs/leaves, got logs=%d leaves=%d", seed, logCount, leafCount)
		}
		return
	}

	t.Fatal("failed to find deterministic taiga1 seed that generates successfully")
}

func TestGenerateTaiga2TreeAtWorldPlacesLogsAndLeaves(t *testing.T) {
	for seed := int64(0); seed < 10000; seed++ {
		blocks := flatGrassChunkForTreeTests()
		rng := util.NewJavaRandom(seed)
		if !generateTaiga2TreeAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
			continue
		}

		logCount := countBlockID(blocks, 0, 0, blockIDLog)
		leafCount := countBlockID(blocks, 0, 0, blockIDLeaves)
		if logCount == 0 || leafCount == 0 {
			t.Fatalf("seed=%d: expected taiga2 logs/leaves, got logs=%d leaves=%d", seed, logCount, leafCount)
		}
		return
	}

	t.Fatal("failed to find deterministic taiga2 seed that generates successfully")
}

func TestTaigaTreeSelectorUsesTaiga1Branch(t *testing.T) {
	// Translation reference:
	// - net.minecraft.src.BiomeGenTaiga.getRandomWorldGenForTrees(...)
	//   nextInt(3) == 0 ? WorldGenTaiga1 : WorldGenTaiga2
	found := false

	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(3) != 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(TaigaBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(3)
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateTaiga1TreeAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: taiga1 branch result mismatch, biome=%t direct=%t", seed, okBiome, okDirect)
		}
		if !okBiome {
			continue
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: taiga1 branch blocks mismatch vs direct generator", seed)
		}

		found = true
		break
	}

	if !found {
		t.Fatal("failed to find deterministic seed where taiga1 branch succeeds")
	}
}

func TestTaigaTreeSelectorUsesTaiga2Branch(t *testing.T) {
	found := false

	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(3) == 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(TaigaBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(3)
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateTaiga2TreeAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: taiga2 branch result mismatch, biome=%t direct=%t", seed, okBiome, okDirect)
		}
		if !okBiome {
			continue
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: taiga2 branch blocks mismatch vs direct generator", seed)
		}

		found = true
		break
	}

	if !found {
		t.Fatal("failed to find deterministic seed where taiga2 branch succeeds")
	}
}
