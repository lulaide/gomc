package storage

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

const sectorBytes = 4096

var emptySector = make([]byte, sectorBytes)

// RegionFile translates net.minecraft.src.RegionFile (.mca anvil sector layout).
type RegionFile struct {
	fileName string
	dataFile *os.File

	offsets         [1024]int32
	chunkTimestamps [1024]int32
	sectorFree      []bool

	sizeDelta    int
	lastModified int64

	mu sync.Mutex
}

func OpenRegionFile(path string) (*RegionFile, error) {
	rf := &RegionFile{
		fileName: path,
	}

	if fi, err := os.Stat(path); err == nil {
		rf.lastModified = fi.ModTime().UnixMilli()
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	rf.dataFile = file

	if err := rf.initialize(); err != nil {
		_ = file.Close()
		return nil, err
	}

	return rf, nil
}

func (rf *RegionFile) initialize() error {
	length, err := rf.dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	if length < sectorBytes {
		if _, err := rf.dataFile.Seek(0, io.SeekStart); err != nil {
			return err
		}
		for i := 0; i < 1024; i++ {
			if err := binary.Write(rf.dataFile, binary.BigEndian, int32(0)); err != nil {
				return err
			}
		}
		for i := 0; i < 1024; i++ {
			if err := binary.Write(rf.dataFile, binary.BigEndian, int32(0)); err != nil {
				return err
			}
		}
		rf.sizeDelta += 8192
	}

	length, err = rf.dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	// Translation target: RegionFile constructor.
	// This keeps the original loop behavior from MCP source.
	remainder := length & 4095
	if remainder != 0 {
		for i := int64(0); i < remainder; i++ {
			if _, err := rf.dataFile.Write([]byte{0}); err != nil {
				return err
			}
		}
	}

	length, err = rf.dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	sectors := int(length / sectorBytes)
	rf.sectorFree = make([]bool, sectors)
	for i := range rf.sectorFree {
		rf.sectorFree[i] = true
	}
	if len(rf.sectorFree) > 0 {
		rf.sectorFree[0] = false
	}
	if len(rf.sectorFree) > 1 {
		rf.sectorFree[1] = false
	}

	if _, err := rf.dataFile.Seek(0, io.SeekStart); err != nil {
		return err
	}
	for i := 0; i < 1024; i++ {
		var v int32
		if err := binary.Read(rf.dataFile, binary.BigEndian, &v); err != nil {
			return err
		}
		rf.offsets[i] = v
		if v == 0 {
			continue
		}
		sectorNumber := int(v >> 8)
		sectorCount := int(v & 255)
		if sectorNumber+sectorCount <= len(rf.sectorFree) {
			for n := 0; n < sectorCount; n++ {
				rf.sectorFree[sectorNumber+n] = false
			}
		}
	}
	for i := 0; i < 1024; i++ {
		var v int32
		if err := binary.Read(rf.dataFile, binary.BigEndian, &v); err != nil {
			return err
		}
		rf.chunkTimestamps[i] = v
	}
	return nil
}

func (rf *RegionFile) outOfBounds(x, z int) bool {
	return x < 0 || x >= 32 || z < 0 || z >= 32
}

func (rf *RegionFile) getOffset(x, z int) int32 {
	return rf.offsets[x+z*32]
}

func (rf *RegionFile) setOffset(x, z int, offset int32) error {
	rf.offsets[x+z*32] = offset
	if _, err := rf.dataFile.Seek(int64((x+z*32)*4), io.SeekStart); err != nil {
		return err
	}
	return binary.Write(rf.dataFile, binary.BigEndian, offset)
}

func (rf *RegionFile) setChunkTimestamp(x, z int, ts int32) error {
	rf.chunkTimestamps[x+z*32] = ts
	if _, err := rf.dataFile.Seek(int64(sectorBytes+(x+z*32)*4), io.SeekStart); err != nil {
		return err
	}
	return binary.Write(rf.dataFile, binary.BigEndian, ts)
}

// IsChunkSaved translates RegionFile#isChunkSaved.
func (rf *RegionFile) IsChunkSaved(x, z int) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.outOfBounds(x, z) {
		return false
	}
	return rf.getOffset(x, z) != 0
}

// GetChunkDataInputStream translates RegionFile#getChunkDataInputStream.
func (rf *RegionFile) GetChunkDataInputStream(x, z int) (io.ReadCloser, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.outOfBounds(x, z) {
		return nil, nil
	}

	offset := rf.getOffset(x, z)
	if offset == 0 {
		return nil, nil
	}

	sectorNumber := int(offset >> 8)
	sectorCount := int(offset & 255)
	if sectorNumber+sectorCount > len(rf.sectorFree) {
		return nil, nil
	}

	if _, err := rf.dataFile.Seek(int64(sectorNumber*sectorBytes), io.SeekStart); err != nil {
		return nil, err
	}

	var length int32
	if err := binary.Read(rf.dataFile, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length > int32(sectorBytes*sectorCount) || length <= 0 {
		return nil, nil
	}

	var compressionType [1]byte
	if _, err := io.ReadFull(rf.dataFile, compressionType[:]); err != nil {
		return nil, err
	}

	payload := make([]byte, int(length)-1)
	if _, err := io.ReadFull(rf.dataFile, payload); err != nil {
		return nil, err
	}

	switch compressionType[0] {
	case 1:
		return gzip.NewReader(bytes.NewReader(payload))
	case 2:
		return zlib.NewReader(bytes.NewReader(payload))
	default:
		return nil, nil
	}
}

type chunkDataOutputStream struct {
	rf     *RegionFile
	x      int
	z      int
	buf    bytes.Buffer
	zw     *zlib.Writer
	closed bool
}

func (s *chunkDataOutputStream) Write(p []byte) (int, error) {
	if s.closed {
		return 0, errors.New("stream closed")
	}
	return s.zw.Write(p)
}

func (s *chunkDataOutputStream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if err := s.zw.Close(); err != nil {
		return err
	}
	return s.rf.writeChunk(s.x, s.z, s.buf.Bytes(), s.buf.Len())
}

// GetChunkDataOutputStream translates RegionFile#getChunkDataOutputStream.
func (rf *RegionFile) GetChunkDataOutputStream(x, z int) (io.WriteCloser, error) {
	if rf.outOfBounds(x, z) {
		return nil, nil
	}

	stream := &chunkDataOutputStream{
		rf: rf,
		x:  x,
		z:  z,
	}
	stream.zw = zlib.NewWriter(&stream.buf)
	return stream, nil
}

func (rf *RegionFile) writeSector(sectorNumber int, compressed []byte, length int) error {
	if _, err := rf.dataFile.Seek(int64(sectorNumber*sectorBytes), io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(rf.dataFile, binary.BigEndian, int32(length+1)); err != nil {
		return err
	}
	if _, err := rf.dataFile.Write([]byte{2}); err != nil {
		return err
	}
	_, err := rf.dataFile.Write(compressed[:length])
	return err
}

// writeChunk translates RegionFile#write(int,int,byte[],int).
func (rf *RegionFile) writeChunk(x, z int, compressed []byte, length int) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	offset := rf.getOffset(x, z)
	sectorNumber := int(offset >> 8)
	sectorCount := int(offset & 255)
	requiredSectors := (length+5)/sectorBytes + 1
	if requiredSectors >= 256 {
		return nil
	}

	if sectorNumber != 0 && sectorCount == requiredSectors {
		if err := rf.writeSector(sectorNumber, compressed, length); err != nil {
			return err
		}
	} else {
		for i := 0; i < sectorCount; i++ {
			if sectorNumber+i >= 0 && sectorNumber+i < len(rf.sectorFree) {
				rf.sectorFree[sectorNumber+i] = true
			}
		}

		start := rf.indexOfFreeSector()
		freeRun := 0
		if start != -1 {
			for i := start; i < len(rf.sectorFree); i++ {
				if freeRun != 0 {
					if rf.sectorFree[i] {
						freeRun++
					} else {
						freeRun = 0
					}
				} else if rf.sectorFree[i] {
					start = i
					freeRun = 1
				}
				if freeRun >= requiredSectors {
					break
				}
			}
		}

		if freeRun >= requiredSectors {
			sectorNumber = start
			if err := rf.setOffset(x, z, int32((start<<8)|requiredSectors)); err != nil {
				return err
			}
			for i := 0; i < requiredSectors; i++ {
				rf.sectorFree[sectorNumber+i] = false
			}
			if err := rf.writeSector(sectorNumber, compressed, length); err != nil {
				return err
			}
		} else {
			endPos, err := rf.dataFile.Seek(0, io.SeekEnd)
			if err != nil {
				return err
			}
			sectorNumber = len(rf.sectorFree)
			for i := 0; i < requiredSectors; i++ {
				if _, err := rf.dataFile.Write(emptySector); err != nil {
					return err
				}
				rf.sectorFree = append(rf.sectorFree, false)
			}
			rf.sizeDelta += sectorBytes * requiredSectors
			if err := rf.writeSector(sectorNumber, compressed, length); err != nil {
				return err
			}
			if err := rf.setOffset(x, z, int32((sectorNumber<<8)|requiredSectors)); err != nil {
				return err
			}

			// keep last modified approximation updated when file grows.
			rf.lastModified = time.Now().UnixMilli()
			_, _ = rf.dataFile.Seek(endPos, io.SeekStart)
		}
	}

	return rf.setChunkTimestamp(x, z, int32(time.Now().UnixMilli()/1000))
}

func (rf *RegionFile) indexOfFreeSector() int {
	for i, free := range rf.sectorFree {
		if free {
			return i
		}
	}
	return -1
}

func (rf *RegionFile) Close() error {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.dataFile == nil {
		return nil
	}
	err := rf.dataFile.Close()
	rf.dataFile = nil
	return err
}
