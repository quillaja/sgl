package sgl

import (
	"fmt"
	"image"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// only need this once in the package
var skyboxProgram *Program

// Skybox is a complete cubemap skybox.
type Skybox struct {
	TextureID uint32
	Vao       *Vao
}

// NewSkybox creates a skybox. It expects faces in this order:
//     +X (right)
//     -X (left)
//     +Y (top)
//     -Y (bottom)
//     +Z (front)
//     -Z (back)
func NewSkybox(faces []*image.RGBA) (*Skybox, error) {
	if skyboxProgram == nil {
		skyboxProgram = NewProgram()
		skyboxProgram.AddShader(VertexShader, skyboxVertexShader,
			[]string{"projection", "view"},
			Attribute{Name: "aPos", Type: gl.FLOAT, Size: 3, Stride: 3 * SizeOfFloat, Offset: 0})
		skyboxProgram.AddShader(FragmentShader, skyboxFragmentShader, []string{"skybox"})
		errBuild := skyboxProgram.Build()
		if errBuild != nil {
			return nil, fmt.Errorf("couldn't build skybox program: %w", errBuild)
		}
	}

	vao := NewVao(skyboxProgram)
	vao.SetVbo(SizeOfFloat*len(skyboxVertices), skyboxVertices, gl.STATIC_DRAW)

	sky := &Skybox{
		TextureID: loadCubemap(faces),
		Vao:       vao,
	}

	return sky, nil
}

// Delete resources.
func (sky *Skybox) Delete() {
	sky.Vao.Delete()
	gl.DeleteTextures(1, &sky.TextureID)
}

// Draw should be called after other objects.
func (sky *Skybox) Draw(view, projection mgl32.Mat4) {
	view = view.Mat3().Mat4() // remove translation from the view matrix
	skyboxProgram.Use()
	skyboxProgram.Vertex().SetMat4("view", 1, &view)
	skyboxProgram.Vertex().SetMat4("projection", 1, &projection)
	// skybox cube
	gl.DepthFunc(gl.LEQUAL) // change depth function so depth test passes when values are equal to depth buffer's content
	gl.BindVertexArray(sky.Vao.VaoID)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, sky.TextureID)
	gl.DrawArrays(gl.TRIANGLES, 0, 36) // actually draws skybox
	gl.BindVertexArray(0)
	gl.DepthFunc(gl.LESS) // set depth function back to default
}

// loads a cubemap texture from 6 individual texture faces
// order:
// +X (right)
// -X (left)
// +Y (top)
// -Y (bottom)
// +Z (front)
// -Z (back)
func loadCubemap(faces []*image.RGBA) uint32 {
	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, textureID)

	for i, face := range faces {
		gl.TexImage2D(
			uint32(gl.TEXTURE_CUBE_MAP_POSITIVE_X+i),
			0,
			gl.RGB, // internal format (don't need alpha)
			int32(face.Bounds().Dx()), int32(face.Bounds().Dy()),
			0,
			gl.RGBA, // image format
			gl.UNSIGNED_BYTE,
			gl.Ptr(face.Pix))
	}
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)

	return textureID
}

const skyboxVertexShader = `#version 330 core
in vec3 aPos;

uniform mat4 projection;
uniform mat4 view;

out vec3 TexCoords;

void main()
{
    TexCoords = aPos;
    vec4 pos = projection * view * vec4(aPos, 1.0);
    gl_Position = pos.xyww;
}`

const skyboxFragmentShader = `#version 330 core
uniform samplerCube skybox;

in vec3 TexCoords;

out vec4 FragColor;

void main()
{    
    FragColor = texture(skybox, TexCoords);
}`

// positions of vertices each face (2 triangles per face).
// centered around origin, whd of 2.
var skyboxVertices = []float32{
	-1.0, 1.0, -1.0,
	-1.0, -1.0, -1.0,
	1.0, -1.0, -1.0,
	1.0, -1.0, -1.0,
	1.0, 1.0, -1.0,
	-1.0, 1.0, -1.0,

	-1.0, -1.0, 1.0,
	-1.0, -1.0, -1.0,
	-1.0, 1.0, -1.0,
	-1.0, 1.0, -1.0,
	-1.0, 1.0, 1.0,
	-1.0, -1.0, 1.0,

	1.0, -1.0, -1.0,
	1.0, -1.0, 1.0,
	1.0, 1.0, 1.0,
	1.0, 1.0, 1.0,
	1.0, 1.0, -1.0,
	1.0, -1.0, -1.0,

	-1.0, -1.0, 1.0,
	-1.0, 1.0, 1.0,
	1.0, 1.0, 1.0,
	1.0, 1.0, 1.0,
	1.0, -1.0, 1.0,
	-1.0, -1.0, 1.0,

	-1.0, 1.0, -1.0,
	1.0, 1.0, -1.0,
	1.0, 1.0, 1.0,
	1.0, 1.0, 1.0,
	-1.0, 1.0, 1.0,
	-1.0, 1.0, -1.0,

	-1.0, -1.0, -1.0,
	-1.0, -1.0, 1.0,
	1.0, -1.0, -1.0,
	1.0, -1.0, -1.0,
	-1.0, -1.0, 1.0,
	1.0, -1.0, 1.0,
}
