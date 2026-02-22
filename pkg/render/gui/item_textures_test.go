//go:build cgo

package gui

import (
	"testing"

	"github.com/lulaide/gomc/pkg/nbt"
)

func TestItemTextureNameForIDSpecialCases(t *testing.T) {
	tests := []struct {
		id     int
		damage int
		want   string
	}{
		{id: 261, damage: 0, want: "bow_standby"},
		{id: 263, damage: 1, want: "charcoal"},
		{id: 346, damage: 0, want: "fishing_rod_uncast"},
		{id: 351, damage: 4, want: "dye_powder_blue"},
		{id: 351, damage: 15, want: "dye_powder_white"},
		{id: 373, damage: 0, want: "potion_bottle_drinkable"},
		{id: 373, damage: 0x4000, want: "potion_bottle_splash"},
		{id: 397, damage: 4, want: "skull_creeper"},
		{id: 264, damage: 0, want: "diamond"},
	}
	for _, tc := range tests {
		got := itemTextureNameForID(tc.id, tc.damage)
		if got != tc.want {
			t.Fatalf("itemTextureNameForID(%d,%d)=%q want=%q", tc.id, tc.damage, got, tc.want)
		}
	}
}

func TestHumanizeTextureToken(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{token: "iron_pickaxe", want: "Iron Pickaxe"},
		{token: "potion_bottle_splash", want: "Splash Potion"},
		{token: "items/record_wait", want: "Record Wait"},
	}
	for _, tc := range tests {
		got := humanizeTextureToken(tc.token)
		if got != tc.want {
			t.Fatalf("humanizeTextureToken(%q)=%q want=%q", tc.token, got, tc.want)
		}
	}
}

func TestItemLangKeyForStackSpecialCases(t *testing.T) {
	tests := []struct {
		id     int
		damage int
		want   string
	}{
		{id: 263, damage: 1, want: "item.charcoal.name"},
		{id: 351, damage: 12, want: "item.dyePowder.lightBlue.name"},
		{id: 397, damage: 4, want: "item.skull.creeper.name"},
		{id: 264, damage: 0, want: "item.diamond.name"},
	}
	for _, tc := range tests {
		got := itemLangKeyForStack(tc.id, tc.damage)
		if got != tc.want {
			t.Fatalf("itemLangKeyForStack(%d,%d)=%q want=%q", tc.id, tc.damage, got, tc.want)
		}
	}
}

func TestBlockLangKeyForID(t *testing.T) {
	if got := blockLangKeyForID(1); got != "tile.stone.name" {
		t.Fatalf("blockLangKeyForID(1)=%q want=%q", got, "tile.stone.name")
	}
	if got := blockLangKeyForID(999); got != "" {
		t.Fatalf("blockLangKeyForID(999)=%q want empty", got)
	}
}

func TestItemRequiresMultipleRenderPasses(t *testing.T) {
	if !itemRequiresMultipleRenderPasses(383) {
		t.Fatal("spawn egg should require multiple render passes")
	}
	if !itemRequiresMultipleRenderPasses(298) {
		t.Fatal("leather helmet should require multiple render passes")
	}
	if itemRequiresMultipleRenderPasses(264) {
		t.Fatal("diamond should not require multiple render passes")
	}
}

func TestItemTextureNameForRenderPass(t *testing.T) {
	tests := []struct {
		itemID int
		damage int
		pass   int
		want   string
	}{
		{itemID: 383, damage: 90, pass: 0, want: "spawn_egg"},
		{itemID: 383, damage: 90, pass: 1, want: "spawn_egg_overlay"},
		{itemID: 298, damage: 0, pass: 0, want: "leather_helmet"},
		{itemID: 298, damage: 0, pass: 1, want: "leather_helmet_overlay"},
		{itemID: 264, damage: 0, pass: 1, want: ""},
	}
	for _, tc := range tests {
		got := itemTextureNameForRenderPass(tc.itemID, tc.damage, tc.pass)
		if got != tc.want {
			t.Fatalf("itemTextureNameForRenderPass(%d,%d,%d)=%q want=%q", tc.itemID, tc.damage, tc.pass, got, tc.want)
		}
	}
}

func TestItemColorForRenderPass(t *testing.T) {
	if got := itemColorForRenderPass(383, 90, 0); got != 15771042 {
		t.Fatalf("spawn egg primary mismatch: got=%d want=%d", got, 15771042)
	}
	if got := itemColorForRenderPass(383, 90, 1); got != 14377823 {
		t.Fatalf("spawn egg secondary mismatch: got=%d want=%d", got, 14377823)
	}
	if got := itemColorForRenderPass(298, 0, 0); got != 10511680 {
		t.Fatalf("leather base color mismatch: got=%d want=%d", got, 10511680)
	}
	if got := itemColorForRenderPass(298, 0, 1); got != 0xFFFFFF {
		t.Fatalf("leather overlay color mismatch: got=%d want=%d", got, 0xFFFFFF)
	}
}

func TestItemColorForRenderPassWithLeatherNBTColor(t *testing.T) {
	stackTag := nbt.NewCompoundTag("")
	displayTag := nbt.NewCompoundTag("display")
	displayTag.SetInteger("color", 0x112233)
	stackTag.SetCompoundTag("display", displayTag)

	if got := itemColorForRenderPassWithTag(298, 0, 0, stackTag); got != 0x112233 {
		t.Fatalf("leather NBT color mismatch: got=0x%06x want=0x112233", got)
	}
	if got := itemColorForRenderPassWithTag(298, 0, 1, stackTag); got != 0xFFFFFF {
		t.Fatalf("leather overlay with NBT color mismatch: got=0x%06x want=0xFFFFFF", got)
	}
}

func TestItemDisplayNameSpawnEggIncludesEntityName(t *testing.T) {
	a := &App{
		langEN: map[string]string{
			"item.monsterPlacer.name": "Spawn",
			"entity.Pig.name":         "Pig",
		},
	}
	if got := a.itemDisplayName(383, 90); got != "Spawn Pig" {
		t.Fatalf("spawn egg display name mismatch: got=%q want=%q", got, "Spawn Pig")
	}
}

func TestItemDisplayNameSpawnEggFallsBackToBaseName(t *testing.T) {
	a := &App{
		langEN: map[string]string{
			"item.monsterPlacer.name": "Spawn",
		},
	}
	if got := a.itemDisplayName(383, 999); got != "Spawn" {
		t.Fatalf("spawn egg fallback name mismatch: got=%q want=%q", got, "Spawn")
	}
}
