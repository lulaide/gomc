package gen

import "github.com/lulaide/gomc/pkg/util"

const (
	biomeIDDesert      byte = 2
	biomeIDDesertHills byte = 17
)

// Translation reference:
// - net.minecraft.src.ChunkProviderGenerate.populate(...)
func (p *ChunkProviderGenerate) populateChunkDecorations(targetChunkX, targetChunkZ int, blocks []byte) {
	if p == nil || len(blocks) == 0 {
		return
	}

	rng := util.NewJavaRandom(p.worldSeed)
	xMul := (rng.NextLong()/2)*2 + 1
	zMul := (rng.NextLong()/2)*2 + 1
	// Translation reference:
	// - ChunkProviderGenerate.populate() acts on a world and can place blocks across chunk borders.
	// Apply nearby populate chunks so cross-border features can affect this target chunk buffer.
	for popChunkX := targetChunkX - 1; popChunkX <= targetChunkX+1; popChunkX++ {
		for popChunkZ := targetChunkZ - 1; popChunkZ <= targetChunkZ+1; popChunkZ++ {
			seed := (int64(popChunkX)*xMul + int64(popChunkZ)*zMul) ^ p.worldSeed
			rng.SetSeed(seed)
			p.populateChunkDecorationsForPopulateChunk(
				rng,
				popChunkX, popChunkZ,
				targetChunkX, targetChunkZ,
				blocks,
			)
		}
	}
}

func (p *ChunkProviderGenerate) populateChunkDecorationsForPopulateChunk(
	rng *util.JavaRandom,
	popChunkX, popChunkZ int,
	targetChunkX, targetChunkZ int,
	blocks []byte,
) {
	if rng == nil {
		return
	}

	popBaseX := popChunkX * 16
	popBaseZ := popChunkZ * 16
	biome := p.biomeAt(popBaseX+16, popBaseZ+16)
	villageGenerated := false

	if biome.ID != biomeIDDesert && biome.ID != biomeIDDesertHills && !villageGenerated && rng.NextInt(4) == 0 {
		x := popBaseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		z := popBaseZ + int(rng.NextInt(16)) + 8
		p.generateLakeAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDWater)
	}

	if !villageGenerated && rng.NextInt(8) == 0 {
		x := popBaseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(int(rng.NextInt(120) + 8)))
		z := popBaseZ + int(rng.NextInt(16)) + 8
		if y < 63 || rng.NextInt(10) == 0 {
			p.generateLakeAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z, blockIDLava)
		}
	}

	for i := 0; i < 8; i++ {
		x := popBaseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(worldHeightLegacyY))
		z := popBaseZ + int(rng.NextInt(16)) + 8
		generateDungeonAtWorld(blocks, rng, targetChunkX, targetChunkZ, x, y, z)
	}

	// BiomeDecorator subset currently translated in Go.
	p.populateOresForPopulateChunk(rng, popChunkX, popChunkZ, targetChunkX, targetChunkZ, blocks)
	p.populateSurfaceForPopulateChunk(rng, popChunkX, popChunkZ, targetChunkX, targetChunkZ, blocks)
	p.populateLiquidSpringsForPopulateChunk(rng, popChunkX, popChunkZ, targetChunkX, targetChunkZ, blocks)
}

// Translation reference:
// - net.minecraft.src.WorldGenLakes.generate(...)
func (p *ChunkProviderGenerate) generateLakeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	liquidID byte,
) bool {
	if rng == nil {
		return false
	}

	startX := worldX - 8
	startZ := worldZ - 8
	for y > 5 {
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, startX, y, startZ, blockIDStone)
		if id != blockIDAir {
			break
		}
		y--
	}
	if y <= 4 {
		return false
	}
	y -= 4

	lake := make([]bool, 2048)
	biomeCache := make(map[int]Biome, 64)
	blobCount := int(rng.NextInt(4)) + 4
	for i := 0; i < blobCount; i++ {
		sizeX := rng.NextDouble()*6.0 + 3.0
		sizeY := rng.NextDouble()*4.0 + 2.0
		sizeZ := rng.NextDouble()*6.0 + 3.0
		centerX := rng.NextDouble()*(16.0-sizeX-2.0) + 1.0 + sizeX/2.0
		centerY := rng.NextDouble()*(8.0-sizeY-4.0) + 2.0 + sizeY/2.0
		centerZ := rng.NextDouble()*(16.0-sizeZ-2.0) + 1.0 + sizeZ/2.0

		for lx := 1; lx < 15; lx++ {
			for lz := 1; lz < 15; lz++ {
				for ly := 1; ly < 7; ly++ {
					dx := (float64(lx) - centerX) / (sizeX / 2.0)
					dy := (float64(ly) - centerY) / (sizeY / 2.0)
					dz := (float64(lz) - centerZ) / (sizeZ / 2.0)
					if dx*dx+dy*dy+dz*dz < 1.0 {
						lake[lakeIndex(lx, lz, ly)] = true
					}
				}
			}
		}
	}

	for lx := 0; lx < 16; lx++ {
		for lz := 0; lz < 16; lz++ {
			for ly := 0; ly < 8; ly++ {
				edge := isLakeEdgeCell(lake, lx, lz, ly)
				if !edge {
					continue
				}

				bx := startX + lx
				by := y + ly
				bz := startZ + lz
				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, bx, by, bz, blockIDStone)

				if ly >= 4 && isLiquidBlockForGen(id) {
					return false
				}
				if ly < 4 && !isSolidBlockForGen(id) && id != liquidID {
					return false
				}
			}
		}
	}

	for lx := 0; lx < 16; lx++ {
		for lz := 0; lz < 16; lz++ {
			for ly := 0; ly < 8; ly++ {
				if !lake[lakeIndex(lx, lz, ly)] {
					continue
				}
				id := blockIDAir
				if ly < 4 {
					id = liquidID
				}
				_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, startX+lx, y+ly, startZ+lz, id)
			}
		}
	}

	for lx := 0; lx < 16; lx++ {
		for lz := 0; lz < 16; lz++ {
			for ly := 4; ly < 8; ly++ {
				if !lake[lakeIndex(lx, lz, ly)] {
					continue
				}

				bx := startX + lx
				by := y + ly
				bz := startZ + lz
				belowID := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, bx, by-1, bz, blockIDStone)
				if belowID != blockIDDirt {
					continue
				}
				if !canBlockSeeSkyAtWorld(blocks, targetChunkX, targetChunkZ, bx, by, bz) {
					continue
				}

				cacheKey := (bx << 16) ^ (bz & 0xFFFF)
				biome, ok := biomeCache[cacheKey]
				if !ok {
					biome = p.biomeAt(bx, bz)
					biomeCache[cacheKey] = biome
				}
				top := blockIDGrass
				if biome.TopBlock == blockIDMycelium {
					top = blockIDMycelium
				}
				_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, bx, by-1, bz, top)
			}
		}
	}

	if liquidID == blockIDLava || liquidID == blockIDLavaFlow {
		for lx := 0; lx < 16; lx++ {
			for lz := 0; lz < 16; lz++ {
				for ly := 0; ly < 8; ly++ {
					edge := isLakeEdgeCell(lake, lx, lz, ly)
					if !edge {
						continue
					}
					if ly >= 4 && rng.NextInt(2) == 0 {
						continue
					}
					bx := startX + lx
					by := y + ly
					bz := startZ + lz
					id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, bx, by, bz, blockIDStone)
					if !isSolidBlockForGen(id) {
						continue
					}
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, bx, by, bz, blockIDStone)
				}
			}
		}
	}

	if liquidID == blockIDWater || liquidID == blockIDWaterFlow {
		for lx := 0; lx < 16; lx++ {
			for lz := 0; lz < 16; lz++ {
				bx := startX + lx
				by := y + 4
				bz := startZ + lz
				if p.isBlockFreezableAtWorld(blocks, targetChunkX, targetChunkZ, bx, by, bz) {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, bx, by, bz, blockIDIce)
				}
			}
		}
	}

	return true
}

func lakeIndex(localX, localZ, localY int) int {
	return (localX*16+localZ)*8 + localY
}

func isLakeEdgeCell(lake []bool, localX, localZ, localY int) bool {
	idx := lakeIndex(localX, localZ, localY)
	if idx < 0 || idx >= len(lake) || lake[idx] {
		return false
	}

	return (localX < 15 && lake[lakeIndex(localX+1, localZ, localY)]) ||
		(localX > 0 && lake[lakeIndex(localX-1, localZ, localY)]) ||
		(localZ < 15 && lake[lakeIndex(localX, localZ+1, localY)]) ||
		(localZ > 0 && lake[lakeIndex(localX, localZ-1, localY)]) ||
		(localY < 7 && lake[lakeIndex(localX, localZ, localY+1)]) ||
		(localY > 0 && lake[lakeIndex(localX, localZ, localY-1)])
}

func isLiquidBlockForGen(id byte) bool {
	return id == blockIDWaterFlow || id == blockIDWater || id == blockIDLavaFlow || id == blockIDLava
}

func isSolidBlockForGen(id byte) bool {
	switch id {
	case blockIDAir, blockIDTallGrass, blockIDFlowerYellow, blockIDFlowerRed, blockIDReed:
		return false
	default:
		return !isLiquidBlockForGen(id)
	}
}

func (p *ChunkProviderGenerate) isBlockFreezableAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int) bool {
	if y < 0 || y >= worldHeightLegacyY {
		return false
	}

	biome := p.biomeAt(worldX, worldZ)
	if biome.Temperature > 0.15 {
		return false
	}

	id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
	if !ok {
		id = blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDStone)
	}
	return id == blockIDWater || id == blockIDWaterFlow
}

// Translation reference:
// - net.minecraft.src.WorldGenDungeons.generate(...)
func generateDungeonAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil || y < 1 || y >= worldHeightLegacyY-4 {
		return false
	}

	const roomHeight = 3
	radiusX := int(rng.NextInt(2)) + 2
	radiusZ := int(rng.NextInt(2)) + 2
	openings := 0

	minX := worldX - radiusX - 1
	maxX := worldX + radiusX + 1
	minY := y - 1
	maxY := y + roomHeight + 1
	minZ := worldZ - radiusZ - 1
	maxZ := worldZ + radiusZ + 1

	for x := minX; x <= maxX; x++ {
		for yy := minY; yy <= maxY; yy++ {
			for z := minZ; z <= maxZ; z++ {
				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDStone)
				solid := isSolidBlockForGen(id)
				if yy == minY && !solid {
					return false
				}
				if yy == maxY && !solid {
					return false
				}
				if (x == minX || x == maxX || z == minZ || z == maxZ) &&
					yy == y &&
					blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDStone) == blockIDAir &&
					blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, x, yy+1, z, blockIDStone) == blockIDAir {
					openings++
				}
			}
		}
	}

	if openings < 1 || openings > 5 {
		return false
	}

	for x := minX; x <= maxX; x++ {
		for yy := y + roomHeight; yy >= minY; yy-- {
			for z := minZ; z <= maxZ; z++ {
				if x != minX && yy != minY && z != minZ && x != maxX && yy != maxY && z != maxZ {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDAir)
					continue
				}

				if yy >= 0 {
					below := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, x, yy-1, z, blockIDStone)
					if !isSolidBlockForGen(below) {
						_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDAir)
						continue
					}
				}

				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, x, yy, z, blockIDStone)
				if isSolidBlockForGen(id) {
					blockID := blockIDCobblestone
					if yy == minY && rng.NextInt(4) != 0 {
						blockID = blockIDMossyCobble
					}
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, yy, z, blockID)
				}
			}
		}
	}

	for chest := 0; chest < 2; chest++ {
		placed := false
		for tries := 0; tries < 3; tries++ {
			cx := worldX + int(rng.NextInt(radiusX*2+1)) - radiusX
			cz := worldZ + int(rng.NextInt(radiusZ*2+1)) - radiusZ
			if blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, cx, y, cz, blockIDStone) != blockIDAir {
				continue
			}

			solidSides := 0
			if isSolidBlockForGen(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, cx-1, y, cz, blockIDStone)) {
				solidSides++
			}
			if isSolidBlockForGen(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, cx+1, y, cz, blockIDStone)) {
				solidSides++
			}
			if isSolidBlockForGen(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, cx, y, cz-1, blockIDStone)) {
				solidSides++
			}
			if isSolidBlockForGen(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, cx, y, cz+1, blockIDStone)) {
				solidSides++
			}
			if solidSides != 1 {
				continue
			}

			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, cx, y, cz, blockIDChest)
			placed = true
			break
		}
		if !placed {
			continue
		}
	}

	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDMobSpawner)
	return true
}

func blockAtWorldForGen(
	blocks []byte,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	outOfChunkDefault byte,
) byte {
	if y < 0 || y >= worldHeightLegacyY {
		return blockIDAir
	}
	id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
	if !ok {
		return outOfChunkDefault
	}
	return id
}
