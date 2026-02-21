package gen

import "sync"

// Translation reference:
// - net.minecraft.src.IntCache
var (
	intCacheMu sync.Mutex

	intCacheSize = 256

	freeSmallArrays [][]int
	inUseSmallArray [][]int

	freeLargeArrays [][]int
	inUseLargeArray [][]int
)

func genLayerGetIntCache(size int) []int {
	intCacheMu.Lock()
	defer intCacheMu.Unlock()

	if size <= 256 {
		if len(freeSmallArrays) == 0 {
			arr := make([]int, 256)
			inUseSmallArray = append(inUseSmallArray, arr)
			return arr
		}
		last := len(freeSmallArrays) - 1
		arr := freeSmallArrays[last]
		freeSmallArrays = freeSmallArrays[:last]
		inUseSmallArray = append(inUseSmallArray, arr)
		return arr
	}

	if size > intCacheSize {
		intCacheSize = size
		freeLargeArrays = freeLargeArrays[:0]
		inUseLargeArray = inUseLargeArray[:0]
		arr := make([]int, intCacheSize)
		inUseLargeArray = append(inUseLargeArray, arr)
		return arr
	}

	if len(freeLargeArrays) == 0 {
		arr := make([]int, intCacheSize)
		inUseLargeArray = append(inUseLargeArray, arr)
		return arr
	}

	last := len(freeLargeArrays) - 1
	arr := freeLargeArrays[last]
	freeLargeArrays = freeLargeArrays[:last]
	inUseLargeArray = append(inUseLargeArray, arr)
	return arr
}

func genLayerResetIntCache() {
	intCacheMu.Lock()
	defer intCacheMu.Unlock()

	if len(freeLargeArrays) != 0 {
		freeLargeArrays = freeLargeArrays[:len(freeLargeArrays)-1]
	}
	if len(freeSmallArrays) != 0 {
		freeSmallArrays = freeSmallArrays[:len(freeSmallArrays)-1]
	}

	freeLargeArrays = append(freeLargeArrays, inUseLargeArray...)
	freeSmallArrays = append(freeSmallArrays, inUseSmallArray...)
	inUseLargeArray = inUseLargeArray[:0]
	inUseSmallArray = inUseSmallArray[:0]
}
