package gen

import "github.com/lulaide/gomc/pkg/util"

// NoiseGeneratorPerlin translates net.minecraft.src.NoiseGeneratorPerlin.
type NoiseGeneratorPerlin struct {
	permutations [512]int
	xCoord       float64
	yCoord       float64
	zCoord       float64
}

func NewNoiseGeneratorPerlin(rand *util.JavaRandom) *NoiseGeneratorPerlin {
	n := &NoiseGeneratorPerlin{
		xCoord: rand.NextDouble() * 256.0,
		yCoord: rand.NextDouble() * 256.0,
		zCoord: rand.NextDouble() * 256.0,
	}

	for i := 0; i < 256; i++ {
		n.permutations[i] = i
	}

	for i := 0; i < 256; i++ {
		j := int(rand.NextInt(256-i)) + i
		n.permutations[i], n.permutations[j] = n.permutations[j], n.permutations[i]
		n.permutations[i+256] = n.permutations[i]
	}

	return n
}

func (n *NoiseGeneratorPerlin) lerp(a, b, t float64) float64 {
	return b + a*(t-b)
}

// func76309A matches NoiseGeneratorPerlin#func_76309_a(int,double,double).
func (n *NoiseGeneratorPerlin) func76309A(hash int, x, z float64) float64 {
	h := hash & 15
	u := float64(1-((h&8)>>3)) * x
	v := 0.0
	if h >= 4 {
		if h != 12 && h != 14 {
			v = z
		} else {
			v = x
		}
	}
	if (h & 1) != 0 {
		u = -u
	}
	if (h & 2) != 0 {
		v = -v
	}
	return u + v
}

func (n *NoiseGeneratorPerlin) grad(hash int, x, y, z float64) float64 {
	h := hash & 15
	u := y
	if h < 8 {
		u = x
	}
	v := 0.0
	if h < 4 {
		v = y
	} else if h != 12 && h != 14 {
		v = z
	} else {
		v = x
	}
	if (h & 1) != 0 {
		u = -u
	}
	if (h & 2) != 0 {
		v = -v
	}
	return u + v
}

// PopulateNoiseArray translates NoiseGeneratorPerlin#populateNoiseArray(...).
func (n *NoiseGeneratorPerlin) PopulateNoiseArray(out []float64, xOffset, yOffset, zOffset float64, xSize, ySize, zSize int, xScale, yScale, zScale, noiseScale float64) {
	if ySize == 1 {
		idx := 0
		invNoiseScale := 1.0 / noiseScale

		for x := 0; x < xSize; x++ {
			dx := xOffset + float64(x)*xScale + n.xCoord
			xFloor := floorDouble(dx)
			xMask := xFloor & 255
			dx -= float64(xFloor)
			fadeX := dx * dx * dx * (dx*(dx*6.0-15.0) + 10.0)

			for z := 0; z < zSize; z++ {
				dz := zOffset + float64(z)*zScale + n.zCoord
				zFloor := floorDouble(dz)
				zMask := zFloor & 255
				dz -= float64(zFloor)
				fadeZ := dz * dz * dz * (dz*(dz*6.0-15.0) + 10.0)

				a := n.permutations[xMask]
				aa := n.permutations[a] + zMask
				b := n.permutations[xMask+1]
				ba := n.permutations[b] + zMask

				xz0 := n.lerp(fadeX, n.func76309A(n.permutations[aa], dx, dz), n.grad(n.permutations[ba], dx-1.0, 0.0, dz))
				xz1 := n.lerp(fadeX, n.grad(n.permutations[aa+1], dx, 0.0, dz-1.0), n.grad(n.permutations[ba+1], dx-1.0, 0.0, dz-1.0))
				value := n.lerp(fadeZ, xz0, xz1)
				out[idx] += value * invNoiseScale
				idx++
			}
		}
		return
	}

	idx := 0
	invNoiseScale := 1.0 / noiseScale
	prevYMask := -1
	var (
		x00 float64
		x10 float64
		x01 float64
		x11 float64
	)

	for x := 0; x < xSize; x++ {
		dx := xOffset + float64(x)*xScale + n.xCoord
		xFloor := floorDouble(dx)
		xMask := xFloor & 255
		dx -= float64(xFloor)
		fadeX := dx * dx * dx * (dx*(dx*6.0-15.0) + 10.0)

		for z := 0; z < zSize; z++ {
			dz := zOffset + float64(z)*zScale + n.zCoord
			zFloor := floorDouble(dz)
			zMask := zFloor & 255
			dz -= float64(zFloor)
			fadeZ := dz * dz * dz * (dz*(dz*6.0-15.0) + 10.0)

			for y := 0; y < ySize; y++ {
				dy := yOffset + float64(y)*yScale + n.yCoord
				yFloor := floorDouble(dy)
				yMask := yFloor & 255
				dy -= float64(yFloor)
				fadeY := dy * dy * dy * (dy*(dy*6.0-15.0) + 10.0)

				if y == 0 || yMask != prevYMask {
					prevYMask = yMask
					a := n.permutations[xMask] + yMask
					aa := n.permutations[a] + zMask
					ab := n.permutations[a+1] + zMask
					b := n.permutations[xMask+1] + yMask
					ba := n.permutations[b] + zMask
					bb := n.permutations[b+1] + zMask

					x00 = n.lerp(fadeX, n.grad(n.permutations[aa], dx, dy, dz), n.grad(n.permutations[ba], dx-1.0, dy, dz))
					x10 = n.lerp(fadeX, n.grad(n.permutations[ab], dx, dy-1.0, dz), n.grad(n.permutations[bb], dx-1.0, dy-1.0, dz))
					x01 = n.lerp(fadeX, n.grad(n.permutations[aa+1], dx, dy, dz-1.0), n.grad(n.permutations[ba+1], dx-1.0, dy, dz-1.0))
					x11 = n.lerp(fadeX, n.grad(n.permutations[ab+1], dx, dy-1.0, dz-1.0), n.grad(n.permutations[bb+1], dx-1.0, dy-1.0, dz-1.0))
				}

				yz0 := n.lerp(fadeY, x00, x10)
				yz1 := n.lerp(fadeY, x01, x11)
				value := n.lerp(fadeZ, yz0, yz1)
				out[idx] += value * invNoiseScale
				idx++
			}
		}
	}
}
