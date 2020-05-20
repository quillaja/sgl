package sgl

import (
	"image"
	"image/draw"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"golang.org/x/image/font/basicfont"
)

func newFontTexture(face *basicfont.Face) (uint32, error) {
	// convert 'alpha' image to normal rgba image that opengl can use
	// rgba := face.Mask.(*image.Alpha)
	rgba := image.NewRGBA(face.Mask.Bounds())
	draw.DrawMask(rgba, rgba.Bounds(), image.White, image.ZP, face.Mask, image.ZP, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	gl.BindTexture(gl.TEXTURE_2D, 0) // unbind texture
	return texture, nil
}

type CharacterDict struct {
	dict          map[rune]Character
	font          uint32
	shader        uint32
	shaderProgram *Program
	fw, fh        float32
}

func NewCharacterDict(font *basicfont.Face) *CharacterDict {
	var err error

	textProgram := NewProgram()
	textProgram.AddShader(gl.VERTEX_SHADER, fontVertexShader,
		[]string{"projection", "model"},
		Attribute{Name: "vertex", Size: 4, Type: gl.FLOAT, Stride: 4 * SizeOfFloat, Offset: 0},
	)
	textProgram.AddShader(gl.FRAGMENT_SHADER, fontFragmentShader,
		[]string{"font", "textColor"},
	)

	err = textProgram.Build()
	if err != nil {
		panic(err)
	}
	defer textProgram.Delete()

	cd := &CharacterDict{
		dict:          makeCharacters(textProgram.ID, font),
		shaderProgram: textProgram,
		shader:        textProgram.ID,
		fw:            float32(font.Width),
		fh:            float32(font.Height + 1),
	}

	cd.font, err = newFontTexture(font)
	if err != nil {
		panic(err)
	}

	return cd
}

func (cd CharacterDict) Delete() {
	gl.DeleteTextures(1, &cd.font)
	for k := range cd.dict {
		cd.dict[k].delete()
	}
	cd.shaderProgram.Delete()
}

// (0, 0) are in the top left of the screen (inverted Y compared to standard opengl)
func (cd CharacterDict) DrawString(text string, x, y, scale float32, color mgl32.Vec3, width, height float32) {
	gl.UseProgram(cd.shader)

	// gl.ActiveTexture(gl.TEXTURE0) // this is implicit here.
	gl.BindTexture(gl.TEXTURE_2D, cd.font) // load texture into uniform 2d texture TEXTURE0

	// 'vars' in vertex shader
	// vertAttrib := uint32(gl.GetAttribLocation(shader, gl.Str("vertex\x00")))
	projectionUniform := gl.GetUniformLocation(cd.shader, gl.Str("projection\x00"))
	modelUniform := gl.GetUniformLocation(cd.shader, gl.Str("model\x00"))

	// 'vars' in fragment shader
	// fontUniform := gl.GetUniformLocation(shader, gl.Str("font\x00"))
	textColorUniform := gl.GetUniformLocation(cd.shader, gl.Str("textColor\x00"))

	// WHY?
	// gl.BindFragDataLocation(cd.shader, 0, gl.Str("color\x00")) // have to set this so frag shader knows where to put its output

	proj := mgl32.Ortho2D(0, width, height, 0) // inverts Y axis so (0,0) is at screen top left
	gl.UniformMatrix4fv(projectionUniform, 1, false, &proj[0])

	gl.Uniform3fv(textColorUniform, 1, &color[0])

	var model mgl32.Mat4
	for i, r := range text {
		model = mgl32.Translate3D(x+scale*(float32(i)*cd.fw), y*scale, 0).Mul4(mgl32.Scale3D(scale, scale, scale))
		c, ok := cd.dict[r]
		if !ok {
			continue
		}

		gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])
		c.draw()
	}

	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.UseProgram(0)
}

type Character struct {
	vao, vbo uint32
}

func (c Character) delete() {
	gl.DeleteBuffers(1, &c.vbo)
	gl.DeleteVertexArrays(1, &c.vao)
}

func (c Character) draw() {
	gl.BindVertexArray(c.vao)              // bind vao once
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4) // draw 4 verticies from VAO
	gl.BindVertexArray(0)
}

func makeCharacters(shader uint32, face *basicfont.Face) map[rune]Character {
	w, h := float32(face.Width), float32(face.Height+1)
	numChars := face.Mask.Bounds().Max.Y / int(h)
	chars := make(map[rune]Character, numChars)
	dtexY := 1.0 / float32(numChars)

	vertAttrib := uint32(gl.GetAttribLocation(shader, gl.Str("vertex\x00")))

	var offset float32
	for _, set := range face.Ranges {
		for r := set.Low; r < set.High; r++ {

			verts := [4 * 4]float32{
				// pos(x,y), tex(u,v)
				// 1, top left
				0, h, 0, (1 + offset) * dtexY,
				// 2, bottom left
				0, 0, 0, offset * dtexY,
				// 3, top right
				w, h, 1, (1 + offset) * dtexY,
				// 4, bottom right
				w, 0, 1, offset * dtexY,
			}

			var c Character
			gl.GenVertexArrays(1, &c.vao)         // make vao
			gl.BindVertexArray(c.vao)             // set vao "current"
			gl.GenBuffers(1, &c.vbo)              // make vbo in current vao
			gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo) // set vbo "current" (in ARRAY_BUFFER slot)

			// load data into current vbo
			gl.BufferData(gl.ARRAY_BUFFER, len(verts)*SizeOfFloat, gl.Ptr(&verts[0]), gl.STATIC_DRAW)
			chars[r] = c

			// associate a vertex attribute with the vbo
			// describe data layout in current vbo
			// (size: 4 float in 1 of this attribute, stride: 4 float * 4 bytes/float in 1 vertex)
			gl.VertexAttribPointer(vertAttrib, 4, gl.FLOAT, false, 4*SizeOfFloat, gl.PtrOffset(0))
			gl.EnableVertexAttribArray(vertAttrib)

			gl.BindVertexArray(0)             // set current vao to "none"
			gl.BindBuffer(gl.ARRAY_BUFFER, 0) // set current vbo to "none"

			offset++
		}
	}

	return chars
}

var fontVertexShader = `
#version 330 core

layout (location = 0) in vec4 vertex; // <vec2 pos, vec2 tex>

uniform mat4 projection;
uniform mat4 model;

out vec2 TexCoords;

void main()
{
    gl_Position = projection * model * vec4(vertex.xy, 0.0, 1.0);
    TexCoords = vertex.zw;
}
` + "\x00"

var fontFragmentShader = `
#version 330 core

in vec2 TexCoords;

uniform sampler2D font;
uniform vec3 textColor;

out vec4 color;

void main()
{    
	float alpha = texture(font, TexCoords).a;
	color = vec4(textColor.xyz, alpha);
}
` + "\x00"
