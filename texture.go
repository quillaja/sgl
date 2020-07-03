package sgl

import (
	"fmt"
	"image"
	"image/draw"
	"os"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// OpenImages opens the images specified by filename and converts them to
// RGBA format.
func OpenImages(filenames ...string) ([]*image.RGBA, error) {
	images := make([]*image.RGBA, 0, len(filenames))

	for _, file := range filenames {
		imgFile, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("could not open %s: %w", file, err)
		}
		img, _, err := image.Decode(imgFile)
		if err != nil {
			return nil, fmt.Errorf("could not decode %s: %w", file, err)
		}

		rgba := image.NewRGBA(img.Bounds())
		if rgba.Stride != rgba.Rect.Size().X*4 { // NOTE: pointless check?
			return nil, fmt.Errorf("unsupported stride")
		}
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

		images = append(images, rgba)
	}

	return images, nil
}

type Texture2D struct {
	ID            uint32
	Width, Height int32
}

func NewTexture2D(rgba *image.RGBA) (*Texture2D, error) {
	texture := &Texture2D{
		Width:  int32(rgba.Bounds().Dx()),
		Height: int32(rgba.Bounds().Dy()),
	}

	gl.GenTextures(1, &texture.ID)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture.ID)
	// TODO: update to sampler object?
	// https://stackoverflow.com/questions/30759028/changing-texture-parameters-at-runtime
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA, // internal texture format
		texture.Width,
		texture.Height,
		0,
		gl.RGBA, // image format
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	gl.BindTexture(gl.TEXTURE_2D, 0) // unbind texture

	return texture, nil
}

func (tex *Texture2D) Delete() {
	gl.DeleteTextures(1, &tex.ID)
}

func (tex *Texture2D) Reload(img *image.RGBA) {
	gl.BindTexture(gl.TEXTURE_2D, tex.ID)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0,
		tex.Width,
		tex.Height,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(img.Pix))
}
