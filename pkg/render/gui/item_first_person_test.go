//go:build cgo

package gui

import "testing"

func TestItemShouldRotateAroundWhenRendering(t *testing.T) {
	tests := []struct {
		itemID int16
		want   bool
	}{
		{itemID: 346, want: true},  // fishing rod
		{itemID: 398, want: true},  // carrot on a stick
		{itemID: 261, want: false}, // bow
		{itemID: 264, want: false}, // diamond
	}

	for _, tc := range tests {
		got := itemShouldRotateAroundWhenRendering(tc.itemID)
		if got != tc.want {
			t.Fatalf("itemShouldRotateAroundWhenRendering(%d)=%t want=%t", tc.itemID, got, tc.want)
		}
	}
}
