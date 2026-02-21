package gen

import "strings"

// WorldType mirrors the subset of net.minecraft.src.WorldType used by
// GenLayer.initializeAllBiomeGenerators(...) in 1.6.4.
type WorldType int

const (
	WorldTypeDefault WorldType = iota
	WorldTypeLargeBiomes
	WorldTypeDefault11
)

func (t WorldType) String() string {
	switch t {
	case WorldTypeLargeBiomes:
		return "largeBiomes"
	case WorldTypeDefault11:
		return "default_1_1"
	default:
		return "default"
	}
}

// ParseWorldType translates level.dat generator settings to 1.6.4 world type.
//
// Translation reference:
// - net.minecraft.src.WorldType.parseWorldType(...)
// - net.minecraft.src.WorldType.getWorldTypeForGeneratorVersion(int)
func ParseWorldType(generatorName string, generatorVersion int32) WorldType {
	switch strings.ToLower(strings.TrimSpace(generatorName)) {
	case "largebiomes":
		return WorldTypeLargeBiomes
	case "default_1_1":
		return WorldTypeDefault11
	case "default":
		if generatorVersion == 0 {
			return WorldTypeDefault11
		}
		return WorldTypeDefault
	default:
		return WorldTypeDefault
	}
}
