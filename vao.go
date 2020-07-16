package sgl

import (
	"github.com/go-gl/gl/v3.3-core/gl"
)

// Easier access to gl "draw mode" types.
const (
	Triangles     = gl.TRIANGLES
	Points        = gl.POINTS
	Lines         = gl.LINES
	TriangleStrip = gl.TRIANGLE_STRIP
	TriangleFan   = gl.TRIANGLE_FAN
)

type Vao struct {
	VaoID         uint32       // id for vao
	Vbo           uint32       // id for vertex buffer object associated with this vao
	Ebo           uint32       // id for element (vertex index) buffer associated with this vao
	DrawMode      uint32       // "mode" for drawing, such as TRIANGLES or LINES
	Tex           []*Texture2D // ids for all textures to be used with this vao (on draw)
	Prog          *Program     // program to load when drawing
	floatsPerVert int32
	vboVertCount  int32
	eboVertCount  int32
}

func NewVao(drawMode uint32, program *Program) *Vao {
	v := &Vao{
		DrawMode: drawMode,
		Prog:     program,
		Tex:      make([]*Texture2D, 0),
	}

	gl.GenVertexArrays(1, &v.VaoID)
	gl.BindVertexArray(v.VaoID)

	v.makeBuffer(gl.ARRAY_BUFFER, &v.Vbo)
	v.makeBuffer(gl.ELEMENT_ARRAY_BUFFER, &v.Ebo)

	v.enableAttribs()

	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)

	return v
}

func (v *Vao) makeBuffer(kind uint32, id *uint32) {
	gl.GenBuffers(1, id)
	gl.BindBuffer(kind, *id)
}

func (v *Vao) enableAttribs() {
	var floatsPerVertex int32
	for _, attrib := range v.Prog.Vertex().Attribs {
		attrib.Enable() // associate this attribute to the vbo
		floatsPerVertex += attrib.Size
	}
	v.floatsPerVert = floatsPerVertex
}

func (v *Vao) Delete() {
	gl.DeleteVertexArrays(1, &v.VaoID)
	gl.DeleteBuffers(1, &v.Vbo)
}

func (v *Vao) CountVerts() int32 {
	if v.eboVertCount > 0 {
		return v.eboVertCount
	}
	return v.vboVertCount
}

func (v *Vao) SetVbo(data []float32) {
	v.SetVboOptions(SizeOfFloat*len(data), data, gl.STATIC_DRAW)
}

func (v *Vao) SetEbo(data []uint32) {
	v.SetEboOptions(SizeOfInt*len(data), data, gl.STATIC_DRAW)
}

func (v *Vao) SetVboOptions(sizeInBytes int, data []float32, usage uint32) {
	v.vboVertCount = int32(len(data)) / v.floatsPerVert
	gl.BindBuffer(gl.ARRAY_BUFFER, v.Vbo)
	gl.BufferData(gl.ARRAY_BUFFER, sizeInBytes, gl.Ptr(data), usage)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0) // unbind
}

func (v *Vao) SetEboOptions(sizeInBytes int, data []uint32, usage uint32) {
	v.eboVertCount = int32(len(data))
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, v.Ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, sizeInBytes, gl.Ptr(data), usage)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0) // unbind
}

// SetTexture associates named uniform (in frag shader) with this
// texture, and also associates the texture with the texture
// "number" of the texture's index in the Tex slice.
// eg. TEXTURE0 is at Tex[0], TEXTURE1 at Tex[1], etc.
func (v *Vao) SetTexture(uniformName string, texture *Texture2D) {
	v.Tex = append(v.Tex, texture)
	texNumber := int32(len(v.Tex) - 1)
	v.Prog.Fragment().SetInt(uniformName, 1, &texNumber)
}

// Draw call Vao.Prog.Use() first!
func (v *Vao) Draw() {
	v.DrawOptions(v.DrawMode, 0, v.CountVerts())
}

// DrawOptions call Vao.Prog.Use() before drawing.
func (v *Vao) DrawOptions(mode uint32, first, count int32) {

	// load texture uniforms in fragment shader
	for i, tex := range v.Tex {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(i))
		gl.BindTexture(gl.TEXTURE_2D, tex.ID)
	}
	gl.ActiveTexture(gl.TEXTURE0) // reset to 0th texture

	gl.BindVertexArray(v.VaoID)
	if v.eboVertCount > 0 {
		gl.DrawElements(mode, count, gl.UNSIGNED_INT, gl.PtrOffset(int(first)))
	} else {
		gl.DrawArrays(mode, first, count)
	}

	gl.BindVertexArray(0) // unbind vao
}
