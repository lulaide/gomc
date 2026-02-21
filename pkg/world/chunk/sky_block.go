package chunk

// EnumSkyBlock translates net.minecraft.src.EnumSkyBlock.
type EnumSkyBlock int

const (
	EnumSkyBlockSky EnumSkyBlock = iota
	EnumSkyBlockBlock
)

// DefaultLightValue mirrors EnumSkyBlock.defaultLightValue.
func (e EnumSkyBlock) DefaultLightValue() int {
	switch e {
	case EnumSkyBlockSky:
		return 15
	case EnumSkyBlockBlock:
		return 0
	default:
		return 0
	}
}
