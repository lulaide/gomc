//go:build cgo

package gui

import "testing"

func TestJavaStringHashCode(t *testing.T) {
	if got := javaStringHashCode("abc"); got != 96354 {
		t.Fatalf("hash mismatch: got=%d want=96354", got)
	}
}

func TestParseCreateWorldSeed(t *testing.T) {
	if got, ok := parseCreateWorldSeed("12345"); !ok || got != 12345 {
		t.Fatalf("numeric seed mismatch: got=(%d,%t)", got, ok)
	}
	if got, ok := parseCreateWorldSeed("abc"); !ok || got != 96354 {
		t.Fatalf("string seed mismatch: got=(%d,%t)", got, ok)
	}
	if got, ok := parseCreateWorldSeed("0"); !ok || got == 0 {
		t.Fatalf("zero seed should resolve to non-zero random seed: got=(%d,%t)", got, ok)
	}
}

func TestMakeUseableFolderName(t *testing.T) {
	if got := makeUseableFolderName("", func(string) bool { return false }); got != "World" {
		t.Fatalf("empty world name mismatch: got=%q want=%q", got, "World")
	}
	if got := makeUseableFolderName("CON", func(string) bool { return false }); got != "_CON_" {
		t.Fatalf("illegal DOS name mismatch: got=%q want=%q", got, "_CON_")
	}
	if got := makeUseableFolderName("a/b", func(string) bool { return false }); got != "a_b" {
		t.Fatalf("slash sanitization mismatch: got=%q want=%q", got, "a_b")
	}

	calls := 0
	got := makeUseableFolderName("World", func(name string) bool {
		calls++
		return name == "World" || name == "World-"
	})
	if got != "World--" {
		t.Fatalf("collision handling mismatch: got=%q want=%q", got, "World--")
	}
	if calls < 2 {
		t.Fatalf("collision callback should be called repeatedly, calls=%d", calls)
	}
}
