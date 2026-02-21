package util

import "math"

const (
	javaRandomMultiplier int64 = 0x5DEECE66D
	javaRandomAddend     int64 = 0xB
	javaRandomMask       int64 = (1 << 48) - 1
)

// JavaRandom translates java.util.Random from Minecraft 1.6.4 runtime behavior.
//
// Translation target:
// - java.util.Random#setSeed(long)
// - java.util.Random#next(int)
// - java.util.Random#nextInt(...)
// - java.util.Random#nextFloat()
// - java.util.Random#nextDouble()
type JavaRandom struct {
	seed                 int64
	nextNextGaussian     float64
	haveNextNextGaussian bool
}

func NewJavaRandom(seed int64) *JavaRandom {
	r := &JavaRandom{}
	r.SetSeed(seed)
	return r
}

func (r *JavaRandom) SetSeed(seed int64) {
	r.seed = (seed ^ javaRandomMultiplier) & javaRandomMask
	r.haveNextNextGaussian = false
}

func (r *JavaRandom) Next(bits int) int32 {
	r.seed = (r.seed*javaRandomMultiplier + javaRandomAddend) & javaRandomMask
	return int32(uint64(r.seed) >> (48 - bits))
}

// NextIntUnbounded matches java.util.Random#nextInt().
func (r *JavaRandom) NextIntUnbounded() int32 {
	return r.Next(32)
}

// NextInt matches java.util.Random#nextInt(int).
func (r *JavaRandom) NextInt(bound int) int32 {
	if bound <= 0 {
		panic("bound must be positive")
	}

	bound32 := int32(bound)
	if (bound32 & -bound32) == bound32 {
		return int32((int64(bound32) * int64(r.Next(31))) >> 31)
	}

	var bits int32
	var value int32
	for {
		bits = r.Next(31)
		value = bits % bound32
		if bits-value+(bound32-1) >= 0 {
			return value
		}
	}
}

func (r *JavaRandom) NextLong() int64 {
	high := int64(r.Next(32))
	low := int64(r.Next(32))
	return (high << 32) + low
}

func (r *JavaRandom) NextBoolean() bool {
	return r.Next(1) != 0
}

func (r *JavaRandom) NextFloat() float32 {
	return float32(r.Next(24)) / (1 << 24)
}

func (r *JavaRandom) NextDouble() float64 {
	high := int64(r.Next(26))
	low := int64(r.Next(27))
	return float64((high<<27)+low) / float64(uint64(1)<<53)
}

func (r *JavaRandom) NextBytes(bytes []byte) {
	i := 0
	for i < len(bytes) {
		rnd := r.NextIntUnbounded()
		n := minInt(len(bytes)-i, 4)
		for j := 0; j < n; j++ {
			bytes[i] = byte(rnd)
			i++
			rnd >>= 8
		}
	}
}

// NextGaussian matches java.util.Random#nextGaussian().
func (r *JavaRandom) NextGaussian() float64 {
	// Translation target: java.util.Random#nextGaussian()
	if r.haveNextNextGaussian {
		r.haveNextNextGaussian = false
		return r.nextNextGaussian
	}

	var v1, v2, s float64
	for {
		v1 = 2*r.NextDouble() - 1
		v2 = 2*r.NextDouble() - 1
		s = v1*v1 + v2*v2
		if s < 1 && s != 0 {
			break
		}
	}

	multiplier := math.Sqrt(-2 * math.Log(s) / s)
	r.nextNextGaussian = v2 * multiplier
	r.haveNextNextGaussian = true
	return v1 * multiplier
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
