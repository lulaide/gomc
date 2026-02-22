//go:build cgo

package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func TestMenuControlsOpensKeyBindingsScreen(t *testing.T) {
	a := &App{
		guiW:       854,
		guiH:       480,
		mainMenu:   true,
		menuScreen: menuScreenControls,
	}
	a.initDefaultKeyBindings()
	a.initControlButtons()
	a.initKeyBindingButtons()
	a.updateControlButtonsState()
	a.updateKeyBindingButtonsState()

	_ = a.handleMenuButton(buttonIDControlKeybinds)
	if a.menuScreen != menuScreenKeyBindings {
		t.Fatalf("controls key bindings click should open key binding screen: got=%d want=%d", a.menuScreen, menuScreenKeyBindings)
	}
}

func TestKeyBindingCaptureAndPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	a := &App{
		guiW:        854,
		guiH:        480,
		mainMenu:    true,
		menuScreen:  menuScreenKeyBindings,
		optionsKV:   make(map[string]string),
		optionsPath: path,
	}
	a.initDefaultKeyBindings()
	a.initKeyBindingButtons()
	a.updateKeyBindingButtonsState()

	idxForward := a.keyBindingIndexByDescription(keyDescForward)
	if idxForward < 0 {
		t.Fatal("forward key binding should exist")
	}
	_ = a.handleMenuButton(buttonIDKeybindBase + idxForward)
	if a.keyBindCapture != idxForward {
		t.Fatalf("expected capture index=%d got=%d", idxForward, a.keyBindCapture)
	}

	a.enqueueKeyPress(glfw.KeyA)
	if !a.tryCaptureKeyBindingFromKeyQueue() {
		t.Fatal("expected keyboard capture to apply binding")
	}
	if a.keyBindingCode(keyDescForward) != 30 {
		t.Fatalf("forward should bind to A (code 30): got=%d", a.keyBindingCode(keyDescForward))
	}

	_ = a.handleMenuButton(buttonIDKeybindBase + idxForward)
	if !a.tryCaptureKeyBindingFromMouse(true, false, false) {
		t.Fatal("expected mouse capture to apply binding")
	}
	if a.keyBindingCode(keyDescForward) != -100 {
		t.Fatalf("forward should bind to mouse left (code -100): got=%d", a.keyBindingCode(keyDescForward))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read options failed: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "key_key.forward:-100") {
		t.Fatalf("saved options missing rebound forward key:\n%s", text)
	}
}
