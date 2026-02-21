package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

var (
	otoInitOnce sync.Once
	otoCtx      *oto.Context
	otoReady    <-chan struct{}
	otoInitErr  error
)

const (
	otoSampleRate   = 44100
	otoChannels     = 2
	otoSampleFormat = oto.FormatSignedInt16LE
)

func initOto() {
	otoCtx, otoReady, otoInitErr = oto.NewContext(otoSampleRate, otoChannels, otoSampleFormat)
}

func ensureOtoReady() bool {
	otoInitOnce.Do(initOto)
	if otoInitErr != nil || otoCtx == nil {
		return false
	}
	select {
	case <-otoReady:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

func playTonePCM(freqHz, durationMs int) bool {
	if freqHz <= 0 || durationMs <= 0 {
		return false
	}
	data := synthTonePCM(freqHz, durationMs)
	return playPCMBytes(data, 0.85)
}

func playPCMBytes(data []byte, volume float64) bool {
	if len(data) == 0 {
		return false
	}
	if !ensureOtoReady() {
		return false
	}
	player := otoCtx.NewPlayer(bytes.NewReader(data))
	if player == nil {
		return false
	}
	if volume <= 0 {
		volume = 1
	}
	player.SetVolume(volume)
	player.Play()
	waitMs := len(data) * 1000 / (otoSampleRate * otoChannels * 2)
	if waitMs < 1 {
		waitMs = 1
	}
	go func(p oto.Player, ms int) {
		time.Sleep(time.Duration(ms+250) * time.Millisecond)
		_ = p.Close()
	}(player, waitMs)
	return true
}

func synthTonePCM(freqHz, durationMs int) []byte {
	totalSamples := otoSampleRate * durationMs / 1000
	if totalSamples <= 0 {
		return nil
	}
	const amp = 0.45
	const fadeMs = 3
	fadeSamples := otoSampleRate * fadeMs / 1000
	if fadeSamples < 1 {
		fadeSamples = 1
	}

	buf := make([]byte, totalSamples*otoChannels*2)
	for i := 0; i < totalSamples; i++ {
		t := float64(i) / float64(otoSampleRate)
		env := 1.0
		if i < fadeSamples {
			env = float64(i) / float64(fadeSamples)
		} else if remain := totalSamples - i; remain < fadeSamples {
			env = float64(remain) / float64(fadeSamples)
		}
		v := math.Sin(2*math.Pi*float64(freqHz)*t) * amp * env
		sample := int16(v * 32767.0)

		off := i * otoChannels * 2
		binary.LittleEndian.PutUint16(buf[off:], uint16(sample))
		binary.LittleEndian.PutUint16(buf[off+2:], uint16(sample))
	}
	return buf
}
