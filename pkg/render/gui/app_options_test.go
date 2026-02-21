//go:build cgo

package gui

import "testing"

func findOptionButton(buttons []*guiButton, id int) *guiButton {
	for _, b := range buttons {
		if b != nil && b.ID == id {
			return b
		}
	}
	return nil
}

func TestOptionFOVLabelAndClamp(t *testing.T) {
	a := &App{}
	a.fovSetting = -1.0
	if got := a.currentFOVDegrees(); got != 70.0 {
		t.Fatalf("fov clamp min mismatch: got=%.2f want=70.00", got)
	}
	if got := a.optionFOVLabel(); got != "FOV: Normal" {
		t.Fatalf("fov normal label mismatch: got=%q", got)
	}

	a.fovSetting = 2.0
	if got := a.currentFOVDegrees(); got != 110.0 {
		t.Fatalf("fov clamp max mismatch: got=%.2f want=110.00", got)
	}
	if got := a.optionFOVLabel(); got != "FOV: 110" {
		t.Fatalf("fov numeric label mismatch: got=%q want=%q", got, "FOV: 110")
	}
}

func TestOptionButtonsFOVAndViewBobbingState(t *testing.T) {
	a := &App{
		guiW:           854,
		guiH:           480,
		renderDistance: 4,
		mouseSens:      0.02,
		fovSetting:     0.0,
		viewBobbing:    true,
	}
	a.initOptionButtons()
	a.updateOptionButtonsState()

	if b := findOptionButton(a.optionButtons, buttonIDOptionViewBobbing); b == nil || b.Label != "View Bobbing: ON" {
		got := "<nil>"
		if b != nil {
			got = b.Label
		}
		t.Fatalf("view bobbing label mismatch: got=%q want=%q", got, "View Bobbing: ON")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionRDMinus); b == nil || b.Enabled {
		t.Fatal("render distance minus should be disabled at minimum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVMinus); b == nil || b.Enabled {
		t.Fatal("fov minus should be disabled at minimum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensMinus); b == nil || b.Enabled {
		t.Fatal("sensitivity minus should be disabled at minimum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionRDPlus); b == nil || !b.Enabled {
		t.Fatal("render distance plus should be enabled above minimum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVPlus); b == nil || !b.Enabled {
		t.Fatal("fov plus should be enabled above minimum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensPlus); b == nil || !b.Enabled {
		t.Fatal("sensitivity plus should be enabled above minimum")
	}

	a.renderDistance = 96
	a.mouseSens = 0.50
	a.fovSetting = 1.0
	a.viewBobbing = false
	a.updateOptionButtonsState()

	if b := findOptionButton(a.optionButtons, buttonIDOptionViewBobbing); b == nil || b.Label != "View Bobbing: OFF" {
		got := "<nil>"
		if b != nil {
			got = b.Label
		}
		t.Fatalf("view bobbing off label mismatch: got=%q want=%q", got, "View Bobbing: OFF")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionRDPlus); b == nil || b.Enabled {
		t.Fatal("render distance plus should be disabled at maximum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVPlus); b == nil || b.Enabled {
		t.Fatal("fov plus should be disabled at maximum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensPlus); b == nil || b.Enabled {
		t.Fatal("sensitivity plus should be disabled at maximum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionRDMinus); b == nil || !b.Enabled {
		t.Fatal("render distance minus should be enabled at maximum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVMinus); b == nil || !b.Enabled {
		t.Fatal("fov minus should be enabled at maximum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensMinus); b == nil || !b.Enabled {
		t.Fatal("sensitivity minus should be enabled at maximum")
	}
}
