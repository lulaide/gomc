//go:build cgo

package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptionsMultiplayerOpensChatSettingsAndSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	a := &App{
		guiW:            854,
		guiH:            480,
		mainMenu:        true,
		menuScreen:      menuScreenOptions,
		optionsPath:     path,
		optionsKV:       make(map[string]string),
		chatVisibility:  0,
		chatColours:     true,
		chatLinks:       true,
		chatLinksPrompt: true,
		showCape:        true,
		serverTextures:  true,
		languageCode:    "en_US",
	}
	a.initOptionButtons()
	a.initChatOptionButtons()
	a.updateOptionButtonsState()
	a.updateChatOptionButtonsState()

	_ = a.handleMenuButton(buttonIDOptionMultiplayer)
	if a.menuScreen != menuScreenChatOptions {
		t.Fatalf("multiplayer settings should open chat settings screen: got=%d want=%d", a.menuScreen, menuScreenChatOptions)
	}

	_ = a.handleMenuButton(buttonIDChatVisibility)
	if a.chatVisibility != 1 {
		t.Fatalf("chat visibility should cycle to 1: got=%d", a.chatVisibility)
	}
	_ = a.handleMenuButton(buttonIDChatLinks)
	if a.chatLinks {
		t.Fatal("chat links should toggle off")
	}
	_ = a.handleMenuButton(buttonIDChatLinksPrompt)
	if !a.chatLinksPrompt {
		t.Fatal("chat links prompt should not toggle while chat links are disabled")
	}
	_ = a.handleMenuButton(buttonIDChatShowCape)
	if a.showCape {
		t.Fatal("show cape should toggle off")
	}

	_ = a.handleMenuButton(buttonIDChatDone)
	if a.menuScreen != menuScreenOptions {
		t.Fatalf("chat done should return to options: got=%d want=%d", a.menuScreen, menuScreenOptions)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read options after chat settings save failed: %v", err)
	}
	out := string(raw)
	checks := []string{
		"chatVisibility:1",
		"chatLinks:false",
		"showCape:false",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("saved options missing %q:\n%s", want, out)
		}
	}
}

func TestOptionsResourcePacksOpensAndReturns(t *testing.T) {
	a := &App{
		guiW:       854,
		guiH:       480,
		mainMenu:   true,
		menuScreen: menuScreenOptions,
	}
	a.initOptionButtons()
	a.initResourceButtons()
	a.updateOptionButtonsState()
	a.updateResourceButtonsState()

	_ = a.handleMenuButton(buttonIDOptionResource)
	if a.menuScreen != menuScreenResourcePacks {
		t.Fatalf("resource packs button should open resource screen: got=%d want=%d", a.menuScreen, menuScreenResourcePacks)
	}
	if !a.handleMenuEscape() {
		t.Fatal("escape should close resource packs screen")
	}
	if a.menuScreen != menuScreenOptions {
		t.Fatalf("escape should return to options: got=%d want=%d", a.menuScreen, menuScreenOptions)
	}
}

func TestOptionsSnooperOpensAndToggles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	a := &App{
		guiW:           854,
		guiH:           480,
		mainMenu:       true,
		menuScreen:     menuScreenOptions,
		snooperEnabled: true,
		optionsPath:    path,
		optionsKV:      make(map[string]string),
		languageCode:   "en_US",
	}
	a.initOptionButtons()
	a.initSnooperButtons()
	a.updateOptionButtonsState()
	a.updateSnooperButtonsState()

	_ = a.handleMenuButton(buttonIDOptionSnooper)
	if a.menuScreen != menuScreenSnooper {
		t.Fatalf("snooper option should open snooper screen: got=%d want=%d", a.menuScreen, menuScreenSnooper)
	}

	_ = a.handleMenuButton(buttonIDSnooperToggle)
	if a.snooperEnabled {
		t.Fatal("snooper toggle should flip to false")
	}
	_ = a.handleMenuButton(buttonIDSnooperDone)
	if a.menuScreen != menuScreenOptions {
		t.Fatalf("snooper done should return to options: got=%d want=%d", a.menuScreen, menuScreenOptions)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read options after snooper save failed: %v", err)
	}
	if !strings.Contains(string(raw), "snooperEnabled:false") {
		t.Fatalf("saved options missing snooperEnabled:false:\n%s", string(raw))
	}
}
