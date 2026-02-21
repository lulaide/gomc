package protocol

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"

	"github.com/lulaide/gomc/pkg/world/chunk"
)

const mapChunkTempBufferSize = 196864

type Packet51MapChunkData struct {
	CompressedData         []byte
	ChunkExistFlag         int32
	ChunkHasAddSectionFlag int32
}

// Packet51MapChunk translates net.minecraft.src.Packet51MapChunk.
type Packet51MapChunk struct {
	XCh int32
	ZCh int32

	YChMin int32
	YChMax int32

	IncludeInitialize bool
	TempLength        int32

	chunkData           []byte
	compressedChunkData []byte
}

func (*Packet51MapChunk) PacketID() uint8 { return 51 }

func (p *Packet51MapChunk) ReadPacketData(r *Reader) error {
	x, err := r.ReadInt32()
	if err != nil {
		return err
	}
	z, err := r.ReadInt32()
	if err != nil {
		return err
	}
	includeByte, err := r.ReadUint8()
	if err != nil {
		return err
	}
	yMin, err := r.ReadInt16()
	if err != nil {
		return err
	}
	yMax, err := r.ReadInt16()
	if err != nil {
		return err
	}
	tempLength, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if tempLength < 0 {
		return fmt.Errorf("invalid compressed chunk data length: %d", tempLength)
	}

	tmp := make([]byte, int(tempLength))
	if _, err := io.ReadFull(r.r, tmp); err != nil {
		return err
	}

	sectionCount := 0
	mask := uint16(yMin)
	for i := 0; i < 16; i++ {
		sectionCount += int((mask >> i) & 1)
	}

	decompressedSize := 12288 * sectionCount
	if includeByte != 0 {
		decompressedSize += 256
	}

	inflater, err := zlib.NewReader(bytes.NewReader(tmp))
	if err != nil {
		return fmt.Errorf("bad compressed data format")
	}
	defer inflater.Close()

	decompressed, err := io.ReadAll(inflater)
	if err != nil {
		return fmt.Errorf("bad compressed data format")
	}
	expanded := make([]byte, decompressedSize)
	copy(expanded, decompressed)

	p.XCh = x
	p.ZCh = z
	p.IncludeInitialize = includeByte != 0
	p.YChMin = int32(yMin)
	p.YChMax = int32(yMax)
	p.TempLength = tempLength
	p.chunkData = tmp
	p.compressedChunkData = expanded
	return nil
}

func (p *Packet51MapChunk) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.XCh); err != nil {
		return err
	}
	if err := w.WriteInt32(p.ZCh); err != nil {
		return err
	}
	if p.IncludeInitialize {
		if err := w.WriteUint8(1); err != nil {
			return err
		}
	} else {
		if err := w.WriteUint8(0); err != nil {
			return err
		}
	}
	if err := w.WriteInt16(int16(p.YChMin & 65535)); err != nil {
		return err
	}
	if err := w.WriteInt16(int16(p.YChMax & 65535)); err != nil {
		return err
	}
	if err := w.WriteInt32(p.TempLength); err != nil {
		return err
	}
	_, err := w.w.Write(p.chunkData[:int(p.TempLength)])
	return err
}

func (p *Packet51MapChunk) PacketSize() int {
	return 17 + int(p.TempLength)
}

func (p *Packet51MapChunk) GetCompressedChunkData() []byte {
	out := make([]byte, len(p.compressedChunkData))
	copy(out, p.compressedChunkData)
	return out
}

func buildMapChunkData(ch *chunk.Chunk, includeInitialize bool, sectionMask int32, hasNoSky bool) Packet51MapChunkData {
	sections := ch.GetBlockStorageArray()
	chunkExistFlag := int32(0)
	chunkHasAddFlag := int32(0)
	addSectionCount := 0

	if includeInitialize {
		ch.SendUpdates = true
	}

	for i := 0; i < len(sections); i++ {
		sec := sections[i]
		if sec == nil {
			continue
		}
		if includeInitialize && sec.IsEmpty() {
			continue
		}
		if (sectionMask & (1 << i)) == 0 {
			continue
		}

		chunkExistFlag |= 1 << i
		if sec.GetBlockMSBArray() != nil {
			chunkHasAddFlag |= 1 << i
			addSectionCount++
		}
	}

	buffer := make([]byte, 0, mapChunkTempBufferSize)

	appendSectionData := func(getData func(*chunk.ExtendedBlockStorage) []byte, requireMSB bool) {
		for i := 0; i < len(sections); i++ {
			sec := sections[i]
			if sec == nil {
				continue
			}
			if includeInitialize && sec.IsEmpty() {
				continue
			}
			if (sectionMask & (1 << i)) == 0 {
				continue
			}
			if requireMSB && sec.GetBlockMSBArray() == nil {
				continue
			}
			buffer = append(buffer, getData(sec)...)
		}
	}

	appendSectionData(func(sec *chunk.ExtendedBlockStorage) []byte { return sec.GetBlockLSBArray() }, false)
	appendSectionData(func(sec *chunk.ExtendedBlockStorage) []byte { return sec.GetMetadataArray().Data }, false)
	appendSectionData(func(sec *chunk.ExtendedBlockStorage) []byte { return sec.GetBlocklightArray().Data }, false)

	if !hasNoSky {
		appendSectionData(func(sec *chunk.ExtendedBlockStorage) []byte {
			sk := sec.GetSkylightArray()
			if sk == nil {
				return make([]byte, 2048)
			}
			return sk.Data
		}, false)
	}

	if addSectionCount > 0 {
		appendSectionData(func(sec *chunk.ExtendedBlockStorage) []byte { return sec.GetBlockMSBArray().Data }, true)
	}

	if includeInitialize {
		buffer = append(buffer, ch.GetBiomeArray()...)
	}

	out := make([]byte, len(buffer))
	copy(out, buffer)
	return Packet51MapChunkData{
		CompressedData:         out,
		ChunkExistFlag:         chunkExistFlag,
		ChunkHasAddSectionFlag: chunkHasAddFlag,
	}
}

func NewPacket51MapChunk(ch *chunk.Chunk, includeInitialize bool, sectionMask int32, hasNoSky bool) (*Packet51MapChunk, error) {
	if ch == nil {
		return nil, fmt.Errorf("nil chunk")
	}
	data := buildMapChunkData(ch, includeInitialize, sectionMask, hasNoSky)

	var compressed bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&compressed, zlib.DefaultCompression)
	if _, err := zw.Write(data.CompressedData); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	packet := &Packet51MapChunk{
		XCh:               ch.XPosition,
		ZCh:               ch.ZPosition,
		IncludeInitialize: includeInitialize,
		YChMax:            data.ChunkHasAddSectionFlag,
		YChMin:            data.ChunkExistFlag,
		TempLength:        int32(compressed.Len()),
		chunkData:         compressed.Bytes(),
	}
	packet.compressedChunkData = data.CompressedData
	return packet, nil
}

// Packet56MapChunks translates net.minecraft.src.Packet56MapChunks.
type Packet56MapChunks struct {
	ChunkPosX   []int32
	ChunkPosZ   []int32
	Field73590A []int32
	Field73588B []int32
	Field73584F [][]byte

	ChunkDataBuffer []byte
	DataLength      int32
	SkyLightSent    bool
}

func (*Packet56MapChunks) PacketID() uint8 { return 56 }

func (p *Packet56MapChunks) ReadPacketData(r *Reader) error {
	countShort, err := r.ReadInt16()
	if err != nil {
		return err
	}
	count := int(uint16(countShort))

	dataLength, err := r.ReadInt32()
	if err != nil {
		return err
	}
	if dataLength < 0 {
		return fmt.Errorf("invalid chunk data length: %d", dataLength)
	}
	skyLightByte, err := r.ReadUint8()
	if err != nil {
		return err
	}

	p.DataLength = dataLength
	p.SkyLightSent = skyLightByte != 0
	p.ChunkPosX = make([]int32, count)
	p.ChunkPosZ = make([]int32, count)
	p.Field73590A = make([]int32, count)
	p.Field73588B = make([]int32, count)
	p.Field73584F = make([][]byte, count)

	compressed := make([]byte, int(dataLength))
	if _, err := io.ReadFull(r.r, compressed); err != nil {
		return err
	}
	p.ChunkDataBuffer = compressed

	inflater, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("bad compressed data format")
	}
	defer inflater.Close()

	decompressedAll, err := io.ReadAll(inflater)
	if err != nil {
		return fmt.Errorf("bad compressed data format")
	}

	offset := 0
	for i := 0; i < count; i++ {
		x, err := r.ReadInt32()
		if err != nil {
			return err
		}
		z, err := r.ReadInt32()
		if err != nil {
			return err
		}
		existFlagRaw, err := r.ReadInt16()
		if err != nil {
			return err
		}
		addFlagRaw, err := r.ReadInt16()
		if err != nil {
			return err
		}

		p.ChunkPosX[i] = x
		p.ChunkPosZ[i] = z
		p.Field73590A[i] = int32(existFlagRaw)
		p.Field73588B[i] = int32(addFlagRaw)

		sectionCount := 0
		addCount := 0
		existFlag := uint16(existFlagRaw)
		addFlag := uint16(addFlagRaw)
		for bit := 0; bit < 16; bit++ {
			sectionCount += int((existFlag >> bit) & 1)
			addCount += int((addFlag >> bit) & 1)
		}

		chunkSize := 2048*4*sectionCount + 256
		chunkSize += 2048 * addCount
		if p.SkyLightSent {
			chunkSize += 2048 * sectionCount
		}

		if offset+chunkSize > len(decompressedAll) {
			return fmt.Errorf("bad compressed data format")
		}
		data := make([]byte, chunkSize)
		copy(data, decompressedAll[offset:offset+chunkSize])
		p.Field73584F[i] = data
		offset += chunkSize
	}
	return nil
}

func (p *Packet56MapChunks) WritePacketData(w *Writer) error {
	if err := w.WriteInt16(int16(len(p.ChunkPosX))); err != nil {
		return err
	}
	if err := w.WriteInt32(p.DataLength); err != nil {
		return err
	}
	if p.SkyLightSent {
		if err := w.WriteUint8(1); err != nil {
			return err
		}
	} else {
		if err := w.WriteUint8(0); err != nil {
			return err
		}
	}
	if _, err := w.w.Write(p.ChunkDataBuffer[:int(p.DataLength)]); err != nil {
		return err
	}

	for i := 0; i < len(p.ChunkPosX); i++ {
		if err := w.WriteInt32(p.ChunkPosX[i]); err != nil {
			return err
		}
		if err := w.WriteInt32(p.ChunkPosZ[i]); err != nil {
			return err
		}
		if err := w.WriteInt16(int16(p.Field73590A[i] & 65535)); err != nil {
			return err
		}
		if err := w.WriteInt16(int16(p.Field73588B[i] & 65535)); err != nil {
			return err
		}
	}
	return nil
}

func (p *Packet56MapChunks) PacketSize() int {
	return 6 + int(p.DataLength) + 12*p.GetNumberOfChunkInPacket()
}

func (p *Packet56MapChunks) GetNumberOfChunkInPacket() int {
	return len(p.ChunkPosX)
}

func (p *Packet56MapChunks) GetChunkPosX(index int) int32 {
	return p.ChunkPosX[index]
}

func (p *Packet56MapChunks) GetChunkPosZ(index int) int32 {
	return p.ChunkPosZ[index]
}

func (p *Packet56MapChunks) GetChunkCompressedData(index int) []byte {
	out := make([]byte, len(p.Field73584F[index]))
	copy(out, p.Field73584F[index])
	return out
}

func NewPacket56MapChunks(chunks []*chunk.Chunk, hasNoSky bool) (*Packet56MapChunks, error) {
	count := len(chunks)
	p := &Packet56MapChunks{
		ChunkPosX:    make([]int32, count),
		ChunkPosZ:    make([]int32, count),
		Field73590A:  make([]int32, count),
		Field73588B:  make([]int32, count),
		Field73584F:  make([][]byte, count),
		SkyLightSent: count > 0 && !hasNoSky,
	}

	combined := make([]byte, 0, mapChunkTempBufferSize*count)
	for i, ch := range chunks {
		if ch == nil {
			continue
		}
		data := buildMapChunkData(ch, true, 65535, hasNoSky)
		combined = append(combined, data.CompressedData...)
		p.ChunkPosX[i] = ch.XPosition
		p.ChunkPosZ[i] = ch.ZPosition
		p.Field73590A[i] = data.ChunkExistFlag
		p.Field73588B[i] = data.ChunkHasAddSectionFlag
		p.Field73584F[i] = data.CompressedData
	}

	var compressed bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&compressed, zlib.DefaultCompression)
	if _, err := zw.Write(combined); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	p.ChunkDataBuffer = compressed.Bytes()
	p.DataLength = int32(compressed.Len())
	return p, nil
}

func init() {
	// Translation target: Packet#addIdClassMapping for implemented chunk data packets.
	_ = Register(51, true, false, func() Packet { return &Packet51MapChunk{} })
	_ = Register(56, true, false, func() Packet { return &Packet56MapChunks{} })
}
