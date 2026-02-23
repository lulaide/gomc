//go:build cgo

package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainMenuLanguageOpensAndClosesLanguageScreen(t *testing.T) {
	a := &App{
		guiW:           854,
		guiH:           480,
		mainMenu:       true,
		menuScreen:     menuScreenMain,
		languageCode:   "en_US",
		languageReturn: menuScreenMain,
	}
	a.initMainButtons()
	a.initLanguageButtons()
	a.updateLanguageButtonsState()

	langBtn := findOptionButton(a.mainButtons, buttonIDMenuLanguage)
	if langBtn == nil || !langBtn.Enabled {
		t.Fatal("main menu language button should be enabled")
	}

	_ = a.handleMenuButton(buttonIDMenuLanguage)
	if a.menuScreen != menuScreenLanguage {
		t.Fatalf("main menu language should open language screen: got=%d want=%d", a.menuScreen, menuScreenLanguage)
	}
	if a.languageReturn != menuScreenMain {
		t.Fatalf("language return target mismatch: got=%d want=%d", a.languageReturn, menuScreenMain)
	}

	_ = a.handleMenuButton(buttonIDLanguageDone)
	if a.menuScreen != menuScreenMain {
		t.Fatalf("language done should return to main menu: got=%d want=%d", a.menuScreen, menuScreenMain)
	}
}

func TestOptionsLanguageOpensLanguageScreenAndEscReturnsToOptions(t *testing.T) {
	a := &App{
		guiW:         854,
		guiH:         480,
		mainMenu:     true,
		menuScreen:   menuScreenOptions,
		languageCode: "en_US",
	}
	a.initOptionButtons()
	a.initLanguageButtons()
	a.updateOptionButtonsState()
	a.updateLanguageButtonsState()

	_ = a.handleMenuButton(buttonIDOptionLanguage)
	if a.menuScreen != menuScreenLanguage {
		t.Fatalf("options language should open language screen: got=%d want=%d", a.menuScreen, menuScreenLanguage)
	}
	if a.languageReturn != menuScreenOptions {
		t.Fatalf("language return target mismatch: got=%d want=%d", a.languageReturn, menuScreenOptions)
	}

	if !a.handleMenuEscape() {
		t.Fatal("escape should close language screen")
	}
	if a.menuScreen != menuScreenOptions {
		t.Fatalf("escape should return to options: got=%d want=%d", a.menuScreen, menuScreenOptions)
	}
}

func TestLanguageMenuSavesLangAndUnicodeOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	a := &App{
		guiW:             854,
		guiH:             480,
		mainMenu:         true,
		menuScreen:       menuScreenLanguage,
		languageReturn:   menuScreenMain,
		optionsPath:      path,
		optionsKV:        make(map[string]string),
		languageCode:     "fr_FR",
		forceUnicodeFont: false,
	}
	a.initLanguageButtons()
	a.updateLanguageButtonsState()

	_ = a.handleMenuButton(buttonIDLanguageEnglish)
	_ = a.handleMenuButton(buttonIDLanguageForceUnicode)

	if a.languageCode != "en_US" {
		t.Fatalf("language selection mismatch: got=%q want=%q", a.languageCode, "en_US")
	}
	if !a.forceUnicodeFont {
		t.Fatal("force unicode should toggle on")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read options after language save failed: %v", err)
	}
	out := string(raw)
	if !strings.Contains(out, "lang:en_US") {
		t.Fatalf("saved options missing lang key:\n%s", out)
	}
	if !strings.Contains(out, "forceUnicodeFont:true") {
		t.Fatalf("saved options missing forceUnicodeFont key:\n%s", out)
	}
}
