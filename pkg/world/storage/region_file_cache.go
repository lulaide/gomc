package storage

import (
	"fmt"
	"io"
	"path/filepath"
	"sync"
)

var (
	cacheMu           sync.Mutex
	regionsByFilename = map[string]*RegionFile{}
)

// CreateOrLoadRegionFile translates RegionFileCache#createOrLoadRegionFile.
func CreateOrLoadRegionFile(worldDir string, chunkX, chunkZ int) (*RegionFile, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	regionDir := filepath.Join(worldDir, "region")
	regionPath := filepath.Join(regionDir, fmt.Sprintf("r.%d.%d.mca", chunkX>>5, chunkZ>>5))

	if rf, ok := regionsByFilename[regionPath]; ok {
		return rf, nil
	}

	if len(regionsByFilename) >= 256 {
		if err := clearRegionFileReferencesLocked(); err != nil {
			return nil, err
		}
	}

	if err := ensureDir(regionDir); err != nil {
		return nil, err
	}

	rf, err := OpenRegionFile(regionPath)
	if err != nil {
		return nil, err
	}
	regionsByFilename[regionPath] = rf
	return rf, nil
}

// ClearRegionFileReferences translates RegionFileCache#clearRegionFileReferences.
func ClearRegionFileReferences() error {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	return clearRegionFileReferencesLocked()
}

func clearRegionFileReferencesLocked() error {
	var firstErr error
	for _, rf := range regionsByFilename {
		if rf == nil {
			continue
		}
		if err := rf.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	regionsByFilename = map[string]*RegionFile{}
	return firstErr
}

// GetChunkInputStream translates RegionFileCache#getChunkInputStream.
func GetChunkInputStream(worldDir string, chunkX, chunkZ int) (io.ReadCloser, error) {
	rf, err := CreateOrLoadRegionFile(worldDir, chunkX, chunkZ)
	if err != nil {
		return nil, err
	}
	return rf.GetChunkDataInputStream(chunkX&31, chunkZ&31)
}

// GetChunkOutputStream translates RegionFileCache#getChunkOutputStream.
func GetChunkOutputStream(worldDir string, chunkX, chunkZ int) (io.WriteCloser, error) {
	rf, err := CreateOrLoadRegionFile(worldDir, chunkX, chunkZ)
	if err != nil {
		return nil, err
	}
	return rf.GetChunkDataOutputStream(chunkX&31, chunkZ&31)
}

func ensureDir(path string) error {
	return mkdirAll(path)
}
