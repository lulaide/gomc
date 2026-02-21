//go:build cgo

package gui

import "testing"

func TestInitPauseButtonsVanillaLabels(t *testing.T) {
	a := &App{
		guiW:        854,
		guiH:        480,
		activeWorld: "New World",
	}
	a.initPauseButtons()

	if len(a.pauseButtons) != 6 {
		t.Fatalf("pause button count mismatch: got=%d want=6", len(a.pauseButtons))
	}
	if a.pauseButtons[0].Label != "Save and Quit to Title" {
		t.Fatalf("singleplayer quit label mismatch: got=%q", a.pauseButtons[0].Label)
	}
	if a.pauseButtons[1].Label != "Return to Game" {
		t.Fatalf("return label mismatch: got=%q", a.pauseButtons[1].Label)
	}
	if a.pauseButtons[2].Label != "Achievements" {
		t.Fatalf("achievements label mismatch: got=%q", a.pauseButtons[2].Label)
	}
	if a.pauseButtons[3].Label != "Statistics" {
		t.Fatalf("statistics label mismatch: got=%q", a.pauseButtons[3].Label)
	}
	if a.pauseButtons[4].Label != "Options..." {
		t.Fatalf("options label mismatch: got=%q", a.pauseButtons[4].Label)
	}
	if a.pauseButtons[5].Label != "Open to LAN" {
		t.Fatalf("open-to-lan label mismatch: got=%q", a.pauseButtons[5].Label)
	}
	if a.pauseButtons[2].Enabled {
		t.Fatal("achievements button should be disabled until implemented")
	}
	if a.pauseButtons[3].Enabled {
		t.Fatal("statistics button should be disabled until implemented")
	}
	if a.pauseButtons[5].Enabled {
		t.Fatal("open-to-lan button should be disabled until implemented")
	}
}

func TestInitPauseButtonsMultiplayerDisconnectLabel(t *testing.T) {
	a := &App{
		guiW: 854,
		guiH: 480,
	}
	a.initPauseButtons()
	if got := a.pauseButtons[0].Label; got != "Disconnect" {
		t.Fatalf("multiplayer quit label mismatch: got=%q want=%q", got, "Disconnect")
	}
}

func TestCurrentPauseButtonsTracksPauseScreen(t *testing.T) {
	a := &App{
		guiW: 854,
		guiH: 480,
	}
	a.initPauseButtons()
	a.initPauseOptionsButtons()

	a.pauseScreen = pauseScreenMain
	mainButtons := a.currentPauseButtons()
	if len(mainButtons) != 6 {
		t.Fatalf("main pause buttons mismatch: got=%d want=6", len(mainButtons))
	}

	a.pauseScreen = pauseScreenOptions
	optionButtons := a.currentPauseButtons()
	if len(optionButtons) == 0 {
		t.Fatal("pause options buttons should not be empty")
	}
	if optionButtons[len(optionButtons)-1].ID != buttonIDOptionDone {
		t.Fatalf("pause options done button mismatch: got id=%d want id=%d", optionButtons[len(optionButtons)-1].ID, buttonIDOptionDone)
	}
}
