package sgl

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Size of common types in bytes.
const (
	SizeOfByte  = 1
	SizeOfFloat = 4 * SizeOfByte
	SizeOfInt   = 4 * SizeOfByte
	SizeOfV3    = 3 * SizeOfFloat
	SizeOfV4    = 4 * SizeOfFloat
	SizeOfM4    = 4 * SizeOfV4
)

// Common vertex attribute types.
const (
	Float32 = gl.FLOAT
	Int32   = gl.INT
	Uint32  = gl.UNSIGNED_INT
	Int8    = gl.BYTE
	Uint8   = gl.UNSIGNED_BYTE
)

func BytesIn(t uint32) int {
	switch t {
	case Float32:
		return SizeOfFloat
	case Int32:
		return SizeOfInt
	case Uint32:
		return SizeOfInt
	case Int8:
		return SizeOfByte
	case Uint8:
		return SizeOfByte
	default:
		return 0
	}
}

/*
uniform       valid
type          functions
----------------------------------------------
float         gl.uniform1f    gl.uniform1fv
vec2          gl.uniform2f    gl.uniform2fv
vec3          gl.uniform3f    gl.uniform3fv
vec4          gl.uniform4f    gl.uniform4fv
int           gl.uniform1i    gl.uniform1iv
ivec2         gl.uniform2i    gl.uniform2iv
ivec3         gl.uniform3i    gl.uniform3iv
ivec4         gl.uniform4i    gl.uniform4iv
sampler2D     gl.uniform1i    gl.uniform1iv
samplerCube   gl.uniform1i    gl.uniform1iv

mat2          gl.uniformMatrix2fv
mat3          gl.uniformMatrix3fv
mat4          gl.uniformMatrix4fv

bool          gl.uniform1i gl.uniform1f gl.uniform1iv gl.uniform1fv
bvec2         gl.uniform2i gl.uniform2f gl.uniform2iv gl.uniform2fv
bvec3         gl.uniform3i gl.uniform3f gl.uniform3iv gl.uniform3fv
bvec4         gl.uniform4i gl.uniform4f gl.uniform4iv gl.uniform4fv
*/

type Attribute struct {
	ID     uint32 // index or location of attribute
	Name   string // name of attribute in the shader GLSL
	Size   int32  // 'numbers' in a single attribute
	Type   uint32 // sgl.Float32 (gl.FLOAT), etc
	Stride int32  // bytes
	Offset int    // bytes
	// Normalized bool // if added, goes after Type
}

// Enable (associate) attribute with "current" VAO/VBO.
func (a *Attribute) Enable() {
	gl.EnableVertexAttribArray(a.ID)
	gl.VertexAttribPointer(a.ID, a.Size, a.Type, false, a.Stride, gl.PtrOffset(a.Offset))
}

// func (a *Attribute) String() string { return fmt.Sprintf("%+v", *a) }

// Aliases for common shader types to avoid slow autocomplete of gl pkg.
const (
	VertexShader   = gl.VERTEX_SHADER
	FragmentShader = gl.FRAGMENT_SHADER
	GeometryShader = gl.GEOMETRY_SHADER
	ComputeShader  = gl.COMPUTE_SHADER
)

type Shader struct {
	ID       uint32
	Type     uint32 // gl.VERTEX_SHADER, gl.FRAGMENT_SHADER, etc
	Source   string
	Attribs  map[string]*Attribute
	Uniforms map[string]int32
}

// Attributes gets a slice of copies of the shader's attributes.
func (s *Shader) Attributes() (copy []Attribute) {
	for _, a := range s.Attribs {
		copy = append(copy, *a)
	}
	return
}

func (s *Shader) SetInt(uniformName string, count int32, val *int32) {
	gl.Uniform1iv(s.Uniforms[uniformName], count, val)
}

func (s *Shader) SetFloat(uniformName string, count int32, val *float32) {
	gl.Uniform1fv(s.Uniforms[uniformName], count, val)
}

func (s *Shader) SetVec2(uniformName string, count int32, val *mgl32.Vec2) {
	gl.Uniform2fv(s.Uniforms[uniformName], count, &(*val)[0])
}

func (s *Shader) SetVec3(uniformName string, count int32, val *mgl32.Vec3) {
	gl.Uniform3fv(s.Uniforms[uniformName], count, &(*val)[0])
}

func (s *Shader) SetVec4(uniformName string, count int32, val *mgl32.Vec4) {
	gl.Uniform4fv(s.Uniforms[uniformName], count, &(*val)[0])
}

func (s *Shader) SetMat4(uniformName string, count int32, val *mgl32.Mat4) {
	gl.UniformMatrix4fv(s.Uniforms[uniformName], count, false, &(*val)[0])
}

// func (s *Shader) String() string { return fmt.Sprintf("%+v", *s) }

type Program struct {
	ID      uint32
	Shaders map[uint32]*Shader // map[type]shader
}

func NewProgram() *Program {
	return &Program{
		Shaders: make(map[uint32]*Shader),
	}
}

func (prog *Program) Vertex() *Shader {
	return prog.Shaders[VertexShader]
}

func (prog *Program) Geometry() *Shader {
	return prog.Shaders[GeometryShader]
}

func (prog *Program) Fragment() *Shader {
	return prog.Shaders[FragmentShader]
}

func (prog *Program) Compute() *Shader {
	return prog.Shaders[ComputeShader]
}

// func (prog *Program) String() string {
// 	var b strings.Builder
// 	b.WriteString(fmt.Sprintf("Program.ID: %d\n", prog.ID))
// 	for t, s := range prog.Shaders {
// 		b.WriteString(fmt.Sprintf(" Type: %d, Shader: %s\n", t, s))
// 	}
// 	return b.String()
// }

func (prog *Program) Use() {
	gl.UseProgram(prog.ID)
}

func (prog *Program) Delete() {
	gl.DeleteProgram(prog.ID)
}

// AddShader creates and associates a shader with this program.
func (prog *Program) AddShader(shaderType uint32, source string, uniformNames []string, attribs ...Attribute) {
	a := make(map[string]*Attribute, len(attribs))
	for i := range attribs {
		a[attribs[i].Name] = &attribs[i]
	}

	u := make(map[string]int32, len(uniformNames))
	for _, name := range uniformNames {
		u[name] = 0
	}

	prog.Shaders[shaderType] = &Shader{
		Type:     shaderType,
		Source:   source,
		Attribs:  a,
		Uniforms: u,
	}
}

func (prog *Program) Compile() error {
	for t, shader := range prog.Shaders {
		id, err := compileShader(shader.Source, t)
		if err != nil {
			return err
		}
		shader.ID = id
	}
	return nil
}

func (prog *Program) Link() error {
	prog.ID = gl.CreateProgram()

	for _, shader := range prog.Shaders {
		gl.AttachShader(prog.ID, shader.ID)
		defer gl.DeleteShader(shader.ID) // should this really be called if linking fails?
	}

	gl.LinkProgram(prog.ID)
	if err := prog.checkLinkStatus(); err != nil {
		return err
	}

	prog.Use()
	for _, shader := range prog.Shaders {
		for _, desc := range shader.Attribs {
			name := desc.Name
			id := uint32(gl.GetAttribLocation(prog.ID, gl.Str(name+"\x00")))
			shader.Attribs[name].ID = id
		}
		for name := range shader.Uniforms {
			id := gl.GetUniformLocation(prog.ID, gl.Str(name+"\x00"))
			shader.Uniforms[name] = id
		}
	}

	return nil
}

// Build will compile and link a program.
func (prog *Program) Build() error {
	err := prog.Compile()
	if err != nil {
		return fmt.Errorf("compile error: %w", err)
	}
	err = prog.Link()
	if err != nil {
		return fmt.Errorf("link error: %w", err)
	}
	return nil
}

func (prog *Program) checkLinkStatus() error {
	var status int32
	gl.GetProgramiv(prog.ID, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(prog.ID, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(prog.ID, logLength, nil, gl.Str(log))

		return fmt.Errorf("failed to link program: %v", log)
	}
	return nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source + "\x00") // gl.Strs() lies about null termination
	defer free()
	gl.ShaderSource(shader, 1, csources, nil)
	gl.CompileShader(shader)

	// this just checks compilation status
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}
