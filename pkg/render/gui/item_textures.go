//go:build cgo

package gui

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/lulaide/gomc/pkg/nbt"
)

var dyeColorTokens = [16]string{
	"black",
	"red",
	"green",
	"brown",
	"blue",
	"purple",
	"cyan",
	"silver",
	"gray",
	"pink",
	"lime",
	"yellow",
	"light_blue",
	"magenta",
	"orange",
	"white",
}

var dyeLangTokens = [16]string{
	"black",
	"red",
	"green",
	"brown",
	"blue",
	"purple",
	"cyan",
	"silver",
	"gray",
	"pink",
	"lime",
	"yellow",
	"lightBlue",
	"magenta",
	"orange",
	"white",
}

type itemTexturePass struct {
	tex *texture2D
	r   float32
	g   float32
	b   float32
}

type spawnEggColorPair struct {
	primary   int
	secondary int
}

var spawnEggColorsByEntityID = map[int]spawnEggColorPair{
	// Translation reference:
	// - net.minecraft.src.EntityList static addMapping(..., primary, secondary)
	50:  {primary: 894731, secondary: 0},         // Creeper
	51:  {primary: 12698049, secondary: 4802889}, // Skeleton
	52:  {primary: 3419431, secondary: 11013646}, // Spider
	54:  {primary: 44975, secondary: 7969893},    // Zombie
	55:  {primary: 5349438, secondary: 8306542},  // Slime
	56:  {primary: 16382457, secondary: 12369084},
	57:  {primary: 15373203, secondary: 5009705},
	58:  {primary: 1447446, secondary: 0},
	59:  {primary: 803406, secondary: 11013646},
	60:  {primary: 7237230, secondary: 3158064},
	61:  {primary: 16167425, secondary: 16775294},
	62:  {primary: 3407872, secondary: 16579584},
	65:  {primary: 4996656, secondary: 986895},    // Bat
	66:  {primary: 3407872, secondary: 5349438},   // Witch
	90:  {primary: 15771042, secondary: 14377823}, // Pig
	91:  {primary: 15198183, secondary: 16758197}, // Sheep
	92:  {primary: 4470310, secondary: 10592673},  // Cow
	93:  {primary: 10592673, secondary: 16711680}, // Chicken
	94:  {primary: 2243405, secondary: 7375001},   // Squid
	95:  {primary: 14144467, secondary: 13545366}, // Wolf
	96:  {primary: 10489616, secondary: 12040119}, // Mooshroom
	98:  {primary: 15720061, secondary: 5653556},  // Ocelot
	100: {primary: 12623485, secondary: 15656192}, // Horse
	120: {primary: 5651507, secondary: 12422002},  // Villager
}

var entityStringIDByID = map[int]string{
	// Translation reference:
	// - net.minecraft.src.EntityList static addMapping(Class,String,int)
	50:  "Creeper",
	51:  "Skeleton",
	52:  "Spider",
	54:  "Zombie",
	55:  "Slime",
	56:  "Ghast",
	57:  "PigZombie",
	58:  "Enderman",
	59:  "CaveSpider",
	60:  "Silverfish",
	61:  "Blaze",
	62:  "LavaSlime",
	65:  "Bat",
	66:  "Witch",
	90:  "Pig",
	91:  "Sheep",
	92:  "Cow",
	93:  "Chicken",
	94:  "Squid",
	95:  "Wolf",
	96:  "MushroomCow",
	98:  "Ozelot",
	100: "EntityHorse",
	120: "Villager",
}

func (a *App) loadItemTextures() error {
	a.itemTextures = make(map[string]*texture2D)
	root := filepath.Join(a.assetsRoot, "textures", "items")
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}

	var loadErrs []string
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			loadErrs = append(loadErrs, err.Error())
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".png") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			loadErrs = append(loadErrs, err.Error())
			return nil
		}
		key := normalizeTextureToken(rel)
		tex, _, err := loadTexture2DWithFlip(path, true, false)
		if err != nil {
			loadErrs = append(loadErrs, fmt.Sprintf("%s: %v", key, err))
			return nil
		}
		a.itemTextures[key] = tex
		base := strings.ToLower(filepath.Base(key))
		if base != key {
			if _, exists := a.itemTextures[base]; !exists {
				a.itemTextures[base] = tex
			}
		}
		return nil
	})
	if walkErr != nil {
		loadErrs = append(loadErrs, walkErr.Error())
	}
	if len(loadErrs) == 0 {
		return nil
	}
	sort.Strings(loadErrs)
	if len(loadErrs) > 8 {
		loadErrs = loadErrs[:8]
	}
	return fmt.Errorf("item texture load partial: %s", strings.Join(loadErrs, "; "))
}

func normalizeTextureToken(token string) string {
	token = strings.TrimSpace(strings.ToLower(filepath.ToSlash(token)))
	token = strings.TrimSuffix(token, ".png")
	return token
}

func (a *App) itemTextureByName(name string) *texture2D {
	if len(a.itemTextures) == 0 {
		return nil
	}
	key := normalizeTextureToken(name)
	if key == "" {
		return nil
	}
	if tex := a.itemTextures[key]; tex != nil {
		return tex
	}
	base := strings.ToLower(filepath.Base(key))
	if tex := a.itemTextures[base]; tex != nil {
		return tex
	}
	return nil
}

// Translation references:
// - net.minecraft.src.ItemArmor#requiresMultipleRenderPasses()
// - net.minecraft.src.ItemMonsterPlacer#requiresMultipleRenderPasses()
func itemRequiresMultipleRenderPasses(itemID int) bool {
	switch itemID {
	case 298, 299, 300, 301, 373, 383, 402:
		return true
	default:
		return false
	}
}

func itemTextureNameForRenderPass(itemID, itemDamage, pass int) string {
	switch itemID {
	case 298, 299, 300, 301:
		base := itemTextureNameForID(itemID, itemDamage)
		if pass == 1 {
			return base + "_overlay"
		}
		return base
	case 383:
		if pass == 1 {
			return "spawn_egg_overlay"
		}
		return "spawn_egg"
	case 373:
		// Translation reference:
		// - net.minecraft.src.ItemPotion#getIconFromDamageForRenderPass(int,int)
		// pass 0: colored overlay, pass 1: bottle icon.
		if pass == 0 {
			return "potion_overlay"
		}
		if (itemDamage & 0x4000) != 0 {
			return "potion_bottle_splash"
		}
		return "potion_bottle_drinkable"
	case 402:
		if pass > 0 {
			return "fireworks_charge_overlay"
		}
		return "fireworks_charge"
	default:
		if pass == 0 {
			return itemTextureNameForID(itemID, itemDamage)
		}
		return ""
	}
}

// Translation references:
// - net.minecraft.src.ItemArmor#getColorFromItemStack(ItemStack,int)
// - net.minecraft.src.ItemMonsterPlacer#getColorFromItemStack(ItemStack,int)
func itemColorForRenderPass(itemID, itemDamage, pass int) int {
	return itemColorForRenderPassWithTag(itemID, itemDamage, pass, nil)
}

func intFromNBTNumericTag(tag nbt.Tag) (int, bool) {
	switch t := tag.(type) {
	case *nbt.ByteTag:
		return int(t.Data), true
	case *nbt.ShortTag:
		return int(t.Data), true
	case *nbt.IntTag:
		return int(t.Data), true
	case *nbt.LongTag:
		return int(t.Data), true
	default:
		return 0, false
	}
}

// Translation references:
// - net.minecraft.src.ItemArmor#getColor(ItemStack)
// - net.minecraft.src.NBTTagCompound#getInteger(String)
func leatherArmorColorFromStackTag(stackTag *nbt.CompoundTag) int {
	const defaultLeatherColor = 10511680
	if stackTag == nil {
		return defaultLeatherColor
	}
	displayTag, ok := stackTag.GetTag("display").(*nbt.CompoundTag)
	if !ok || displayTag == nil {
		return defaultLeatherColor
	}
	colorTag := displayTag.GetTag("color")
	if colorTag == nil {
		return defaultLeatherColor
	}
	if color, ok := intFromNBTNumericTag(colorTag); ok {
		return color & 0xFFFFFF
	}
	return defaultLeatherColor
}

func itemColorForRenderPassWithTag(itemID, itemDamage, pass int, itemTag *nbt.CompoundTag) int {
	switch itemID {
	case 298, 299, 300, 301:
		if pass > 0 {
			return 0xFFFFFF
		}
		return leatherArmorColorFromStackTag(itemTag)
	case 383:
		pair, ok := spawnEggColorsByEntityID[itemDamage]
		if !ok {
			return 0xFFFFFF
		}
		if pass == 0 {
			return pair.primary
		}
		return pair.secondary
	case 373:
		// Translation reference:
		// - net.minecraft.src.ItemPotion#getColorFromItemStack(ItemStack,int)
		if pass > 0 {
			return 0xFFFFFF
		}
		return potionLiquidColorFromStack(itemDamage, itemTag)
	case 402:
		// Translation reference:
		// - net.minecraft.src.ItemFireworkCharge#getColorFromItemStack(ItemStack,int)
		if pass != 1 {
			return 0xFFFFFF
		}
		return fireworkChargeColorFromStackTag(itemTag)
	default:
		return 0xFFFFFF
	}
}

// Translation references:
// - net.minecraft.src.ItemFireworkCharge#func_92108_a(ItemStack,String)
// - net.minecraft.src.ItemFireworkCharge#getColorFromItemStack(ItemStack,int)
func fireworkChargeColorFromStackTag(itemTag *nbt.CompoundTag) int {
	const defaultFireworkColor = 9079434
	if itemTag == nil {
		return defaultFireworkColor
	}
	explosionTag, ok := itemTag.GetTag("Explosion").(*nbt.CompoundTag)
	if !ok || explosionTag == nil {
		return defaultFireworkColor
	}
	colorsTag, ok := explosionTag.GetTag("Colors").(*nbt.IntArrayTag)
	if !ok || colorsTag == nil || len(colorsTag.Ints) == 0 {
		return defaultFireworkColor
	}
	if len(colorsTag.Ints) == 1 {
		return int(colorsTag.Ints[0]) & 0xFFFFFF
	}

	var sumR, sumG, sumB int
	for _, c := range colorsTag.Ints {
		color := int(c)
		sumR += (color & 0xFF0000) >> 16
		sumG += (color & 0x00FF00) >> 8
		sumB += color & 0x0000FF
	}
	n := len(colorsTag.Ints)
	return ((sumR / n) << 16) | ((sumG / n) << 8) | (sumB / n)
}

func (a *App) itemTexturePasses(itemID, itemDamage int16) []itemTexturePass {
	return a.itemTexturePassesWithTag(itemID, itemDamage, nil)
}

func (a *App) itemTexturePassesWithTag(itemID, itemDamage int16, itemTag *nbt.CompoundTag) []itemTexturePass {
	id := int(itemID)
	damage := int(itemDamage)
	if id <= 0 {
		return nil
	}

	if id <= 255 {
		if tex := a.blockTextureForFaceMeta(id, damage, faceUp); tex != nil {
			return []itemTexturePass{{tex: tex, r: 1, g: 1, b: 1}}
		}
		if tex := a.blockTextureForFaceMeta(id, damage, faceNorth); tex != nil {
			return []itemTexturePass{{tex: tex, r: 1, g: 1, b: 1}}
		}
		return nil
	}

	if itemRequiresMultipleRenderPasses(id) {
		out := make([]itemTexturePass, 0, 2)
		for pass := 0; pass <= 1; pass++ {
			name := itemTextureNameForRenderPass(id, damage, pass)
			if name == "" {
				continue
			}
			tex := a.itemTextureByName(name)
			if tex == nil {
				continue
			}
			a.prepareDynamicItemTexture(id, tex)
			r, g, b := rgbIntToFloat(itemColorForRenderPassWithTag(id, damage, pass, itemTag))
			out = append(out, itemTexturePass{tex: tex, r: r, g: g, b: b})
		}
		if len(out) > 0 {
			return out
		}
	}

	if tex := a.itemTextureForStack(id, damage); tex != nil {
		a.prepareDynamicItemTexture(id, tex)
		r, g, b := rgbIntToFloat(itemColorForRenderPassWithTag(id, damage, 0, itemTag))
		return []itemTexturePass{{tex: tex, r: r, g: g, b: b}}
	}

	return nil
}

// Translation references:
// - net.minecraft.src.TextureClock#updateAnimation()
// - net.minecraft.src.TextureCompass#updateCompass(...)
func (a *App) prepareDynamicItemTexture(itemID int, tex *texture2D) {
	if tex == nil || len(tex.animatedFrames) <= 1 {
		return
	}
	switch itemID {
	case 347: // clock
		a.updateClockTextureFrame(tex)
	case 345: // compass
		a.updateCompassTextureFrame(tex)
	}
}

func (a *App) updateClockTextureFrame(tex *texture2D) {
	if tex == nil || len(tex.animatedFrames) <= 1 {
		return
	}
	updateTick := time.Now().UnixMilli() / 50
	if updateTick == a.clockUpdateTick {
		return
	}
	a.clockUpdateTick = updateTick

	target := 0.0
	if a.session != nil {
		snap := a.session.Snapshot()
		target = rand.Float64()
		if snap.Dimension == 0 {
			target = float64(celestialAngle(snap.WorldTime, 1.0))
		}
	}

	d := target - a.clockAngle
	for d < -0.5 {
		d++
	}
	for d >= 0.5 {
		d--
	}
	if d < -1.0 {
		d = -1.0
	}
	if d > 1.0 {
		d = 1.0
	}
	a.clockDelta += d * 0.1
	a.clockDelta *= 0.8
	a.clockAngle += a.clockDelta

	frames := len(tex.animatedFrames)
	frame := int((a.clockAngle+1.0)*float64(frames)) % frames
	if frame < 0 {
		frame = (frame + frames) % frames
	}
	tex.setAnimatedFrame(frame)
}

func (a *App) updateCompassTextureFrame(tex *texture2D) {
	if tex == nil || len(tex.animatedFrames) <= 1 {
		return
	}
	updateTick := time.Now().UnixMilli() / 50
	if updateTick == a.compassUpdateTick {
		return
	}
	a.compassUpdateTick = updateTick

	target := 0.0
	if a.session != nil {
		snap := a.session.Snapshot()
		target = rand.Float64() * math.Pi * 2.0
		if snap.Dimension == 0 {
			dx := float64(snap.SpawnX) - snap.PlayerX
			dz := float64(snap.SpawnZ) - snap.PlayerZ
			yaw := math.Mod(float64(snap.PlayerYaw), 360.0)
			target = -((yaw-90.0)*math.Pi/180.0 - math.Atan2(dz, dx))
		}
	}

	d := target - a.compassAngle
	for d < -math.Pi {
		d += math.Pi * 2.0
	}
	for d >= math.Pi {
		d -= math.Pi * 2.0
	}
	if d < -1.0 {
		d = -1.0
	}
	if d > 1.0 {
		d = 1.0
	}
	a.compassDelta += d * 0.1
	a.compassDelta *= 0.8
	a.compassAngle += a.compassDelta

	frames := len(tex.animatedFrames)
	frame := int((a.compassAngle/(math.Pi*2.0)+1.0)*float64(frames)) % frames
	if frame < 0 {
		frame = (frame + frames) % frames
	}
	tex.setAnimatedFrame(frame)
}

// Translation references:
// - net.minecraft.src.Item (setTextureName registrations)
// - net.minecraft.src.ItemDye (damage -> dye color)
// - net.minecraft.src.ItemSkull (damage -> skull variant)
func itemTextureNameForID(itemID, itemDamage int) string {
	switch itemID {
	case 261:
		return "bow_standby"
	case 263:
		if itemDamage&1 == 1 {
			return "charcoal"
		}
	case 346:
		return "fishing_rod_uncast"
	case 351:
		return "dye_powder_" + dyeColorTokens[itemDamage&15]
	case 373:
		if itemDamage&0x4000 != 0 {
			return "potion_bottle_splash"
		}
		return "potion_bottle_drinkable"
	case 397:
		switch itemDamage & 255 {
		case 1:
			return "skull_wither"
		case 2:
			return "skull_zombie"
		case 3:
			return "skull_steve"
		case 4:
			return "skull_creeper"
		default:
			return "skull_skeleton"
		}
	}

	def, ok := itemTextureDefs[itemID]
	if !ok {
		return ""
	}
	return def.TextureName
}

func (a *App) itemTextureForStack(itemID, itemDamage int) *texture2D {
	if itemID <= 0 {
		return nil
	}
	if itemID <= 255 {
		if tex := a.blockTextureForFaceMeta(itemID, itemDamage, faceUp); tex != nil {
			return tex
		}
		if tex := a.blockTextureForFaceMeta(itemID, itemDamage, faceNorth); tex != nil {
			return tex
		}
	}

	if name := itemTextureNameForID(itemID, itemDamage); name != "" {
		if tex := a.itemTextureByName(name); tex != nil {
			return tex
		}
	}

	if def, ok := itemTextureDefs[itemID]; ok {
		return a.itemTextureByName(def.TextureName)
	}
	return nil
}

func (a *App) drawItemStackIcon(itemID, itemDamage int16, x, y, size int) bool {
	return a.drawItemStackIconWithTag(itemID, itemDamage, nil, x, y, size)
}

func (a *App) drawItemStackIconWithTag(itemID, itemDamage int16, itemTag *nbt.CompoundTag, x, y, size int) bool {
	id := int(itemID)
	if id <= 0 || size <= 0 {
		return false
	}

	passes := a.itemTexturePassesWithTag(itemID, itemDamage, itemTag)
	if len(passes) > 0 {
		for _, pass := range passes {
			if pass.tex == nil {
				continue
			}
			gl.Color4f(pass.r, pass.g, pass.b, 1)
			drawTexturedRect(pass.tex, float32(x), float32(y), float32(size), float32(size), 0, 0, pass.tex.Width, pass.tex.Height)
		}
		gl.Color4f(1, 1, 1, 1)
		return true
	}

	r, g, b := colorForBlock(id)
	cr := int(r*255.0 + 0.5)
	cg := int(g*255.0 + 0.5)
	cb := int(b*255.0 + 0.5)
	color := (0xB0 << 24) | (cr << 16) | (cg << 8) | cb
	drawSolidRect(x+1, y+1, x+size-1, y+size-1, color)
	return false
}

func (a *App) itemDisplayName(itemID, itemDamage int16) string {
	return a.itemDisplayNameWithTag(itemID, itemDamage, nil)
}

func (a *App) itemDisplayNameWithTag(itemID, itemDamage int16, itemTag *nbt.CompoundTag) string {
	id := int(itemID)
	if id <= 0 {
		return ""
	}

	if id == 373 {
		damage := int(itemDamage)
		// Translation reference:
		// - net.minecraft.src.ItemPotion#getItemDisplayName(ItemStack)
		if damage == 0 {
			if localized, ok := a.langEN["item.emptyPotion.name"]; ok && strings.TrimSpace(localized) != "" {
				return strings.TrimSpace(localized)
			}
			return "Water Bottle"
		}

		splashPrefix := ""
		if (damage & 16384) != 0 {
			if localized, ok := a.langEN["potion.prefix.grenade"]; ok && strings.TrimSpace(localized) != "" {
				splashPrefix = strings.TrimSpace(localized) + " "
			}
		}

		effects := potionEffectsFromItemStack(damage, itemTag, false)
		if len(effects) > 0 {
			if key, ok := potionNameLangKeyByID[effects[0].potionID]; ok && key != "" {
				postfixKey := key + ".postfix"
				if localized, ok := a.langEN[postfixKey]; ok && strings.TrimSpace(localized) != "" {
					return strings.TrimSpace(splashPrefix + strings.TrimSpace(localized))
				}
			}
		}

		prefix := ""
		if prefixKey := potionPrefixLangKeyFromDamage(damage); prefixKey != "" {
			if localized, ok := a.langEN[prefixKey]; ok && strings.TrimSpace(localized) != "" {
				prefix = strings.TrimSpace(localized)
			}
		}

		base := ""
		if key := itemLangKeyForStack(id, damage); key != "" {
			if localized, ok := a.langEN[key]; ok && strings.TrimSpace(localized) != "" {
				base = strings.TrimSpace(localized)
			}
		}
		if base == "" {
			base = "Potion"
		}
		if prefix != "" {
			return strings.TrimSpace(prefix + " " + base)
		}
		return base
	}

	if id == 383 {
		base := ""
		if key := itemLangKeyForStack(id, int(itemDamage)); key != "" {
			if localized, ok := a.langEN[key]; ok && localized != "" {
				base = localized
			}
		}
		if base == "" {
			base = "Spawn Egg"
		}
		if entityKey, ok := entityStringIDByID[int(itemDamage)]; ok && entityKey != "" {
			if localized, ok := a.langEN["entity."+entityKey+".name"]; ok && localized != "" {
				return strings.TrimSpace(base + " " + localized)
			}
		}
		return base
	}

	if key := blockLangKeyForID(id); key != "" {
		if localized, ok := a.langEN[key]; ok && localized != "" {
			return localized
		}
	}
	if key := itemLangKeyForStack(id, int(itemDamage)); key != "" {
		if localized, ok := a.langEN[key]; ok && localized != "" {
			return localized
		}
	}

	var token string
	if id <= 255 {
		token = normalizeTextureToken(a.blockTextureNameForFace(id, int(itemDamage), faceUp))
		if token == "" {
			token = normalizeTextureToken(a.blockTextureNameForFace(id, int(itemDamage), faceNorth))
		}
	} else {
		token = normalizeTextureToken(itemTextureNameForID(id, int(itemDamage)))
	}
	if token == "" {
		return fmt.Sprintf("ID %d", id)
	}
	return humanizeTextureToken(token)
}

func humanizeTextureToken(token string) string {
	token = normalizeTextureToken(token)
	token = strings.TrimPrefix(token, "blocks/")
	token = strings.TrimPrefix(token, "items/")
	token = filepath.Base(token)

	switch token {
	case "bow_standby":
		return "Bow"
	case "fishing_rod_uncast":
		return "Fishing Rod"
	case "potion_bottle_drinkable":
		return "Potion"
	case "potion_bottle_splash":
		return "Splash Potion"
	}

	words := strings.Fields(strings.ReplaceAll(token, "_", " "))
	if len(words) == 0 {
		return ""
	}
	for i := range words {
		rs := []rune(words[i])
		if len(rs) == 0 {
			continue
		}
		rs[0] = unicode.ToUpper(rs[0])
		for j := 1; j < len(rs); j++ {
			rs[j] = unicode.ToLower(rs[j])
		}
		words[i] = string(rs)
	}
	return strings.Join(words, " ")
}

func blockLangKeyForID(blockID int) string {
	if blockID <= 0 {
		return ""
	}
	if unlocalized, ok := blockUnlocalizedNames[blockID]; ok && unlocalized != "" {
		return "tile." + unlocalized + ".name"
	}
	return ""
}

func itemLangKeyForStack(itemID, itemDamage int) string {
	if itemID <= 0 {
		return ""
	}
	switch itemID {
	case 263:
		if itemDamage&1 == 1 {
			return "item.charcoal.name"
		}
	case 351:
		return "item.dyePowder." + dyeLangTokens[itemDamage&15] + ".name"
	case 397:
		switch itemDamage & 255 {
		case 1:
			return "item.skull.wither.name"
		case 2:
			return "item.skull.zombie.name"
		case 3:
			return "item.skull.char.name"
		case 4:
			return "item.skull.creeper.name"
		default:
			return "item.skull.skeleton.name"
		}
	}

	if def, ok := itemTextureDefs[itemID]; ok && def.UnlocalizedName != "" {
		return "item." + def.UnlocalizedName + ".name"
	}
	return ""
}
