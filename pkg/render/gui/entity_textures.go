//go:build cgo

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (a *App) loadEntityTextures() error {
	a.entityTextures = make(map[string]*texture2D)
	root := filepath.Join(a.assetsRoot, "textures", "entity")
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
		key := filepath.ToSlash(rel)
		// Keep top-left UV convention for entity skin pixel coordinates.
		// (Matches ModelBox/TexturedQuad offsets used in this renderer path.)
		tex, _, err := loadTexture2DWithFlip(path, true, false)
		if err != nil {
			loadErrs = append(loadErrs, fmt.Sprintf("%s: %v", key, err))
			return nil
		}
		a.entityTextures[key] = tex
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
	return fmt.Errorf("entity texture load partial: %s", strings.Join(loadErrs, "; "))
}

func (a *App) entityTextureForType(typeID int8) *texture2D {
	if len(a.entityTextures) == 0 {
		return nil
	}

	switch typeID {
	case 0:
		return a.entityTextureByPath("steve.png")
	case 50:
		return a.entityTextureByPath("creeper/creeper.png")
	case 51:
		return a.entityTextureByPath("skeleton/skeleton.png")
	case 52:
		return a.entityTextureByPath("spider/spider.png")
	case 54:
		return a.entityTextureByPath("zombie/zombie.png")
	case 55:
		return a.entityTextureByPath("slime/slime.png")
	case 57:
		return a.entityTextureByPath("zombie_pigman.png")
	case 58:
		return a.entityTextureByPath("enderman/enderman.png")
	case 59:
		return a.entityTextureByPath("spider/cave_spider.png")
	case 60:
		return a.entityTextureByPath("silverfish.png")
	case 61:
		return a.entityTextureByPath("blaze.png")
	case 62:
		return a.entityTextureByPath("slime/magmacube.png")
	case 65:
		return a.entityTextureByPath("bat.png")
	case 66:
		return a.entityTextureByPath("witch.png")
	case 90:
		return a.entityTextureByPath("pig/pig.png")
	case 91:
		return a.entityTextureByPath("sheep/sheep.png")
	case 92:
		return a.entityTextureByPath("cow/cow.png")
	case 93:
		return a.entityTextureByPath("chicken.png")
	case 94:
		return a.entityTextureByPath("squid.png")
	case 95:
		return a.entityTextureByPath("wolf/wolf.png")
	case 96:
		return a.entityTextureByPath("cow/mooshroom.png")
	case 97:
		return a.entityTextureByPath("snowman.png")
	case 98:
		return a.entityTextureByPath("cat/ocelot.png")
	case 99:
		return a.entityTextureByPath("iron_golem.png")
	case 100:
		// Variant metadata is not tracked yet; pick a stable vanilla horse skin.
		return a.entityTextureByPath("horse/horse_brown.png")
	case 120:
		return a.entityTextureByPath("villager/villager.png")
	default:
		return nil
	}
}

func (a *App) entityTextureByPath(path string) *texture2D {
	if len(a.entityTextures) == 0 {
		return nil
	}
	key := filepath.ToSlash(path)
	if tex := a.entityTextures[key]; tex != nil {
		return tex
	}
	key = strings.ToLower(key)
	for k, tex := range a.entityTextures {
		if strings.ToLower(k) == key {
			return tex
		}
	}
	return nil
}
