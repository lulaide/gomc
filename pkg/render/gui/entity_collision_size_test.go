//go:build cgo

package gui

import (
	"testing"

	netclient "github.com/lulaide/gomc/pkg/network/client"
)

func TestEntityCollisionSizeDroppedItemMatchesVanillaAABB(t *testing.T) {
	width, height := entityCollisionSize(netclient.EntitySnapshot{Type: 2})
	if width != 0.25 || height != 0.25 {
		t.Fatalf("dropped item collision size mismatch: got=(%.2f,%.2f) want=(0.25,0.25)", width, height)
	}
}
