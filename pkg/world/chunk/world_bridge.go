package chunk

// WorldBridge defines Chunk's required world callbacks.
//
// Translation target:
// - net.minecraft.src.World calls inside Chunk methods
type WorldBridge interface {
	HasNoSky() bool
	IsRemote() bool

	MarkBlockForRenderUpdate(x, y, z int)
	MarkBlocksDirtyVertical(x, z, yFrom, yTo int)

	DoChunksNearChunkExist(x, y, z, radius int) bool
	UpdateLightByType(kind EnumSkyBlock, x, y, z int)

	GetChunkHeightMapMinimum(x, z int) int
	GetHeightValue(x, z int) int

	OnBlockPreDestroy(blockID, x, y, z, metadata int)
	BreakBlock(blockID, x, y, z, oldBlockID, oldMetadata int)
	OnBlockAdded(blockID, x, y, z int)

	RemoveBlockTileEntity(x, y, z int)
	EnsureBlockTileEntity(blockID, x, y, z int)
	UpdateBlockTileEntityInfo(x, y, z int)
}

type nopWorldBridge struct{}

func (nopWorldBridge) HasNoSky() bool { return false }
func (nopWorldBridge) IsRemote() bool { return false }

func (nopWorldBridge) MarkBlockForRenderUpdate(x, y, z int) {}
func (nopWorldBridge) MarkBlocksDirtyVertical(x, z, yFrom, yTo int) {
}

func (nopWorldBridge) DoChunksNearChunkExist(x, y, z, radius int) bool { return false }
func (nopWorldBridge) UpdateLightByType(kind EnumSkyBlock, x, y, z int) {
}

func (nopWorldBridge) GetChunkHeightMapMinimum(x, z int) int { return 0 }
func (nopWorldBridge) GetHeightValue(x, z int) int           { return 0 }

func (nopWorldBridge) OnBlockPreDestroy(blockID, x, y, z, metadata int)         {}
func (nopWorldBridge) BreakBlock(blockID, x, y, z, oldBlockID, oldMetadata int) {}
func (nopWorldBridge) OnBlockAdded(blockID, x, y, z int)                        {}
func (nopWorldBridge) RemoveBlockTileEntity(x, y, z int)                        {}
func (nopWorldBridge) EnsureBlockTileEntity(blockID, x, y, z int)               {}
func (nopWorldBridge) UpdateBlockTileEntityInfo(x, y, z int)                    {}
