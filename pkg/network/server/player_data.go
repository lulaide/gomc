package server

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lulaide/gomc/pkg/nbt"
	"github.com/lulaide/gomc/pkg/network/protocol"
)

type persistedPlayerState struct {
	X        float64
	Y        float64
	Z        float64
	Yaw      float32
	Pitch    float32
	OnGround bool

	Health        float32
	Food          int16
	Sat           float32
	FoodExhaust   float32
	FoodTickTimer int
	Experience    float32
	ExperienceLvl int32
	ExperienceTot int32

	GameType    int8
	HeldSlot    int16
	HasSpawn    bool
	SpawnX      int32
	SpawnY      int32
	SpawnZ      int32
	SpawnForced bool

	Inventory [playerWindowSlots]*protocol.ItemStack
}

func defaultPersistedPlayerState() persistedPlayerState {
	return persistedPlayerState{
		X:             defaultSpawnX,
		Y:             defaultSpawnY,
		Z:             defaultSpawnZ,
		Yaw:           0,
		Pitch:         0,
		OnGround:      false,
		Health:        20.0,
		Food:          20,
		Sat:           5.0,
		FoodExhaust:   0,
		FoodTickTimer: 0,
		Experience:    0,
		ExperienceLvl: 0,
		ExperienceTot: 0,
		GameType:      0,
		HeldSlot:      0,
		HasSpawn:      false,
		SpawnX:        0,
		SpawnY:        0,
		SpawnZ:        0,
		SpawnForced:   false,
	}
}

func cloneInventoryArray(in [playerWindowSlots]*protocol.ItemStack) [playerWindowSlots]*protocol.ItemStack {
	var out [playerWindowSlots]*protocol.ItemStack
	for i := 0; i < playerWindowSlots; i++ {
		out[i] = cloneItemStack(in[i])
	}
	return out
}

func sanitizePlayerFileName(name string) string {
	if name == "" {
		return "player"
	}
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "player"
	}
	return b.String()
}

func (s *StatusServer) playerDataPath(username string) string {
	worldDir := s.cfg.WorldDir
	if worldDir == "" {
		worldDir = "world"
	}
	return filepath.Join(worldDir, "playerdata", sanitizePlayerFileName(username)+".dat")
}

func tagAsFloat32(tag nbt.Tag, def float32) float32 {
	switch t := tag.(type) {
	case *nbt.FloatTag:
		return t.Data
	case *nbt.DoubleTag:
		return float32(t.Data)
	case *nbt.IntTag:
		return float32(t.Data)
	case *nbt.ShortTag:
		return float32(t.Data)
	case *nbt.ByteTag:
		return float32(t.Data)
	default:
		return def
	}
}

func tagAsFloat64(tag nbt.Tag, def float64) float64 {
	switch t := tag.(type) {
	case *nbt.DoubleTag:
		return t.Data
	case *nbt.FloatTag:
		return float64(t.Data)
	case *nbt.LongTag:
		return float64(t.Data)
	case *nbt.IntTag:
		return float64(t.Data)
	case *nbt.ShortTag:
		return float64(t.Data)
	case *nbt.ByteTag:
		return float64(t.Data)
	default:
		return def
	}
}

func tagAsInt32(tag nbt.Tag, def int32) int32 {
	switch t := tag.(type) {
	case *nbt.IntTag:
		return t.Data
	case *nbt.ShortTag:
		return int32(t.Data)
	case *nbt.ByteTag:
		return int32(t.Data)
	case *nbt.LongTag:
		return int32(t.Data)
	default:
		return def
	}
}

func tagAsBoolean(tag nbt.Tag, def bool) bool {
	switch t := tag.(type) {
	case *nbt.ByteTag:
		return t.Data != 0
	default:
		return def
	}
}

func stateToNBT(state persistedPlayerState) *nbt.CompoundTag {
	root := nbt.NewCompoundTag("")

	pos := nbt.NewListTag("Pos")
	pos.AppendTag(nbt.NewDoubleTag("", state.X))
	pos.AppendTag(nbt.NewDoubleTag("", state.Y))
	pos.AppendTag(nbt.NewDoubleTag("", state.Z))
	root.SetTag("Pos", pos)

	rot := nbt.NewListTag("Rotation")
	rot.AppendTag(nbt.NewFloatTag("", state.Yaw))
	rot.AppendTag(nbt.NewFloatTag("", state.Pitch))
	root.SetTag("Rotation", rot)

	root.SetBoolean("OnGround", state.OnGround)
	root.SetFloat("Health", state.Health)
	root.SetInteger("foodLevel", int32(state.Food))
	root.SetInteger("foodTickTimer", int32(state.FoodTickTimer))
	root.SetFloat("foodSaturationLevel", state.Sat)
	root.SetFloat("foodExhaustionLevel", state.FoodExhaust)
	root.SetFloat("XpP", state.Experience)
	root.SetInteger("XpLevel", state.ExperienceLvl)
	root.SetInteger("XpTotal", state.ExperienceTot)
	root.SetInteger("playerGameType", int32(state.GameType))
	root.SetInteger("SelectedItemSlot", int32(state.HeldSlot))
	if state.HasSpawn {
		root.SetInteger("SpawnX", state.SpawnX)
		root.SetInteger("SpawnY", state.SpawnY)
		root.SetInteger("SpawnZ", state.SpawnZ)
		root.SetBoolean("SpawnForced", state.SpawnForced)
	}

	inv := nbt.NewListTag("Inventory")
	for slot := 0; slot < playerWindowSlots; slot++ {
		stack := state.Inventory[slot]
		if stack == nil || stack.StackSize <= 0 {
			continue
		}
		item := nbt.NewCompoundTag("")
		item.SetByte("Slot", int8(slot))
		item.SetShort("id", stack.ItemID)
		item.SetByte("Count", stack.StackSize)
		item.SetShort("Damage", stack.ItemDamage)
		if stack.Tag != nil {
			if copied, ok := stack.Tag.Copy().(*nbt.CompoundTag); ok {
				item.SetTag("tag", copied)
			}
		}
		inv.AppendTag(item)
	}
	root.SetTag("Inventory", inv)
	return root
}

func stateFromNBT(root *nbt.CompoundTag) persistedPlayerState {
	state := defaultPersistedPlayerState()
	if root == nil {
		return state
	}

	if tag, ok := root.GetTag("Pos").(*nbt.ListTag); ok && tag.TagCount() >= 3 {
		state.X = tagAsFloat64(tag.TagAt(0), state.X)
		state.Y = tagAsFloat64(tag.TagAt(1), state.Y)
		state.Z = tagAsFloat64(tag.TagAt(2), state.Z)
	}
	if tag, ok := root.GetTag("Rotation").(*nbt.ListTag); ok && tag.TagCount() >= 2 {
		state.Yaw = tagAsFloat32(tag.TagAt(0), state.Yaw)
		state.Pitch = tagAsFloat32(tag.TagAt(1), state.Pitch)
	}

	state.OnGround = tagAsBoolean(root.GetTag("OnGround"), state.OnGround)
	state.Health = tagAsFloat32(root.GetTag("Health"), state.Health)
	state.Food = int16(tagAsInt32(root.GetTag("foodLevel"), int32(state.Food)))
	state.FoodTickTimer = int(tagAsInt32(root.GetTag("foodTickTimer"), int32(state.FoodTickTimer)))
	state.Sat = tagAsFloat32(root.GetTag("foodSaturationLevel"), state.Sat)
	state.FoodExhaust = tagAsFloat32(root.GetTag("foodExhaustionLevel"), state.FoodExhaust)
	state.Experience = tagAsFloat32(root.GetTag("XpP"), state.Experience)
	state.ExperienceLvl = tagAsInt32(root.GetTag("XpLevel"), state.ExperienceLvl)
	state.ExperienceTot = tagAsInt32(root.GetTag("XpTotal"), state.ExperienceTot)
	state.GameType = int8(tagAsInt32(root.GetTag("playerGameType"), int32(state.GameType)))
	state.HeldSlot = int16(tagAsInt32(root.GetTag("SelectedItemSlot"), int32(state.HeldSlot)))
	if root.GetTag("SpawnX") != nil && root.GetTag("SpawnY") != nil && root.GetTag("SpawnZ") != nil {
		state.HasSpawn = true
		state.SpawnX = tagAsInt32(root.GetTag("SpawnX"), state.SpawnX)
		state.SpawnY = tagAsInt32(root.GetTag("SpawnY"), state.SpawnY)
		state.SpawnZ = tagAsInt32(root.GetTag("SpawnZ"), state.SpawnZ)
		state.SpawnForced = tagAsBoolean(root.GetTag("SpawnForced"), state.SpawnForced)
	}
	if state.HeldSlot < 0 || state.HeldSlot >= hotbarSlotCount {
		state.HeldSlot = 0
	}

	if inv, ok := root.GetTag("Inventory").(*nbt.ListTag); ok {
		for i := 0; i < inv.TagCount(); i++ {
			item, ok := inv.TagAt(i).(*nbt.CompoundTag)
			if !ok {
				continue
			}
			slot := int(tagAsInt32(item.GetTag("Slot"), -1))
			if slot < 0 || slot >= playerWindowSlots {
				continue
			}
			stack := &protocol.ItemStack{
				ItemID:     int16(tagAsInt32(item.GetTag("id"), -1)),
				StackSize:  int8(tagAsInt32(item.GetTag("Count"), 0)),
				ItemDamage: int16(tagAsInt32(item.GetTag("Damage"), 0)),
			}
			if stack.ItemID < 0 || stack.StackSize <= 0 {
				continue
			}
			if stack.StackSize > 64 {
				stack.StackSize = 64
			}
			if tag, ok := item.GetTag("tag").(*nbt.CompoundTag); ok {
				if copied, ok := tag.Copy().(*nbt.CompoundTag); ok {
					stack.Tag = copied
				}
			}
			state.Inventory[slot] = stack
		}
	}

	return state
}

func (s *StatusServer) savePlayerState(username string, state persistedPlayerState) error {
	if !s.cfg.PersistWorld || username == "" {
		return nil
	}

	path := s.playerDataPath(username)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := nbt.WriteCompressed(stateToNBT(state), f); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}

func (s *StatusServer) loadPlayerState(username string) (persistedPlayerState, bool) {
	if !s.cfg.PersistWorld || username == "" {
		return defaultPersistedPlayerState(), false
	}

	path := s.playerDataPath(username)
	f, err := os.Open(path)
	if err != nil {
		return defaultPersistedPlayerState(), false
	}
	defer f.Close()

	root, err := nbt.ReadCompressed(f)
	if err != nil {
		return defaultPersistedPlayerState(), false
	}
	return stateFromNBT(root), true
}
