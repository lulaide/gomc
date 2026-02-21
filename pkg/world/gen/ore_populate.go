package gen

import (
	"math"

	"github.com/lulaide/gomc/pkg/util"
)

const (
	blockIDCoalOre     byte = 16
	blockIDIronOre     byte = 15
	blockIDGoldOre     byte = 14
	blockIDLapisOre    byte = 21
	blockIDDiamondOre  byte = 56
	blockIDRedstoneOre byte = 73
	worldHeightLegacyY      = 128
)

type oreFeature struct {
	oreID      byte
	veinSize   int
	attempts   int
	minY       int
	maxY       int
	lapisStyle bool
}

var overworldOreFeatures = []oreFeature{
	{oreID: blockIDDirt, veinSize: 32, attempts: 20, minY: 0, maxY: 128},
	{oreID: blockIDGravel, veinSize: 32, attempts: 10, minY: 0, maxY: 128},
	{oreID: blockIDCoalOre, veinSize: 16, attempts: 20, minY: 0, maxY: 128},
	{oreID: blockIDIronOre, veinSize: 8, attempts: 20, minY: 0, maxY: 64},
	{oreID: blockIDGoldOre, veinSize: 8, attempts: 2, minY: 0, maxY: 32},
	{oreID: blockIDRedstoneOre, veinSize: 7, attempts: 8, minY: 0, maxY: 16},
	{oreID: blockIDDiamondOre, veinSize: 7, attempts: 1, minY: 0, maxY: 16},
	{oreID: blockIDLapisOre, veinSize: 6, attempts: 1, minY: 16, maxY: 16, lapisStyle: true},
}

// Translation reference:
// - net.minecraft.src.ChunkProviderGenerate.populate(...) seed derivation
// - net.minecraft.src.BiomeDecorator.generateOres()
func (p *ChunkProviderGenerate) populateOres(targetChunkX, targetChunkZ int, blocks []byte) {
	if p == nil || len(blocks) == 0 {
		return
	}

	rng := util.NewJavaRandom(p.worldSeed)
	xMul := (rng.NextLong()/2)*2 + 1
	zMul := (rng.NextLong()/2)*2 + 1

	// Ore veins generated in adjacent chunks can cross into the current chunk.
	for popChunkX := targetChunkX - 1; popChunkX <= targetChunkX+1; popChunkX++ {
		for popChunkZ := targetChunkZ - 1; popChunkZ <= targetChunkZ+1; popChunkZ++ {
			seed := (int64(popChunkX)*xMul + int64(popChunkZ)*zMul) ^ p.worldSeed
			rng.SetSeed(seed)
			p.populateOresForPopulateChunk(rng, popChunkX, popChunkZ, targetChunkX, targetChunkZ, blocks)
		}
	}
}

func (p *ChunkProviderGenerate) populateOresForPopulateChunk(
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

	for _, f := range overworldOreFeatures {
		for i := 0; i < f.attempts; i++ {
			originX := baseX + int(rng.NextInt(16))
			originZ := baseZ + int(rng.NextInt(16))
			originY := 0
			if f.lapisStyle {
				originY = int(rng.NextInt(f.maxY)) + int(rng.NextInt(f.maxY)) + (f.minY - f.maxY)
			} else {
				span := f.maxY - f.minY
				if span <= 0 {
					continue
				}
				originY = int(rng.NextInt(span)) + f.minY
			}
			generateMinableIntoChunk(
				blocks,
				rng,
				targetChunkX, targetChunkZ,
				originX, originY, originZ,
				f.oreID,
				f.veinSize,
				blockIDStone,
			)
		}
	}
}

// Translation reference:
// - net.minecraft.src.WorldGenMinable.generate(...)
func generateMinableIntoChunk(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	originX, originY, originZ int,
	oreID byte,
	veinSize int,
	replaceID byte,
) bool {
	if rng == nil || veinSize <= 0 {
		return false
	}

	angle := float64(rng.NextFloat()) * math.Pi
	startX := float64(float32(originX+8) + float32(math.Sin(angle))*float32(veinSize)/8.0)
	endX := float64(float32(originX+8) - float32(math.Sin(angle))*float32(veinSize)/8.0)
	startZ := float64(float32(originZ+8) + float32(math.Cos(angle))*float32(veinSize)/8.0)
	endZ := float64(float32(originZ+8) - float32(math.Cos(angle))*float32(veinSize)/8.0)
	startY := float64(originY + int(rng.NextInt(3)) - 2)
	endY := float64(originY + int(rng.NextInt(3)) - 2)

	targetMinX := targetChunkX * 16
	targetMinZ := targetChunkZ * 16
	targetMaxX := targetMinX + 15
	targetMaxZ := targetMinZ + 15
	placed := false

	for i := 0; i <= veinSize; i++ {
		t := float64(i) / float64(veinSize)
		centerX := startX + (endX-startX)*t
		centerY := startY + (endY-startY)*t
		centerZ := startZ + (endZ-startZ)*t
		var26 := rng.NextDouble() * float64(veinSize) / 16.0
		radiusXZ := float64(float32(math.Sin(float64(float32(i)*float32(math.Pi)/float32(veinSize)))+1.0))*var26 + 1.0
		radiusY := float64(float32(math.Sin(float64(float32(i)*float32(math.Pi)/float32(veinSize)))+1.0))*var26 + 1.0

		minX := floorDouble(centerX - radiusXZ/2.0)
		maxX := floorDouble(centerX + radiusXZ/2.0)
		minY := floorDouble(centerY - radiusY/2.0)
		maxY := floorDouble(centerY + radiusY/2.0)
		minZ := floorDouble(centerZ - radiusXZ/2.0)
		maxZ := floorDouble(centerZ + radiusXZ/2.0)

		for x := minX; x <= maxX; x++ {
			normX := (float64(x) + 0.5 - centerX) / (radiusXZ / 2.0)
			if normX*normX >= 1.0 {
				continue
			}
			for y := minY; y <= maxY; y++ {
				if y < 0 || y >= worldHeightLegacyY {
					continue
				}
				normY := (float64(y) + 0.5 - centerY) / (radiusY / 2.0)
				if normX*normX+normY*normY >= 1.0 {
					continue
				}
				for z := minZ; z <= maxZ; z++ {
					normZ := (float64(z) + 0.5 - centerZ) / (radiusXZ / 2.0)
					if normX*normX+normY*normY+normZ*normZ >= 1.0 {
						continue
					}
					if x < targetMinX || x > targetMaxX || z < targetMinZ || z > targetMaxZ {
						continue
					}
					localX := x - targetMinX
					localZ := z - targetMinZ
					idx := chunkBlockIndex(localX, localZ, y)
					if idx < 0 || idx >= len(blocks) {
						continue
					}
					if blocks[idx] != replaceID {
						continue
					}
					blocks[idx] = oreID
					placed = true
				}
			}
		}
	}

	return placed
}
