package gen

import (
	"math"

	"github.com/lulaide/gomc/pkg/util"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

const (
	blockIDAir              byte = 0
	blockIDStone            byte = 1
	blockIDGrass            byte = 2
	blockIDDirt             byte = 3
	blockIDCobblestone      byte = 4
	blockIDBedrock          byte = 7
	blockIDSapling          byte = 6
	blockIDWaterFlow        byte = 8
	blockIDWater            byte = 9
	blockIDLavaFlow         byte = 10
	blockIDLava             byte = 11
	blockIDSand             byte = 12
	blockIDGravel           byte = 13
	blockIDLog              byte = 17
	blockIDLeaves           byte = 18
	blockIDSandstone        byte = 24
	blockIDTallGrass        byte = 31
	blockIDDeadBush         byte = 32
	blockIDFlowerYellow     byte = 37
	blockIDFlowerRed        byte = 38
	blockIDMushroomBrown    byte = 39
	blockIDMushroomRed      byte = 40
	blockIDCactus           byte = 81
	blockIDMossyCobble      byte = 48
	blockIDMobSpawner       byte = 52
	blockIDChest            byte = 54
	blockIDFarmland         byte = 60
	blockIDIce              byte = 79
	blockIDClay             byte = 82
	blockIDReed             byte = 83
	blockIDPumpkin          byte = 86
	blockIDMushroomCapBrown byte = 99
	blockIDMushroomCapRed   byte = 100
	blockIDVine             byte = 106
	blockIDMycelium         byte = 110
	blockIDWaterLily        byte = 111
	blockIDCocoaPlant       byte = 127
)

// ChunkProviderGenerate translates the core terrain generation path from
// net.minecraft.src.ChunkProviderGenerate.
//
// Implemented methods:
// - generateTerrain(...)
// - replaceBlocksForBiome(...)
// - initializeNoiseField(...)
// - provideChunk(...) (terrain + biome replacement only; map features/caves pending)
type ChunkProviderGenerate struct {
	rand        *util.JavaRandom
	biomeSource BiomeSource
	worldSeed   int64

	noiseGen1 *NoiseGeneratorOctaves
	noiseGen2 *NoiseGeneratorOctaves
	noiseGen3 *NoiseGeneratorOctaves
	noiseGen4 *NoiseGeneratorOctaves
	noiseGen5 *NoiseGeneratorOctaves
	noiseGen6 *NoiseGeneratorOctaves

	noiseArray []float64
	stoneNoise []float64

	biomesForGeneration []Biome
	noise1              []float64
	noise2              []float64
	noise3              []float64
	noise5              []float64
	noise6              []float64

	parabolicField []float32
	caveGenerator  *mapGenCaves
	ravineGen      *mapGenRavine
}

func NewChunkProviderGenerate(seed int64, biomeSource BiomeSource) *ChunkProviderGenerate {
	if biomeSource == nil {
		biomeSource = NewFixedBiomeSource(PlainsBiome)
	}

	r := util.NewJavaRandom(seed)
	return &ChunkProviderGenerate{
		rand:          r,
		biomeSource:   biomeSource,
		worldSeed:     seed,
		noiseGen1:     NewNoiseGeneratorOctaves(r, 16),
		noiseGen2:     NewNoiseGeneratorOctaves(r, 16),
		noiseGen3:     NewNoiseGeneratorOctaves(r, 8),
		noiseGen4:     NewNoiseGeneratorOctaves(r, 4),
		noiseGen5:     NewNoiseGeneratorOctaves(r, 10),
		noiseGen6:     NewNoiseGeneratorOctaves(r, 16),
		stoneNoise:    make([]float64, 256),
		caveGenerator: newMapGenCaves(seed),
		ravineGen:     newMapGenRavine(seed),
	}
}

// GenerateChunk is the entrypoint used by server world state.
func (p *ChunkProviderGenerate) GenerateChunk(chunkX, chunkZ int32) *chunk.Chunk {
	return p.provideChunk(int(chunkX), int(chunkZ))
}

// provideChunk translates ChunkProviderGenerate#provideChunk(int,int), excluding
// caves/ravines/structures/decorators for now.
func (p *ChunkProviderGenerate) provideChunk(chunkX, chunkZ int) *chunk.Chunk {
	p.rand.SetSeed(int64(chunkX)*341873128712 + int64(chunkZ)*132897987541)

	blocks := make([]byte, 32768)
	blockMetadata := make([]byte, len(blocks))
	registerBlockMetadataBuffer(blocks, blockMetadata)
	defer unregisterBlockMetadataBuffer(blocks)

	p.generateTerrain(chunkX, chunkZ, blocks)
	p.biomesForGeneration = p.biomeSource.LoadBlockGeneratorData(p.biomesForGeneration, chunkX*16, chunkZ*16, 16, 16)
	p.replaceBlocksForBiome(chunkX, chunkZ, blocks, p.biomesForGeneration)
	if p.caveGenerator != nil {
		p.caveGenerator.generate(chunkX, chunkZ, blocks, p.biomesForGeneration)
	}
	if p.ravineGen != nil {
		p.ravineGen.generate(chunkX, chunkZ, blocks, p.biomesForGeneration)
	}
	p.populateChunkDecorations(chunkX, chunkZ, blocks)

	ch := chunkFromLegacyByteArray(nil, int32(chunkX), int32(chunkZ), blocks, blockMetadata)
	biomes := ch.GetBiomeArray()
	for i := 0; i < len(biomes) && i < len(p.biomesForGeneration); i++ {
		biomes[i] = p.biomesForGeneration[i].ID
	}
	ch.SetBiomeArray(biomes)
	ch.GenerateSkylightMap()
	return ch
}

// generateTerrain translates ChunkProviderGenerate#generateTerrain(int,int,byte[]).
func (p *ChunkProviderGenerate) generateTerrain(chunkX, chunkZ int, blocks []byte) {
	const (
		var4 = 4
		var5 = 16
		var6 = 63
		var8 = 17
	)
	var7 := var4 + 1
	var9 := var4 + 1

	p.biomesForGeneration = p.biomeSource.GetBiomesForGeneration(
		p.biomesForGeneration,
		chunkX*4-2,
		chunkZ*4-2,
		var7+5,
		var9+5,
	)
	p.noiseArray = p.initializeNoiseField(p.noiseArray, chunkX*var4, 0, chunkZ*var4, var7, var8, var9)

	for var10 := 0; var10 < var4; var10++ {
		for var11 := 0; var11 < var4; var11++ {
			for var12 := 0; var12 < var5; var12++ {
				var13 := 0.125
				var15 := p.noiseArray[((var10+0)*var9+var11+0)*var8+var12+0]
				var17 := p.noiseArray[((var10+0)*var9+var11+1)*var8+var12+0]
				var19 := p.noiseArray[((var10+1)*var9+var11+0)*var8+var12+0]
				var21 := p.noiseArray[((var10+1)*var9+var11+1)*var8+var12+0]
				var23 := (p.noiseArray[((var10+0)*var9+var11+0)*var8+var12+1] - var15) * var13
				var25 := (p.noiseArray[((var10+0)*var9+var11+1)*var8+var12+1] - var17) * var13
				var27 := (p.noiseArray[((var10+1)*var9+var11+0)*var8+var12+1] - var19) * var13
				var29 := (p.noiseArray[((var10+1)*var9+var11+1)*var8+var12+1] - var21) * var13

				for var31 := 0; var31 < 8; var31++ {
					var32 := 0.25
					var34 := var15
					var36 := var17
					var38 := (var19 - var15) * var32
					var40 := (var21 - var17) * var32

					for var42 := 0; var42 < 4; var42++ {
						var43 := (var42+var10*4)<<11 | (var11 * 4 << 7) | (var12*8 + var31)
						var44 := 128
						var43 -= var44
						var45 := 0.25
						var49 := (var36 - var34) * var45
						var47 := var34 - var49

						for var51 := 0; var51 < 4; var51++ {
							var47 += var49
							var43 += var44
							if var47 > 0.0 {
								blocks[var43] = blockIDStone
							} else if var12*8+var31 < var6 {
								blocks[var43] = blockIDWater
							} else {
								blocks[var43] = blockIDAir
							}
						}

						var34 += var38
						var36 += var40
					}

					var15 += var23
					var17 += var25
					var19 += var27
					var21 += var29
				}
			}
		}
	}
}

// replaceBlocksForBiome translates ChunkProviderGenerate#replaceBlocksForBiome(...).
func (p *ChunkProviderGenerate) replaceBlocksForBiome(chunkX, chunkZ int, blocks []byte, biomes []Biome) {
	const (
		seaLevel = 63
		scale    = 0.03125
	)

	p.stoneNoise = p.noiseGen4.GenerateNoiseOctaves(
		p.stoneNoise,
		chunkX*16,
		chunkZ*16,
		0,
		16,
		16,
		1,
		scale*2.0,
		scale*2.0,
		scale*2.0,
	)

	for var8 := 0; var8 < 16; var8++ {
		for var9 := 0; var9 < 16; var9++ {
			biome := biomes[var9+var8*16]
			temperature := biome.Temperature
			var12 := int(p.stoneNoise[var8+var9*16]/3.0 + 3.0 + p.rand.NextDouble()*0.25)
			var13 := -1
			var14 := biome.TopBlock
			var15 := biome.FillerBlock

			for var16 := 127; var16 >= 0; var16-- {
				var17 := (var9*16+var8)*128 + var16
				if var16 <= int(p.rand.NextInt(5)) {
					blocks[var17] = blockIDBedrock
					continue
				}

				var18 := blocks[var17]
				if var18 == blockIDAir {
					var13 = -1
					continue
				}
				if var18 != blockIDStone {
					continue
				}

				if var13 == -1 {
					if var12 <= 0 {
						var14 = blockIDAir
						var15 = blockIDStone
					} else if var16 >= seaLevel-4 && var16 <= seaLevel+1 {
						var14 = biome.TopBlock
						var15 = biome.FillerBlock
					}

					if var16 < seaLevel && var14 == blockIDAir {
						if temperature < 0.15 {
							var14 = blockIDIce
						} else {
							var14 = blockIDWater
						}
					}

					var13 = var12
					if var16 >= seaLevel-1 {
						blocks[var17] = var14
					} else {
						blocks[var17] = var15
					}
				} else if var13 > 0 {
					var13--
					blocks[var17] = var15

					if var13 == 0 && var15 == blockIDSand {
						var13 = int(p.rand.NextInt(4))
						var15 = blockIDSandstone
					}
				}
			}
		}
	}
}

// initializeNoiseField translates ChunkProviderGenerate#initializeNoiseField(...).
func (p *ChunkProviderGenerate) initializeNoiseField(out []float64, par2, par3, par4, par5, par6, par7 int) []float64 {
	want := par5 * par6 * par7
	if out == nil || len(out) != want {
		out = make([]float64, want)
	}

	if p.parabolicField == nil {
		p.parabolicField = make([]float32, 25)
		for x := -2; x <= 2; x++ {
			for z := -2; z <= 2; z++ {
				v := float32(10.0 / math.Sqrt(float64(float32(x*x+z*z)+0.2)))
				p.parabolicField[x+2+(z+2)*5] = v
			}
		}
	}

	var44 := 684.412
	var45 := 684.412
	p.noise5 = p.noiseGen5.GenerateNoiseOctaves2D(p.noise5, par2, par4, par5, par7, 1.121, 1.121, 0.5)
	p.noise6 = p.noiseGen6.GenerateNoiseOctaves2D(p.noise6, par2, par4, par5, par7, 200.0, 200.0, 0.5)
	p.noise3 = p.noiseGen3.GenerateNoiseOctaves(p.noise3, par2, par3, par4, par5, par6, par7, var44/80.0, var45/160.0, var44/80.0)
	p.noise1 = p.noiseGen1.GenerateNoiseOctaves(p.noise1, par2, par3, par4, par5, par6, par7, var44, var45, var44)
	p.noise2 = p.noiseGen2.GenerateNoiseOctaves(p.noise2, par2, par3, par4, par5, par6, par7, var44, var45, var44)

	var12 := 0
	var13 := 0

	for var14 := 0; var14 < par5; var14++ {
		for var15 := 0; var15 < par7; var15++ {
			var16 := float32(0.0)
			var17 := float32(0.0)
			var18 := float32(0.0)
			var19 := 2
			var20 := p.biomesForGeneration[var14+2+(var15+2)*(par5+5)]

			for var21 := -var19; var21 <= var19; var21++ {
				for var22 := -var19; var22 <= var19; var22++ {
					var23 := p.biomesForGeneration[var14+var21+2+(var15+var22+2)*(par5+5)]
					var24 := p.parabolicField[var21+2+(var22+2)*5] / (var23.MinHeight + 2.0)
					if var23.MinHeight > var20.MinHeight {
						var24 /= 2.0
					}
					var16 += var23.MaxHeight * var24
					var17 += var23.MinHeight * var24
					var18 += var24
				}
			}

			var16 /= var18
			var17 /= var18
			var16 = var16*0.9 + 0.1
			var17 = (var17*4.0 - 1.0) / 8.0

			var46 := p.noise6[var13] / 8000.0
			if var46 < 0.0 {
				var46 = -var46 * 0.3
			}
			var46 = var46*3.0 - 2.0
			if var46 < 0.0 {
				var46 /= 2.0
				if var46 < -1.0 {
					var46 = -1.0
				}
				var46 /= 1.4
				var46 /= 2.0
			} else {
				if var46 > 1.0 {
					var46 = 1.0
				}
				var46 /= 8.0
			}

			var13++

			for var47 := 0; var47 < par6; var47++ {
				var48 := float64(var17)
				var26 := float64(var16)
				var48 += var46 * 0.2
				var48 = var48 * float64(par6) / 16.0
				var28 := float64(par6)/2.0 + var48*4.0
				var30 := 0.0
				var32 := (float64(var47) - var28) * 12.0 * 128.0 / 128.0 / var26
				if var32 < 0.0 {
					var32 *= 4.0
				}

				var34 := p.noise1[var12] / 512.0
				var36 := p.noise2[var12] / 512.0
				var38 := (p.noise3[var12]/10.0 + 1.0) / 2.0
				if var38 < 0.0 {
					var30 = var34
				} else if var38 > 1.0 {
					var30 = var36
				} else {
					var30 = var34 + (var36-var34)*var38
				}

				var30 -= var32
				if var47 > par6-4 {
					var40 := float64(float32(var47-(par6-4)) / 3.0)
					var30 = var30*(1.0-var40) + -10.0*var40
				}

				out[var12] = var30
				var12++
			}
		}
	}

	return out
}

// chunkFromLegacyByteArray translates Chunk(World, byte[], int, int) construction
// used by ChunkProviderGenerate in 1.6.4.
func chunkFromLegacyByteArray(world chunk.WorldBridge, chunkX, chunkZ int32, blocks []byte, blockMetadata []byte) *chunk.Chunk {
	ch := chunk.NewChunk(world, chunkX, chunkZ)
	height := len(blocks) / 256
	arr := ch.GetBlockStorageArray()
	withMetadata := len(blockMetadata) == len(blocks)

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < height; y++ {
				id := int(blocks[x<<11|z<<7|y] & 0xFF)
				if id == 0 {
					continue
				}

				sectionY := y >> 4
				section := arr[sectionY]
				if section == nil {
					section = chunk.NewExtendedBlockStorage(sectionY<<4, true)
					arr[sectionY] = section
				}
				section.SetExtBlockID(x, y&15, z, id)
				if withMetadata {
					meta := int(blockMetadata[x<<11|z<<7|y] & 0xF)
					section.SetExtBlockMetadata(x, y&15, z, meta)
				}
			}
		}
	}

	ch.SetBlockStorageArray(arr)
	return ch
}
