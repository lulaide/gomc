//go:build cgo

package gui

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"os"
	"time"

	"github.com/go-gl/gl/v2.1/gl"
)

type texture2D struct {
	ID     uint32
	Width  int
	Height int

	animatedFrames [][]byte
	animatedFrame  int
}

func (t *texture2D) setWrapRepeat(repeat bool) {
	if t == nil || t.ID == 0 {
		return
	}
	mode := int32(gl.CLAMP_TO_EDGE)
	if repeat {
		mode = int32(gl.REPEAT)
	}
	t.bind()
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, mode)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, mode)
}

func newEmptyTexture2D(width, height int, nearest bool) (*texture2D, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid texture size %dx%d", width, height)
	}

	var id uint32
	gl.GenTextures(1, &id)
	gl.BindTexture(gl.TEXTURE_2D, id)

	filter := int32(gl.LINEAR)
	if nearest {
		filter = int32(gl.NEAREST)
	}
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, int32(gl.CLAMP_TO_EDGE))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, int32(gl.CLAMP_TO_EDGE))
	gl.TexImage2D(gl.TEXTURE_2D, 0, int32(gl.RGBA8), int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	return &texture2D{ID: id, Width: width, Height: height}, nil
}

func loadTexture2D(path string, nearest bool) (*texture2D, *image.RGBA, error) {
	return loadTexture2DWithFlip(path, nearest, true)
}

func loadTexture2DWithFlip(path string, nearest bool, flipVertical bool) (*texture2D, *image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open texture %q: %w", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, nil, fmt.Errorf("decode texture %q: %w", path, err)
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)
	if flipVertical {
		flipRGBAInPlace(rgba)
	}

	uploadWidth := rgba.Bounds().Dx()
	uploadHeight := rgba.Bounds().Dy()
	uploadPix := rgba.Pix
	animatedFrames, frameSize := extractAnimatedStripFrames(rgba)
	animatedFrame := -1
	if len(animatedFrames) > 1 && frameSize > 0 {
		// Translation reference:
		// - net.minecraft.src.TextureWaterFX / TextureWaterFlowFX / TextureLavaFX
		// Legacy animated textures are vertical strips. Keep a square GL texture and
		// advance frames at runtime, matching vanilla's tick-driven animation.
		uploadWidth = frameSize
		uploadHeight = frameSize
		animatedFrame = len(animatedFrames) - 1
		uploadPix = animatedFrames[animatedFrame]
	}

	var id uint32
	gl.GenTextures(1, &id)
	gl.BindTexture(gl.TEXTURE_2D, id)

	filter := int32(gl.LINEAR)
	if nearest {
		filter = int32(gl.NEAREST)
	}
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, int32(gl.CLAMP_TO_EDGE))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, int32(gl.CLAMP_TO_EDGE))
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		int32(gl.RGBA8),
		int32(uploadWidth),
		int32(uploadHeight),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(uploadPix),
	)

	return &texture2D{
		ID:             id,
		Width:          uploadWidth,
		Height:         uploadHeight,
		animatedFrames: animatedFrames,
		animatedFrame:  animatedFrame,
	}, rgba, nil
}

func extractAnimatedStripFrames(rgba *image.RGBA) ([][]byte, int) {
	if rgba == nil {
		return nil, 0
	}
	width := rgba.Bounds().Dx()
	height := rgba.Bounds().Dy()
	if width <= 0 || height <= width || height%width != 0 {
		return nil, 0
	}

	frameCount := height / width
	if frameCount <= 1 {
		return nil, 0
	}

	rowSize := width * 4
	frames := make([][]byte, frameCount)
	for frame := 0; frame < frameCount; frame++ {
		pix := make([]byte, width*width*4)
		baseY := frame * width
		for y := 0; y < width; y++ {
			srcOff := (baseY+y)*rgba.Stride + 0
			dstOff := y * rowSize
			copy(pix[dstOff:dstOff+rowSize], rgba.Pix[srcOff:srcOff+rowSize])
		}
		frames[frame] = pix
	}
	return frames, width
}

func (t *texture2D) advanceAnimatedFrame(now time.Time) {
	if t == nil || t.ID == 0 || len(t.animatedFrames) <= 1 || t.Width <= 0 || t.Height <= 0 {
		return
	}

	frame := int((now.UnixMilli() / 50) % int64(len(t.animatedFrames)))
	frame = len(t.animatedFrames) - 1 - frame
	if frame == t.animatedFrame {
		return
	}

	t.bind()
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		0,
		0,
		int32(t.Width),
		int32(t.Height),
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(t.animatedFrames[frame]),
	)
	t.animatedFrame = frame
}

func (t *texture2D) setAnimatedFrame(frame int) {
	if t == nil || t.ID == 0 || len(t.animatedFrames) <= 1 || t.Width <= 0 || t.Height <= 0 {
		return
	}
	n := len(t.animatedFrames)
	if n <= 0 {
		return
	}
	frame %= n
	if frame < 0 {
		frame += n
	}
	if frame == t.animatedFrame {
		return
	}

	t.bind()
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,
		0,
		0,
		int32(t.Width),
		int32(t.Height),
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(t.animatedFrames[frame]),
	)
	t.animatedFrame = frame
}

func (t *texture2D) bind() {
	if t == nil {
		return
	}
	gl.BindTexture(gl.TEXTURE_2D, t.ID)
}

func (t *texture2D) delete() {
	if t == nil || t.ID == 0 {
		return
	}
	id := t.ID
	gl.DeleteTextures(1, &id)
	t.ID = 0
}

func flipRGBAInPlace(img *image.RGBA) {
	if img == nil {
		return
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	stride := img.Stride
	rowBuf := make([]byte, w*4)
	for y := 0; y < h/2; y++ {
		top := y * stride
		bottom := (h - 1 - y) * stride
		copy(rowBuf, img.Pix[top:top+w*4])
		copy(img.Pix[top:top+w*4], img.Pix[bottom:bottom+w*4])
		copy(img.Pix[bottom:bottom+w*4], rowBuf)
	}
}

// src coordinates are top-left based (same convention as vanilla GUI textures).
func drawTexturedRect(tex *texture2D, dstX, dstY, dstW, dstH float32, srcX, srcY, srcW, srcH int) {
	if tex == nil || tex.Width == 0 || tex.Height == 0 {
		return
	}

	u0 := float32(srcX) / float32(tex.Width)
	v0 := float32(srcY) / float32(tex.Height)
	u1 := float32(srcX+srcW) / float32(tex.Width)
	v1 := float32(srcY+srcH) / float32(tex.Height)

	tex.bind()
	gl.Begin(gl.QUADS)
	gl.TexCoord2f(u0, v0)
	gl.Vertex2f(dstX, dstY)
	gl.TexCoord2f(u1, v0)
	gl.Vertex2f(dstX+dstW, dstY)
	gl.TexCoord2f(u1, v1)
	gl.Vertex2f(dstX+dstW, dstY+dstH)
	gl.TexCoord2f(u0, v1)
	gl.Vertex2f(dstX, dstY+dstH)
	gl.End()
}

func drawTexturedRectUV(tex *texture2D, dstX, dstY, dstW, dstH, u0, v0, u1, v1 float32) {
	if tex == nil || tex.Width == 0 || tex.Height == 0 {
		return
	}

	tex.bind()
	gl.Begin(gl.QUADS)
	gl.TexCoord2f(u0, v0)
	gl.Vertex2f(dstX, dstY)
	gl.TexCoord2f(u1, v0)
	gl.Vertex2f(dstX+dstW, dstY)
	gl.TexCoord2f(u1, v1)
	gl.Vertex2f(dstX+dstW, dstY+dstH)
	gl.TexCoord2f(u0, v1)
	gl.Vertex2f(dstX, dstY+dstH)
	gl.End()
}
