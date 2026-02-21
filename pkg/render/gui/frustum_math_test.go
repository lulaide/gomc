package gui

import "testing"

func TestFrustumIdentityContainsOrigin(t *testing.T) {
	proj := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	model := proj
	f := newViewFrustumFromGL(proj, model)
	if !f.valid {
		t.Fatal("expected frustum to be valid")
	}
	if !f.aabbVisible(-0.5, -0.5, -0.5, 0.5, 0.5, 0.5) {
		t.Fatal("origin box should be visible")
	}
}

func TestFrustumIdentityRejectsOutsideX(t *testing.T) {
	proj := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	model := proj
	f := newViewFrustumFromGL(proj, model)
	if !f.valid {
		t.Fatal("expected frustum to be valid")
	}
	if f.aabbVisible(2.0, -0.1, -0.1, 3.0, 0.1, 0.1) {
		t.Fatal("box outside +X should be culled")
	}
}

func TestFrustumInvalidWhenZeroMatrices(t *testing.T) {
	var proj [16]float32
	var model [16]float32
	f := newViewFrustumFromGL(proj, model)
	if f.valid {
		t.Fatal("expected invalid frustum for zero matrices")
	}
}
