//go:build cgo

package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		renderDistance: 0,
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
	if b := findOptionButton(a.optionButtons, buttonIDOptionRDMinus); b == nil || !b.Enabled {
		t.Fatal("render distance minus should be enabled at far")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVMinus); b == nil || b.Enabled {
		t.Fatal("fov minus should be disabled at minimum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensMinus); b == nil || b.Enabled {
		t.Fatal("sensitivity minus should be disabled at minimum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionRDPlus); b == nil || b.Enabled {
		t.Fatal("render distance plus should be disabled at far")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVPlus); b == nil || !b.Enabled {
		t.Fatal("fov plus should be enabled above minimum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensPlus); b == nil || !b.Enabled {
		t.Fatal("sensitivity plus should be enabled above minimum")
	}

	a.renderDistance = 3
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
	if b := findOptionButton(a.optionButtons, buttonIDOptionRDPlus); b == nil || !b.Enabled {
		t.Fatal("render distance plus should be enabled at tiny")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVPlus); b == nil || b.Enabled {
		t.Fatal("fov plus should be disabled at maximum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensPlus); b == nil || b.Enabled {
		t.Fatal("sensitivity plus should be disabled at maximum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionRDMinus); b == nil || b.Enabled {
		t.Fatal("render distance minus should be disabled at tiny")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVMinus); b == nil || !b.Enabled {
		t.Fatal("fov minus should be enabled at maximum")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensMinus); b == nil || !b.Enabled {
		t.Fatal("sensitivity minus should be enabled at maximum")
	}
}

func TestRenderDistanceModeMappings(t *testing.T) {
	if got := renderDistanceModeToChunks(0); got != 16 {
		t.Fatalf("mode far chunks mismatch: got=%d want=16", got)
	}
	if got := renderDistanceModeToChunks(1); got != 8 {
		t.Fatalf("mode normal chunks mismatch: got=%d want=8", got)
	}
	if got := renderDistanceModeToChunks(2); got != 4 {
		t.Fatalf("mode short chunks mismatch: got=%d want=4", got)
	}
	if got := renderDistanceModeToChunks(3); got != 2 {
		t.Fatalf("mode tiny chunks mismatch: got=%d want=2", got)
	}
	if got := renderDistanceChunksToMode(10); got != 1 {
		t.Fatalf("chunks->mode mismatch for 10: got=%d want=1(normal)", got)
	}
	if got := renderDistanceChunksToMode(2); got != 3 {
		t.Fatalf("chunks->mode mismatch for 2: got=%d want=3(tiny)", got)
	}
}

func TestOptionsFileLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "options.txt")
	content := strings.Join([]string{
		"fov:0.25",
		"viewDistance:2",
		"bobView:false",
		"mouseSensitivity:0.30",
		"fpsLimit:2",
		"difficulty:3",
		"customKey:customValue",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp options failed: %v", err)
	}

	a := &App{
		optionsPath:        path,
		optionsKV:          make(map[string]string),
		mouseSens:          0.14,
		renderDistance:     1,
		viewBobbing:        true,
		limitFramerateMode: 1,
		optionDifficulty:   1,
	}
	a.loadOptionsFile()
	if a.fovSetting != 0.25 {
		t.Fatalf("loaded fov mismatch: got=%.2f want=0.25", a.fovSetting)
	}
	if a.renderDistance != 2 {
		t.Fatalf("loaded viewDistance mismatch: got=%d want=2", a.renderDistance)
	}
	if a.viewBobbing {
		t.Fatal("loaded bobView should be false")
	}
	if a.mouseSens != 0.30 {
		t.Fatalf("loaded sensitivity mismatch: got=%.2f want=0.30", a.mouseSens)
	}
	if a.limitFramerateMode != 2 {
		t.Fatalf("loaded fpsLimit mismatch: got=%d want=2", a.limitFramerateMode)
	}
	if a.optionDifficulty != 3 {
		t.Fatalf("loaded difficulty mismatch: got=%d want=3", a.optionDifficulty)
	}

	a.fovSetting = 1.0
	a.renderDistance = 0
	a.viewBobbing = true
	a.mouseSens = 0.5
	a.limitFramerateMode = 0
	a.optionDifficulty = 0
	a.saveOptionsFile()

	updatedBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved options failed: %v", err)
	}
	updated := string(updatedBytes)
	checks := []string{
		"fov:1.000000",
		"viewDistance:0",
		"bobView:true",
		"mouseSensitivity:0.500000",
		"fpsLimit:0",
		"difficulty:0",
		"customKey:customValue",
	}
	for _, want := range checks {
		if !strings.Contains(updated, want) {
			t.Fatalf("saved options missing %q\nsaved:\n%s", want, updated)
		}
	}
}
