package gui

import (
	"math"
	"testing"
)

func almostEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestMovementDeltaFromYaw_VanillaSemantics(t *testing.T) {
	dx, dz := movementDeltaFromYaw(0, 1, 0, 1)
	if !almostEq(dx, 0) || !almostEq(dz, 1) {
		t.Fatalf("yaw0 forward: got (%.6f, %.6f), want (0, 1)", dx, dz)
	}

	dx, dz = movementDeltaFromYaw(180, 1, 0, 1)
	if !almostEq(dx, 0) || !almostEq(dz, -1) {
		t.Fatalf("yaw180 forward: got (%.6f, %.6f), want (0, -1)", dx, dz)
	}

	dx, dz = movementDeltaFromYaw(90, 1, 0, 1)
	if !almostEq(dx, -1) || !almostEq(dz, 0) {
		t.Fatalf("yaw90 forward: got (%.6f, %.6f), want (-1, 0)", dx, dz)
	}
}

func TestMovementDeltaFromYaw_StrafeLeftRight(t *testing.T) {
	// In vanilla key mapping: A (left) => strafe +1, D (right) => strafe -1.
	dxL, dzL := movementDeltaFromYaw(0, 0, 1, 1)
	dxR, dzR := movementDeltaFromYaw(0, 0, -1, 1)

	if !almostEq(dxL, 1) || !almostEq(dzL, 0) {
		t.Fatalf("strafe left at yaw0: got (%.6f, %.6f), want (1, 0)", dxL, dzL)
	}
	if !almostEq(dxR, -1) || !almostEq(dzR, 0) {
		t.Fatalf("strafe right at yaw0: got (%.6f, %.6f), want (-1, 0)", dxR, dzR)
	}
}

func TestLookDirectionFromYawPitch(t *testing.T) {
	dx, dy, dz := lookDirectionFromYawPitch(0, 0)
	if !almostEq(dx, 0) || !almostEq(dy, 0) || !almostEq(dz, 1) {
		t.Fatalf("yaw0 pitch0 look: got (%.6f, %.6f, %.6f), want (0, 0, 1)", dx, dy, dz)
	}
}
