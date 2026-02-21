package server

import (
	"testing"

	"github.com/lulaide/gomc/pkg/network/protocol"
)

func TestPlayerDataStateRoundTrip(t *testing.T) {
	srv := NewStatusServer(StatusConfig{
		PersistWorld: true,
		WorldDir:     t.TempDir(),
	})

	state := defaultPersistedPlayerState()
	state.X = 12.25
	state.Y = 66.0
	state.Z = -3.5
	state.Yaw = 45
	state.Pitch = 12
	state.OnGround = true
	state.Health = 18.5
	state.Food = 16
	state.Sat = 2.5
	state.FoodExhaust = 7.25
	state.FoodTickTimer = 42
	state.GameType = 1
	state.HeldSlot = 2
	state.Inventory[38] = &protocol.ItemStack{
		ItemID:     1,
		StackSize:  7,
		ItemDamage: 3,
	}

	if err := srv.savePlayerState("Steve", state); err != nil {
		t.Fatalf("savePlayerState failed: %v", err)
	}

	loaded, ok := srv.loadPlayerState("Steve")
	if !ok {
		t.Fatal("expected persisted player state to load")
	}
	if loaded.X != state.X || loaded.Y != state.Y || loaded.Z != state.Z {
		t.Fatalf("position mismatch: got=(%f,%f,%f) want=(%f,%f,%f)", loaded.X, loaded.Y, loaded.Z, state.X, state.Y, state.Z)
	}
	if loaded.Food != state.Food || loaded.Sat != state.Sat || loaded.FoodExhaust != state.FoodExhaust || loaded.FoodTickTimer != state.FoodTickTimer {
		t.Fatalf("food state mismatch: got=(%d,%f,%f,%d) want=(%d,%f,%f,%d)",
			loaded.Food, loaded.Sat, loaded.FoodExhaust, loaded.FoodTickTimer,
			state.Food, state.Sat, state.FoodExhaust, state.FoodTickTimer)
	}
	if loaded.GameType != 1 || loaded.HeldSlot != 2 {
		t.Fatalf("mode/slot mismatch: got=(%d,%d)", loaded.GameType, loaded.HeldSlot)
	}
	if loaded.Inventory[38] == nil || loaded.Inventory[38].ItemID != 1 || loaded.Inventory[38].StackSize != 7 || loaded.Inventory[38].ItemDamage != 3 {
		t.Fatalf("inventory slot mismatch: %#v", loaded.Inventory[38])
	}
}
