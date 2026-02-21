//go:build cgo

package gui

import "testing"

func nearlyEqual(a, b float32) bool {
	const eps = 1e-6
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

func assertRect(t *testing.T, got uvRect, u0, v0, u1, v1 float32) {
	t.Helper()
	if !nearlyEqual(got.u0, u0) || !nearlyEqual(got.v0, v0) || !nearlyEqual(got.u1, u1) || !nearlyEqual(got.v1, v1) {
		t.Fatalf("uv mismatch got={u0:%f v0:%f u1:%f v1:%f} want={u0:%f v0:%f u1:%f v1:%f}",
			got.u0, got.v0, got.u1, got.v1, u0, v0, u1, v1)
	}
}

// Translation reference:
// - net.minecraft.src.ModelBox constructor (line75-line80 in MCP 1.6.4)
func TestCuboidUVFromTextureOffset_ModelBoxLayout8x8x8(t *testing.T) {
	tex := &texture2D{Width: 64, Height: 32}
	uv := cuboidUVFromTextureOffset(tex, 0, 0, 8, 8, 8)

	assertRect(t, uv.East, 16.0/64.0, 8.0/32.0, 24.0/64.0, 16.0/32.0)
	assertRect(t, uv.West, 0.0/64.0, 8.0/32.0, 8.0/64.0, 16.0/32.0)
	assertRect(t, uv.Down, 8.0/64.0, 0.0/32.0, 16.0/64.0, 8.0/32.0)
	assertRect(t, uv.Up, 16.0/64.0, 8.0/32.0, 24.0/64.0, 0.0/32.0)
	assertRect(t, uv.North, 8.0/64.0, 8.0/32.0, 16.0/64.0, 16.0/32.0)
	assertRect(t, uv.South, 24.0/64.0, 8.0/32.0, 32.0/64.0, 16.0/32.0)
}

func TestCuboidUVFromTextureOffset_ModelBoxLayoutArm4x12x4(t *testing.T) {
	tex := &texture2D{Width: 64, Height: 32}
	uv := cuboidUVFromTextureOffset(tex, 40, 16, 4, 12, 4)

	assertRect(t, uv.East, 48.0/64.0, 20.0/32.0, 52.0/64.0, 32.0/32.0)
	assertRect(t, uv.West, 40.0/64.0, 20.0/32.0, 44.0/64.0, 32.0/32.0)
	assertRect(t, uv.Down, 44.0/64.0, 16.0/32.0, 48.0/64.0, 20.0/32.0)
	assertRect(t, uv.Up, 48.0/64.0, 20.0/32.0, 52.0/64.0, 16.0/32.0)
	assertRect(t, uv.North, 44.0/64.0, 20.0/32.0, 48.0/64.0, 32.0/32.0)
	assertRect(t, uv.South, 52.0/64.0, 20.0/32.0, 56.0/64.0, 32.0/32.0)
}
