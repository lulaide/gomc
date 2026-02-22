//go:build cgo

package gui

import (
	"testing"
	"time"
)

func TestHeldItemUseProfile(t *testing.T) {
	tests := []struct {
		itemID   int16
		damage   int16
		wantOK   bool
		wantAct  itemUseAction
		wantMax  int
		wantAE   bool
		testName string
	}{
		{itemID: 267, damage: 0, wantOK: true, wantAct: itemUseActionBlock, wantMax: 72000, testName: "sword"},
		{itemID: 261, damage: 0, wantOK: true, wantAct: itemUseActionBow, wantMax: 72000, testName: "bow"},
		{itemID: 373, damage: 0, wantOK: true, wantAct: itemUseActionDrink, wantMax: 32, testName: "potion-drink"},
		{itemID: 373, damage: 0x4000, wantOK: false, testName: "potion-splash"},
		{itemID: 322, damage: 0, wantOK: true, wantAct: itemUseActionEat, wantMax: 32, wantAE: true, testName: "golden-apple"},
		{itemID: 1, damage: 0, wantOK: false, testName: "stone-block"},
	}
	for _, tc := range tests {
		got, ok := heldItemUseProfile(tc.itemID, tc.damage)
		if ok != tc.wantOK {
			t.Fatalf("%s ok mismatch: got=%t want=%t", tc.testName, ok, tc.wantOK)
		}
		if !ok {
			continue
		}
		if got.Action != tc.wantAct || got.MaxUseDuration != tc.wantMax || got.AlwaysEdible != tc.wantAE {
			t.Fatalf("%s profile mismatch: got=%+v", tc.testName, got)
		}
	}
}

func TestLocalUseRemainingTicks(t *testing.T) {
	a := &App{
		localUsingItem: true,
		localUseStart:  time.Now().Add(-500 * time.Millisecond),
		localUseMax:    32,
	}
	remaining := a.localUseRemainingTicks(time.Now())
	if remaining < 21 || remaining > 22 {
		t.Fatalf("remaining mismatch: got=%d want about 22", remaining)
	}
}

func TestBowTextureNameForDrawTicks(t *testing.T) {
	tests := []struct {
		drawTicks int
		want      string
	}{
		{drawTicks: 0, want: "bow_standby"},
		{drawTicks: 1, want: "bow_pulling_0"},
		{drawTicks: 14, want: "bow_pulling_1"},
		{drawTicks: 18, want: "bow_pulling_2"},
	}
	for _, tc := range tests {
		got := bowTextureNameForDrawTicks(tc.drawTicks)
		if got != tc.want {
			t.Fatalf("drawTicks=%d texture mismatch: got=%q want=%q", tc.drawTicks, got, tc.want)
		}
	}
}
