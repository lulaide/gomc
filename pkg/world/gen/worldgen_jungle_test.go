package gen

import (
	"bytes"
	"testing"

	"github.com/lulaide/gomc/pkg/util"
)

func TestGenerateJungleShrubAtWorldPlacesLogAndLeaves(t *testing.T) {
	blocks := flatGrassChunkForTreeTests()
	rng := util.NewJavaRandom(12001)

	if !generateJungleShrubAtWorld(blocks, rng, 0, 0, 8, 70, 8) {
		t.Fatal("expected jungle shrub generation to return true")
	}

	logCount := countBlockID(blocks, 0, 0, blockIDLog)
	leafCount := countBlockID(blocks, 0, 0, blockIDLeaves)
	if logCount == 0 || leafCount == 0 {
		t.Fatalf("expected jungle shrub logs/leaves, got logs=%d leaves=%d", logCount, leafCount)
	}
}

func TestGenerateJungleTreeAtWorldPlacesLogsLeavesAndVines(t *testing.T) {
	for seed := int64(0); seed < 20000; seed++ {
		blocks := flatGrassChunkForTreeTests()
		rng := util.NewJavaRandom(seed)
		if !generateJungleTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8, 4, true) {
			continue
		}

		logCount := countBlockID(blocks, 0, 0, blockIDLog)
		leafCount := countBlockID(blocks, 0, 0, blockIDLeaves)
		vineCount := countBlockID(blocks, 0, 0, blockIDVine)
		if logCount > 0 && leafCount > 0 && vineCount > 0 {
			return
		}
	}

	t.Fatal("failed to find deterministic jungle tree seed with logs, leaves, and vines")
}

func TestGenerateHugeJungleTreeAtWorldPlacesTrunkAndLeaves(t *testing.T) {
	for seed := int64(0); seed < 20000; seed++ {
		blocks := flatGrassChunkForTreeTests()
		rng := util.NewJavaRandom(seed)
		if !generateHugeJungleTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8, 10) {
			continue
		}

		logCount := countBlockID(blocks, 0, 0, blockIDLog)
		leafCount := countBlockID(blocks, 0, 0, blockIDLeaves)
		if logCount == 0 || leafCount == 0 {
			t.Fatalf("seed=%d: expected huge jungle logs/leaves, got logs=%d leaves=%d", seed, logCount, leafCount)
		}

		baseCoords := [][2]int{{8, 8}, {9, 8}, {8, 9}, {9, 9}}
		for _, c := range baseCoords {
			id := blockAtWorldForGen(blocks, 0, 0, c[0], 64, c[1], blockIDAir)
			if id != blockIDLog {
				t.Fatalf("seed=%d: expected 2x2 trunk log at (%d,64,%d), got id=%d", seed, c[0], c[1], id)
			}
		}
		return
	}

	t.Fatal("failed to find deterministic huge jungle tree seed that generates successfully")
}

func TestGenerateWorldVinesAtWorldPlacesVinesWhenSideAttachable(t *testing.T) {
	blocks := flatGrassChunkForTreeTests()
	for y := 64; y < worldHeightLegacyY; y++ {
		_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, 9, y, 8, blockIDStone)
	}

	rng := util.NewJavaRandom(12002)
	if !generateWorldVinesAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
		t.Fatal("expected world vines generation to return true")
	}

	vineCount := countBlockID(blocks, 0, 0, blockIDVine)
	if vineCount == 0 {
		t.Fatal("expected world vines generator to place at least one vine")
	}
}

func TestJungleTreeSelectorUsesBigTreeBranch(t *testing.T) {
	// Translation reference:
	// - net.minecraft.src.BiomeGenJungle.getRandomWorldGenForTrees(...)
	//   nextInt(10)==0 ? BigTree : ...
	found := false
	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(10) != 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(JungleBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(10)
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateBigTreeAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: jungle big tree branch result mismatch, biome=%t direct=%t", seed, okBiome, okDirect)
		}
		if !okBiome {
			continue
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: jungle big tree branch blocks mismatch vs direct generator", seed)
		}
		found = true
		break
	}
	if !found {
		t.Fatal("failed to find deterministic seed where jungle big tree branch succeeds")
	}
}

func TestJungleTreeSelectorUsesShrubBranch(t *testing.T) {
	found := false
	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(10) == 0 {
			continue
		}
		if selector.NextInt(2) != 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(JungleBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(10)
		_ = rngDirect.NextInt(2)
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateJungleShrubAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: jungle shrub branch result mismatch, biome=%t direct=%t", seed, okBiome, okDirect)
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: jungle shrub branch blocks mismatch vs direct generator", seed)
		}
		found = true
		break
	}
	if !found {
		t.Fatal("failed to find deterministic seed where jungle shrub branch matches direct generator")
	}
}

func TestJungleTreeSelectorUsesHugeTreeBranch(t *testing.T) {
	found := false
	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(10) == 0 {
			continue
		}
		if selector.NextInt(2) == 0 {
			continue
		}
		if selector.NextInt(3) != 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(JungleBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(10)
		_ = rngDirect.NextInt(2)
		_ = rngDirect.NextInt(3)
		baseHeight := 10 + int(rngDirect.NextInt(20))
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateHugeJungleTreeAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8, baseHeight)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: jungle huge tree branch result mismatch, biome=%t direct=%t", seed, okBiome, okDirect)
		}
		if !okBiome {
			continue
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: jungle huge tree branch blocks mismatch vs direct generator", seed)
		}
		found = true
		break
	}
	if !found {
		t.Fatal("failed to find deterministic seed where jungle huge tree branch succeeds")
	}
}

func TestJungleTreeSelectorUsesSmallTreeBranch(t *testing.T) {
	found := false
	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(10) == 0 {
			continue
		}
		if selector.NextInt(2) == 0 {
			continue
		}
		if selector.NextInt(3) == 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(JungleBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(10)
		_ = rngDirect.NextInt(2)
		_ = rngDirect.NextInt(3)
		minTreeHeight := 4 + int(rngDirect.NextInt(7))
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateJungleTreeAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8, minTreeHeight, true)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: jungle small tree branch result mismatch, biome=%t direct=%t", seed, okBiome, okDirect)
		}
		if !okBiome {
			continue
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: jungle small tree branch blocks mismatch vs direct generator", seed)
		}
		found = true
		break
	}
	if !found {
		t.Fatal("failed to find deterministic seed where jungle small tree branch succeeds")
	}
}
