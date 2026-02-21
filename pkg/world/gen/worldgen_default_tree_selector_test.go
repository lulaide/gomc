package gen

import (
	"bytes"
	"testing"

	"github.com/lulaide/gomc/pkg/util"
)

func TestDefaultTreeSelectorUsesBigTreeBranch(t *testing.T) {
	// Translation reference:
	// - net.minecraft.src.BiomeGenBase.getRandomWorldGenForTrees(...)
	//   nextInt(10)==0 ? WorldGenBigTree : WorldGenTrees
	found := false
	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(10) != 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(ExtremeHillsBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(10)
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateBigTreeAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: default big tree branch result mismatch, biome=%t direct=%t", seed, okBiome, okDirect)
		}
		if !okBiome {
			continue
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: default big tree branch blocks mismatch vs direct generator", seed)
		}
		found = true
		break
	}

	if !found {
		t.Fatal("failed to find deterministic seed where default big tree branch succeeds")
	}
}

func TestDefaultTreeSelectorUsesNormalTreeBranch(t *testing.T) {
	found := false
	for seed := int64(0); seed < 100000; seed++ {
		selector := util.NewJavaRandom(seed)
		if selector.NextInt(10) == 0 {
			continue
		}

		blocksBiome := flatGrassChunkForTreeTests()
		rngBiome := util.NewJavaRandom(seed)
		okBiome := generateBiomeTreeAtWorld(ExtremeHillsBiome, blocksBiome, rngBiome, 0, 0, 8, 64, 8)

		rngDirect := util.NewJavaRandom(seed)
		_ = rngDirect.NextInt(10)
		blocksDirect := flatGrassChunkForTreeTests()
		okDirect := generateTreeAtWorld(blocksDirect, rngDirect, 0, 0, 8, 64, 8, 4)

		if okBiome != okDirect {
			t.Fatalf("seed=%d: default normal tree branch result mismatch, biome=%t direct=%t", seed, okBiome, okDirect)
		}
		if !okBiome {
			continue
		}
		if !bytes.Equal(blocksBiome, blocksDirect) {
			t.Fatalf("seed=%d: default normal tree branch blocks mismatch vs direct generator", seed)
		}
		found = true
		break
	}

	if !found {
		t.Fatal("failed to find deterministic seed where default normal tree branch succeeds")
	}
}
