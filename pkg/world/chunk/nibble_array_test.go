package chunk

import "testing"

func TestNibbleArraySetGet(t *testing.T) {
	n := NewNibbleArray(4096, 4)

	n.Set(1, 2, 3, 0xA)
	n.Set(2, 2, 3, 0x5)

	if got := n.Get(1, 2, 3); got != 0xA {
		t.Fatalf("get mismatch at (1,2,3): got=%d want=10", got)
	}
	if got := n.Get(2, 2, 3); got != 0x5 {
		t.Fatalf("get mismatch at (2,2,3): got=%d want=5", got)
	}
}
