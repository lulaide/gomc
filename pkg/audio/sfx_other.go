//go:build !windows

package audio

import "fmt"

func playToneFallback(_ int, _ int) bool {
	// Fallback when platform-native tone is unavailable.
	fmt.Print("\a")
	return true
}
