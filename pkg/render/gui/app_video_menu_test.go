//go:build cgo

package gui

import (
	"path/filepath"
	"testing"
)

func TestOptionVideoLabelInMainMenu(t *testing.T) {
	a := &App{
		guiW:               854,
		guiH:               480,
		mainMenu:           true,
		limitFramerateMode: 1,
	}
	a.initOptionButtons()
	a.updateOptionButtonsState()

	b := findOptionButton(a.optionButtons, buttonIDOptionVideo)
	if b == nil {
		t.Fatal("video option button should exist")
	}
	if b.Label != "Video Settings..." {
		t.Fatalf("main menu video option label mismatch: got=%q want=%q", b.Label, "Video Settings...")
	}
}

func TestHandleMenuButton_OpenAndCloseVideoMenu(t *testing.T) {
	a := &App{
		guiW:       854,
		guiH:       480,
		mainMenu:   true,
		menuScreen: menuScreenOptions,
	}
	a.initOptionButtons()
	a.initVideoButtons()
	a.updateOptionButtonsState()
	a.updateVideoButtonsState()

	if ok := a.handleMenuButton(buttonIDOptionVideo); !ok {
		t.Fatal("option video click should keep app running")
	}
	if a.menuScreen != menuScreenVideo {
		t.Fatalf("menu screen should switch to video settings: got=%d want=%d", a.menuScreen, menuScreenVideo)
	}

	if ok := a.handleMenuButton(buttonIDVideoDone); !ok {
		t.Fatal("video done click should keep app running")
	}
	if a.menuScreen != menuScreenOptions {
		t.Fatalf("menu screen should return to options: got=%d want=%d", a.menuScreen, menuScreenOptions)
	}
}

func TestHandleMenuButton_VideoSettingsApplyAndPersist(t *testing.T) {
	a := &App{
		guiW:               854,
		guiH:               480,
		mainMenu:           true,
		menuScreen:         menuScreenVideo,
		fovSetting:         0.0,
		renderDistance:     0,
		limitFramerateMode: 1,
		viewBobbing:        true,
		optionsKV:          make(map[string]string),
		optionsPath:        filepath.Join(t.TempDir(), "options.txt"),
	}
	a.initOptionButtons()
	a.initVideoButtons()

	_ = a.handleMenuButton(buttonIDVideoFOVPlus)
	_ = a.handleMenuButton(buttonIDVideoRDMinus)
	_ = a.handleMenuButton(buttonIDVideoFPS)
	_ = a.handleMenuButton(buttonIDVideoViewBobbing)

	if a.fovSetting <= 0.0 {
		t.Fatalf("video fov plus should increase fovSetting, got=%.2f", a.fovSetting)
	}
	if a.renderDistance != 1 {
		t.Fatalf("video render distance minus should move towards tiny: got=%d want=1", a.renderDistance)
	}
	if a.limitFramerateMode != 2 {
		t.Fatalf("video fps button should cycle framerate mode: got=%d want=2", a.limitFramerateMode)
	}
	if a.viewBobbing {
		t.Fatal("video view bobbing should toggle off")
	}
}
