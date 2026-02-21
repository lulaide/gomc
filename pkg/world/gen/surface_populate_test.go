package gen

import (
	"testing"

	"github.com/lulaide/gomc/pkg/util"
)

func TestGenerateFlowerPatchAtWorldPlacesOnGrass(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDGrass)
		}
	}

	rng := util.NewJavaRandom(1234)
	generateFlowerPatchAtWorld(blocks, rng, 0, 0, 8, 64, 8, blockIDFlowerYellow)

	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDFlowerYellow {
					continue
				}
				below, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y-1, z)
				if !ok || below != blockIDGrass {
					t.Fatalf("flower placed on invalid block at (%d,%d,%d): below=%d", x, y, z, below)
				}
				count++
			}
		}
	}

	if count == 0 {
		t.Fatal("expected at least one flower placement")
	}
}

func TestGenerateTallGrassPatchAtWorldPlacesOnGrass(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDGrass)
		}
	}

	rng := util.NewJavaRandom(4321)
	generateTallGrassPatchAtWorld(blocks, rng, 0, 0, 8, 70, 8, 1)

	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDTallGrass {
					continue
				}
				below, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y-1, z)
				if !ok || below != blockIDGrass {
					t.Fatalf("tall grass placed on invalid block at (%d,%d,%d): below=%d", x, y, z, below)
				}
				count++
			}
		}
	}

	if count == 0 {
		t.Fatal("expected at least one tall grass placement")
	}
}

func TestGenerateReedPatchAtWorldRequiresWaterAdjacency(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			baseID := blockIDGrass
			if x%2 == 0 {
				baseID = blockIDWater
			}
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 62, z, baseID)
		}
	}

	rng := util.NewJavaRandom(9876)
	generateReedPatchAtWorld(blocks, rng, 0, 0, 8, 63, 8)

	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDReed {
					continue
				}
				if !canReedStayAtWorld(blocks, 0, 0, x, y, z) {
					t.Fatalf("reed placed at invalid location (%d,%d,%d)", x, y, z)
				}
				count++
			}
		}
	}

	if count == 0 {
		t.Fatal("expected at least one reed placement")
	}
}

func TestGeneratePumpkinPatchAtWorldPlacesOnGrass(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDGrass)
		}
	}

	rng := util.NewJavaRandom(2468)
	generatePumpkinPatchAtWorld(blocks, rng, 0, 0, 8, 64, 8)

	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDPumpkin {
					continue
				}
				below, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y-1, z)
				if !ok || below != blockIDGrass {
					t.Fatalf("pumpkin placed on invalid block at (%d,%d,%d): below=%d", x, y, z, below)
				}
				count++
			}
		}
	}

	if count == 0 {
		t.Fatal("expected at least one pumpkin placement")
	}
}

func TestGenerateTreeAtWorldPlacesLogAndLeaves(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDGrass)
		}
	}

	rng := util.NewJavaRandom(13579)
	if !generateTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8, 4) {
		t.Fatal("expected tree generation to succeed on flat grass")
	}

	base, ok := blockAtWorldInTargetChunk(blocks, 0, 0, 8, 63, 8)
	if !ok || base != blockIDDirt {
		t.Fatalf("expected trunk base to be dirt, got=%d ok=%t", base, ok)
	}

	logCount := 0
	leafCount := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok {
					continue
				}
				if id == blockIDLog {
					logCount++
				}
				if id == blockIDLeaves {
					leafCount++
				}
			}
		}
	}

	if logCount == 0 {
		t.Fatal("expected at least one log block from tree generation")
	}
	if leafCount == 0 {
		t.Fatal("expected at least one leaves block from tree generation")
	}
}

func TestGenerateDeadBushPatchAtWorldPlacesOnSand(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDSand)
		}
	}

	rng := util.NewJavaRandom(7777)
	for i := 0; i < 20; i++ {
		generateDeadBushPatchAtWorld(blocks, rng, 0, 0, 8, 70, 8)
	}

	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDDeadBush {
					continue
				}
				below, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y-1, z)
				if !ok || below != blockIDSand {
					t.Fatalf("dead bush placed on invalid block at (%d,%d,%d): below=%d", x, y, z, below)
				}
				count++
			}
		}
	}

	if count == 0 {
		t.Fatal("expected at least one dead bush placement")
	}
}

func TestGenerateCactusPatchAtWorldPlacesWithVanillaConstraints(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDSand)
		}
	}

	rng := util.NewJavaRandom(8888)
	for i := 0; i < 40; i++ {
		generateCactusPatchAtWorld(blocks, rng, 0, 0, 8, 64, 8)
	}

	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDCactus {
					continue
				}
				if !canCactusStayAtWorld(blocks, 0, 0, x, y, z) {
					t.Fatalf("cactus placed at invalid location (%d,%d,%d)", x, y, z)
				}
				count++
			}
		}
	}

	if count == 0 {
		t.Fatal("expected at least one cactus placement")
	}
}

func TestGenerateMushroomPatchAtWorldPlacesOnMycelium(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDMycelium)
		}
	}

	rng := util.NewJavaRandom(9191)
	for i := 0; i < 20; i++ {
		generateMushroomPatchAtWorld(blocks, rng, 0, 0, 8, 64, 8, blockIDMushroomBrown)
	}

	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDMushroomBrown {
					continue
				}
				if !canMushroomStayAtWorld(blocks, 0, 0, x, y, z) {
					t.Fatalf("mushroom placed at invalid location (%d,%d,%d)", x, y, z)
				}
				count++
			}
		}
	}
	if count == 0 {
		t.Fatal("expected at least one mushroom placement")
	}
}

func TestGenerateWaterLilyPatchAtWorldPlacesOnStillWater(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 62, z, blockIDWater)
		}
	}

	rng := util.NewJavaRandom(9292)
	for i := 0; i < 20; i++ {
		generateWaterLilyPatchAtWorld(blocks, rng, 0, 0, 8, 63, 8)
	}

	count := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok || id != blockIDWaterLily {
					continue
				}
				if !canWaterLilyStayAtWorld(blocks, 0, 0, x, y, z) {
					t.Fatalf("waterlily placed at invalid location (%d,%d,%d)", x, y, z)
				}
				count++
			}
		}
	}
	if count == 0 {
		t.Fatal("expected at least one waterlily placement")
	}
}

func TestGenerateBigMushroomAtWorldPlacesCapBlocks(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDMycelium)
		}
	}

	rng := util.NewJavaRandom(9393)
	if !generateBigMushroomAtWorld(blocks, rng, 0, 0, 8, 64, 8, 0) {
		t.Fatal("expected big mushroom generation to succeed")
	}

	capCount := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok {
					continue
				}
				if id == blockIDMushroomCapBrown {
					capCount++
				}
			}
		}
	}

	if capCount == 0 {
		t.Fatal("expected big mushroom cap blocks to be placed")
	}
}

func TestGenerateForestTreeAtWorldPlacesLogsAndLeaves(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDGrass)
		}
	}

	rng := util.NewJavaRandom(9494)
	if !generateForestTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
		t.Fatal("expected forest tree generation to succeed")
	}

	logCount := 0
	leafCount := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok {
					continue
				}
				if id == blockIDLog {
					logCount++
				}
				if id == blockIDLeaves {
					leafCount++
				}
			}
		}
	}
	if logCount == 0 || leafCount == 0 {
		t.Fatalf("expected forest tree logs/leaves, got logs=%d leaves=%d", logCount, leafCount)
	}
}

func TestGenerateSwampTreeAtWorldPlacesLogsAndLeaves(t *testing.T) {
	blocks := make([]byte, 32768)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			_ = setBlockAtWorldInTargetChunk(blocks, 0, 0, x, 63, z, blockIDGrass)
		}
	}

	rng := util.NewJavaRandom(9595)
	if !generateSwampTreeAtWorld(blocks, rng, 0, 0, 8, 64, 8) {
		t.Fatal("expected swamp tree generation to succeed")
	}

	logCount := 0
	leafCount := 0
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 1; y < 128; y++ {
				id, ok := blockAtWorldInTargetChunk(blocks, 0, 0, x, y, z)
				if !ok {
					continue
				}
				if id == blockIDLog {
					logCount++
				}
				if id == blockIDLeaves {
					leafCount++
				}
			}
		}
	}
	if logCount == 0 || leafCount == 0 {
		t.Fatalf("expected swamp tree logs/leaves, got logs=%d leaves=%d", logCount, leafCount)
	}
}
