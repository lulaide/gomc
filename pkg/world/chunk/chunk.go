package chunk

import (
	"math"

	"github.com/lulaide/gomc/pkg/world/block"
)

const (
	chunkWidth               = 16
	chunkSections            = 16
	precipitationHeightUnset = -999
)

// Chunk translates core data + lighting behavior from net.minecraft.src.Chunk.
type Chunk struct {
	storageArrays   []*ExtendedBlockStorage
	blockBiomeArray []byte

	PrecipitationHeightMap []int
	updateSkylightColumns  []bool
	HeightMap              []int

	XPosition int32
	ZPosition int32

	IsChunkLoaded        bool
	IsTerrainPopulated   bool
	IsModified           bool
	HasEntities          bool
	SendUpdates          bool
	HeightMapMinimum     int
	InhabitedTime        int64
	isGapLightingUpdated bool

	queuedLightChecks int
	world             WorldBridge
}

func NewChunk(world WorldBridge, xPos, zPos int32) *Chunk {
	if world == nil {
		world = nopWorldBridge{}
	}

	ch := &Chunk{
		storageArrays:          make([]*ExtendedBlockStorage, chunkSections),
		blockBiomeArray:        make([]byte, chunkWidth*chunkWidth),
		PrecipitationHeightMap: make([]int, chunkWidth*chunkWidth),
		updateSkylightColumns:  make([]bool, chunkWidth*chunkWidth),
		HeightMap:              make([]int, chunkWidth*chunkWidth),
		XPosition:              xPos,
		ZPosition:              zPos,
		queuedLightChecks:      4096,
		world:                  world,
	}

	for i := range ch.PrecipitationHeightMap {
		ch.PrecipitationHeightMap[i] = precipitationHeightUnset
	}
	for i := range ch.blockBiomeArray {
		ch.blockBiomeArray[i] = 0xFF
	}

	return ch
}

// NewChunkFromColumn copies persisted chunk data into runtime chunk state.
func NewChunkFromColumn(world WorldBridge, col *Column) *Chunk {
	if col == nil {
		return NewChunk(world, 0, 0)
	}

	ch := NewChunk(world, col.XPos, col.ZPos)
	ch.IsTerrainPopulated = col.IsTerrainPopulated
	ch.InhabitedTime = col.InhabitedTime
	ch.HasEntities = col.HasEntities

	if len(col.HeightMap) == len(ch.HeightMap) {
		for i := range ch.HeightMap {
			ch.HeightMap[i] = int(col.HeightMap[i])
		}
	}

	arr := col.GetStorageArrays()
	if arr != nil {
		ch.storageArrays = make([]*ExtendedBlockStorage, chunkSections)
		copy(ch.storageArrays, arr)
	}

	biomes := col.GetBiomeArray()
	if len(biomes) == len(ch.blockBiomeArray) {
		copy(ch.blockBiomeArray, biomes)
	}

	return ch
}

// ToColumn copies runtime chunk data into storage-facing column state.
func (c *Chunk) ToColumn() *Column {
	col := NewColumn(c.XPosition, c.ZPosition)
	col.IsTerrainPopulated = c.IsTerrainPopulated
	col.InhabitedTime = c.InhabitedTime
	col.HasEntities = c.HasEntities

	for i := range c.HeightMap {
		col.HeightMap[i] = int32(c.HeightMap[i])
	}

	arr := make([]*ExtendedBlockStorage, chunkSections)
	copy(arr, c.storageArrays)
	col.SetStorageArrays(arr)

	biomes := make([]byte, len(c.blockBiomeArray))
	copy(biomes, c.blockBiomeArray)
	col.SetBiomeArray(biomes)
	return col
}

func (c *Chunk) IsAtLocation(x, z int32) bool {
	return x == c.XPosition && z == c.ZPosition
}

func (c *Chunk) GetHeightValue(x, z int) int {
	return c.HeightMap[z<<4|x]
}

func (c *Chunk) GetTopFilledSegment() int {
	for i := len(c.storageArrays) - 1; i >= 0; i-- {
		if c.storageArrays[i] != nil {
			return c.storageArrays[i].GetYLocation()
		}
	}
	return 0
}

func (c *Chunk) GetBlockStorageArray() []*ExtendedBlockStorage {
	return c.storageArrays
}

func (c *Chunk) SetBlockStorageArray(arr []*ExtendedBlockStorage) {
	if arr == nil {
		c.storageArrays = make([]*ExtendedBlockStorage, chunkSections)
		return
	}
	if len(arr) == chunkSections {
		c.storageArrays = arr
		return
	}
	c.storageArrays = make([]*ExtendedBlockStorage, chunkSections)
	copy(c.storageArrays, arr)
}

func (c *Chunk) GetBiomeArray() []byte {
	return c.blockBiomeArray
}

func (c *Chunk) SetBiomeArray(biomes []byte) {
	if len(biomes) == 0 {
		c.blockBiomeArray = make([]byte, chunkWidth*chunkWidth)
		for i := range c.blockBiomeArray {
			c.blockBiomeArray[i] = 0xFF
		}
		return
	}

	if len(biomes) == chunkWidth*chunkWidth {
		c.blockBiomeArray = biomes
		return
	}

	c.blockBiomeArray = make([]byte, chunkWidth*chunkWidth)
	copy(c.blockBiomeArray, biomes)
}

// GenerateHeightMap translates Chunk#generateHeightMap().
func (c *Chunk) GenerateHeightMap() {
	top := c.GetTopFilledSegment()

	for x := 0; x < chunkWidth; x++ {
		z := 0
		for z < chunkWidth {
			c.PrecipitationHeightMap[x+(z<<4)] = precipitationHeightUnset
			y := top + 16 - 1

			for {
				if y > 0 {
					blockID := c.GetBlockID(x, y-1, z)
					if block.GetLightOpacity(blockID) == 0 {
						y--
						continue
					}

					c.HeightMap[z<<4|x] = y
				}

				z++
				break
			}
		}
	}

	c.IsModified = true
}

// GenerateSkylightMap translates Chunk#generateSkylightMap().
func (c *Chunk) GenerateSkylightMap() {
	top := c.GetTopFilledSegment()
	c.HeightMapMinimum = math.MaxInt32

	for x := 0; x < chunkWidth; x++ {
		z := 0
		for z < chunkWidth {
			c.PrecipitationHeightMap[x+(z<<4)] = precipitationHeightUnset
			y := top + 16 - 1

			for {
				if y > 0 {
					if c.GetBlockLightOpacity(x, y-1, z) == 0 {
						y--
						continue
					}

					c.HeightMap[z<<4|x] = y
					if y < c.HeightMapMinimum {
						c.HeightMapMinimum = y
					}
				}

				if !c.world.HasNoSky() {
					light := 15
					yy := top + 16 - 1

					for {
						light -= c.GetBlockLightOpacity(x, yy, z)

						if light > 0 {
							section := c.storageArrays[yy>>4]
							if section != nil {
								section.SetExtSkylightValue(x, yy&15, z, light)
								c.world.MarkBlockForRenderUpdate(int(c.XPosition<<4)+x, yy, int(c.ZPosition<<4)+z)
							}
						}

						yy--
						if !(yy > 0 && light > 0) {
							break
						}
					}
				}

				z++
				break
			}
		}
	}

	c.IsModified = true

	for x := 0; x < chunkWidth; x++ {
		for z := 0; z < chunkWidth; z++ {
			c.propagateSkylightOcclusion(x, z)
		}
	}
}

func (c *Chunk) propagateSkylightOcclusion(x, z int) {
	c.updateSkylightColumns[x+z*16] = true
	c.isGapLightingUpdated = true
}

func (c *Chunk) updateSkylightDo() {
	if !c.world.DoChunksNearChunkExist(int(c.XPosition)*16+8, 0, int(c.ZPosition)*16+8, 16) {
		return
	}

	for x := 0; x < chunkWidth; x++ {
		for z := 0; z < chunkWidth; z++ {
			idx := x + z*16
			if !c.updateSkylightColumns[idx] {
				continue
			}

			c.updateSkylightColumns[idx] = false
			h := c.GetHeightValue(x, z)
			worldX := int(c.XPosition)*16 + x
			worldZ := int(c.ZPosition)*16 + z

			minH := c.world.GetChunkHeightMapMinimum(worldX-1, worldZ)
			eastH := c.world.GetChunkHeightMapMinimum(worldX+1, worldZ)
			northH := c.world.GetChunkHeightMapMinimum(worldX, worldZ-1)
			southH := c.world.GetChunkHeightMapMinimum(worldX, worldZ+1)

			if eastH < minH {
				minH = eastH
			}
			if northH < minH {
				minH = northH
			}
			if southH < minH {
				minH = southH
			}

			c.checkSkylightNeighborHeight(worldX, worldZ, minH)
			c.checkSkylightNeighborHeight(worldX-1, worldZ, h)
			c.checkSkylightNeighborHeight(worldX+1, worldZ, h)
			c.checkSkylightNeighborHeight(worldX, worldZ-1, h)
			c.checkSkylightNeighborHeight(worldX, worldZ+1, h)
		}
	}

	c.isGapLightingUpdated = false
}

func (c *Chunk) checkSkylightNeighborHeight(worldX, worldZ, height int) {
	h := c.world.GetHeightValue(worldX, worldZ)
	if h > height {
		c.updateSkylightNeighborHeight(worldX, worldZ, height, h+1)
	} else if h < height {
		c.updateSkylightNeighborHeight(worldX, worldZ, h, height+1)
	}
}

func (c *Chunk) updateSkylightNeighborHeight(worldX, worldZ, yFrom, yTo int) {
	if yTo <= yFrom || !c.world.DoChunksNearChunkExist(worldX, 0, worldZ, 16) {
		return
	}

	for y := yFrom; y < yTo; y++ {
		c.world.UpdateLightByType(EnumSkyBlockSky, worldX, y, worldZ)
	}

	c.IsModified = true
}

func (c *Chunk) relightBlock(x, y, z int) {
	oldHeight := c.HeightMap[z<<4|x] & 255
	newHeight := oldHeight
	if y > oldHeight {
		newHeight = y
	}

	for newHeight > 0 && c.GetBlockLightOpacity(x, newHeight-1, z) == 0 {
		newHeight--
	}

	if newHeight == oldHeight {
		return
	}

	c.world.MarkBlocksDirtyVertical(x+int(c.XPosition)*16, z+int(c.ZPosition)*16, newHeight, oldHeight)
	c.HeightMap[z<<4|x] = newHeight

	worldX := int(c.XPosition)*16 + x
	worldZ := int(c.ZPosition)*16 + z

	if !c.world.HasNoSky() {
		if newHeight < oldHeight {
			for yy := newHeight; yy < oldHeight; yy++ {
				section := c.storageArrays[yy>>4]
				if section != nil {
					section.SetExtSkylightValue(x, yy&15, z, 15)
					c.world.MarkBlockForRenderUpdate(int(c.XPosition<<4)+x, yy, int(c.ZPosition<<4)+z)
				}
			}
		} else {
			for yy := oldHeight; yy < newHeight; yy++ {
				section := c.storageArrays[yy>>4]
				if section != nil {
					section.SetExtSkylightValue(x, yy&15, z, 0)
					c.world.MarkBlockForRenderUpdate(int(c.XPosition<<4)+x, yy, int(c.ZPosition<<4)+z)
				}
			}
		}

		light := 15
		for newHeight > 0 && light > 0 {
			newHeight--
			opacity := c.GetBlockLightOpacity(x, newHeight, z)
			if opacity == 0 {
				opacity = 1
			}

			light -= opacity
			if light < 0 {
				light = 0
			}

			section := c.storageArrays[newHeight>>4]
			if section != nil {
				section.SetExtSkylightValue(x, newHeight&15, z, light)
			}
		}
	}

	curHeight := c.HeightMap[z<<4|x]
	minY := oldHeight
	maxY := curHeight
	if curHeight < oldHeight {
		minY = curHeight
		maxY = oldHeight
	}

	if curHeight < c.HeightMapMinimum {
		c.HeightMapMinimum = curHeight
	}

	if !c.world.HasNoSky() {
		c.updateSkylightNeighborHeight(worldX-1, worldZ, minY, maxY)
		c.updateSkylightNeighborHeight(worldX+1, worldZ, minY, maxY)
		c.updateSkylightNeighborHeight(worldX, worldZ-1, minY, maxY)
		c.updateSkylightNeighborHeight(worldX, worldZ+1, minY, maxY)
		c.updateSkylightNeighborHeight(worldX, worldZ, minY, maxY)
	}

	c.IsModified = true
}

func (c *Chunk) GetBlockLightOpacity(x, y, z int) int {
	return block.GetLightOpacity(c.GetBlockID(x, y, z))
}

// GetBlockID translates Chunk#getBlockID(int,int,int).
func (c *Chunk) GetBlockID(x, y, z int) int {
	if y>>4 >= len(c.storageArrays) {
		return 0
	}

	section := c.storageArrays[y>>4]
	if section == nil {
		return 0
	}
	return section.GetExtBlockID(x, y&15, z)
}

// GetBlockMetadata translates Chunk#getBlockMetadata(int,int,int).
func (c *Chunk) GetBlockMetadata(x, y, z int) int {
	if y>>4 >= len(c.storageArrays) {
		return 0
	}

	section := c.storageArrays[y>>4]
	if section == nil {
		return 0
	}
	return section.GetExtBlockMetadata(x, y&15, z)
}

// SetBlockIDWithMetadata translates Chunk#setBlockIDWithMetadata(...).
func (c *Chunk) SetBlockIDWithMetadata(x, y, z, newBlockID, metadata int) bool {
	idx := z<<4 | x
	if y >= c.PrecipitationHeightMap[idx]-1 {
		c.PrecipitationHeightMap[idx] = precipitationHeightUnset
	}

	oldHeight := c.HeightMap[idx]
	oldBlockID := c.GetBlockID(x, y, z)
	oldMetadata := c.GetBlockMetadata(x, y, z)
	if oldBlockID == newBlockID && oldMetadata == metadata {
		return false
	}

	section := c.storageArrays[y>>4]
	createdSection := false
	if section == nil {
		if newBlockID == 0 {
			return false
		}

		section = NewExtendedBlockStorage(y>>4<<4, !c.world.HasNoSky())
		c.storageArrays[y>>4] = section
		createdSection = y >= oldHeight
	}

	worldX := int(c.XPosition)*16 + x
	worldZ := int(c.ZPosition)*16 + z

	if oldBlockID != 0 && !c.world.IsRemote() {
		c.world.OnBlockPreDestroy(oldBlockID, worldX, y, worldZ, oldMetadata)
	}

	section.SetExtBlockID(x, y&15, z, newBlockID)

	if oldBlockID != 0 {
		if !c.world.IsRemote() {
			c.world.BreakBlock(oldBlockID, worldX, y, worldZ, oldBlockID, oldMetadata)
		} else if block.IsTileEntityProvider(oldBlockID) && oldBlockID != newBlockID {
			c.world.RemoveBlockTileEntity(worldX, y, worldZ)
		}
	}

	if section.GetExtBlockID(x, y&15, z) != newBlockID {
		return false
	}

	section.SetExtBlockMetadata(x, y&15, z, metadata)

	if createdSection {
		c.GenerateSkylightMap()
	} else {
		if block.GetLightOpacity(newBlockID&4095) > 0 {
			if y >= oldHeight {
				c.relightBlock(x, y+1, z)
			}
		} else if y == oldHeight-1 {
			c.relightBlock(x, y, z)
		}

		c.propagateSkylightOcclusion(x, z)
	}

	// TileEntity lifecycle wiring keeps call ordering from Chunk#setBlockIDWithMetadata.
	if newBlockID != 0 {
		if !c.world.IsRemote() {
			c.world.OnBlockAdded(newBlockID, worldX, y, worldZ)
		}
		if block.IsTileEntityProvider(newBlockID) {
			c.world.EnsureBlockTileEntity(newBlockID, worldX, y, worldZ)
			c.world.UpdateBlockTileEntityInfo(worldX, y, worldZ)
		}
	} else if oldBlockID > 0 && block.IsTileEntityProvider(oldBlockID) {
		c.world.UpdateBlockTileEntityInfo(worldX, y, worldZ)
	}

	c.IsModified = true
	return true
}

// SetBlockMetadata translates Chunk#setBlockMetadata(...).
func (c *Chunk) SetBlockMetadata(x, y, z, metadata int) bool {
	section := c.storageArrays[y>>4]
	if section == nil {
		return false
	}

	oldMeta := section.GetExtBlockMetadata(x, y&15, z)
	if oldMeta == metadata {
		return false
	}

	c.IsModified = true
	section.SetExtBlockMetadata(x, y&15, z, metadata)

	blockID := section.GetExtBlockID(x, y&15, z)
	if blockID > 0 && block.IsTileEntityProvider(blockID) {
		worldX := int(c.XPosition)*16 + x
		worldZ := int(c.ZPosition)*16 + z
		c.world.UpdateBlockTileEntityInfo(worldX, y, worldZ)
	}
	return true
}

// GetSavedLightValue translates Chunk#getSavedLightValue(...).
func (c *Chunk) GetSavedLightValue(kind EnumSkyBlock, x, y, z int) int {
	section := c.storageArrays[y>>4]
	if section == nil {
		if c.CanBlockSeeTheSky(x, y, z) {
			return kind.DefaultLightValue()
		}
		return 0
	}

	if kind == EnumSkyBlockSky {
		if c.world.HasNoSky() {
			return 0
		}
		return section.GetExtSkylightValue(x, y&15, z)
	}

	if kind == EnumSkyBlockBlock {
		return section.GetExtBlocklightValue(x, y&15, z)
	}

	return kind.DefaultLightValue()
}

// SetLightValue translates Chunk#setLightValue(...).
func (c *Chunk) SetLightValue(kind EnumSkyBlock, x, y, z, value int) {
	section := c.storageArrays[y>>4]
	if section == nil {
		section = NewExtendedBlockStorage(y>>4<<4, !c.world.HasNoSky())
		c.storageArrays[y>>4] = section
		c.GenerateSkylightMap()
	}

	c.IsModified = true

	if kind == EnumSkyBlockSky {
		if !c.world.HasNoSky() {
			section.SetExtSkylightValue(x, y&15, z, value)
		}
	} else if kind == EnumSkyBlockBlock {
		section.SetExtBlocklightValue(x, y&15, z, value)
	}
}

func (c *Chunk) CanBlockSeeTheSky(x, y, z int) bool {
	return y >= c.HeightMap[z<<4|x]
}

// GetPrecipitationHeight translates Chunk#getPrecipitationHeight(int,int).
func (c *Chunk) GetPrecipitationHeight(x, z int) int {
	idx := x | z<<4
	precipitationHeight := c.PrecipitationHeightMap[idx]

	if precipitationHeight == precipitationHeightUnset {
		y := c.GetTopFilledSegment() + 15
		precipitationHeight = -1

		for y > 0 && precipitationHeight == -1 {
			blockID := c.GetBlockID(x, y, z)
			moves := false
			liquid := false
			if blockID != 0 {
				moves = block.BlocksMovement(blockID)
				liquid = block.IsLiquid(blockID)
			}

			if !moves && !liquid {
				y--
			} else {
				precipitationHeight = y + 1
			}
		}

		c.PrecipitationHeightMap[idx] = precipitationHeight
	}

	return precipitationHeight
}

func (c *Chunk) UpdateSkylight() {
	if c.isGapLightingUpdated && !c.world.HasNoSky() {
		c.updateSkylightDo()
	}
}
