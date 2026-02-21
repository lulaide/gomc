package gen

import "github.com/lulaide/gomc/pkg/util"

// Translation reference:
// - net.minecraft.src.WorldGenTaiga1.generate(...)
func generateTaiga1TreeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	height := int(rng.NextInt(5)) + 7
	trunkToLeafStart := height - int(rng.NextInt(2)) - 3
	leafHeight := height - trunkToLeafStart
	leafRadiusMax := 1 + int(rng.NextInt(leafHeight+1))
	canGrow := true

	if y < 1 || y+height+1 > worldHeightLegacyY {
		return false
	}

	for yy := y; yy <= y+1+height && canGrow; yy++ {
		radius := 0
		if yy-y >= trunkToLeafStart {
			radius = leafRadiusMax
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

	leafRadius := 0
	for yy := y + height; yy >= y+trunkToLeafStart; yy-- {
		for xx := worldX - leafRadius; xx <= worldX+leafRadius; xx++ {
			dx := xx - worldX
			for zz := worldZ - leafRadius; zz <= worldZ+leafRadius; zz++ {
				dz := zz - worldZ
				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if (absInt(dx) != leafRadius || absInt(dz) != leafRadius || leafRadius <= 0) && !isOpaqueCubeForGen(id) {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDLeaves, 1)
				}
			}
		}

		if leafRadius >= 1 && yy == y+trunkToLeafStart+1 {
			leafRadius--
		} else if leafRadius < leafRadiusMax {
			leafRadius++
		}
	}

	for i := 0; i < height-1; i++ {
		yy := y + i
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDLog, 1)
		}
	}

	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenTaiga2.generate(...)
func generateTaiga2TreeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	height := int(rng.NextInt(4)) + 6
	trunkTopClear := 1 + int(rng.NextInt(2))
	crownDepth := height - trunkTopClear
	crownRadiusMax := 2 + int(rng.NextInt(2))
	canGrow := true

	if y < 1 || y+height+1 > worldHeightLegacyY {
		return false
	}

	for yy := y; yy <= y+1+height && canGrow; yy++ {
		radius := 0
		if yy-y >= trunkTopClear {
			radius = crownRadiusMax
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

	leafRadius := int(rng.NextInt(2))
	leafRadiusStepThreshold := 1
	leafRadiusReset := 0

	for level := 0; level <= crownDepth; level++ {
		yy := y + height - level
		for xx := worldX - leafRadius; xx <= worldX+leafRadius; xx++ {
			dx := xx - worldX
			for zz := worldZ - leafRadius; zz <= worldZ+leafRadius; zz++ {
				dz := zz - worldZ
				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if (absInt(dx) != leafRadius || absInt(dz) != leafRadius || leafRadius <= 0) && !isOpaqueCubeForGen(id) {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDLeaves, 1)
				}
			}
		}

		if leafRadius >= leafRadiusStepThreshold {
			leafRadius = leafRadiusReset
			leafRadiusReset = 1
			leafRadiusStepThreshold++
			if leafRadiusStepThreshold > crownRadiusMax {
				leafRadiusStepThreshold = crownRadiusMax
			}
		} else {
			leafRadius++
		}
	}

	trunkTrim := int(rng.NextInt(3))
	for i := 0; i < height-trunkTrim; i++ {
		yy := y + i
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDLog, 1)
		}
	}

	return true
}
