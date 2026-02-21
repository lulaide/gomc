//go:build windows

package audio

import "syscall"

var (
	kernel32DLL     = syscall.NewLazyDLL("kernel32.dll")
	beepProc        = kernel32DLL.NewProc("Beep")
	user32DLL       = syscall.NewLazyDLL("user32.dll")
	messageBeepProc = user32DLL.NewProc("MessageBeep")
)

func playToneFallback(freqHz, durationMs int) bool {
	if freqHz < 37 || freqHz > 32767 {
		freqHz = 750
	}
	if durationMs <= 0 {
		durationMs = 80
	}

	if err := beepProc.Find(); err == nil {
		if r, _, _ := beepProc.Call(uintptr(freqHz), uintptr(durationMs)); r != 0 {
			return true
		}
	}
	if err := messageBeepProc.Find(); err == nil {
		if r, _, _ := messageBeepProc.Call(uintptr(0xFFFFFFFF)); r != 0 {
			return true
		}
	}
	return false
}
