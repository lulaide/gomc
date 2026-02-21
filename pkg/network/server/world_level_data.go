package server

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lulaide/gomc/pkg/nbt"
)

type worldLevelData struct {
	RandomSeed       int64
	GeneratorName    string
	GeneratorVersion int32
	GeneratorOptions string
	SpawnX           int32
	SpawnY           int32
	SpawnZ           int32
}

func defaultWorldLevelData() worldLevelData {
	return worldLevelData{
		RandomSeed:       time.Now().UnixNano() / int64(time.Millisecond),
		GeneratorName:    "default",
		GeneratorVersion: 1,
		GeneratorOptions: "",
		SpawnX:           0,
		SpawnY:           64,
		SpawnZ:           0,
	}
}

// loadWorldLevelData reads world seed/generator/spawn metadata from level.dat.
// Returns `exists=false` when level.dat does not exist.
func loadWorldLevelData(worldDir string) (data worldLevelData, exists bool, err error) {
	data = defaultWorldLevelData()
	if strings.TrimSpace(worldDir) == "" {
		return data, false, nil
	}

	levelPath := filepath.Join(worldDir, "level.dat")
	f, openErr := os.Open(levelPath)
	if openErr != nil {
		if os.IsNotExist(openErr) {
			return data, false, nil
		}
		return data, false, openErr
	}
	defer f.Close()

	root, readErr := nbt.ReadCompressed(f)
	if readErr != nil {
		return data, false, readErr
	}
	if root == nil {
		return data, true, nil
	}

	compound, ok := root.GetTag("Data").(*nbt.CompoundTag)
	if !ok || compound == nil {
		return data, true, nil
	}

	data.RandomSeed = nbtTagAsInt64(compound.GetTag("RandomSeed"), data.RandomSeed)
	if name, ok := nbtTagAsString(compound.GetTag("generatorName")); ok {
		data.GeneratorName = name
	}
	data.GeneratorVersion = int32(nbtTagAsInt64(compound.GetTag("generatorVersion"), int64(data.GeneratorVersion)))
	if options, ok := nbtTagAsString(compound.GetTag("generatorOptions")); ok {
		data.GeneratorOptions = options
	}
	data.SpawnX = int32(nbtTagAsInt64(compound.GetTag("SpawnX"), int64(data.SpawnX)))
	data.SpawnY = int32(nbtTagAsInt64(compound.GetTag("SpawnY"), int64(data.SpawnY)))
	data.SpawnZ = int32(nbtTagAsInt64(compound.GetTag("SpawnZ"), int64(data.SpawnZ)))
	return data, true, nil
}

func nbtTagAsString(tag nbt.Tag) (string, bool) {
	if v, ok := tag.(*nbt.StringTag); ok {
		return v.Data, true
	}
	return "", false
}

func nbtTagAsInt64(tag nbt.Tag, fallback int64) int64 {
	switch v := tag.(type) {
	case *nbt.ByteTag:
		return int64(v.Data)
	case *nbt.ShortTag:
		return int64(v.Data)
	case *nbt.IntTag:
		return int64(v.Data)
	case *nbt.LongTag:
		return v.Data
	default:
		return fallback
	}
}
