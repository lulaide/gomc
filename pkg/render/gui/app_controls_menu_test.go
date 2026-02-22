//go:build cgo

package gui

import (
	"path/filepath"
	"testing"
)

func TestMainMenuOptionsHideDirectSliders(t *testing.T) {
	a := &App{
		guiW:     854,
		guiH:     480,
		mainMenu: true,
	}
	a.initOptionButtons()
	a.updateOptionButtonsState()

	if b := findOptionButton(a.optionButtons, buttonIDOptionRDMinus); b == nil || b.Visible {
		t.Fatal("main menu options should hide render-distance slider buttons")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionFOVMinus); b == nil || b.Visible {
		t.Fatal("main menu options should hide fov slider buttons")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionSensMinus); b == nil || b.Visible {
		t.Fatal("main menu options should hide sensitivity slider buttons")
	}
	if b := findOptionButton(a.optionButtons, buttonIDOptionViewBobbing); b == nil || b.Visible {
		t.Fatal("main menu options should hide view bobbing direct toggle")
	}
}

func TestMainMenuOptionControlsOpensControlsScreen(t *testing.T) {
	a := &App{
		guiW:       854,
		guiH:       480,
		mainMenu:   true,
		menuScreen: menuScreenOptions,
	}
	a.initOptionButtons()
	a.initControlButtons()
	a.updateOptionButtonsState()
	a.updateControlButtonsState()

	_ = a.handleMenuButton(buttonIDOptionControls)
	if a.menuScreen != menuScreenControls {
		t.Fatalf("main menu controls should open controls screen: got=%d want=%d", a.menuScreen, menuScreenControls)
	}
}

func TestControlsMenuUpdatesMouseSettings(t *testing.T) {
	a := &App{
		guiW:       854,
		guiH:       480,
		mainMenu:   true,
		menuScreen: menuScreenControls,
		mouseSens:  0.20,
		optionsKV:  make(map[string]string),
		optionsPath: filepath.Join(
			t.TempDir(),
			"options.txt",
		),
	}
	a.initControlButtons()
	a.updateControlButtonsState()

	_ = a.handleMenuButton(buttonIDControlSensPlus)
	if a.mouseSens <= 0.20 {
		t.Fatalf("controls sensitivity plus should increase value, got=%.2f", a.mouseSens)
	}

	_ = a.handleMenuButton(buttonIDControlInvert)
	if !a.invertMouse {
		t.Fatal("controls invert button should toggle invertMouse on")
	}

	_ = a.handleMenuButton(buttonIDControlDone)
	if a.menuScreen != menuScreenOptions {
		t.Fatalf("controls done should return to options: got=%d want=%d", a.menuScreen, menuScreenOptions)
	}
}
