package gen

// Biome captures the subset of BiomeGenBase fields used by ChunkProviderGenerate.
//
// Translation target:
// - net.minecraft.src.BiomeGenBase (biomeID/topBlock/fillerBlock/minHeight/maxHeight/temperature)
type Biome struct {
	ID          byte
	TopBlock    byte
	FillerBlock byte
	MinHeight   float32
	MaxHeight   float32
	Temperature float32
}

const (
	BiomeIDOcean               = 0
	BiomeIDPlains              = 1
	BiomeIDDesert              = 2
	BiomeIDExtremeHills        = 3
	BiomeIDForest              = 4
	BiomeIDTaiga               = 5
	BiomeIDSwampland           = 6
	BiomeIDRiver               = 7
	BiomeIDFrozenOcean         = 10
	BiomeIDFrozenRiver         = 11
	BiomeIDIcePlains           = 12
	BiomeIDIceMountains        = 13
	BiomeIDMushroomIsland      = 14
	BiomeIDMushroomIslandShore = 15
	BiomeIDBeach               = 16
	BiomeIDDesertHills         = 17
	BiomeIDForestHills         = 18
	BiomeIDTaigaHills          = 19
	BiomeIDExtremeHillsEdge    = 20
	BiomeIDJungle              = 21
	BiomeIDJungleHills         = 22
)

const (
	biomeTopGrass    = 2
	biomeTopSand     = 12
	biomeTopMycelium = 110
	biomeFillDirt    = 3
	biomeFillSand    = 12
)

var (
	biomesByID  [256]Biome
	biomeExists [256]bool

	// Translation reference:
	// - net.minecraft.src.BiomeGenBase static biome declarations.
	OceanBiome = registerBiome(Biome{
		ID:          BiomeIDOcean,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   -1.0,
		MaxHeight:   0.4,
		Temperature: 0.5,
	})
	PlainsBiome = registerBiome(Biome{
		ID:          BiomeIDPlains,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.1,
		MaxHeight:   0.3,
		Temperature: 0.8,
	})
	DesertBiome = registerBiome(Biome{
		ID:          BiomeIDDesert,
		TopBlock:    biomeTopSand,
		FillerBlock: biomeFillSand,
		MinHeight:   0.1,
		MaxHeight:   0.2,
		Temperature: 2.0,
	})
	ExtremeHillsBiome = registerBiome(Biome{
		ID:          BiomeIDExtremeHills,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.3,
		MaxHeight:   1.5,
		Temperature: 0.2,
	})
	ForestBiome = registerBiome(Biome{
		ID:          BiomeIDForest,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.1,
		MaxHeight:   0.3,
		Temperature: 0.7,
	})
	TaigaBiome = registerBiome(Biome{
		ID:          BiomeIDTaiga,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.1,
		MaxHeight:   0.4,
		Temperature: 0.05,
	})
	SwamplandBiome = registerBiome(Biome{
		ID:          BiomeIDSwampland,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   -0.2,
		MaxHeight:   0.1,
		Temperature: 0.8,
	})
	RiverBiome = registerBiome(Biome{
		ID:          BiomeIDRiver,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   -0.5,
		MaxHeight:   0.0,
		Temperature: 0.5,
	})
	FrozenOceanBiome = registerBiome(Biome{
		ID:          BiomeIDFrozenOcean,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   -1.0,
		MaxHeight:   0.5,
		Temperature: 0.0,
	})
	FrozenRiverBiome = registerBiome(Biome{
		ID:          BiomeIDFrozenRiver,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   -0.5,
		MaxHeight:   0.0,
		Temperature: 0.0,
	})
	IcePlainsBiome = registerBiome(Biome{
		ID:          BiomeIDIcePlains,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.1,
		MaxHeight:   0.3,
		Temperature: 0.0,
	})
	IceMountainsBiome = registerBiome(Biome{
		ID:          BiomeIDIceMountains,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.3,
		MaxHeight:   1.3,
		Temperature: 0.0,
	})
	MushroomIslandBiome = registerBiome(Biome{
		ID:          BiomeIDMushroomIsland,
		TopBlock:    biomeTopMycelium,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.2,
		MaxHeight:   1.0,
		Temperature: 0.9,
	})
	MushroomIslandShoreBiome = registerBiome(Biome{
		ID:          BiomeIDMushroomIslandShore,
		TopBlock:    biomeTopMycelium,
		FillerBlock: biomeFillDirt,
		MinHeight:   -1.0,
		MaxHeight:   0.1,
		Temperature: 0.9,
	})
	BeachBiome = registerBiome(Biome{
		ID:          BiomeIDBeach,
		TopBlock:    biomeTopSand,
		FillerBlock: biomeFillSand,
		MinHeight:   0.0,
		MaxHeight:   0.1,
		Temperature: 0.8,
	})
	DesertHillsBiome = registerBiome(Biome{
		ID:          BiomeIDDesertHills,
		TopBlock:    biomeTopSand,
		FillerBlock: biomeFillSand,
		MinHeight:   0.3,
		MaxHeight:   0.8,
		Temperature: 2.0,
	})
	ForestHillsBiome = registerBiome(Biome{
		ID:          BiomeIDForestHills,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.3,
		MaxHeight:   0.7,
		Temperature: 0.7,
	})
	TaigaHillsBiome = registerBiome(Biome{
		ID:          BiomeIDTaigaHills,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.3,
		MaxHeight:   0.8,
		Temperature: 0.05,
	})
	ExtremeHillsEdgeBiome = registerBiome(Biome{
		ID:          BiomeIDExtremeHillsEdge,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.2,
		MaxHeight:   0.8,
		Temperature: 0.2,
	})
	JungleBiome = registerBiome(Biome{
		ID:          BiomeIDJungle,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   0.2,
		MaxHeight:   0.4,
		Temperature: 1.2,
	})
	JungleHillsBiome = registerBiome(Biome{
		ID:          BiomeIDJungleHills,
		TopBlock:    biomeTopGrass,
		FillerBlock: biomeFillDirt,
		MinHeight:   1.8,
		MaxHeight:   0.5,
		Temperature: 1.2,
	})
)

func registerBiome(b Biome) Biome {
	i := int(b.ID)
	if i >= 0 && i < len(biomesByID) {
		biomesByID[i] = b
		biomeExists[i] = true
	}
	return b
}

func BiomeByID(id int) Biome {
	if id >= 0 && id < len(biomesByID) && biomeExists[id] {
		return biomesByID[id]
	}
	return PlainsBiome
}

// BiomeSource mirrors the biome query methods used by ChunkProviderGenerate.
//
// Translation target:
// - net.minecraft.src.WorldChunkManager#getBiomesForGeneration(...)
// - net.minecraft.src.WorldChunkManager#loadBlockGeneratorData(...)
type BiomeSource interface {
	GetBiomesForGeneration(reuse []Biome, x, z, width, depth int) []Biome
	LoadBlockGeneratorData(reuse []Biome, x, z, width, depth int) []Biome
}
