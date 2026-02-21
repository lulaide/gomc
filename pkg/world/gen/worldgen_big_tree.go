package gen

import (
	"math"

	"github.com/lulaide/gomc/pkg/util"
)

var bigTreeOtherCoordPairs = [...]int{2, 0, 0, 1, 2, 1}

// bigTreeGenerator translates net.minecraft.src.WorldGenBigTree.
type bigTreeGenerator struct {
	rand         *util.JavaRandom
	blocks       []byte
	targetChunkX int
	targetChunkZ int

	basePos [3]int

	heightLimit int
	height      int

	heightAttenuation float64
	branchDensity     float64
	branchSlope       float64
	scaleWidth        float64
	leafDensity       float64

	trunkSize         int
	heightLimitLimit  int
	leafDistanceLimit int

	leafNodes [][4]int
}

func newBigTreeGenerator(
	parentRand *util.JavaRandom,
	blocks []byte,
	targetChunkX, targetChunkZ int,
) *bigTreeGenerator {
	g := &bigTreeGenerator{
		rand:              util.NewJavaRandom(0),
		blocks:            blocks,
		targetChunkX:      targetChunkX,
		targetChunkZ:      targetChunkZ,
		heightAttenuation: 0.618,
		branchDensity:     1.0,
		branchSlope:       0.381,
		scaleWidth:        1.0,
		leafDensity:       1.0,
		trunkSize:         1,
		heightLimitLimit:  12,
		leafDistanceLimit: 4,
	}
	if parentRand != nil {
		g.rand.SetSeed(parentRand.NextLong())
	}
	return g
}

func (g *bigTreeGenerator) getBlock(x, y, z int) byte {
	return blockAtWorldForGen(g.blocks, g.targetChunkX, g.targetChunkZ, x, y, z, blockIDStone)
}

func (g *bigTreeGenerator) setBlock(x, y, z int, id byte) {
	_ = setBlockAtWorldInTargetChunk(g.blocks, g.targetChunkX, g.targetChunkZ, x, y, z, id)
}

func (g *bigTreeGenerator) setBlockWithMetadata(x, y, z int, id, metadata byte) {
	_ = setBlockAtWorldInTargetChunk(g.blocks, g.targetChunkX, g.targetChunkZ, x, y, z, id, metadata)
}

func (g *bigTreeGenerator) layerSize(layerY int) float64 {
	if float64(layerY) < float64(g.heightLimit)*0.3 {
		return -1.618
	}

	half := float64(g.heightLimit) / 2.0
	rel := half - float64(layerY)

	var size float64
	if rel == 0.0 {
		size = half
	} else if math.Abs(rel) >= half {
		size = 0.0
	} else {
		size = math.Sqrt(math.Pow(math.Abs(half), 2.0) - math.Pow(math.Abs(rel), 2.0))
	}
	return size * 0.5
}

func (g *bigTreeGenerator) leafSize(layer int) float64 {
	if layer >= 0 && layer < g.leafDistanceLimit {
		if layer != 0 && layer != g.leafDistanceLimit-1 {
			return 3.0
		}
		return 2.0
	}
	return -1.0
}

func (g *bigTreeGenerator) genTreeLayer(x, y, z int, radius float64, axis int, blockID byte) {
	r := int(radius + 0.618)
	axisA := bigTreeOtherCoordPairs[axis]
	axisB := bigTreeOtherCoordPairs[axis+3]

	base := [3]int{x, y, z}
	point := [3]int{}
	point[axis] = base[axis]

	for da := -r; da <= r; da++ {
		point[axisA] = base[axisA] + da
		for db := -r; db <= r; db++ {
			dist := math.Pow(math.Abs(float64(da))+0.5, 2.0) + math.Pow(math.Abs(float64(db))+0.5, 2.0)
			if dist > radius*radius {
				continue
			}
			point[axisB] = base[axisB] + db
			id := g.getBlock(point[0], point[1], point[2])
			if id != blockIDAir && id != blockIDLeaves {
				continue
			}
			g.setBlock(point[0], point[1], point[2], blockID)
		}
	}
}

func (g *bigTreeGenerator) generateLeafNode(x, y, z int) {
	for yy := y; yy < y+g.leafDistanceLimit; yy++ {
		size := g.leafSize(yy - y)
		g.genTreeLayer(x, yy, z, size, 1, blockIDLeaves)
	}
}

func (g *bigTreeGenerator) placeBlockLine(from, to [3]int, blockID byte) {
	delta := [3]int{
		to[0] - from[0],
		to[1] - from[1],
		to[2] - from[2],
	}

	major := 0
	if absInt(delta[1]) > absInt(delta[major]) {
		major = 1
	}
	if absInt(delta[2]) > absInt(delta[major]) {
		major = 2
	}
	if delta[major] == 0 {
		return
	}

	axisA := bigTreeOtherCoordPairs[major]
	axisB := bigTreeOtherCoordPairs[major+3]
	step := 1
	if delta[major] < 0 {
		step = -1
	}
	ratioA := float64(delta[axisA]) / float64(delta[major])
	ratioB := float64(delta[axisB]) / float64(delta[major])

	point := [3]int{}
	for i, end := 0, delta[major]+step; i != end; i += step {
		point[major] = floorDouble(float64(from[major]+i) + 0.5)
		point[axisA] = floorDouble(float64(from[axisA]) + float64(i)*ratioA + 0.5)
		point[axisB] = floorDouble(float64(from[axisB]) + float64(i)*ratioB + 0.5)
		meta := byte(0)
		if blockID == blockIDLog {
			absMajor := absInt(delta[major])
			if absInt(delta[0]) == absMajor {
				meta = 4
			} else if absInt(delta[2]) == absMajor {
				meta = 8
			}
		}
		g.setBlockWithMetadata(point[0], point[1], point[2], blockID, meta)
	}
}

func (g *bigTreeGenerator) checkBlockLine(from, to [3]int) int {
	delta := [3]int{
		to[0] - from[0],
		to[1] - from[1],
		to[2] - from[2],
	}

	major := 0
	if absInt(delta[1]) > absInt(delta[major]) {
		major = 1
	}
	if absInt(delta[2]) > absInt(delta[major]) {
		major = 2
	}
	if delta[major] == 0 {
		return -1
	}

	axisA := bigTreeOtherCoordPairs[major]
	axisB := bigTreeOtherCoordPairs[major+3]
	step := 1
	if delta[major] < 0 {
		step = -1
	}
	ratioA := float64(delta[axisA]) / float64(delta[major])
	ratioB := float64(delta[axisB]) / float64(delta[major])

	point := [3]int{}
	i := 0
	end := delta[major] + step
	for ; i != end; i += step {
		point[major] = from[major] + i
		point[axisA] = floorDouble(float64(from[axisA]) + float64(i)*ratioA)
		point[axisB] = floorDouble(float64(from[axisB]) + float64(i)*ratioB)
		id := g.getBlock(point[0], point[1], point[2])
		if id != blockIDAir && id != blockIDLeaves {
			break
		}
	}

	if i == end {
		return -1
	}
	return absInt(i)
}

func (g *bigTreeGenerator) leafNodeNeedsBase(dy int) bool {
	return float64(dy) >= float64(g.heightLimit)*0.2
}

func (g *bigTreeGenerator) generateLeafNodeList() {
	g.height = int(float64(g.heightLimit) * g.heightAttenuation)
	if g.height >= g.heightLimit {
		g.height = g.heightLimit - 1
	}

	maxNodes := int(1.382 + math.Pow(g.leafDensity*float64(g.heightLimit)/13.0, 2.0))
	if maxNodes < 1 {
		maxNodes = 1
	}

	nodes := make([][4]int, maxNodes*g.heightLimit)
	y := g.basePos[1] + g.heightLimit - g.leafDistanceLimit
	count := 1
	trunkTop := g.basePos[1] + g.height
	rel := y - g.basePos[1]
	nodes[0] = [4]int{g.basePos[0], y, g.basePos[2], trunkTop}
	y--

	for rel >= 0 {
		layer := g.layerSize(rel)
		if layer < 0.0 {
			y--
			rel--
			continue
		}

		for i := 0; i < maxNodes; i++ {
			r := g.scaleWidth * layer * (float64(g.rand.NextFloat()) + 0.328)
			theta := float64(g.rand.NextFloat()) * 2.0 * math.Pi
			nodeX := floorDouble(r*math.Sin(theta) + float64(g.basePos[0]) + 0.5)
			nodeZ := floorDouble(r*math.Cos(theta) + float64(g.basePos[2]) + 0.5)
			nodeBase := [3]int{nodeX, y, nodeZ}
			nodeTop := [3]int{nodeX, y + g.leafDistanceLimit, nodeZ}

			if g.checkBlockLine(nodeBase, nodeTop) != -1 {
				continue
			}

			branchBase := [3]int{g.basePos[0], g.basePos[1], g.basePos[2]}
			dist := math.Sqrt(math.Pow(float64(absInt(g.basePos[0]-nodeX)), 2.0) + math.Pow(float64(absInt(g.basePos[2]-nodeZ)), 2.0))
			slope := dist * g.branchSlope
			if float64(nodeBase[1])-slope > float64(trunkTop) {
				branchBase[1] = trunkTop
			} else {
				branchBase[1] = int(float64(nodeBase[1]) - slope)
			}

			if g.checkBlockLine(branchBase, nodeBase) == -1 {
				nodes[count] = [4]int{nodeX, y, nodeZ, branchBase[1]}
				count++
			}
		}

		y--
		rel--
	}

	g.leafNodes = make([][4]int, count)
	copy(g.leafNodes, nodes[:count])
}

func (g *bigTreeGenerator) generateLeaves() {
	for _, node := range g.leafNodes {
		g.generateLeafNode(node[0], node[1], node[2])
	}
}

func (g *bigTreeGenerator) generateTrunk() {
	from := [3]int{g.basePos[0], g.basePos[1], g.basePos[2]}
	to := [3]int{g.basePos[0], g.basePos[1] + g.height, g.basePos[2]}
	g.placeBlockLine(from, to, blockIDLog)

	if g.trunkSize == 2 {
		from[0]++
		to[0]++
		g.placeBlockLine(from, to, blockIDLog)
		from[2]++
		to[2]++
		g.placeBlockLine(from, to, blockIDLog)
		from[0]--
		to[0]--
		g.placeBlockLine(from, to, blockIDLog)
	}
}

func (g *bigTreeGenerator) generateLeafNodeBases() {
	branchBase := [3]int{g.basePos[0], g.basePos[1], g.basePos[2]}
	for _, node := range g.leafNodes {
		nodePos := [3]int{node[0], node[1], node[2]}
		branchBase[1] = node[3]
		if g.leafNodeNeedsBase(branchBase[1] - g.basePos[1]) {
			g.placeBlockLine(branchBase, nodePos, blockIDLog)
		}
	}
}

func (g *bigTreeGenerator) validTreeLocation() bool {
	base := [3]int{g.basePos[0], g.basePos[1], g.basePos[2]}
	top := [3]int{g.basePos[0], g.basePos[1] + g.heightLimit - 1, g.basePos[2]}
	ground := g.getBlock(g.basePos[0], g.basePos[1]-1, g.basePos[2])
	if ground != blockIDGrass && ground != blockIDDirt {
		return false
	}

	line := g.checkBlockLine(base, top)
	if line == -1 {
		return true
	}
	if line < 6 {
		return false
	}
	g.heightLimit = line
	return true
}

func (g *bigTreeGenerator) generate(worldX, y, worldZ int) bool {
	g.basePos[0] = worldX
	g.basePos[1] = y
	g.basePos[2] = worldZ

	if g.heightLimit == 0 {
		g.heightLimit = 5 + int(g.rand.NextInt(g.heightLimitLimit))
	}
	if !g.validTreeLocation() {
		return false
	}

	g.generateLeafNodeList()
	g.generateLeaves()
	g.generateTrunk()
	g.generateLeafNodeBases()
	return true
}

// Translation reference:
// - net.minecraft.src.WorldGenBigTree.generate(...)
func generateBigTreeAtWorld(
	blocks []byte,
	rng *util.JavaRandom,
	targetChunkX, targetChunkZ int,
	worldX, y, worldZ int,
) bool {
	if rng == nil {
		return false
	}
	g := newBigTreeGenerator(rng, blocks, targetChunkX, targetChunkZ)
	return g.generate(worldX, y, worldZ)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
