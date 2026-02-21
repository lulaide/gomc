package gen

import "testing"

func TestGenLayerBiomeSourceDeterministic(t *testing.T) {
	a := NewGenLayerBiomeSource(12345, WorldTypeDefault)
	b := NewGenLayerBiomeSource(12345, WorldTypeDefault)

	ba := a.LoadBlockGeneratorData(nil, -64, 96, 48, 48)
	bb := b.LoadBlockGeneratorData(nil, -64, 96, 48, 48)
	if len(ba) != len(bb) {
		t.Fatalf("biome buffer length mismatch: %d != %d", len(ba), len(bb))
	}
	for i := range ba {
		if ba[i].ID != bb[i].ID {
			t.Fatalf("determinism mismatch at %d: %d != %d", i, ba[i].ID, bb[i].ID)
		}
	}
}

func TestGenLayerBiomeSourceProducesVariety(t *testing.T) {
	s := NewGenLayerBiomeSource(424242, WorldTypeDefault)
	biomes := s.LoadBlockGeneratorData(nil, -256, -256, 128, 128)
	seen := make(map[byte]struct{})
	for _, b := range biomes {
		seen[b.ID] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("expected multiple biome IDs, got %d", len(seen))
	}
}

func TestGenLayerBiomeSourceDefault11HasNoJungle(t *testing.T) {
	s := NewGenLayerBiomeSource(99887766, WorldTypeDefault11)
	biomes := s.LoadBlockGeneratorData(nil, -512, -512, 256, 256)
	for i, b := range biomes {
		if b.ID == BiomeIDJungle || b.ID == BiomeIDJungleHills {
			t.Fatalf("unexpected jungle biome for default_1_1 at idx=%d id=%d", i, b.ID)
		}
	}
}
