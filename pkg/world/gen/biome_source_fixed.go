package gen

// FixedBiomeSource returns one biome for all positions.
// It is kept for deterministic tests and debug worlds.
type FixedBiomeSource struct {
	biome Biome
}

func NewFixedBiomeSource(biome Biome) *FixedBiomeSource {
	return &FixedBiomeSource{biome: biome}
}

func (s *FixedBiomeSource) GetBiomesForGeneration(reuse []Biome, x, z, width, depth int) []Biome {
	return fillFixedBiomeSlice(reuse, width, depth, s.biome)
}

func (s *FixedBiomeSource) LoadBlockGeneratorData(reuse []Biome, x, z, width, depth int) []Biome {
	return fillFixedBiomeSlice(reuse, width, depth, s.biome)
}

func fillFixedBiomeSlice(reuse []Biome, width, depth int, biome Biome) []Biome {
	want := width * depth
	if cap(reuse) < want {
		reuse = make([]Biome, want)
	} else {
		reuse = reuse[:want]
	}
	for i := range reuse {
		reuse[i] = biome
	}
	return reuse
}
