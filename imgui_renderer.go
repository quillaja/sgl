package sgl

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
)

// openGL3 implements a renderer based on github.com/go-gl/gl (v3.3-core).
type openGL3 struct {
	imguiIO imgui.IO

	glslVersion            string
	fontTexture            uint32
	shaderHandle           uint32
	vertHandle             uint32
	fragHandle             uint32
	attribLocationTex      int32
	attribLocationProjMtx  int32
	attribLocationPosition int32
	attribLocationUV       int32
	attribLocationColor    int32
	vboHandle              uint32
	elementsHandle         uint32
}

// newOpenGL3 attempts to initialize a renderer.
// An OpenGL context has to be established before calling this function.
func newOpenGL3(io imgui.IO) (*openGL3, error) {
	// assumes  gl.Init() has been called

	renderer := &openGL3{
		imguiIO:     io,
		glslVersion: "#version 330 core",
	}
	renderer.createDeviceObjects()
	return renderer, nil
}

// Dispose cleans up the resources.
func (renderer *openGL3) Dispose() {
	renderer.invalidateDeviceObjects()
}

// PreRender clears the framebuffer.
func (renderer *openGL3) PreRender(clearColor [3]float32) {
	gl.ClearColor(clearColor[0], clearColor[1], clearColor[2], 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// Render translates the ImGui draw data to OpenGL3 commands.
func (renderer *openGL3) Render(displaySize [2]float32, framebufferSize [2]float32, drawData imgui.DrawData) {
	// Avoid rendering when minimized, scale coordinates for retina displays (screen coordinates != framebuffer coordinates)
	displayWidth, displayHeight := displaySize[0], displaySize[1]
	fbWidth, fbHeight := framebufferSize[0], framebufferSize[1]
	if (fbWidth <= 0) || (fbHeight <= 0) {
		return
	}
	drawData.ScaleClipRects(imgui.Vec2{
		X: fbWidth / displayWidth,
		Y: fbHeight / displayHeight,
	})

	// Backup GL state
	var lastActiveTexture int32
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &lastActiveTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	var lastProgram int32
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &lastProgram)
	var lastTexture int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	var lastSampler int32
	gl.GetIntegerv(gl.SAMPLER_BINDING, &lastSampler)
	var lastArrayBuffer int32
	gl.GetIntegerv(gl.ARRAY_BUFFER_BINDING, &lastArrayBuffer)
	var lastElementArrayBuffer int32
	gl.GetIntegerv(gl.ELEMENT_ARRAY_BUFFER_BINDING, &lastElementArrayBuffer)
	var lastVertexArray int32
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &lastVertexArray)
	var lastPolygonMode [2]int32
	gl.GetIntegerv(gl.POLYGON_MODE, &lastPolygonMode[0])
	var lastViewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &lastViewport[0])
	var lastScissorBox [4]int32
	gl.GetIntegerv(gl.SCISSOR_BOX, &lastScissorBox[0])
	var lastBlendSrcRgb int32
	gl.GetIntegerv(gl.BLEND_SRC_RGB, &lastBlendSrcRgb)
	var lastBlendDstRgb int32
	gl.GetIntegerv(gl.BLEND_DST_RGB, &lastBlendDstRgb)
	var lastBlendSrcAlpha int32
	gl.GetIntegerv(gl.BLEND_SRC_ALPHA, &lastBlendSrcAlpha)
	var lastBlendDstAlpha int32
	gl.GetIntegerv(gl.BLEND_DST_ALPHA, &lastBlendDstAlpha)
	var lastBlendEquationRgb int32
	gl.GetIntegerv(gl.BLEND_EQUATION_RGB, &lastBlendEquationRgb)
	var lastBlendEquationAlpha int32
	gl.GetIntegerv(gl.BLEND_EQUATION_ALPHA, &lastBlendEquationAlpha)
	lastEnableBlend := gl.IsEnabled(gl.BLEND)
	lastEnableCullFace := gl.IsEnabled(gl.CULL_FACE)
	lastEnableDepthTest := gl.IsEnabled(gl.DEPTH_TEST)
	lastEnableScissorTest := gl.IsEnabled(gl.SCISSOR_TEST)

	// Setup render state: alpha-blending enabled, no face culling, no depth testing, scissor enabled, polygon fill
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.SCISSOR_TEST)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

	// Setup viewport, orthographic projection matrix
	// Our visible imgui space lies from draw_data->DisplayPos (top left) to draw_data->DisplayPos+data_data->DisplaySize (bottom right).
	// DisplayMin is typically (0,0) for single viewport apps.
	gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))
	orthoProjection := [4][4]float32{
		{2.0 / displayWidth, 0.0, 0.0, 0.0},
		{0.0, 2.0 / -displayHeight, 0.0, 0.0},
		{0.0, 0.0, -1.0, 0.0},
		{-1.0, 1.0, 0.0, 1.0},
	}
	gl.UseProgram(renderer.shaderHandle)
	gl.Uniform1i(renderer.attribLocationTex, 0)
	gl.UniformMatrix4fv(renderer.attribLocationProjMtx, 1, false, &orthoProjection[0][0])
	gl.BindSampler(0, 0) // Rely on combined texture/sampler state.

	// Recreate the VAO every time
	// (This is to easily allow multiple GL contexts. VAO are not shared among GL contexts, and
	// we don't track creation/deletion of windows so we don't have an obvious key to use to cache them.)
	var vaoHandle uint32
	gl.GenVertexArrays(1, &vaoHandle)
	gl.BindVertexArray(vaoHandle)
	gl.BindBuffer(gl.ARRAY_BUFFER, renderer.vboHandle)
	gl.EnableVertexAttribArray(uint32(renderer.attribLocationPosition))
	gl.EnableVertexAttribArray(uint32(renderer.attribLocationUV))
	gl.EnableVertexAttribArray(uint32(renderer.attribLocationColor))
	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	gl.VertexAttribPointer(uint32(renderer.attribLocationPosition), 2, gl.FLOAT, false, int32(vertexSize), gl.PtrOffset(vertexOffsetPos))
	gl.VertexAttribPointer(uint32(renderer.attribLocationUV), 2, gl.FLOAT, false, int32(vertexSize), gl.PtrOffset(vertexOffsetUv))
	gl.VertexAttribPointer(uint32(renderer.attribLocationColor), 4, gl.UNSIGNED_BYTE, true, int32(vertexSize), gl.PtrOffset(vertexOffsetCol))
	indexSize := imgui.IndexBufferLayout()
	drawType := gl.UNSIGNED_SHORT
	if indexSize == 4 {
		drawType = gl.UNSIGNED_INT
	}

	// Draw
	for _, list := range drawData.CommandLists() {
		var indexBufferOffset int

		vertexBuffer, vertexBufferSize := list.VertexBuffer()
		gl.BindBuffer(gl.ARRAY_BUFFER, renderer.vboHandle)
		gl.BufferData(gl.ARRAY_BUFFER, vertexBufferSize, vertexBuffer, gl.STREAM_DRAW)

		indexBuffer, indexBufferSize := list.IndexBuffer()
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, renderer.elementsHandle)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, indexBufferSize, indexBuffer, gl.STREAM_DRAW)

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {
				gl.BindTexture(gl.TEXTURE_2D, uint32(cmd.TextureID()))
				clipRect := cmd.ClipRect()
				gl.Scissor(int32(clipRect.X), int32(fbHeight)-int32(clipRect.W), int32(clipRect.Z-clipRect.X), int32(clipRect.W-clipRect.Y))
				gl.DrawElements(gl.TRIANGLES, int32(cmd.ElementCount()), uint32(drawType), gl.PtrOffset(indexBufferOffset))
			}
			indexBufferOffset += cmd.ElementCount() * indexSize
		}
	}
	gl.DeleteVertexArrays(1, &vaoHandle)

	// Restore modified GL state
	gl.UseProgram(uint32(lastProgram))
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
	gl.BindSampler(0, uint32(lastSampler))
	gl.ActiveTexture(uint32(lastActiveTexture))
	gl.BindVertexArray(uint32(lastVertexArray))
	gl.BindBuffer(gl.ARRAY_BUFFER, uint32(lastArrayBuffer))
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, uint32(lastElementArrayBuffer))
	gl.BlendEquationSeparate(uint32(lastBlendEquationRgb), uint32(lastBlendEquationAlpha))
	gl.BlendFuncSeparate(uint32(lastBlendSrcRgb), uint32(lastBlendDstRgb), uint32(lastBlendSrcAlpha), uint32(lastBlendDstAlpha))
	if lastEnableBlend {
		gl.Enable(gl.BLEND)
	} else {
		gl.Disable(gl.BLEND)
	}
	if lastEnableCullFace {
		gl.Enable(gl.CULL_FACE)
	} else {
		gl.Disable(gl.CULL_FACE)
	}
	if lastEnableDepthTest {
		gl.Enable(gl.DEPTH_TEST)
	} else {
		gl.Disable(gl.DEPTH_TEST)
	}
	if lastEnableScissorTest {
		gl.Enable(gl.SCISSOR_TEST)
	} else {
		gl.Disable(gl.SCISSOR_TEST)
	}
	gl.PolygonMode(gl.FRONT_AND_BACK, uint32(lastPolygonMode[0]))
	gl.Viewport(lastViewport[0], lastViewport[1], lastViewport[2], lastViewport[3])
	gl.Scissor(lastScissorBox[0], lastScissorBox[1], lastScissorBox[2], lastScissorBox[3])
}

func (renderer *openGL3) createDeviceObjects() {
	// Backup GL state
	var lastTexture int32
	var lastArrayBuffer int32
	var lastVertexArray int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	gl.GetIntegerv(gl.ARRAY_BUFFER_BINDING, &lastArrayBuffer)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &lastVertexArray)

	vertexShader := renderer.glslVersion + `
uniform mat4 ProjMtx;
in vec2 Position;
in vec2 UV;
in vec4 Color;
out vec2 Frag_UV;
out vec4 Frag_Color;
void main()
{
	Frag_UV = UV;
	Frag_Color = Color;
	gl_Position = ProjMtx * vec4(Position.xy,0,1);
}
`
	fragmentShader := renderer.glslVersion + `
uniform sampler2D Texture;
in vec2 Frag_UV;
in vec4 Frag_Color;
out vec4 Out_Color;
void main()
{
	// see NOTE in createFontsTexture()
	Out_Color = Frag_Color * texture(Texture, Frag_UV);
}
`
	renderer.shaderHandle = gl.CreateProgram()
	renderer.vertHandle = gl.CreateShader(gl.VERTEX_SHADER)
	renderer.fragHandle = gl.CreateShader(gl.FRAGMENT_SHADER)

	glShaderSource := func(handle uint32, source string) {
		csource, free := gl.Strs(source + "\x00")
		defer free()

		gl.ShaderSource(handle, 1, csource, nil)
	}

	glShaderSource(renderer.vertHandle, vertexShader)
	glShaderSource(renderer.fragHandle, fragmentShader)
	gl.CompileShader(renderer.vertHandle)
	gl.CompileShader(renderer.fragHandle)
	gl.AttachShader(renderer.shaderHandle, renderer.vertHandle)
	gl.AttachShader(renderer.shaderHandle, renderer.fragHandle)
	gl.LinkProgram(renderer.shaderHandle)

	renderer.attribLocationTex = gl.GetUniformLocation(renderer.shaderHandle, gl.Str("Texture"+"\x00"))
	renderer.attribLocationProjMtx = gl.GetUniformLocation(renderer.shaderHandle, gl.Str("ProjMtx"+"\x00"))
	renderer.attribLocationPosition = gl.GetAttribLocation(renderer.shaderHandle, gl.Str("Position"+"\x00"))
	renderer.attribLocationUV = gl.GetAttribLocation(renderer.shaderHandle, gl.Str("UV"+"\x00"))
	renderer.attribLocationColor = gl.GetAttribLocation(renderer.shaderHandle, gl.Str("Color"+"\x00"))

	gl.GenBuffers(1, &renderer.vboHandle)
	gl.GenBuffers(1, &renderer.elementsHandle)

	renderer.createFontsTexture()

	// Restore modified GL state
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
	gl.BindBuffer(gl.ARRAY_BUFFER, uint32(lastArrayBuffer))
	gl.BindVertexArray(uint32(lastVertexArray))
}

func (renderer *openGL3) createFontsTexture() {
	// NOTE: the font textures are loaded in RGBA format so that the fragment
	// shader can display color images. It uses 4 bytes per pixel for the font
	// atlas instead of 1 byte per pixel (ie 4x the memory). Maybe a big
	// deal with lots of fonts or large character sets (CJK), but for now this is fine.
	// NOTE: later, found "sizzle mask" option. Seems to work well. Allows RED
	// format images to reduce storage size, but can make opengl use R channel
	// as A channel during shader reads, and also set RGB channels to 1.

	// Build texture atlas
	io := imgui.CurrentIO()
	image := io.Fonts().TextureDataAlpha8()

	// Upload texture to graphics system
	var lastTexture int32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	gl.GenTextures(1, &renderer.fontTexture)
	gl.BindTexture(gl.TEXTURE_2D, renderer.fontTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	swizzleMask := []int32{gl.ONE, gl.ONE, gl.ONE, gl.RED}
	gl.TexParameteriv(gl.TEXTURE_2D, gl.TEXTURE_SWIZZLE_RGBA, &swizzleMask[0])
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height),
		0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)

	// Store our identifier
	io.Fonts().SetTextureID(imgui.TextureID(renderer.fontTexture))

	// Restore state
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))
}

func (renderer *openGL3) invalidateDeviceObjects() {
	if renderer.vboHandle != 0 {
		gl.DeleteBuffers(1, &renderer.vboHandle)
	}
	renderer.vboHandle = 0
	if renderer.elementsHandle != 0 {
		gl.DeleteBuffers(1, &renderer.elementsHandle)
	}
	renderer.elementsHandle = 0

	if (renderer.shaderHandle != 0) && (renderer.vertHandle != 0) {
		gl.DetachShader(renderer.shaderHandle, renderer.vertHandle)
	}
	if renderer.vertHandle != 0 {
		gl.DeleteShader(renderer.vertHandle)
	}
	renderer.vertHandle = 0

	if (renderer.shaderHandle != 0) && (renderer.fragHandle != 0) {
		gl.DetachShader(renderer.shaderHandle, renderer.fragHandle)
	}
	if renderer.fragHandle != 0 {
		gl.DeleteShader(renderer.fragHandle)
	}
	renderer.fragHandle = 0

	if renderer.shaderHandle != 0 {
		gl.DeleteProgram(renderer.shaderHandle)
	}
	renderer.shaderHandle = 0

	if renderer.fontTexture != 0 {
		gl.DeleteTextures(1, &renderer.fontTexture)
		imgui.CurrentIO().Fonts().SetTextureID(0)
		renderer.fontTexture = 0
	}
}
