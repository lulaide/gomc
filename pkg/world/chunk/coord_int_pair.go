package chunk

import "fmt"

// CoordIntPair translates net.minecraft.src.ChunkCoordIntPair.
type CoordIntPair struct {
	ChunkXPos int32
	ChunkZPos int32
}

func NewCoordIntPair(x, z int32) CoordIntPair {
	return CoordIntPair{
		ChunkXPos: x,
		ChunkZPos: z,
	}
}

// ChunkXZToInt translates ChunkCoordIntPair#chunkXZ2Int.
func ChunkXZToInt(x, z int32) int64 {
	return int64(uint64(uint32(x)) | (uint64(uint32(z)) << 32))
}

// HashCode translates ChunkCoordIntPair#hashCode.
func (c CoordIntPair) HashCode() int32 {
	v := ChunkXZToInt(c.ChunkXPos, c.ChunkZPos)
	lo := int32(v)
	hi := int32(v >> 32)
	return lo ^ hi
}

func (c CoordIntPair) Equals(other CoordIntPair) bool {
	return other.ChunkXPos == c.ChunkXPos && other.ChunkZPos == c.ChunkZPos
}

func (c CoordIntPair) GetCenterXPos() int32 {
	return (c.ChunkXPos << 4) + 8
}

func (c CoordIntPair) GetCenterZPos() int32 {
	return (c.ChunkZPos << 4) + 8
}

func (c CoordIntPair) String() string {
	return fmt.Sprintf("[%d, %d]", c.ChunkXPos, c.ChunkZPos)
}
