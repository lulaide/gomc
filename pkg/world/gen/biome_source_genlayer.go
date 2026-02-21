package gen

// GenLayerBiomeSource translates the WorldChunkManager biome sampling path used
// by ChunkProviderGenerate in Minecraft 1.6.4.
//
// Translation reference:
// - net.minecraft.src.WorldChunkManager
// - net.minecraft.src.GenLayer.initializeAllBiomeGenerators(...)
type GenLayerBiomeSource struct {
	genBiomes       biomeGenLayer
	biomeIndexLayer biomeGenLayer
}

func NewGenLayerBiomeSource(seed int64, worldType WorldType) *GenLayerBiomeSource {
	genBiomes, biomeIndexLayer := initializeAllBiomeGenerators(seed, worldType)
	return &GenLayerBiomeSource{
		genBiomes:       genBiomes,
		biomeIndexLayer: biomeIndexLayer,
	}
}

func (s *GenLayerBiomeSource) GetBiomesForGeneration(reuse []Biome, x, z, width, depth int) []Biome {
	genLayerResetIntCache()
	reuse = ensureBiomeSlice(reuse, width*depth)
	ints := s.genBiomes.getInts(x, z, width, depth)
	for i := 0; i < width*depth; i++ {
		reuse[i] = BiomeByID(ints[i])
	}
	return reuse
}

func (s *GenLayerBiomeSource) LoadBlockGeneratorData(reuse []Biome, x, z, width, depth int) []Biome {
	genLayerResetIntCache()
	reuse = ensureBiomeSlice(reuse, width*depth)
	ints := s.biomeIndexLayer.getInts(x, z, width, depth)
	for i := 0; i < width*depth; i++ {
		reuse[i] = BiomeByID(ints[i])
	}
	return reuse
}

func ensureBiomeSlice(reuse []Biome, want int) []Biome {
	if cap(reuse) < want {
		return make([]Biome, want)
	}
	return reuse[:want]
}
