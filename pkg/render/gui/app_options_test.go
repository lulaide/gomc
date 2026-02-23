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
		guiW:               854,
		guiH:               480,
		renderDistance:     0,
		mouseSens:          0.0,
		fovSetting:         0.0,
		invertMouse:        false,
		viewBobbing:        true,
		fancyGraphics:      true,
		cloudsEnabled:      true,
		guiScaleMode:       0,
		limitFramerateMode: 1,
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
	if b := findOptionButton(a.optionButtons, buttonIDOptionVideo); b == nil || !b.Enabled || b.Label != "Max Framerate: Balanced" {
		got := "<nil>"
		enabled := false
		if b != nil {
			got = b.Label
			enabled = b.Enabled
		}
		t.Fatalf("framerate button mismatch: got label=%q enabled=%t", got, enabled)
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionLanguage); b == nil || !b.Enabled || b.Label != "GUI Scale: Auto" {
		got := "<nil>"
		enabled := false
		if b != nil {
			got = b.Label
			enabled = b.Enabled
		}
		t.Fatalf("gui scale button mismatch: got label=%q enabled=%t", got, enabled)
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionControls); b == nil || !b.Enabled || b.Label != "Invert Mouse: OFF" {
		got := "<nil>"
		enabled := false
		if b != nil {
			got = b.Label
			enabled = b.Enabled
		}
		t.Fatalf("invert mouse button mismatch: got label=%q enabled=%t", got, enabled)
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionMusic); b == nil || !b.Enabled || b.Label != "Graphics: Fancy" {
		got := "<nil>"
		enabled := false
		if b != nil {
			got = b.Label
			enabled = b.Enabled
		}
		t.Fatalf("graphics button mismatch: got label=%q enabled=%t", got, enabled)
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSnooper); b == nil || !b.Enabled || b.Label != "Clouds: ON" {
		got := "<nil>"
		enabled := false
		if b != nil {
			got = b.Label
			enabled = b.Enabled
		}
		t.Fatalf("clouds button mismatch: got label=%q enabled=%t", got, enabled)
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
	a.mouseSens = 1.0
	a.fovSetting = 1.0
	a.invertMouse = true
	a.viewBobbing = false
	a.fancyGraphics = false
	a.cloudsEnabled = false
	a.guiScaleMode = 3
	a.updateOptionButtonsState()

	if b := findOptionButton(a.optionButtons, buttonIDOptionViewBobbing); b == nil || b.Label != "View Bobbing: OFF" {
		got := "<nil>"
		if b != nil {
			got = b.Label
		}
		t.Fatalf("view bobbing off label mismatch: got=%q want=%q", got, "View Bobbing: OFF")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionLanguage); b == nil || b.Label != "GUI Scale: Large" {
		got := "<nil>"
		if b != nil {
			got = b.Label
		}
		t.Fatalf("gui scale label mismatch at max: got=%q want=%q", got, "GUI Scale: Large")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionControls); b == nil || b.Label != "Invert Mouse: ON" {
		got := "<nil>"
		if b != nil {
			got = b.Label
		}
		t.Fatalf("invert mouse label mismatch at on: got=%q want=%q", got, "Invert Mouse: ON")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionMusic); b == nil || b.Label != "Graphics: Fast" {
		got := "<nil>"
		if b != nil {
			got = b.Label
		}
		t.Fatalf("graphics label mismatch at fast: got=%q want=%q", got, "Graphics: Fast")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSnooper); b == nil || b.Label != "Clouds: OFF" {
		got := "<nil>"
		if b != nil {
			got = b.Label
		}
		t.Fatalf("clouds label mismatch at off: got=%q want=%q", got, "Clouds: OFF")
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

func TestPauseOptionVideoOpensVideoScreenAndCyclesFramerate(t *testing.T) {
	a := &App{
		mainMenu:           false,
		pauseScreen:        pauseScreenOptions,
		optionButtons:      []*guiButton{},
		videoButtons:       []*guiButton{},
		limitFramerateMode: 1,
		optionsKV:          make(map[string]string),
		optionsPath:        filepath.Join(t.TempDir(), "options.txt"),
	}
	a.handlePauseOptionButton(buttonIDOptionVideo)
	if a.pauseScreen != pauseScreenVideo {
		t.Fatalf("pause option video should open video screen: got=%d want=%d", a.pauseScreen, pauseScreenVideo)
	}
	a.handlePauseVideoButton(buttonIDVideoFPS)
	if a.limitFramerateMode != 2 {
		t.Fatalf("first framerate cycle mismatch: got=%d want=2", a.limitFramerateMode)
	}
	a.handlePauseVideoButton(buttonIDVideoFPS)
	if a.limitFramerateMode != 0 {
		t.Fatalf("second framerate cycle mismatch: got=%d want=0", a.limitFramerateMode)
	}
}

func TestPauseVideoGUIScaleCycles(t *testing.T) {
	a := &App{
		mainMenu:     false,
		pauseScreen:  pauseScreenVideo,
		width:        1280,
		height:       720,
		guiScaleMode: 0,
		videoButtons: []*guiButton{},
		optionsKV:    make(map[string]string),
		optionsPath:  filepath.Join(t.TempDir(), "options.txt"),
	}
	a.handlePauseVideoButton(buttonIDVideoGUIScale)
	if a.guiScaleMode != 1 {
		t.Fatalf("first gui scale cycle mismatch: got=%d want=1", a.guiScaleMode)
	}
	a.handlePauseVideoButton(buttonIDVideoGUIScale)
	a.handlePauseVideoButton(buttonIDVideoGUIScale)
	a.handlePauseVideoButton(buttonIDVideoGUIScale)
	if a.guiScaleMode != 0 {
		t.Fatalf("gui scale should wrap to auto: got=%d want=0", a.guiScaleMode)
	}
}

func TestPauseOptionControlsOpensControlsScreenAndTogglesInvertMouse(t *testing.T) {
	a := &App{
		mainMenu:       false,
		pauseScreen:    pauseScreenOptions,
		invertMouse:    false,
		optionButtons:  []*guiButton{},
		controlButtons: []*guiButton{},
		optionsKV:      make(map[string]string),
		optionsPath:    filepath.Join(t.TempDir(), "options.txt"),
	}
	a.handlePauseOptionButton(buttonIDOptionControls)
	if a.pauseScreen != pauseScreenControls {
		t.Fatalf("pause option controls should open controls screen: got=%d want=%d", a.pauseScreen, pauseScreenControls)
	}
	a.handlePauseControlButton(buttonIDControlInvert)
	if !a.invertMouse {
		t.Fatal("invert mouse should toggle on")
	}
	a.handlePauseControlButton(buttonIDControlInvert)
	if a.invertMouse {
		t.Fatal("invert mouse should toggle off")
	}
}

func TestPauseVideoGraphicsAndCloudsToggle(t *testing.T) {
	a := &App{
		mainMenu:      false,
		pauseScreen:   pauseScreenVideo,
		fancyGraphics: true,
		cloudsEnabled: true,
		videoButtons:  []*guiButton{},
		optionsKV:     make(map[string]string),
		optionsPath:   filepath.Join(t.TempDir(), "options.txt"),
	}
	a.handlePauseVideoButton(buttonIDVideoGraphics)
	if a.fancyGraphics {
		t.Fatal("graphics should toggle to fast")
	}
	a.handlePauseVideoButton(buttonIDVideoGraphics)
	if !a.fancyGraphics {
		t.Fatal("graphics should toggle back to fancy")
	}
	a.handlePauseVideoButton(buttonIDVideoClouds)
	if a.cloudsEnabled {
		t.Fatal("clouds should toggle off")
	}
	a.handlePauseVideoButton(buttonIDVideoClouds)
	if !a.cloudsEnabled {
		t.Fatal("clouds should toggle on")
	}
}

func TestShouldRenderCloudsRespectsDistanceAndToggle(t *testing.T) {
	a := &App{renderDistance: 0, cloudsEnabled: true}
	if !a.shouldRenderClouds() {
		t.Fatal("clouds should render at far when enabled")
	}
	a.renderDistance = 1
	if !a.shouldRenderClouds() {
		t.Fatal("clouds should render at normal when enabled")
	}
	a.renderDistance = 2
	if a.shouldRenderClouds() {
		t.Fatal("clouds should not render at short distance")
	}
	a.renderDistance = 0
	a.cloudsEnabled = false
	if a.shouldRenderClouds() {
		t.Fatal("clouds should not render when disabled")
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
		"music:0.40",
		"sound:0.70",
		"fov:0.25",
		"invertYMouse:true",
		"viewDistance:2",
		"guiScale:3",
		"bobView:false",
		"lang:en_US",
		"forceUnicodeFont:true",
		"chatVisibility:2",
		"chatColors:false",
		"chatLinks:false",
		"chatLinksPrompt:false",
		"serverTextures:false",
		"showCape:false",
		"mouseSensitivity:0.30",
		"fpsLimit:2",
		"difficulty:3",
		"fancyGraphics:false",
		"clouds:false",
		"key_key.forward:44",
		"customKey:customValue",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp options failed: %v", err)
	}

	a := &App{
		optionsPath:        path,
		optionsKV:          make(map[string]string),
		musicVolume:        1.0,
		soundVolume:        1.0,
		mouseSens:          0.14,
		renderDistance:     1,
		invertMouse:        false,
		viewBobbing:        true,
		fancyGraphics:      true,
		cloudsEnabled:      true,
		guiScaleMode:       0,
		limitFramerateMode: 1,
		optionDifficulty:   1,
	}
	a.loadOptionsFile()
	if a.musicVolume != 0.40 {
		t.Fatalf("loaded music mismatch: got=%.2f want=0.40", a.musicVolume)
	}
	if a.soundVolume != 0.70 {
		t.Fatalf("loaded sound mismatch: got=%.2f want=0.70", a.soundVolume)
	}
	if a.fovSetting != 0.25 {
		t.Fatalf("loaded fov mismatch: got=%.2f want=0.25", a.fovSetting)
	}
	if a.renderDistance != 2 {
		t.Fatalf("loaded viewDistance mismatch: got=%d want=2", a.renderDistance)
	}
	if a.viewBobbing {
		t.Fatal("loaded bobView should be false")
	}
	if !a.invertMouse {
		t.Fatal("loaded invertYMouse should be true")
	}
	if a.guiScaleMode != 3 {
		t.Fatalf("loaded guiScale mismatch: got=%d want=3", a.guiScaleMode)
	}
	if a.mouseSens != 0.30 {
		t.Fatalf("loaded sensitivity mismatch: got=%.2f want=0.30", a.mouseSens)
	}
	if a.languageCode != "en_US" {
		t.Fatalf("loaded language mismatch: got=%q want=%q", a.languageCode, "en_US")
	}
	if !a.forceUnicodeFont {
		t.Fatal("loaded forceUnicodeFont should be true")
	}
	if a.chatVisibility != 2 {
		t.Fatalf("loaded chatVisibility mismatch: got=%d want=2", a.chatVisibility)
	}
	if a.chatColours {
		t.Fatal("loaded chatColors should be false")
	}
	if a.chatLinks {
		t.Fatal("loaded chatLinks should be false")
	}
	if a.chatLinksPrompt {
		t.Fatal("loaded chatLinksPrompt should be false")
	}
	if a.serverTextures {
		t.Fatal("loaded serverTextures should be false")
	}
	if a.showCape {
		t.Fatal("loaded showCape should be false")
	}
	if a.limitFramerateMode != 2 {
		t.Fatalf("loaded fpsLimit mismatch: got=%d want=2", a.limitFramerateMode)
	}
	if a.optionDifficulty != 3 {
		t.Fatalf("loaded difficulty mismatch: got=%d want=3", a.optionDifficulty)
	}
	if a.fancyGraphics {
		t.Fatal("loaded fancyGraphics should be false")
	}
	if a.cloudsEnabled {
		t.Fatal("loaded clouds should be false")
	}
	if a.keyBindingCode(keyDescForward) != 44 {
		t.Fatalf("loaded key_key.forward mismatch: got=%d want=44", a.keyBindingCode(keyDescForward))
	}

	a.musicVolume = 0.35
	a.soundVolume = 0.65
	a.fovSetting = 1.0
	a.renderDistance = 0
	a.invertMouse = false
	a.viewBobbing = true
	a.guiScaleMode = 2
	a.mouseSens = 0.5
	a.languageCode = "en_US"
	a.forceUnicodeFont = false
	a.chatVisibility = 1
	a.chatColours = true
	a.chatLinks = true
	a.chatLinksPrompt = true
	a.serverTextures = true
	a.showCape = true
	a.limitFramerateMode = 0
	a.optionDifficulty = 0
	a.fancyGraphics = true
	a.cloudsEnabled = true
	a.setKeyBindingByIndex(a.keyBindingIndexByDescription(keyDescForward), 17)
	a.saveOptionsFile()

	updatedBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved options failed: %v", err)
	}
	updated := string(updatedBytes)
	checks := []string{
		"music:0.350000",
		"sound:0.650000",
		"fov:1.000000",
		"invertYMouse:false",
		"viewDistance:0",
		"guiScale:2",
		"bobView:true",
		"lang:en_US",
		"forceUnicodeFont:false",
		"chatVisibility:1",
		"chatColors:true",
		"chatLinks:true",
		"chatLinksPrompt:true",
		"serverTextures:true",
		"showCape:true",
		"mouseSensitivity:0.500000",
		"fpsLimit:0",
		"difficulty:0",
		"fancyGraphics:true",
		"clouds:true",
		"key_key.forward:17",
		"customKey:customValue",
	}
	for _, want := range checks {
		if !strings.Contains(updated, want) {
			t.Fatalf("saved options missing %q\nsaved:\n%s", want, updated)
		}
	}
}
