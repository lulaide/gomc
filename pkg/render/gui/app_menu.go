//go:build cgo

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lulaide/gomc/pkg/audio"
	"github.com/lulaide/gomc/pkg/nbt"
)

type menuScreen int

const (
	menuScreenMain menuScreen = iota
	menuScreenSingleplayer
	menuScreenMultiplayer
	menuScreenOptions
	menuScreenVideo
	menuScreenControls
	menuScreenKeyBindings
	menuScreenSounds
	menuScreenCreateWorld
	menuScreenRenameWorld
)

type menuTextField int

const (
	textFieldNone menuTextField = iota
	textFieldCreateName
	textFieldCreateSeed
	textFieldRenameName
)

type singleWorldMeta struct {
	DirName        string
	DisplayName    string
	LastPlayedMS   int64
	LastPlayedText string
	GameType       int
	Hardcore       bool
}

func (m singleWorldMeta) modeText() string {
	if m.Hardcore {
		return "Hardcore Mode"
	}
	switch m.GameType {
	case 1:
		return "Creative Mode"
	case 2:
		return "Adventure Mode"
	default:
		return "Survival Mode"
	}
}

const (
	buttonIDMenuBack = 1000

	buttonIDSinglePlay     = 1101
	buttonIDSingleCreate   = 1102
	buttonIDSingleDelete   = 1103
	buttonIDSingleRename   = 1104
	buttonIDSingleRecreate = 1105

	buttonIDMultiJoin    = 1201
	buttonIDMultiDirect  = 1202
	buttonIDMultiAdd     = 1203
	buttonIDMultiEdit    = 1204
	buttonIDMultiDelete  = 1205
	buttonIDMultiRefresh = 1206

	buttonIDOptionMusic       = 1301
	buttonIDOptionVideo       = 1302
	buttonIDOptionControls    = 1303
	buttonIDOptionLanguage    = 1304
	buttonIDOptionDifficulty  = 1305
	buttonIDOptionSnooper     = 1306
	buttonIDOptionMultiplayer = 1307
	buttonIDOptionResource    = 1308
	buttonIDOptionRDMinus     = 1310
	buttonIDOptionRDPlus      = 1311
	buttonIDOptionSensMinus   = 1312
	buttonIDOptionSensPlus    = 1313
	buttonIDOptionFOVMinus    = 1314
	buttonIDOptionFOVPlus     = 1315
	buttonIDOptionViewBobbing = 1316
	buttonIDOptionDone        = 1399

	buttonIDVideoGraphics    = 1601
	buttonIDVideoClouds      = 1602
	buttonIDVideoRDMinus     = 1603
	buttonIDVideoRDPlus      = 1604
	buttonIDVideoFOVMinus    = 1605
	buttonIDVideoFOVPlus     = 1606
	buttonIDVideoFPS         = 1607
	buttonIDVideoGUIScale    = 1608
	buttonIDVideoViewBobbing = 1609
	buttonIDVideoDone        = 1699

	buttonIDControlSensMinus   = 1701
	buttonIDControlSensPlus    = 1702
	buttonIDControlInvert      = 1703
	buttonIDControlDone        = 1704
	buttonIDControlKeybinds    = 1705
	buttonIDControlTouchscreen = 1706

	buttonIDSoundMusicMinus = 1801
	buttonIDSoundMusicPlus  = 1802
	buttonIDSoundSoundMinus = 1803
	buttonIDSoundSoundPlus  = 1804
	buttonIDSoundDone       = 1805
	buttonIDKeybindBase     = 1900
	buttonIDKeybindDone     = 1999

	buttonIDCreateDone          = 1401
	buttonIDCreateCancel        = 1402
	buttonIDCreateGameMode      = 1403
	buttonIDCreateMoreOptions   = 1404
	buttonIDCreateMapFeatures   = 1405
	buttonIDCreateWorldType     = 1406
	buttonIDCreateAllowCommands = 1407
	buttonIDCreateBonusChest    = 1408
	buttonIDCreateCustomize     = 1409

	buttonIDRenameDone   = 1501
	buttonIDRenameCancel = 1502
)

func (a *App) initAllMenuButtons() {
	a.initMainButtons()
	a.initSingleButtons()
	a.initMultiButtons()
	a.initOptionButtons()
	a.initVideoButtons()
	a.initControlButtons()
	a.initKeyBindingButtons()
	a.initSoundButtons()
	a.initCreateButtons()
	a.initRenameButtons()
	if len(a.singleWorlds) == 0 {
		a.refreshSingleplayerWorlds()
	}
	a.updateSingleButtonsState()
	a.updateOptionButtonsState()
	a.updateVideoButtonsState()
	a.updateControlButtonsState()
	a.updateKeyBindingButtonsState()
	a.updateSoundButtonsState()
	a.updateCreateButtonsState()
	a.updateRenameButtonsState()
}

func (a *App) initSingleButtons() {
	w, h := a.uiWidth(), a.uiHeight()
	baseY := h - 52
	a.singleButtons = []*guiButton{
		newButton(buttonIDSinglePlay, w/2-154, baseY, 150, 20, "Play Selected World"),
		newButton(buttonIDSingleCreate, w/2+4, baseY, 150, 20, "Create New World"),
		newButton(buttonIDSingleRename, w/2-154, baseY+24, 72, 20, "Rename"),
		newButton(buttonIDSingleDelete, w/2-76, baseY+24, 72, 20, "Delete"),
		newButton(buttonIDSingleRecreate, w/2+4, baseY+24, 72, 20, "Re-Create"),
		newButton(buttonIDMenuBack, w/2+82, baseY+24, 72, 20, "Cancel"),
	}
	a.singleButtons[2].Enabled = false // Rename
	a.singleButtons[3].Enabled = false // Delete
	a.singleButtons[4].Enabled = false // Re-Create
}

func (a *App) initMultiButtons() {
	w, h := a.uiWidth(), a.uiHeight()
	baseY := h - 52
	a.multiButtons = []*guiButton{
		newButton(buttonIDMultiEdit, w/2-154, baseY+24, 70, 20, "Edit"),
		newButton(buttonIDMultiDelete, w/2-74, baseY+24, 70, 20, "Delete"),
		newButton(buttonIDMultiJoin, w/2-154, baseY, 100, 20, "Join Server"),
		newButton(buttonIDMultiDirect, w/2-50, baseY, 100, 20, "Direct Connect"),
		newButton(buttonIDMultiAdd, w/2+54, baseY, 100, 20, "Add server"),
		newButton(buttonIDMultiRefresh, w/2+4, baseY+24, 70, 20, "Refresh"),
		newButton(buttonIDMenuBack, w/2+80, baseY+24, 75, 20, "Cancel"),
	}
	a.multiButtons[0].Enabled = false // Edit
	a.multiButtons[1].Enabled = false // Delete
	a.multiButtons[2].Enabled = true  // Join
	a.multiButtons[3].Enabled = true  // Direct
	a.multiButtons[4].Enabled = false // Add
	a.multiButtons[5].Enabled = false // Refresh
}

func (a *App) initOptionButtons() {
	w, h := a.uiWidth(), a.uiHeight()
	baseY := h/6 + 12
	a.optionButtons = []*guiButton{
		newButton(buttonIDOptionMusic, w/2-155, baseY, 150, 20, "Music & Sounds..."),
		newButton(buttonIDOptionVideo, w/2+5, baseY, 150, 20, "Video Settings..."),
		newButton(buttonIDOptionControls, w/2-155, baseY+24, 150, 20, "Controls..."),
		newButton(buttonIDOptionLanguage, w/2+5, baseY+24, 150, 20, "Language..."),
		newButton(buttonIDOptionDifficulty, w/2-155, baseY+48, 150, 20, "Difficulty: Easy"),
		newButton(buttonIDOptionViewBobbing, w/2+5, baseY+48, 150, 20, "View Bobbing: ON"),
		newButton(buttonIDOptionRDMinus, w/2-100, baseY+82, 20, 20, "-"),
		newButton(buttonIDOptionRDPlus, w/2+80, baseY+82, 20, 20, "+"),
		newButton(buttonIDOptionFOVMinus, w/2-100, baseY+106, 20, 20, "-"),
		newButton(buttonIDOptionFOVPlus, w/2+80, baseY+106, 20, 20, "+"),
		newButton(buttonIDOptionSensMinus, w/2-100, baseY+130, 20, 20, "-"),
		newButton(buttonIDOptionSensPlus, w/2+80, baseY+130, 20, 20, "+"),
		newButton(buttonIDOptionSnooper, w/2-100, baseY+164, 98, 20, "Clouds: ON"),
		newButton(buttonIDOptionMultiplayer, w/2-155, baseY+188, 150, 20, "Multiplayer Settings..."),
		newButton(buttonIDOptionResource, w/2+5, baseY+188, 150, 20, "Resource Packs..."),
		newButton(buttonIDOptionDone, w/2+2, baseY+164, 98, 20, "Done"),
	}
}

func (a *App) initVideoButtons() {
	w, h := a.uiWidth(), a.uiHeight()
	baseY := h/6 + 20
	a.videoButtons = []*guiButton{
		newButton(buttonIDVideoGraphics, w/2-155, baseY, 150, 20, "Graphics: Fancy"),
		newButton(buttonIDVideoClouds, w/2+5, baseY, 150, 20, "Clouds: ON"),
		newButton(buttonIDVideoRDMinus, w/2-100, baseY+24, 20, 20, "-"),
		newButton(buttonIDVideoRDPlus, w/2+80, baseY+24, 20, 20, "+"),
		newButton(buttonIDVideoFOVMinus, w/2-100, baseY+48, 20, 20, "-"),
		newButton(buttonIDVideoFOVPlus, w/2+80, baseY+48, 20, 20, "+"),
		newButton(buttonIDVideoFPS, w/2-155, baseY+72, 150, 20, "Max Framerate: Balanced"),
		newButton(buttonIDVideoGUIScale, w/2+5, baseY+72, 150, 20, "GUI Scale: Auto"),
		newButton(buttonIDVideoViewBobbing, w/2-75, baseY+96, 150, 20, "View Bobbing: ON"),
		newButton(buttonIDVideoDone, w/2-100, baseY+132, 200, 20, "Done"),
	}
}

func (a *App) initControlButtons() {
	w, h := a.uiWidth(), a.uiHeight()
	baseY := h/6 + 20
	a.controlButtons = []*guiButton{
		newButton(buttonIDControlKeybinds, w/2-155, baseY, 150, 20, "Key Bindings..."),
		newButton(buttonIDControlTouchscreen, w/2+5, baseY, 150, 20, "Touchscreen Mode"),
		newButton(buttonIDControlSensMinus, w/2-100, baseY+24, 20, 20, "-"),
		newButton(buttonIDControlSensPlus, w/2+80, baseY+24, 20, 20, "+"),
		newButton(buttonIDControlInvert, w/2-75, baseY+48, 150, 20, "Invert Mouse: OFF"),
		newButton(buttonIDControlDone, w/2-100, baseY+84, 200, 20, "Done"),
	}
}

func (a *App) initKeyBindingButtons() {
	if len(a.keyBindings) == 0 {
		a.initDefaultKeyBindings()
	}
	w, h := a.uiWidth(), a.uiHeight()
	left := w/2 - 155
	buttons := make([]*guiButton, 0, len(a.keyBindings)+1)
	for i := range a.keyBindings {
		x := left + (i%2)*160
		y := h/6 + 24*(i>>1)
		buttons = append(buttons, newButton(buttonIDKeybindBase+i, x, y, 70, 20, a.keyBindingButtonLabel(i)))
	}
	buttons = append(buttons, newButton(buttonIDKeybindDone, w/2-100, h/6+168, 200, 20, "Done"))
	a.keyBindButtons = buttons
}

func (a *App) initSoundButtons() {
	w, h := a.uiWidth(), a.uiHeight()
	baseY := h/6 + 20
	a.soundButtons = []*guiButton{
		newButton(buttonIDSoundMusicMinus, w/2-100, baseY, 20, 20, "-"),
		newButton(buttonIDSoundMusicPlus, w/2+80, baseY, 20, 20, "+"),
		newButton(buttonIDSoundSoundMinus, w/2-100, baseY+24, 20, 20, "-"),
		newButton(buttonIDSoundSoundPlus, w/2+80, baseY+24, 20, 20, "+"),
		newButton(buttonIDSoundDone, w/2-100, baseY+60, 200, 20, "Done"),
	}
}

func (a *App) initCreateButtons() {
	w, h := a.uiWidth(), a.uiHeight()
	a.createButtons = []*guiButton{
		newButton(buttonIDCreateDone, w/2-155, h-28, 150, 20, "Create New World"),
		newButton(buttonIDCreateCancel, w/2+5, h-28, 150, 20, "Cancel"),
		newButton(buttonIDCreateGameMode, w/2-75, 115, 150, 20, "Game Mode: Survival"),
		newButton(buttonIDCreateMoreOptions, w/2-75, 187, 150, 20, "More World Options..."),
		newButton(buttonIDCreateMapFeatures, w/2-155, 100, 150, 20, "Generate Structures: ON"),
		newButton(buttonIDCreateWorldType, w/2+5, 100, 150, 20, "World Type: Default"),
		newButton(buttonIDCreateAllowCommands, w/2-155, 151, 150, 20, "Allow Cheats: OFF"),
		newButton(buttonIDCreateBonusChest, w/2+5, 151, 150, 20, "Bonus Chest: OFF"),
		newButton(buttonIDCreateCustomize, w/2+5, 120, 150, 20, "Customize"),
	}
}

func (a *App) initRenameButtons() {
	w, h := a.uiWidth(), a.uiHeight()
	a.renameButtons = []*guiButton{
		newButton(buttonIDRenameDone, w/2-100, h/4+96+12, 200, 20, "Rename"),
		newButton(buttonIDRenameCancel, w/2-100, h/4+120+12, 200, 20, "Cancel"),
	}
}

func (a *App) updateSingleButtonsState() {
	canPlay := len(a.singleWorlds) > 0 && a.selectedWorld >= 0 && a.selectedWorld < len(a.singleWorlds)
	for _, b := range a.singleButtons {
		if b == nil {
			continue
		}
		if b.ID == buttonIDSinglePlay || b.ID == buttonIDSingleDelete || b.ID == buttonIDSingleRename || b.ID == buttonIDSingleRecreate {
			b.Enabled = canPlay
		}
	}
}

func (a *App) updateOptionButtonsState() {
	difficultyName := []string{"Peaceful", "Easy", "Normal", "Hard"}[a.optionDifficulty&3]
	viewBobbingLabel := "View Bobbing: OFF"
	if a.viewBobbing {
		viewBobbingLabel = "View Bobbing: ON"
	}
	for _, b := range a.optionButtons {
		if b == nil {
			continue
		}
		b.Visible = true
		switch b.ID {
		case buttonIDOptionDifficulty:
			b.Label = "Difficulty: " + difficultyName
			b.Enabled = true
		case buttonIDOptionMusic:
			if a.mainMenu {
				b.Label = "Music & Sounds..."
				b.Enabled = true
			} else {
				b.Label = a.optionGraphicsLabel()
				b.Enabled = true
			}
		case buttonIDOptionVideo:
			if a.mainMenu {
				b.Label = "Video Settings..."
				b.Enabled = true
			} else {
				b.Label = a.optionFramerateLabel()
				b.Enabled = true
			}
		case buttonIDOptionControls:
			if a.mainMenu {
				b.Label = "Controls..."
				b.Enabled = true
			} else {
				b.Label = a.optionInvertMouseLabel()
				b.Enabled = true
			}
		case buttonIDOptionLanguage:
			if a.mainMenu {
				b.Label = "Language..."
				b.Enabled = true
			} else {
				b.Label = a.optionGUIScaleLabel()
				b.Enabled = true
			}
		case buttonIDOptionViewBobbing:
			if a.mainMenu {
				b.Visible = false
				b.Enabled = false
			} else {
				b.Label = viewBobbingLabel
				b.Enabled = true
			}
		case buttonIDOptionSnooper:
			if a.mainMenu {
				b.Label = "Snooper Settings..."
				b.Enabled = true
			} else {
				b.Label = a.optionCloudsLabel()
				b.Enabled = true
			}
		case buttonIDOptionMultiplayer:
			b.Label = "Multiplayer Settings..."
			b.Enabled = true
		case buttonIDOptionResource:
			b.Label = "Resource Packs..."
			b.Enabled = true
		case buttonIDOptionRDMinus:
			if a.mainMenu {
				b.Visible = false
				b.Enabled = false
			} else {
				b.Enabled = a.renderDistance < 3
			}
		case buttonIDOptionRDPlus:
			if a.mainMenu {
				b.Visible = false
				b.Enabled = false
			} else {
				b.Enabled = a.renderDistance > 0
			}
		case buttonIDOptionFOVMinus:
			if a.mainMenu {
				b.Visible = false
				b.Enabled = false
			} else {
				b.Enabled = a.fovSetting > 0.0
			}
		case buttonIDOptionFOVPlus:
			if a.mainMenu {
				b.Visible = false
				b.Enabled = false
			} else {
				b.Enabled = a.fovSetting < 1.0
			}
		case buttonIDOptionSensMinus:
			if a.mainMenu {
				b.Visible = false
				b.Enabled = false
			} else {
				b.Enabled = a.mouseSens > 0.0
			}
		case buttonIDOptionSensPlus:
			if a.mainMenu {
				b.Visible = false
				b.Enabled = false
			} else {
				b.Enabled = a.mouseSens < 1.0
			}
		case buttonIDOptionDone:
			b.Enabled = true
			b.Visible = true
		}
	}
}

func (a *App) updateVideoButtonsState() {
	viewBobbingLabel := "View Bobbing: OFF"
	if a.viewBobbing {
		viewBobbingLabel = "View Bobbing: ON"
	}
	for _, b := range a.videoButtons {
		if b == nil {
			continue
		}
		switch b.ID {
		case buttonIDVideoGraphics:
			b.Label = a.optionGraphicsLabel()
			b.Enabled = true
		case buttonIDVideoClouds:
			b.Label = a.optionCloudsLabel()
			b.Enabled = true
		case buttonIDVideoFPS:
			b.Label = a.optionFramerateLabel()
			b.Enabled = true
		case buttonIDVideoGUIScale:
			b.Label = a.optionGUIScaleLabel()
			b.Enabled = true
		case buttonIDVideoViewBobbing:
			b.Label = viewBobbingLabel
			b.Enabled = true
		case buttonIDVideoRDMinus:
			b.Enabled = a.renderDistance < 3
		case buttonIDVideoRDPlus:
			b.Enabled = a.renderDistance > 0
		case buttonIDVideoFOVMinus:
			b.Enabled = a.fovSetting > 0.0
		case buttonIDVideoFOVPlus:
			b.Enabled = a.fovSetting < 1.0
		case buttonIDVideoDone:
			b.Enabled = true
		}
	}
}

func (a *App) updateControlButtonsState() {
	for _, b := range a.controlButtons {
		if b == nil {
			continue
		}
		b.Visible = true
		switch b.ID {
		case buttonIDControlKeybinds:
			b.Label = "Key Bindings..."
			b.Enabled = true
		case buttonIDControlTouchscreen:
			b.Label = "Touchscreen Mode"
			b.Enabled = false
		case buttonIDControlSensMinus:
			b.Enabled = a.mouseSens > 0.0
		case buttonIDControlSensPlus:
			b.Enabled = a.mouseSens < 1.0
		case buttonIDControlInvert:
			b.Label = a.optionInvertMouseLabel()
			b.Enabled = true
		case buttonIDControlDone:
			b.Enabled = true
		}
	}
}

func (a *App) updateKeyBindingButtonsState() {
	if len(a.keyBindings) == 0 {
		a.initDefaultKeyBindings()
	}
	if len(a.keyBindButtons) != len(a.keyBindings)+1 {
		a.initKeyBindingButtons()
	}
	for i := range a.keyBindings {
		b := a.keyBindButtons[i]
		if b == nil {
			continue
		}
		b.Visible = true
		b.Enabled = true
		b.Label = a.keyBindingButtonLabel(i)
	}
	if len(a.keyBindButtons) > 0 {
		done := a.keyBindButtons[len(a.keyBindButtons)-1]
		done.Visible = true
		done.Enabled = true
		done.Label = "Done"
	}
}

func (a *App) updateSoundButtonsState() {
	for _, b := range a.soundButtons {
		if b == nil {
			continue
		}
		b.Visible = true
		switch b.ID {
		case buttonIDSoundMusicMinus:
			b.Enabled = a.musicVolume > 0.0
		case buttonIDSoundMusicPlus:
			b.Enabled = a.musicVolume < 1.0
		case buttonIDSoundSoundMinus:
			b.Enabled = a.soundVolume > 0.0
		case buttonIDSoundSoundPlus:
			b.Enabled = a.soundVolume < 1.0
		case buttonIDSoundDone:
			b.Enabled = true
		}
	}
}

func (a *App) updateCreateButtonsState() {
	a.updateCreateFolderName()
	a.createWorldTypeID = normalizeCreateWorldTypeID(a.createWorldTypeID)
	modeIsHardcore := a.createWorldMode == 1

	featuresLabel := "Generate Structures: OFF"
	if a.createMapFeature {
		featuresLabel = "Generate Structures: ON"
	}
	cheatsLabel := "Allow Cheats: OFF"
	if a.createAllowCheats && !modeIsHardcore {
		cheatsLabel = "Allow Cheats: ON"
	}
	bonusLabel := "Bonus Chest: OFF"
	if a.createBonusChest && !modeIsHardcore {
		bonusLabel = "Bonus Chest: ON"
	}
	worldTypeLabel := "World Type: " + createWorldTypeName(a.createWorldTypeID)
	moreOptionsLabel := "More World Options..."
	if a.createMoreOptions {
		moreOptionsLabel = "Done"
	}
	nameReady := len(strings.TrimSpace(a.createWorldName)) > 0

	for _, b := range a.createButtons {
		if b == nil {
			continue
		}
		switch b.ID {
		case buttonIDCreateDone:
			b.Enabled = nameReady
			b.Visible = true
		case buttonIDCreateCancel:
			b.Visible = true
		case buttonIDCreateMoreOptions:
			b.Label = moreOptionsLabel
			b.Visible = true
		case buttonIDCreateGameMode:
			b.Label = "Game Mode: " + createModeName(a.createWorldMode)
			b.Visible = !a.createMoreOptions
			b.Enabled = true
		case buttonIDCreateMapFeatures:
			b.Label = featuresLabel
			b.Visible = a.createMoreOptions
			b.Enabled = true
		case buttonIDCreateWorldType:
			b.Label = worldTypeLabel
			b.Visible = a.createMoreOptions
			b.Enabled = true
		case buttonIDCreateAllowCommands:
			b.Label = cheatsLabel
			b.Visible = a.createMoreOptions
			b.Enabled = !modeIsHardcore
		case buttonIDCreateBonusChest:
			b.Label = bonusLabel
			b.Visible = a.createMoreOptions
			b.Enabled = !modeIsHardcore
		case buttonIDCreateCustomize:
			b.Label = "Customize"
			b.Visible = a.createMoreOptions && a.createWorldTypeID == 1
			b.Enabled = false
		}
	}
}

func (a *App) updateRenameButtonsState() {
	nameReady := len(strings.TrimSpace(a.renameWorldName)) > 0
	for _, b := range a.renameButtons {
		if b == nil {
			continue
		}
		if b.ID == buttonIDRenameDone {
			b.Enabled = nameReady
		}
		b.Visible = true
	}
}

func createModeName(mode int) string {
	switch mode {
	case 1:
		return "Hardcore"
	case 2:
		return "Creative"
	default:
		return "Survival"
	}
}

func createModeDescription(mode int) (string, string) {
	switch mode {
	case 1:
		return "Same as survival mode, locked at hardest", "difficulty, and one life only"
	case 2:
		return "Unlimited resources, free flying and", "destroy blocks instantly"
	default:
		return "Search for resources, crafting, gain", "levels, health and hunger"
	}
}

func normalizeCreateWorldTypeID(id int) int {
	switch id {
	case 0, 1, 2:
		return id
	default:
		return 0
	}
}

func createWorldTypeName(id int) string {
	switch normalizeCreateWorldTypeID(id) {
	case 1:
		return "Superflat"
	case 2:
		return "Large Biomes"
	default:
		return "Default"
	}
}

func createWorldTypeGenerator(id int) (name string, version int32) {
	switch normalizeCreateWorldTypeID(id) {
	case 1:
		return "flat", 0
	case 2:
		return "largeBiomes", 0
	default:
		return "default", 1
	}
}

func worldTypeIDFromGenerator(name string, version int64) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "flat":
		return 1
	case "largebiomes":
		return 2
	case "default_1_1":
		return 0
	case "default":
		if version == 0 {
			return 0
		}
		return 0
	default:
		return 0
	}
}

var illegalWorldNames = []string{
	"CON", "COM", "PRN", "AUX", "CLOCK$", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
}

func makeUseableFolderName(raw string, exists func(name string) bool) string {
	name := strings.TrimSpace(raw)
	replacer := strings.NewReplacer(
		"/", "_",
		"\n", "_",
		"\r", "_",
		"\t", "_",
		"\x00", "_",
		"\f", "_",
		"`", "_",
		"?", "_",
		"*", "_",
		"\\", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\"", "_",
		":", "_",
		".", "_",
	)
	name = replacer.Replace(name)
	if strings.TrimSpace(name) == "" {
		name = "World"
	}
	for _, illegal := range illegalWorldNames {
		if strings.EqualFold(name, illegal) {
			name = "_" + name + "_"
			break
		}
	}
	for exists(name) {
		name += "-"
	}
	return name
}

func (a *App) updateCreateFolderName() {
	a.createFolderName = makeUseableFolderName(a.createWorldName, func(name string) bool {
		if name == "" {
			return false
		}
		if _, err := os.Stat(name); err == nil {
			return true
		}
		return false
	})
	if a.createFolderName == "" {
		a.createFolderName = "World"
	}
}

func cycleCreateWorldType(id int) int {
	switch normalizeCreateWorldTypeID(id) {
	case 0:
		return 1
	case 1:
		return 2
	default:
		return 0
	}
}

func (a *App) cycleCreateGameMode() {
	switch a.createWorldMode {
	case 0:
		if !a.createCheatsTog {
			a.createAllowCheats = false
		}
		a.createWorldMode = 1
	case 1:
		if !a.createCheatsTog {
			a.createAllowCheats = true
		}
		a.createWorldMode = 2
	case 2:
		if !a.createCheatsTog {
			a.createAllowCheats = false
		}
		a.createWorldMode = 0
	default:
		a.createWorldMode = 0
	}
}

func (a *App) currentMenuButtons() []*guiButton {
	switch a.menuScreen {
	case menuScreenSingleplayer:
		return a.singleButtons
	case menuScreenMultiplayer:
		return a.multiButtons
	case menuScreenOptions:
		return a.optionButtons
	case menuScreenVideo:
		return a.videoButtons
	case menuScreenControls:
		return a.controlButtons
	case menuScreenKeyBindings:
		return a.keyBindButtons
	case menuScreenSounds:
		return a.soundButtons
	case menuScreenCreateWorld:
		return a.createButtons
	case menuScreenRenameWorld:
		return a.renameButtons
	default:
		return a.mainButtons
	}
}

func (a *App) handleMainMenuInput(leftMouse, rightMouse, middleMouse, enterPressed bool) bool {
	a.processMenuTextInput(enterPressed)

	if a.menuScreen == menuScreenKeyBindings {
		if a.tryCaptureKeyBindingFromMouse(
			leftMouse && !a.prevLeftMouse,
			rightMouse && !a.prevRightMouse,
			middleMouse && !a.prevMiddleMouse,
		) {
			return true
		}
	}

	if leftMouse && !a.prevLeftMouse {
		switch a.menuScreen {
		case menuScreenSingleplayer:
			if a.handleSingleWorldListClick(a.mouseX, a.mouseY) {
				a.updateSingleButtonsState()
				return true
			}
		case menuScreenCreateWorld:
			if a.handleCreateWorldFieldClick(a.mouseX, a.mouseY) {
				return true
			}
		case menuScreenRenameWorld:
			if a.handleRenameWorldFieldClick(a.mouseX, a.mouseY) {
				return true
			}
		}
	}

	if leftMouse && !a.prevLeftMouse {
		for _, b := range a.currentMenuButtons() {
			if b == nil || !b.Enabled || !b.contains(a.mouseX, a.mouseY) {
				continue
			}
			if a.soundVolume > 0 {
				audio.PlaySoundKey("random.click", a.soundVolume, 1.0)
			}
			return a.handleMenuButton(b.ID)
		}
	}
	return true
}

func (a *App) processMenuTextInput(enterPressed bool) {
	if a.menuScreen != menuScreenCreateWorld && a.menuScreen != menuScreenRenameWorld {
		a.activeTextField = textFieldNone
		a.typedRuneQueue = a.typedRuneQueue[:0]
		a.prevBacksp = false
		a.prevTab = false
		return
	}
	if a.menuScreen == menuScreenCreateWorld && !a.createMoreOptions && a.activeTextField == textFieldCreateSeed {
		a.activeTextField = textFieldCreateName
	}

	runes := a.consumeTypedRunes()
	for _, ch := range runes {
		a.appendTextFieldRune(ch)
	}

	backspace := a.window.GetKey(glfw.KeyBackspace) == glfw.Press
	if backspace && !a.prevBacksp {
		a.backspaceActiveTextField()
	}
	a.prevBacksp = backspace

	tabPressed := a.window.GetKey(glfw.KeyTab) == glfw.Press
	if tabPressed && !a.prevTab {
		a.cycleTextFieldFocus()
	}
	a.prevTab = tabPressed

	if enterPressed && !a.prevEnter {
		switch a.menuScreen {
		case menuScreenCreateWorld:
			_ = a.handleMenuButton(buttonIDCreateDone)
		case menuScreenRenameWorld:
			_ = a.handleMenuButton(buttonIDRenameDone)
		}
	}
}

func (a *App) appendTextFieldRune(ch rune) {
	if ch < 32 || ch == 127 {
		return
	}
	switch a.activeTextField {
	case textFieldCreateName:
		if len([]rune(a.createWorldName)) >= 32 {
			return
		}
		a.createWorldName += string(ch)
		a.updateCreateButtonsState()
	case textFieldCreateSeed:
		if len([]rune(a.createWorldSeed)) >= 64 {
			return
		}
		a.createWorldSeed += string(ch)
	case textFieldRenameName:
		if len([]rune(a.renameWorldName)) >= 32 {
			return
		}
		a.renameWorldName += string(ch)
		a.updateRenameButtonsState()
	}
}

func (a *App) backspaceActiveTextField() {
	switch a.activeTextField {
	case textFieldCreateName:
		a.createWorldName = trimLastRune(a.createWorldName)
		a.updateCreateButtonsState()
	case textFieldCreateSeed:
		a.createWorldSeed = trimLastRune(a.createWorldSeed)
	case textFieldRenameName:
		a.renameWorldName = trimLastRune(a.renameWorldName)
		a.updateRenameButtonsState()
	}
}

func (a *App) cycleTextFieldFocus() {
	switch a.menuScreen {
	case menuScreenCreateWorld:
		if !a.createMoreOptions {
			a.activeTextField = textFieldCreateName
			return
		}
		if a.activeTextField == textFieldCreateName {
			a.activeTextField = textFieldCreateSeed
		} else {
			a.activeTextField = textFieldCreateName
		}
	case menuScreenRenameWorld:
		a.activeTextField = textFieldRenameName
	}
}

func trimLastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

func (a *App) enqueueTypedRune(ch rune) {
	if !a.mainMenu {
		if a.chatInputOpen {
			a.typedRuneQueue = append(a.typedRuneQueue, ch)
		}
		return
	}
	if a.menuScreen == menuScreenCreateWorld || a.menuScreen == menuScreenRenameWorld {
		a.typedRuneQueue = append(a.typedRuneQueue, ch)
	}
}

func (a *App) consumeTypedRunes() []rune {
	if len(a.typedRuneQueue) == 0 {
		return nil
	}
	out := make([]rune, len(a.typedRuneQueue))
	copy(out, a.typedRuneQueue)
	a.typedRuneQueue = a.typedRuneQueue[:0]
	return out
}

func (a *App) handleCreateWorldFieldClick(mx, my int) bool {
	x1, y1, x2, y2 := a.createNameFieldRect()
	if !a.createMoreOptions && mx >= x1 && mx < x2 && my >= y1 && my < y2 {
		a.activeTextField = textFieldCreateName
		return true
	}
	x1, y1, x2, y2 = a.createSeedFieldRect()
	if a.createMoreOptions && mx >= x1 && mx < x2 && my >= y1 && my < y2 {
		a.activeTextField = textFieldCreateSeed
		return true
	}
	return false
}

func (a *App) handleRenameWorldFieldClick(mx, my int) bool {
	x1, y1, x2, y2 := a.renameFieldRect()
	if mx >= x1 && mx < x2 && my >= y1 && my < y2 {
		a.activeTextField = textFieldRenameName
		return true
	}
	return false
}

func (a *App) createNameFieldRect() (x1, y1, x2, y2 int) {
	w := a.uiWidth()
	return w/2 - 100, 60, w/2 + 100, 80
}

func (a *App) createSeedFieldRect() (x1, y1, x2, y2 int) {
	w := a.uiWidth()
	return w/2 - 100, 60, w/2 + 100, 80
}

func (a *App) renameFieldRect() (x1, y1, x2, y2 int) {
	w := a.uiWidth()
	return w/2 - 100, 60, w/2 + 100, 80
}

func (a *App) handleMenuButton(id int) bool {
	switch a.menuScreen {
	case menuScreenMain:
		switch id {
		case buttonIDMenuSingleplayer:
			a.menuScreen = menuScreenSingleplayer
			a.refreshSingleplayerWorlds()
		case buttonIDMenuMultiplayer:
			a.menuScreen = menuScreenMultiplayer
		case buttonIDMenuOnline:
			a.menuStatus = "Minecraft Realms is not implemented yet."
		case buttonIDMenuOptions:
			a.menuScreen = menuScreenOptions
			a.updateOptionButtonsState()
		case buttonIDMenuLanguage:
			a.menuStatus = "Language screen is not implemented yet."
		case buttonIDMenuQuit:
			return false
		}
	case menuScreenSingleplayer:
		switch id {
		case buttonIDMenuBack:
			a.menuScreen = menuScreenMain
			a.menuStatus = ""
		case buttonIDSinglePlay:
			a.enterWorldFromMenu()
		case buttonIDSingleCreate:
			a.openCreateWorldScreen()
		case buttonIDSingleDelete:
			a.deleteSelectedWorld()
		case buttonIDSingleRename:
			a.openRenameWorldScreen()
		case buttonIDSingleRecreate:
			a.openRecreateWorldScreen()
		}
	case menuScreenMultiplayer:
		switch id {
		case buttonIDMenuBack:
			a.menuScreen = menuScreenMain
			a.menuStatus = ""
		case buttonIDMultiJoin, buttonIDMultiDirect:
			a.enterWorldFromMenu()
		case buttonIDMultiAdd, buttonIDMultiEdit, buttonIDMultiDelete, buttonIDMultiRefresh:
			a.menuStatus = "Server list management is not implemented yet."
		}
	case menuScreenOptions:
		changed := false
		switch id {
		case buttonIDOptionDone:
			a.menuScreen = menuScreenMain
			a.menuStatus = ""
		case buttonIDOptionDifficulty:
			a.optionDifficulty = (a.optionDifficulty + 1) & 3
			changed = true
		case buttonIDOptionRDMinus:
			if !a.mainMenu && a.renderDistance < 3 {
				a.renderDistance++
				changed = true
			}
		case buttonIDOptionRDPlus:
			if !a.mainMenu && a.renderDistance > 0 {
				a.renderDistance--
				changed = true
			}
		case buttonIDOptionFOVMinus:
			if !a.mainMenu {
				a.fovSetting -= 0.05
				if a.fovSetting < 0.0 {
					a.fovSetting = 0.0
				}
				changed = true
			}
		case buttonIDOptionFOVPlus:
			if !a.mainMenu {
				a.fovSetting += 0.05
				if a.fovSetting > 1.0 {
					a.fovSetting = 1.0
				}
				changed = true
			}
		case buttonIDOptionViewBobbing:
			if !a.mainMenu {
				a.viewBobbing = !a.viewBobbing
				changed = true
			}
		case buttonIDOptionVideo:
			a.menuScreen = menuScreenVideo
			a.keyBindCapture = -1
			a.menuStatus = ""
		case buttonIDOptionLanguage:
			if a.mainMenu {
				a.menuStatus = "Language screen is not implemented yet."
			} else {
				a.guiScaleMode = (a.guiScaleMode + 1) % len(guiScaleModeNames)
				a.updateGUIMetrics()
				changed = true
			}
		case buttonIDOptionControls:
			if a.mainMenu {
				a.menuScreen = menuScreenControls
				a.keyBindCapture = -1
				a.menuStatus = ""
			} else {
				a.invertMouse = !a.invertMouse
				changed = true
			}
		case buttonIDOptionMusic:
			if a.mainMenu {
				a.menuScreen = menuScreenSounds
				a.keyBindCapture = -1
				a.menuStatus = ""
			} else {
				a.fancyGraphics = !a.fancyGraphics
				changed = true
			}
		case buttonIDOptionSnooper:
			if a.mainMenu {
				a.menuStatus = "Snooper Settings are not implemented yet."
			} else {
				a.cloudsEnabled = !a.cloudsEnabled
				changed = true
			}
		case buttonIDOptionSensMinus:
			if !a.mainMenu {
				a.mouseSens -= 0.02
				if a.mouseSens < 0.0 {
					a.mouseSens = 0.0
				}
				changed = true
			}
		case buttonIDOptionSensPlus:
			if !a.mainMenu {
				a.mouseSens += 0.02
				if a.mouseSens > 1.0 {
					a.mouseSens = 1.0
				}
				changed = true
			}
		case buttonIDOptionMultiplayer:
			a.menuStatus = "Multiplayer Settings screen is not implemented yet."
		case buttonIDOptionResource:
			a.menuStatus = "Resource Packs screen is not implemented yet."
		}
		a.updateOptionButtonsState()
		a.updateVideoButtonsState()
		a.updateControlButtonsState()
		a.updateKeyBindingButtonsState()
		a.updateSoundButtonsState()
		if changed {
			a.saveOptionsFile()
		}
	case menuScreenVideo:
		changed := false
		switch id {
		case buttonIDVideoDone:
			a.menuScreen = menuScreenOptions
			a.menuStatus = ""
		case buttonIDVideoGraphics:
			a.fancyGraphics = !a.fancyGraphics
			changed = true
		case buttonIDVideoClouds:
			a.cloudsEnabled = !a.cloudsEnabled
			changed = true
		case buttonIDVideoRDMinus:
			if a.renderDistance < 3 {
				a.renderDistance++
				changed = true
			}
		case buttonIDVideoRDPlus:
			if a.renderDistance > 0 {
				a.renderDistance--
				changed = true
			}
		case buttonIDVideoFOVMinus:
			a.fovSetting -= 0.05
			if a.fovSetting < 0.0 {
				a.fovSetting = 0.0
			}
			changed = true
		case buttonIDVideoFOVPlus:
			a.fovSetting += 0.05
			if a.fovSetting > 1.0 {
				a.fovSetting = 1.0
			}
			changed = true
		case buttonIDVideoFPS:
			a.limitFramerateMode = (a.limitFramerateMode + 1) % len(framerateModeNames)
			changed = true
		case buttonIDVideoGUIScale:
			a.guiScaleMode = (a.guiScaleMode + 1) % len(guiScaleModeNames)
			a.updateGUIMetrics()
			changed = true
		case buttonIDVideoViewBobbing:
			a.viewBobbing = !a.viewBobbing
			changed = true
		}
		a.updateOptionButtonsState()
		a.updateVideoButtonsState()
		a.updateControlButtonsState()
		a.updateKeyBindingButtonsState()
		a.updateSoundButtonsState()
		if changed {
			a.saveOptionsFile()
		}
	case menuScreenControls:
		changed := false
		switch id {
		case buttonIDControlDone:
			a.menuScreen = menuScreenOptions
			a.keyBindCapture = -1
			a.menuStatus = ""
		case buttonIDControlSensMinus:
			a.mouseSens -= 0.02
			if a.mouseSens < 0.0 {
				a.mouseSens = 0.0
			}
			changed = true
		case buttonIDControlSensPlus:
			a.mouseSens += 0.02
			if a.mouseSens > 1.0 {
				a.mouseSens = 1.0
			}
			changed = true
		case buttonIDControlInvert:
			a.invertMouse = !a.invertMouse
			changed = true
		case buttonIDControlKeybinds:
			a.menuScreen = menuScreenKeyBindings
			a.keyBindCapture = -1
			a.clearKeyPressQueue()
			a.menuStatus = ""
		case buttonIDControlTouchscreen:
			a.menuStatus = "Touchscreen mode is not available on desktop."
		}
		a.updateControlButtonsState()
		a.updateKeyBindingButtonsState()
		a.updateSoundButtonsState()
		if changed {
			a.saveOptionsFile()
		}
	case menuScreenKeyBindings:
		switch {
		case id == buttonIDKeybindDone:
			a.menuScreen = menuScreenControls
			a.keyBindCapture = -1
			a.clearKeyPressQueue()
			a.menuStatus = ""
		case id >= buttonIDKeybindBase:
			idx := id - buttonIDKeybindBase
			if idx >= 0 && idx < len(a.keyBindings) {
				a.keyBindCapture = idx
				a.clearKeyPressQueue()
				a.menuStatus = ""
			}
		}
		a.updateKeyBindingButtonsState()
	case menuScreenSounds:
		changed := false
		switch id {
		case buttonIDSoundDone:
			a.menuScreen = menuScreenOptions
			a.menuStatus = ""
		case buttonIDSoundMusicMinus:
			if a.musicVolume > 0.0 {
				a.musicVolume = clampFloat64(a.musicVolume-0.1, 0.0, 1.0)
				changed = true
			}
		case buttonIDSoundMusicPlus:
			if a.musicVolume < 1.0 {
				a.musicVolume = clampFloat64(a.musicVolume+0.1, 0.0, 1.0)
				changed = true
			}
		case buttonIDSoundSoundMinus:
			if a.soundVolume > 0.0 {
				a.soundVolume = clampFloat64(a.soundVolume-0.1, 0.0, 1.0)
				changed = true
			}
		case buttonIDSoundSoundPlus:
			if a.soundVolume < 1.0 {
				a.soundVolume = clampFloat64(a.soundVolume+0.1, 0.0, 1.0)
				changed = true
			}
		}
		a.updateSoundButtonsState()
		if changed {
			a.saveOptionsFile()
		}
	case menuScreenCreateWorld:
		switch id {
		case buttonIDCreateCancel:
			a.menuScreen = menuScreenSingleplayer
			a.activeTextField = textFieldNone
			a.typedRuneQueue = a.typedRuneQueue[:0]
			a.menuStatus = ""
		case buttonIDCreateDone:
			a.createWorldFromEditor()
		case buttonIDCreateMoreOptions:
			a.createMoreOptions = !a.createMoreOptions
			if a.createMoreOptions {
				a.activeTextField = textFieldCreateSeed
			} else {
				a.activeTextField = textFieldCreateName
			}
			a.updateCreateButtonsState()
		case buttonIDCreateGameMode:
			a.cycleCreateGameMode()
			a.updateCreateButtonsState()
		case buttonIDCreateMapFeatures:
			a.createMapFeature = !a.createMapFeature
			a.updateCreateButtonsState()
		case buttonIDCreateWorldType:
			a.createWorldTypeID = cycleCreateWorldType(a.createWorldTypeID)
			a.updateCreateButtonsState()
		case buttonIDCreateAllowCommands:
			a.createCheatsTog = true
			a.createAllowCheats = !a.createAllowCheats
			a.updateCreateButtonsState()
		case buttonIDCreateBonusChest:
			a.createBonusChest = !a.createBonusChest
			a.updateCreateButtonsState()
		case buttonIDCreateCustomize:
			a.menuStatus = "Superflat customization is not implemented yet."
		}
	case menuScreenRenameWorld:
		switch id {
		case buttonIDRenameCancel:
			a.menuScreen = menuScreenSingleplayer
			a.activeTextField = textFieldNone
			a.typedRuneQueue = a.typedRuneQueue[:0]
			a.menuStatus = ""
		case buttonIDRenameDone:
			a.renameWorldFromEditor()
		}
	}
	return true
}

func (a *App) enterWorldFromMenu() {
	if a.menuScreen == menuScreenSingleplayer {
		if a.selectedWorld < 0 || a.selectedWorld >= len(a.singleWorlds) {
			a.menuStatus = "Select a world first."
			return
		}
		world := a.singleWorlds[a.selectedWorld]
		if a.playWorldFn != nil {
			session, err := a.playWorldFn(world)
			if err != nil {
				a.menuStatus = fmt.Sprintf("Failed to load %s: %v", world, err)
				return
			}
			a.replaceSession(session)
		}
		a.activeWorld = world
	}
	if a.session == nil {
		a.menuStatus = "No active world session."
		return
	}
	a.mainMenu = false
	a.menuScreen = menuScreenMain
	a.menuStatus = ""
	a.firstMouse = true
	a.applyCursorMode()
}

func (a *App) refreshSingleplayerWorlds() {
	current := a.activeWorld
	if a.selectedWorld >= 0 && a.selectedWorld < len(a.singleWorlds) {
		current = a.singleWorlds[a.selectedWorld]
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		a.singleWorlds = nil
		a.selectedWorld = -1
		a.menuStatus = "Unable to read local worlds."
		a.updateSingleButtonsState()
		return
	}

	worlds := make([]string, 0, len(entries))
	worldMeta := make(map[string]singleWorldMeta, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		if isWorldDirectory(name) {
			worlds = append(worlds, name)
			worldMeta[name] = loadSingleWorldMeta(name)
		}
	}
	// Translation reference:
	// - net.minecraft.src.GuiSelectWorld#func_82387_i()
	// - net.minecraft.src.SaveFormatComparator (lastPlayed desc)
	sort.Slice(worlds, func(i, j int) bool {
		mi := worldMeta[worlds[i]]
		mj := worldMeta[worlds[j]]
		if mi.LastPlayedMS != mj.LastPlayedMS {
			return mi.LastPlayedMS > mj.LastPlayedMS
		}
		return strings.ToLower(worlds[i]) < strings.ToLower(worlds[j])
	})
	a.singleWorlds = worlds
	a.singleWorldMeta = worldMeta

	if len(a.singleWorlds) == 0 {
		a.selectedWorld = -1
	} else {
		a.selectedWorld = 0
		if current != "" {
			for i, v := range a.singleWorlds {
				if v == current {
					a.selectedWorld = i
					break
				}
			}
		}
	}
	a.updateSingleButtonsState()
}

func isWorldDirectory(name string) bool {
	if st, err := os.Stat(filepath.Join(name, "level.dat")); err == nil && !st.IsDir() {
		return true
	}
	if st, err := os.Stat(filepath.Join(name, "region")); err == nil && st.IsDir() {
		return true
	}
	return strings.HasPrefix(strings.ToLower(name), "world")
}

func worldLastModifiedText(name string) string {
	st, err := os.Stat(filepath.Join(name, "level.dat"))
	if err != nil {
		st, err = os.Stat(name)
		if err != nil {
			return "unknown"
		}
	}
	return st.ModTime().Format("1/2/06 3:04 PM")
}

func loadSingleWorldMeta(name string) singleWorldMeta {
	meta := singleWorldMeta{
		DirName:        name,
		DisplayName:    name,
		LastPlayedText: worldLastModifiedText(name),
		GameType:       0,
		Hardcore:       false,
	}

	levelPath := filepath.Join(name, "level.dat")
	f, err := os.Open(levelPath)
	if err != nil {
		return meta
	}
	defer f.Close()

	root, err := nbt.ReadCompressed(f)
	if err != nil || root == nil {
		return meta
	}

	data, ok := root.GetTag("Data").(*nbt.CompoundTag)
	if !ok || data == nil {
		return meta
	}

	if t, ok := data.GetTag("LevelName").(*nbt.StringTag); ok {
		s := strings.TrimSpace(t.Data)
		if s != "" {
			meta.DisplayName = s
		}
	}

	if ms, ok := nbtTagAsInt64(data.GetTag("LastPlayed")); ok && ms > 0 {
		meta.LastPlayedMS = ms
		meta.LastPlayedText = formatWorldLastPlayed(ms)
	}

	if gameType, ok := nbtTagAsInt64(data.GetTag("GameType")); ok {
		meta.GameType = int(gameType)
	}

	if hardcore, ok := data.GetTag("hardcore").(*nbt.ByteTag); ok {
		meta.Hardcore = hardcore.Data != 0
	}

	return meta
}

func nbtTagAsInt64(tag nbt.Tag) (int64, bool) {
	switch t := tag.(type) {
	case *nbt.ByteTag:
		return int64(t.Data), true
	case *nbt.ShortTag:
		return int64(t.Data), true
	case *nbt.IntTag:
		return int64(t.Data), true
	case *nbt.LongTag:
		return t.Data, true
	default:
		return 0, false
	}
}

func nbtTagAsBool(tag nbt.Tag) (bool, bool) {
	switch t := tag.(type) {
	case *nbt.ByteTag:
		return t.Data != 0, true
	case *nbt.ShortTag:
		return t.Data != 0, true
	case *nbt.IntTag:
		return t.Data != 0, true
	case *nbt.LongTag:
		return t.Data != 0, true
	default:
		return false, false
	}
}

func sourceDataString(data *nbt.CompoundTag, key string) string {
	if data == nil {
		return ""
	}
	if t, ok := data.GetTag(key).(*nbt.StringTag); ok {
		return t.Data
	}
	return ""
}

func sourceDataInt64(data *nbt.CompoundTag, key string) int64 {
	if data == nil {
		return 0
	}
	v, _ := nbtTagAsInt64(data.GetTag(key))
	return v
}

func parseCreateWorldSeed(seedInput string) (int64, bool) {
	seed := strings.TrimSpace(seedInput)
	if seed == "" {
		return 0, false
	}

	if parsed, err := strconv.ParseInt(seed, 10, 64); err == nil {
		if parsed != 0 {
			return parsed, true
		}
		return currentMillis(), true
	}

	hash := int64(javaStringHashCode(seed))
	if hash == 0 {
		hash = currentMillis()
	}
	return hash, true
}

func javaStringHashCode(s string) int32 {
	var h int32
	for _, u := range utf16.Encode([]rune(s)) {
		h = h*31 + int32(u)
	}
	return h
}

func formatWorldLastPlayed(ms int64) string {
	return time.Unix(0, ms*int64(time.Millisecond)).Local().Format("1/2/06 3:04 PM")
}

func (a *App) openCreateWorldScreen() {
	a.recreateSource = ""
	a.createWorldName = a.nextUniqueWorldDisplayName("New World")
	a.createWorldSeed = ""
	a.createWorldMode = 0
	a.createMapFeature = true
	a.createAllowCheats = false
	a.createCheatsTog = false
	a.createBonusChest = false
	a.createWorldTypeID = 0
	a.createGeneratorOp = ""
	a.createMoreOptions = false
	a.menuScreen = menuScreenCreateWorld
	a.activeTextField = textFieldCreateName
	a.typedRuneQueue = a.typedRuneQueue[:0]
	a.prevBacksp = false
	a.prevTab = false
	a.menuStatus = ""
	a.updateCreateButtonsState()
}

func (a *App) openRenameWorldScreen() {
	worldDir, ok := a.selectedWorldDir()
	if !ok {
		return
	}
	meta := a.singleWorldMeta[worldDir]
	displayName := strings.TrimSpace(meta.DisplayName)
	if displayName == "" {
		displayName = worldDir
	}
	a.renameWorldDir = worldDir
	a.renameWorldName = displayName
	a.menuScreen = menuScreenRenameWorld
	a.activeTextField = textFieldRenameName
	a.typedRuneQueue = a.typedRuneQueue[:0]
	a.prevBacksp = false
	a.prevTab = false
	a.menuStatus = ""
	a.updateRenameButtonsState()
}

func (a *App) openRecreateWorldScreen() {
	sourceWorld, ok := a.selectedWorldDir()
	if !ok {
		return
	}
	sourceMeta := a.singleWorldMeta[sourceWorld]
	baseName := strings.TrimSpace(sourceMeta.DisplayName)
	if baseName == "" {
		baseName = sourceWorld
	}

	_, sourceData, err := loadOrCreateLevelData(sourceWorld, baseName, sourceMeta.GameType, sourceMeta.Hardcore)
	if err != nil {
		a.menuStatus = fmt.Sprintf("Re-create world failed: %v", err)
		return
	}

	a.recreateSource = sourceWorld
	a.createWorldName = a.nextUniqueWorldDisplayName("Copy of " + baseName)
	a.createWorldSeed = ""
	if seed, ok := nbtTagAsInt64(sourceData.GetTag("RandomSeed")); ok {
		a.createWorldSeed = strconv.FormatInt(seed, 10)
	}
	a.createWorldTypeID = worldTypeIDFromGenerator(sourceDataString(sourceData, "generatorName"), sourceDataInt64(sourceData, "generatorVersion"))
	a.createGeneratorOp = sourceDataString(sourceData, "generatorOptions")
	a.createMapFeature = true
	if v, ok := nbtTagAsBool(sourceData.GetTag("MapFeatures")); ok {
		a.createMapFeature = v
	}
	a.createAllowCheats = false
	if v, ok := nbtTagAsBool(sourceData.GetTag("allowCommands")); ok {
		a.createAllowCheats = v
	}
	a.createCheatsTog = false
	a.createBonusChest = false
	switch {
	case sourceMeta.Hardcore:
		a.createWorldMode = 1
	case sourceMeta.GameType == 1:
		a.createWorldMode = 2
	default:
		a.createWorldMode = 0
	}
	a.createMoreOptions = false
	a.menuScreen = menuScreenCreateWorld
	a.activeTextField = textFieldCreateName
	a.typedRuneQueue = a.typedRuneQueue[:0]
	a.prevBacksp = false
	a.prevTab = false
	a.menuStatus = "Re-create world: settings copied."
	a.updateCreateButtonsState()
}

func (a *App) createWorldFromEditor() {
	displayName := sanitizeDisplayName(a.createWorldName, "New World")
	worldDir := makeUseableFolderName(displayName, func(name string) bool {
		if name == "" {
			return false
		}
		if _, err := os.Stat(name); err == nil {
			return true
		}
		return false
	})

	gameType := 0
	hardcore := false
	switch a.createWorldMode {
	case 1:
		hardcore = true
	case 2:
		gameType = 1
	}

	var templateData *nbt.CompoundTag
	if a.recreateSource != "" {
		recreateMeta := a.singleWorldMeta[a.recreateSource]
		templateName := strings.TrimSpace(recreateMeta.DisplayName)
		if templateName == "" {
			templateName = a.recreateSource
		}
		_, sourceData, err := loadOrCreateLevelData(a.recreateSource, templateName, recreateMeta.GameType, recreateMeta.Hardcore)
		if err != nil {
			a.menuStatus = fmt.Sprintf("Create world failed: %v", err)
			return
		}
		templateData = sourceData
	}

	if err := os.MkdirAll(filepath.Join(worldDir, "region"), 0o755); err != nil {
		a.menuStatus = fmt.Sprintf("Create world failed: %v", err)
		return
	}

	root := nbt.NewCompoundTag("")
	data := defaultLevelData(displayName, gameType, hardcore, templateData)
	data.SetBoolean("MapFeatures", a.createMapFeature)
	data.SetBoolean("allowCommands", a.createAllowCheats && !hardcore)
	generatorName, generatorVersion := createWorldTypeGenerator(a.createWorldTypeID)
	data.SetString("generatorName", generatorName)
	data.SetInteger("generatorVersion", generatorVersion)
	data.SetString("generatorOptions", a.createGeneratorOp)
	if seed, ok := parseCreateWorldSeed(a.createWorldSeed); ok {
		data.SetLong("RandomSeed", seed)
	}
	data.SetLong("LastPlayed", currentMillis())
	root.SetTag("Data", data)
	if err := writeLevelDat(worldDir, root); err != nil {
		a.menuStatus = fmt.Sprintf("Create world failed: %v", err)
		return
	}

	a.refreshSingleplayerWorlds()
	a.selectWorldByDir(worldDir)
	a.updateSingleButtonsState()
	a.menuStatus = fmt.Sprintf("Created world %s.", displayName)
	a.activeTextField = textFieldNone
	a.recreateSource = ""
	a.menuScreen = menuScreenSingleplayer
	a.enterWorldFromMenu()
}

func (a *App) renameWorldFromEditor() {
	worldDir := a.renameWorldDir
	if worldDir == "" {
		a.menuStatus = "Rename world failed: no world selected."
		return
	}

	meta := a.singleWorldMeta[worldDir]
	baseName := strings.TrimSpace(meta.DisplayName)
	if baseName == "" {
		baseName = worldDir
	}
	newName := sanitizeDisplayName(a.renameWorldName, baseName)
	root, data, err := loadOrCreateLevelData(worldDir, baseName, meta.GameType, meta.Hardcore)
	if err != nil {
		a.menuStatus = fmt.Sprintf("Rename world failed: %v", err)
		return
	}
	data.SetString("LevelName", newName)
	data.SetLong("LastPlayed", currentMillis())
	root.SetTag("Data", data)
	if err := writeLevelDat(worldDir, root); err != nil {
		a.menuStatus = fmt.Sprintf("Rename world failed: %v", err)
		return
	}

	a.refreshSingleplayerWorlds()
	a.selectWorldByDir(worldDir)
	a.menuScreen = menuScreenSingleplayer
	a.activeTextField = textFieldNone
	a.menuStatus = fmt.Sprintf("Renamed world to %s.", newName)
}

func (a *App) selectedWorldDir() (string, bool) {
	if a.selectedWorld < 0 || a.selectedWorld >= len(a.singleWorlds) {
		return "", false
	}
	return a.singleWorlds[a.selectedWorld], true
}

func (a *App) selectWorldByDir(worldDir string) {
	if worldDir == "" {
		return
	}
	for i, v := range a.singleWorlds {
		if v == worldDir {
			a.selectedWorld = i
			return
		}
	}
}

func (a *App) nextWorldDirName() string {
	for i := 1; i <= 9999; i++ {
		candidate := fmt.Sprintf("World%d", i)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return fmt.Sprintf("World%d", time.Now().Unix()%100000)
}

func (a *App) nextUniqueWorldDisplayName(base string) string {
	base = sanitizeDisplayName(base, "New World")
	if !a.worldDisplayNameExists(base) {
		return base
	}
	for i := 1; i <= 9999; i++ {
		candidate := sanitizeDisplayName(fmt.Sprintf("%s (%d)", base, i), base)
		if !a.worldDisplayNameExists(candidate) {
			return candidate
		}
	}
	return sanitizeDisplayName(fmt.Sprintf("%s %d", base, time.Now().Unix()%100000), base)
}

func (a *App) worldDisplayNameExists(displayName string) bool {
	for worldDir, meta := range a.singleWorldMeta {
		if strings.EqualFold(worldDir, displayName) || strings.EqualFold(meta.DisplayName, displayName) {
			return true
		}
	}
	return false
}

func (a *App) deleteSelectedWorld() {
	worldDir, ok := a.selectedWorldDir()
	if !ok {
		return
	}
	if worldDir == "" || strings.Contains(worldDir, "..") {
		a.menuStatus = "Invalid world path."
		return
	}
	if err := os.RemoveAll(worldDir); err != nil {
		a.menuStatus = fmt.Sprintf("Delete world failed: %v", err)
		return
	}
	if a.activeWorld == worldDir {
		a.activeWorld = ""
	}
	a.refreshSingleplayerWorlds()
	a.menuStatus = fmt.Sprintf("Deleted world %s.", worldDir)
}

func loadOrCreateLevelData(worldDir, fallbackLevelName string, gameType int, hardcore bool) (*nbt.CompoundTag, *nbt.CompoundTag, error) {
	levelPath := filepath.Join(worldDir, "level.dat")
	f, err := os.Open(levelPath)
	if err == nil {
		defer f.Close()
		root, readErr := nbt.ReadCompressed(f)
		if readErr != nil {
			return nil, nil, readErr
		}
		if root == nil {
			root = nbt.NewCompoundTag("")
		}
		if data, ok := root.GetTag("Data").(*nbt.CompoundTag); ok && data != nil {
			return root, data, nil
		}
		data := defaultLevelData(fallbackLevelName, gameType, hardcore, nil)
		root.SetTag("Data", data)
		return root, data, nil
	}
	if !os.IsNotExist(err) {
		return nil, nil, err
	}

	root := nbt.NewCompoundTag("")
	data := defaultLevelData(fallbackLevelName, gameType, hardcore, nil)
	root.SetTag("Data", data)
	return root, data, nil
}

func defaultLevelData(levelName string, gameType int, hardcore bool, template *nbt.CompoundTag) *nbt.CompoundTag {
	nowMS := currentMillis()
	var data *nbt.CompoundTag
	if template != nil {
		if copied, ok := template.Copy().(*nbt.CompoundTag); ok {
			data = copied
		}
	}
	if data == nil {
		data = nbt.NewCompoundTag("Data")
	}

	levelName = sanitizeDisplayName(levelName, "New World")
	// Translation reference:
	// - net.minecraft.src.WorldInfo (Data compound fields in level.dat)
	data.SetString("LevelName", levelName)
	data.SetInteger("GameType", int32(gameType))
	data.SetBoolean("hardcore", hardcore)
	data.SetBoolean("MapFeatures", true)

	if _, ok := data.GetTag("generatorName").(*nbt.StringTag); !ok {
		data.SetString("generatorName", "default")
	}
	if _, ok := data.GetTag("generatorVersion").(*nbt.IntTag); !ok {
		data.SetInteger("generatorVersion", 1)
	}
	if _, ok := data.GetTag("RandomSeed").(*nbt.LongTag); !ok {
		data.SetLong("RandomSeed", nowMS)
	}

	data.SetLong("Time", 0)
	data.SetLong("DayTime", 0)
	data.SetLong("LastPlayed", nowMS)
	data.SetInteger("version", 19133)
	if _, ok := data.GetTag("SpawnX").(*nbt.IntTag); !ok {
		data.SetInteger("SpawnX", 0)
	}
	if _, ok := data.GetTag("SpawnY").(*nbt.IntTag); !ok {
		data.SetInteger("SpawnY", 64)
	}
	if _, ok := data.GetTag("SpawnZ").(*nbt.IntTag); !ok {
		data.SetInteger("SpawnZ", 0)
	}
	data.RemoveTag("Player")
	return data
}

func writeLevelDat(worldDir string, root *nbt.CompoundTag) error {
	if err := os.MkdirAll(worldDir, 0o755); err != nil {
		return err
	}
	levelPath := filepath.Join(worldDir, "level.dat")
	tempPath := levelPath + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	if err := nbt.WriteCompressed(root, f); err != nil {
		_ = f.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	_ = os.Remove(levelPath)
	return os.Rename(tempPath, levelPath)
}

func sanitizeDisplayName(name, fallback string) string {
	out := strings.TrimSpace(name)
	if out == "" {
		out = fallback
	}
	if out == "" {
		out = "World"
	}
	runes := []rune(out)
	if len(runes) > 32 {
		out = string(runes[:32])
	}
	return out
}

func currentMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func (a *App) singleListRect() (x1, y1, x2, y2 int) {
	w, h := a.uiWidth(), a.uiHeight()
	return w/2 - 110, 32, w/2 + 110, h - 64
}

func (a *App) handleSingleWorldListClick(mx, my int) bool {
	x1, y1, x2, y2 := a.singleListRect()
	if mx < x1 || mx >= x2 || my < y1 || my >= y2 {
		return false
	}
	if len(a.singleWorlds) == 0 {
		return false
	}
	rowH := 36
	row := (my - (y1 + 4)) / rowH
	if row < 0 || row >= len(a.singleWorlds) {
		return false
	}
	a.selectedWorld = row
	name := a.singleWorlds[row]
	meta, ok := a.singleWorldMeta[name]
	if !ok {
		meta = loadSingleWorldMeta(name)
		a.singleWorldMeta[name] = meta
	}
	displayName := meta.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = name
	}
	a.menuStatus = fmt.Sprintf("Selected world: %s", displayName)
	return true
}

func (a *App) drawMenuScreen() {
	switch a.menuScreen {
	case menuScreenSingleplayer:
		a.drawSingleplayerMenu()
	case menuScreenMultiplayer:
		a.drawMultiplayerMenu()
	case menuScreenOptions:
		a.drawOptionsMenu()
	case menuScreenVideo:
		a.drawVideoMenu()
	case menuScreenControls:
		a.drawControlsMenu()
	case menuScreenKeyBindings:
		a.drawKeyBindingsMenu()
	case menuScreenSounds:
		a.drawSoundsMenu()
	case menuScreenCreateWorld:
		a.drawCreateWorldMenu()
	case menuScreenRenameWorld:
		a.drawRenameWorldMenu()
	default:
		a.drawMainMenu()
	}
}

func (a *App) handleMenuEscape() bool {
	switch a.menuScreen {
	case menuScreenCreateWorld, menuScreenRenameWorld:
		a.menuScreen = menuScreenSingleplayer
		a.activeTextField = textFieldNone
		a.typedRuneQueue = a.typedRuneQueue[:0]
		a.menuStatus = ""
		return true
	case menuScreenVideo:
		a.menuScreen = menuScreenOptions
		a.menuStatus = ""
		return true
	case menuScreenControls:
		a.menuScreen = menuScreenOptions
		a.keyBindCapture = -1
		a.menuStatus = ""
		return true
	case menuScreenKeyBindings:
		a.menuScreen = menuScreenControls
		a.keyBindCapture = -1
		a.clearKeyPressQueue()
		a.menuStatus = ""
		return true
	case menuScreenSounds:
		a.menuScreen = menuScreenOptions
		a.keyBindCapture = -1
		a.menuStatus = ""
		return true
	case menuScreenMain:
		return false
	default:
		a.menuScreen = menuScreenMain
		a.menuStatus = ""
		return true
	}
}

func (a *App) drawMenuBackground(title string) {
	w, h := a.uiWidth(), a.uiHeight()
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	begin2D(w, h)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	if a.texOptionsBG != nil {
		// Translation reference:
		// - net.minecraft.src.GuiScreen.drawBackground()
		a.drawTiledOptionsBackground(0, 0, w, h, 0x404040)
	} else {
		drawSolidRect(0, 0, w, h, 0xFF202020)
	}

	if a.font != nil {
		a.font.drawCenteredString(title, w/2, 20, 0xFFFFFF)
	}
}

func (a *App) drawTiledOptionsBackground(x1, y1, x2, y2 int, rgb int) {
	if a.texOptionsBG == nil {
		drawSolidRect(x1, y1, x2, y2, 0xFF202020)
		return
	}
	r := float32((rgb>>16)&0xFF) / 255.0
	g := float32((rgb>>8)&0xFF) / 255.0
	b := float32(rgb&0xFF) / 255.0

	gl.Enable(gl.TEXTURE_2D)
	// Translation reference:
	// - net.minecraft.src.GuiScreen.drawBackground()
	// - net.minecraft.src.GuiSlot.overlayBackground()
	// Vanilla backgrounds use UV scaled by 32.0F with repeating wrap.
	a.texOptionsBG.setWrapRepeat(true)
	gl.Color4f(r, g, b, 1)
	u0 := float32(x1) / 32.0
	v0 := float32(y1) / 32.0
	u1 := float32(x2) / 32.0
	v1 := float32(y2) / 32.0
	drawTexturedRectUV(a.texOptionsBG, float32(x1), float32(y1), float32(x2-x1), float32(y2-y1), u0, v0, u1, v1)
	gl.Color4f(1, 1, 1, 1)
}

// Translation reference:
// - net.minecraft.src.GuiSlot.drawScreen()
func (a *App) drawSlotMenuBackground(title string) {
	w, h := a.uiWidth(), a.uiHeight()
	top, bottom := 32, h-64

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	begin2D(w, h)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// GuiSlot base background uses options_background tinted 0x202020.
	a.drawTiledOptionsBackground(0, 0, w, h, 0x202020)

	// GuiSlot overlayBackground() top/bottom with 0x404040.
	a.drawTiledOptionsBackground(0, 0, w, top, 0x404040)
	a.drawTiledOptionsBackground(0, bottom, w, h, 0x404040)

	// GuiSlot 4px fade masks on top/bottom.
	drawGradientRect(0, top, w, top+4, 0xFF000000, 0x00000000)
	drawGradientRect(0, bottom-4, w, bottom, 0x00000000, 0xFF000000)

	if a.font != nil {
		a.font.drawCenteredString(title, w/2, 20, 0xFFFFFF)
	}
}

func (a *App) drawMenuStatusLine() {
	if a.font == nil || a.menuStatus == "" {
		return
	}
	a.font.drawCenteredString(a.menuStatus, a.uiWidth()/2, a.uiHeight()-14, 0xE0E0E0)
}

func (a *App) drawSingleplayerMenu() {
	a.drawSlotMenuBackground("Select world")

	x1, y1, x2, y2 := a.singleListRect()

	if a.font != nil {
		if len(a.singleWorlds) == 0 {
			a.font.drawCenteredString("No worlds found.", a.uiWidth()/2, y1+20, 0xAAAAAA)
		} else {
			rowH := 36
			for i, name := range a.singleWorlds {
				rowY := y1 + 4 + i*rowH
				if rowY+rowH > y2 {
					break
				}
				if i == a.selectedWorld {
					// Translation reference:
					// - net.minecraft.src.GuiSlot selection box colors.
					drawSolidRect(x1, rowY+34, x2, rowY-2, 0xFF808080)
					drawSolidRect(x1+1, rowY+33, x2-1, rowY-1, 0xFF000000)
				}

				meta, ok := a.singleWorldMeta[name]
				if !ok {
					meta = loadSingleWorldMeta(name)
					a.singleWorldMeta[name] = meta
				}
				worldTitle := meta.DisplayName
				if strings.TrimSpace(worldTitle) == "" {
					worldTitle = name
				}
				worldInfo := fmt.Sprintf("%s (%s)", name, meta.LastPlayedText)
				worldMode := meta.modeText()
				titleColor := 0xFFFFFF
				modeColor := 0x808080
				if meta.Hardcore {
					modeColor = 0xFF5555
				}
				if i == a.selectedWorld {
					titleColor = 0xFFFFA0
				}
				a.font.drawString(worldTitle, x1+2, rowY+1, titleColor)
				a.font.drawString(worldInfo, x1+2, rowY+12, 0x808080)
				a.font.drawString(worldMode, x1+2, rowY+22, modeColor)
			}
		}
	}

	for _, b := range a.singleButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}
	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawCreateWorldMenu() {
	a.drawMenuBackground("Create New World")
	a.updateCreateButtonsState()

	nameRectX1, nameRectY1, nameRectX2, nameRectY2 := a.createNameFieldRect()
	seedRectX1, seedRectY1, seedRectX2, seedRectY2 := a.createSeedFieldRect()
	lineColor := 0xA0A0A0
	showSeedField := a.createMoreOptions

	if a.font != nil {
		if a.createMoreOptions {
			a.font.drawString("Seed for the World Generator", seedRectX1, 47, lineColor)
			a.font.drawString("Leave blank for a random seed", seedRectX1, 85, lineColor)
			a.font.drawString("Villages, dungeons etc", a.uiWidth()/2-150, 122, lineColor)
			a.font.drawString("Commands like /gamemode, /xp", a.uiWidth()/2-150, 172, lineColor)
		} else {
			line1, line2 := createModeDescription(a.createWorldMode)
			a.font.drawString("World Name", nameRectX1, 47, lineColor)
			a.font.drawString("Will be saved in: "+a.createFolderName, nameRectX1, 85, lineColor)
			a.font.drawString(line1, a.uiWidth()/2-100, 137, lineColor)
			a.font.drawString(line2, a.uiWidth()/2-100, 149, lineColor)
		}
	}
	if showSeedField {
		a.drawTextField(seedRectX1, seedRectY1, seedRectX2, seedRectY2, a.createWorldSeed, a.activeTextField == textFieldCreateSeed)
	} else {
		a.drawTextField(nameRectX1, nameRectY1, nameRectX2, nameRectY2, a.createWorldName, a.activeTextField == textFieldCreateName)
	}

	for _, b := range a.createButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}
	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawRenameWorldMenu() {
	a.drawMenuBackground("Rename World")
	x1, y1, x2, y2 := a.renameFieldRect()
	if a.font != nil {
		a.updateRenameButtonsState()
		a.font.drawString("Enter Name", x1, 47, 0xA0A0A0)
	}
	a.drawTextField(x1, y1, x2, y2, a.renameWorldName, a.activeTextField == textFieldRenameName)
	for _, b := range a.renameButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}
	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawTextField(x1, y1, x2, y2 int, text string, active bool) {
	border := 0xFF707070
	if active {
		border = 0xFFA0A0A0
	}
	drawSolidRect(x1-1, y1-1, x2+1, y2+1, border)
	drawSolidRect(x1, y1, x2, y2, 0xFF000000)
	if a.font == nil {
		return
	}
	content := text
	if active && (time.Now().UnixMilli()/300)%2 == 0 {
		content += "_"
	}
	a.font.drawString(content, x1+4, y1+6, 0xE0E0E0)
}

func (a *App) drawMultiplayerMenu() {
	a.drawSlotMenuBackground("Play Multiplayer")

	panelX1 := a.uiWidth()/2 - 110
	panelY1 := 32
	panelX2 := a.uiWidth()/2 + 110
	panelY2 := a.uiHeight() - 64
	_ = panelY2

	if a.font != nil {
		rowY := panelY1 + 4
		drawSolidRect(panelX1, rowY+34, panelX2, rowY-2, 0xFF808080)
		drawSolidRect(panelX1+1, rowY+33, panelX2-1, rowY-1, 0xFF000000)
		a.font.drawString("GoMC Local Server", panelX1+2, rowY+1, 0xFFFFFF)
		a.font.drawString("Local offline-compatible test server", panelX1+2, rowY+12, 0x808080)
		a.font.drawString("127.0.0.1:25565", panelX1+2, rowY+23, 0x303030)
	}

	for _, b := range a.multiButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}
	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawOptionsMenu() {
	a.drawMenuBackground("Options")
	a.updateOptionButtonsState()

	for _, b := range a.optionButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}

	if !a.mainMenu && a.font != nil {
		baseY := a.uiHeight()/6 + 12
		rd := a.optionRenderDistanceLabel()
		fov := a.optionFOVLabel()
		sens := fmt.Sprintf("Sensitivity: %d%%", a.sensitivityPercent())
		a.font.drawCenteredString(rd, a.uiWidth()/2, baseY+88+6, 0xFFFFFF)
		a.font.drawCenteredString(fov, a.uiWidth()/2, baseY+112+6, 0xFFFFFF)
		a.font.drawCenteredString(sens, a.uiWidth()/2, baseY+136+6, 0xFFFFFF)
	}

	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawControlsMenu() {
	a.drawMenuBackground("Controls")
	a.updateControlButtonsState()

	for _, b := range a.controlButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}

	if a.font != nil {
		baseY := a.uiHeight()/6 + 20
		sens := fmt.Sprintf("Sensitivity: %d%%", a.sensitivityPercent())
		a.font.drawCenteredString(sens, a.uiWidth()/2, baseY+24+6, 0xFFFFFF)
	}

	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawVideoMenu() {
	a.drawMenuBackground("Video Settings")
	a.updateVideoButtonsState()

	for _, b := range a.videoButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}

	if a.font != nil {
		baseY := a.uiHeight()/6 + 20
		rd := a.optionRenderDistanceLabel()
		fov := a.optionFOVLabel()
		a.font.drawCenteredString(rd, a.uiWidth()/2, baseY+24+6, 0xFFFFFF)
		a.font.drawCenteredString(fov, a.uiWidth()/2, baseY+48+6, 0xFFFFFF)
	}

	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawSoundsMenu() {
	a.drawMenuBackground("Music & Sounds")
	a.updateSoundButtonsState()

	for _, b := range a.soundButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}

	if a.font != nil {
		baseY := a.uiHeight()/6 + 20
		a.font.drawCenteredString(a.optionMusicVolumeLabel(), a.uiWidth()/2, baseY+6, 0xFFFFFF)
		a.font.drawCenteredString(a.optionSoundVolumeLabel(), a.uiWidth()/2, baseY+24+6, 0xFFFFFF)
	}

	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}

func (a *App) drawKeyBindingLabels() {
	if a.font == nil {
		return
	}
	for i := range a.keyBindings {
		if i >= len(a.keyBindButtons) {
			break
		}
		b := a.keyBindButtons[i]
		if b == nil || !b.Visible {
			continue
		}
		a.font.drawString(a.keyBindings[i].Label, b.X+b.Width+6, b.Y+7, 0xFFFFFF)
	}
}

func (a *App) drawKeyBindingsMenu() {
	a.drawMenuBackground("Controls")
	a.updateKeyBindingButtonsState()

	for _, b := range a.keyBindButtons {
		b.draw(a.font, a.texWidgets, a.mouseX, a.mouseY)
	}
	a.drawKeyBindingLabels()

	a.drawMenuStatusLine()
	gl.Disable(gl.BLEND)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
}
