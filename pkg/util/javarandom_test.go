package util

import "testing"

func TestJavaRandom_Seed1Vectors(t *testing.T) {
	r := NewJavaRandom(1)

	if got := r.NextIntUnbounded(); got != -1155869325 {
		t.Fatalf("NextIntUnbounded mismatch: got=%d want=%d", got, -1155869325)
	}
	if got := r.NextInt(100); got != 88 {
		t.Fatalf("NextInt(100) mismatch: got=%d want=%d", got, 88)
	}
	if got := r.NextLong(); got != 7564655870752979346 {
		t.Fatalf("NextLong mismatch: got=%d want=%d", got, int64(7564655870752979346))
	}
	if got := r.NextBoolean(); got != false {
		t.Fatalf("NextBoolean mismatch: got=%v want=false", got)
	}
	if got := r.NextFloat(); got != float32(0.036235332) {
		t.Fatalf("NextFloat mismatch: got=%0.9f want=%0.9f", got, float32(0.036235332))
	}
	if got := r.NextDouble(); got != 0.3327170559595112 {
		t.Fatalf("NextDouble mismatch: got=%0.16f want=%0.16f", got, 0.3327170559595112)
	}

	buf := make([]byte, 8)
	r.NextBytes(buf)
	want := []byte{0xD5, 0xD9, 0xBE, 0xF7, 0x68, 0x08, 0xF3, 0xB5}
	for i := range want {
		if buf[i] != want[i] {
			t.Fatalf("NextBytes mismatch at index %d: got=%d want=%d", i, int8(buf[i]), int8(want[i]))
		}
	}
}

func TestJavaRandom_Seed2Vectors(t *testing.T) {
	r := NewJavaRandom(123456789)

	if got := r.NextIntUnbounded(); got != -1442945365 {
		t.Fatalf("seed2 NextIntUnbounded mismatch: got=%d want=%d", got, -1442945365)
	}
	if got := r.NextInt(1024); got != 781 {
		t.Fatalf("seed2 NextInt(1024) mismatch: got=%d want=%d", got, 781)
	}
	if got := r.NextLong(); got != 8429272609719263920 {
		t.Fatalf("seed2 NextLong mismatch: got=%d want=%d", got, int64(8429272609719263920))
	}
	if got := r.NextFloat(); got != float32(0.39050645) {
		t.Fatalf("seed2 NextFloat mismatch: got=%0.8f want=%0.8f", got, float32(0.39050645))
	}
	if got := r.NextDouble(); got != 0.21659655710073045 {
		t.Fatalf("seed2 NextDouble mismatch: got=%0.17f want=%0.17f", got, 0.21659655710073045)
	}
}

func TestJavaRandom_NextIntBoundPanic(t *testing.T) {
	r := NewJavaRandom(1)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for non-positive bound")
		}
	}()
	_ = r.NextInt(0)
}
