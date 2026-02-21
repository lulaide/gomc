package gen

import "testing"

func TestChunkProviderGenerateDeterministic(t *testing.T) {
	providerA := NewChunkProviderGenerate(12345, NewFixedBiomeSource(PlainsBiome))
	providerB := NewChunkProviderGenerate(12345, NewFixedBiomeSource(PlainsBiome))

	chA := providerA.GenerateChunk(0, 0)
	chB := providerB.GenerateChunk(0, 0)

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < 128; y++ {
				idA := chA.GetBlockID(x, y, z)
				idB := chB.GetBlockID(x, y, z)
				if idA != idB {
					t.Fatalf("determinism mismatch at (%d,%d,%d): a=%d b=%d", x, y, z, idA, idB)
				}
			}
		}
	}
}

func TestChunkProviderGenerateVariesBySeed(t *testing.T) {
	providerA := NewChunkProviderGenerate(11111, NewFixedBiomeSource(PlainsBiome))
	providerB := NewChunkProviderGenerate(22222, NewFixedBiomeSource(PlainsBiome))

	chA := providerA.GenerateChunk(0, 0)
	chB := providerB.GenerateChunk(0, 0)

	foundDiff := false
	for x := 0; x < 16 && !foundDiff; x++ {
		for z := 0; z < 16 && !foundDiff; z++ {
			for y := 0; y < 128; y++ {
				if chA.GetBlockID(x, y, z) != chB.GetBlockID(x, y, z) {
					foundDiff = true
					break
				}
			}
		}
	}

	if !foundDiff {
		t.Fatal("expected terrain to differ for different seeds")
	}
}

func TestChunkProviderGenerateBiomeArrayFromSource(t *testing.T) {
	provider := NewChunkProviderGenerate(123, NewFixedBiomeSource(PlainsBiome))
	ch := provider.GenerateChunk(2, -3)

	biomes := ch.GetBiomeArray()
	for i, id := range biomes {
		if id != PlainsBiome.ID {
			t.Fatalf("biome mismatch at idx=%d: got=%d want=%d", i, id, PlainsBiome.ID)
		}
	}
}

func TestChunkProviderGenerateUndergroundFeatures(t *testing.T) {
	provider := NewChunkProviderGenerate(12345, NewFixedBiomeSource(PlainsBiome))
	oreIDs := map[int]struct{}{
		int(blockIDCoalOre):     {},
		int(blockIDIronOre):     {},
		int(blockIDGoldOre):     {},
		int(blockIDRedstoneOre): {},
		int(blockIDDiamondOre):  {},
		int(blockIDLapisOre):    {},
	}

	foundOre := false
	foundCaveAir := false

	for chunkX := int32(-3); chunkX <= 3 && !(foundOre && foundCaveAir); chunkX++ {
		for chunkZ := int32(-3); chunkZ <= 3 && !(foundOre && foundCaveAir); chunkZ++ {
			ch := provider.GenerateChunk(chunkX, chunkZ)
			for x := 0; x < 16 && !(foundOre && foundCaveAir); x++ {
				for z := 0; z < 16 && !(foundOre && foundCaveAir); z++ {
					for y := 5; y < 64 && !(foundOre && foundCaveAir); y++ {
						id := ch.GetBlockID(x, y, z)
						if _, ok := oreIDs[id]; ok {
							foundOre = true
						}
						// Ignore deep lava floor and only consider carved underground air.
						if id == int(blockIDAir) && y >= 11 {
							foundCaveAir = true
						}
					}
				}
			}
		}
	}

	if !foundOre {
		t.Fatal("expected at least one ore block in sampled chunks")
	}
	if !foundCaveAir {
		t.Fatal("expected carved underground air in sampled chunks")
	}
}

func TestChunkProviderGenerateSurfaceDecorations(t *testing.T) {
	provider := NewChunkProviderGenerate(24680, NewFixedBiomeSource(PlainsBiome))

	foundTallGrass := false
	foundFlower := false
	foundClay := false

	for chunkX := int32(-6); chunkX <= 6 && !(foundTallGrass && foundFlower && foundClay); chunkX++ {
		for chunkZ := int32(-6); chunkZ <= 6 && !(foundTallGrass && foundFlower && foundClay); chunkZ++ {
			ch := provider.GenerateChunk(chunkX, chunkZ)
			for x := 0; x < 16 && !(foundTallGrass && foundFlower && foundClay); x++ {
				for z := 0; z < 16 && !(foundTallGrass && foundFlower && foundClay); z++ {
					for y := 1; y < 128 && !(foundTallGrass && foundFlower && foundClay); y++ {
						id := ch.GetBlockID(x, y, z)
						switch id {
						case int(blockIDTallGrass):
							foundTallGrass = true
						case int(blockIDFlowerYellow), int(blockIDFlowerRed):
							foundFlower = true
						case int(blockIDClay):
							foundClay = true
						}
					}
				}
			}
		}
	}

	if !foundTallGrass {
		t.Fatal("expected at least one tall grass block in sampled chunks")
	}
	if !foundFlower {
		t.Fatal("expected at least one flower block in sampled chunks")
	}
	if !foundClay {
		t.Fatal("expected at least one clay patch block in sampled chunks")
	}
}
