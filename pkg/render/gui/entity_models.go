//go:build cgo

package gui

import (
	"math"

	"github.com/go-gl/gl/v2.1/gl"
	netclient "github.com/lulaide/gomc/pkg/network/client"
)

// Translation references:
// - net.minecraft.src.EntityList (numeric entity ids)
// - net.minecraft.src.ModelBiped
// - net.minecraft.src.ModelQuadruped
// - net.minecraft.src.ModelCreeper
// - net.minecraft.src.ModelSpider
// - net.minecraft.src.ModelRenderer#addBox (texture unwrapping layout)

type entityModelKind int

const (
	entityModelBiped entityModelKind = iota
	entityModelEnderman
	entityModelPig
	entityModelSheep
	entityModelCow
	entityModelChicken
	entityModelBat
	entityModelSquid
	entityModelWolf
	entityModelOcelot
	entityModelQuadruped
	entityModelCreeper
	entityModelSpider
	entityModelSlime
	entityModelVillager
	entityModelFallback
)

type entityModelProfile struct {
	kind     entityModelKind
	scale    float32
	baseLift float32
	colorR   float32
	colorG   float32
	colorB   float32

	thinLimbs  bool
	tall       bool
	zombieArms bool
}

type uvRect struct {
	u0 float32
	v0 float32
	u1 float32
	v1 float32
}

type cuboidUV struct {
	Down  uvRect
	Up    uvRect
	North uvRect
	South uvRect
	West  uvRect
	East  uvRect
}

var fullFaces = visibleFaces{
	Down:  true,
	Up:    true,
	North: true,
	South: true,
	West:  true,
	East:  true,
}

var sheepFleeceColorTable = [16][3]float32{
	{1.0, 1.0, 1.0},
	{0.85, 0.5, 0.2},
	{0.7, 0.3, 0.85},
	{0.4, 0.6, 0.85},
	{0.9, 0.9, 0.2},
	{0.5, 0.8, 0.1},
	{0.95, 0.5, 0.65},
	{0.3, 0.3, 0.3},
	{0.6, 0.6, 0.6},
	{0.3, 0.5, 0.6},
	{0.5, 0.25, 0.7},
	{0.2, 0.3, 0.7},
	{0.4, 0.3, 0.2},
	{0.4, 0.5, 0.2},
	{0.6, 0.2, 0.2},
	{0.1, 0.1, 0.1},
}

func sheepFleeceRGB(color int8) (float32, float32, float32) {
	idx := int(color) & 15
	rgb := sheepFleeceColorTable[idx]
	return rgb[0], rgb[1], rgb[2]
}

func (a *App) drawEntityModel(ent netclient.EntitySnapshot, animTime float64) {
	profile := modelProfileForEntityType(ent.Type)
	tex := a.entityTextureForType(ent.Type)
	if ent.Type == 54 && ent.ZombieVillager {
		if villagerTex := a.entityTextureByPath("zombie/zombie_villager.png"); villagerTex != nil {
			tex = villagerTex
		}
	}
	if ent.Type == 51 && ent.SkeletonType == 1 {
		if witherTex := a.entityTextureByPath("skeleton/wither_skeleton.png"); witherTex != nil {
			tex = witherTex
		}
	}
	if ent.Type == 54 && ent.ZombieChild {
		// Translation reference:
		// - net.minecraft.src.RenderBiped#func_82422_c (child scale path)
		profile.scale *= 0.5
	}

	yaw := angleFromByte(ent.Yaw)
	headYaw := angleFromByte(ent.HeadYaw)
	headRelYaw := normalizeDegrees(headYaw - yaw)
	pitch := clampf(angleFromByte(ent.Pitch), -50, 50)
	swing := float32(math.Sin(animTime*6.0 + float64(ent.EntityID)*0.61))
	swingDeg := swing * 30.0
	if ent.Type == 0 && ent.SwingProgress > 0 {
		// Translation reference:
		// - net.minecraft.src.ModelBiped#onGround attack swing curve
		attack := float32(math.Sin(math.Sqrt(float64(ent.SwingProgress)) * math.Pi))
		swingDeg = attack * 70.0
	}

	lift := profile.baseLift
	if ent.Sneaking {
		lift -= 0.08
	}

	gl.PushMatrix()
	gl.Translatef(float32(ent.X), float32(ent.Y)+lift, float32(ent.Z))
	gl.Rotatef(180.0-yaw, 0, 1, 0)
	gl.Scalef(profile.scale, profile.scale, profile.scale)

	switch profile.kind {
	case entityModelBiped:
		drawBipedModel(profile, tex, headRelYaw, pitch, swingDeg, ent.Sneaking)
	case entityModelEnderman:
		drawEndermanModel(profile, tex, headRelYaw, pitch, swingDeg, ent.Sneaking)
	case entityModelPig:
		drawPigModel(profile, tex, headRelYaw, pitch, swingDeg)
	case entityModelSheep:
		drawSheepModel(profile, tex, headRelYaw, pitch, swingDeg)
		// Translation reference:
		// - net.minecraft.src.RenderSheep#setWoolColorAndRender
		if !ent.SheepSheared {
			if fur := a.entityTextureByPath("sheep/sheep_fur.png"); fur != nil {
				fr, fg, fb := sheepFleeceRGB(ent.SheepColor)
				drawSheepFurModel(profile, fur, headRelYaw, pitch, swingDeg, fr, fg, fb)
			}
		}
	case entityModelCow:
		drawCowModel(profile, tex, headRelYaw, pitch, swingDeg)
	case entityModelChicken:
		drawChickenModel(profile, tex, headRelYaw, pitch, swingDeg, float32(animTime))
	case entityModelBat:
		drawBatModel(profile, tex, headRelYaw, pitch, float32(animTime))
	case entityModelSquid:
		drawSquidModel(profile, tex, float32(animTime))
	case entityModelWolf:
		drawWolfModel(profile, tex, headRelYaw, pitch, swingDeg, float32(animTime))
	case entityModelOcelot:
		drawOcelotModel(profile, tex, headRelYaw, pitch, swingDeg, float32(animTime))
	case entityModelQuadruped:
		drawQuadrupedModel(profile, tex, pitch, swingDeg)
	case entityModelCreeper:
		drawCreeperModel(profile, tex, headRelYaw, pitch, swingDeg)
	case entityModelSpider:
		drawSpiderModel(profile, tex, headRelYaw, pitch, swingDeg)
	case entityModelSlime:
		drawSlimeModel(profile, tex, swing, ent.SlimeSize)
	case entityModelVillager:
		drawVillagerModel(profile, tex, headRelYaw, pitch, swingDeg)
	default:
		drawFallbackModel(profile, tex)
	}

	gl.PopMatrix()
}

func modelProfileForEntityType(t int8) entityModelProfile {
	switch t {
	case 0: // player
		return entityModelProfile{kind: entityModelBiped, scale: 0.92, colorR: 0.88, colorG: 0.74, colorB: 0.62}
	case 50: // creeper
		return entityModelProfile{kind: entityModelCreeper, scale: 0.95, colorR: 0.40, colorG: 0.63, colorB: 0.28}
	case 51: // skeleton
		return entityModelProfile{kind: entityModelBiped, scale: 0.92, colorR: 0.90, colorG: 0.90, colorB: 0.90, thinLimbs: true}
	case 52: // spider
		return entityModelProfile{kind: entityModelSpider, scale: 1.00, colorR: 0.23, colorG: 0.20, colorB: 0.17}
	case 54: // zombie
		return entityModelProfile{kind: entityModelBiped, scale: 0.92, colorR: 0.40, colorG: 0.62, colorB: 0.34, zombieArms: true}
	case 55: // slime
		return entityModelProfile{kind: entityModelSlime, scale: 0.95, colorR: 0.43, colorG: 0.71, colorB: 0.43}
	case 57: // pig zombie
		return entityModelProfile{kind: entityModelBiped, scale: 0.92, colorR: 0.70, colorG: 0.57, colorB: 0.45, zombieArms: true}
	case 58: // enderman
		return entityModelProfile{kind: entityModelEnderman, scale: 0.92, colorR: 0.09, colorG: 0.08, colorB: 0.11}
	case 59: // cave spider
		return entityModelProfile{kind: entityModelSpider, scale: 0.72, colorR: 0.12, colorG: 0.28, colorB: 0.42}
	case 60: // silverfish
		return entityModelProfile{kind: entityModelFallback, scale: 0.45, colorR: 0.55, colorG: 0.55, colorB: 0.58}
	case 61: // blaze
		return entityModelProfile{kind: entityModelFallback, scale: 0.92, colorR: 0.93, colorG: 0.66, colorB: 0.24}
	case 62: // magma cube
		return entityModelProfile{kind: entityModelSlime, scale: 0.95, colorR: 0.52, colorG: 0.17, colorB: 0.08}
	case 65: // bat
		return entityModelProfile{kind: entityModelBat, scale: 0.35, colorR: 0.24, colorG: 0.20, colorB: 0.20}
	case 66: // witch
		return entityModelProfile{kind: entityModelBiped, scale: 0.92, colorR: 0.63, colorG: 0.58, colorB: 0.70}
	case 90: // pig
		return entityModelProfile{kind: entityModelPig, scale: 0.90, colorR: 0.92, colorG: 0.66, colorB: 0.68}
	case 91: // sheep
		return entityModelProfile{kind: entityModelSheep, scale: 0.95, colorR: 0.92, colorG: 0.92, colorB: 0.92}
	case 92: // cow
		return entityModelProfile{kind: entityModelCow, scale: 0.95, colorR: 0.45, colorG: 0.33, colorB: 0.22}
	case 93: // chicken
		return entityModelProfile{kind: entityModelChicken, scale: 0.55, colorR: 0.93, colorG: 0.90, colorB: 0.86}
	case 94: // squid
		return entityModelProfile{kind: entityModelSquid, scale: 0.82, colorR: 0.24, colorG: 0.32, colorB: 0.48}
	case 95: // wolf
		return entityModelProfile{kind: entityModelWolf, scale: 0.75, colorR: 0.72, colorG: 0.72, colorB: 0.72}
	case 96: // mooshroom
		return entityModelProfile{kind: entityModelCow, scale: 0.95, colorR: 0.66, colorG: 0.24, colorB: 0.20}
	case 98: // ocelot
		return entityModelProfile{kind: entityModelOcelot, scale: 0.66, colorR: 0.87, colorG: 0.71, colorB: 0.42}
	case 99: // iron golem
		return entityModelProfile{kind: entityModelBiped, scale: 1.25, colorR: 0.80, colorG: 0.78, colorB: 0.71}
	case 100: // horse
		return entityModelProfile{kind: entityModelQuadruped, scale: 1.15, colorR: 0.48, colorG: 0.36, colorB: 0.24}
	case 120: // villager
		return entityModelProfile{kind: entityModelVillager, scale: 0.92, colorR: 0.74, colorG: 0.63, colorB: 0.52}
	default:
		return entityModelProfile{kind: entityModelFallback, scale: 0.92, colorR: 0.68, colorG: 0.68, colorB: 0.72}
	}
}

func drawBipedModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg float32, isSneak bool) {
	// Translation reference:
	// - net.minecraft.src.ModelBiped (constructor + setRotationAngles)
	bodyR, bodyG, bodyB := p.colorR*0.82, p.colorG*0.82, p.colorB*0.82
	limbR, limbG, limbB := p.colorR*0.94, p.colorG*0.94, p.colorB*0.94
	if tex != nil {
		bodyR, bodyG, bodyB = 1, 1, 1
		limbR, limbG, limbB = 1, 1, 1
	}

	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 8, 8, 8)
	bodyUV := cuboidUVFromTextureOffset(tex, 16, 16, 8, 12, 4)

	armW, armH, armD := 4, 12, 4
	legW, legH, legD := 4, 12, 4
	rightArmBoxX, leftArmBoxX := float32(-3), float32(-1)
	rightArmPivotX, leftArmPivotX := float32(-5), float32(5)
	rightLegPivotX, leftLegPivotX := float32(-1.9), float32(1.9)
	if p.thinLimbs {
		// Skeleton-like 2px limbs.
		armW, armD = 2, 2
		legW, legD = 2, 2
		rightArmBoxX, leftArmBoxX = -1, -1
		rightLegPivotX, leftLegPivotX = -2.0, 2.0
	}
	armUV := cuboidUVFromTextureOffset(tex, 40, 16, armW, armH, armD)
	legUV := cuboidUVFromTextureOffset(tex, 0, 16, legW, legH, legD)

	swingRad := swingDeg * float32(math.Pi/180.0)
	rightArmX := swingRad
	leftArmX := -swingRad
	rightArmY, leftArmY := float32(0), float32(0)
	rightArmZ, leftArmZ := float32(0), float32(0)
	rightLegX := -swingRad * 1.4
	leftLegX := swingRad * 1.4

	bodyX := float32(0)
	headPivotY := float32(0)
	legPivotY := float32(12)
	legPivotZ := float32(0.1)
	if isSneak {
		bodyX = 0.5
		rightArmX += 0.4
		leftArmX += 0.4
		legPivotY = 9.0
		legPivotZ = 4.0
		headPivotY = 1.0
	}
	if p.zombieArms {
		rightArmY = -0.1
		leftArmY = 0.1
		rightArmX = -float32(math.Pi)/2 + swingRad*0.5
		leftArmX = -float32(math.Pi)/2 - swingRad*0.5
	}

	drawSourcePartRad(tex, headUV, 0, headPivotY, 0, -4, -8, -4, 8, 8, 8, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, 1, 1, 1)
	drawSourcePartRad(tex, bodyUV, 0, 0, 0, -4, 0, -2, 8, 12, 4, bodyX, 0, 0, false, bodyR, bodyG, bodyB)

	drawSourcePartRad(tex, armUV, rightArmPivotX, 2, 0, rightArmBoxX, -2, -2, armW, armH, armD, rightArmX, rightArmY, rightArmZ, false, limbR, limbG, limbB)
	drawSourcePartRad(tex, armUV, leftArmPivotX, 2, 0, leftArmBoxX, -2, -2, armW, armH, armD, leftArmX, leftArmY, leftArmZ, true, limbR, limbG, limbB)
	drawSourcePartRad(tex, legUV, rightLegPivotX, legPivotY, legPivotZ, -float32(legW)/2, 0, -float32(legD)/2, legW, legH, legD, rightLegX, 0, 0, false, limbR*0.96, limbG*0.96, limbB*0.96)
	drawSourcePartRad(tex, legUV, leftLegPivotX, legPivotY, legPivotZ, -float32(legW)/2, 0, -float32(legD)/2, legW, legH, legD, leftLegX, 0, 0, true, limbR*0.96, limbG*0.96, limbB*0.96)
}

func drawEndermanModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg float32, isSneak bool) {
	// Translation reference:
	// - net.minecraft.src.ModelEnderman
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 8, 8, 8)
	bodyUV := cuboidUVFromTextureOffset(tex, 32, 16, 8, 12, 4)
	limbUV := cuboidUVFromTextureOffset(tex, 56, 0, 2, 30, 2)

	swingRad := swingDeg * float32(math.Pi/180.0)
	rightArmX := clampf(swingRad*0.5, -0.4, 0.4)
	leftArmX := clampf(-swingRad*0.5, -0.4, 0.4)
	rightLegX := clampf((-swingRad*1.4)*0.5, -0.4, 0.4)
	leftLegX := clampf((swingRad*1.4)*0.5, -0.4, 0.4)
	if isSneak {
		rightArmX += 0.15
		leftArmX += 0.15
	}

	// var8 = -14 in ModelEnderman#setRotationAngles.
	drawSourcePartRad(tex, headUV, 0, -13, 0, -4, -8, -4, 8, 8, 8, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, bodyUV, 0, -14, 0, -4, 0, -2, 8, 12, 4, 0, 0, 0, false, r*0.82, g*0.82, b*0.82)
	drawSourcePartRad(tex, limbUV, -3, -12, 0, -1, -2, -1, 2, 30, 2, rightArmX, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, limbUV, 5, -12, 0, -1, -2, -1, 2, 30, 2, leftArmX, 0, 0, true, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, limbUV, -2, -5, 0, -1, 0, -1, 2, 30, 2, rightLegX, 0, 0, false, r*0.90, g*0.90, b*0.90)
	drawSourcePartRad(tex, limbUV, 2, -5, 0, -1, 0, -1, 2, 30, 2, leftLegX, 0, 0, true, r*0.90, g*0.90, b*0.90)
}

func drawPigModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg float32) {
	// Translation reference:
	// - net.minecraft.src.ModelPig
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 8, 8, 8)
	snoutUV := cuboidUVFromTextureOffset(tex, 16, 16, 4, 3, 1)
	bodyUV := cuboidUVFromTextureOffset(tex, 28, 8, 10, 16, 8)
	legUV := cuboidUVFromTextureOffset(tex, 0, 16, 4, 6, 4)

	swingRad := swingDeg * float32(math.Pi/180.0)
	legA := swingRad * 1.4
	drawSourcePartRad(tex, headUV, 0, 12, -6, -4, -4, -8, 8, 8, 8, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, snoutUV, 0, 12, -6, -2, 0, -9, 4, 3, 1, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, bodyUV, 0, 11, 2, -5, -10, -7, 10, 16, 8, float32(math.Pi)/2, 0, 0, false, r*0.86, g*0.86, b*0.86)
	drawSourcePartRad(tex, legUV, -3, 18, 7, -2, 0, -2, 4, 6, 4, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 3, 18, 7, -2, 0, -2, 4, 6, 4, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, -3, 18, -5, -2, 0, -2, 4, 6, 4, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 3, 18, -5, -2, 0, -2, 4, 6, 4, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
}

func drawSheepModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg float32) {
	// Translation reference:
	// - net.minecraft.src.ModelSheep2
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 6, 6, 8)
	bodyUV := cuboidUVFromTextureOffset(tex, 28, 8, 8, 16, 6)
	legUV := cuboidUVFromTextureOffset(tex, 0, 16, 4, 12, 4)

	swingRad := swingDeg * float32(math.Pi/180.0)
	legA := swingRad * 1.4
	drawSourcePartRad(tex, headUV, 0, 6, -8, -3, -4, -6, 6, 6, 8, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, bodyUV, 0, 5, 2, -4, -10, -7, 8, 16, 6, float32(math.Pi)/2, 0, 0, false, r*0.88, g*0.88, b*0.88)
	drawSourcePartRad(tex, legUV, -3, 12, 7, -2, 0, -2, 4, 12, 4, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 3, 12, 7, -2, 0, -2, 4, 12, 4, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, -3, 12, -5, -2, 0, -2, 4, 12, 4, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 3, 12, -5, -2, 0, -2, 4, 12, 4, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
}

func drawSheepFurModel(_ entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg, r, g, b float32) {
	// Translation reference:
	// - net.minecraft.src.ModelSheep1
	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 6, 6, 6)
	bodyUV := cuboidUVFromTextureOffset(tex, 28, 8, 8, 16, 6)
	legUV := cuboidUVFromTextureOffset(tex, 0, 16, 4, 6, 4)

	swingRad := swingDeg * float32(math.Pi/180.0)
	legA := swingRad * 1.4
	drawSourcePartRadInflate(tex, headUV, 0, 6, -8, -3, -4, -4, 6, 6, 6, 0.6, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRadInflate(tex, bodyUV, 0, 5, 2, -4, -10, -7, 8, 16, 6, 1.75, float32(math.Pi)/2, 0, 0, false, r*0.88, g*0.88, b*0.88)
	drawSourcePartRadInflate(tex, legUV, -3, 12, 7, -2, 0, -2, 4, 6, 4, 0.5, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRadInflate(tex, legUV, 3, 12, 7, -2, 0, -2, 4, 6, 4, 0.5, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRadInflate(tex, legUV, -3, 12, -5, -2, 0, -2, 4, 6, 4, 0.5, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRadInflate(tex, legUV, 3, 12, -5, -2, 0, -2, 4, 6, 4, 0.5, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
}

func drawCowModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg float32) {
	// Translation reference:
	// - net.minecraft.src.ModelCow
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 8, 8, 6)
	hornUV := cuboidUVFromTextureOffset(tex, 22, 0, 1, 3, 1)
	bodyUV := cuboidUVFromTextureOffset(tex, 18, 4, 12, 18, 10)
	udderUV := cuboidUVFromTextureOffset(tex, 52, 0, 4, 6, 1)
	legUV := cuboidUVFromTextureOffset(tex, 0, 16, 4, 12, 4)

	swingRad := swingDeg * float32(math.Pi/180.0)
	legA := swingRad * 1.4
	drawSourcePartRad(tex, headUV, 0, 4, -8, -4, -4, -6, 8, 8, 6, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, hornUV, 0, 4, -8, -5, -5, -4, 1, 3, 1, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, hornUV, 0, 4, -8, 4, -5, -4, 1, 3, 1, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, bodyUV, 0, 5, 2, -6, -10, -7, 12, 18, 10, float32(math.Pi)/2, 0, 0, false, r*0.86, g*0.86, b*0.86)
	drawSourcePartRad(tex, udderUV, 0, 5, 2, -2, 2, -8, 4, 6, 1, float32(math.Pi)/2, 0, 0, false, r*0.86, g*0.86, b*0.86)
	drawSourcePartRad(tex, legUV, -4, 12, 7, -2, 0, -2, 4, 12, 4, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 4, 12, 7, -2, 0, -2, 4, 12, 4, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, -4, 12, -6, -2, 0, -2, 4, 12, 4, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 4, 12, -6, -2, 0, -2, 4, 12, 4, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
}

func drawChickenModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg, ageTicks float32) {
	// Translation reference:
	// - net.minecraft.src.ModelChicken
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 4, 6, 3)
	billUV := cuboidUVFromTextureOffset(tex, 14, 0, 4, 2, 2)
	chinUV := cuboidUVFromTextureOffset(tex, 14, 4, 2, 2, 2)
	bodyUV := cuboidUVFromTextureOffset(tex, 0, 9, 6, 8, 6)
	legUV := cuboidUVFromTextureOffset(tex, 26, 0, 3, 5, 3)
	wingUV := cuboidUVFromTextureOffset(tex, 24, 13, 1, 4, 6)

	swingRad := swingDeg * float32(math.Pi/180.0)
	legR := swingRad * 1.4
	legL := -legR
	wingFlap := float32(math.Sin(float64(ageTicks*2.5))) * 0.6

	drawSourcePartRad(tex, headUV, 0, 15, -4, -2, -6, -2, 4, 6, 3, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, billUV, 0, 15, -4, -2, -4, -4, 4, 2, 2, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, chinUV, 0, 15, -4, -1, -2, -3, 2, 2, 2, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, bodyUV, 0, 16, 0, -3, -4, -3, 6, 8, 6, float32(math.Pi)/2, 0, 0, false, r*0.86, g*0.86, b*0.86)
	drawSourcePartRad(tex, legUV, -2, 19, 1, -1, 0, -3, 3, 5, 3, legR, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 1, 19, 1, -1, 0, -3, 3, 5, 3, legL, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, wingUV, -4, 13, 0, 0, 0, -3, 1, 4, 6, 0, 0, wingFlap, false, r, g, b)
	drawSourcePartRad(tex, wingUV, 4, 13, 0, -1, 0, -3, 1, 4, 6, 0, 0, -wingFlap, false, r, g, b)
}

func drawBatModel(p entityModelProfile, tex *texture2D, headYaw, pitch, ageTicks float32) {
	// Translation reference:
	// - net.minecraft.src.ModelBat
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 6, 6, 6)
	earUV := cuboidUVFromTextureOffset(tex, 24, 0, 3, 4, 1)
	bodyUV := cuboidUVFromTextureOffset(tex, 0, 16, 6, 12, 6)
	bodyTailUV := cuboidUVFromTextureOffset(tex, 0, 34, 10, 6, 1)
	wingUV := cuboidUVFromTextureOffset(tex, 42, 0, 10, 16, 1)
	outerWingUV := cuboidUVFromTextureOffset(tex, 24, 16, 8, 12, 1)

	bodyRotX := float32(math.Pi)/4 + float32(math.Cos(float64(ageTicks*0.1)))*0.15
	wingY := float32(math.Cos(float64(ageTicks*1.3))) * float32(math.Pi) * 0.25
	outerWingY := wingY * 0.5

	drawSourcePartRad(tex, headUV, 0, 0, 0, -3, -3, -3, 6, 6, 6, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, earUV, 0, 0, 0, -4, -6, -2, 3, 4, 1, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, earUV, 0, 0, 0, 1, -6, -2, 3, 4, 1, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, true, r, g, b)

	drawSourcePartRad(tex, bodyUV, 0, 0, 0, -3, 4, -3, 6, 12, 6, bodyRotX, 0, 0, false, r*0.86, g*0.86, b*0.86)
	drawSourcePartRad(tex, bodyTailUV, 0, 0, 0, -5, 16, 0, 10, 6, 1, bodyRotX, 0, 0, false, r*0.86, g*0.86, b*0.86)

	gl.PushMatrix()
	gl.Rotatef(-bodyRotX*180.0/float32(math.Pi), 1, 0, 0)
	drawSourcePartRad(tex, wingUV, 0, 0, 0, -12, 1, 1.5, 10, 16, 1, 0, wingY, 0, false, r, g, b)
	drawSourcePartRad(tex, outerWingUV, -12, 1, 1.5, -8, 1, 0, 8, 12, 1, 0, outerWingY, 0, false, r, g, b)
	drawSourcePartRad(tex, wingUV, 0, 0, 0, 2, 1, 1.5, 10, 16, 1, 0, -wingY, 0, true, r, g, b)
	drawSourcePartRad(tex, outerWingUV, 12, 1, 1.5, 0, 1, 0, 8, 12, 1, 0, -outerWingY, 0, true, r, g, b)
	gl.PopMatrix()
}

func drawSquidModel(p entityModelProfile, tex *texture2D, ageTicks float32) {
	// Translation reference:
	// - net.minecraft.src.ModelSquid
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	bodyUV := cuboidUVFromTextureOffset(tex, 0, 0, 12, 16, 12)
	tentacleUV := cuboidUVFromTextureOffset(tex, 48, 0, 2, 18, 2)

	drawSourcePartRad(tex, bodyUV, 0, 8, 0, -6, -8, -6, 12, 16, 12, 0, 0, 0, false, r*0.86, g*0.86, b*0.86)
	for i := 0; i < 8; i++ {
		theta := float64(i) * math.Pi * 2.0 / 8.0
		px := float32(math.Cos(theta) * 5.0)
		pz := float32(math.Sin(theta) * 5.0)
		rotY := float32(float64(i)*math.Pi*-2.0/8.0 + math.Pi/2.0)
		drawSourcePartRad(tex, tentacleUV, px, 15, pz, -1, 0, -1, 2, 18, 2, ageTicks, rotY, 0, false, r, g, b)
	}
}

func drawWolfModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg, ageTicks float32) {
	// Translation reference:
	// - net.minecraft.src.ModelWolf
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 6, 6, 4)
	earUV := cuboidUVFromTextureOffset(tex, 16, 14, 2, 2, 1)
	noseUV := cuboidUVFromTextureOffset(tex, 0, 10, 3, 3, 4)
	bodyUV := cuboidUVFromTextureOffset(tex, 18, 14, 6, 9, 6)
	maneUV := cuboidUVFromTextureOffset(tex, 21, 0, 8, 6, 7)
	legUV := cuboidUVFromTextureOffset(tex, 0, 18, 2, 8, 2)
	tailUV := cuboidUVFromTextureOffset(tex, 9, 18, 2, 8, 2)

	swingRad := swingDeg * float32(math.Pi/180.0)
	legA := swingRad * 1.4
	tailY := float32(math.Cos(float64(ageTicks*0.6662))) * 1.4 * 0.8
	tailX := ageTicks

	drawSourcePartRad(tex, headUV, -1, 13.5, -7, -3, -3, -2, 6, 6, 4, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, earUV, -1, 13.5, -7, -3, -5, 0, 2, 2, 1, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, earUV, -1, 13.5, -7, 1, -5, 0, 2, 2, 1, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, noseUV, -1, 13.5, -7, -1.5, 0, -5, 3, 3, 4, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)

	drawSourcePartRad(tex, bodyUV, 0, 14, 2, -4, -2, -3, 6, 9, 6, float32(math.Pi)/2, 0, 0, false, r*0.85, g*0.85, b*0.85)
	drawSourcePartRad(tex, maneUV, -1, 14, 2, -4, -3, -3, 8, 6, 7, float32(math.Pi)/2, 0, 0, false, r*0.93, g*0.93, b*0.93)
	drawSourcePartRad(tex, legUV, -2.5, 16, 7, -1, 0, -1, 2, 8, 2, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 0.5, 16, 7, -1, 0, -1, 2, 8, 2, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, -2.5, 16, -4, -1, 0, -1, 2, 8, 2, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, legUV, 0.5, 16, -4, -1, 0, -1, 2, 8, 2, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, tailUV, -1, 12, 8, -1, 0, -1, 2, 8, 2, tailX, tailY, 0, false, r, g, b)
}

func drawOcelotModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg, ageTicks float32) {
	// Translation reference:
	// - net.minecraft.src.ModelOcelot (walking/default state)
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	headMainUV := cuboidUVFromTextureOffset(tex, 0, 0, 5, 4, 5)
	headNoseUV := cuboidUVFromTextureOffset(tex, 0, 24, 3, 2, 2)
	ear1UV := cuboidUVFromTextureOffset(tex, 0, 10, 1, 1, 2)
	ear2UV := cuboidUVFromTextureOffset(tex, 6, 10, 1, 1, 2)
	bodyUV := cuboidUVFromTextureOffset(tex, 20, 0, 4, 16, 6)
	tail1UV := cuboidUVFromTextureOffset(tex, 0, 15, 1, 8, 1)
	tail2UV := cuboidUVFromTextureOffset(tex, 4, 15, 1, 8, 1)
	backLegUV := cuboidUVFromTextureOffset(tex, 8, 13, 2, 6, 2)
	frontLegUV := cuboidUVFromTextureOffset(tex, 40, 0, 2, 10, 2)

	swingRad := swingDeg * float32(math.Pi/180.0)
	legA := swingRad
	tail2X := 1.7278761 + float32(math.Pi)/4*float32(math.Cos(float64(ageTicks*0.6662)))*0.8

	drawSourcePartRad(tex, headMainUV, 0, 15, -9, -2.5, -2, -3, 5, 4, 5, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, headNoseUV, 0, 15, -9, -1.5, 0, -4, 3, 2, 2, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, ear1UV, 0, 15, -9, -2, -3, 0, 1, 1, 2, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, ear2UV, 0, 15, -9, 1, -3, 0, 1, 1, 2, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)

	drawSourcePartRad(tex, bodyUV, 0, 12, -10, -2, 3, -8, 4, 16, 6, float32(math.Pi)/2, 0, 0, false, r*0.85, g*0.85, b*0.85)
	drawSourcePartRad(tex, tail1UV, 0, 15, 8, -0.5, 0, 0, 1, 8, 1, 0.9, 0, 0, false, r, g, b)
	drawSourcePartRad(tex, tail2UV, 0, 20, 14, -0.5, 0, 0, 1, 8, 1, tail2X, 0, 0, false, r, g, b)

	drawSourcePartRad(tex, backLegUV, 1.1, 18, 5, -1, 0, 1, 2, 6, 2, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, backLegUV, -1.1, 18, 5, -1, 0, 1, 2, 6, 2, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, frontLegUV, 1.2, 13.8, -5, -1, 0, 0, 2, 10, 2, -legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
	drawSourcePartRad(tex, frontLegUV, -1.2, 13.8, -5, -1, 0, 0, 2, 10, 2, legA, 0, 0, false, r*0.95, g*0.95, b*0.95)
}

func drawVillagerModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg float32) {
	// Translation reference:
	// - net.minecraft.src.ModelVillager (constructor + setRotationAngles)
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}

	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 8, 10, 8)
	noseUV := cuboidUVFromTextureOffset(tex, 24, 0, 2, 4, 2)
	bodyUV := cuboidUVFromTextureOffset(tex, 16, 20, 8, 12, 6)
	armUV := cuboidUVFromTextureOffset(tex, 44, 22, 4, 8, 4)
	armMidUV := cuboidUVFromTextureOffset(tex, 40, 38, 8, 4, 4)
	legUV := cuboidUVFromTextureOffset(tex, 0, 22, 4, 12, 4)

	swingRad := swingDeg * float32(math.Pi/180.0)
	rightLegX := -swingRad * 0.7
	leftLegX := swingRad * 0.7

	// Head + integrated nose.
	drawSourcePartRad(tex, headUV, 0, 0, 0, -4, -10, -4, 8, 10, 8, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, noseUV, 0, 0, 0, -1, -3, -6, 2, 4, 2, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)

	drawSourcePartRad(tex, bodyUV, 0, 0, 0, -4, 0, -3, 8, 12, 6, 0, 0, 0, false, r, g, b)
	drawSourcePartRad(tex, armUV, 0, 2, 0, -8, -2, -2, 4, 8, 4, -0.75, 0, 0, false, r, g, b)
	drawSourcePartRad(tex, armUV, 0, 2, 0, 4, -2, -2, 4, 8, 4, -0.75, 0, 0, false, r, g, b)
	drawSourcePartRad(tex, armMidUV, 0, 2, 0, -4, 2, -2, 8, 4, 4, -0.75, 0, 0, false, r, g, b)

	drawSourcePartRad(tex, legUV, -2, 12, 0, -2, 0, -2, 4, 12, 4, rightLegX, 0, 0, false, r*0.96, g*0.96, b*0.96)
	drawSourcePartRad(tex, legUV, 2, 12, 0, -2, 0, -2, 4, 12, 4, leftLegX, 0, 0, true, r*0.96, g*0.96, b*0.96)
}

func drawQuadrupedModel(p entityModelProfile, tex *texture2D, pitch, swingDeg float32) {
	bodyR, bodyG, bodyB := p.colorR*0.86, p.colorG*0.86, p.colorB*0.86
	legR, legG, legB := p.colorR*0.95, p.colorG*0.95, p.colorB*0.95
	if tex != nil {
		bodyR, bodyG, bodyB = 1, 1, 1
		legR, legG, legB = 1, 1, 1
	}

	bodyW, bodyH, bodyD := float32(0.80), float32(0.55), float32(1.20)
	headW, headH, headD := float32(0.55), float32(0.55), float32(0.55)
	legW, legH, legD := float32(0.22), float32(0.62), float32(0.22)
	legOffsetX := float32(0.26)
	legOffsetZ := float32(0.35)

	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 8, 8, 8)
	bodyUV := cuboidUVFromTextureOffset(tex, 28, 8, 10, 16, 8)
	legUV := cuboidUVFromTextureOffset(tex, 0, 16, 4, 6, 4)

	drawModelPartUV(0, 0.90, 0, bodyW, bodyH, bodyD, tex, bodyUV, bodyR, bodyG, bodyB)
	drawModelPartRotUV(0, 1.00, -0.74, headW, headH, headD, pitch, 0, 0, tex, headUV, 1, 1, 1)

	drawModelLimbUV(-legOffsetX, legH, -legOffsetZ, legW, legH, legD, swingDeg, tex, legUV, legR, legG, legB)
	drawModelLimbUV(legOffsetX, legH, -legOffsetZ, legW, legH, legD, -swingDeg, tex, legUV, legR, legG, legB)
	drawModelLimbUV(-legOffsetX, legH, legOffsetZ, legW, legH, legD, -swingDeg, tex, legUV, legR, legG, legB)
	drawModelLimbUV(legOffsetX, legH, legOffsetZ, legW, legH, legD, swingDeg, tex, legUV, legR, legG, legB)
}

func drawCreeperModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg float32) {
	// Translation reference:
	// - net.minecraft.src.ModelCreeper
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}

	headUV := cuboidUVFromTextureOffset(tex, 0, 0, 8, 8, 8)
	bodyUV := cuboidUVFromTextureOffset(tex, 16, 16, 8, 12, 4)
	legUV := cuboidUVFromTextureOffset(tex, 0, 16, 4, 6, 4)
	swingRad := swingDeg * float32(math.Pi/180.0)
	legA := swingRad * 1.4

	drawSourcePartRad(tex, headUV, 0, 4, 0, -4, -8, -4, 8, 8, 8, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, bodyUV, 0, 4, 0, -4, 0, -2, 8, 12, 4, 0, 0, 0, false, r*0.82, g*0.82, b*0.82)
	drawSourcePartRad(tex, legUV, -2, 16, 4, -2, 0, -2, 4, 6, 4, legA, 0, 0, false, r*0.92, g*0.92, b*0.92)
	drawSourcePartRad(tex, legUV, 2, 16, 4, -2, 0, -2, 4, 6, 4, -legA, 0, 0, false, r*0.92, g*0.92, b*0.92)
	drawSourcePartRad(tex, legUV, -2, 16, -4, -2, 0, -2, 4, 6, 4, -legA, 0, 0, false, r*0.92, g*0.92, b*0.92)
	drawSourcePartRad(tex, legUV, 2, 16, -4, -2, 0, -2, 4, 6, 4, legA, 0, 0, false, r*0.92, g*0.92, b*0.92)
}

func drawSpiderModel(p entityModelProfile, tex *texture2D, headYaw, pitch, swingDeg float32) {
	// Translation reference:
	// - net.minecraft.src.ModelSpider (constructor + setRotationAngles)
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}

	headUV := cuboidUVFromTextureOffset(tex, 32, 4, 8, 8, 8)
	neckUV := cuboidUVFromTextureOffset(tex, 0, 0, 6, 6, 6)
	bodyUV := cuboidUVFromTextureOffset(tex, 0, 12, 10, 8, 12)
	legUV := cuboidUVFromTextureOffset(tex, 18, 0, 16, 2, 2)

	drawSourcePartRad(tex, headUV, 0, 15, -3, -4, -4, -8, 8, 8, 8, pitch*float32(math.Pi/180.0), headYaw*float32(math.Pi/180.0), 0, false, r, g, b)
	drawSourcePartRad(tex, neckUV, 0, 15, 0, -3, -3, -3, 6, 6, 6, 0, 0, 0, false, r*0.90, g*0.90, b*0.90)
	drawSourcePartRad(tex, bodyUV, 0, 15, 9, -5, -4, -6, 10, 8, 12, 0, 0, 0, false, r*0.82, g*0.82, b*0.82)

	phase := swingDeg * float32(math.Pi/180.0)
	limbSwing := phase * 2.0
	limbAmount := float32(0.85)
	baseZ := float32(math.Pi / 4.0)
	baseY := float32(0.3926991)
	y := [8]float32{
		baseY * 2.0, -baseY * 2.0,
		baseY * 1.0, -baseY * 1.0,
		-baseY * 1.0, baseY * 1.0,
		-baseY * 2.0, baseY * 2.0,
	}
	z := [8]float32{
		-baseZ, baseZ,
		-baseZ * 0.74, baseZ * 0.74,
		-baseZ * 0.74, baseZ * 0.74,
		-baseZ, baseZ,
	}

	var11 := -(float32(math.Cos(float64(limbSwing*0.6662*2.0+0.0))) * 0.4) * limbAmount
	var12 := -(float32(math.Cos(float64(limbSwing*0.6662*2.0+float32(math.Pi)))) * 0.4) * limbAmount
	var13 := -(float32(math.Cos(float64(limbSwing*0.6662*2.0+float32(math.Pi)/2.0))) * 0.4) * limbAmount
	var14 := -(float32(math.Cos(float64(limbSwing*0.6662*2.0+float32(math.Pi)*1.5))) * 0.4) * limbAmount
	var15 := float32(math.Abs(float64(float32(math.Sin(float64(limbSwing*0.6662+0.0))) * 0.4 * limbAmount)))
	var16 := float32(math.Abs(float64(float32(math.Sin(float64(limbSwing*0.6662+float32(math.Pi)))) * 0.4 * limbAmount)))
	var17 := float32(math.Abs(float64(float32(math.Sin(float64(limbSwing*0.6662+float32(math.Pi)/2.0))) * 0.4 * limbAmount)))
	var18 := float32(math.Abs(float64(float32(math.Sin(float64(limbSwing*0.6662+float32(math.Pi)*1.5))) * 0.4 * limbAmount)))

	y[0] += var11
	y[1] += -var11
	y[2] += var12
	y[3] += -var12
	y[4] += var13
	y[5] += -var13
	y[6] += var14
	y[7] += -var14
	z[0] += var15
	z[1] += -var15
	z[2] += var16
	z[3] += -var16
	z[4] += var17
	z[5] += -var17
	z[6] += var18
	z[7] += -var18

	// Left legs (1,3,5,7): addBox(-15,-1,-1,16,2,2) at x=-4.
	// Right legs (2,4,6,8): addBox(-1,-1,-1,16,2,2) at x=4.
	legZPos := [4]float32{2, 1, 0, -1}
	for i := 0; i < 4; i++ {
		drawSourcePartRad(tex, legUV, -4, 15, legZPos[i], -15, -1, -1, 16, 2, 2, 0, y[i*2], z[i*2], false, r*0.92, g*0.92, b*0.92)
		drawSourcePartRad(tex, legUV, 4, 15, legZPos[i], -1, -1, -1, 16, 2, 2, 0, y[i*2+1], z[i*2+1], false, r*0.92, g*0.92, b*0.92)
	}
}

func drawSlimeModel(p entityModelProfile, tex *texture2D, squash float32, slimeSize int8) {
	// Translation reference:
	// - net.minecraft.src.ModelSlime
	// - net.minecraft.src.RenderSlime#scaleSlime(EntitySlime,float)
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	outerUV := cuboidUVFromTextureOffset(tex, 0, 0, 8, 8, 8)
	innerUV := cuboidUVFromTextureOffset(tex, 0, 16, 6, 6, 6)
	rightEyeUV := cuboidUVFromTextureOffset(tex, 32, 0, 2, 2, 2)
	leftEyeUV := cuboidUVFromTextureOffset(tex, 32, 4, 2, 2, 2)
	mouthUV := cuboidUVFromTextureOffset(tex, 32, 8, 1, 1, 1)

	size := float32(slimeSize)
	if size < 1.0 {
		size = 1.0
	}
	squish := squash / (size*0.5 + 1.0)
	inv := float32(1.0) / (squish + 1.0)

	gl.PushMatrix()
	gl.Scalef(inv*size, (1.0/inv)*size, inv*size)
	drawSourcePartRad(tex, outerUV, 0, 0, 0, -4, 16, -4, 8, 8, 8, 0, 0, 0, false, r*0.82, g*0.82, b*0.82)
	drawSourcePartRad(tex, innerUV, 0, 0, 0, -3, 17, -3, 6, 6, 6, 0, 0, 0, false, r, g, b)
	drawSourcePartRad(tex, rightEyeUV, 0, 0, 0, -3.25, 18, -3.5, 2, 2, 2, 0, 0, 0, false, r, g, b)
	drawSourcePartRad(tex, leftEyeUV, 0, 0, 0, 1.25, 18, -3.5, 2, 2, 2, 0, 0, 0, false, r, g, b)
	drawSourcePartRad(tex, mouthUV, 0, 0, 0, 0, 21, -3.5, 1, 1, 1, 0, 0, 0, false, r, g, b)
	gl.PopMatrix()
}

func drawFallbackModel(p entityModelProfile, tex *texture2D) {
	r, g, b := p.colorR, p.colorG, p.colorB
	if tex != nil {
		r, g, b = 1, 1, 1
	}
	drawModelPartUV(0, 0.50, 0, 0.60, 1.00, 0.60, tex, fullTextureUV(), r, g, b)
}

func drawModelLimbUV(px, py, pz, sx, sy, sz, rotX float32, tex *texture2D, uv cuboidUV, r, g, b float32) {
	gl.PushMatrix()
	gl.Translatef(px, py, pz)
	gl.Rotatef(rotX, 1, 0, 0)
	gl.Translatef(0, -sy*0.5, 0)
	drawModelPartUV(0, 0, 0, sx, sy, sz, tex, uv, r, g, b)
	gl.PopMatrix()
}

func drawModelPartRotUV(cx, cy, cz, sx, sy, sz, rotX, rotY, rotZ float32, tex *texture2D, uv cuboidUV, r, g, b float32) {
	gl.PushMatrix()
	gl.Translatef(cx, cy, cz)
	if rotY != 0 {
		gl.Rotatef(rotY, 0, 1, 0)
	}
	if rotX != 0 {
		gl.Rotatef(rotX, 1, 0, 0)
	}
	if rotZ != 0 {
		gl.Rotatef(rotZ, 0, 0, 1)
	}
	drawModelPartUV(0, 0, 0, sx, sy, sz, tex, uv, r, g, b)
	gl.PopMatrix()
}

func drawModelPartUV(cx, cy, cz, sx, sy, sz float32, tex *texture2D, uv cuboidUV, r, g, b float32) {
	drawModelPartUVEx(cx, cy, cz, sx, sy, sz, tex, uv, false, r, g, b)
}

type modelVertex struct {
	x float32
	y float32
	z float32
}

func drawModelPartUVEx(cx, cy, cz, sx, sy, sz float32, tex *texture2D, uv cuboidUV, mirror bool, r, g, b float32) {
	x0 := cx - sx*0.5
	y0 := cy - sy*0.5
	z0 := cz - sz*0.5
	x1 := x0 + sx
	y1 := y0 + sy
	z1 := z0 + sz

	if tex == nil {
		gl.PushMatrix()
		gl.Translatef(x0, y0, z0)
		gl.Scalef(sx, sy, sz)
		drawCubeFaces(0, 0, 0, r, g, b, fullFaces)
		gl.PopMatrix()
		return
	}

	if mirror {
		// ModelBox(mirror=true): swap x bounds before creating vertices.
		x0, x1 = x1, x0
	}

	v000 := modelVertex{x: x0, y: y0, z: z0}
	v100 := modelVertex{x: x1, y: y0, z: z0}
	v110 := modelVertex{x: x1, y: y1, z: z0}
	v010 := modelVertex{x: x0, y: y1, z: z0}
	v001 := modelVertex{x: x0, y: y0, z: z1}
	v101 := modelVertex{x: x1, y: y0, z: z1}
	v111 := modelVertex{x: x1, y: y1, z: z1}
	v011 := modelVertex{x: x0, y: y1, z: z1}

	emitModelQuad := func(a, b, c, d modelVertex, faceUV uvRect, cr, cg, cb float32) {
		if mirror {
			// Translation reference:
			// - net.minecraft.src.TexturedQuad#flipFace()
			// flipFace reverses vertex order but keeps each vertex UV attached.
			// For the reversed order [d,c,b,a], UVs become [u1,v1],[u0,v1],[u0,v0],[u1,v0].
			emitTexturedFaceFlipped(d.x, d.y, d.z, c.x, c.y, c.z, b.x, b.y, b.z, a.x, a.y, a.z, faceUV, cr, cg, cb)
			return
		}
		emitTexturedFace(a.x, a.y, a.z, b.x, b.y, b.z, c.x, c.y, c.z, d.x, d.y, d.z, faceUV, cr, cg, cb)
	}

	tex.bind()
	gl.Enable(gl.TEXTURE_2D)
	gl.Begin(gl.QUADS)
	// Follow ModelBox quad construction order exactly after Y-up conversion:
	// quad0(East), quad1(West), quad2(Down), quad3(Up), quad4(North), quad5(South).
	// Note: after converting model Y-down to world Y-up, "Down"/"Up" UV labels land on top/bottom planes respectively.
	emitModelQuad(v111, v110, v100, v101, uv.East, r*0.65, g*0.65, b*0.65)
	emitModelQuad(v010, v011, v001, v000, uv.West, r*0.65, g*0.65, b*0.65)
	emitModelQuad(v111, v011, v010, v110, uv.Down, r*1.00, g*1.00, b*1.00)
	emitModelQuad(v100, v000, v001, v101, uv.Up, r*0.52, g*0.52, b*0.52)
	emitModelQuad(v110, v010, v000, v100, uv.North, r*0.80, g*0.80, b*0.80)
	emitModelQuad(v011, v111, v101, v001, uv.South, r*0.80, g*0.80, b*0.80)
	gl.End()
}

// drawModelPartUVRawEx follows ModelBox quad order directly without Y-up remapping.
// This is required by first-person arm rendering (RenderPlayer#renderFirstPersonArm),
// which uses ModelRenderer raw coordinates.
func drawModelPartUVRawEx(cx, cy, cz, sx, sy, sz float32, tex *texture2D, uv cuboidUV, mirror bool, r, g, b float32) {
	x0 := cx - sx*0.5
	y0 := cy - sy*0.5
	z0 := cz - sz*0.5
	x1 := x0 + sx
	y1 := y0 + sy
	z1 := z0 + sz

	if tex == nil {
		gl.PushMatrix()
		gl.Translatef(x0, y0, z0)
		gl.Scalef(sx, sy, sz)
		drawCubeFaces(0, 0, 0, r, g, b, fullFaces)
		gl.PopMatrix()
		return
	}

	if mirror {
		x0, x1 = x1, x0
	}

	v000 := modelVertex{x: x0, y: y0, z: z0}
	v100 := modelVertex{x: x1, y: y0, z: z0}
	v110 := modelVertex{x: x1, y: y1, z: z0}
	v010 := modelVertex{x: x0, y: y1, z: z0}
	v001 := modelVertex{x: x0, y: y0, z: z1}
	v101 := modelVertex{x: x1, y: y0, z: z1}
	v111 := modelVertex{x: x1, y: y1, z: z1}
	v011 := modelVertex{x: x0, y: y1, z: z1}

	emitModelQuad := func(a, b, c, d modelVertex, faceUV uvRect, cr, cg, cb float32) {
		if mirror {
			emitTexturedFaceFlipped(d.x, d.y, d.z, c.x, c.y, c.z, b.x, b.y, b.z, a.x, a.y, a.z, faceUV, cr, cg, cb)
			return
		}
		emitTexturedFace(a.x, a.y, a.z, b.x, b.y, b.z, c.x, c.y, c.z, d.x, d.y, d.z, faceUV, cr, cg, cb)
	}

	tex.bind()
	gl.Enable(gl.TEXTURE_2D)
	gl.Begin(gl.QUADS)
	// Exact ModelBox quads:
	// quad0(East), quad1(West), quad2(Down/y-min), quad3(Up/y-max), quad4(North/z-min), quad5(South/z-max).
	emitModelQuad(v101, v100, v110, v111, uv.East, r*0.65, g*0.65, b*0.65)
	emitModelQuad(v000, v001, v011, v010, uv.West, r*0.65, g*0.65, b*0.65)
	emitModelQuad(v101, v001, v000, v100, uv.Down, r*0.52, g*0.52, b*0.52)
	emitModelQuad(v110, v010, v011, v111, uv.Up, r*1.00, g*1.00, b*1.00)
	emitModelQuad(v100, v000, v010, v110, uv.North, r*0.80, g*0.80, b*0.80)
	emitModelQuad(v001, v101, v111, v011, uv.South, r*0.80, g*0.80, b*0.80)
	gl.End()
}

func emitTexturedFace(
	x0, y0, z0 float32,
	x1, y1, z1 float32,
	x2, y2, z2 float32,
	x3, y3, z3 float32,
	uv uvRect,
	r, g, b float32,
) {
	gl.Color3f(r, g, b)
	// Match TexturedQuad ctor mapping used by ModelBox:
	// v0=(u2,v1), v1=(u1,v1), v2=(u1,v2), v3=(u2,v2).
	gl.TexCoord2f(uv.u1, uv.v0)
	gl.Vertex3f(x0, y0, z0)
	gl.TexCoord2f(uv.u0, uv.v0)
	gl.Vertex3f(x1, y1, z1)
	gl.TexCoord2f(uv.u0, uv.v1)
	gl.Vertex3f(x2, y2, z2)
	gl.TexCoord2f(uv.u1, uv.v1)
	gl.Vertex3f(x3, y3, z3)
}

func emitTexturedFaceFlipped(
	x0, y0, z0 float32,
	x1, y1, z1 float32,
	x2, y2, z2 float32,
	x3, y3, z3 float32,
	uv uvRect,
	r, g, b float32,
) {
	gl.Color3f(r, g, b)
	gl.TexCoord2f(uv.u1, uv.v1)
	gl.Vertex3f(x0, y0, z0)
	gl.TexCoord2f(uv.u0, uv.v1)
	gl.Vertex3f(x1, y1, z1)
	gl.TexCoord2f(uv.u0, uv.v0)
	gl.Vertex3f(x2, y2, z2)
	gl.TexCoord2f(uv.u1, uv.v0)
	gl.Vertex3f(x3, y3, z3)
}

func fullTextureUV() cuboidUV {
	r := uvRect{u0: 0, v0: 0, u1: 1, v1: 1}
	return cuboidUV{
		Down:  r,
		Up:    r,
		North: r,
		South: r,
		West:  r,
		East:  r,
	}
}

func cuboidUVFromTextureOffset(tex *texture2D, offU, offV, dx, dy, dz int) cuboidUV {
	if tex == nil || tex.Width <= 0 || tex.Height <= 0 || dx <= 0 || dy <= 0 || dz <= 0 {
		return fullTextureUV()
	}
	w := float32(tex.Width)
	h := float32(tex.Height)

	// Exact ModelBox texture rectangles (line75-line80 in MCP 1.6.4).
	east := pxRect(offU+dz+dx, offV+dz, dz, dy, w, h)
	west := pxRect(offU+0, offV+dz, dz, dy, w, h)
	down := pxRect(offU+dz, offV+0, dx, dz, w, h)
	up := uvRect{
		u0: float32(offU+dz+dx) / w,
		v0: float32(offV+dz) / h,
		u1: float32(offU+dz+dx+dx) / w,
		v1: float32(offV) / h,
	}
	north := pxRect(offU+dz, offV+dz, dx, dy, w, h)
	south := pxRect(offU+dz+dx+dz, offV+dz, dx, dy, w, h)

	return cuboidUV{
		Down:  down,
		Up:    up,
		North: north,
		South: south,
		West:  west,
		East:  east,
	}
}

func pxRect(u, v, ww, hh int, texW, texH float32) uvRect {
	return uvRect{
		u0: float32(u) / texW,
		v0: float32(v) / texH,
		u1: float32(u+ww) / texW,
		v1: float32(v+hh) / texH,
	}
}

func drawSourcePartRad(
	tex *texture2D,
	uv cuboidUV,
	pivotX, pivotY, pivotZ float32,
	boxX, boxY, boxZ float32,
	boxW, boxH, boxD int,
	rotXRad, rotYRad, rotZRad float32,
	mirror bool,
	r, g, b float32,
) {
	drawSourcePartRadInflate(tex, uv, pivotX, pivotY, pivotZ, boxX, boxY, boxZ, boxW, boxH, boxD, 0, rotXRad, rotYRad, rotZRad, mirror, r, g, b)
}

func drawSourcePartRadInflate(
	tex *texture2D,
	uv cuboidUV,
	pivotX, pivotY, pivotZ float32,
	boxX, boxY, boxZ float32,
	boxW, boxH, boxD int,
	inflate float32,
	rotXRad, rotYRad, rotZRad float32,
	mirror bool,
	r, g, b float32,
) {
	if boxW <= 0 || boxH <= 0 || boxD <= 0 {
		return
	}

	// ModelRenderer coordinates are in "pixels" with Y-down, scale=1/16.
	// Convert to world Y-up and keep vanilla's rotation order Z -> Y -> X.
	const inv16 = float32(1.0 / 16.0)
	pX := pivotX * inv16
	pY := (24.0 - pivotY) * inv16
	pZ := pivotZ * inv16

	bx := boxX - inflate
	by := boxY - inflate
	bz := boxZ - inflate
	bw := float32(boxW) + inflate*2
	bh := float32(boxH) + inflate*2
	bd := float32(boxD) + inflate*2

	lcX := (bx + bw*0.5) * inv16
	lcY := -(by + bh*0.5) * inv16
	lcZ := (bz + bd*0.5) * inv16

	sX := bw * inv16
	sY := bh * inv16
	sZ := bd * inv16

	gl.PushMatrix()
	gl.Translatef(pX, pY, pZ)
	if rotZRad != 0 {
		gl.Rotatef(-rotZRad*180.0/float32(math.Pi), 0, 0, 1)
	}
	if rotYRad != 0 {
		gl.Rotatef(rotYRad*180.0/float32(math.Pi), 0, 1, 0)
	}
	if rotXRad != 0 {
		gl.Rotatef(-rotXRad*180.0/float32(math.Pi), 1, 0, 0)
	}
	drawModelPartUVEx(lcX, lcY, lcZ, sX, sY, sZ, tex, uv, mirror, r, g, b)
	gl.PopMatrix()
}

func angleFromByte(v int8) float32 {
	return float32(uint8(v)) * 360.0 / 256.0
}

func normalizeDegrees(v float32) float32 {
	for v >= 180 {
		v -= 360
	}
	for v < -180 {
		v += 360
	}
	return v
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
