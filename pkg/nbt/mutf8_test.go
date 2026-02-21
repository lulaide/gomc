package nbt

import "testing"

func TestModifiedUTF8RoundTrip(t *testing.T) {
	input := "A\x00B𐐷中文"
	encoded, err := encodeModifiedUTF8(input)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := decodeModifiedUTF8(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded != input {
		t.Fatalf("roundtrip mismatch: got=%q want=%q", decoded, input)
	}
}
