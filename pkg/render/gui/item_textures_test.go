//go:build cgo

package gui

import "testing"

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
