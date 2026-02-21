package gen

import (
	"math"

	"github.com/lulaide/gomc/pkg/util"
)

const (
	mapGenRangeChunks = 8
	chunkHeight128    = 128
)

type mapGenCaves struct {
	rand      *util.JavaRandom
	worldSeed int64
	rngRange  int
}

func newMapGenCaves(worldSeed int64) *mapGenCaves {
	return &mapGenCaves{
		rand:      util.NewJavaRandom(0),
		worldSeed: worldSeed,
		rngRange:  mapGenRangeChunks,
	}
}

// Translation reference:
// - net.minecraft.src.MapGenBase.generate(...)
// - net.minecraft.src.MapGenCaves.recursiveGenerate(...)
func (g *mapGenCaves) generate(targetChunkX, targetChunkZ int, blocks []byte, biomes []Biome) {
	rng := g.rand
	rng.SetSeed(g.worldSeed)
	seedX := rng.NextLong()
	seedZ := rng.NextLong()

	for chunkX := targetChunkX - g.rngRange; chunkX <= targetChunkX+g.rngRange; chunkX++ {
		for chunkZ := targetChunkZ - g.rngRange; chunkZ <= targetChunkZ+g.rngRange; chunkZ++ {
			seed := int64(chunkX)*seedX ^ int64(chunkZ)*seedZ ^ g.worldSeed
			rng.SetSeed(seed)
			g.recursiveGenerate(chunkX, chunkZ, targetChunkX, targetChunkZ, blocks, biomes)
		}
	}
}

func (g *mapGenCaves) recursiveGenerate(chunkX, chunkZ, targetChunkX, targetChunkZ int, blocks []byte, biomes []Biome) {
	rng := g.rand
	count := int(rng.NextInt(int(rng.NextInt(int(rng.NextInt(40)+1)) + 1)))
	if rng.NextInt(15) != 0 {
		count = 0
	}

	for i := 0; i < count; i++ {
		startX := float64(chunkX*16 + int(rng.NextInt(16)))
		startY := float64(int(rng.NextInt(int(rng.NextInt(120) + 8))))
		startZ := float64(chunkZ*16 + int(rng.NextInt(16)))
		branches := 1

		if rng.NextInt(4) == 0 {
			g.generateLargeCaveNode(rng.NextLong(), targetChunkX, targetChunkZ, blocks, biomes, startX, startY, startZ)
			branches += int(rng.NextInt(4))
		}

		for j := 0; j < branches; j++ {
			yaw := rng.NextFloat() * float32(math.Pi) * 2.0
			pitch := (rng.NextFloat() - 0.5) * 2.0 / 8.0
			size := rng.NextFloat()*2.0 + rng.NextFloat()
			if rng.NextInt(10) == 0 {
				size *= rng.NextFloat()*rng.NextFloat()*3.0 + 1.0
			}
			g.generateCaveNode(
				rng.NextLong(),
				targetChunkX,
				targetChunkZ,
				blocks,
				biomes,
				startX,
				startY,
				startZ,
				size,
				yaw,
				pitch,
				0,
				0,
				1.0,
			)
		}
	}
}

func (g *mapGenCaves) generateLargeCaveNode(seed int64, targetChunkX, targetChunkZ int, blocks []byte, biomes []Biome, x, y, z float64) {
	g.generateCaveNode(
		seed,
		targetChunkX,
		targetChunkZ,
		blocks,
		biomes,
		x,
		y,
		z,
		1.0+g.rand.NextFloat()*6.0,
		0.0,
		0.0,
		-1,
		-1,
		0.5,
	)
}

// Translation reference:
// - net.minecraft.src.MapGenCaves.generateCaveNode(...)
func (g *mapGenCaves) generateCaveNode(
	seed int64,
	targetChunkX, targetChunkZ int,
	blocks []byte,
	biomes []Biome,
	x, y, z float64,
	size float32,
	yaw, pitch float32,
	start, end int,
	verticalScale float64,
) {
	chunkMidX := float64(targetChunkX*16 + 8)
	chunkMidZ := float64(targetChunkZ*16 + 8)
	yawChange := float32(0)
	pitchChange := float32(0)
	rng := util.NewJavaRandom(seed)

	if end <= 0 {
		maxLen := g.rngRange*16 - 16
		end = maxLen - int(rng.NextInt(maxLen/4))
	}

	isLarge := false
	if start == -1 {
		start = end / 2
		isLarge = true
	}

	branchPoint := int(rng.NextInt(end/2)) + end/4
	flatten := rng.NextInt(6) == 0

	for ; start < end; start++ {
		radiusXZ := 1.5 + math.Sin(float64(float32(start)*float32(math.Pi)/float32(end)))*float64(size)
		radiusY := radiusXZ * verticalScale
		cosPitch := float32(math.Cos(float64(pitch)))
		sinPitch := float32(math.Sin(float64(pitch)))
		x += float64(float32(math.Cos(float64(yaw))) * cosPitch)
		y += float64(sinPitch)
		z += float64(float32(math.Sin(float64(yaw))) * cosPitch)

		if flatten {
			pitch *= 0.92
		} else {
			pitch *= 0.7
		}

		pitch += pitchChange * 0.1
		yaw += yawChange * 0.1
		pitchChange *= 0.9
		yawChange *= 0.75
		pitchChange += (rng.NextFloat() - rng.NextFloat()) * rng.NextFloat() * 2.0
		yawChange += (rng.NextFloat() - rng.NextFloat()) * rng.NextFloat() * 4.0

		if !isLarge && start == branchPoint && size > 1.0 && end > 0 {
			g.generateCaveNode(
				rng.NextLong(),
				targetChunkX, targetChunkZ,
				blocks, biomes,
				x, y, z,
				rng.NextFloat()*0.5+0.5,
				yaw-float32(math.Pi)/2.0,
				pitch/3.0,
				start, end,
				1.0,
			)
			g.generateCaveNode(
				rng.NextLong(),
				targetChunkX, targetChunkZ,
				blocks, biomes,
				x, y, z,
				rng.NextFloat()*0.5+0.5,
				yaw+float32(math.Pi)/2.0,
				pitch/3.0,
				start, end,
				1.0,
			)
			return
		}

		if !isLarge && rng.NextInt(4) == 0 {
			continue
		}

		dxCenter := x - chunkMidX
		dzCenter := z - chunkMidZ
		remain := float64(end - start)
		maxDist := float64(size + 2.0 + 16.0)
		if dxCenter*dxCenter+dzCenter*dzCenter-remain*remain > maxDist*maxDist {
			return
		}

		if x < chunkMidX-16.0-radiusXZ*2.0 || z < chunkMidZ-16.0-radiusXZ*2.0 ||
			x > chunkMidX+16.0+radiusXZ*2.0 || z > chunkMidZ+16.0+radiusXZ*2.0 {
			continue
		}

		minX := floorDouble(x-radiusXZ) - targetChunkX*16 - 1
		maxX := floorDouble(x+radiusXZ) - targetChunkX*16 + 1
		minY := floorDouble(y-radiusY) - 1
		maxY := floorDouble(y+radiusY) + 1
		minZ := floorDouble(z-radiusXZ) - targetChunkZ*16 - 1
		maxZ := floorDouble(z+radiusXZ) - targetChunkZ*16 + 1

		if minX < 0 {
			minX = 0
		}
		if maxX > 16 {
			maxX = 16
		}
		if minY < 1 {
			minY = 1
		}
		if maxY > 120 {
			maxY = 120
		}
		if minZ < 0 {
			minZ = 0
		}
		if maxZ > 16 {
			maxZ = 16
		}

		containsWater := false
		for lx := minX; !containsWater && lx < maxX; lx++ {
			for lz := minZ; !containsWater && lz < maxZ; lz++ {
				for ly := maxY + 1; !containsWater && ly >= minY-1; ly-- {
					if ly < 0 || ly >= chunkHeight128 {
						continue
					}
					idx := chunkBlockIndex(lx, lz, ly)
					id := blocks[idx]
					if id == blockIDWaterFlow || id == blockIDWater {
						containsWater = true
					}
					if ly != minY-1 && lx != minX && lx != maxX-1 && lz != minZ && lz != maxZ-1 {
						ly = minY
					}
				}
			}
		}

		if containsWater {
			continue
		}

		for lx := minX; lx < maxX; lx++ {
			normX := (float64(lx+targetChunkX*16) + 0.5 - x) / radiusXZ
			for lz := minZ; lz < maxZ; lz++ {
				normZ := (float64(lz+targetChunkZ*16) + 0.5 - z) / radiusXZ
				index := chunkBlockIndex(lx, lz, maxY)
				hadGrass := false
				if normX*normX+normZ*normZ >= 1.0 {
					continue
				}
				for ly := maxY - 1; ly >= minY; ly-- {
					normY := (float64(ly) + 0.5 - y) / radiusY
					if normY <= -0.7 || normX*normX+normY*normY+normZ*normZ >= 1.0 {
						index--
						continue
					}
					id := blocks[index]
					if id == blockIDGrass {
						hadGrass = true
					}
					if id == blockIDStone || id == blockIDDirt || id == blockIDGrass {
						if ly < 10 {
							blocks[index] = blockIDLavaFlow
						} else {
							blocks[index] = blockIDAir
							if hadGrass && index > 0 && blocks[index-1] == blockIDDirt {
								blocks[index-1] = biomeTopBlockAtLocal(lx, lz, biomes)
							}
						}
					}
					index--
				}
			}
		}

		if isLarge {
			break
		}
	}
}

type mapGenRavine struct {
	rand      *util.JavaRandom
	worldSeed int64
	rngRange  int
	yShape    []float32
}

func newMapGenRavine(worldSeed int64) *mapGenRavine {
	return &mapGenRavine{
		rand:      util.NewJavaRandom(0),
		worldSeed: worldSeed,
		rngRange:  mapGenRangeChunks,
		yShape:    make([]float32, 1024),
	}
}

// Translation reference:
// - net.minecraft.src.MapGenBase.generate(...)
// - net.minecraft.src.MapGenRavine.recursiveGenerate(...)
func (g *mapGenRavine) generate(targetChunkX, targetChunkZ int, blocks []byte, biomes []Biome) {
	rng := g.rand
	rng.SetSeed(g.worldSeed)
	seedX := rng.NextLong()
	seedZ := rng.NextLong()

	for chunkX := targetChunkX - g.rngRange; chunkX <= targetChunkX+g.rngRange; chunkX++ {
		for chunkZ := targetChunkZ - g.rngRange; chunkZ <= targetChunkZ+g.rngRange; chunkZ++ {
			seed := int64(chunkX)*seedX ^ int64(chunkZ)*seedZ ^ g.worldSeed
			rng.SetSeed(seed)
			g.recursiveGenerate(chunkX, chunkZ, targetChunkX, targetChunkZ, blocks, biomes)
		}
	}
}

func (g *mapGenRavine) recursiveGenerate(chunkX, chunkZ, targetChunkX, targetChunkZ int, blocks []byte, biomes []Biome) {
	rng := g.rand
	if rng.NextInt(50) != 0 {
		return
	}

	startX := float64(chunkX*16 + int(rng.NextInt(16)))
	startY := float64(int(rng.NextInt(int(rng.NextInt(40)+8)) + 20))
	startZ := float64(chunkZ*16 + int(rng.NextInt(16)))
	for i := 0; i < 1; i++ {
		yaw := rng.NextFloat() * float32(math.Pi) * 2.0
		pitch := (rng.NextFloat() - 0.5) * 2.0 / 8.0
		width := (rng.NextFloat()*2.0 + rng.NextFloat()) * 2.0
		g.generateRavine(
			rng.NextLong(),
			targetChunkX, targetChunkZ,
			blocks, biomes,
			startX, startY, startZ,
			width, yaw, pitch,
			0, 0, 3.0,
		)
	}
}

// Translation reference:
// - net.minecraft.src.MapGenRavine.generateRavine(...)
func (g *mapGenRavine) generateRavine(
	seed int64,
	targetChunkX, targetChunkZ int,
	blocks []byte,
	biomes []Biome,
	x, y, z float64,
	width float32,
	yaw, pitch float32,
	start, end int,
	verticalScale float64,
) {
	rng := util.NewJavaRandom(seed)
	chunkMidX := float64(targetChunkX*16 + 8)
	chunkMidZ := float64(targetChunkZ*16 + 8)
	yawChange := float32(0)
	pitchChange := float32(0)

	if end <= 0 {
		maxLen := g.rngRange*16 - 16
		end = maxLen - int(rng.NextInt(maxLen/4))
	}

	isLarge := false
	if start == -1 {
		start = end / 2
		isLarge = true
	}

	shape := float32(1.0)
	for i := 0; i < 128; i++ {
		if i == 0 || rng.NextInt(3) == 0 {
			shape = 1.0 + rng.NextFloat()*rng.NextFloat()
		}
		g.yShape[i] = shape * shape
	}

	for ; start < end; start++ {
		radiusXZ := 1.5 + math.Sin(float64(float32(start)*float32(math.Pi)/float32(end)))*float64(width)
		radiusY := radiusXZ * verticalScale
		radiusXZ *= float64(rng.NextFloat())*0.25 + 0.75
		radiusY *= float64(rng.NextFloat())*0.25 + 0.75

		cosPitch := float32(math.Cos(float64(pitch)))
		sinPitch := float32(math.Sin(float64(pitch)))
		x += float64(float32(math.Cos(float64(yaw))) * cosPitch)
		y += float64(sinPitch)
		z += float64(float32(math.Sin(float64(yaw))) * cosPitch)

		pitch *= 0.7
		pitch += pitchChange * 0.05
		yaw += yawChange * 0.05
		pitchChange *= 0.8
		yawChange *= 0.5
		pitchChange += (rng.NextFloat() - rng.NextFloat()) * rng.NextFloat() * 2.0
		yawChange += (rng.NextFloat() - rng.NextFloat()) * rng.NextFloat() * 4.0

		if !isLarge && rng.NextInt(4) == 0 {
			continue
		}

		dxCenter := x - chunkMidX
		dzCenter := z - chunkMidZ
		remain := float64(end - start)
		maxDist := float64(width + 2.0 + 16.0)
		if dxCenter*dxCenter+dzCenter*dzCenter-remain*remain > maxDist*maxDist {
			return
		}

		if x < chunkMidX-16.0-radiusXZ*2.0 || z < chunkMidZ-16.0-radiusXZ*2.0 ||
			x > chunkMidX+16.0+radiusXZ*2.0 || z > chunkMidZ+16.0+radiusXZ*2.0 {
			continue
		}

		minX := floorDouble(x-radiusXZ) - targetChunkX*16 - 1
		maxX := floorDouble(x+radiusXZ) - targetChunkX*16 + 1
		minY := floorDouble(y-radiusY) - 1
		maxY := floorDouble(y+radiusY) + 1
		minZ := floorDouble(z-radiusXZ) - targetChunkZ*16 - 1
		maxZ := floorDouble(z+radiusXZ) - targetChunkZ*16 + 1

		if minX < 0 {
			minX = 0
		}
		if maxX > 16 {
			maxX = 16
		}
		if minY < 1 {
			minY = 1
		}
		if maxY > 120 {
			maxY = 120
		}
		if minZ < 0 {
			minZ = 0
		}
		if maxZ > 16 {
			maxZ = 16
		}

		containsWater := false
		for lx := minX; !containsWater && lx < maxX; lx++ {
			for lz := minZ; !containsWater && lz < maxZ; lz++ {
				for ly := maxY + 1; !containsWater && ly >= minY-1; ly-- {
					if ly < 0 || ly >= chunkHeight128 {
						continue
					}
					idx := chunkBlockIndex(lx, lz, ly)
					id := blocks[idx]
					if id == blockIDWaterFlow || id == blockIDWater {
						containsWater = true
					}
					if ly != minY-1 && lx != minX && lx != maxX-1 && lz != minZ && lz != maxZ-1 {
						ly = minY
					}
				}
			}
		}
		if containsWater {
			continue
		}

		for lx := minX; lx < maxX; lx++ {
			normX := (float64(lx+targetChunkX*16) + 0.5 - x) / radiusXZ
			for lz := minZ; lz < maxZ; lz++ {
				normZ := (float64(lz+targetChunkZ*16) + 0.5 - z) / radiusXZ
				index := chunkBlockIndex(lx, lz, maxY)
				hadGrass := false
				if normX*normX+normZ*normZ >= 1.0 {
					continue
				}
				for ly := maxY - 1; ly >= minY; ly-- {
					normY := (float64(ly) + 0.5 - y) / radiusY
					if (normX*normX+normZ*normZ)*float64(g.yShape[ly])+normY*normY/6.0 >= 1.0 {
						index--
						continue
					}
					id := blocks[index]
					if id == blockIDGrass {
						hadGrass = true
					}
					if id == blockIDStone || id == blockIDDirt || id == blockIDGrass {
						if ly < 10 {
							blocks[index] = blockIDLavaFlow
						} else {
							blocks[index] = blockIDAir
							if hadGrass && index > 0 && blocks[index-1] == blockIDDirt {
								blocks[index-1] = biomeTopBlockAtLocal(lx, lz, biomes)
							}
						}
					}
					index--
				}
			}
		}

		if isLarge {
			break
		}
	}
}

func chunkBlockIndex(localX, localZ, y int) int {
	return (localX*16+localZ)*chunkHeight128 + y
}

func biomeTopBlockAtLocal(localX, localZ int, biomes []Biome) byte {
	if len(biomes) == 0 {
		return blockIDGrass
	}
	idx := localZ + localX*16
	if idx < 0 || idx >= len(biomes) {
		return blockIDGrass
	}
	return biomes[idx].TopBlock
}
