//go:build cgo

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
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
	id := int(itemID)
	if id <= 0 || size <= 0 {
		return false
	}
	tex := a.itemTextureForStack(id, int(itemDamage))
	if tex != nil {
		drawTexturedRect(tex, float32(x), float32(y), float32(size), float32(size), 0, 0, tex.Width, tex.Height)
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
	id := int(itemID)
	if id <= 0 {
		return ""
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
