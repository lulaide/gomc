package gen

import (
	"testing"

	"github.com/lulaide/gomc/pkg/util"
)

func newFlatGrassChunkWithMetadata() ([]byte, func()) {
	blocks := flatGrassChunkForTreeTests()
	metadata := make([]byte, len(blocks))
	registerBlockMetadataBuffer(blocks, metadata)
	cleanup := func() {
		unregisterBlockMetadataBuffer(blocks)
	}
	return blocks, cleanup
}

func countBlockWithMetadata(blocks []byte, targetChunkX, targetChunkZ int, id, meta byte) int {
	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < worldHeightLegacyY; y++ {
				blockID, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z)
				if !ok || blockID != id {
					continue
				}
				blockMeta, ok := blockMetadataAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z)
				if !ok || blockMeta != meta {
					continue
				}
				count++
			}
		}
	}
	return count
}

func hasOnlyValidVineMetadata(blocks []byte, targetChunkX, targetChunkZ int) bool {
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < worldHeightLegacyY; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z)
				if !ok || id != blockIDVine {
					continue
				}
				meta, ok := blockMetadataAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z)
				if !ok {
					return false
				}
				if meta != 1 && meta != 2 && meta != 4 && meta != 8 {
					return false
				}
			}
		}
	}
	return true
}

func TestForestTreeUsesBirchMetadata(t *testing.T) {
	blocks, cleanup := newFlatGrassChunkWithMetadata()
	defer cleanup()
	rng := util.NewJavaRandom(9494)
	if !generateForestTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
		t.Fatal("expected forest tree generation to succeed")
	}

	logs := countBlockWithMetadata(blocks, 0, 0, blockIDLog, 2)
	leaves := countBlockWithMetadata(blocks, 0, 0, blockIDLeaves, 2)
	if logs == 0 || leaves == 0 {
		t.Fatalf("expected birch metadata on forest tree, got logs(meta=2)=%d leaves(meta=2)=%d", logs, leaves)
	}
}

func TestTaigaTreesUseSpruceMetadata(t *testing.T) {
	found := false
	for seed := int64(0); seed < 10000; seed++ {
		blocks, cleanup := newFlatGrassChunkWithMetadata()
		rng := util.NewJavaRandom(seed)
		ok := generateTaiga1TreeAtWorld(blocks, rng, 0, 0, 8, 64, 8)
		if !ok {
			cleanup()
			continue
		}

		logs := countBlockWithMetadata(blocks, 0, 0, blockIDLog, 1)
		leaves := countBlockWithMetadata(blocks, 0, 0, blockIDLeaves, 1)
		if logs == 0 || leaves == 0 {
			cleanup()
			t.Fatalf("seed=%d: expected spruce metadata on taiga tree, got logs(meta=1)=%d leaves(meta=1)=%d", seed, logs, leaves)
		}
		cleanup()
		found = true
		break
	}

	if !found {
		t.Fatal("failed to find deterministic taiga generation seed for metadata test")
	}
}

func TestJungleTreeUsesJungleMetadataAndVineFacing(t *testing.T) {
	found := false
	for seed := int64(0); seed < 20000; seed++ {
		blocks, cleanup := newFlatGrassChunkWithMetadata()
		rng := util.NewJavaRandom(seed)
		if !generateJungleTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8, 4, true) {
			cleanup()
			continue
		}

		logs := countBlockWithMetadata(blocks, 0, 0, blockIDLog, 3)
		leaves := countBlockWithMetadata(blocks, 0, 0, blockIDLeaves, 3)
		vines := countBlockID(blocks, 0, 0, blockIDVine)
		if logs == 0 || leaves == 0 || vines == 0 {
			cleanup()
			continue
		}
		if !hasOnlyValidVineMetadata(blocks, 0, 0) {
			cleanup()
			t.Fatalf("seed=%d: jungle vines contain invalid facing metadata", seed)
		}

		cleanup()
		found = true
		break
	}

	if !found {
		t.Fatal("failed to find deterministic jungle seed with logs/leaves/vines metadata")
	}
}

func TestWorldVinesGeneratorWritesFacingMetadata(t *testing.T) {
	blocks, cleanup := newFlatGrassChunkWithMetadata()
	defer cleanup()
	for y := 64; y < worldHeightLegacyY; y++ {
		_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, 9, y, 8, blockIDStone)
	}

	rng := util.NewJavaRandom(12002)
	if !generateWorldVinesAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
		t.Fatal("expected world vines generation to return true")
	}
	if countBlockID(blocks, 0, 0, blockIDVine) == 0 {
		t.Fatal("expected world vines generator to place vines")
	}
	if !hasOnlyValidVineMetadata(blocks, 0, 0) {
		t.Fatal("world vines generator wrote invalid vine metadata")
	}
}

func TestBigTreeWritesLogAxisMetadataOnBranches(t *testing.T) {
	found := false
	for seed := int64(0); seed < 20000; seed++ {
		blocks, cleanup := newFlatGrassChunkWithMetadata()
		rng := util.NewJavaRandom(seed)
		if !generateBigTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
			cleanup()
			continue
		}

		axisX := countBlockWithMetadata(blocks, 0, 0, blockIDLog, 4)
		axisZ := countBlockWithMetadata(blocks, 0, 0, blockIDLog, 8)
		if axisX > 0 || axisZ > 0 {
			cleanup()
			found = true
			break
		}
		cleanup()
	}

	if !found {
		t.Fatal("failed to find deterministic big tree seed with horizontal log axis metadata")
	}
}

func TestBigMushroomWritesCapAndStemMetadata(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDMycelium)
		}
	}
	meta := make([]byte, len(blocks))
	registerBlockMetadataBuffer(blocks, meta)
	defer unregisterBlockMetadataBuffer(blocks)

	rng := util.NewJavaRandom(9393)
	if !generateBigMushroomAtWorld(blocks, rng, 0, 0, 8, 64, 8, 0) {
		t.Fatal("expected big mushroom generation to succeed")
	}

	capMetaCount := 0
	stemMetaCount := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < worldHeightLegacyY; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDMushroomCapBrown {
					continue
				}
				m, ok := blockMetadataAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok {
					continue
				}
				if m == 10 {
					stemMetaCount++
				}
				if m >= 1 && m <= 9 {
					capMetaCount++
				}
			}
		}
	}

	if stemMetaCount == 0 || capMetaCount == 0 {
		t.Fatalf("expected big mushroom stem/cap metadata, got stem(meta=10)=%d cap(meta=1..9)=%d", stemMetaCount, capMetaCount)
	}
}

func TestJungleGrassCanGenerateFernMetadata(t *testing.T) {
	found := false
	for seed := int64(0); seed < 5000; seed++ {
		blocks, cleanup := newFlatGrassChunkWithMetadata()
		rng := util.NewJavaRandom(seed)

		for i := 0; i < 8; i++ {
			_ = generateBiomeTallGrassPatchAtWorld(JungleBiome, blocks, rng, 0, 0, 8, 70, 8)
		}

		fernCount := countBlockWithMetadata(blocks, 0, 0, blockIDTallGrass, 2)
		cleanup()
		if fernCount > 0 {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("failed to find deterministic jungle grass seed with fern metadata")
	}
}
