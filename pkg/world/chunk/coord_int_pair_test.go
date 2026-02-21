package chunk

import "testing"

func TestCoordIntPairHashAndCenter(t *testing.T) {
	p := NewCoordIntPair(2, -3)

	if got := p.GetCenterXPos(); got != 40 {
		t.Fatalf("center x mismatch: got=%d want=40", got)
	}
	if got := p.GetCenterZPos(); got != -40 {
		t.Fatalf("center z mismatch: got=%d want=-40", got)
	}

	wantHash := int32(ChunkXZToInt(2, -3)) ^ int32(ChunkXZToInt(2, -3)>>32)
	if got := p.HashCode(); got != wantHash {
		t.Fatalf("hash mismatch: got=%d want=%d", got, wantHash)
	}
}

func TestCoordIntPairChunkXZToInt(t *testing.T) {
	v := ChunkXZToInt(-1, 1)
	lo := int32(v)
	hi := int32(v >> 32)
	if lo != -1 || hi != 1 {
		t.Fatalf("packed mismatch: lo=%d hi=%d", lo, hi)
	}
}
