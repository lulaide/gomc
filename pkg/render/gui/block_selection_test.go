//go:build cgo

package gui

import "testing"

func TestBlockSelectionRelativeAABBs_DeadBushBounds(t *testing.T) {
	boxes := blockSelectionRelativeAABBs(32, 0)
	if len(boxes) != 1 {
		t.Fatalf("dead bush box count mismatch: got=%d want=1", len(boxes))
	}
	bb := boxes[0]
	if bb.minX != 0.1 || bb.minY != 0.0 || bb.minZ != 0.1 || bb.maxX != 0.9 || bb.maxY != 0.8 || bb.maxZ != 0.9 {
		t.Fatalf(
			"dead bush bounds mismatch: got=(%.4f,%.4f,%.4f)->(%.4f,%.4f,%.4f)",
			bb.minX, bb.minY, bb.minZ, bb.maxX, bb.maxY, bb.maxZ,
		)
	}
}

func TestBlockSelectionRelativeAABBs_LiquidNotSelectable(t *testing.T) {
	if boxes := blockSelectionRelativeAABBs(8, 0); len(boxes) != 0 {
		t.Fatalf("water should not be selectable: got boxes=%d", len(boxes))
	}
	if boxes := blockSelectionRelativeAABBs(10, 0); len(boxes) != 0 {
		t.Fatalf("lava should not be selectable: got boxes=%d", len(boxes))
	}
}

func TestBlockSelectionRelativeAABBs_CropsHeightByAge(t *testing.T) {
	boxes := blockSelectionRelativeAABBs(59, 7)
	if len(boxes) != 1 {
		t.Fatalf("crop box count mismatch: got=%d want=1", len(boxes))
	}
	if boxes[0].maxY != 1.0 {
		t.Fatalf("mature crop maxY mismatch: got=%.4f want=1.0000", boxes[0].maxY)
	}

	boxes = blockSelectionRelativeAABBs(59, 0)
	if len(boxes) != 1 {
		t.Fatalf("crop box count mismatch at age0: got=%d want=1", len(boxes))
	}
	if boxes[0].maxY != 0.125 {
		t.Fatalf("age0 crop maxY mismatch: got=%.4f want=0.1250", boxes[0].maxY)
	}
}

func TestVineSelectionRelativeAABB_MetaWestPlate(t *testing.T) {
	bb := vineSelectionRelativeAABB(2)
	if bb.minX != 0.0 || bb.maxX != 0.0625 {
		t.Fatalf("vine west plate X bounds mismatch: got=(%.4f,%.4f) want=(0.0000,0.0625)", bb.minX, bb.maxX)
	}
	if bb.minY != 0.0 || bb.maxY != 1.0 || bb.minZ != 0.0 || bb.maxZ != 1.0 {
		t.Fatalf(
			"vine west plate YZ bounds mismatch: got=(%.4f,%.4f,%.4f,%.4f)",
			bb.minY, bb.maxY, bb.minZ, bb.maxZ,
		)
	}
}
