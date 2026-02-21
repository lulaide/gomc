package chunk

// Column is a minimal chunk-level data carrier aligned with fields used by
// AnvilChunkLoader read/write paths.
//
// Translation target:
// - net.minecraft.src.Chunk fields used in AnvilChunkLoader
type Column struct {
	XPos int32
	ZPos int32

	HeightMap          []int32
	IsTerrainPopulated bool
	InhabitedTime      int64
	HasEntities        bool

	storageArrays []*ExtendedBlockStorage
	biomeArray    []byte
}

func NewColumn(xPos, zPos int32) *Column {
	biomes := make([]byte, 256)
	for i := range biomes {
		biomes[i] = 0xFF
	}

	return &Column{
		XPos:          xPos,
		ZPos:          zPos,
		HeightMap:     make([]int32, 256),
		storageArrays: make([]*ExtendedBlockStorage, 16),
		biomeArray:    biomes,
	}
}

func (c *Column) GetStorageArrays() []*ExtendedBlockStorage {
	return c.storageArrays
}

func (c *Column) SetStorageArrays(arr []*ExtendedBlockStorage) {
	if arr == nil {
		c.storageArrays = make([]*ExtendedBlockStorage, 16)
		return
	}
	if len(arr) == 16 {
		c.storageArrays = arr
		return
	}

	c.storageArrays = make([]*ExtendedBlockStorage, 16)
	copy(c.storageArrays, arr)
}

func (c *Column) GetBiomeArray() []byte {
	return c.biomeArray
}

func (c *Column) SetBiomeArray(v []byte) {
	if len(v) == 0 {
		c.biomeArray = make([]byte, 256)
		for i := range c.biomeArray {
			c.biomeArray[i] = 0xFF
		}
		return
	}
	c.biomeArray = v
}
