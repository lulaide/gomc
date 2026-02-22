//go:build cgo

package gui

import "testing"

func TestPotionLiquidColorFromDamageDefault(t *testing.T) {
	// Translation reference:
	// - net.minecraft.src.PotionHelper#calcPotionLiquidColor(Collection)
	//   default color when effect list is empty.
	if got := potionLiquidColorFromDamage(0); got != 3694022 {
		t.Fatalf("potionLiquidColorFromDamage(0)=0x%06x want=0x%06x", got, 3694022)
	}
}

func TestParsePotionEffectExprBasic(t *testing.T) {
	if got := parsePotionEffectExpr("0", 0, 1, 1); got != 1 {
		t.Fatalf("parse expr bit-set mismatch: got=%d want=1", got)
	}
	if got := parsePotionEffectExpr("0", 0, 1, 0); got != 0 {
		t.Fatalf("parse expr bit-unset mismatch: got=%d want=0", got)
	}

	req := "0 & !1 & !2 & !3 & 0+6"
	if got := parsePotionEffectExpr(req, 0, len(req), (1<<0)|(1<<6)); got <= 0 {
		t.Fatalf("parse regen requirement should match for bits 0+6: got=%d", got)
	}
	if got := parsePotionEffectExpr(req, 0, len(req), (1<<0)|(1<<1)|(1<<6)); got > 0 {
		t.Fatalf("parse regen requirement should fail when bit1 set: got=%d", got)
	}
}
