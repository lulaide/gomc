package gen

import "github.com/lulaide/gomc/pkg/util"

type biomeDecoratorConfig struct {
	sandPerChunk         int
	sandPerChunk2        int
	clayPerChunk         int
	treesPerChunk        int
	bigMushroomsPerChunk int
	flowersPerChunk      int
	grassPerChunk        int
	deadBushPerChunk     int
	mushroomsPerChunk    int
	waterlilyPerChunk    int
	cactiPerChunk        int
	reedsPerChunk        int
}

func defaultBiomeDecoratorConfig() biomeDecoratorConfig {
	// Translation reference:
	// - net.minecraft.src.BiomeDecorator.BiomeDecorator(...)
	return biomeDecoratorConfig{
		sandPerChunk:         1,
		sandPerChunk2:        3,
		clayPerChunk:         1,
		treesPerChunk:        0,
		bigMushroomsPerChunk: 0,
		flowersPerChunk:      2,
		grassPerChunk:        1,
		deadBushPerChunk:     0,
		mushroomsPerChunk:    0,
		waterlilyPerChunk:    0,
		cactiPerChunk:        0,
		reedsPerChunk:        0,
	}
}

func biomeDecoratorConfigFor(biome Biome) biomeDecoratorConfig {
	cfg := defaultBiomeDecoratorConfig()

	switch biome.ID {
	// Translation reference:
	// - net.minecraft.src.BiomeGenPlains.BiomeGenPlains(...)
	case BiomeIDPlains:
		cfg.treesPerChunk = -999
		cfg.flowersPerChunk = 4
		cfg.grassPerChunk = 10
	// Translation reference:
	// - net.minecraft.src.BiomeGenForest.BiomeGenForest(...)
	// - net.minecraft.src.BiomeGenTaiga.BiomeGenTaiga(...)
	case BiomeIDForest, BiomeIDForestHills:
		cfg.treesPerChunk = 10
		cfg.grassPerChunk = 2
	case BiomeIDTaiga, BiomeIDTaigaHills:
		cfg.treesPerChunk = 10
		cfg.grassPerChunk = 1
	// Translation reference:
	// - net.minecraft.src.BiomeGenSwamp.BiomeGenSwamp(...)
	case BiomeIDSwampland:
		cfg.treesPerChunk = 2
		cfg.flowersPerChunk = -999
		cfg.deadBushPerChunk = 1
		cfg.mushroomsPerChunk = 8
		cfg.reedsPerChunk = 10
		cfg.clayPerChunk = 1
		cfg.waterlilyPerChunk = 4
	// Translation reference:
	// - net.minecraft.src.BiomeGenDesert.BiomeGenDesert(...)
	case BiomeIDDesert, BiomeIDDesertHills:
		cfg.treesPerChunk = -999
		cfg.deadBushPerChunk = 2
		cfg.reedsPerChunk = 50
		cfg.cactiPerChunk = 10
	// Translation reference:
	// - net.minecraft.src.BiomeGenBeach.BiomeGenBeach(...)
	case BiomeIDBeach:
		cfg.treesPerChunk = -999
		cfg.deadBushPerChunk = 0
		cfg.reedsPerChunk = 0
		cfg.cactiPerChunk = 0
	// Translation reference:
	// - net.minecraft.src.BiomeGenJungle.BiomeGenJungle(...)
	case BiomeIDJungle, BiomeIDJungleHills:
		cfg.treesPerChunk = 50
		cfg.flowersPerChunk = 4
		cfg.grassPerChunk = 25
	// Translation reference:
	// - net.minecraft.src.BiomeGenMushroomIsland.BiomeGenMushroomIsland(...)
	case BiomeIDMushroomIsland, BiomeIDMushroomIslandShore:
		cfg.treesPerChunk = -100
		cfg.flowersPerChunk = -100
		cfg.grassPerChunk = -100
		cfg.mushroomsPerChunk = 1
		cfg.bigMushroomsPerChunk = 1
	}
	return cfg
}

func (p *ChunkProviderGenerate) biomeAt(worldX, worldZ int) Biome {
	if p == nil || p.biomeSource == nil {
		return PlainsBiome
	}
	biomes := p.biomeSource.LoadBlockGeneratorData(nil, worldX, worldZ, 1, 1)
	if len(biomes) == 0 {
		return PlainsBiome
	}
	return biomes[0]
}

// Translation subset reference:
// - net.minecraft.src.ChunkProviderGenerate.populate(...)
// - net.minecraft.src.BiomeDecorator.decorate(...) sand/clay/flowers/grass loops
func (p *ChunkProviderGenerate) populateSurface(targetChunkX, targetChunkZ int, blocks []byte) {
	if p == nil || len(blocks) == 0 {
		return
	}

	rng := util.NewJavaRandom(p.worldSeed)
	xMul := (rng.NextLong()/2)*2 + 1
	zMul := (rng.NextLong()/2)*2 + 1

	// Surface decoration from adjacent populate chunks can place into this chunk.
	for popChunkX := targetChunkX - 1; popChunkX <= targetChunkX+1; popChunkX++ {
		for popChunkZ := targetChunkZ - 1; popChunkZ <= targetChunkZ+1; popChunkZ++ {
			seed := (int64(popChunkX)*xMul + int64(popChunkZ)*zMul) ^ p.worldSeed
			rng.SetSeed(seed)
			p.populateSurfaceForPopulateChunk(rng, popChunkX, popChunkZ, targetChunkX, targetChunkZ, blocks)
		}
	}
}

func (p *ChunkProviderGenerate) populateSurfaceForPopulateChunk(
	rng *util.JavaRandom,
	popChunkX, popChunkZ int,
	targetChunkX, targetChunkZ int,
	blocks []byte,
) {
	if rng == nil {
		return
	}

	baseX := popChunkX * 16
	baseZ := popChunkZ * 16
	biome := p.biomeAt(baseX+16, baseZ+16)
	cfg := biomeDecoratorConfigFor(biome)

	for i := 0; i < cfg.sandPerChunk2; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := topSolidOrLiquidAtWorld(blocks, targetChunkX, targetChunkZ, x, z)
		if y >= 0 {
			generateSandPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDSand, 7)
		}
	}

	for i := 0; i < cfg.clayPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := topSolidOrLiquidAtWorld(blocks, targetChunkX, targetChunkZ, x, z)
		if y >= 0 {
			generateClayPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, 4)
		}
	}

	for i := 0; i < cfg.sandPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := topSolidOrLiquidAtWorld(blocks, targetChunkX, targetChunkZ, x, z)
		if y >= 0 {
			generateSandPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDGravel, 6)
		}
	}

	treeAttempts := cfg.treesPerChunk
	if rng.NextInt(10) == 0 {
		treeAttempts++
	}
	for i := 0; i < treeAttempts; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := heightValueAtWorld(blocks, targetChunkX, targetChunkZ, x, z)
		generateBiomeTreeAtWorld(biome, blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}

	for i := 0; i < cfg.bigMushroomsPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := heightValueAtWorld(blocks, targetChunkX, targetChunkZ, x, z)
		generateBigMushroomAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, -1)
	}

	for i := 0; i < cfg.flowersPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		z := baseZ + int(rng.NextInt(16)) + 8
		generateFlowerPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDFlowerYellow)

		if rng.NextInt(4) == 0 {
			x = baseX + int(rng.NextInt(16)) + 8
			y = int(rng.NextInt(worldHeightLegacyY))
			z = baseZ + int(rng.NextInt(16)) + 8
			generateFlowerPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDFlowerRed)
		}
	}

	for i := 0; i < cfg.grassPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		z := baseZ + int(rng.NextInt(16)) + 8
		generateBiomeTallGrassPatchAtWorld(biome, blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}

	for i := 0; i < cfg.deadBushPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		z := baseZ + int(rng.NextInt(16)) + 8
		generateDeadBushPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}

	for i := 0; i < cfg.waterlilyPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		for y > 0 {
			id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y-1, z)
			if !ok || id != blockIDAir {
				break
			}
			y--
		}
		generateWaterLilyPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}

	for i := 0; i < cfg.mushroomsPerChunk; i++ {
		if rng.NextInt(4) == 0 {
			x := baseX + int(rng.NextInt(16)) + 8
			z := baseZ + int(rng.NextInt(16)) + 8
			y := heightValueAtWorld(blocks, targetChunkX, targetChunkZ, x, z)
			generateMushroomPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDMushroomBrown)
		}
		if rng.NextInt(8) == 0 {
			x := baseX + int(rng.NextInt(16)) + 8
			z := baseZ + int(rng.NextInt(16)) + 8
			y := int(rng.NextInt(worldHeightLegacyY))
			generateMushroomPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDMushroomRed)
		}
	}

	if rng.NextInt(4) == 0 {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		generateMushroomPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDMushroomBrown)
	}

	if rng.NextInt(8) == 0 {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		generateMushroomPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDMushroomRed)
	}

	for i := 0; i < cfg.reedsPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		generateReedPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}
	for i := 0; i < 10; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		z := baseZ + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		generateReedPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}

	if rng.NextInt(32) == 0 {
		x := baseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		z := baseZ + int(rng.NextInt(16)) + 8
		generatePumpkinPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}

	for i := 0; i < cfg.cactiPerChunk; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		z := baseZ + int(rng.NextInt(16)) + 8
		generateCactusPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}

	// Translation reference:
	// - net.minecraft.src.BiomeGenJungle.decorate(...)
	if biome.ID == BiomeIDJungle || biome.ID == BiomeIDJungleHills {
		for i := 0; i < 50; i++ {
			x := baseX + int(rng.NextInt(16)) + 8
			z := baseZ + int(rng.NextInt(16)) + 8
			generateWorldVinesAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, 64, z)
		}
	}
}

// Translation reference:
// - net.minecraft.src.World#getTopSolidOrLiquidBlock(...)
func topSolidOrLiquidAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, worldZ int) int {
	for y := worldHeightLegacyY - 1; y > 0; y-- {
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
		if !ok {
			return -1
		}
		if id != blockIDAir && blocksMovementForTopSolidOrLiquid(id) && id != blockIDLeaves {
			return y + 1
		}
	}
	return -1
}

func blocksMovementForTopSolidOrLiquid(id byte) bool {
	switch id {
	case blockIDAir,
		blockIDWaterFlow, blockIDWater,
		blockIDLavaFlow, blockIDLava,
		blockIDTallGrass, blockIDFlowerYellow, blockIDFlowerRed:
		return false
	default:
		return true
	}
}

// Translation reference:
// - net.minecraft.src.Block.opaqueCubeLookup[id] reads used by world generators.
func isOpaqueCubeForGen(id byte) bool {
	switch id {
	case blockIDAir,
		blockIDWaterFlow, blockIDWater,
		blockIDLavaFlow, blockIDLava,
		blockIDTallGrass, blockIDDeadBush,
		blockIDFlowerYellow, blockIDFlowerRed,
		blockIDLeaves, blockIDReed,
		blockIDMushroomBrown, blockIDMushroomRed,
		blockIDWaterLily, blockIDCactus:
		return false
	default:
		return true
	}
}

// Translation reference:
// - net.minecraft.src.World#getHeightValue(...)
func heightValueAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, worldZ int) int {
	for y := worldHeightLegacyY - 1; y >= 0; y-- {
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
		if !ok {
			return 0
		}
		if id != blockIDAir {
			return y + 1
		}
	}
	return 0
}

func generateBiomeTreeAtWorld(
	biome Biome,
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	switch biome.ID {
	// Translation reference:
	// - net.minecraft.src.BiomeGenSwamp.getRandomWorldGenForTrees(...)
	case BiomeIDSwampland:
		return generateSwampTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ)
	// Translation reference:
	// - net.minecraft.src.BiomeGenForest.getRandomWorldGenForTrees(...)
	case BiomeIDForest, BiomeIDForestHills:
		if rng.NextInt(5) == 0 {
			return generateForestTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ)
		}
		if rng.NextInt(10) == 0 {
			return generateBigTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ)
		}
		return generateTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ, 4)
	// Translation reference:
	// - net.minecraft.src.BiomeGenTaiga.getRandomWorldGenForTrees(...)
	case BiomeIDTaiga, BiomeIDTaigaHills:
		if rng.NextInt(3) == 0 {
			return generateTaiga1TreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ)
		}
		return generateTaiga2TreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ)
	// Translation reference:
	// - net.minecraft.src.BiomeGenJungle.getRandomWorldGenForTrees(...)
	case BiomeIDJungle, BiomeIDJungleHills:
		if rng.NextInt(10) == 0 {
			return generateBigTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ)
		}
		if rng.NextInt(2) == 0 {
			return generateJungleShrubAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ)
		}
		if rng.NextInt(3) == 0 {
			baseHeight := 10 + int(rng.NextInt(20))
			return generateHugeJungleTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ, baseHeight)
		}
		minTreeHeight := 4 + int(rng.NextInt(7))
		return generateJungleTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ, minTreeHeight, true)
	default:
		// Translation reference:
		// - net.minecraft.src.BiomeGenBase.getRandomWorldGenForTrees(...)
		if rng.NextInt(10) == 0 {
			return generateBigTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ)
		}
		return generateTreeAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ, 4)
	}
}

func generateBiomeTallGrassPatchAtWorld(
	biome Biome,
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	grassMeta := byte(1)
	// Translation reference:
	// - net.minecraft.src.BiomeGenJungle.getRandomWorldGenForGrass(...)
	if biome.ID == BiomeIDJungle || biome.ID == BiomeIDJungleHills {
		if rng.NextInt(4) == 0 {
			grassMeta = 2
		}
	}
	return generateTallGrassPatchAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, y, worldZ, grassMeta)
}

// Translation reference:
// - net.minecraft.src.WorldGenForest.generate(...)
func generateForestTreeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	height := int(rng.NextInt(3)) + 5
	canGrow := true

	if y < 1 || y+height+1 > worldHeightLegacyY {
		return false
	}

	for yy := y; yy <= y+1+height && canGrow; yy++ {
		radius := 1
		if yy == y {
			radius = 0
		}
		if yy >= y+1+height-2 {
			radius = 2
		}
		for xx := worldX - radius; xx <= worldX+radius && canGrow; xx++ {
			for zz := worldZ - radius; zz <= worldZ+radius && canGrow; zz++ {
				if yy < 0 || yy >= worldHeightLegacyY {
					canGrow = false
					continue
				}
				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if id != blockIDAir && id != blockIDLeaves {
					canGrow = false
				}
			}
		}
	}
	if !canGrow {
		return false
	}

	below := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ, blockIDStone)
	if (below != blockIDGrass && below != blockIDDirt) || y >= worldHeightLegacyY-height-1 {
		return false
	}
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ, blockIDDirt)

	for yy := y - 3 + height; yy <= y+height; yy++ {
		dy := yy - (y + height)
		radius := 1 - dy/2
		for xx := worldX - radius; xx <= worldX+radius; xx++ {
			dx := xx - worldX
			if dx < 0 {
				dx = -dx
			}
			for zz := worldZ - radius; zz <= worldZ+radius; zz++ {
				dz := zz - worldZ
				if dz < 0 {
					dz = -dz
				}
				if dx != radius || dz != radius || dy == 0 || rng.NextInt(2) == 0 {
					id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
					if id == blockIDAir || id == blockIDLeaves {
						_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDLeaves, 2)
					}
				}
			}
		}
	}

	for i := 0; i < height; i++ {
		yy := y + i
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDLog, 2)
		}
	}
	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenSwamp.generate(...)
func generateSwampTreeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	height := int(rng.NextInt(4)) + 5
	for blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ, blockIDAir) == blockIDWater {
		y--
	}

	canGrow := true
	if y < 1 || y+height+1 > worldHeightLegacyY {
		return false
	}

	for yy := y; yy <= y+1+height && canGrow; yy++ {
		radius := 1
		if yy == y {
			radius = 0
		}
		if yy >= y+1+height-2 {
			radius = 3
		}

		for xx := worldX - radius; xx <= worldX+radius && canGrow; xx++ {
			for zz := worldZ - radius; zz <= worldZ+radius && canGrow; zz++ {
				if yy < 0 || yy >= worldHeightLegacyY {
					canGrow = false
					continue
				}
				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if id != blockIDAir && id != blockIDLeaves {
					if id != blockIDWater && id != blockIDWaterFlow {
						canGrow = false
					} else if yy > y {
						canGrow = false
					}
				}
			}
		}
	}

	if !canGrow {
		return false
	}

	below := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ, blockIDStone)
	if (below != blockIDGrass && below != blockIDDirt) || y >= worldHeightLegacyY-height-1 {
		return false
	}
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ, blockIDDirt)

	for yy := y - 3 + height; yy <= y+height; yy++ {
		dy := yy - (y + height)
		radius := 2 - dy/2

		for xx := worldX - radius; xx <= worldX+radius; xx++ {
			dx := xx - worldX
			if dx < 0 {
				dx = -dx
			}
			for zz := worldZ - radius; zz <= worldZ+radius; zz++ {
				dz := zz - worldZ
				if dz < 0 {
					dz = -dz
				}
				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if (dx != radius || dz != radius || dy == 0 || rng.NextInt(2) == 0) && !isOpaqueCubeForGen(id) {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDLeaves)
				}
			}
		}
	}

	for i := 0; i < height; i++ {
		yy := y + i
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves || id == blockIDWaterFlow || id == blockIDWater {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDLog)
		}
	}

	for yy := y - 3 + height; yy <= y+height; yy++ {
		dy := yy - (y + height)
		radius := 2 - dy/2
		for xx := worldX - radius; xx <= worldX+radius; xx++ {
			for zz := worldZ - radius; zz <= worldZ+radius; zz++ {
				if blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir) != blockIDLeaves {
					continue
				}
				if rng.NextInt(4) == 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx-1, yy, zz, blockIDAir) == blockIDAir {
					generateVinesAtWorld(blocks, targetChunkX, targetChunkZ, xx-1, yy, zz, 8)
				}
				if rng.NextInt(4) == 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx+1, yy, zz, blockIDAir) == blockIDAir {
					generateVinesAtWorld(blocks, targetChunkX, targetChunkZ, xx+1, yy, zz, 2)
				}
				if rng.NextInt(4) == 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz-1, blockIDAir) == blockIDAir {
					generateVinesAtWorld(blocks, targetChunkX, targetChunkZ, xx, yy, zz-1, 1)
				}
				if rng.NextInt(4) == 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz+1, blockIDAir) == blockIDAir {
					generateVinesAtWorld(blocks, targetChunkX, targetChunkZ, xx, yy, zz+1, 4)
				}
			}
		}
	}

	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenSwamp.generateVines(...)
func generateVinesAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int, vineMeta byte) {
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDVine, vineMeta)
	vineLength := 4
	for {
		y--
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDAir)
		if id != blockIDAir || vineLength <= 0 {
			return
		}
		_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDVine, vineMeta)
		vineLength--
	}
}

// Translation reference:
// - net.minecraft.src.WorldGenTrees.generate(...)
func generateTreeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	minTreeHeight int,
) bool {
	if rng == nil {
		return false
	}

	height := int(rng.NextInt(3)) + minTreeHeight
	if y < 1 || y+height+1 > worldHeightLegacyY {
		return false
	}

	canGrow := true
	for yy := y; yy <= y+1+height && canGrow; yy++ {
		radius := 1
		if yy == y {
			radius = 0
		}
		if yy >= y+height-1 {
			radius = 2
		}

		for xx := worldX - radius; xx <= worldX+radius && canGrow; xx++ {
			for zz := worldZ - radius; zz <= worldZ+radius && canGrow; zz++ {
				if yy < 0 || yy >= worldHeightLegacyY {
					canGrow = false
					continue
				}
				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if id != blockIDAir &&
					id != blockIDLeaves &&
					id != blockIDGrass &&
					id != blockIDDirt &&
					id != blockIDLog &&
					id != blockIDSapling {
					canGrow = false
				}
			}
		}
	}
	if !canGrow {
		return false
	}

	below := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ, blockIDStone)
	if (below != blockIDGrass && below != blockIDDirt) || y >= worldHeightLegacyY-height-1 {
		return false
	}

	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ, blockIDDirt)

	for yy := y - 3 + height; yy <= y+height; yy++ {
		dy := yy - (y + height)
		radius := 1 - dy/2

		for xx := worldX - radius; xx <= worldX+radius; xx++ {
			dx := xx - worldX
			if dx < 0 {
				dx = -dx
			}
			for zz := worldZ - radius; zz <= worldZ+radius; zz++ {
				dz := zz - worldZ
				if dz < 0 {
					dz = -dz
				}

				if (dx == radius && dz == radius) && (rng.NextInt(2) == 0 || dy == 0) {
					continue
				}

				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if id == blockIDAir || id == blockIDLeaves {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDLeaves)
				}
			}
		}
	}

	for i := 0; i < height; i++ {
		yy := y + i
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDLog)
		}
	}
	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenBigMushroom.generate(...)
func generateBigMushroomAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	mushroomType int,
) bool {
	if rng == nil {
		return false
	}

	mType := int(rng.NextInt(2))
	if mushroomType >= 0 {
		mType = mushroomType
	}

	height := int(rng.NextInt(3)) + 4
	canGrow := true

	if y < 1 || y+height+1 >= worldHeightLegacyY {
		return false
	}

	for yy := y; yy <= y+1+height && canGrow; yy++ {
		radius := 3
		if yy <= y+3 {
			radius = 0
		}

		for xx := worldX - radius; xx <= worldX+radius && canGrow; xx++ {
			for zz := worldZ - radius; zz <= worldZ+radius && canGrow; zz++ {
				if yy < 0 || yy >= worldHeightLegacyY {
					canGrow = false
					continue
				}

				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if id != blockIDAir && id != blockIDLeaves {
					canGrow = false
				}
			}
		}
	}

	if !canGrow {
		return false
	}

	below := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ, blockIDStone)
	if below != blockIDDirt && below != blockIDGrass && below != blockIDMycelium {
		return false
	}

	capStartY := y + height
	if mType == 1 {
		capStartY = y + height - 3
	}

	capBlockID := blockIDMushroomCapBrown
	if mType == 1 {
		capBlockID = blockIDMushroomCapRed
	}

	for yy := capStartY; yy <= y+height; yy++ {
		radius := 1
		if yy < y+height {
			radius++
		}
		if mType == 0 {
			radius = 3
		}

		for xx := worldX - radius; xx <= worldX+radius; xx++ {
			for zz := worldZ - radius; zz <= worldZ+radius; zz++ {
				meta := 5
				if xx == worldX-radius {
					meta--
				}
				if xx == worldX+radius {
					meta++
				}
				if zz == worldZ-radius {
					meta -= 3
				}
				if zz == worldZ+radius {
					meta += 3
				}

				if mType == 0 || yy < y+height {
					if (xx == worldX-radius || xx == worldX+radius) && (zz == worldZ-radius || zz == worldZ+radius) {
						continue
					}

					if xx == worldX-(radius-1) && zz == worldZ-radius {
						meta = 1
					}
					if xx == worldX-radius && zz == worldZ-(radius-1) {
						meta = 1
					}
					if xx == worldX+(radius-1) && zz == worldZ-radius {
						meta = 3
					}
					if xx == worldX+radius && zz == worldZ-(radius-1) {
						meta = 3
					}
					if xx == worldX-(radius-1) && zz == worldZ+radius {
						meta = 7
					}
					if xx == worldX-radius && zz == worldZ+(radius-1) {
						meta = 7
					}
					if xx == worldX+(radius-1) && zz == worldZ+radius {
						meta = 9
					}
					if xx == worldX+radius && zz == worldZ+(radius-1) {
						meta = 9
					}
				}

				if meta == 5 && yy < y+height {
					meta = 0
				}

				// Keep MCP condition form as-is, including the original always-false
				// right term `(par4 >= par4 + var7 - 1)` from 1.6.4 decompilation.
				if (meta != 0 || y >= y+height-1) &&
					!isOpaqueCubeForGen(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)) {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, capBlockID, byte(meta))
				}
			}
		}
	}

	for i := 0; i < height; i++ {
		yy := y + i
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDAir)
		if !isOpaqueCubeForGen(id) {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, capBlockID, 10)
		}
	}
	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenSand.generate(...)
func generateSandPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	sandID byte,
	radius int,
) bool {
	if rng == nil || radius <= 2 {
		return false
	}

	center, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
	if !ok || (center != blockIDWaterFlow && center != blockIDWater) {
		return false
	}

	r := int(rng.NextInt(radius-2)) + 2
	const yRadius = 2
	placed := false
	for x := worldX - r; x <= worldX+r; x++ {
		for z := worldZ - r; z <= worldZ+r; z++ {
			dx := x - worldX
			dz := z - worldZ
			if dx*dx+dz*dz > r*r {
				continue
			}
			for yy := y - yRadius; yy <= y+yRadius; yy++ {
				id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
				if !ok {
					continue
				}
				if id == blockIDDirt || id == blockIDGrass {
					if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, sandID) {
						placed = true
					}
				}
			}
		}
	}
	return placed
}

// Translation reference:
// - net.minecraft.src.WorldGenClay.generate(...)
func generateClayPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	numberOfBlocks int,
) bool {
	if rng == nil || numberOfBlocks <= 2 {
		return false
	}

	center, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
	if !ok || (center != blockIDWaterFlow && center != blockIDWater) {
		return false
	}

	r := int(rng.NextInt(numberOfBlocks-2)) + 2
	const yRadius = 1
	placed := false
	for x := worldX - r; x <= worldX+r; x++ {
		for z := worldZ - r; z <= worldZ+r; z++ {
			dx := x - worldX
			dz := z - worldZ
			if dx*dx+dz*dz > r*r {
				continue
			}
			for yy := y - yRadius; yy <= y+yRadius; yy++ {
				id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
				if !ok {
					continue
				}
				if id == blockIDDirt || id == blockIDClay {
					if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDClay) {
						placed = true
					}
				}
			}
		}
	}
	return placed
}

// Translation reference:
// - net.minecraft.src.WorldGenFlowers.generate(...)
func generateFlowerPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	flowerID byte,
) bool {
	if rng == nil {
		return false
	}

	placed := false
	for i := 0; i < 64; i++ {
		x := worldX + int(rng.NextInt(8)) - int(rng.NextInt(8))
		yy := y + int(rng.NextInt(4)) - int(rng.NextInt(4))
		z := worldZ + int(rng.NextInt(8)) - int(rng.NextInt(8))
		if yy <= 0 || yy >= worldHeightLegacyY {
			continue
		}

		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
		if !ok || id != blockIDAir {
			continue
		}
		if !canFlowerStayAtWorld(blocks, targetChunkX, targetChunkZ, x, yy, z) {
			continue
		}

		if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, flowerID) {
			placed = true
		}
	}
	return placed
}

// Translation reference:
// - net.minecraft.src.WorldGenTallGrass.generate(...)
func generateTallGrassPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	grassMeta byte,
) bool {
	if rng == nil {
		return false
	}

	for y > 0 {
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDAir)
		if id != blockIDAir && id != blockIDLeaves {
			break
		}
		y--
	}

	placed := false
	for i := 0; i < 128; i++ {
		x := worldX + int(rng.NextInt(8)) - int(rng.NextInt(8))
		yy := y + int(rng.NextInt(4)) - int(rng.NextInt(4))
		z := worldZ + int(rng.NextInt(8)) - int(rng.NextInt(8))
		if yy <= 0 || yy >= worldHeightLegacyY {
			continue
		}

		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDAir)
		if id != blockIDAir {
			continue
		}
		if !canFlowerStayAtWorld(blocks, targetChunkX, targetChunkZ, x, yy, z) {
			continue
		}

		if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDTallGrass, grassMeta) {
			placed = true
		}
	}
	return placed
}

// Translation reference:
// - net.minecraft.src.WorldGenDeadBush.generate(...)
func generateDeadBushPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	for y > 0 {
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
		if !ok || (id != blockIDAir && id != blockIDLeaves) {
			break
		}
		y--
	}

	placed := false
	for i := 0; i < 4; i++ {
		x := worldX + int(rng.NextInt(8)) - int(rng.NextInt(8))
		yy := y + int(rng.NextInt(4)) - int(rng.NextInt(4))
		z := worldZ + int(rng.NextInt(8)) - int(rng.NextInt(8))
		if yy <= 0 || yy >= worldHeightLegacyY {
			continue
		}
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
		if !ok || id != blockIDAir {
			continue
		}
		if !canDeadBushStayAtWorld(blocks, targetChunkX, targetChunkZ, x, yy, z) {
			continue
		}
		if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDDeadBush) {
			placed = true
		}
	}
	return placed
}

func canDeadBushStayAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int) bool {
	if y <= 0 || y >= worldHeightLegacyY {
		return false
	}
	below, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ)
	if !ok {
		return false
	}
	return below == blockIDSand
}

// Translation reference:
// - net.minecraft.src.WorldGenFlowers.generate(...), used for mushrooms too.
func generateMushroomPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	mushroomID byte,
) bool {
	if rng == nil {
		return false
	}
	if mushroomID != blockIDMushroomBrown && mushroomID != blockIDMushroomRed {
		return false
	}

	placed := false
	for i := 0; i < 64; i++ {
		x := worldX + int(rng.NextInt(8)) - int(rng.NextInt(8))
		yy := y + int(rng.NextInt(4)) - int(rng.NextInt(4))
		z := worldZ + int(rng.NextInt(8)) - int(rng.NextInt(8))
		if yy <= 0 || yy >= worldHeightLegacyY {
			continue
		}

		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
		if !ok || id != blockIDAir {
			continue
		}
		if !canMushroomStayAtWorld(blocks, targetChunkX, targetChunkZ, x, yy, z) {
			continue
		}

		if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, mushroomID) {
			placed = true
		}
	}
	return placed
}

// Translation reference:
// - net.minecraft.src.BlockMushroom.canBlockStay(...)
func canMushroomStayAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int) bool {
	if y <= 0 || y >= worldHeightLegacyY {
		return false
	}
	below, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ)
	if !ok {
		return false
	}
	if below == blockIDMycelium {
		return true
	}
	// In generation buffer we do not have full block-light data; use sky-visibility
	// as an equivalent bright/dark discriminator for mushroom placement.
	if canBlockSeeSkyAtWorld(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ) {
		return false
	}
	return canMushroomGrowOnBlockID(below)
}

// Translation reference:
// - net.minecraft.src.BlockMushroom.canThisPlantGrowOnThisBlockID(...)
func canMushroomGrowOnBlockID(id byte) bool {
	switch id {
	case blockIDAir,
		blockIDWaterFlow, blockIDWater,
		blockIDLavaFlow, blockIDLava,
		blockIDTallGrass, blockIDFlowerYellow, blockIDFlowerRed,
		blockIDLeaves, blockIDReed, blockIDMushroomBrown, blockIDMushroomRed, blockIDWaterLily:
		return false
	default:
		return true
	}
}

// Translation reference:
// - net.minecraft.src.WorldGenWaterlily.generate(...)
func generateWaterLilyPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	placed := false
	for i := 0; i < 10; i++ {
		x := worldX + int(rng.NextInt(8)) - int(rng.NextInt(8))
		yy := y + int(rng.NextInt(4)) - int(rng.NextInt(4))
		z := worldZ + int(rng.NextInt(8)) - int(rng.NextInt(8))
		if yy <= 0 || yy >= worldHeightLegacyY {
			continue
		}

		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
		if !ok || id != blockIDAir {
			continue
		}
		if !canWaterLilyStayAtWorld(blocks, targetChunkX, targetChunkZ, x, yy, z) {
			continue
		}
		if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDWaterLily) {
			placed = true
		}
	}
	return placed
}

// Translation reference:
// - net.minecraft.src.BlockLilyPad.canBlockStay(...)
func canWaterLilyStayAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int) bool {
	if y <= 0 || y >= worldHeightLegacyY {
		return false
	}
	below, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ)
	if !ok {
		return false
	}
	return below == blockIDWater
}

func canFlowerStayAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int) bool {
	if y <= 0 || y >= worldHeightLegacyY {
		return false
	}
	below, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ)
	if !ok || !canPlantGrowOnBlockID(below) {
		return false
	}

	// Translation note:
	// BlockFlower#canBlockStay uses (full light >= 8 || can see sky). During
	// terrain generation here, only sky light is modeled in this local buffer.
	return canBlockSeeSkyAtWorld(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
}

func canPlantGrowOnBlockID(id byte) bool {
	return id == blockIDGrass || id == blockIDDirt || id == blockIDFarmland
}

// Translation reference:
// - net.minecraft.src.WorldGenReed.generate(...)
func generateReedPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	placed := false
	for i := 0; i < 20; i++ {
		x := worldX + int(rng.NextInt(4)) - int(rng.NextInt(4))
		z := worldZ + int(rng.NextInt(4)) - int(rng.NextInt(4))
		yy := y

		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
		if !ok || id != blockIDAir {
			continue
		}
		if !hasAdjacentWaterAtYMinusOne(blocks, targetChunkX, targetChunkZ, x, yy, z) {
			continue
		}

		height := 2 + int(rng.NextInt(int(rng.NextInt(3)+1)))
		for h := 0; h < height; h++ {
			if canReedStayAtWorld(blocks, targetChunkX, targetChunkZ, x, yy+h, z) {
				if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy+h, z, blockIDReed) {
					placed = true
				}
			}
		}
	}
	return placed
}

func hasAdjacentWaterAtYMinusOne(blocks []byte, targetChunkX, targetChunkZ int, x, y, z int) bool {
	adjacent := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, off := range adjacent {
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x+off[0], y-1, z+off[1])
		if !ok {
			continue
		}
		if id == blockIDWaterFlow || id == blockIDWater {
			return true
		}
	}
	return false
}

// Translation reference:
// - net.minecraft.src.BlockReed.canPlaceBlockAt(...)
func canReedStayAtWorld(blocks []byte, targetChunkX, targetChunkZ int, x, y, z int) bool {
	if y <= 0 || y >= worldHeightLegacyY {
		return false
	}
	below, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y-1, z)
	if !ok {
		return false
	}
	if below == blockIDReed {
		return true
	}
	if below != blockIDGrass && below != blockIDDirt && below != blockIDSand {
		return false
	}
	return hasAdjacentWaterAtYMinusOne(blocks, targetChunkX, targetChunkZ, x, y, z)
}

// Translation reference:
// - net.minecraft.src.WorldGenCactus.generate(...)
func generateCactusPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	placed := false
	for i := 0; i < 10; i++ {
		x := worldX + int(rng.NextInt(8)) - int(rng.NextInt(8))
		yy := y + int(rng.NextInt(4)) - int(rng.NextInt(4))
		z := worldZ + int(rng.NextInt(8)) - int(rng.NextInt(8))
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
		if !ok || id != blockIDAir {
			continue
		}

		height := 1 + int(rng.NextInt(int(rng.NextInt(3)+1)))
		for h := 0; h < height; h++ {
			if !canCactusStayAtWorld(blocks, targetChunkX, targetChunkZ, x, yy+h, z) {
				continue
			}
			if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy+h, z, blockIDCactus) {
				placed = true
			}
		}
	}
	return placed
}

func canCactusStayAtWorld(blocks []byte, targetChunkX, targetChunkZ int, x, y, z int) bool {
	if y <= 0 || y >= worldHeightLegacyY {
		return false
	}
	neighbors := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, n := range neighbors {
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x+n[0], y, z+n[1])
		if !ok {
			return false
		}
		if blocksMovementForTopSolidOrLiquid(id) {
			return false
		}
	}
	below, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y-1, z)
	if !ok {
		return false
	}
	return below == blockIDCactus || below == blockIDSand
}

// Translation reference:
// - net.minecraft.src.WorldGenPumpkin.generate(...)
func generatePumpkinPatchAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	placed := false
	for i := 0; i < 64; i++ {
		x := worldX + int(rng.NextInt(8)) - int(rng.NextInt(8))
		yy := y + int(rng.NextInt(4)) - int(rng.NextInt(4))
		z := worldZ + int(rng.NextInt(8)) - int(rng.NextInt(8))
		if yy <= 0 || yy >= worldHeightLegacyY {
			continue
		}

		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z)
		if !ok || id != blockIDAir {
			continue
		}
		below, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy-1, z)
		if !ok || below != blockIDGrass {
			continue
		}

		if setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDPumpkin) {
			placed = true
		}
	}
	return placed
}

// Translation reference:
// - net.minecraft.src.World#canBlockSeeTheSky(...)
func canBlockSeeSkyAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int) bool {
	for yy := y + 1; yy < worldHeightLegacyY; yy++ {
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ)
		if !ok {
			return false
		}
		if lightOpacityForSkyCheck(id) > 0 {
			return false
		}
	}
	return true
}

func lightOpacityForSkyCheck(id byte) int {
	switch id {
	case blockIDAir, blockIDTallGrass, blockIDFlowerYellow, blockIDFlowerRed:
		return 0
	case blockIDWaterFlow, blockIDWater, blockIDIce:
		return 3
	case blockIDLeaves:
		return 1
	default:
		return 255
	}
}

func worldCoordInTargetChunk(targetChunkX, targetChunkZ int, worldX, worldZ int) bool {
	minX := targetChunkX * 16
	minZ := targetChunkZ * 16
	return worldX >= minX && worldX < minX+16 && worldZ >= minZ && worldZ < minZ+16
}
