//go:build cgo

package gui

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	faceDown = iota
	faceUp
	faceNorth
	faceSouth
	faceWest
	faceEast
)

type blockTextureDef struct {
	Up    string
	Down  string
	North string
	South string
	West  string
	East  string
	// SideOverlay mirrors Fancy Graphics grass/leaves side overlay passes.
	SideOverlay string

	TintR float32
	TintG float32
	TintB float32
}

func blockAll(name string) blockTextureDef {
	return blockTextureDef{
		Up:    name,
		Down:  name,
		North: name,
		South: name,
		West:  name,
		East:  name,
		TintR: 1,
		TintG: 1,
		TintB: 1,
	}
}

func blockTopBottomSide(top, bottom, side string) blockTextureDef {
	return blockTextureDef{
		Up:    top,
		Down:  bottom,
		North: side,
		South: side,
		West:  side,
		East:  side,
		TintR: 1,
		TintG: 1,
		TintB: 1,
	}
}

func blockWithFront(top, bottom, side, front string) blockTextureDef {
	return blockTextureDef{
		Up:    top,
		Down:  bottom,
		North: front,
		South: side,
		West:  side,
		East:  side,
		TintR: 1,
		TintG: 1,
		TintB: 1,
	}
}

func defaultBlockTextureDefs() map[int]blockTextureDef {
	return map[int]blockTextureDef{
		1: blockAll("stone.png"),
		2: {
			Up:          "grass_top.png",
			Down:        "dirt.png",
			North:       "grass_side.png",
			South:       "grass_side.png",
			West:        "grass_side.png",
			East:        "grass_side.png",
			SideOverlay: "grass_side_overlay.png",
			TintR:       1,
			TintG:       1,
			TintB:       1,
		},
		3:   blockAll("dirt.png"),
		4:   blockAll("cobblestone.png"),
		5:   blockAll("planks_oak.png"),
		6:   blockAll("sapling_oak.png"),
		7:   blockAll("bedrock.png"),
		8:   blockTopBottomSide("water_still.png", "water_still.png", "water_flow.png"),
		9:   blockTopBottomSide("water_still.png", "water_still.png", "water_flow.png"),
		10:  blockTopBottomSide("lava_still.png", "lava_still.png", "lava_flow.png"),
		11:  blockTopBottomSide("lava_still.png", "lava_still.png", "lava_flow.png"),
		12:  blockAll("sand.png"),
		13:  blockAll("gravel.png"),
		14:  blockAll("gold_ore.png"),
		15:  blockAll("iron_ore.png"),
		16:  blockAll("coal_ore.png"),
		17:  blockTopBottomSide("log_oak_top.png", "log_oak_top.png", "log_oak.png"),
		18:  blockAll("leaves_oak.png"),
		19:  blockAll("sponge.png"),
		20:  blockAll("glass.png"),
		21:  blockAll("lapis_ore.png"),
		22:  blockAll("lapis_block.png"),
		24:  blockTopBottomSide("sandstone_top.png", "sandstone_bottom.png", "sandstone_normal.png"),
		31:  blockAll("tallgrass.png"),
		32:  blockAll("deadbush.png"),
		35:  blockAll("wool_colored_white.png"),
		37:  blockAll("flower_dandelion.png"),
		38:  blockAll("flower_rose.png"),
		39:  blockAll("mushroom_brown.png"),
		40:  blockAll("mushroom_red.png"),
		41:  blockAll("gold_block.png"),
		42:  blockAll("iron_block.png"),
		43:  blockTopBottomSide("stone_slab_top.png", "stone_slab_top.png", "stone_slab_side.png"),
		44:  blockTopBottomSide("stone_slab_top.png", "stone_slab_top.png", "stone_slab_side.png"),
		45:  blockAll("brick.png"),
		46:  blockTopBottomSide("tnt_top.png", "tnt_bottom.png", "tnt_side.png"),
		47:  blockAll("bookshelf.png"),
		48:  blockAll("cobblestone_mossy.png"),
		49:  blockAll("obsidian.png"),
		52:  blockAll("mob_spawner.png"),
		54:  blockAll("planks_oak.png"), // temporary chest placeholder until TileEntityChest model render path
		56:  blockAll("diamond_ore.png"),
		57:  blockAll("diamond_block.png"),
		58:  blockWithFront("crafting_table_top.png", "planks_oak.png", "crafting_table_side.png", "crafting_table_front.png"),
		60:  blockTopBottomSide("farmland_dry.png", "dirt.png", "dirt.png"),
		61:  blockWithFront("furnace_top.png", "furnace_top.png", "furnace_side.png", "furnace_front_off.png"),
		62:  blockWithFront("furnace_top.png", "furnace_top.png", "furnace_side.png", "furnace_front_on.png"),
		73:  blockAll("redstone_ore.png"),
		78:  blockAll("snow.png"),
		79:  blockAll("ice.png"),
		80:  blockAll("snow.png"),
		81:  blockTopBottomSide("cactus_top.png", "cactus_bottom.png", "cactus_side.png"),
		82:  blockAll("clay.png"),
		83:  blockAll("reeds.png"),
		86:  blockWithFront("pumpkin_top.png", "pumpkin_top.png", "pumpkin_side.png", "pumpkin_face_off.png"),
		87:  blockAll("netherrack.png"),
		88:  blockAll("soul_sand.png"),
		89:  blockAll("glowstone.png"),
		99:  blockAll("mushroom_block_skin_brown.png"),
		100: blockAll("mushroom_block_skin_red.png"),
		98:  blockAll("stonebrick.png"),
		103: blockTopBottomSide("melon_top.png", "melon_top.png", "melon_side.png"),
		106: blockAll("vine.png"),
		110: blockTopBottomSide("mycelium_top.png", "dirt.png", "mycelium_side.png"),
		111: blockAll("waterlily.png"),
		112: blockAll("nether_brick.png"),
		121: blockAll("end_stone.png"),
		125: blockAll("planks_oak.png"),
		126: blockAll("planks_oak.png"),
		127: blockAll("cocoa_stage_2.png"),
		129: blockAll("emerald_ore.png"),
		133: blockAll("emerald_block.png"),
		152: blockAll("redstone_block.png"),
		155: blockTopBottomSide("quartz_block_top.png", "quartz_block_bottom.png", "quartz_block_side.png"),
	}
}

func (a *App) loadBlockTextures() error {
	a.blockTextureDefs = defaultBlockTextureDefs()
	a.blockTextures = make(map[string]*texture2D)

	needed := make(map[string]struct{})
	for _, def := range a.blockTextureDefs {
		for _, n := range [...]string{def.Up, def.Down, def.North, def.South, def.West, def.East, def.SideOverlay} {
			if n != "" {
				needed[n] = struct{}{}
			}
		}
	}
	// Translation reference:
	// - net.minecraft.src.BlockLeaves#setGraphicsLevel(boolean)
	// Both fancy and opaque leaves variants are used depending on graphics setting.
	for _, n := range [...]string{
		"log_oak.png",
		"log_oak_top.png",
		"log_spruce.png",
		"log_spruce_top.png",
		"log_birch.png",
		"log_birch_top.png",
		"log_jungle.png",
		"log_jungle_top.png",
		"planks_spruce.png",
		"planks_birch.png",
		"planks_jungle.png",
		"sapling_spruce.png",
		"sapling_birch.png",
		"sapling_jungle.png",
		"sandstone_carved.png",
		"sandstone_smooth.png",
		"fern.png",
		"wool_colored_orange.png",
		"wool_colored_magenta.png",
		"wool_colored_light_blue.png",
		"wool_colored_yellow.png",
		"wool_colored_lime.png",
		"wool_colored_pink.png",
		"wool_colored_gray.png",
		"wool_colored_silver.png",
		"wool_colored_cyan.png",
		"wool_colored_purple.png",
		"wool_colored_blue.png",
		"wool_colored_brown.png",
		"wool_colored_green.png",
		"wool_colored_red.png",
		"wool_colored_black.png",
		"leaves_oak.png",
		"leaves_spruce.png",
		"leaves_birch.png",
		"leaves_jungle.png",
		"leaves_oak_opaque.png",
		"leaves_spruce_opaque.png",
		"leaves_birch_opaque.png",
		"leaves_jungle_opaque.png",
	} {
		needed[n] = struct{}{}
	}

	if len(needed) == 0 {
		return nil
	}

	missing := make([]string, 0)
	for name := range needed {
		path := filepath.Join(a.assetsRoot, "textures", "blocks", name)
		tex, _, err := loadTexture2D(path, true)
		if err != nil {
			missing = append(missing, name)
			continue
		}
		a.blockTextures[name] = tex
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		totalMissing := len(missing)
		if len(missing) > 8 {
			missing = missing[:8]
		}
		return fmt.Errorf(
			"block texture load incomplete from %q: missing %d/%d (%s)",
			filepath.Join(a.assetsRoot, "textures", "blocks"),
			totalMissing,
			len(needed),
			strings.Join(missing, ", "),
		)
	}

	// Translation reference:
	// - net.minecraft.src.FoliageColorReloadListener
	// - net.minecraft.src.GrassColorReloadListener
	// Missing color maps are non-fatal; fallback tint path remains active.
	if err := a.loadBiomeColorMaps(); err != nil {
		fmt.Printf("gui warning: biome colormap load failed: %v\n", err)
	}
	return nil
}

func (a *App) loadBiomeColorMaps() error {
	grassMap, err := loadColorMapPNG(filepath.Join(a.assetsRoot, "textures", "colormap", "grass.png"))
	if err != nil {
		return err
	}
	foliageMap, err := loadColorMapPNG(filepath.Join(a.assetsRoot, "textures", "colormap", "foliage.png"))
	if err != nil {
		return err
	}
	a.grassColorMap = grassMap
	a.foliageColorMap = foliageMap
	return nil
}

func loadColorMapPNG(path string) ([]uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %q: %w", path, err)
	}

	b := img.Bounds()
	if b.Dx() != 256 || b.Dy() != 256 {
		return nil, fmt.Errorf("unexpected colormap size for %q: got %dx%d want 256x256", path, b.Dx(), b.Dy())
	}

	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)

	out := make([]uint32, 256*256)
	idx := 0
	for y := 0; y < 256; y++ {
		row := y * rgba.Stride
		for x := 0; x < 256; x++ {
			off := row + x*4
			r := uint32(rgba.Pix[off+0])
			g := uint32(rgba.Pix[off+1])
			b := uint32(rgba.Pix[off+2])
			out[idx] = (r << 16) | (g << 8) | b
			idx++
		}
	}
	return out, nil
}

func (a *App) blockTextureForFace(blockID int, face int) *texture2D {
	return a.blockTextureForFaceMeta(blockID, 0, face)
}

func (a *App) blockTextureForFaceMeta(blockID, blockMeta, face int) *texture2D {
	if name := a.blockTextureNameForFace(blockID, blockMeta, face); name != "" {
		return a.blockTextures[name]
	}
	return nil
}

func (a *App) blockTextureNameForFace(blockID, blockMeta, face int) string {
	def, ok := a.blockTextureDefs[blockID]
	if !ok {
		return ""
	}

	speciesByMeta := func(meta int) string {
		switch meta {
		case 1:
			return "spruce"
		case 2:
			return "birch"
		case 3:
			return "jungle"
		default:
			return "oak"
		}
	}

	// Translation reference:
	// - net.minecraft.src.BlockLeaves / BlockLeavesBase#setGraphicsLevel(boolean)
	// - metadata variants: oak/spruce/birch/jungle.
	if blockID == 18 {
		base := "leaves_oak"
		switch blockMeta & 3 {
		case 1:
			base = "leaves_spruce"
		case 2:
			base = "leaves_birch"
		case 3:
			base = "leaves_jungle"
		}
		if !a.fancyGraphics {
			base += "_opaque"
		}
		return base + ".png"
	}
	if blockID == 5 {
		// Translation reference:
		// - net.minecraft.src.BlockWood.getIcon(int side, int metadata)
		meta := blockMeta
		if meta < 0 || meta > 3 {
			meta = 0
		}
		return "planks_" + speciesByMeta(meta) + ".png"
	}
	if blockID == 6 {
		// Translation reference:
		// - net.minecraft.src.BlockSapling.getIcon(int side, int metadata)
		return "sapling_" + speciesByMeta(blockMeta&3) + ".png"
	}
	if blockID == 17 {
		// Translation reference:
		// - net.minecraft.src.BlockLog.getIcon(int side, int metadata)
		base := speciesByMeta(blockMeta & 3)
		sideTex := "log_" + base + ".png"
		topTex := "log_" + base + "_top.png"
		orientation := blockMeta & 12
		switch orientation {
		case 0:
			if face == faceUp || face == faceDown {
				return topTex
			}
		case 4:
			if face == faceEast || face == faceWest {
				return topTex
			}
		case 8:
			if face == faceNorth || face == faceSouth {
				return topTex
			}
		}
		return sideTex
	}
	if blockID == 24 {
		// Translation reference:
		// - net.minecraft.src.BlockSandStone.getIcon(int side, int metadata)
		variant := blockMeta
		if variant < 0 || variant > 2 {
			variant = 0
		}
		if face == faceUp || (face == faceDown && (variant == 1 || variant == 2)) {
			return "sandstone_top.png"
		}
		if face == faceDown {
			return "sandstone_bottom.png"
		}
		switch variant {
		case 1:
			return "sandstone_carved.png"
		case 2:
			return "sandstone_smooth.png"
		default:
			return "sandstone_normal.png"
		}
	}
	if blockID == 31 {
		// Translation reference:
		// - net.minecraft.src.BlockTallGrass.getIcon(int side, int metadata)
		switch blockMeta {
		case 1:
			return "tallgrass.png"
		case 2:
			return "fern.png"
		default:
			return "deadbush.png"
		}
	}
	if blockID == 35 {
		// Translation reference:
		// - net.minecraft.src.BlockColored.registerIcons(...)
		// - net.minecraft.src.BlockColored.getIcon(int side, int metadata)
		// Metadata order is white->black, texture registration uses reversed dye names.
		woolByMeta := [...]string{
			"white",
			"orange",
			"magenta",
			"light_blue",
			"yellow",
			"lime",
			"pink",
			"gray",
			"silver",
			"cyan",
			"purple",
			"blue",
			"brown",
			"green",
			"red",
			"black",
		}
		meta := blockMeta % len(woolByMeta)
		if meta < 0 {
			meta += len(woolByMeta)
		}
		return "wool_colored_" + woolByMeta[meta] + ".png"
	}
	if blockID == 43 || blockID == 44 {
		// Translation reference:
		// - net.minecraft.src.BlockStep.getIcon(int side, int metadata)
		// Metadata low 3 bits select the slab material.
		slabMeta := blockMeta & 7
		faceForType := face
		if blockID == 43 && (blockMeta&8) != 0 {
			// Seamless double slab uses top texture lookup for all sides.
			faceForType = faceUp
		}
		switch slabMeta {
		case 0: // stone slab top+side pair
			if faceForType == faceUp || faceForType == faceDown {
				return "stone_slab_top.png"
			}
			return "stone_slab_side.png"
		case 1: // sandstone slab (default sandstone variant only)
			return a.blockTextureNameForFace(24, 0, faceForType)
		case 2: // wood slab delegates to Block.planks side texture (oak)
			return "planks_oak.png"
		case 3:
			return "cobblestone.png"
		case 4:
			return "brick.png"
		case 5:
			return "stonebrick.png"
		case 6:
			return "nether_brick.png"
		case 7:
			if faceForType == faceUp || faceForType == faceDown {
				return "quartz_block_top.png"
			}
			return "quartz_block_side.png"
		default:
			return "stone_slab_top.png"
		}
	}
	if blockID == 125 || blockID == 126 {
		// Translation reference:
		// - net.minecraft.src.BlockWoodSlab.getIcon(int side, int metadata)
		// Delegates to planks icon with metadata & 7, and BlockWood clamps unknown values.
		meta := blockMeta & 7
		if meta < 0 || meta > 3 {
			meta = 0
		}
		return "planks_" + speciesByMeta(meta) + ".png"
	}

	switch face {
	case faceDown:
		return def.Down
	case faceUp:
		return def.Up
	case faceNorth:
		return def.North
	case faceSouth:
		return def.South
	case faceWest:
		return def.West
	case faceEast:
		return def.East
	}
	return ""
}

func (a *App) blockSideOverlayTexture(blockID int) *texture2D {
	def, ok := a.blockTextureDefs[blockID]
	if !ok || def.SideOverlay == "" {
		return nil
	}
	return a.blockTextures[def.SideOverlay]
}

func (a *App) blockFaceTint(blockID int, face int) (float32, float32, float32) {
	// Fallback path for call sites without world coordinates.
	return a.blockFaceTintAt(0, 0, 0, blockID, 0, face)
}

func (a *App) blockFaceTintAt(x, y, z, blockID, blockMeta, face int) (float32, float32, float32) {
	_ = y
	switch blockID {
	case 2: // grass block
		// Translation reference:
		// - net.minecraft.src.RenderBlocks#renderStandardBlockWithColorMultiplier(...)
		// Vanilla tints only grass top; side color comes from overlay pass.
		if face == faceUp {
			return a.biomeGrassTintAt(x, z)
		}
	case 31: // tall grass (plains biome tint approximation)
		// Translation reference:
		// - net.minecraft.src.BlockTallGrass#colorMultiplier(...)
		// meta=0 (deadbush variant) stays white; grass/fern use biome tint.
		if (blockMeta & 3) == 0 {
			return 1, 1, 1
		}
		return a.biomeGrassTintAt(x, z)
	case 18: // leaves (oak/spruce/birch variants by metadata)
		switch blockMeta & 3 {
		case 1: // spruce
			return rgbIntToFloat(6396257) // ColorizerFoliage.getFoliageColorPine()
		case 2: // birch
			return rgbIntToFloat(8431445) // ColorizerFoliage.getFoliageColorBirch()
		default: // oak/jungle
			return a.biomeFoliageTintAt(x, z)
		}
	case 106: // vine
		// Translation reference:
		// - net.minecraft.src.BlockVine#colorMultiplier(...)
		return a.biomeFoliageTintAt(x, z)
	case 8, 9: // water
		// Translation reference:
		// - net.minecraft.src.BlockFluid#colorMultiplier(...)
		// 1.6.4 water color is the 3x3 average of biome waterColorMultiplier.
		return a.biomeWaterTintAt(x, z)
	case 111: // lily pad
		return 32.0 / 255.0, 128.0 / 255.0, 48.0 / 255.0
	}

	if def, ok := a.blockTextureDefs[blockID]; ok {
		if def.TintR != 0 || def.TintG != 0 || def.TintB != 0 {
			return def.TintR, def.TintG, def.TintB
		}
	}
	return 1, 1, 1
}

func (a *App) blockSideOverlayTint(blockID int) (float32, float32, float32) {
	return a.blockSideOverlayTintAt(0, 0, 0, blockID)
}

func (a *App) blockSideOverlayTintAt(x, y, z, blockID int) (float32, float32, float32) {
	_ = y
	switch blockID {
	case 2:
		return a.biomeGrassTintAt(x, z)
	default:
		return 1, 1, 1
	}
}

func (a *App) biomeGrassTintAt(x, z int) (float32, float32, float32) {
	if c, ok := a.biomeColorAt(x, z, false); ok {
		return rgbIntToFloat(int(c))
	}
	// Plains-like fallback.
	return 0.49, 0.78, 0.31
}

func (a *App) biomeFoliageTintAt(x, z int) (float32, float32, float32) {
	if c, ok := a.biomeColorAt(x, z, true); ok {
		return rgbIntToFloat(int(c))
	}
	// ColorizerFoliage.getFoliageColorBasic()
	return rgbIntToFloat(4764952)
}

func (a *App) biomeWaterTintAt(x, z int) (float32, float32, float32) {
	if a.session == nil {
		return 1, 1, 1
	}

	sumR := 0
	sumG := 0
	sumB := 0
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			biomeID, ok := a.session.BiomeAt(x+dx, z+dz)
			if !ok {
				// Unknown biome payload defaults to plains in vanilla generation.
				biomeID = 1
			}
			c := biomeWaterColorMultiplier(biomeID)
			sumR += (c >> 16) & 255
			sumG += (c >> 8) & 255
			sumB += c & 255
		}
	}
	return float32(sumR) / (255.0 * 9.0), float32(sumG) / (255.0 * 9.0), float32(sumB) / (255.0 * 9.0)
}

func biomeWaterColorMultiplier(biomeID int) int {
	// Translation reference:
	// - net.minecraft.src.BiomeGenBase#waterColorMultiplier
	// - net.minecraft.src.BiomeGenSwamp constructor sets 0xE0FFAE
	switch biomeID {
	case 6: // swampland
		return 0xE0FFAE
	default:
		return 0xFFFFFF
	}
}

func (a *App) biomeColorAt(x, z int, foliage bool) (uint32, bool) {
	if a.session == nil {
		return 0, false
	}

	biomeID, ok := a.session.BiomeAt(x, z)
	if !ok {
		// Unknown biome payload defaults to plains in vanilla generation.
		biomeID = 1
	}

	temp, rain, swampBlend := biomeTemperatureRainfall(biomeID)
	var c uint32
	if foliage {
		var okColor bool
		c, okColor = sampleBiomeColorMap(a.foliageColorMap, temp, rain)
		if !okColor {
			return 0, false
		}
		if swampBlend {
			// Translation reference:
			// - net.minecraft.src.BiomeGenSwamp#getBiomeFoliageColor()
			c = ((c & 0xFEFEFE) + 0x4E0E4E) / 2
		}
		return c, true
	}

	var okColor bool
	c, okColor = sampleBiomeColorMap(a.grassColorMap, temp, rain)
	if !okColor {
		return 0, false
	}
	if swampBlend {
		// Translation reference:
		// - net.minecraft.src.BiomeGenSwamp#getBiomeGrassColor()
		c = ((c & 0xFEFEFE) + 0x4E0E4E) / 2
	}
	return c, true
}

func biomeTemperatureRainfall(biomeID int) (temperature float64, rainfall float64, swampBlend bool) {
	// Translation reference:
	// - net.minecraft.src.BiomeGenBase static biome declarations.
	switch biomeID {
	case 1: // plains
		return 0.8, 0.4, false
	case 2: // desert
		return 2.0, 0.0, false
	case 3: // extreme hills
		return 0.2, 0.3, false
	case 4: // forest
		return 0.7, 0.8, false
	case 5: // taiga
		return 0.05, 0.8, false
	case 6: // swampland
		return 0.8, 0.9, true
	case 8: // hell
		return 2.0, 0.0, false
	case 10: // frozen ocean
		return 0.0, 0.5, false
	case 11: // frozen river
		return 0.0, 0.5, false
	case 12: // ice plains
		return 0.0, 0.5, false
	case 13: // ice mountains
		return 0.0, 0.5, false
	case 14: // mushroom island
		return 0.9, 1.0, false
	case 15: // mushroom island shore
		return 0.9, 1.0, false
	case 16: // beach
		return 0.8, 0.4, false
	case 17: // desert hills
		return 2.0, 0.0, false
	case 18: // forest hills
		return 0.7, 0.8, false
	case 19: // taiga hills
		return 0.05, 0.8, false
	case 20: // extreme hills edge
		return 0.2, 0.3, false
	case 21: // jungle
		return 1.2, 0.9, false
	case 22: // jungle hills
		return 1.2, 0.9, false
	default:
		// ocean/river/sky and unknown IDs inherit BiomeGenBase defaults.
		return 0.5, 0.5, false
	}
}

func sampleBiomeColorMap(colorMap []uint32, temperature, rainfall float64) (uint32, bool) {
	if len(colorMap) < 256*256 {
		return 0, false
	}
	temperature = clamp01Float64(temperature)
	rainfall = clamp01Float64(rainfall)
	rainfall *= temperature

	x := int((1.0 - temperature) * 255.0)
	y := int((1.0 - rainfall) * 255.0)
	if x < 0 {
		x = 0
	} else if x > 255 {
		x = 255
	}
	if y < 0 {
		y = 0
	} else if y > 255 {
		y = 255
	}
	return colorMap[(y<<8)|x], true
}

func clamp01Float64(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func rgbIntToFloat(color int) (float32, float32, float32) {
	r := float32((color>>16)&0xFF) / 255.0
	g := float32((color>>8)&0xFF) / 255.0
	b := float32(color&0xFF) / 255.0
	return r, g, b
}
