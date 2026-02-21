package gen

import (
	"sync"
	"unsafe"
)

var generationMetadataBuffers sync.Map // map[uintptr][]byte

func blockBufferKey(blocks []byte) uintptr {
	if len(blocks) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&blocks[0]))
}

func registerBlockMetadataBuffer(blocks []byte, metadata []byte) {
	key := blockBufferKey(blocks)
	if key == 0 || len(metadata) != len(blocks) {
		return
	}
	generationMetadataBuffers.Store(key, metadata)
}

func unregisterBlockMetadataBuffer(blocks []byte) {
	key := blockBufferKey(blocks)
	if key == 0 {
		return
	}
	generationMetadataBuffers.Delete(key)
}

func metadataBufferForBlocks(blocks []byte) []byte {
	key := blockBufferKey(blocks)
	if key == 0 {
		return nil
	}
	value, ok := generationMetadataBuffers.Load(key)
	if !ok {
		return nil
	}
	meta, _ := value.([]byte)
	return meta
}
