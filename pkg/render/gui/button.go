//go:build cgo

package gui

import "github.com/go-gl/gl/v2.1/gl"

type guiButton struct {
	ID      int
	X       int
	Y       int
	Width   int
	Height  int
	Label   string
	Enabled bool
	Visible bool
	Hovered bool
}

func newButton(id, x, y, width, height int, label string) *guiButton {
	return &guiButton{
		ID:      id,
		X:       x,
		Y:       y,
		Width:   width,
		Height:  height,
		Label:   label,
		Enabled: true,
		Visible: true,
	}
}

func (b *guiButton) contains(px, py int) bool {
	if b == nil || !b.Visible {
		return false
	}
	return px >= b.X && py >= b.Y && px < b.X+b.Width && py < b.Y+b.Height
}

func (b *guiButton) draw(font *fontRenderer, widgets *texture2D, mouseX, mouseY int) {
	if b == nil || !b.Visible {
		return
	}

	b.Hovered = b.contains(mouseX, mouseY)

	if widgets != nil {
		gl.Color4f(1, 1, 1, 1)
		hoverState := 1
		if !b.Enabled {
			hoverState = 0
		} else if b.Hovered {
			hoverState = 2
		}

		srcY := 46 + hoverState*20
		leftW := b.Width / 2
		rightW := b.Width - leftW

		drawTexturedRect(widgets, float32(b.X), float32(b.Y), float32(leftW), float32(b.Height), 0, srcY, leftW, b.Height)
		drawTexturedRect(widgets, float32(b.X+leftW), float32(b.Y), float32(rightW), float32(b.Height), 200-rightW, srcY, rightW, b.Height)
	} else {
		bg := 0xA0000000
		if !b.Enabled {
			bg = 0x70000000
		} else if b.Hovered {
			bg = 0xB0404040
		}
		drawSolidRect(b.X, b.Y, b.X+b.Width, b.Y+b.Height, bg)
		drawGradientRect(b.X, b.Y, b.X+b.Width, b.Y+1, 0xA0FFFFFF, 0x30FFFFFF)
		drawGradientRect(b.X, b.Y+b.Height-1, b.X+b.Width, b.Y+b.Height, 0x30000000, 0x90000000)
	}

	if font == nil {
		return
	}
	color := 14737632
	if !b.Enabled {
		color = -6250336
	} else if b.Hovered {
		color = 16777120
	}
	font.drawCenteredString(b.Label, b.X+b.Width/2, b.Y+(b.Height-8)/2, color)
}
