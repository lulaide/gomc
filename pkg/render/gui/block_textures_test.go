//go:build cgo

package gui

import "testing"

func TestLeafTextureNameByMetaAndGraphics(t *testing.T) {
	a := &App{
		blockTextureDefs: defaultBlockTextureDefs(),
		fancyGraphics:    true,
	}

	fancyCases := []struct {
		meta int
		want string
	}{
		{meta: 0, want: "leaves_oak.png"},
		{meta: 1, want: "leaves_spruce.png"},
		{meta: 2, want: "leaves_birch.png"},
		{meta: 3, want: "leaves_jungle.png"},
		{meta: 7, want: "leaves_jungle.png"},
	}
	for _, tc := range fancyCases {
		if got := a.blockTextureNameForFace(18, tc.meta, faceUp); got != tc.want {
			t.Fatalf("fancy leaves meta=%d mismatch: got=%q want=%q", tc.meta, got, tc.want)
		}
	}

	a.fancyGraphics = false
	opaqueCases := []struct {
		meta int
		want string
	}{
		{meta: 0, want: "leaves_oak_opaque.png"},
		{meta: 1, want: "leaves_spruce_opaque.png"},
		{meta: 2, want: "leaves_birch_opaque.png"},
		{meta: 3, want: "leaves_jungle_opaque.png"},
	}
	for _, tc := range opaqueCases {
		if got := a.blockTextureNameForFace(18, tc.meta, faceNorth); got != tc.want {
			t.Fatalf("opaque leaves meta=%d mismatch: got=%q want=%q", tc.meta, got, tc.want)
		}
	}
}

func TestBlockTextureNameFallbackForUnknownBlock(t *testing.T) {
	a := &App{
		blockTextureDefs: defaultBlockTextureDefs(),
		fancyGraphics:    true,
	}
	if got := a.blockTextureNameForFace(9999, 0, faceUp); got != "" {
		t.Fatalf("unknown block texture name should be empty, got=%q", got)
	}
}

func TestLogTextureNameByMetaAndAxis(t *testing.T) {
	a := &App{
		blockTextureDefs: defaultBlockTextureDefs(),
		fancyGraphics:    true,
	}

	cases := []struct {
		meta int
		face int
		want string
	}{
		{meta: 0, face: faceUp, want: "log_oak_top.png"},
		{meta: 0, face: faceNorth, want: "log_oak.png"},
		{meta: 1, face: faceDown, want: "log_spruce_top.png"},
		{meta: 2, face: faceEast, want: "log_birch.png"},
		{meta: 4, face: faceEast, want: "log_oak_top.png"},
		{meta: 5, face: faceUp, want: "log_spruce.png"},
		{meta: 8, face: faceSouth, want: "log_oak_top.png"},
		{meta: 11, face: faceEast, want: "log_jungle.png"},
	}
	for _, tc := range cases {
		if got := a.blockTextureNameForFace(17, tc.meta, tc.face); got != tc.want {
			t.Fatalf("log meta=%d face=%d mismatch: got=%q want=%q", tc.meta, tc.face, got, tc.want)
		}
	}
}

func TestMetadataTextureNameMappings(t *testing.T) {
	a := &App{
		blockTextureDefs: defaultBlockTextureDefs(),
		fancyGraphics:    true,
	}

	cases := []struct {
		blockID int
		meta    int
		face    int
		want    string
	}{
		{blockID: 5, meta: 2, face: faceNorth, want: "planks_birch.png"},
		{blockID: 6, meta: 9, face: faceUp, want: "sapling_spruce.png"},
		{blockID: 24, meta: 0, face: faceDown, want: "sandstone_bottom.png"},
		{blockID: 24, meta: 1, face: faceDown, want: "sandstone_top.png"},
		{blockID: 24, meta: 1, face: faceEast, want: "sandstone_carved.png"},
		{blockID: 24, meta: 2, face: faceNorth, want: "sandstone_smooth.png"},
		{blockID: 60, meta: 0, face: faceUp, want: "farmland_dry.png"},
		{blockID: 60, meta: 7, face: faceUp, want: "farmland_wet.png"},
		{blockID: 60, meta: 3, face: faceEast, want: "dirt.png"},
		{blockID: 31, meta: 0, face: faceNorth, want: "deadbush.png"},
		{blockID: 31, meta: 1, face: faceNorth, want: "tallgrass.png"},
		{blockID: 31, meta: 2, face: faceUp, want: "fern.png"},
		{blockID: 35, meta: 0, face: faceNorth, want: "wool_colored_white.png"},
		{blockID: 35, meta: 15, face: faceUp, want: "wool_colored_black.png"},
		{blockID: 35, meta: 16, face: faceUp, want: "wool_colored_white.png"},
		{blockID: 97, meta: 0, face: faceWest, want: "stone.png"},
		{blockID: 97, meta: 1, face: faceWest, want: "cobblestone.png"},
		{blockID: 97, meta: 2, face: faceWest, want: "stonebrick.png"},
		{blockID: 98, meta: 0, face: faceWest, want: "stonebrick.png"},
		{blockID: 98, meta: 1, face: faceWest, want: "stonebrick_mossy.png"},
		{blockID: 98, meta: 2, face: faceWest, want: "stonebrick_cracked.png"},
		{blockID: 98, meta: 3, face: faceWest, want: "stonebrick_carved.png"},
		{blockID: 43, meta: 0, face: faceWest, want: "stone_slab_side.png"},
		{blockID: 43, meta: 8, face: faceWest, want: "stone_slab_top.png"},
		{blockID: 44, meta: 9, face: faceWest, want: "sandstone_normal.png"},
		{blockID: 44, meta: 7, face: faceDown, want: "quartz_block_top.png"},
		{blockID: 127, meta: 0, face: faceNorth, want: "cocoa_stage_0.png"},
		{blockID: 127, meta: 4, face: faceNorth, want: "cocoa_stage_1.png"},
		{blockID: 127, meta: 8, face: faceNorth, want: "cocoa_stage_2.png"},
		{blockID: 126, meta: 3, face: faceSouth, want: "planks_jungle.png"},
		{blockID: 126, meta: 12, face: faceSouth, want: "planks_oak.png"},
	}
	for _, tc := range cases {
		if got := a.blockTextureNameForFace(tc.blockID, tc.meta, tc.face); got != tc.want {
			t.Fatalf("block=%d meta=%d face=%d mismatch: got=%q want=%q", tc.blockID, tc.meta, tc.face, got, tc.want)
		}
	}
}

func TestTallGrassTintFollowsMetadata(t *testing.T) {
	a := &App{
		blockTextureDefs: defaultBlockTextureDefs(),
		fancyGraphics:    true,
	}

	r, g, b := a.blockFaceTintAt(0, 64, 0, 31, 0, faceNorth)
	if r != 1 || g != 1 || b != 1 {
		t.Fatalf("tallgrass meta=0 tint should be white, got=(%f,%f,%f)", r, g, b)
	}

	r, g, b = a.blockFaceTintAt(0, 64, 0, 31, 1, faceNorth)
	if r == 1 && g == 1 && b == 1 {
		t.Fatalf("tallgrass meta=1 tint should be biome-tinted, got=(%f,%f,%f)", r, g, b)
	}
}
