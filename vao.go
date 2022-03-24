package sgl

import (
	"fmt"
	"reflect"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// Easier access to gl "draw mode" types.
const (
	Points        = gl.POINTS
	Lines         = gl.LINES
	LineStrip     = gl.LINE_STRIP
	Triangles     = gl.TRIANGLES
	TriangleStrip = gl.TRIANGLE_STRIP
	TriangleFan   = gl.TRIANGLE_FAN
)

const (
	StaticDraw  = gl.STATIC_DRAW
	DynamicDraw = gl.DYNAMIC_DRAW
)

type Buffer struct {
	ID           uint32
	Name         string
	Attributes   []Attribute // associated vertex attributes, nil for element buffers
	target       uint32      // ARRAY_BUFFER or ELEMENT_ARRAY_BUFFER
	usage        uint32      // StaticDraw or DynamicDraw
	bytesPerItem int         // bytes in each "vertex" (for VBO) or index (for EBO)
	count        int         // total items (ie vertices or indices)
	size         int         // total size in bytes
}

func (b *Buffer) Count() int      { return b.count }            // number of vertices
func (b *Buffer) Size() int       { return b.size }             // size of buffer capacity in bytes
func (b *Buffer) Bytes(n int) int { return n * b.bytesPerItem } // calculates the number of bytes in n vertices

func (b *Buffer) Bind() {
	gl.BindBuffer(b.target, b.ID)
}

func (b *Buffer) UnBind() {
	gl.BindBuffer(b.target, 0)
}

// used with VBOs (not EBOs)
func (b *Buffer) enableAttribs() {
	b.Bind()
	for _, attrib := range b.Attributes {
		attrib.Enable()
		if err := CheckError(); err != nil {
			fmt.Println("Buffer.EnableAttribs()", err)
		}
		b.bytesPerItem += int(attrib.Size) * BytesIn(attrib.Type)
	}
	b.UnBind()
}

// reserve memory for the buffer (BufferData(nil))
func (b *Buffer) Allocate(vertexCount int, usage uint32) {
	if b.target == gl.ELEMENT_ARRAY_BUFFER {
		// ASSUME that an EBO will use only uint32
		// bytesPerItem for VBO is calculated in EnableAttribs()
		b.bytesPerItem = SizeOfInt
	}

	b.size = b.Bytes(vertexCount)
	b.usage = usage
	b.Bind()
	gl.BufferData(b.target, b.size, gl.Ptr(nil), b.usage)
	b.UnBind()
}

// allocates and fills buffer with data (BufferData(data))
func (b *Buffer) Initalize(data interface{}) {
	// if t.Kind() != reflect.Slice {
	// return // silent failure
	// }

	t := reflect.TypeOf(data)
	switch b.target {
	case gl.ARRAY_BUFFER:
		dataSize := int(t.Elem().Size())                // bytes in single element of data
		b.size = reflect.ValueOf(data).Len() * dataSize // bytes in entire slice
		b.count = b.size / b.bytesPerItem               // bytesPerItem calculated in EnableAttribs()
	case gl.ELEMENT_ARRAY_BUFFER:
		b.bytesPerItem = SizeOfInt // ASSUME that only uint32 will be used for indices
		b.count = reflect.ValueOf(data).Len()
		b.size = b.count * b.bytesPerItem
	}

	if b.usage == 0 {
		b.usage = StaticDraw // set to static draw if not yet set (by Allocate())
	}
	b.Bind()
	gl.BufferData(b.target, b.size, gl.Ptr(data), b.usage)
	if err := CheckError(); err != nil {
		fmt.Println("Buffer.Set()", err)
	}
	b.UnBind()
}

// set some slice of the buffer to data (BufferSubData(data))
func (b *Buffer) Set(startVertex, countVertices int, data interface{}) {
	// size already set in Allocate()
	// bytesPerVertex already determined elsewhere
	b.count = countVertices
	b.Bind()
	gl.BufferSubData(b.target, b.Bytes(startVertex), b.Bytes(countVertices), gl.Ptr(data))
	b.UnBind()
}

// data MUST be a slice with length >= b.Bytes(countVertices)
func (b *Buffer) Get(startVertex, countVertices int, data interface{}) {
	b.Bind()
	gl.GetBufferSubData(b.target, b.Bytes(startVertex), b.Bytes(countVertices), gl.Ptr(data))
	b.UnBind()
}

func (b *Buffer) Delete() {
	gl.DeleteBuffers(1, &b.ID)
}

func NewVbo(name string, attribs ...Attribute) *Buffer {
	b := &Buffer{
		Name:       name,
		Attributes: attribs,
		target:     gl.ARRAY_BUFFER,
	}
	gl.GenBuffers(1, &b.ID)
	return b
}

func NewEbo() *Buffer {
	b := &Buffer{
		Name:   "EBO",
		target: gl.ELEMENT_ARRAY_BUFFER,
	}
	gl.GenBuffers(1, &b.ID)
	return b
}

// current limitations of Vao:
// 1) can accept only float32 type for vbo
// 2) can only do interlaced verts in the single vbo

type Vao struct {
	ID       uint32             // id for vao
	Vbo      map[string]*Buffer // VBOs associated with this vao, by name
	Ebo      *Buffer            // the element (vertex index) buffer associated with this vao
	DrawMode uint32             // "mode" for drawing, such as TRIANGLES or LINES
	// Tex      []*Texture2D       // ids for all textures to be used with this vao (on draw) TODO: i think textures shouldn't be part of the Vao
	// Prog     *Program           // program to load when drawing
}

// panics if no VBOs are provided.
func NewVao(drawMode uint32, vbos ...*Buffer) *Vao {
	v := &Vao{
		DrawMode: drawMode,
		// Prog:     program,
		// Tex:      make([]*Texture2D, 0),
		Vbo: make(map[string]*Buffer),
	}

	gl.GenVertexArrays(1, &v.ID)
	gl.BindVertexArray(v.ID)

	// if i wanted to make the vao use separate vbos for each vertex attribute,
	// (eg VVVNNN instead of interlaced VNVNVN), i would have do for each vbo
	// (1) bind the vbo, then (2) enable the specific attribute (3) unbind the vbo.
	// this would require a way for the user to specify associations between
	// vbos and attribs. Currently a single interlaced vbo is all that's possible.
	if len(vbos) == 0 {
		panic("no vbo (*Buffer) provided")
	}

	for _, vbo := range vbos {
		v.Vbo[vbo.Name] = vbo
		vbo.enableAttribs() // binds and unbinds
	}

	v.Ebo = NewEbo() // just gets id, doesn't set up ebo or allocate
	v.Ebo.Bind()     // necessary?

	gl.BindVertexArray(0)
	v.Ebo.UnBind()

	return v
}

// func (v *Vao) makeBuffer(kind uint32, id *uint32) {
// 	gl.GenBuffers(1, id)
// 	gl.BindBuffer(kind, *id)
// }

// func (v *Vao) enableAttribs() {
// 	var floatsPerVertex int32
// 	for _, attrib := range v.Prog.Vertex().Attribs {
// 		attrib.Enable() // associate this attribute to the vbo
// 		floatsPerVertex += attrib.Size
// 	}
// 	v.floatsPerVert = floatsPerVertex
// }

func (v *Vao) Delete() {
	gl.DeleteVertexArrays(1, &v.ID)
	for _, vbo := range v.Vbo {
		vbo.Delete()
	}
	v.Ebo.Delete()
}

// determined from vertex shader program attributes
// func (v *Vao) FloatsPerVertex() int { return int(v.floatsPerVert) }

// Size in bytes of n vertices (according to size defined by shader attribs)
// can be used to calculate the size of n vertices.
// can be used to calculate the offset of nth vertex.
// func (v *Vao) Bytes(n int) int {
// 	return n * SizeOfFloat * int(v.floatsPerVert)
// }

// func (v *Vao) SizeOfVbo() (bytes int) { return v.vboTotalSize }
// func (v *Vao) SizeOfEbo() (bytes int) { return v.eboTotalSize }

// func (v *Vao) CountVertices() int { return int(v.vboVertCount) }
// func (v *Vao) CountElements() int { return int(v.eboElemCount) }

// func (v *Vao) count() int32 {
// 	if v.eboElemCount > 0 {
// 		return v.eboElemCount
// 	}
// 	return v.vboVertCount
// }

func (v *Vao) count() int32 {
	// TODO: this is a hack. i'm just guessing that the calculations for Count() on all
	// VBO *Buffers will match (i mean, they sorta have to...), and so here I just choose
	// the first one and use it for count.
	var count int32
	if v.Ebo.Count() > 0 {
		count = int32(v.Ebo.Count())
	} else {
		for _, vbo := range v.Vbo {
			count = int32(vbo.Count())
			break
		}
	}
	return count
}

// func (v *Vao) SetVbo(data []float32) {
// 	v.SetVboOptions(SizeOfFloat*len(data), data, gl.STATIC_DRAW)
// }

// func (v *Vao) SetEbo(data []uint32) {
// 	v.SetEboOptions(SizeOfInt*len(data), data, gl.STATIC_DRAW)
// }

// func (v *Vao) SetVboOptions(sizeInBytes int, data []float32, usage uint32) {
// 	v.vboVertCount = int32(len(data)) / v.floatsPerVert
// 	v.vboTotalSize = sizeInBytes
// 	if data == nil { // go-gl doesn't like nil buffers
// 		data = make([]float32, sizeInBytes/SizeOfFloat)
// 	}
// 	gl.BindBuffer(gl.ARRAY_BUFFER, v.Vbo)
// 	gl.BufferData(gl.ARRAY_BUFFER, sizeInBytes, gl.Ptr(data), usage)
// 	gl.BindBuffer(gl.ARRAY_BUFFER, 0) // unbind
// }

// func (v *Vao) SetEboOptions(sizeInBytes int, data []uint32, usage uint32) {
// 	v.eboElemCount = int32(len(data))
// 	v.eboTotalSize = sizeInBytes
// 	if data == nil { // go-gl doesn't like nil buffers
// 		data = make([]uint32, sizeInBytes/SizeOfInt)
// 	}
// 	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, v.Ebo)
// 	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, sizeInBytes, gl.Ptr(data), usage)
// 	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0) // unbind
// }

// func (v *Vao) SetVboSubBuffer(startInBytes, sizeInBytes int, data []float32) {
// 	v.vboVertCount += int32(len(data)) / v.floatsPerVert
// 	v.vboTotalSize += sizeInBytes
// 	gl.BindBuffer(gl.ARRAY_BUFFER, v.Vbo)
// 	gl.BufferSubData(gl.ARRAY_BUFFER, startInBytes, sizeInBytes, gl.Ptr(data))
// 	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
// }

// func (v *Vao) SetEboSubBuffer(startInBytes, sizeInBytes int, data []uint32) {
// 	v.eboElemCount += int32(len(data))
// 	v.eboTotalSize += sizeInBytes
// 	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, v.Ebo)
// 	gl.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, startInBytes, sizeInBytes, gl.Ptr(data))
// 	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
// }

// // sizeInBytes = NumVerticesDesired * FloatsPerVertex * SizeOfFloat
// // MUST: len(data) >= sizeInBytes/SizeOfFloat
// func (v *Vao) GetVboBuffer(startInBytes, sizeInBytes int, data []float32) {
// 	// data MUST have a len large enough for the buffer
// 	gl.BindBuffer(gl.ARRAY_BUFFER, v.Vbo)
// 	gl.GetBufferSubData(gl.ARRAY_BUFFER, startInBytes, sizeInBytes, gl.Ptr(data))
// 	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
// }

// // sizeInBytes = NumIndicesDesired * SizeOfInt
// // MUST: len(data) >= sizeInBytes/SizeOfInt
// func (v *Vao) GetEboBuffer(startInBytes, sizeInBytes int, data []uint32) {
// 	// data MUST have a len large enough for the buffer
// 	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, v.Ebo)
// 	gl.GetBufferSubData(gl.ARRAY_BUFFER, startInBytes, sizeInBytes, gl.Ptr(data))
// 	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
// }

// SetTexture associates named uniform (in frag shader) with this
// texture, and also associates the texture with the texture
// "number" of the texture's index in the Tex slice.
// eg. TEXTURE0 is at Tex[0], TEXTURE1 at Tex[1], etc.
// func (v *Vao) SetTexture(uniformName string, texture *Texture2D) {
// 	v.Tex = append(v.Tex, texture)
// 	texNumber := int32(len(v.Tex) - 1)
// 	v.Prog.Fragment().SetInt(uniformName, 1, &texNumber) // TODO: hmm, should this be here or in Draw()?
// }

// Draw call Vao.Prog.Use() first!
func (v *Vao) Draw() {

	v.DrawOptions(v.DrawMode, 0, v.count())
}

// DrawOptions call Vao.Prog.Use() before drawing.
func (v *Vao) DrawOptions(mode uint32, first, count int32) {

	// load texture uniforms in fragment shader
	// for i, tex := range v.Tex {
	// 	gl.ActiveTexture(gl.TEXTURE0 + uint32(i))
	// 	gl.BindTexture(gl.TEXTURE_2D, tex.ID)
	// }
	// gl.ActiveTexture(gl.TEXTURE0) // reset to 0th texture

	gl.BindVertexArray(v.ID)
	if v.Ebo.Count() > 0 {
		gl.DrawElements(mode, count, Uint32, gl.PtrOffset(int(first)))
	} else {
		gl.DrawArrays(mode, first, count)
	}

	gl.BindVertexArray(0) // unbind vao
}
