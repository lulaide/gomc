package chunk

import "github.com/lulaide/gomc/pkg/world/block"

const sectionVolume = 16 * 16 * 16

// ExtendedBlockStorage translates net.minecraft.src.ExtendedBlockStorage.
type ExtendedBlockStorage struct {
	yBase int

	blockRefCount int
	tickRefCount  int

	blockLSBArray      []byte
	blockMSBArray      *NibbleArray
	blockMetadataArray *NibbleArray
	blocklightArray    *NibbleArray
	skylightArray      *NibbleArray
}

func NewExtendedBlockStorage(yBase int, hasSky bool) *ExtendedBlockStorage {
	lsb := make([]byte, sectionVolume)
	s := &ExtendedBlockStorage{
		yBase:              yBase,
		blockLSBArray:      lsb,
		blockMetadataArray: NewNibbleArray(len(lsb), 4),
		blocklightArray:    NewNibbleArray(len(lsb), 4),
	}
	if hasSky {
		s.skylightArray = NewNibbleArray(len(lsb), 4)
	}
	return s
}

func (s *ExtendedBlockStorage) index(x, y, z int) int {
	return y<<8 | z<<4 | x
}

// GetExtBlockID merges block LSB/MSB to return 12-bit id.
//
// Translation target: ExtendedBlockStorage#getExtBlockID
func (s *ExtendedBlockStorage) GetExtBlockID(x, y, z int) int {
	idx := s.index(x, y, z)
	lsb := int(s.blockLSBArray[idx]) & 0xFF
	if s.blockMSBArray != nil {
		return (s.blockMSBArray.Get(x, y, z) << 8) | lsb
	}
	return lsb
}

// SetExtBlockID updates block id and reference counters.
//
// Translation target: ExtendedBlockStorage#setExtBlockID
func (s *ExtendedBlockStorage) SetExtBlockID(x, y, z, blockID int) {
	idx := s.index(x, y, z)
	previous := int(s.blockLSBArray[idx]) & 0xFF
	if s.blockMSBArray != nil {
		previous |= s.blockMSBArray.Get(x, y, z) << 8
	}

	if previous == 0 && blockID != 0 {
		s.blockRefCount++
		if block.GetTickRandomly(blockID) {
			s.tickRefCount++
		}
	} else if previous != 0 && blockID == 0 {
		s.blockRefCount--
		if block.GetTickRandomly(previous) {
			s.tickRefCount--
		}
	} else if block.GetTickRandomly(previous) && !block.GetTickRandomly(blockID) {
		s.tickRefCount--
	} else if !block.GetTickRandomly(previous) && block.GetTickRandomly(blockID) {
		s.tickRefCount++
	}

	s.blockLSBArray[idx] = byte(blockID & 0xFF)
	if blockID > 255 {
		if s.blockMSBArray == nil {
			s.blockMSBArray = NewNibbleArray(len(s.blockLSBArray), 4)
		}
		s.blockMSBArray.Set(x, y, z, (blockID&0xF00)>>8)
	} else if s.blockMSBArray != nil {
		s.blockMSBArray.Set(x, y, z, 0)
	}
}

func (s *ExtendedBlockStorage) GetExtBlockMetadata(x, y, z int) int {
	return s.blockMetadataArray.Get(x, y, z)
}

func (s *ExtendedBlockStorage) SetExtBlockMetadata(x, y, z, metadata int) {
	s.blockMetadataArray.Set(x, y, z, metadata)
}

func (s *ExtendedBlockStorage) IsEmpty() bool {
	return s.blockRefCount == 0
}

func (s *ExtendedBlockStorage) GetNeedsRandomTick() bool {
	return s.tickRefCount > 0
}

func (s *ExtendedBlockStorage) GetYLocation() int {
	return s.yBase
}

func (s *ExtendedBlockStorage) SetExtSkylightValue(x, y, z, value int) {
	if s.skylightArray != nil {
		s.skylightArray.Set(x, y, z, value)
	}
}

func (s *ExtendedBlockStorage) GetExtSkylightValue(x, y, z int) int {
	if s.skylightArray == nil {
		return 0
	}
	return s.skylightArray.Get(x, y, z)
}

func (s *ExtendedBlockStorage) SetExtBlocklightValue(x, y, z, value int) {
	s.blocklightArray.Set(x, y, z, value)
}

func (s *ExtendedBlockStorage) GetExtBlocklightValue(x, y, z int) int {
	return s.blocklightArray.Get(x, y, z)
}

// RemoveInvalidBlocks recomputes counters and clears unknown block ids.
//
// Translation target: ExtendedBlockStorage#removeInvalidBlocks
func (s *ExtendedBlockStorage) RemoveInvalidBlocks() {
	s.blockRefCount = 0
	s.tickRefCount = 0

	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				id := s.GetExtBlockID(x, y, z)
				if id <= 0 {
					continue
				}
				if !block.Exists(id) {
					s.blockLSBArray[s.index(x, y, z)] = 0
					if s.blockMSBArray != nil {
						s.blockMSBArray.Set(x, y, z, 0)
					}
					continue
				}

				s.blockRefCount++
				if block.GetTickRandomly(id) {
					s.tickRefCount++
				}
			}
		}
	}
}

func (s *ExtendedBlockStorage) GetBlockLSBArray() []byte {
	return s.blockLSBArray
}

func (s *ExtendedBlockStorage) ClearMSBArray() {
	s.blockMSBArray = nil
}

func (s *ExtendedBlockStorage) GetBlockMSBArray() *NibbleArray {
	return s.blockMSBArray
}

func (s *ExtendedBlockStorage) GetMetadataArray() *NibbleArray {
	return s.blockMetadataArray
}

func (s *ExtendedBlockStorage) GetBlocklightArray() *NibbleArray {
	return s.blocklightArray
}

func (s *ExtendedBlockStorage) GetSkylightArray() *NibbleArray {
	return s.skylightArray
}

func (s *ExtendedBlockStorage) SetBlockLSBArray(v []byte) {
	s.blockLSBArray = v
}

func (s *ExtendedBlockStorage) SetBlockMSBArray(v *NibbleArray) {
	s.blockMSBArray = v
}

func (s *ExtendedBlockStorage) SetBlockMetadataArray(v *NibbleArray) {
	s.blockMetadataArray = v
}

func (s *ExtendedBlockStorage) SetBlocklightArray(v *NibbleArray) {
	s.blocklightArray = v
}

func (s *ExtendedBlockStorage) SetSkylightArray(v *NibbleArray) {
	s.skylightArray = v
}

func (s *ExtendedBlockStorage) CreateBlockMSBArray() *NibbleArray {
	s.blockMSBArray = NewNibbleArray(len(s.blockLSBArray), 4)
	return s.blockMSBArray
}
