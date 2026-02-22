//go:build cgo

package gui

import (
	"fmt"

	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	keyDescAttack    = "key.attack"
	keyDescUse       = "key.use"
	keyDescForward   = "key.forward"
	keyDescLeft      = "key.left"
	keyDescBack      = "key.back"
	keyDescRight     = "key.right"
	keyDescJump      = "key.jump"
	keyDescSneak     = "key.sneak"
	keyDescDrop      = "key.drop"
	keyDescInventory = "key.inventory"
	keyDescChat      = "key.chat"
	keyDescPlayer    = "key.playerlist"
	keyDescPick      = "key.pickItem"
	keyDescCommand   = "key.command"
)

type keyBindingConfig struct {
	Description string
	Label       string
	DefaultCode int
	Code        int
}

func defaultKeyBindings() []keyBindingConfig {
	// Translation reference:
	// - net.minecraft.src.GameSettings keyBind* defaults
	return []keyBindingConfig{
		{Description: keyDescAttack, Label: "Attack", DefaultCode: -100, Code: -100},
		{Description: keyDescUse, Label: "Use Item", DefaultCode: -99, Code: -99},
		{Description: keyDescForward, Label: "Forward", DefaultCode: 17, Code: 17},
		{Description: keyDescLeft, Label: "Left", DefaultCode: 30, Code: 30},
		{Description: keyDescBack, Label: "Back", DefaultCode: 31, Code: 31},
		{Description: keyDescRight, Label: "Right", DefaultCode: 32, Code: 32},
		{Description: keyDescJump, Label: "Jump", DefaultCode: 57, Code: 57},
		{Description: keyDescSneak, Label: "Sneak", DefaultCode: 42, Code: 42},
		{Description: keyDescDrop, Label: "Drop", DefaultCode: 16, Code: 16},
		{Description: keyDescInventory, Label: "Inventory", DefaultCode: 18, Code: 18},
		{Description: keyDescChat, Label: "Chat", DefaultCode: 20, Code: 20},
		{Description: keyDescPlayer, Label: "Player List", DefaultCode: 15, Code: 15},
		{Description: keyDescPick, Label: "Pick Block", DefaultCode: -98, Code: -98},
		{Description: keyDescCommand, Label: "Command", DefaultCode: 53, Code: 53},
	}
}

func (a *App) initDefaultKeyBindings() {
	a.keyBindings = defaultKeyBindings()
	a.keyBindCapture = -1
}

func (a *App) keyBindingIndexByDescription(desc string) int {
	for i := range a.keyBindings {
		if a.keyBindings[i].Description == desc {
			return i
		}
	}
	return -1
}

func (a *App) keyBindingCode(desc string) int {
	idx := a.keyBindingIndexByDescription(desc)
	if idx < 0 {
		return 0
	}
	return a.keyBindings[idx].Code
}

func (a *App) isKeyBindingDown(desc string) bool {
	return a.isMCKeyDown(a.keyBindingCode(desc))
}

func (a *App) isMCKeyDown(code int) bool {
	if a.window == nil {
		return false
	}
	if code < 0 {
		button := -100 - code
		if button < 0 || button > 7 {
			return false
		}
		return a.window.GetMouseButton(glfw.MouseButton(button)) == glfw.Press
	}
	key, ok := mcKeyCodeToGLFWKey(code)
	if !ok {
		return false
	}
	return a.window.GetKey(key) == glfw.Press
}

func (a *App) setKeyBindingByIndex(index, mcCode int) {
	if index < 0 || index >= len(a.keyBindings) {
		return
	}
	a.keyBindings[index].Code = mcCode
}

func (a *App) keyBindingHasConflict(index int) bool {
	if index < 0 || index >= len(a.keyBindings) {
		return false
	}
	code := a.keyBindings[index].Code
	for i := range a.keyBindings {
		if i == index {
			continue
		}
		if a.keyBindings[i].Code == code {
			return true
		}
	}
	return false
}

func (a *App) keyBindingButtonLabel(index int) string {
	if index < 0 || index >= len(a.keyBindings) {
		return ""
	}
	name := keyDisplayName(a.keyBindings[index].Code)
	if a.keyBindCapture == index {
		return "\u00a7f> \u00a7e??? \u00a7f<"
	}
	if a.keyBindingHasConflict(index) {
		return "\u00a7c" + name
	}
	return name
}

func (a *App) isCapturingKeyBinding() bool {
	return a.keyBindCapture >= 0 && a.keyBindCapture < len(a.keyBindings)
}

func (a *App) enqueueKeyPress(key glfw.Key) {
	a.keyPressQueue = append(a.keyPressQueue, key)
}

func (a *App) clearKeyPressQueue() {
	a.keyPressQueue = a.keyPressQueue[:0]
}

func (a *App) consumeKeyPress() (glfw.Key, bool) {
	if len(a.keyPressQueue) == 0 {
		return 0, false
	}
	key := a.keyPressQueue[0]
	a.keyPressQueue = a.keyPressQueue[1:]
	return key, true
}

func (a *App) tryCaptureKeyBindingFromKeyQueue() bool {
	if !a.isCapturingKeyBinding() {
		return false
	}
	for {
		key, ok := a.consumeKeyPress()
		if !ok {
			return false
		}
		mcCode, mapped := glfwKeyToMCKeyCode(key)
		if !mapped {
			continue
		}
		a.setKeyBindingByIndex(a.keyBindCapture, mcCode)
		a.keyBindCapture = -1
		a.updateKeyBindingButtonsState()
		a.saveOptionsFile()
		return true
	}
}

func (a *App) tryCaptureKeyBindingFromMouse(leftClick, rightClick, middleClick bool) bool {
	if !a.isCapturingKeyBinding() {
		return false
	}
	button := -1
	if leftClick {
		button = 0
	} else if rightClick {
		button = 1
	} else if middleClick {
		button = 2
	}
	if button < 0 {
		return false
	}
	a.setKeyBindingByIndex(a.keyBindCapture, -100+button)
	a.keyBindCapture = -1
	a.updateKeyBindingButtonsState()
	a.saveOptionsFile()
	return true
}

// Translation reference:
// - net.minecraft.src.GameSettings.getKeyDisplayString(int)
// - org.lwjgl.input.Keyboard.getKeyName(int)
func keyDisplayName(code int) string {
	if code < 0 {
		return fmt.Sprintf("Button %d", code+101)
	}
	switch code {
	case 0:
		return "NONE"
	case 1:
		return "ESCAPE"
	case 12:
		return "MINUS"
	case 13:
		return "EQUALS"
	case 14:
		return "BACK"
	case 15:
		return "TAB"
	case 28:
		return "RETURN"
	case 29:
		return "LCONTROL"
	case 39:
		return "SEMICOLON"
	case 40:
		return "APOSTROPHE"
	case 41:
		return "GRAVE"
	case 42:
		return "LSHIFT"
	case 43:
		return "BACKSLASH"
	case 52:
		return "PERIOD"
	case 53:
		return "SLASH"
	case 54:
		return "RSHIFT"
	case 55:
		return "MULTIPLY"
	case 56:
		return "LMENU"
	case 57:
		return "SPACE"
	case 58:
		return "CAPITAL"
	case 69:
		return "NUMLOCK"
	case 70:
		return "SCROLL"
	case 71:
		return "NUMPAD7"
	case 72:
		return "NUMPAD8"
	case 73:
		return "NUMPAD9"
	case 74:
		return "SUBTRACT"
	case 75:
		return "NUMPAD4"
	case 76:
		return "NUMPAD5"
	case 77:
		return "NUMPAD6"
	case 78:
		return "ADD"
	case 79:
		return "NUMPAD1"
	case 80:
		return "NUMPAD2"
	case 81:
		return "NUMPAD3"
	case 82:
		return "NUMPAD0"
	case 83:
		return "DECIMAL"
	case 87:
		return "F11"
	case 88:
		return "F12"
	case 100:
		return "F13"
	case 101:
		return "F14"
	case 102:
		return "F15"
	case 156:
		return "NUMPADENTER"
	case 157:
		return "RCONTROL"
	case 181:
		return "DIVIDE"
	case 184:
		return "RMENU"
	case 197:
		return "PAUSE"
	case 199:
		return "HOME"
	case 200:
		return "UP"
	case 201:
		return "PRIOR"
	case 203:
		return "LEFT"
	case 205:
		return "RIGHT"
	case 207:
		return "END"
	case 208:
		return "DOWN"
	case 209:
		return "NEXT"
	case 210:
		return "INSERT"
	case 211:
		return "DELETE"
	case 219:
		return "LMETA"
	case 220:
		return "RMETA"
	case 221:
		return "APPS"
	}
	if code >= 2 && code <= 11 {
		chars := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}
		return chars[code-2]
	}
	if code >= 16 && code <= 25 {
		chars := []string{"Q", "W", "E", "R", "T", "Y", "U", "I", "O", "P"}
		return chars[code-16]
	}
	if code >= 30 && code <= 38 {
		chars := []string{"A", "S", "D", "F", "G", "H", "J", "K", "L"}
		return chars[code-30]
	}
	if code >= 44 && code <= 50 {
		chars := []string{"Z", "X", "C", "V", "B", "N", "M"}
		return chars[code-44]
	}
	if code >= 59 && code <= 68 {
		return fmt.Sprintf("F%d", code-58)
	}
	return fmt.Sprintf("KEY%d", code)
}

func mcKeyCodeToGLFWKey(code int) (glfw.Key, bool) {
	switch code {
	case 1:
		return glfw.KeyEscape, true
	case 2:
		return glfw.Key1, true
	case 3:
		return glfw.Key2, true
	case 4:
		return glfw.Key3, true
	case 5:
		return glfw.Key4, true
	case 6:
		return glfw.Key5, true
	case 7:
		return glfw.Key6, true
	case 8:
		return glfw.Key7, true
	case 9:
		return glfw.Key8, true
	case 10:
		return glfw.Key9, true
	case 11:
		return glfw.Key0, true
	case 12:
		return glfw.KeyMinus, true
	case 13:
		return glfw.KeyEqual, true
	case 14:
		return glfw.KeyBackspace, true
	case 15:
		return glfw.KeyTab, true
	case 16:
		return glfw.KeyQ, true
	case 17:
		return glfw.KeyW, true
	case 18:
		return glfw.KeyE, true
	case 19:
		return glfw.KeyR, true
	case 20:
		return glfw.KeyT, true
	case 21:
		return glfw.KeyY, true
	case 22:
		return glfw.KeyU, true
	case 23:
		return glfw.KeyI, true
	case 24:
		return glfw.KeyO, true
	case 25:
		return glfw.KeyP, true
	case 26:
		return glfw.KeyLeftBracket, true
	case 27:
		return glfw.KeyRightBracket, true
	case 28:
		return glfw.KeyEnter, true
	case 29:
		return glfw.KeyLeftControl, true
	case 30:
		return glfw.KeyA, true
	case 31:
		return glfw.KeyS, true
	case 32:
		return glfw.KeyD, true
	case 33:
		return glfw.KeyF, true
	case 34:
		return glfw.KeyG, true
	case 35:
		return glfw.KeyH, true
	case 36:
		return glfw.KeyJ, true
	case 37:
		return glfw.KeyK, true
	case 38:
		return glfw.KeyL, true
	case 39:
		return glfw.KeySemicolon, true
	case 40:
		return glfw.KeyApostrophe, true
	case 41:
		return glfw.KeyGraveAccent, true
	case 42:
		return glfw.KeyLeftShift, true
	case 43:
		return glfw.KeyBackslash, true
	case 44:
		return glfw.KeyZ, true
	case 45:
		return glfw.KeyX, true
	case 46:
		return glfw.KeyC, true
	case 47:
		return glfw.KeyV, true
	case 48:
		return glfw.KeyB, true
	case 49:
		return glfw.KeyN, true
	case 50:
		return glfw.KeyM, true
	case 51:
		return glfw.KeyComma, true
	case 52:
		return glfw.KeyPeriod, true
	case 53:
		return glfw.KeySlash, true
	case 54:
		return glfw.KeyRightShift, true
	case 55:
		return glfw.KeyKPMultiply, true
	case 56:
		return glfw.KeyLeftAlt, true
	case 57:
		return glfw.KeySpace, true
	case 58:
		return glfw.KeyCapsLock, true
	case 59:
		return glfw.KeyF1, true
	case 60:
		return glfw.KeyF2, true
	case 61:
		return glfw.KeyF3, true
	case 62:
		return glfw.KeyF4, true
	case 63:
		return glfw.KeyF5, true
	case 64:
		return glfw.KeyF6, true
	case 65:
		return glfw.KeyF7, true
	case 66:
		return glfw.KeyF8, true
	case 67:
		return glfw.KeyF9, true
	case 68:
		return glfw.KeyF10, true
	case 69:
		return glfw.KeyNumLock, true
	case 70:
		return glfw.KeyScrollLock, true
	case 71:
		return glfw.KeyKP7, true
	case 72:
		return glfw.KeyKP8, true
	case 73:
		return glfw.KeyKP9, true
	case 74:
		return glfw.KeyKPSubtract, true
	case 75:
		return glfw.KeyKP4, true
	case 76:
		return glfw.KeyKP5, true
	case 77:
		return glfw.KeyKP6, true
	case 78:
		return glfw.KeyKPAdd, true
	case 79:
		return glfw.KeyKP1, true
	case 80:
		return glfw.KeyKP2, true
	case 81:
		return glfw.KeyKP3, true
	case 82:
		return glfw.KeyKP0, true
	case 83:
		return glfw.KeyKPDecimal, true
	case 87:
		return glfw.KeyF11, true
	case 88:
		return glfw.KeyF12, true
	case 100:
		return glfw.KeyF13, true
	case 101:
		return glfw.KeyF14, true
	case 102:
		return glfw.KeyF15, true
	case 156:
		return glfw.KeyKPEnter, true
	case 157:
		return glfw.KeyRightControl, true
	case 181:
		return glfw.KeyKPDivide, true
	case 184:
		return glfw.KeyRightAlt, true
	case 197:
		return glfw.KeyPause, true
	case 199:
		return glfw.KeyHome, true
	case 200:
		return glfw.KeyUp, true
	case 201:
		return glfw.KeyPageUp, true
	case 203:
		return glfw.KeyLeft, true
	case 205:
		return glfw.KeyRight, true
	case 207:
		return glfw.KeyEnd, true
	case 208:
		return glfw.KeyDown, true
	case 209:
		return glfw.KeyPageDown, true
	case 210:
		return glfw.KeyInsert, true
	case 211:
		return glfw.KeyDelete, true
	case 219:
		return glfw.KeyLeftSuper, true
	case 220:
		return glfw.KeyRightSuper, true
	case 221:
		return glfw.KeyMenu, true
	default:
		return 0, false
	}
}

func glfwKeyToMCKeyCode(key glfw.Key) (int, bool) {
	switch key {
	case glfw.KeyEscape:
		return 1, true
	case glfw.Key1:
		return 2, true
	case glfw.Key2:
		return 3, true
	case glfw.Key3:
		return 4, true
	case glfw.Key4:
		return 5, true
	case glfw.Key5:
		return 6, true
	case glfw.Key6:
		return 7, true
	case glfw.Key7:
		return 8, true
	case glfw.Key8:
		return 9, true
	case glfw.Key9:
		return 10, true
	case glfw.Key0:
		return 11, true
	case glfw.KeyMinus:
		return 12, true
	case glfw.KeyEqual:
		return 13, true
	case glfw.KeyBackspace:
		return 14, true
	case glfw.KeyTab:
		return 15, true
	case glfw.KeyQ:
		return 16, true
	case glfw.KeyW:
		return 17, true
	case glfw.KeyE:
		return 18, true
	case glfw.KeyR:
		return 19, true
	case glfw.KeyT:
		return 20, true
	case glfw.KeyY:
		return 21, true
	case glfw.KeyU:
		return 22, true
	case glfw.KeyI:
		return 23, true
	case glfw.KeyO:
		return 24, true
	case glfw.KeyP:
		return 25, true
	case glfw.KeyLeftBracket:
		return 26, true
	case glfw.KeyRightBracket:
		return 27, true
	case glfw.KeyEnter:
		return 28, true
	case glfw.KeyLeftControl:
		return 29, true
	case glfw.KeyA:
		return 30, true
	case glfw.KeyS:
		return 31, true
	case glfw.KeyD:
		return 32, true
	case glfw.KeyF:
		return 33, true
	case glfw.KeyG:
		return 34, true
	case glfw.KeyH:
		return 35, true
	case glfw.KeyJ:
		return 36, true
	case glfw.KeyK:
		return 37, true
	case glfw.KeyL:
		return 38, true
	case glfw.KeySemicolon:
		return 39, true
	case glfw.KeyApostrophe:
		return 40, true
	case glfw.KeyGraveAccent:
		return 41, true
	case glfw.KeyLeftShift:
		return 42, true
	case glfw.KeyBackslash:
		return 43, true
	case glfw.KeyZ:
		return 44, true
	case glfw.KeyX:
		return 45, true
	case glfw.KeyC:
		return 46, true
	case glfw.KeyV:
		return 47, true
	case glfw.KeyB:
		return 48, true
	case glfw.KeyN:
		return 49, true
	case glfw.KeyM:
		return 50, true
	case glfw.KeyComma:
		return 51, true
	case glfw.KeyPeriod:
		return 52, true
	case glfw.KeySlash:
		return 53, true
	case glfw.KeyRightShift:
		return 54, true
	case glfw.KeyKPMultiply:
		return 55, true
	case glfw.KeyLeftAlt:
		return 56, true
	case glfw.KeySpace:
		return 57, true
	case glfw.KeyCapsLock:
		return 58, true
	case glfw.KeyF1:
		return 59, true
	case glfw.KeyF2:
		return 60, true
	case glfw.KeyF3:
		return 61, true
	case glfw.KeyF4:
		return 62, true
	case glfw.KeyF5:
		return 63, true
	case glfw.KeyF6:
		return 64, true
	case glfw.KeyF7:
		return 65, true
	case glfw.KeyF8:
		return 66, true
	case glfw.KeyF9:
		return 67, true
	case glfw.KeyF10:
		return 68, true
	case glfw.KeyNumLock:
		return 69, true
	case glfw.KeyScrollLock:
		return 70, true
	case glfw.KeyKP7:
		return 71, true
	case glfw.KeyKP8:
		return 72, true
	case glfw.KeyKP9:
		return 73, true
	case glfw.KeyKPSubtract:
		return 74, true
	case glfw.KeyKP4:
		return 75, true
	case glfw.KeyKP5:
		return 76, true
	case glfw.KeyKP6:
		return 77, true
	case glfw.KeyKPAdd:
		return 78, true
	case glfw.KeyKP1:
		return 79, true
	case glfw.KeyKP2:
		return 80, true
	case glfw.KeyKP3:
		return 81, true
	case glfw.KeyKP0:
		return 82, true
	case glfw.KeyKPDecimal:
		return 83, true
	case glfw.KeyF11:
		return 87, true
	case glfw.KeyF12:
		return 88, true
	case glfw.KeyF13:
		return 100, true
	case glfw.KeyF14:
		return 101, true
	case glfw.KeyF15:
		return 102, true
	case glfw.KeyKPEnter:
		return 156, true
	case glfw.KeyRightControl:
		return 157, true
	case glfw.KeyKPDivide:
		return 181, true
	case glfw.KeyRightAlt:
		return 184, true
	case glfw.KeyPause:
		return 197, true
	case glfw.KeyHome:
		return 199, true
	case glfw.KeyUp:
		return 200, true
	case glfw.KeyPageUp:
		return 201, true
	case glfw.KeyLeft:
		return 203, true
	case glfw.KeyRight:
		return 205, true
	case glfw.KeyEnd:
		return 207, true
	case glfw.KeyDown:
		return 208, true
	case glfw.KeyPageDown:
		return 209, true
	case glfw.KeyInsert:
		return 210, true
	case glfw.KeyDelete:
		return 211, true
	case glfw.KeyLeftSuper:
		return 219, true
	case glfw.KeyRightSuper:
		return 220, true
	case glfw.KeyMenu:
		return 221, true
	default:
		return 0, false
	}
}
