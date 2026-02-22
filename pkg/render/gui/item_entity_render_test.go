//go:build cgo

package gui

import (
	"testing"

	"github.com/lulaide/gomc/pkg/util"
)

func TestDroppedItemRenderCopiesThresholds(t *testing.T) {
	tests := []struct {
		count int8
		want  int
	}{
		{count: 1, want: 1},
		{count: 2, want: 2},
		{count: 6, want: 3},
		{count: 21, want: 4},
		{count: 41, want: 5},
	}
	for _, tc := range tests {
		got := droppedItemRenderCopies(tc.count)
		if got != tc.want {
			t.Fatalf("droppedItemRenderCopies(%d)=%d want=%d", tc.count, got, tc.want)
		}
	}
}

func TestDroppedItemRenderFancySpriteCopiesThresholds(t *testing.T) {
	tests := []struct {
		count int8
		want  int
	}{
		{count: 1, want: 1},
		{count: 2, want: 2},
		{count: 15, want: 2},
		{count: 16, want: 3},
		{count: 31, want: 3},
		{count: 32, want: 4},
		{count: 64, want: 4},
	}
	for _, tc := range tests {
		got := droppedItemRenderFancySpriteCopies(tc.count)
		if got != tc.want {
			t.Fatalf("droppedItemRenderFancySpriteCopies(%d)=%d want=%d", tc.count, got, tc.want)
		}
	}
}

func TestDroppedItemRenderRandomOffsetDeterministic(t *testing.T) {
	r1 := util.NewJavaRandom(187)
	r2 := util.NewJavaRandom(187)

	for i := 0; i < 5; i++ {
		x1, y1, z1 := droppedItemRenderRandomOffset(r1, 0.3)
		x2, y2, z2 := droppedItemRenderRandomOffset(r2, 0.3)
		if x1 != x2 || y1 != y2 || z1 != z2 {
			t.Fatalf("offset mismatch at %d: (%f,%f,%f) vs (%f,%f,%f)", i, x1, y1, z1, x2, y2, z2)
		}
	}
}
