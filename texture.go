package sgl

import (
	"fmt"
	"image"
	"image/draw"
	"os"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type Texture2D struct {
	ID            uint32
	Width, Height int32
}

func NewTextureFile(file string) (*Texture2D, error) {
	imgFile, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("texture %q not found on disk: %v", file, err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	return NewTextureRGBA(rgba)
}

func NewTextureRGBA(rgba *image.RGBA) (*Texture2D, error) {
	texture := &Texture2D{
		Width:  int32(rgba.Rect.Size().X),
		Height: int32(rgba.Rect.Size().Y),
	}

	gl.GenTextures(1, &texture.ID)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture.ID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		texture.Width,
		texture.Height,
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	gl.BindTexture(gl.TEXTURE_2D, 0) // unbind texture

	return texture, nil
}
