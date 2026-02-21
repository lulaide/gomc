//go:build cgo

package gui

import (
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/go-gl/gl/v2.1/gl"
)

type fontRenderer struct {
	tex          *texture2D
	charWidth    [256]int
	glyphIndex   map[rune]int
	colorCode    [32]int
	defaultColor int
}

func loadFontRenderer(asciiPath, fontTxtPath string) (*fontRenderer, error) {
	tex, rgba, err := loadTexture2DWithFlip(asciiPath, true, false)
	if err != nil {
		return nil, err
	}

	allowed, err := loadAllowedCharacters(fontTxtPath)
	if err != nil || allowed == "" {
		// Translation reference:
		// - net.minecraft.src.ChatAllowedCharacters.getAllowedCharacters()
		// Vanilla swallows font.txt read failures; keep ASCII fallback so GUI text stays visible.
		allowed = defaultAllowedCharacters()
	}

	f := &fontRenderer{
		tex:          tex,
		glyphIndex:   make(map[rune]int, len(allowed)),
		defaultColor: 0xFFFFFF,
	}
	runeIdx := 0
	for _, r := range allowed {
		// Match Java indexOf() semantics: use first occurrence.
		if _, exists := f.glyphIndex[r]; !exists {
			f.glyphIndex[r] = runeIdx
		}
		runeIdx++
	}
	f.readFontTextureMetrics(rgba)
	f.initColorCodes()
	return f, nil
}

func (f *fontRenderer) delete() {
	if f == nil {
		return
	}
	f.tex.delete()
}

func (f *fontRenderer) initColorCodes() {
	for i := 0; i < 32; i++ {
		extra := ((i >> 3) & 1) * 85
		r := ((i >> 2) & 1) * 170
		g := ((i >> 1) & 1) * 170
		b := (i & 1) * 170
		r += extra
		g += extra
		b += extra
		if i == 6 {
			r += 85
		}
		if i >= 16 {
			r /= 4
			g /= 4
			b /= 4
		}
		f.colorCode[i] = (r&255)<<16 | (g&255)<<8 | (b & 255)
	}
}

func loadAllowedCharacters(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", path, err)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var b strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		b.WriteString(line)
	}
	return b.String(), nil
}

func defaultAllowedCharacters() string {
	var b strings.Builder
	for ch := 32; ch <= 126; ch++ {
		b.WriteByte(byte(ch))
	}
	return b.String()
}

// Translation reference:
// - net.minecraft.src.FontRenderer.readFontTexture()
func (f *fontRenderer) readFontTextureMetrics(img *image.RGBA) {
	if img == nil {
		return
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	cellH := h / 16
	cellW := w / 16
	padding := 1
	scale := 8.0 / float64(cellW)

	for ch := 0; ch < 256; ch++ {
		cellX := ch % 16
		cellY := ch / 16
		if ch == 32 {
			f.charWidth[ch] = 3 + padding
		}

		col := cellW - 1
		for ; col >= 0; col-- {
			pixelX := cellX*cellW + col
			empty := true
			for row := 0; row < cellH; row++ {
				pixelY := cellY*cellH + row
				if img.RGBAAt(pixelX, pixelY).A != 0 {
					empty = false
					break
				}
			}
			if !empty {
				break
			}
		}

		col++
		f.charWidth[ch] = int(0.5+float64(col)*scale) + padding
	}
}

func (f *fontRenderer) getStringWidth(text string) int {
	if f == nil || text == "" {
		return 0
	}

	width := 0
	bold := false
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\u00A7' && i+1 < len(runes) {
			i++
			code := rune(strings.ToLower(string(runes[i]))[0])
			if code == 'l' {
				bold = true
			} else if code == 'r' || (code >= '0' && code <= '9') || (code >= 'a' && code <= 'f') {
				bold = false
			}
			continue
		}
		w := f.getCharWidth(ch)
		width += w
		if bold {
			width++
		}
	}
	return width
}

func (f *fontRenderer) getCharWidth(ch rune) int {
	if ch == '\u00A7' {
		return 0
	}
	if ch == ' ' {
		return 4
	}
	glyph, ok := f.glyphForRune(ch)
	if !ok {
		return 0
	}
	return f.charWidth[glyph]
}

func (f *fontRenderer) glyphForRune(ch rune) (int, bool) {
	// Keep vanilla ASCII UI text stable even when font.txt is mismatched/garbled.
	if ch >= 32 && ch <= 126 {
		return int(ch), true
	}
	if idx, ok := f.glyphIndex[ch]; ok {
		glyph := idx + 32
		if glyph >= 0 && glyph < len(f.charWidth) {
			return glyph, true
		}
	}
	// Fallback keeps Latin GUI text visible when font.txt mapping is broken/mismatched.
	if ch >= 32 && ch <= 255 {
		return int(ch), true
	}
	return 0, false
}

func (f *fontRenderer) drawString(text string, x, y int, color int) int {
	return f.drawStringInternal(text, float32(x), float32(y), color, false)
}

func (f *fontRenderer) drawStringWithShadow(text string, x, y int, color int) int {
	shadowColor := ((color & 0xFCFCFC) >> 2) | (color & 0xFF000000)
	f.drawStringInternal(text, float32(x+1), float32(y+1), shadowColor, true)
	return f.drawStringInternal(text, float32(x), float32(y), color, false)
}

func (f *fontRenderer) drawCenteredString(text string, centerX, y int, color int) int {
	w := f.getStringWidth(text)
	return f.drawString(text, centerX-w/2, y, color)
}

func (f *fontRenderer) drawStringInternal(text string, x, y float32, color int, shadow bool) int {
	if f == nil || f.tex == nil || text == "" {
		return int(x)
	}

	gl.Enable(gl.TEXTURE_2D)
	f.tex.bind()

	if (color & 0xFF000000) == 0 {
		color |= 0xFF000000
	}

	alpha := float32((color>>24)&0xFF) / 255.0
	red := float32((color>>16)&0xFF) / 255.0
	green := float32((color>>8)&0xFF) / 255.0
	blue := float32(color&0xFF) / 255.0

	gl.Color4f(red, green, blue, alpha)
	posX := x
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\u00A7' && i+1 < len(runes) {
			i++
			codeRune := runes[i]
			code := strings.ToLower(string(codeRune))
			if len(code) == 1 {
				idx := strings.IndexRune("0123456789abcdef", rune(code[0]))
				if idx >= 0 {
					if shadow {
						idx += 16
					}
					c := f.colorCode[idx]
					cr := float32((c>>16)&0xFF) / 255.0
					cg := float32((c>>8)&0xFF) / 255.0
					cb := float32(c&0xFF) / 255.0
					gl.Color4f(cr, cg, cb, alpha)
				} else if code[0] == 'r' {
					gl.Color4f(red, green, blue, alpha)
				}
			}
			continue
		}

		if ch == ' ' {
			posX += 4
			continue
		}

		glyph, ok := f.glyphForRune(ch)
		if !ok {
			continue
		}

		charWidth := f.charWidth[glyph]
		if charWidth <= 0 {
			continue
		}

		srcW := charWidth - 1
		if srcW <= 0 {
			srcW = 1
		}
		srcX := (glyph % 16) * 8
		srcY := (glyph / 16) * 8
		drawTexturedRect(f.tex, posX, y, float32(srcW), 8, srcX, srcY, srcW, 8)
		posX += float32(charWidth)
	}

	gl.Color4f(1, 1, 1, 1)
	return int(posX)
}
