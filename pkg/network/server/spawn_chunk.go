package server

import (
	"sync"

	"github.com/lulaide/gomc/pkg/world/block"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

var worldBlockPropsOnce sync.Once

func ensureOverworldBlockProperties() {
	worldBlockPropsOnce.Do(func() {
		// Translation reference:
		// - net.minecraft.src.Block.lightOpacity[]
		// - material movement/liquid checks used by Chunk#getPrecipitationHeight
		block.SetLightOpacity(1, 255)  // stone
		block.SetLightOpacity(2, 255)  // grass
		block.SetLightOpacity(3, 255)  // dirt
		block.SetLightOpacity(4, 255)  // cobblestone
		block.SetLightOpacity(7, 255)  // bedrock
		block.SetLightOpacity(8, 3)    // flowing water
		block.SetLightOpacity(9, 3)    // still water
		block.SetLightOpacity(10, 255) // flowing lava
		block.SetLightOpacity(11, 255) // still lava
		block.SetLightOpacity(12, 255) // sand
		block.SetLightOpacity(13, 255) // gravel
		block.SetLightOpacity(14, 255) // gold ore
		block.SetLightOpacity(15, 255) // iron ore
		block.SetLightOpacity(16, 255) // coal ore
		block.SetLightOpacity(17, 255) // oak log
		block.SetLightOpacity(18, 1)   // oak leaves
		block.SetLightOpacity(21, 255) // lapis ore
		block.SetLightOpacity(24, 255) // sandstone
		block.SetLightOpacity(31, 0)   // tall grass
		block.SetLightOpacity(37, 0)   // dandelion
		block.SetLightOpacity(38, 0)   // rose
		block.SetLightOpacity(48, 255) // mossy cobblestone
		block.SetLightOpacity(52, 255) // mob spawner
		block.SetLightOpacity(54, 255) // chest
		block.SetLightOpacity(56, 255) // diamond ore
		block.SetLightOpacity(73, 255) // redstone ore
		block.SetLightOpacity(79, 3)   // ice
		block.SetLightOpacity(82, 255) // clay block
		block.SetLightOpacity(83, 0)   // reeds
		block.SetLightOpacity(86, 255) // pumpkin

		block.SetMaterialProperties(1, true, false)
		block.SetMaterialProperties(2, true, false)
		block.SetMaterialProperties(3, true, false)
		block.SetMaterialProperties(4, true, false)
		block.SetMaterialProperties(7, true, false)
		block.SetMaterialProperties(8, false, true)
		block.SetMaterialProperties(9, false, true)
		block.SetMaterialProperties(10, false, true)
		block.SetMaterialProperties(11, false, true)
		block.SetMaterialProperties(12, true, false)
		block.SetMaterialProperties(13, true, false)
		block.SetMaterialProperties(14, true, false)
		block.SetMaterialProperties(15, true, false)
		block.SetMaterialProperties(16, true, false)
		block.SetMaterialProperties(17, true, false)
		block.SetMaterialProperties(18, true, false)
		block.SetMaterialProperties(21, true, false)
		block.SetMaterialProperties(24, true, false)
		block.SetMaterialProperties(31, false, false)
		block.SetMaterialProperties(37, false, false)
		block.SetMaterialProperties(38, false, false)
		block.SetMaterialProperties(48, true, false)
		block.SetMaterialProperties(52, true, false)
		block.SetMaterialProperties(54, true, false)
		block.SetMaterialProperties(56, true, false)
		block.SetMaterialProperties(73, true, false)
		block.SetMaterialProperties(79, true, false)
		block.SetMaterialProperties(82, true, false)
		block.SetMaterialProperties(83, false, false)
		block.SetMaterialProperties(86, true, false)
	})
}

func ensureFlatWorldBlockProperties() {
	ensureOverworldBlockProperties()
}

// buildSpawnChunk creates a simple flat spawn chunk for initial login view.
func buildSpawnChunk() *chunk.Chunk {
	return buildFlatChunkAt(0, 0)
}

func buildFlatChunkAt(chunkX, chunkZ int32) *chunk.Chunk {
	ensureFlatWorldBlockProperties()
	ch := chunk.NewChunk(nil, chunkX, chunkZ)

	sec := chunk.NewExtendedBlockStorage(0, true)
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			sec.SetExtBlockID(x, 0, z, 7)
			sec.SetExtBlockID(x, 1, z, 3)
			sec.SetExtBlockID(x, 2, z, 3)
			sec.SetExtBlockID(x, 3, z, 3)
			sec.SetExtBlockID(x, 4, z, 2)
		}
	}

	arr := ch.GetBlockStorageArray()
	arr[0] = sec
	ch.SetBlockStorageArray(arr)

	biomes := ch.GetBiomeArray()
	for i := range biomes {
		biomes[i] = 1 // plains
	}
	ch.SetBiomeArray(biomes)

	ch.GenerateSkylightMap()
	return ch
}
