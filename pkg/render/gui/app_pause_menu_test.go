//go:build cgo

package gui

import (
	"path/filepath"
	"testing"
)

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

func TestCurrentPauseButtonsTracksVideoAndControlsSubscreens(t *testing.T) {
	a := &App{
		guiW:     854,
		guiH:     480,
		mainMenu: false,
	}
	a.initPauseButtons()
	a.initPauseOptionsButtons()

	a.pauseScreen = pauseScreenVideo
	videoButtons := a.currentPauseButtons()
	if len(videoButtons) == 0 {
		t.Fatal("pause video buttons should not be empty")
	}
	if videoButtons[len(videoButtons)-1].ID != buttonIDVideoDone {
		t.Fatalf("pause video done button mismatch: got id=%d want id=%d", videoButtons[len(videoButtons)-1].ID, buttonIDVideoDone)
	}

	a.pauseScreen = pauseScreenControls
	controlButtons := a.currentPauseButtons()
	if len(controlButtons) == 0 {
		t.Fatal("pause controls buttons should not be empty")
	}
	if controlButtons[len(controlButtons)-1].ID != buttonIDControlDone {
		t.Fatalf("pause controls done button mismatch: got id=%d want id=%d", controlButtons[len(controlButtons)-1].ID, buttonIDControlDone)
	}

	a.pauseScreen = pauseScreenKeyBindings
	keybindButtons := a.currentPauseButtons()
	if len(keybindButtons) == 0 {
		t.Fatal("pause key binding buttons should not be empty")
	}
	if keybindButtons[len(keybindButtons)-1].ID != buttonIDKeybindDone {
		t.Fatalf("pause key binding done button mismatch: got id=%d want id=%d", keybindButtons[len(keybindButtons)-1].ID, buttonIDKeybindDone)
	}

	a.pauseScreen = pauseScreenSounds
	soundButtons := a.currentPauseButtons()
	if len(soundButtons) == 0 {
		t.Fatal("pause sounds buttons should not be empty")
	}
	if soundButtons[len(soundButtons)-1].ID != buttonIDSoundDone {
		t.Fatalf("pause sounds done button mismatch: got id=%d want id=%d", soundButtons[len(soundButtons)-1].ID, buttonIDSoundDone)
	}
}

func TestPauseOptionMusicOpensSoundsAndAdjustsVolume(t *testing.T) {
	a := &App{
		guiW:          854,
		guiH:          480,
		mainMenu:      false,
		pauseScreen:   pauseScreenOptions,
		musicVolume:   1.0,
		soundVolume:   1.0,
		optionButtons: []*guiButton{},
		soundButtons:  []*guiButton{},
		optionsKV:     make(map[string]string),
		optionsPath:   filepath.Join(t.TempDir(), "options.txt"),
	}

	a.handlePauseOptionButton(buttonIDOptionMusic)
	if a.pauseScreen != pauseScreenSounds {
		t.Fatalf("pause option music should open sounds screen: got=%d want=%d", a.pauseScreen, pauseScreenSounds)
	}

	a.handlePauseSoundButton(buttonIDSoundMusicMinus)
	if a.musicVolume != 0.9 {
		t.Fatalf("music volume minus mismatch: got=%.2f want=0.90", a.musicVolume)
	}
	a.handlePauseSoundButton(buttonIDSoundSoundMinus)
	if a.soundVolume != 0.9 {
		t.Fatalf("sound volume minus mismatch: got=%.2f want=0.90", a.soundVolume)
	}

	a.handlePauseSoundButton(buttonIDSoundDone)
	if a.pauseScreen != pauseScreenOptions {
		t.Fatalf("pause sounds done should return to options: got=%d want=%d", a.pauseScreen, pauseScreenOptions)
	}
}

func TestPauseOptionsOpenLanguageChatAndResourceSubscreens(t *testing.T) {
	a := &App{
		guiW:                854,
		guiH:                480,
		mainMenu:            false,
		pauseScreen:         pauseScreenOptions,
		languageCode:        "en_US",
		snooperEnabled:      true,
		chatVisibility:      0,
		chatColours:         true,
		chatLinks:           true,
		chatLinksPrompt:     true,
		chatOpacity:         1.0,
		chatScale:           1.0,
		chatWidth:           1.0,
		chatHeightFocused:   1.0,
		chatHeightUnfocused: 0.44366196,
		showCape:            true,
		optionsKV:           make(map[string]string),
		optionsPath:         filepath.Join(t.TempDir(), "options.txt"),
	}
	a.initPauseButtons()
	a.initPauseOptionsButtons()

	a.handlePauseOptionButton(buttonIDOptionLanguage)
	if a.pauseScreen != pauseScreenLanguage {
		t.Fatalf("pause options language should open language screen: got=%d want=%d", a.pauseScreen, pauseScreenLanguage)
	}
	a.handlePauseLanguageButton(buttonIDLanguageDone)
	if a.pauseScreen != pauseScreenOptions {
		t.Fatalf("pause language done should return to options: got=%d want=%d", a.pauseScreen, pauseScreenOptions)
	}

	a.handlePauseOptionButton(buttonIDOptionSnooper)
	if a.pauseScreen != pauseScreenSnooper {
		t.Fatalf("pause options snooper should open snooper screen: got=%d want=%d", a.pauseScreen, pauseScreenSnooper)
	}
	a.handlePauseSnooperButton(buttonIDSnooperToggle)
	if a.snooperEnabled {
		t.Fatal("pause snooper toggle should flip to false")
	}
	a.handlePauseSnooperButton(buttonIDSnooperDone)
	if a.pauseScreen != pauseScreenOptions {
		t.Fatalf("pause snooper done should return to options: got=%d want=%d", a.pauseScreen, pauseScreenOptions)
	}

	a.handlePauseOptionButton(buttonIDOptionMultiplayer)
	if a.pauseScreen != pauseScreenChatOptions {
		t.Fatalf("pause options multiplayer should open chat settings: got=%d want=%d", a.pauseScreen, pauseScreenChatOptions)
	}
	a.handlePauseChatOptionButton(buttonIDChatVisibility)
	if a.chatVisibility != 1 {
		t.Fatalf("chat visibility should cycle in pause screen: got=%d want=1", a.chatVisibility)
	}
	a.handlePauseChatOptionButton(buttonIDChatWidthMinus)
	if a.chatWidth >= 1.0 {
		t.Fatalf("chat width should decrease in pause chat options: got=%.2f", a.chatWidth)
	}
	a.handlePauseChatOptionButton(buttonIDChatDone)
	if a.pauseScreen != pauseScreenOptions {
		t.Fatalf("pause chat done should return to options: got=%d want=%d", a.pauseScreen, pauseScreenOptions)
	}

	a.handlePauseOptionButton(buttonIDOptionResource)
	if a.pauseScreen != pauseScreenResourcePacks {
		t.Fatalf("pause options resource should open resource packs: got=%d want=%d", a.pauseScreen, pauseScreenResourcePacks)
	}
	a.handlePauseResourceButton(buttonIDResourceDone)
	if a.pauseScreen != pauseScreenOptions {
		t.Fatalf("pause resource done should return to options: got=%d want=%d", a.pauseScreen, pauseScreenOptions)
	}
}
