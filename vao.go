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

// Doesn't look like this is going to work with generics like i had hoped,
// but the data structure might still be useful.
type Buffer struct {
	ID           uint32
	Attributes   []string // names of associated vertex attributes, nil for element buffers
	target       uint32   // ARRAY_BUFFER or ELEMENT_ARRAY_BUFFER
	usage        uint32   // StaticDraw or DynamicDraw
	bytesPerItem int      // bytes in each "vertex" (for VBO) or index (for EBO)
	count        int      // total items (ie vertices or indices)
	size         int      // total size in bytes
}

func (b *Buffer) Count() int      { return b.count }
func (b *Buffer) Size() int       { return b.size }
func (b *Buffer) Bytes(n int) int { return n * b.bytesPerItem }

func (b *Buffer) Bind() {
	gl.BindBuffer(b.target, b.ID)
}

func (b *Buffer) UnBind() {
	gl.BindBuffer(b.target, 0)
}

// used with VBOs (not EBOs)
// be sure to call Bind() before and UnBind() after
func (b *Buffer) enableAttribs(attribs map[string]*Attribute) {
	for _, name := range b.Attributes {
		if attrib, found := attribs[name]; found {
			fmt.Println("found", name)
			attrib.Enable()
			if err := CheckError(); err != nil {
				fmt.Println("Buffer.EnableAttribs()", err)
			}
			b.bytesPerItem += int(attrib.Size) * BytesIn(attrib.Type)
		}
	}
}

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

func (b *Buffer) Set(data interface{}) {
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

func (b *Buffer) SetSub(startVertex, countVertices int, data interface{}) {
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

func NewVbo(attributeNames ...string) *Buffer {
	b := &Buffer{
		Attributes: attributeNames,
		target:     gl.ARRAY_BUFFER,
	}
	gl.GenBuffers(1, &b.ID)
	return b
}

func NewEbo() *Buffer {
	b := &Buffer{
		target: gl.ELEMENT_ARRAY_BUFFER,
	}
	gl.GenBuffers(1, &b.ID)
	return b
}

// current limitations of Vao:
// 1) can accept only float32 type for vbo
// 2) can only do interlaced verts in the single vbo

type Vao struct {
	ID            uint32       // id for vao
	Vbo           uint32       // id for vertex buffer object associated with this vao
	Ebo           uint32       // id for element (vertex index) buffer associated with this vao
	DrawMode      uint32       // "mode" for drawing, such as TRIANGLES or LINES
	Tex           []*Texture2D // ids for all textures to be used with this vao (on draw)
	Prog          *Program     // program to load when drawing
	floatsPerVert int32        // as determined from the program attributes
	vboVertCount  int32        // number of "vertices", which could be any combination of positions, normals, uvs, etc.
	vboTotalSize  int          // in bytes
	eboElemCount  int32        // number of indicies, which is usually the number of triangles * 3
	eboTotalSize  int          // in bytes
	VBOS          map[string]*Buffer
}

func NewVao(drawMode uint32, program *Program, vboToAttrib map[string][]string) *Vao {
	v := &Vao{
		DrawMode: drawMode,
		Prog:     program,
		Tex:      make([]*Texture2D, 0),
		VBOS:     make(map[string]*Buffer),
	}

	gl.GenVertexArrays(1, &v.ID)
	gl.BindVertexArray(v.ID)

	// if i wanted to make the vao use separate vbos for each vertex attribute,
	// (eg VVVNNN instead of interlaced VNVNVN), i would have do for each vbo
	// (1) bind the vbo, then (2) enable the specific attribute (3) unbind the vbo.
	// this would require a way for the user to specify associations between
	// vbos and attribs. Currently a single interlaced vbo is all that's possible.
	if len(vboToAttrib) == 0 {
		v.makeBuffer(gl.ARRAY_BUFFER, &v.Vbo)
		v.enableAttribs()
	} else {
		for name, attribNames := range vboToAttrib {
			vbo := NewVbo(attribNames...)
			vbo.Bind()
			vbo.enableAttribs(program.Vertex().Attribs)
			vbo.UnBind()
			v.VBOS[name] = vbo
		}
	}

	// v.makeBuffer(gl.ARRAY_BUFFER, &v.Vbo)
	v.makeBuffer(gl.ELEMENT_ARRAY_BUFFER, &v.Ebo)

	// v.enableAttribs()

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
	gl.DeleteVertexArrays(1, &v.ID)
	gl.DeleteBuffers(1, &v.Vbo)
	gl.DeleteBuffers(1, &v.Ebo)
}

// determined from vertex shader program attributes
func (v *Vao) FloatsPerVertex() int { return int(v.floatsPerVert) }

// Size in bytes of n vertices (according to size defined by shader attribs)
// can be used to calculate the size of n vertices.
// can be used to calculate the offset of nth vertex.
func (v *Vao) Bytes(n int) int {
	return n * SizeOfFloat * int(v.floatsPerVert)
}

func (v *Vao) SizeOfVbo() (bytes int) { return v.vboTotalSize }
func (v *Vao) SizeOfEbo() (bytes int) { return v.eboTotalSize }

func (v *Vao) CountVertices() int { return int(v.vboVertCount) }
func (v *Vao) CountElements() int { return int(v.eboElemCount) }

func (v *Vao) count() int32 {
	if v.eboElemCount > 0 {
		return v.eboElemCount
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
	v.vboTotalSize = sizeInBytes
	if data == nil { // go-gl doesn't like nil buffers
		data = make([]float32, sizeInBytes/SizeOfFloat)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, v.Vbo)
	gl.BufferData(gl.ARRAY_BUFFER, sizeInBytes, gl.Ptr(data), usage)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0) // unbind
}

func (v *Vao) SetEboOptions(sizeInBytes int, data []uint32, usage uint32) {
	v.eboElemCount = int32(len(data))
	v.eboTotalSize = sizeInBytes
	if data == nil { // go-gl doesn't like nil buffers
		data = make([]uint32, sizeInBytes/SizeOfInt)
	}
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, v.Ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, sizeInBytes, gl.Ptr(data), usage)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0) // unbind
}

func (v *Vao) SetVboSubBuffer(startInBytes, sizeInBytes int, data []float32) {
	v.vboVertCount += int32(len(data)) / v.floatsPerVert
	v.vboTotalSize += sizeInBytes
	gl.BindBuffer(gl.ARRAY_BUFFER, v.Vbo)
	gl.BufferSubData(gl.ARRAY_BUFFER, startInBytes, sizeInBytes, gl.Ptr(data))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func (v *Vao) SetEboSubBuffer(startInBytes, sizeInBytes int, data []uint32) {
	v.eboElemCount += int32(len(data))
	v.eboTotalSize += sizeInBytes
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, v.Ebo)
	gl.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, startInBytes, sizeInBytes, gl.Ptr(data))
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
}

// sizeInBytes = NumVerticesDesired * FloatsPerVertex * SizeOfFloat
// MUST: len(data) >= sizeInBytes/SizeOfFloat
func (v *Vao) GetVboBuffer(startInBytes, sizeInBytes int, data []float32) {
	// data MUST have a len large enough for the buffer
	gl.BindBuffer(gl.ARRAY_BUFFER, v.Vbo)
	gl.GetBufferSubData(gl.ARRAY_BUFFER, startInBytes, sizeInBytes, gl.Ptr(data))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

// sizeInBytes = NumIndicesDesired * SizeOfInt
// MUST: len(data) >= sizeInBytes/SizeOfInt
func (v *Vao) GetEboBuffer(startInBytes, sizeInBytes int, data []uint32) {
	// data MUST have a len large enough for the buffer
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, v.Ebo)
	gl.GetBufferSubData(gl.ARRAY_BUFFER, startInBytes, sizeInBytes, gl.Ptr(data))
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
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
	v.DrawOptions(v.DrawMode, 0, v.count())
}

// DrawOptions call Vao.Prog.Use() before drawing.
func (v *Vao) DrawOptions(mode uint32, first, count int32) {

	// load texture uniforms in fragment shader
	for i, tex := range v.Tex {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(i))
		gl.BindTexture(gl.TEXTURE_2D, tex.ID)
	}
	gl.ActiveTexture(gl.TEXTURE0) // reset to 0th texture

	gl.BindVertexArray(v.ID)
	if v.eboElemCount > 0 {
		gl.DrawElements(mode, count, gl.UNSIGNED_INT, gl.PtrOffset(int(first)))
	} else {
		gl.DrawArrays(mode, first, count)
	}

	gl.BindVertexArray(0) // unbind vao
}
