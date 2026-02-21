package gen

const (
	genLayerMultiplier int64 = 6364136223846793005
	genLayerAddend     int64 = 1442695040888963407
)

type biomeGenLayer interface {
	initWorldGenSeed(seed int64)
	getInts(x, z, width, height int) []int
}

type baseGenLayer struct {
	worldGenSeed int64
	parent       biomeGenLayer
	chunkSeed    int64
	baseSeed     int64
}

func newBaseGenLayer(seed int64) baseGenLayer {
	baseSeed := seed
	baseSeed = baseSeed*(baseSeed*genLayerMultiplier+genLayerAddend) + seed
	baseSeed = baseSeed*(baseSeed*genLayerMultiplier+genLayerAddend) + seed
	baseSeed = baseSeed*(baseSeed*genLayerMultiplier+genLayerAddend) + seed
	return baseGenLayer{
		baseSeed: baseSeed,
	}
}

func (l *baseGenLayer) initWorldGenSeed(seed int64) {
	l.worldGenSeed = seed
	if l.parent != nil {
		l.parent.initWorldGenSeed(seed)
	}

	l.worldGenSeed = l.worldGenSeed*(l.worldGenSeed*genLayerMultiplier+genLayerAddend) + l.baseSeed
	l.worldGenSeed = l.worldGenSeed*(l.worldGenSeed*genLayerMultiplier+genLayerAddend) + l.baseSeed
	l.worldGenSeed = l.worldGenSeed*(l.worldGenSeed*genLayerMultiplier+genLayerAddend) + l.baseSeed
}

func (l *baseGenLayer) initChunkSeed(xSeed, zSeed int64) {
	l.chunkSeed = l.worldGenSeed
	l.chunkSeed = l.chunkSeed*(l.chunkSeed*genLayerMultiplier+genLayerAddend) + xSeed
	l.chunkSeed = l.chunkSeed*(l.chunkSeed*genLayerMultiplier+genLayerAddend) + zSeed
	l.chunkSeed = l.chunkSeed*(l.chunkSeed*genLayerMultiplier+genLayerAddend) + xSeed
	l.chunkSeed = l.chunkSeed*(l.chunkSeed*genLayerMultiplier+genLayerAddend) + zSeed
}

func (l *baseGenLayer) nextInt(bound int) int {
	v := int((l.chunkSeed >> 24) % int64(bound))
	if v < 0 {
		v += bound
	}
	l.chunkSeed = l.chunkSeed*(l.chunkSeed*genLayerMultiplier+genLayerAddend) + l.worldGenSeed
	return v
}

type genLayerIsland struct {
	baseGenLayer
}

func newGenLayerIsland(seed int64) *genLayerIsland {
	return &genLayerIsland{baseGenLayer: newBaseGenLayer(seed)}
}

func (l *genLayerIsland) getInts(x, z, width, height int) []int {
	out := genLayerGetIntCache(width * height)
	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			l.initChunkSeed(int64(x+dx), int64(z+dz))
			if l.nextInt(10) == 0 {
				out[dx+dz*width] = 1
			} else {
				out[dx+dz*width] = 0
			}
		}
	}
	if x > -width && x <= 0 && z > -height && z <= 0 {
		out[-x+-z*width] = 1
	}
	return out
}

type genLayerFuzzyZoom struct {
	baseGenLayer
}

func newGenLayerFuzzyZoom(seed int64, parent biomeGenLayer) *genLayerFuzzyZoom {
	l := &genLayerFuzzyZoom{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerFuzzyZoom) getInts(x, z, width, height int) []int {
	px := x >> 1
	pz := z >> 1
	pw := (width >> 1) + 3
	ph := (height >> 1) + 3
	parentInts := l.parent.getInts(px, pz, pw, ph)
	zoom := genLayerGetIntCache(pw * 2 * ph * 2)
	zw := pw << 1

	for iz := 0; iz < ph-1; iz++ {
		row2 := iz << 1
		dst := row2 * zw
		nw := parentInts[(iz+0)*pw]
		sw := parentInts[(iz+1)*pw]
		for ix := 0; ix < pw-1; ix++ {
			l.initChunkSeed(int64((ix+px)<<1), int64((iz+pz)<<1))
			ne := parentInts[ix+1+(iz+0)*pw]
			se := parentInts[ix+1+(iz+1)*pw]
			zoom[dst] = nw
			zoom[dst+zw] = l.choose2(nw, sw)
			dst++
			zoom[dst] = l.choose2(nw, ne)
			zoom[dst+zw] = l.choose4(nw, ne, sw, se)
			dst++
			nw = ne
			sw = se
		}
	}

	out := genLayerGetIntCache(width * height)
	for dz := 0; dz < height; dz++ {
		srcOff := (dz+(z&1))*zw + (x & 1)
		dstOff := dz * width
		copy(out[dstOff:dstOff+width], zoom[srcOff:srcOff+width])
	}
	return out
}

func (l *genLayerFuzzyZoom) choose2(a, b int) int {
	if l.nextInt(2) == 0 {
		return a
	}
	return b
}

func (l *genLayerFuzzyZoom) choose4(a, b, c, d int) int {
	switch l.nextInt(4) {
	case 0:
		return a
	case 1:
		return b
	case 2:
		return c
	default:
		return d
	}
}

type genLayerAddIsland struct {
	baseGenLayer
}

func newGenLayerAddIsland(seed int64, parent biomeGenLayer) *genLayerAddIsland {
	l := &genLayerAddIsland{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerAddIsland) getInts(x, z, width, height int) []int {
	px := x - 1
	pz := z - 1
	pw := width + 2
	ph := height + 2
	parentInts := l.parent.getInts(px, pz, pw, ph)
	out := genLayerGetIntCache(width * height)

	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			nw := parentInts[dx+0+(dz+0)*pw]
			ne := parentInts[dx+2+(dz+0)*pw]
			sw := parentInts[dx+0+(dz+2)*pw]
			se := parentInts[dx+2+(dz+2)*pw]
			center := parentInts[dx+1+(dz+1)*pw]
			l.initChunkSeed(int64(dx+x), int64(dz+z))

			if center == 0 && (nw != 0 || ne != 0 || sw != 0 || se != 0) {
				count := 1
				chosen := 1
				if nw != 0 && l.nextInt(count) == 0 {
					chosen = nw
				}
				count++
				if ne != 0 && l.nextInt(count) == 0 {
					chosen = ne
				}
				count++
				if sw != 0 && l.nextInt(count) == 0 {
					chosen = sw
				}
				count++
				if se != 0 && l.nextInt(count) == 0 {
					chosen = se
				}
				if l.nextInt(3) == 0 {
					out[dx+dz*width] = chosen
				} else if chosen == BiomeIDIcePlains {
					out[dx+dz*width] = BiomeIDFrozenOcean
				} else {
					out[dx+dz*width] = 0
				}
				continue
			}

			if center > 0 && (nw == 0 || ne == 0 || sw == 0 || se == 0) {
				if l.nextInt(5) == 0 {
					if center == BiomeIDIcePlains {
						out[dx+dz*width] = BiomeIDFrozenOcean
					} else {
						out[dx+dz*width] = 0
					}
				} else {
					out[dx+dz*width] = center
				}
				continue
			}

			out[dx+dz*width] = center
		}
	}
	return out
}

type genLayerZoom struct {
	baseGenLayer
}

func newGenLayerZoom(seed int64, parent biomeGenLayer) *genLayerZoom {
	l := &genLayerZoom{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func genLayerMagnify(seed int64, parent biomeGenLayer, times int) biomeGenLayer {
	var out biomeGenLayer = parent
	for i := 0; i < times; i++ {
		out = newGenLayerZoom(seed+int64(i), out)
	}
	return out
}

func (l *genLayerZoom) getInts(x, z, width, height int) []int {
	px := x >> 1
	pz := z >> 1
	pw := (width >> 1) + 3
	ph := (height >> 1) + 3
	parentInts := l.parent.getInts(px, pz, pw, ph)
	zoom := genLayerGetIntCache(pw * 2 * ph * 2)
	zw := pw << 1

	for iz := 0; iz < ph-1; iz++ {
		row2 := iz << 1
		dst := row2 * zw
		nw := parentInts[(iz+0)*pw]
		sw := parentInts[(iz+1)*pw]
		for ix := 0; ix < pw-1; ix++ {
			l.initChunkSeed(int64((ix+px)<<1), int64((iz+pz)<<1))
			ne := parentInts[ix+1+(iz+0)*pw]
			se := parentInts[ix+1+(iz+1)*pw]
			zoom[dst] = nw
			zoom[dst+zw] = l.choose2(nw, sw)
			dst++
			zoom[dst] = l.choose2(nw, ne)
			zoom[dst+zw] = l.modeOrRandom(nw, ne, sw, se)
			dst++
			nw = ne
			sw = se
		}
	}

	out := genLayerGetIntCache(width * height)
	for dz := 0; dz < height; dz++ {
		srcOff := (dz+(z&1))*zw + (x & 1)
		dstOff := dz * width
		copy(out[dstOff:dstOff+width], zoom[srcOff:srcOff+width])
	}
	return out
}

func (l *genLayerZoom) choose2(a, b int) int {
	if l.nextInt(2) == 0 {
		return a
	}
	return b
}

func (l *genLayerZoom) modeOrRandom(a, b, c, d int) int {
	switch {
	case b == c && c == d:
		return b
	case a == b && a == c:
		return a
	case a == b && a == d:
		return a
	case a == c && a == d:
		return a
	case a == b && c != d:
		return a
	case a == c && b != d:
		return a
	case a == d && b != c:
		return a
	case b == a && c != d:
		return b
	case b == c && a != d:
		return b
	case b == d && a != c:
		return b
	case c == a && b != d:
		return c
	case c == b && a != d:
		return c
	case c == d && a != b:
		return c
	case d == a && b != c:
		return c
	case d == b && a != c:
		return c
	case d == c && a != b:
		return c
	}
	switch l.nextInt(4) {
	case 0:
		return a
	case 1:
		return b
	case 2:
		return c
	default:
		return d
	}
}

type genLayerAddSnow struct {
	baseGenLayer
}

func newGenLayerAddSnow(seed int64, parent biomeGenLayer) *genLayerAddSnow {
	l := &genLayerAddSnow{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerAddSnow) getInts(x, z, width, height int) []int {
	px := x - 1
	pz := z - 1
	pw := width + 2
	ph := height + 2
	parentInts := l.parent.getInts(px, pz, pw, ph)
	out := genLayerGetIntCache(width * height)

	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			center := parentInts[dx+1+(dz+1)*pw]
			l.initChunkSeed(int64(dx+x), int64(dz+z))
			if center == 0 {
				out[dx+dz*width] = 0
			} else {
				v := l.nextInt(5)
				if v == 0 {
					v = BiomeIDIcePlains
				} else {
					v = 1
				}
				out[dx+dz*width] = v
			}
		}
	}
	return out
}

type genLayerAddMushroomIsland struct {
	baseGenLayer
}

func newGenLayerAddMushroomIsland(seed int64, parent biomeGenLayer) *genLayerAddMushroomIsland {
	l := &genLayerAddMushroomIsland{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerAddMushroomIsland) getInts(x, z, width, height int) []int {
	px := x - 1
	pz := z - 1
	pw := width + 2
	ph := height + 2
	parentInts := l.parent.getInts(px, pz, pw, ph)
	out := genLayerGetIntCache(width * height)

	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			nw := parentInts[dx+0+(dz+0)*pw]
			ne := parentInts[dx+2+(dz+0)*pw]
			sw := parentInts[dx+0+(dz+2)*pw]
			se := parentInts[dx+2+(dz+2)*pw]
			center := parentInts[dx+1+(dz+1)*pw]
			l.initChunkSeed(int64(dx+x), int64(dz+z))
			if center == 0 && nw == 0 && ne == 0 && sw == 0 && se == 0 && l.nextInt(100) == 0 {
				out[dx+dz*width] = BiomeIDMushroomIsland
			} else {
				out[dx+dz*width] = center
			}
		}
	}
	return out
}

type genLayerRiverInit struct {
	baseGenLayer
}

func newGenLayerRiverInit(seed int64, parent biomeGenLayer) *genLayerRiverInit {
	l := &genLayerRiverInit{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerRiverInit) getInts(x, z, width, height int) []int {
	parentInts := l.parent.getInts(x, z, width, height)
	out := genLayerGetIntCache(width * height)
	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			l.initChunkSeed(int64(dx+x), int64(dz+z))
			if parentInts[dx+dz*width] > 0 {
				out[dx+dz*width] = l.nextInt(2) + 2
			} else {
				out[dx+dz*width] = 0
			}
		}
	}
	return out
}

type genLayerRiver struct {
	baseGenLayer
}

func newGenLayerRiver(seed int64, parent biomeGenLayer) *genLayerRiver {
	l := &genLayerRiver{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerRiver) getInts(x, z, width, height int) []int {
	px := x - 1
	pz := z - 1
	pw := width + 2
	ph := height + 2
	parentInts := l.parent.getInts(px, pz, pw, ph)
	out := genLayerGetIntCache(width * height)
	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			w := parentInts[dx+0+(dz+1)*pw]
			e := parentInts[dx+2+(dz+1)*pw]
			n := parentInts[dx+1+(dz+0)*pw]
			s := parentInts[dx+1+(dz+2)*pw]
			c := parentInts[dx+1+(dz+1)*pw]
			if c != 0 && w != 0 && e != 0 && n != 0 && s != 0 && c == w && c == e && c == n && c == s {
				out[dx+dz*width] = -1
			} else {
				out[dx+dz*width] = BiomeIDRiver
			}
		}
	}
	return out
}

type genLayerSmooth struct {
	baseGenLayer
}

func newGenLayerSmooth(seed int64, parent biomeGenLayer) *genLayerSmooth {
	l := &genLayerSmooth{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerSmooth) getInts(x, z, width, height int) []int {
	px := x - 1
	pz := z - 1
	pw := width + 2
	ph := height + 2
	parentInts := l.parent.getInts(px, pz, pw, ph)
	out := genLayerGetIntCache(width * height)

	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			w := parentInts[dx+0+(dz+1)*pw]
			e := parentInts[dx+2+(dz+1)*pw]
			n := parentInts[dx+1+(dz+0)*pw]
			s := parentInts[dx+1+(dz+2)*pw]
			center := parentInts[dx+1+(dz+1)*pw]

			if w == e && n == s {
				l.initChunkSeed(int64(dx+x), int64(dz+z))
				if l.nextInt(2) == 0 {
					center = w
				} else {
					center = n
				}
			} else {
				if w == e {
					center = w
				}
				if n == s {
					center = n
				}
			}
			out[dx+dz*width] = center
		}
	}
	return out
}

type genLayerBiome struct {
	baseGenLayer
	allowedBiomes []int
}

func newGenLayerBiome(seed int64, parent biomeGenLayer, worldType WorldType) *genLayerBiome {
	l := &genLayerBiome{
		baseGenLayer: newBaseGenLayer(seed),
		allowedBiomes: []int{
			BiomeIDDesert,
			BiomeIDForest,
			BiomeIDExtremeHills,
			BiomeIDSwampland,
			BiomeIDPlains,
			BiomeIDTaiga,
			BiomeIDJungle,
		},
	}
	l.parent = parent
	if worldType == WorldTypeDefault11 {
		l.allowedBiomes = []int{
			BiomeIDDesert,
			BiomeIDForest,
			BiomeIDExtremeHills,
			BiomeIDSwampland,
			BiomeIDPlains,
			BiomeIDTaiga,
		}
	}
	return l
}

func (l *genLayerBiome) getInts(x, z, width, height int) []int {
	parentInts := l.parent.getInts(x, z, width, height)
	out := genLayerGetIntCache(width * height)

	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			l.initChunkSeed(int64(dx+x), int64(dz+z))
			v := parentInts[dx+dz*width]
			if v == 0 {
				out[dx+dz*width] = 0
			} else if v == BiomeIDMushroomIsland {
				out[dx+dz*width] = v
			} else if v == 1 {
				out[dx+dz*width] = l.allowedBiomes[l.nextInt(len(l.allowedBiomes))]
			} else {
				rnd := l.allowedBiomes[l.nextInt(len(l.allowedBiomes))]
				if rnd == BiomeIDTaiga {
					out[dx+dz*width] = rnd
				} else {
					out[dx+dz*width] = BiomeIDIcePlains
				}
			}
		}
	}
	return out
}

type genLayerHills struct {
	baseGenLayer
}

func newGenLayerHills(seed int64, parent biomeGenLayer) *genLayerHills {
	l := &genLayerHills{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerHills) getInts(x, z, width, height int) []int {
	parentInts := l.parent.getInts(x-1, z-1, width+2, height+2)
	out := genLayerGetIntCache(width * height)
	stride := width + 2

	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			l.initChunkSeed(int64(dx+x), int64(dz+z))
			center := parentInts[dx+1+(dz+1)*stride]
			if l.nextInt(3) == 0 {
				mutated := center
				switch center {
				case BiomeIDDesert:
					mutated = BiomeIDDesertHills
				case BiomeIDForest:
					mutated = BiomeIDForestHills
				case BiomeIDTaiga:
					mutated = BiomeIDTaigaHills
				case BiomeIDPlains:
					mutated = BiomeIDForest
				case BiomeIDIcePlains:
					mutated = BiomeIDIceMountains
				case BiomeIDJungle:
					mutated = BiomeIDJungleHills
				}
				if mutated == center {
					out[dx+dz*width] = center
				} else {
					n := parentInts[dx+1+(dz+0)*stride]
					e := parentInts[dx+2+(dz+1)*stride]
					w := parentInts[dx+0+(dz+1)*stride]
					s := parentInts[dx+1+(dz+2)*stride]
					if n == center && e == center && w == center && s == center {
						out[dx+dz*width] = mutated
					} else {
						out[dx+dz*width] = center
					}
				}
			} else {
				out[dx+dz*width] = center
			}
		}
	}
	return out
}

type genLayerShore struct {
	baseGenLayer
}

func newGenLayerShore(seed int64, parent biomeGenLayer) *genLayerShore {
	l := &genLayerShore{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerShore) getInts(x, z, width, height int) []int {
	parentInts := l.parent.getInts(x-1, z-1, width+2, height+2)
	out := genLayerGetIntCache(width * height)
	stride := width + 2
	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			l.initChunkSeed(int64(dx+x), int64(dz+z))
			center := parentInts[dx+1+(dz+1)*stride]
			n := parentInts[dx+1+(dz+0)*stride]
			e := parentInts[dx+2+(dz+1)*stride]
			w := parentInts[dx+0+(dz+1)*stride]
			s := parentInts[dx+1+(dz+2)*stride]

			if center == BiomeIDMushroomIsland {
				if n != BiomeIDOcean && e != BiomeIDOcean && w != BiomeIDOcean && s != BiomeIDOcean {
					out[dx+dz*width] = center
				} else {
					out[dx+dz*width] = BiomeIDMushroomIslandShore
				}
			} else if center != BiomeIDOcean && center != BiomeIDRiver && center != BiomeIDSwampland && center != BiomeIDExtremeHills {
				if n != BiomeIDOcean && e != BiomeIDOcean && w != BiomeIDOcean && s != BiomeIDOcean {
					out[dx+dz*width] = center
				} else {
					out[dx+dz*width] = BiomeIDBeach
				}
			} else if center == BiomeIDExtremeHills {
				if n == BiomeIDExtremeHills && e == BiomeIDExtremeHills && w == BiomeIDExtremeHills && s == BiomeIDExtremeHills {
					out[dx+dz*width] = center
				} else {
					out[dx+dz*width] = BiomeIDExtremeHillsEdge
				}
			} else {
				out[dx+dz*width] = center
			}
		}
	}
	return out
}

type genLayerSwampRivers struct {
	baseGenLayer
}

func newGenLayerSwampRivers(seed int64, parent biomeGenLayer) *genLayerSwampRivers {
	l := &genLayerSwampRivers{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerSwampRivers) getInts(x, z, width, height int) []int {
	parentInts := l.parent.getInts(x-1, z-1, width+2, height+2)
	out := genLayerGetIntCache(width * height)
	stride := width + 2
	for dz := 0; dz < height; dz++ {
		for dx := 0; dx < width; dx++ {
			l.initChunkSeed(int64(dx+x), int64(dz+z))
			center := parentInts[dx+1+(dz+1)*stride]
			if (center != BiomeIDSwampland || l.nextInt(6) != 0) && ((center != BiomeIDJungle && center != BiomeIDJungleHills) || l.nextInt(8) != 0) {
				out[dx+dz*width] = center
			} else {
				out[dx+dz*width] = BiomeIDRiver
			}
		}
	}
	return out
}

type genLayerRiverMix struct {
	baseGenLayer
	biomePatternGeneratorChain biomeGenLayer
	riverPatternGeneratorChain biomeGenLayer
}

func newGenLayerRiverMix(seed int64, biomeChain biomeGenLayer, riverChain biomeGenLayer) *genLayerRiverMix {
	return &genLayerRiverMix{
		baseGenLayer:               newBaseGenLayer(seed),
		biomePatternGeneratorChain: biomeChain,
		riverPatternGeneratorChain: riverChain,
	}
}

func (l *genLayerRiverMix) initWorldGenSeed(seed int64) {
	l.biomePatternGeneratorChain.initWorldGenSeed(seed)
	l.riverPatternGeneratorChain.initWorldGenSeed(seed)
	l.baseGenLayer.initWorldGenSeed(seed)
}

func (l *genLayerRiverMix) getInts(x, z, width, height int) []int {
	biomes := l.biomePatternGeneratorChain.getInts(x, z, width, height)
	rivers := l.riverPatternGeneratorChain.getInts(x, z, width, height)
	out := genLayerGetIntCache(width * height)
	for i := 0; i < width*height; i++ {
		if biomes[i] == BiomeIDOcean {
			out[i] = biomes[i]
		} else if rivers[i] >= 0 {
			if biomes[i] == BiomeIDIcePlains {
				out[i] = BiomeIDFrozenRiver
			} else if biomes[i] != BiomeIDMushroomIsland && biomes[i] != BiomeIDMushroomIslandShore {
				out[i] = rivers[i]
			} else {
				out[i] = BiomeIDMushroomIslandShore
			}
		} else {
			out[i] = biomes[i]
		}
	}
	return out
}

type genLayerVoronoiZoom struct {
	baseGenLayer
}

func newGenLayerVoronoiZoom(seed int64, parent biomeGenLayer) *genLayerVoronoiZoom {
	l := &genLayerVoronoiZoom{baseGenLayer: newBaseGenLayer(seed)}
	l.parent = parent
	return l
}

func (l *genLayerVoronoiZoom) getInts(x, z, width, height int) []int {
	x -= 2
	z -= 2
	const shift = 2
	cellSize := 1 << shift
	px := x >> shift
	pz := z >> shift
	pw := (width >> shift) + 3
	ph := (height >> shift) + 3
	parentInts := l.parent.getInts(px, pz, pw, ph)
	zw := pw << shift
	zh := ph << shift
	zoom := genLayerGetIntCache(zw * zh)

	for iz := 0; iz < ph-1; iz++ {
		nw := parentInts[(iz+0)*pw]
		sw := parentInts[(iz+1)*pw]
		for ix := 0; ix < pw-1; ix++ {
			jitterScale := float64(cellSize) * 0.9

			l.initChunkSeed(int64((ix+px)<<shift), int64((iz+pz)<<shift))
			x0 := (float64(l.nextInt(1024))/1024.0 - 0.5) * jitterScale
			z0 := (float64(l.nextInt(1024))/1024.0 - 0.5) * jitterScale

			l.initChunkSeed(int64((ix+px+1)<<shift), int64((iz+pz)<<shift))
			x1 := (float64(l.nextInt(1024))/1024.0-0.5)*jitterScale + float64(cellSize)
			z1 := (float64(l.nextInt(1024))/1024.0 - 0.5) * jitterScale

			l.initChunkSeed(int64((ix+px)<<shift), int64((iz+pz+1)<<shift))
			x2 := (float64(l.nextInt(1024))/1024.0 - 0.5) * jitterScale
			z2 := (float64(l.nextInt(1024))/1024.0-0.5)*jitterScale + float64(cellSize)

			l.initChunkSeed(int64((ix+px+1)<<shift), int64((iz+pz+1)<<shift))
			x3 := (float64(l.nextInt(1024))/1024.0-0.5)*jitterScale + float64(cellSize)
			z3 := (float64(l.nextInt(1024))/1024.0-0.5)*jitterScale + float64(cellSize)

			ne := parentInts[ix+1+(iz+0)*pw]
			se := parentInts[ix+1+(iz+1)*pw]

			for subZ := 0; subZ < cellSize; subZ++ {
				dst := ((iz << shift) + subZ) * zw
				dst += ix << shift
				for subX := 0; subX < cellSize; subX++ {
					d0 := sq(float64(subZ)-z0) + sq(float64(subX)-x0)
					d1 := sq(float64(subZ)-z1) + sq(float64(subX)-x1)
					d2 := sq(float64(subZ)-z2) + sq(float64(subX)-x2)
					d3 := sq(float64(subZ)-z3) + sq(float64(subX)-x3)

					if d0 < d1 && d0 < d2 && d0 < d3 {
						zoom[dst] = nw
					} else if d1 < d0 && d1 < d2 && d1 < d3 {
						zoom[dst] = ne
					} else if d2 < d0 && d2 < d1 && d2 < d3 {
						zoom[dst] = sw
					} else {
						zoom[dst] = se
					}
					dst++
				}
			}

			nw = ne
			sw = se
		}
	}

	out := genLayerGetIntCache(width * height)
	mask := cellSize - 1
	for dz := 0; dz < height; dz++ {
		srcOff := (dz+(z&mask))*(pw<<shift) + (x & mask)
		dstOff := dz * width
		copy(out[dstOff:dstOff+width], zoom[srcOff:srcOff+width])
	}
	return out
}

func sq(v float64) float64 {
	return v * v
}

func initializeAllBiomeGenerators(seed int64, worldType WorldType) (biomeGenLayer, biomeGenLayer) {
	island := newGenLayerIsland(1)
	fuzzy := newGenLayerFuzzyZoom(2000, island)
	addIsland1 := newGenLayerAddIsland(1, fuzzy)
	zoom1 := newGenLayerZoom(2001, addIsland1)
	addIsland2 := newGenLayerAddIsland(2, zoom1)
	addSnow := newGenLayerAddSnow(2, addIsland2)
	zoom2 := newGenLayerZoom(2002, addSnow)
	addIsland3 := newGenLayerAddIsland(3, zoom2)
	zoom3 := newGenLayerZoom(2003, addIsland3)
	addIsland4 := newGenLayerAddIsland(4, zoom3)
	mushroom := newGenLayerAddMushroomIsland(5, addIsland4)

	biomeSize := 4
	if worldType == WorldTypeLargeBiomes {
		biomeSize = 6
	}

	riverBase := genLayerMagnify(1000, mushroom, 0)
	riverInit := newGenLayerRiverInit(100, riverBase)
	riverZoom := genLayerMagnify(1000, riverInit, biomeSize+2)
	river := newGenLayerRiver(1, riverZoom)
	riverSmooth := newGenLayerSmooth(1000, river)

	biomeBase := genLayerMagnify(1000, mushroom, 0)
	biomeLayer := newGenLayerBiome(200, biomeBase, worldType)
	biomeZoom := genLayerMagnify(1000, biomeLayer, 2)
	var hills biomeGenLayer = newGenLayerHills(1000, biomeZoom)

	for i := 0; i < biomeSize; i++ {
		hills = newGenLayerZoom(int64(1000+i), hills)
		if i == 0 {
			hills = newGenLayerAddIsland(3, hills)
		}
		if i == 1 {
			hills = newGenLayerShore(1000, hills)
			hills = newGenLayerSwampRivers(1000, hills)
		}
	}

	biomeSmooth := newGenLayerSmooth(1000, hills)
	mix := newGenLayerRiverMix(100, biomeSmooth, riverSmooth)
	voronoi := newGenLayerVoronoiZoom(10, mix)

	mix.initWorldGenSeed(seed)
	voronoi.initWorldGenSeed(seed)
	return mix, voronoi
}
