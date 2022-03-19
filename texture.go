package sgl

import (
	"fmt"
	"image"
	"image/draw"
	"os"

	"github.com/go-gl/gl/v3.3-core/gl"
)

func imageToRGBA(img image.Image) *image.RGBA {
	var bounds image.Rectangle
	switch img.(type) {
	case *image.Uniform:
		// Uniform images have huge bounds
		bounds = image.Rect(0, 0, 2, 2)
	default:
		bounds = img.Bounds()
	}
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
	return rgba
}

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

		images = append(images, imageToRGBA(img))
	}

	return images, nil
}

type Texture2D struct {
	ID            uint32
	Width, Height int32
}

/*
-made with tex2d
texture_2d			1 image
texture_cubemap		6 images
-made with tex3d
texture_2d_array 	N 2d images
texture_3d			1 3d image

params
filters: linear, nearest + (MIN or MAX)
wraps: clamp_to_edge, clamp_to_border, repeat, mirrored_repeat + (S,T,R)
border color
swizzle
mipmaps -- unsupported for now. SHOULD set TEXTURE_MAX_LEVEL to 0 (default is 1000).

internal formats/source formats
GL_DEPTH_COMPONENT
GL_DEPTH_STENCIL (only internal?)
GL_STENCIL_INDEX
GL_RED
GL_RG
GL_RGB (also BGR)
GL_RGBA (also BGR)
GL_RGB(A)16F -- for hdr, gbuffer, etc.

bits/channel type
*8 GL_UNSIGNED_BYTE
8 GL_BYTE
*16 GL_UNSIGNED_SHORT
16 GL_SHORT
32 GL_UNSIGNED_INT
32 GL_INT
*32 GL_FLOAT

-- setup + reload --
width/height/depth
texture kind
source format
internal format
number type
-- setup only --
filter[min,max]
wraps[s,r,t]
border color[rgba]
swizzle[rgba]

"supported" golang pkg image types:
Alpha	uint8
Alpha16 uint16 as [2]uint8 (big endian)
Gray	uint8 (seems same as Alpha, but represents RGB triplet)
Gray	uint16
NRGBA	uint8 (rgba)
NRGBA64 uint16 (rgba)
RGBA	uint8 (rgba)
RGBA64	uint16 (rgba)
Uniform	convert to small-sized(2x2) RGBA (or RGB if alpha=1)
others	convert to RGBA
*/

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
	gl.BindTexture(gl.TEXTURE_2D, 0)
}

// ReadImage gets a Go image from the texture.
func (tex *Texture2D) ReadImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(tex.Width), int(tex.Height)))
	gl.BindTexture(gl.TEXTURE_2D, tex.ID)
	gl.GetTexImage(gl.TEXTURE_2D, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))
	gl.BindTexture(gl.TEXTURE_2D, 0)

	flipVertically(img)
	return img
}
