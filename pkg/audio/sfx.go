package audio

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var lastPlayUnixMs atomic.Int64
var audioDiagOnce sync.Once

func Init() {
	InitWithAssets("")
}

func InitWithAssets(assetsRoot string) {
	initSoundLibrary(assetsRoot)
	if msg := soundLibraryDiagnostic(); msg != "" {
		fmt.Println(msg)
	}
	go ensureOtoReady()
}

func PlayDigBlock(blockID int) {
	if blockID <= 0 {
		return
	}
	keys, vol, pitch := digSoundForBlock(blockID)
	playKeys(keys, vol, pitch)
}

func PlaySwing() {
	// Vanilla 1.6.4 does not play a generic local "swing whoosh" for empty attack.
	// Keep silent here; entity/block sounds come from their own actions/events.
}

func PlayPlaceHeldItem(itemID int) {
	if itemID <= 0 || itemID > 255 {
		return
	}
	keys, vol, pitch := placeSoundForBlock(itemID)
	playKeys(keys, vol, pitch)
}

func PlaySoundKey(key string, volume, pitch float64) {
	playKeys([]string{key}, volume, pitch)
}

func playKeys(keys []string, volume, pitch float64) {
	if len(keys) == 0 {
		return
	}
	now := time.Now().UnixMilli()
	prev := lastPlayUnixMs.Load()
	if now-prev < 10 {
		return
	}
	lastPlayUnixMs.Store(now)

	if playSoundKeys(keys, volume, pitch) {
		return
	}
	audioDiagOnce.Do(func() {
		if msg := soundLibraryDiagnostic(); msg != "" {
			fmt.Println(msg)
		} else {
			fmt.Printf("audio: no mapped ogg sample for keys=%v\n", keys)
		}
	})
}
