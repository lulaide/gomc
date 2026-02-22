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
