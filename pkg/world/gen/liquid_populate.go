package gen

import "github.com/lulaide/gomc/pkg/util"

// Translation reference:
// - net.minecraft.src.BiomeDecorator.decorate() liquid spring loops
// - net.minecraft.src.WorldGenLiquids.generate(...)
func (p *ChunkProviderGenerate) populateLiquids(targetChunkX, targetChunkZ int, blocks []byte) {
	if p == nil || len(blocks) == 0 {
		return
	}

	rng := util.NewJavaRandom(p.worldSeed)
	xMul := (rng.NextLong()/2)*2 + 1
	zMul := (rng.NextLong()/2)*2 + 1

	// Springs generated from adjacent populate chunks can spill into this chunk.
	for popChunkX := targetChunkX - 1; popChunkX <= targetChunkX+1; popChunkX++ {
		for popChunkZ := targetChunkZ - 1; popChunkZ <= targetChunkZ+1; popChunkZ++ {
			seed := (int64(popChunkX)*xMul + int64(popChunkZ)*zMul) ^ p.worldSeed
			rng.SetSeed(seed)
			p.populateLiquidSpringsForPopulateChunk(rng, popChunkX, popChunkZ, targetChunkX, targetChunkZ, blocks)
		}
	}
}

func (p *ChunkProviderGenerate) populateLiquidSpringsForPopulateChunk(
	rng *util.JavaRandom,
	popChunkX, popChunkZ int,
	targetChunkX, targetChunkZ int,
	blocks []byte,
) {
	if rng == nil {
		return
	}

	baseX := popChunkX * 16
	baseZ := popChunkZ * 16
	for i := 0; i < 50; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(int(rng.NextInt(120) + 8)))
		z := baseZ + int(rng.NextInt(16)) + 8
		generateLiquidSpringAtWorld(blocks, targetChunkX, targetChunkZ, x, y, z, blockIDWaterFlow)
	}
	for i := 0; i < 20; i++ {
		x := baseX + int(rng.NextInt(16)) + 8
		y := int(rng.NextInt(int(rng.NextInt(int(rng.NextInt(112)+8)) + 8)))
		z := baseZ + int(rng.NextInt(16)) + 8
		generateLiquidSpringAtWorld(blocks, targetChunkX, targetChunkZ, x, y, z, blockIDLavaFlow)
	}
}

func generateLiquidSpringAtWorld(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int, liquidID byte) bool {
	if y <= 0 || y >= worldHeightLegacyY-1 {
		return false
	}

	up, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y+1, worldZ)
	if !ok || up != blockIDStone {
		return false
	}
	down, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y-1, worldZ)
	if !ok || down != blockIDStone {
		return false
	}
	center, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ)
	if !ok || (center != blockIDAir && center != blockIDStone) {
		return false
	}

	stoneSides := 0
	airSides := 0
	sideOffsets := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, o := range sideOffsets {
		id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX+o[0], y, worldZ+o[1])
		if !ok {
			// Keep generation deterministic and local to this chunk-only buffer.
			return false
		}
		if id == blockIDStone {
			stoneSides++
		}
		if id == blockIDAir {
			airSides++
		}
	}
	if stoneSides == 3 && airSides == 1 {
		return setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, worldX, y, worldZ, liquidID)
	}
	return false
}

func blockAtWorldInTargetChunk(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int) (byte, bool) {
	if y < 0 || y >= worldHeightLegacyY {
		return 0, false
	}
	minX := targetChunkX * 16
	minZ := targetChunkZ * 16
	localX := worldX - minX
	localZ := worldZ - minZ
	if localX < 0 || localX >= 16 || localZ < 0 || localZ >= 16 {
		return 0, false
	}
	idx := chunkBlockIndex(localX, localZ, y)
	if idx < 0 || idx >= len(blocks) {
		return 0, false
	}
	return blocks[idx], true
}

func blockMetadataAtWorldInTargetChunk(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int) (byte, bool) {
	if y < 0 || y >= worldHeightLegacyY {
		return 0, false
	}
	minX := targetChunkX * 16
	minZ := targetChunkZ * 16
	localX := worldX - minX
	localZ := worldZ - minZ
	if localX < 0 || localX >= 16 || localZ < 0 || localZ >= 16 {
		return 0, false
	}
	idx := chunkBlockIndex(localX, localZ, y)
	meta := metadataBufferForBlocks(blocks)
	if idx < 0 || idx >= len(blocks) || len(meta) != len(blocks) {
		return 0, false
	}
	return meta[idx] & 0xF, true
}

func setBlockAtWorldInTargetChunk(blocks []byte, targetChunkX, targetChunkZ int, worldX, y, worldZ int, id byte, metadata ...byte) bool {
	if y < 0 || y >= worldHeightLegacyY {
		return false
	}
	minX := targetChunkX * 16
	minZ := targetChunkZ * 16
	localX := worldX - minX
	localZ := worldZ - minZ
	if localX < 0 || localX >= 16 || localZ < 0 || localZ >= 16 {
		return false
	}
	idx := chunkBlockIndex(localX, localZ, y)
	if idx < 0 || idx >= len(blocks) {
		return false
	}
	blocks[idx] = id
	metaBuf := metadataBufferForBlocks(blocks)
	if len(metaBuf) == len(blocks) {
		meta := byte(0)
		if len(metadata) > 0 {
			meta = metadata[0] & 0xF
		}
		metaBuf[idx] = meta
	}
	return true
}
