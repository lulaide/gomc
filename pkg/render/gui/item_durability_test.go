//go:build cgo

package gui

import "testing"

func TestItemMaxDurability(t *testing.T) {
	tests := []struct {
		id   int16
		want int16
		ok   bool
	}{
		{id: 259, want: 64, ok: true},   // flint_and_steel
		{id: 261, want: 384, ok: true},  // bow
		{id: 276, want: 1561, ok: true}, // diamond_sword
		{id: 294, want: 32, ok: true},   // gold_hoe
		{id: 310, want: 363, ok: true},  // diamond_helmet
		{id: 315, want: 112, ok: true},  // golden_chestplate
		{id: 359, want: 238, ok: true},  // shears
		{id: 398, want: 25, ok: true},   // carrot_on_a_stick
		{id: 264, want: 0, ok: false},   // diamond (material item, not damageable)
	}
	for _, tc := range tests {
		got, ok := itemMaxDurability(tc.id)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("itemMaxDurability(%d)=(%d,%t) want=(%d,%t)", tc.id, got, ok, tc.want, tc.ok)
		}
	}
}
