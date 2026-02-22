//go:build cgo

package gui

// Translation references:
// - net.minecraft.src.ItemTool#ItemTool(int,int,EnumToolMaterial,Block[])
// - net.minecraft.src.ItemSword#ItemSword(int,EnumToolMaterial)
// - net.minecraft.src.ItemHoe#ItemHoe(int,EnumToolMaterial)
// - net.minecraft.src.ItemArmor#ItemArmor(int,EnumArmorMaterial,int,int)
// - net.minecraft.src.EnumToolMaterial
// - net.minecraft.src.EnumArmorMaterial#getDurability(int)
// - net.minecraft.src.ItemBow#ItemBow(int)
// - net.minecraft.src.ItemFlintAndSteel#ItemFlintAndSteel(int)
// - net.minecraft.src.ItemFishingRod#ItemFishingRod(int)
// - net.minecraft.src.ItemShears#ItemShears(int)
// - net.minecraft.src.ItemCarrotOnAStick#ItemCarrotOnAStick(int)
var itemMaxDurabilityByID = map[int16]int16{
	// Tools (EnumToolMaterial + ItemTool/ItemSword/ItemHoe)
	256: 250, 257: 250, 258: 250, // iron shovel/pickaxe/axe
	267: 250, // iron sword
	292: 250, // iron hoe

	268: 59, 269: 59, 270: 59, 271: 59, // wood sword/shovel/pickaxe/axe
	290: 59, // wood hoe

	272: 131, 273: 131, 274: 131, 275: 131, // stone sword/shovel/pickaxe/axe
	291: 131, // stone hoe

	276: 1561, 277: 1561, 278: 1561, 279: 1561, // diamond sword/shovel/pickaxe/axe
	293: 1561, // diamond hoe

	283: 32, 284: 32, 285: 32, 286: 32, // gold sword/shovel/pickaxe/axe
	294: 32, // gold hoe

	// Armor (EnumArmorMaterial#getDurability(slot), slot: 0 helmet, 1 chest, 2 legs, 3 boots)
	298: 55, 299: 80, 300: 75, 301: 65, // leather
	302: 165, 303: 240, 304: 225, 305: 195, // chain
	306: 165, 307: 240, 308: 225, 309: 195, // iron
	310: 363, 311: 528, 312: 495, 313: 429, // diamond
	314: 77, 315: 112, 316: 105, 317: 91, // gold

	// Other damageable items
	259: 64,  // flint and steel
	261: 384, // bow
	346: 64,  // fishing rod
	359: 238, // shears
	398: 25,  // carrot on a stick
}

func itemMaxDurability(itemID int16) (int16, bool) {
	maxDamage, ok := itemMaxDurabilityByID[itemID]
	return maxDamage, ok && maxDamage > 0
}
