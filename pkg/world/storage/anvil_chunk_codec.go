package storage

import (
	"github.com/lulaide/gomc/pkg/nbt"
	"github.com/lulaide/gomc/pkg/tick"
	"github.com/lulaide/gomc/pkg/world/chunk"
)

// TileTickRecord translates "TileTicks" list entries used by AnvilChunkLoader.
type TileTickRecord struct {
	BlockID  int32
	X        int32
	Y        int32
	Z        int32
	Delay    int32
	Priority int32
}

// EncodeChunkLevelNBT translates the chunk "Level" payload written by:
// net.minecraft.src.AnvilChunkLoader#writeChunkToNBT
func EncodeChunkLevelNBT(col *chunk.Column, totalWorldTime int64, hasNoSky bool, pending []*tick.NextTickListEntry) *nbt.CompoundTag {
	level := nbt.NewCompoundTag("Level")
	level.SetInteger("xPos", col.XPos)
	level.SetInteger("zPos", col.ZPos)
	level.SetLong("LastUpdate", totalWorldTime)
	level.SetIntArray("HeightMap", col.HeightMap)
	level.SetBoolean("TerrainPopulated", col.IsTerrainPopulated)
	level.SetLong("InhabitedTime", col.InhabitedTime)

	sections := nbt.NewListTag("Sections")
	hasSky := !hasNoSky
	for _, sec := range col.GetStorageArrays() {
		if sec == nil {
			continue
		}

		s := nbt.NewCompoundTag("")
		s.SetByte("Y", int8((sec.GetYLocation()>>4)&255))
		s.SetByteArray("Blocks", sec.GetBlockLSBArray())

		if sec.GetBlockMSBArray() != nil {
			s.SetByteArray("Add", sec.GetBlockMSBArray().Data)
		}

		s.SetByteArray("Data", sec.GetMetadataArray().Data)
		s.SetByteArray("BlockLight", sec.GetBlocklightArray().Data)

		if hasSky {
			if sec.GetSkylightArray() != nil {
				s.SetByteArray("SkyLight", sec.GetSkylightArray().Data)
			} else {
				s.SetByteArray("SkyLight", make([]byte, len(sec.GetBlocklightArray().Data)))
			}
		} else {
			s.SetByteArray("SkyLight", make([]byte, len(sec.GetBlocklightArray().Data)))
		}

		sections.AppendTag(s)
	}
	level.SetTag("Sections", sections)
	level.SetByteArray("Biomes", col.GetBiomeArray())

	if len(pending) > 0 {
		ticks := nbt.NewListTag("TileTicks")
		for _, e := range pending {
			t := nbt.NewCompoundTag("")
			t.SetInteger("i", int32(e.BlockID))
			t.SetInteger("x", int32(e.XCoord))
			t.SetInteger("y", int32(e.YCoord))
			t.SetInteger("z", int32(e.ZCoord))
			t.SetInteger("t", int32(e.ScheduledTime-totalWorldTime))
			t.SetInteger("p", int32(e.Priority))
			ticks.AppendTag(t)
		}
		level.SetTag("TileTicks", ticks)
	}

	return level
}

// DecodeChunkLevelNBT translates section and tile tick reads from:
// net.minecraft.src.AnvilChunkLoader#readChunkFromNBT
func DecodeChunkLevelNBT(level *nbt.CompoundTag, hasNoSky bool) (*chunk.Column, []TileTickRecord) {
	xPos := getInt(level, "xPos")
	zPos := getInt(level, "zPos")

	col := chunk.NewColumn(xPos, zPos)
	col.HeightMap = append([]int32(nil), getIntArray(level, "HeightMap")...)
	col.IsTerrainPopulated = getBoolean(level, "TerrainPopulated")
	col.InhabitedTime = getLong(level, "InhabitedTime")

	sectionsTag := getList(level, "Sections")
	hasSky := !hasNoSky
	storageArrays := make([]*chunk.ExtendedBlockStorage, 16)

	for i := 0; i < sectionsTag.TagCount(); i++ {
		sectionCompound, ok := sectionsTag.TagAt(i).(*nbt.CompoundTag)
		if !ok {
			continue
		}

		y := uint8(getByte(sectionCompound, "Y"))
		sec := chunk.NewExtendedBlockStorage(int(y)<<4, hasSky)
		sec.SetBlockLSBArray(getByteArray(sectionCompound, "Blocks"))

		if sectionCompound.HasKey("Add") {
			sec.SetBlockMSBArray(chunk.NewNibbleArrayFromData(getByteArray(sectionCompound, "Add"), 4))
		}

		sec.SetBlockMetadataArray(chunk.NewNibbleArrayFromData(getByteArray(sectionCompound, "Data"), 4))
		sec.SetBlocklightArray(chunk.NewNibbleArrayFromData(getByteArray(sectionCompound, "BlockLight"), 4))

		if hasSky {
			sec.SetSkylightArray(chunk.NewNibbleArrayFromData(getByteArray(sectionCompound, "SkyLight"), 4))
		}

		sec.RemoveInvalidBlocks()
		if int(y) < len(storageArrays) {
			storageArrays[int(y)] = sec
		}
	}
	col.SetStorageArrays(storageArrays)

	if level.HasKey("Biomes") {
		col.SetBiomeArray(getByteArray(level, "Biomes"))
	}

	var tileTicks []TileTickRecord
	if level.HasKey("TileTicks") {
		ticks := getList(level, "TileTicks")
		tileTicks = make([]TileTickRecord, 0, ticks.TagCount())
		for i := 0; i < ticks.TagCount(); i++ {
			tickTag, ok := ticks.TagAt(i).(*nbt.CompoundTag)
			if !ok {
				continue
			}
			tileTicks = append(tileTicks, TileTickRecord{
				BlockID:  getInt(tickTag, "i"),
				X:        getInt(tickTag, "x"),
				Y:        getInt(tickTag, "y"),
				Z:        getInt(tickTag, "z"),
				Delay:    getInt(tickTag, "t"),
				Priority: getInt(tickTag, "p"),
			})
		}
	}

	return col, tileTicks
}

func getByte(comp *nbt.CompoundTag, key string) int8 {
	tag := comp.GetTag(key)
	if v, ok := tag.(*nbt.ByteTag); ok {
		return v.Data
	}
	return 0
}

func getInt(comp *nbt.CompoundTag, key string) int32 {
	tag := comp.GetTag(key)
	if v, ok := tag.(*nbt.IntTag); ok {
		return v.Data
	}
	return 0
}

func getLong(comp *nbt.CompoundTag, key string) int64 {
	tag := comp.GetTag(key)
	if v, ok := tag.(*nbt.LongTag); ok {
		return v.Data
	}
	return 0
}

func getBoolean(comp *nbt.CompoundTag, key string) bool {
	return getByte(comp, key) != 0
}

func getByteArray(comp *nbt.CompoundTag, key string) []byte {
	tag := comp.GetTag(key)
	if v, ok := tag.(*nbt.ByteArrayTag); ok {
		cp := make([]byte, len(v.Bytes))
		copy(cp, v.Bytes)
		return cp
	}
	return []byte{}
}

func getIntArray(comp *nbt.CompoundTag, key string) []int32 {
	tag := comp.GetTag(key)
	if v, ok := tag.(*nbt.IntArrayTag); ok {
		cp := make([]int32, len(v.Ints))
		copy(cp, v.Ints)
		return cp
	}
	return []int32{}
}

func getList(comp *nbt.CompoundTag, key string) *nbt.ListTag {
	tag := comp.GetTag(key)
	if v, ok := tag.(*nbt.ListTag); ok {
		return v
	}
	return nbt.NewListTag(key)
}
