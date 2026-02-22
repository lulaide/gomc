//go:build cgo

package gui

import (
	"math"
	"strings"
	"sync"

	"github.com/lulaide/gomc/pkg/nbt"
)

type potionDef struct {
	liquidColor   int
	effectiveness float64
	instant       bool
	usable        bool
}

type potionEffectEval struct {
	potionID  int
	duration  int
	amplifier int
}

var potionDefsByID = map[int]potionDef{
	// Translation reference:
	// - net.minecraft.src.Potion static registrations (id, bad/good, liquidColor, effectiveness, instant)
	1:  {liquidColor: 8171462, effectiveness: 1.0, instant: false, usable: false},   // moveSpeed
	2:  {liquidColor: 5926017, effectiveness: 0.5, instant: false, usable: false},   // moveSlowdown (bad)
	3:  {liquidColor: 14270531, effectiveness: 1.5, instant: false, usable: false},  // digSpeed
	5:  {liquidColor: 9643043, effectiveness: 1.0, instant: false, usable: false},   // damageBoost
	6:  {liquidColor: 16262179, effectiveness: 1.0, instant: true, usable: false},   // heal
	7:  {liquidColor: 4393481, effectiveness: 0.5, instant: true, usable: false},    // harm (bad)
	10: {liquidColor: 13458603, effectiveness: 0.25, instant: false, usable: false}, // regeneration
	11: {liquidColor: 10044730, effectiveness: 1.0, instant: false, usable: false},  // resistance
	12: {liquidColor: 14981690, effectiveness: 1.0, instant: false, usable: false},  // fireResistance
	14: {liquidColor: 8356754, effectiveness: 1.0, instant: false, usable: false},   // invisibility
	16: {liquidColor: 2039713, effectiveness: 1.0, instant: false, usable: false},   // nightVision
	18: {liquidColor: 4738376, effectiveness: 0.5, instant: false, usable: false},   // weakness (bad)
	19: {liquidColor: 5149489, effectiveness: 0.25, instant: false, usable: false},  // poison
}

var potionRequirementsByID = map[int]string{
	// Translation reference:
	// - net.minecraft.src.PotionHelper static initializer: potionRequirements.put(...)
	10: "0 & !1 & !2 & !3 & 0+6", // regeneration
	1:  "!0 & 1 & !2 & !3 & 1+6", // moveSpeed
	12: "0 & 1 & !2 & !3 & 0+6",  // fireResistance
	6:  "0 & !1 & 2 & !3",        // heal
	19: "!0 & !1 & 2 & !3 & 2+6", // poison
	18: "!0 & !1 & !2 & 3 & 3+6", // weakness
	7:  "!0 & !1 & 2 & 3",        // harm
	2:  "!0 & 1 & !2 & 3 & 3+6",  // moveSlowdown
	5:  "0 & !1 & !2 & 3 & 3+6",  // damageBoost
	16: "!0 & 1 & 2 & !3 & 2+6",  // nightVision
	14: "!0 & 1 & 2 & 3 & 2+6",   // invisibility
}

var potionAmplifiersByID = map[int]string{
	// Translation reference:
	// - net.minecraft.src.PotionHelper static initializer: potionAmplifiers.put(...)
	1:  "5", // moveSpeed
	3:  "5", // digSpeed
	5:  "5", // damageBoost
	10: "5", // regeneration
	7:  "5", // harm
	6:  "5", // heal
	11: "5", // resistance
	19: "5", // poison
}

var potionNameLangKeyByID = map[int]string{
	// Translation reference:
	// - net.minecraft.src.Potion static setPotionName(...)
	1:  "potion.moveSpeed",
	2:  "potion.moveSlowdown",
	3:  "potion.digSpeed",
	4:  "potion.digSlowDown",
	5:  "potion.damageBoost",
	6:  "potion.heal",
	7:  "potion.harm",
	8:  "potion.jump",
	9:  "potion.confusion",
	10: "potion.regeneration",
	11: "potion.resistance",
	12: "potion.fireResistance",
	13: "potion.waterBreathing",
	14: "potion.invisibility",
	15: "potion.blindness",
	16: "potion.nightVision",
	17: "potion.hunger",
	18: "potion.weakness",
	19: "potion.poison",
	20: "potion.wither",
}

var potionPrefixLangKeys = [32]string{
	// Translation reference:
	// - net.minecraft.src.PotionHelper.potionPrefixes
	"potion.prefix.mundane",
	"potion.prefix.uninteresting",
	"potion.prefix.bland",
	"potion.prefix.clear",
	"potion.prefix.milky",
	"potion.prefix.diffuse",
	"potion.prefix.artless",
	"potion.prefix.thin",
	"potion.prefix.awkward",
	"potion.prefix.flat",
	"potion.prefix.bulky",
	"potion.prefix.bungling",
	"potion.prefix.buttered",
	"potion.prefix.smooth",
	"potion.prefix.suave",
	"potion.prefix.debonair",
	"potion.prefix.thick",
	"potion.prefix.elegant",
	"potion.prefix.fancy",
	"potion.prefix.charming",
	"potion.prefix.dashing",
	"potion.prefix.refined",
	"potion.prefix.cordial",
	"potion.prefix.sparkling",
	"potion.prefix.potent",
	"potion.prefix.foul",
	"potion.prefix.odorless",
	"potion.prefix.rank",
	"potion.prefix.harsh",
	"potion.prefix.acrid",
	"potion.prefix.gross",
	"potion.prefix.stinky",
}

var (
	potionColorCacheMu sync.RWMutex
	potionColorCache   = make(map[int]int)
)

// Translation references:
// - net.minecraft.src.ItemPotion#getColorFromDamage(int)
// - net.minecraft.src.PotionHelper#func_77915_a(int,boolean)
func potionLiquidColorFromDamage(damage int) int {
	potionColorCacheMu.RLock()
	if color, ok := potionColorCache[damage]; ok {
		potionColorCacheMu.RUnlock()
		return color
	}
	potionColorCacheMu.RUnlock()

	color := calcPotionLiquidColor(potionEffectsFromDamage(damage, false))

	potionColorCacheMu.Lock()
	potionColorCache[damage] = color
	potionColorCacheMu.Unlock()
	return color
}

// Translation references:
// - net.minecraft.src.ItemPotion#getColorFromDamage(int)
// - net.minecraft.src.ItemPotion#getEffects(ItemStack)
func potionLiquidColorFromStack(damage int, itemTag *nbt.CompoundTag) int {
	// Fast cached path for vanilla damage-only potions.
	if itemTag == nil || !itemTag.HasKey("CustomPotionEffects") {
		return potionLiquidColorFromDamage(damage)
	}
	return calcPotionLiquidColor(potionEffectsFromItemStack(damage, itemTag, false))
}

// Translation references:
// - net.minecraft.src.PotionHelper#calcPotionLiquidColor(Collection)
func calcPotionLiquidColor(effects []potionEffectEval) int {
	const defaultPotionColor = 3694022
	if len(effects) == 0 {
		return defaultPotionColor
	}

	var sumR, sumG, sumB, samples float64
	for _, eff := range effects {
		def, ok := potionDefsByID[eff.potionID]
		if !ok {
			continue
		}
		color := def.liquidColor
		r := float64((color>>16)&255) / 255.0
		g := float64((color>>8)&255) / 255.0
		b := float64(color&255) / 255.0
		for i := 0; i <= eff.amplifier; i++ {
			sumR += r
			sumG += g
			sumB += b
			samples++
		}
	}
	if samples <= 0 {
		return defaultPotionColor
	}

	rr := int(sumR / samples * 255.0)
	gg := int(sumG / samples * 255.0)
	bb := int(sumB / samples * 255.0)
	return (rr << 16) | (gg << 8) | bb
}

// Translation references:
// - net.minecraft.src.PotionHelper#getPotionEffects(int,boolean)
func potionEffectsFromDamage(data int, includeUsable bool) []potionEffectEval {
	out := make([]potionEffectEval, 0, 8)

	for potionID := 0; potionID < 32; potionID++ {
		def, ok := potionDefsByID[potionID]
		if !ok {
			continue
		}
		if def.usable && !includeUsable {
			continue
		}

		req, ok := potionRequirementsByID[potionID]
		if !ok || req == "" {
			continue
		}
		level := parsePotionEffectExpr(req, 0, len(req), data)
		if level <= 0 {
			continue
		}

		amp := 0
		if ampExpr, ok := potionAmplifiersByID[potionID]; ok && ampExpr != "" {
			amp = parsePotionEffectExpr(ampExpr, 0, len(ampExpr), data)
			if amp < 0 {
				amp = 0
			}
		}

		duration := 1
		if !def.instant {
			duration = 1200 * (level*3 + (level-1)*2)
			duration >>= amp
			duration = int(math.Round(float64(duration) * def.effectiveness))
			if (data & 16384) != 0 {
				duration = int(math.Round(float64(duration)*0.75 + 0.5))
			}
		}

		out = append(out, potionEffectEval{
			potionID:  potionID,
			duration:  duration,
			amplifier: amp,
		})
	}
	return out
}

// Translation references:
// - net.minecraft.src.ItemPotion#getEffects(ItemStack)
func potionEffectsFromItemStack(damage int, itemTag *nbt.CompoundTag, includeUsable bool) []potionEffectEval {
	if itemTag != nil && itemTag.HasKey("CustomPotionEffects") {
		// ItemPotion checks only hasKey and then reads CustomPotionEffects list.
		if listTag, ok := itemTag.GetTag("CustomPotionEffects").(*nbt.ListTag); ok && listTag != nil {
			return customPotionEffectsFromNBTList(listTag)
		}
		return nil
	}
	return potionEffectsFromDamage(damage, includeUsable)
}

// Translation references:
// - net.minecraft.src.PotionEffect#readCustomPotionEffectFromNBT(NBTTagCompound)
func customPotionEffectsFromNBTList(listTag *nbt.ListTag) []potionEffectEval {
	if listTag == nil || listTag.TagCount() == 0 {
		return nil
	}
	out := make([]potionEffectEval, 0, listTag.TagCount())
	for _, raw := range listTag.TagList {
		effTag, ok := raw.(*nbt.CompoundTag)
		if !ok || effTag == nil {
			continue
		}
		id, ok := intFromNBTNumericTag(effTag.GetTag("Id"))
		if !ok {
			continue
		}
		amp, ok := intFromNBTNumericTag(effTag.GetTag("Amplifier"))
		if !ok {
			amp = 0
		}
		duration, ok := intFromNBTNumericTag(effTag.GetTag("Duration"))
		if !ok {
			duration = 0
		}
		out = append(out, potionEffectEval{
			potionID:  id & 0xFF,
			duration:  duration,
			amplifier: amp & 0xFF,
		})
	}
	return out
}

func potionCheckFlag(value, bit int) bool {
	return (value & (1 << bit)) != 0
}

func potionIsFlagSet(value, bit int) int {
	if potionCheckFlag(value, bit) {
		return 1
	}
	return 0
}

func potionIsFlagUnset(value, bit int) int {
	if potionCheckFlag(value, bit) {
		return 0
	}
	return 1
}

// Translation references:
// - net.minecraft.src.PotionHelper#func_77909_a(int)
// - net.minecraft.src.PotionHelper#func_77908_a(int,int,int,int,int,int)
func potionPrefixIndexFromDamage(data int) int {
	return potionPrefixBitsToIndex(data, 5, 4, 3, 2, 1)
}

func potionPrefixBitsToIndex(data, bitA, bitB, bitC, bitD, bitE int) int {
	prefix := 0
	if potionCheckFlag(data, bitA) {
		prefix |= 16
	}
	if potionCheckFlag(data, bitB) {
		prefix |= 8
	}
	if potionCheckFlag(data, bitC) {
		prefix |= 4
	}
	if potionCheckFlag(data, bitD) {
		prefix |= 2
	}
	if potionCheckFlag(data, bitE) {
		prefix |= 1
	}
	return prefix
}

func potionPrefixLangKeyFromDamage(data int) string {
	idx := potionPrefixIndexFromDamage(data)
	if idx < 0 || idx >= len(potionPrefixLangKeys) {
		return ""
	}
	return potionPrefixLangKeys[idx]
}

func potionCountSetFlags(value int) int {
	count := 0
	for value > 0 {
		value &= value - 1
		count++
	}
	return count
}

// Translation reference:
// - net.minecraft.src.PotionHelper#func_77904_a(...)
func potionEvalTerm(invert, multiply, negative bool, comparator, bitIndex, weight, data int) int {
	result := 0
	if invert {
		result = potionIsFlagUnset(data, bitIndex)
	} else if comparator != -1 {
		flagCount := potionCountSetFlags(data)
		if comparator == 0 && flagCount == bitIndex {
			result = 1
		} else if comparator == 1 && flagCount > bitIndex {
			result = 1
		} else if comparator == 2 && flagCount < bitIndex {
			result = 1
		}
	} else {
		result = potionIsFlagSet(data, bitIndex)
	}

	if multiply {
		result *= weight
	}
	if negative {
		result *= -1
	}
	return result
}

// Translation reference:
// - net.minecraft.src.PotionHelper#parsePotionEffects(String,int,int,int)
func parsePotionEffectExpr(expr string, start, end, data int) int {
	if start < len(expr) && end >= 0 && start < end {
		orIdx := strings.IndexByte(expr[start:], '|')
		if orIdx >= 0 {
			orIdx += start
		}
		if orIdx >= 0 && orIdx < end {
			left := parsePotionEffectExpr(expr, start, orIdx-1, data)
			if left > 0 {
				return left
			}
			right := parsePotionEffectExpr(expr, orIdx+1, end, data)
			if right > 0 {
				return right
			}
			return 0
		}

		andIdx := strings.IndexByte(expr[start:], '&')
		if andIdx >= 0 {
			andIdx += start
		}
		if andIdx >= 0 && andIdx < end {
			left := parsePotionEffectExpr(expr, start, andIdx-1, data)
			if left <= 0 {
				return 0
			}
			right := parsePotionEffectExpr(expr, andIdx+1, end, data)
			if right <= 0 {
				return 0
			}
			if left > right {
				return left
			}
			return right
		}

		hasMultiplier := false
		hasWeight := false
		hasToken := false
		invert := false
		negative := false
		comparator := -1
		bitIndex := 0
		weight := 0
		total := 0

		flush := func() {
			if !hasToken {
				return
			}
			total += potionEvalTerm(invert, hasWeight, negative, comparator, bitIndex, weight, data)
			invert = false
			negative = false
			hasMultiplier = false
			hasWeight = false
			hasToken = false
			bitIndex = 0
			weight = 0
			comparator = -1
		}

		for i := start; i < end; i++ {
			ch := expr[i]
			if ch >= '0' && ch <= '9' {
				if hasMultiplier {
					weight = int(ch - '0')
					hasWeight = true
				} else {
					bitIndex *= 10
					bitIndex += int(ch - '0')
					hasToken = true
				}
				continue
			}

			switch ch {
			case '*':
				hasMultiplier = true
			case '!':
				flush()
				invert = true
			case '-':
				flush()
				negative = true
			case '+':
				flush()
			case '=':
				flush()
				comparator = 0
			case '<':
				flush()
				comparator = 2
			case '>':
				flush()
				comparator = 1
			}
		}
		flush()
		return total
	}
	return 0
}
