//go:build cgo

package gui

import netclient "github.com/lulaide/gomc/pkg/network/client"

func (a *App) handlePickBlockAction() {
	if a.session == nil {
		return
	}

	snap := a.session.Snapshot()
	target := a.pickBlockTarget(snap, interactReach)
	if !target.Hit {
		return
	}

	blockID, meta, ok := a.session.BlockAt(target.X, target.Y, target.Z)
	if !ok || blockID <= 0 {
		return
	}

	itemID, itemDamage, ok := pickItemForBlock(blockID, meta)
	if !ok || itemID <= 0 {
		return
	}

	if slot, found := findHotbarSlotForPick(a.session.InventorySnapshot(), itemID, itemDamage); found {
		_ = a.session.SelectHotbar(slot)
		return
	}

	// Translation reference:
	// - net.minecraft.src.Minecraft.clickMiddleMouseButton()
	// - net.minecraft.src.InventoryPlayer.setCurrentItem(..., creative=true)
	if !snap.IsCreative {
		return
	}

	slot := snap.HeldSlot
	if slot < 0 || slot > 8 {
		slot = 0
	}

	inv := a.session.InventorySnapshot()
	for i := int16(0); i < 9; i++ {
		st := inv[36+i]
		if st.ItemID <= 0 || st.StackSize <= 0 {
			slot = i
			break
		}
	}

	_ = a.session.SetCreativeHotbarSlot(slot, itemID, itemDamage, 1)
	_ = a.session.SelectHotbar(slot)
}

func findHotbarSlotForPick(inv [45]netclient.InventorySlotSnapshot, itemID, itemDamage int16) (int16, bool) {
	for i := int16(0); i < 9; i++ {
		st := inv[36+i]
		if st.ItemID == itemID && st.StackSize > 0 && st.ItemDamage == itemDamage {
			return i, true
		}
	}
	for i := int16(0); i < 9; i++ {
		st := inv[36+i]
		if st.ItemID == itemID && st.StackSize > 0 {
			return i, true
		}
	}
	return 0, false
}

func pickItemForBlock(blockID, meta int) (itemID, itemDamage int16, ok bool) {
	switch blockID {
	case 26: // bed
		return 355, 0, true
	case 117: // brewing stand
		return 379, 0, true
	case 92: // cake
		return 354, 0, true
	case 118: // cauldron
		return 380, 0, true
	case 127: // cocoa -> brown dye
		return 351, 3, true
	case 149, 150: // comparator (active/inactive)
		return 404, 0, true
	case 59: // wheat crops
		return 295, 0, true
	case 64: // wooden door
		return 324, 0, true
	case 71: // iron door
		return 330, 0, true
	case 60: // farmland -> dirt
		return 3, 0, true
	case 140: // flower pot
		return 390, 0, true
	case 62: // lit furnace -> furnace
		return 61, 0, true
	case 52, 90, 119, 122, 36: // spawner/portal/end portal/dragon egg/moving piston
		return 0, 0, false
	case 99: // brown mushroom cap
		return 39, 0, true
	case 100: // red mushroom cap
		return 40, 0, true
	case 115: // nether wart
		return 372, 0, true
	case 34: // piston extension
		if meta&8 != 0 {
			return 29, 0, true
		}
		return 33, 0, true
	case 124: // lit redstone lamp
		return 123, 0, true
	case 93, 94: // repeater
		return 356, 0, true
	case 55: // redstone wire
		return 331, 0, true
	case 63, 68: // signs
		return 323, 0, true
	case 144: // skull
		return 397, 0, true
	case 83: // reeds
		return 338, 0, true
	case 104: // pumpkin stem
		return 361, 0, true
	case 105: // melon stem
		return 362, 0, true
	case 132: // tripwire
		return 287, 0, true
	case 43: // double slab -> single slab
		return 44, int16(meta & 7), true
	case 125: // double wood slab -> single
		return 126, int16(meta & 7), true
	}

	if blockID <= 0 {
		return 0, 0, false
	}
	return int16(blockID), normalizePickDamage(blockID, meta), true
}

func normalizePickDamage(blockID, meta int) int16 {
	switch blockID {
	case 6: // sapling
		return int16(meta & 3)
	case 17, 18: // logs/leaves
		return int16(meta & 3)
	case 24: // sandstone
		return int16(meta & 3)
	case 35: // wool
		return int16(meta & 15)
	case 43, 44, 125, 126: // slab variants
		return int16(meta & 7)
	default:
		return int16(meta)
	}
}
