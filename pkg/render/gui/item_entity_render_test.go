//go:build cgo

package gui

import "testing"

func TestDroppedItemRenderCopiesThresholds(t *testing.T) {
	tests := []struct {
		count int8
		want  int
	}{
		{count: 1, want: 1},
		{count: 2, want: 2},
		{count: 6, want: 3},
		{count: 21, want: 4},
	}
	for _, tc := range tests {
		got := droppedItemRenderCopies(tc.count)
		if got != tc.want {
			t.Fatalf("droppedItemRenderCopies(%d)=%d want=%d", tc.count, got, tc.want)
		}
	}
}
