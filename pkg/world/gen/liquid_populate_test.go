package gen

import "testing"

func TestGenerateLiquidSpringAtWorldPlacesWhenPatternMatches(t *testing.T) {
	blocks := make([]byte, 32768)
	for i := range blocks {
		blocks[i] = blockIDStone
	}

	targetChunkX, targetChunkZ := 0, 0
	x, y, z := 8, 50, 8

	// Center can be air or stone per WorldGenLiquids.
	if !setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z, blockIDAir) {
		t.Fatal("failed to set center block")
	}
	// Exactly one air opening on side.
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x+1, y, z, blockIDAir)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x-1, y, z, blockIDStone)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z-1, blockIDStone)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z+1, blockIDStone)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y-1, z, blockIDStone)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y+1, z, blockIDStone)

	if !generateLiquidSpringAtWorld(blocks, targetChunkX, targetChunkZ, x, y, z, blockIDWaterFlow) {
		t.Fatal("expected spring placement to succeed")
	}

	id, ok := blockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z)
	if !ok {
		t.Fatal("failed to read center block")
	}
	if id != blockIDWaterFlow {
		t.Fatalf("liquid placement mismatch: got=%d want=%d", id, blockIDWaterFlow)
	}
}

func TestGenerateLiquidSpringAtWorldRejectsWhenTopNotStone(t *testing.T) {
	blocks := make([]byte, 32768)
	for i := range blocks {
		blocks[i] = blockIDStone
	}

	targetChunkX, targetChunkZ := 0, 0
	x, y, z := 8, 50, 8
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z, blockIDAir)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y+1, z, blockIDAir)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y-1, z, blockIDStone)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x-1, y, z, blockIDStone)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x+1, y, z, blockIDStone)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z-1, blockIDStone)
	_ = setBlockAtWorldInTargetChunk(blocks, targetChunkX, targetChunkZ, x, y, z+1, blockIDAir)

	if generateLiquidSpringAtWorld(blocks, targetChunkX, targetChunkZ, x, y, z, blockIDLavaFlow) {
		t.Fatal("expected spring generation to fail when top block is not stone")
	}
}
