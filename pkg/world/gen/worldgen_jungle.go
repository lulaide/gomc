package gen

import (
	"math"

	"github.com/lulaide/gomc/pkg/util"
)

// Translation reference:
// - net.minecraft.src.WorldGenShrub.generate(...)
func generateJungleShrubAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
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

	ground := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDAir)
	if ground == blockIDDirt || ground == blockIDGrass {
		y++
		_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDLog, 3)

		for yy := y; yy <= y+2; yy++ {
			dy := yy - y
			radius := 2 - dy

			for xx := worldX - radius; xx <= worldX+radius; xx++ {
				dx := xx - worldX
				for zz := worldZ - radius; zz <= worldZ+radius; zz++ {
					dz := zz - worldZ
					if (absInt(dx) != radius || absInt(dz) != radius || rng.NextInt(2) != 0) &&
						!isOpaqueCubeForGen(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)) {
						_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDLeaves, 0)
					}
				}
			}
		}
	}

	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenHugeTrees.generate(...)
func generateHugeJungleTreeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	baseHeight int,
) bool {
	if rng == nil {
		return false
	}
	if baseHeight <= 0 {
		baseHeight = 10
	}

	height := int(rng.NextInt(3)) + baseHeight
	canGrow := true

	if y < 1 || y+height+1 > worldHeightLegacyY {
		return false
	}

	for yy := y; yy <= y+1+height; yy++ {
		radius := 2
		if yy == y {
			radius = 1
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
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+1, y-1, worldZ, blockIDDirt)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ+1, blockIDDirt)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+1, y-1, worldZ+1, blockIDDirt)

	growHugeJungleLeavesAtWorld(blocks, rng, targetChunkX, targetChunkZ, worldX, worldZ, y+height, 2)

	for branchY := y + height - 2 - int(rng.NextInt(4)); branchY > y+height/2; branchY -= 2 + int(rng.NextInt(4)) {
		angle := float64(float32(rng.NextFloat()) * float32(math.Pi) * 2.0)
		branchX := worldX + int(0.5+math.Cos(angle)*4.0)
		branchZ := worldZ + int(0.5+math.Sin(angle)*4.0)
		growHugeJungleLeavesAtWorld(blocks, rng, targetChunkX, targetChunkZ, branchX, branchZ, branchY, 0)

		for i := 0; i < 5; i++ {
			logX := worldX + int(1.5+math.Cos(angle)*float64(i))
			logZ := worldZ + int(1.5+math.Sin(angle)*float64(i))
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, logX, branchY-3+i/2, logZ, blockIDLog, 3)
		}
	}

	for dy := 0; dy < height; dy++ {
		yy := y + dy
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDLog, 3)

			if dy > 0 {
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX-1, yy, worldZ, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX-1, yy, worldZ, blockIDVine, 8)
				}
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ-1, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ-1, blockIDVine, 1)
				}
			}
		}

		if dy >= height-1 {
			continue
		}

		id = blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ, blockIDLog, 3)

			if dy > 0 {
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX+2, yy, worldZ, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+2, yy, worldZ, blockIDVine, 2)
				}
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ-1, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ-1, blockIDVine, 1)
				}
			}
		}

		id = blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ+1, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ+1, blockIDLog, 3)

			if dy > 0 {
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX+2, yy, worldZ+1, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+2, yy, worldZ+1, blockIDVine, 2)
				}
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ+2, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ+2, blockIDVine, 4)
				}
			}
		}

		id = blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ+1, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ+1, blockIDLog, 3)

			if dy > 0 {
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX-1, yy, worldZ+1, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX-1, yy, worldZ+1, blockIDVine, 8)
				}
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ+2, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ+2, blockIDVine, 4)
				}
			}
		}
	}

	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenHugeTrees.growLeaves(...)
func growHugeJungleLeavesAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, worldZ, y, radiusBase int,
) {
	const branchRadius = 2

	for yy := y - branchRadius; yy <= y; yy++ {
		relY := yy - y
		radius := radiusBase + 1 - relY

		for xx := worldX - radius; xx <= worldX+radius+1; xx++ {
			dx := xx - worldX
			for zz := worldZ - radius; zz <= worldZ+radius+1; zz++ {
				dz := zz - worldZ
				if (dx >= 0 || dz >= 0 || dx*dx+dz*dz <= radius*radius) &&
					(dx <= 0 && dz <= 0 || dx*dx+dz*dz <= (radius+1)*(radius+1)) &&
					(rng.NextInt(4) != 0 || dx*dx+dz*dz <= (radius-1)*(radius-1)) {
					id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
					if id == blockIDAir || id == blockIDLeaves {
						_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDLeaves, 3)
					}
				}
			}
		}
	}
}

// Translation reference:
// - net.minecraft.src.WorldGenTrees.generate(...)
func generateJungleTreeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	minTreeHeight int,
	vinesGrow bool,
) bool {
	if rng == nil {
		return false
	}

	height := int(rng.NextInt(3)) + minTreeHeight
	canGrow := true

	if y < 1 || y+height+1 > worldHeightLegacyY {
		return false
	}

	for yy := y; yy <= y+1+height; yy++ {
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
			for zz := worldZ - radius; zz <= worldZ+radius; zz++ {
				dz := zz - worldZ
				if absInt(dx) == radius && absInt(dz) == radius && (rng.NextInt(2) != 0 && dy != 0) {
					continue
				}

				id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDAir)
				if id == blockIDAir || id == blockIDLeaves {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, xx, yy, zz, blockIDLeaves, 3)
				}
			}
		}
	}

	for dy := 0; dy < height; dy++ {
		yy := y + dy
		id := blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDAir)
		if id == blockIDAir || id == blockIDLeaves {
			_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ, blockIDLog, 3)

			if vinesGrow && dy > 0 {
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX-1, yy, worldZ, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX-1, yy, worldZ, blockIDVine, 8)
				}
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+1, yy, worldZ, blockIDVine, 2)
				}
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ-1, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ-1, blockIDVine, 1)
				}
				if rng.NextInt(3) > 0 && blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ+1, blockIDAir) == blockIDAir {
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, yy, worldZ+1, blockIDVine, 4)
				}
			}
		}
	}

	if vinesGrow {
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

		if rng.NextInt(5) == 0 && height > 5 {
			offsetX := [...]int{0, -1, 0, 1}
			offsetZ := [...]int{1, 0, -1, 0}
			rotateOpposite := [...]int{2, 3, 0, 1}

			for level := 0; level < 2; level++ {
				for dir := 0; dir < 4; dir++ {
					if rng.NextInt(4-level) != 0 {
						continue
					}
					age := int(rng.NextInt(3))
					meta := byte((age << 2) | dir)
					cocoaX := worldX + offsetX[rotateOpposite[dir]]
					cocoaY := y + height - 5 + level
					cocoaZ := worldZ + offsetZ[rotateOpposite[dir]]
					_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, cocoaX, cocoaY, cocoaZ, blockIDCocoaPlant, meta)
				}
			}
		}
	}

	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenVines.generate(...)
func generateWorldVinesAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}

	startX := worldX
	startZ := worldZ

	for y < worldHeightLegacyY {
		if blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, blockIDAir) == blockIDAir {
			for side := 2; side <= 5; side++ {
				if canPlaceVineOnSideAtWorld(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, side) {
					_ = setBlockAtWorldInTargetChunk(
						blocks,
						targetChunkX,
						targetChunkZ,
						worldX,
						y,
						worldZ,
						blockIDVine,
						worldVineMetadataForAttachSide(side),
					)
					break
				}
			}
		} else {
			worldX = startX + int(rng.NextInt(4)) - int(rng.NextInt(4))
			worldZ = startZ + int(rng.NextInt(4)) - int(rng.NextInt(4))
		}
		y++
	}

	return true
}

func canPlaceVineOnSideAtWorld(
	blocks []byte,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
	side int,
) bool {
	switch side {
	case 1:
		return canVineAttachToID(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y+1, worldZ, blockIDAir))
	case 2:
		return canVineAttachToID(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ+1, blockIDAir))
	case 3:
		return canVineAttachToID(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ-1, blockIDAir))
	case 4:
		return canVineAttachToID(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX+1, y, worldZ, blockIDAir))
	case 5:
		return canVineAttachToID(blockAtWorldForGen(blocks, targetChunkX, targetChunkZ, worldX-1, y, worldZ, blockIDAir))
	default:
		return false
	}
}

func canVineAttachToID(id byte) bool {
	// Translation note:
	// BlockVine.canBePlacedOn(...) requires renderAsNormalBlock && blocksMovement.
	// In this ID-only generation path we approximate that with opaque-cube lookup.
	return isOpaqueCubeForGen(id)
}

func worldVineMetadataForAttachSide(side int) byte {
	// Translation reference:
	// - net.minecraft.src.WorldGenVines.generate(...)
	//   1 << Direction.facingToDirection[Facing.oppositeSide[side]]
	switch side {
	case 2:
		return 1
	case 3:
		return 4
	case 4:
		return 8
	case 5:
		return 2
	default:
		return 0
	}
}
