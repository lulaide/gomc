package gen

import "github.com/lulaide/gomc/pkg/util"

// NoiseGeneratorOctaves translates net.minecraft.src.NoiseGeneratorOctaves.
type NoiseGeneratorOctaves struct {
	generatorCollection []*NoiseGeneratorPerlin
	octaves             int
}

func NewNoiseGeneratorOctaves(rand *util.JavaRandom, octaves int) *NoiseGeneratorOctaves {
	gens := make([]*NoiseGeneratorPerlin, octaves)
	for i := 0; i < octaves; i++ {
		gens[i] = NewNoiseGeneratorPerlin(rand)
	}
	return &NoiseGeneratorOctaves{
		generatorCollection: gens,
		octaves:             octaves,
	}
}

// GenerateNoiseOctaves translates NoiseGeneratorOctaves#generateNoiseOctaves(..., int,int,int,int,int,int,double,double,double).
func (n *NoiseGeneratorOctaves) GenerateNoiseOctaves(out []float64, xOffset, yOffset, zOffset, xSize, ySize, zSize int, xScale, yScale, zScale float64) []float64 {
	want := xSize * ySize * zSize
	if out == nil || len(out) != want {
		out = make([]float64, want)
	} else {
		for i := range out {
			out[i] = 0
		}
	}

	amplitude := 1.0
	for i := 0; i < n.octaves; i++ {
		dx := float64(xOffset) * amplitude * xScale
		dy := float64(yOffset) * amplitude * yScale
		dz := float64(zOffset) * amplitude * zScale

		xBase := floorDoubleLong(dx)
		zBase := floorDoubleLong(dz)
		dx -= float64(xBase)
		dz -= float64(zBase)

		xBase %= 16777216
		zBase %= 16777216
		dx += float64(xBase)
		dz += float64(zBase)

		n.generatorCollection[i].PopulateNoiseArray(
			out,
			dx,
			dy,
			dz,
			xSize,
			ySize,
			zSize,
			xScale*amplitude,
			yScale*amplitude,
			zScale*amplitude,
			amplitude,
		)
		amplitude /= 2.0
	}

	return out
}

// GenerateNoiseOctaves2D translates the 2D overload:
// NoiseGeneratorOctaves#generateNoiseOctaves(double[], int, int, int, int, double, double, double)
func (n *NoiseGeneratorOctaves) GenerateNoiseOctaves2D(out []float64, xOffset, zOffset, xSize, zSize int, xScale, zScale, _ float64) []float64 {
	return n.GenerateNoiseOctaves(out, xOffset, 10, zOffset, xSize, 1, zSize, xScale, 1.0, zScale)
}
