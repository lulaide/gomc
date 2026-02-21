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

func TestBlockSelectionRelativeAABBs_Farmland(t *testing.T) {
	boxes := blockSelectionRelativeAABBs(60, 0)
	if len(boxes) != 1 {
		t.Fatalf("farmland box count mismatch: got=%d want=1", len(boxes))
	}
	if boxes[0].maxY != 0.9375 {
		t.Fatalf("farmland maxY mismatch: got=%.4f want=0.9375", boxes[0].maxY)
	}
}

func TestBlockSelectionRelativeAABBs_HalfSlabTopBottom(t *testing.T) {
	bottom := blockSelectionRelativeAABBs(44, 0)
	if len(bottom) != 1 {
		t.Fatalf("bottom slab box count mismatch: got=%d want=1", len(bottom))
	}
	if bottom[0].minY != 0.0 || bottom[0].maxY != 0.5 {
		t.Fatalf("bottom slab Y bounds mismatch: got=(%.4f,%.4f) want=(0.0000,0.5000)", bottom[0].minY, bottom[0].maxY)
	}

	top := blockSelectionRelativeAABBs(44, 8)
	if len(top) != 1 {
		t.Fatalf("top slab box count mismatch: got=%d want=1", len(top))
	}
	if top[0].minY != 0.5 || top[0].maxY != 1.0 {
		t.Fatalf("top slab Y bounds mismatch: got=(%.4f,%.4f) want=(0.5000,1.0000)", top[0].minY, top[0].maxY)
	}
}

func TestBlockSelectionRelativeAABBs_SnowLayer(t *testing.T) {
	boxes := blockSelectionRelativeAABBs(78, 0)
	if len(boxes) != 1 {
		t.Fatalf("snow layer box count mismatch: got=%d want=1", len(boxes))
	}
	if boxes[0].maxY != 0.125 {
		t.Fatalf("snow meta0 maxY mismatch: got=%.4f want=0.1250", boxes[0].maxY)
	}

	boxes = blockSelectionRelativeAABBs(78, 7)
	if len(boxes) != 1 {
		t.Fatalf("snow layer box count mismatch at meta7: got=%d want=1", len(boxes))
	}
	if boxes[0].maxY != 1.0 {
		t.Fatalf("snow meta7 maxY mismatch: got=%.4f want=1.0000", boxes[0].maxY)
	}
}

func TestBlockSelectionRelativeAABBs_CactusInset(t *testing.T) {
	boxes := blockSelectionRelativeAABBs(81, 0)
	if len(boxes) != 1 {
		t.Fatalf("cactus box count mismatch: got=%d want=1", len(boxes))
	}
	bb := boxes[0]
	if bb.minX != 0.0625 || bb.minZ != 0.0625 || bb.maxX != 0.9375 || bb.maxZ != 0.9375 || bb.maxY != 1.0 {
		t.Fatalf(
			"cactus bounds mismatch: got=(%.4f,%.4f,%.4f)->(%.4f,%.4f,%.4f)",
			bb.minX, bb.minY, bb.minZ, bb.maxX, bb.maxY, bb.maxZ,
		)
	}
}

func TestBlockCollisionRelativeAABBs_LiquidAndRailNoCollision(t *testing.T) {
	if boxes := blockCollisionRelativeAABBs(8, 0); len(boxes) != 0 {
		t.Fatalf("water should have no collision: got boxes=%d", len(boxes))
	}
	if boxes := blockCollisionRelativeAABBs(66, 0); len(boxes) != 0 {
		t.Fatalf("rail should have no collision: got boxes=%d", len(boxes))
	}
}

func TestBlockCollisionRelativeAABBs_SoulSandAndCactus(t *testing.T) {
	soul := blockCollisionRelativeAABBs(88, 0)
	if len(soul) != 1 {
		t.Fatalf("soul sand collision box count mismatch: got=%d want=1", len(soul))
	}
	if soul[0].maxY != 0.875 {
		t.Fatalf("soul sand maxY mismatch: got=%.4f want=0.8750", soul[0].maxY)
	}

	cactus := blockCollisionRelativeAABBs(81, 0)
	if len(cactus) != 1 {
		t.Fatalf("cactus collision box count mismatch: got=%d want=1", len(cactus))
	}
	if cactus[0].maxY != 0.9375 || cactus[0].minX != 0.0625 || cactus[0].maxX != 0.9375 {
		t.Fatalf(
			"cactus collision bounds mismatch: got=(%.4f,%.4f,%.4f)->(%.4f,%.4f,%.4f)",
			cactus[0].minX, cactus[0].minY, cactus[0].minZ, cactus[0].maxX, cactus[0].maxY, cactus[0].maxZ,
		)
	}
}

func TestBlockCollisionRelativeAABBs_SnowAndSlab(t *testing.T) {
	snow0 := blockCollisionRelativeAABBs(78, 0)
	if len(snow0) != 0 {
		t.Fatalf("snow meta0 should have no collision: got boxes=%d", len(snow0))
	}

	snow7 := blockCollisionRelativeAABBs(78, 7)
	if len(snow7) != 1 || snow7[0].maxY != 0.875 {
		t.Fatalf("snow meta7 collision mismatch: boxes=%d maxY=%.4f want=1,0.8750", len(snow7), snow7[0].maxY)
	}

	topSlab := blockCollisionRelativeAABBs(44, 8)
	if len(topSlab) != 1 || topSlab[0].minY != 0.5 || topSlab[0].maxY != 1.0 {
		t.Fatalf("top slab collision mismatch: boxes=%d y=(%.4f,%.4f)", len(topSlab), topSlab[0].minY, topSlab[0].maxY)
	}
}
