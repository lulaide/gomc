//go:build cgo

package gui

import (
	"path/filepath"
	"testing"
)

func TestHandleMenuButton_OpenAndCloseSoundsMenu(t *testing.T) {
	a := &App{
		guiW:       854,
		guiH:       480,
		mainMenu:   true,
		menuScreen: menuScreenOptions,
	}
	a.initOptionButtons()
	a.initSoundButtons()
	a.updateOptionButtonsState()
	a.updateSoundButtonsState()

	if ok := a.handleMenuButton(buttonIDOptionMusic); !ok {
		t.Fatal("option music click should keep app running")
	}
	if a.menuScreen != menuScreenSounds {
		t.Fatalf("menu screen should switch to sounds settings: got=%d want=%d", a.menuScreen, menuScreenSounds)
	}

	if ok := a.handleMenuButton(buttonIDSoundDone); !ok {
		t.Fatal("sounds done click should keep app running")
	}
	if a.menuScreen != menuScreenOptions {
		t.Fatalf("menu screen should return to options: got=%d want=%d", a.menuScreen, menuScreenOptions)
	}
}

func TestHandleMenuButton_SoundsSettingsApplyAndPersist(t *testing.T) {
	a := &App{
		guiW:        854,
		guiH:        480,
		mainMenu:    true,
		menuScreen:  menuScreenSounds,
		musicVolume: 0.3,
		soundVolume: 0.7,
		optionsKV:   make(map[string]string),
		optionsPath: filepath.Join(t.TempDir(), "options.txt"),
	}
	a.initSoundButtons()
	a.updateSoundButtonsState()

	_ = a.handleMenuButton(buttonIDSoundMusicPlus)
	_ = a.handleMenuButton(buttonIDSoundSoundMinus)

	if a.musicVolume != 0.4 {
		t.Fatalf("music volume plus mismatch: got=%.2f want=0.40", a.musicVolume)
	}
	if a.soundVolume != 0.6 {
		t.Fatalf("sound volume minus mismatch: got=%.2f want=0.60", a.soundVolume)
	}

	_ = a.handleMenuButton(buttonIDSoundDone)
	if a.menuScreen != menuScreenOptions {
		t.Fatalf("sounds done should return to options: got=%d want=%d", a.menuScreen, menuScreenOptions)
	}
}
