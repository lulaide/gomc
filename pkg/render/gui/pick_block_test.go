//go:build cgo

package gui

import "testing"

func TestPickItemForBlockMappings(t *testing.T) {
	item, dmg, ok := pickItemForBlock(55, 0) // redstone wire
	if !ok || item != 331 || dmg != 0 {
		t.Fatalf("redstone wire pick mismatch: ok=%t item=%d dmg=%d", ok, item, dmg)
	}

	item, dmg, ok = pickItemForBlock(60, 7) // farmland
	if !ok || item != 3 || dmg != 0 {
		t.Fatalf("farmland pick mismatch: ok=%t item=%d dmg=%d", ok, item, dmg)
	}

	_, _, ok = pickItemForBlock(90, 0) // portal
	if ok {
		t.Fatal("portal should not be pickable")
	}

	item, dmg, ok = pickItemForBlock(43, 11) // double slab meta
	if !ok || item != 44 || dmg != 3 {
		t.Fatalf("double slab pick mismatch: ok=%t item=%d dmg=%d", ok, item, dmg)
	}
}
