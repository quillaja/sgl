package sgl

import (
	"fmt"
	"math"
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/inkyblackness/imgui-go/v2"
)

// GLFW implements a platform based on github.com/go-gl/glfw (v3.3).
type GLFW struct {
	IO       imgui.IO
	imguiCtx *imgui.Context

	Renderer *OpenGL3
	Window   *glfw.Window

	WinDims WindowMetric

	time             float64
	mouseJustPressed [3]bool
}

// WindowMetric contains info on the window position (X, Y),
// size (W, H), and windowed/fullscreen status.
type WindowMetric struct {
	X, Y       int
	W, H       int
	Fullscreen bool
}

// NewGLFW attempts to initialize a GLFW context/window/imgui etc.
func NewGLFW(title string, size WindowMetric, font *imgui.FontAtlas) (*GLFW, error) {
	runtime.LockOSThread()
	var platform *GLFW

	err := glfw.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize glfw: %w", err)
	}

	// i always just use these, so just set them here to simplify window creation
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(size.W, size.H, title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("failed to create window: %w", err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	window.SetPos(size.X, size.Y)
	defer func() {
		if window != nil {
			if size.Fullscreen {
				platform.Fullscreen(true, 0, 0)
			}
			window.Show()
		}
	}()

	// imgui initialization things
	imgctx := imgui.CreateContext(font)
	io := imgui.CurrentIO()

	glrenderer, err := NewOpenGL3(io)
	if err != nil {
		panic(err)
	}

	platform = &GLFW{
		IO:       io,
		imguiCtx: imgctx,
		Window:   window,
		Renderer: glrenderer,
	}
	platform.setKeyMapping()
	platform.installCallbacks()

	// save initial window position and size
	platform.WinDims.X, platform.WinDims.Y = platform.Window.GetPos()
	platform.WinDims.W, platform.WinDims.H = platform.Window.GetSize()

	return platform, nil
}

// Fullscreen toggles windowed and fullscreen modes. Parameters width and height
// will set screen resolution only for fullscreen mode, and values of 0 will
// use the current resolution.
func (platform *GLFW) Fullscreen(full bool, width, height int) (setWidth, setHeight int) {
	if full {
		m := glfw.GetPrimaryMonitor()
		if width <= 0 {
			width = m.GetVideoMode().Width
		}
		if height <= 0 {
			height = m.GetVideoMode().Height
		}
		platform.Window.SetMonitor(m, 0, 0, width, height, glfw.DontCare)
		platform.WinDims.Fullscreen = true
		return width, height
	}

	d := platform.WinDims
	platform.Window.SetMonitor(nil, d.X, d.Y, d.W, d.H, glfw.DontCare)
	platform.WinDims.Fullscreen = false
	return d.W, d.H
}

// Dispose cleans up the resources.
func (platform *GLFW) Dispose() {
	platform.Window.Destroy()
	glfw.Terminate()
	platform.Renderer.Dispose()
	platform.imguiCtx.Destroy()
}

// ShouldClose returns true if the window is to be closed.
func (platform *GLFW) ShouldClose() bool {
	return platform.Window.ShouldClose()
}

// PollEvents handles all pending window events.
func (platform *GLFW) PollEvents() {
	glfw.PollEvents()
}

// SwapBuffers performs a buffer swap.
func (platform *GLFW) SwapBuffers() {
	platform.Window.SwapBuffers()
}

// ClearBuffers clears color buffer and optionally depth buffer.
func (platform *GLFW) ClearBuffers(depthBuffer bool) {
	if depthBuffer {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		return
	}
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// RenderImgui will perform the beginning and ending steps of rendering
// the imgui constructed by calls to the imgui pkg in the 'gui' function.
func (platform *GLFW) RenderImgui(gui func()) {
	// start 'frame'
	platform.NewFrame()
	imgui.NewFrame()

	gui()

	// end 'frame'
	imgui.Render()

	// render gui
	drawdata := imgui.RenderedDrawData()
	platform.Renderer.Render(platform.DisplaySize(), platform.FramebufferSize(), drawdata)
}

// Aspect returns aspect ratio.
func (platform *GLFW) Aspect() float32 {
	size := platform.DisplaySize()
	return size[0] / size[1]
}

// DisplaySize returns the dimension of the display.
func (platform *GLFW) DisplaySize() [2]float32 {
	w, h := platform.Window.GetSize()
	return [2]float32{float32(w), float32(h)}
}

// FramebufferSize returns the dimension of the framebuffer.
func (platform *GLFW) FramebufferSize() [2]float32 {
	w, h := platform.Window.GetFramebufferSize()
	return [2]float32{float32(w), float32(h)}
}

// NewFrame marks the begin of a render pass. It forwards all current state to imgui IO.
func (platform *GLFW) NewFrame() {
	// Setup display size (every frame to accommodate for window resizing)
	displaySize := platform.DisplaySize()
	platform.IO.SetDisplaySize(imgui.Vec2{X: displaySize[0], Y: displaySize[1]})

	// Setup time step
	currentTime := glfw.GetTime()
	platform.IO.SetDeltaTime(float32(currentTime - platform.time))
	platform.time = currentTime

	// Setup inputs
	if platform.Window.GetAttrib(glfw.Focused) != 0 {
		x, y := platform.Window.GetCursorPos()
		platform.IO.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
	} else {
		platform.IO.SetMousePosition(imgui.Vec2{X: -math.MaxFloat32, Y: -math.MaxFloat32})
	}

	for i := 0; i < len(platform.mouseJustPressed); i++ {
		down := platform.mouseJustPressed[i] || (platform.Window.GetMouseButton(glfwButtonIDByIndex[i]) == glfw.Press)
		platform.IO.SetMouseButtonDown(i, down)
		platform.mouseJustPressed[i] = false
	}
}

// CapturesKeyboard returns true if Imgui is capturing keyboard input.
func (platform *GLFW) CapturesKeyboard() bool {
	return platform.IO.WantCaptureKeyboard()
}

// CapturesMouse returns true if Imgui is capturing mouse input.
func (platform *GLFW) CapturesMouse() bool {
	return platform.IO.WantCaptureMouse()
}

func (platform *GLFW) setKeyMapping() {
	// Keyboard mapping. ImGui will use those indices to peek into the io.KeysDown[] array.
	platform.IO.KeyMap(imgui.KeyTab, int(glfw.KeyTab))
	platform.IO.KeyMap(imgui.KeyLeftArrow, int(glfw.KeyLeft))
	platform.IO.KeyMap(imgui.KeyRightArrow, int(glfw.KeyRight))
	platform.IO.KeyMap(imgui.KeyUpArrow, int(glfw.KeyUp))
	platform.IO.KeyMap(imgui.KeyDownArrow, int(glfw.KeyDown))
	platform.IO.KeyMap(imgui.KeyPageUp, int(glfw.KeyPageUp))
	platform.IO.KeyMap(imgui.KeyPageDown, int(glfw.KeyPageDown))
	platform.IO.KeyMap(imgui.KeyHome, int(glfw.KeyHome))
	platform.IO.KeyMap(imgui.KeyEnd, int(glfw.KeyEnd))
	platform.IO.KeyMap(imgui.KeyInsert, int(glfw.KeyInsert))
	platform.IO.KeyMap(imgui.KeyDelete, int(glfw.KeyDelete))
	platform.IO.KeyMap(imgui.KeyBackspace, int(glfw.KeyBackspace))
	platform.IO.KeyMap(imgui.KeySpace, int(glfw.KeySpace))
	platform.IO.KeyMap(imgui.KeyEnter, int(glfw.KeyEnter))
	platform.IO.KeyMap(imgui.KeyEscape, int(glfw.KeyEscape))
	platform.IO.KeyMap(imgui.KeyA, int(glfw.KeyA))
	platform.IO.KeyMap(imgui.KeyC, int(glfw.KeyC))
	platform.IO.KeyMap(imgui.KeyV, int(glfw.KeyV))
	platform.IO.KeyMap(imgui.KeyX, int(glfw.KeyX))
	platform.IO.KeyMap(imgui.KeyY, int(glfw.KeyY))
	platform.IO.KeyMap(imgui.KeyZ, int(glfw.KeyZ))
}

func (platform *GLFW) installCallbacks() {
	platform.Window.SetMouseButtonCallback(platform.mouseButtonChange)
	platform.Window.SetScrollCallback(platform.mouseScrollChange)
	platform.Window.SetKeyCallback(platform.keyChange)
	platform.Window.SetCharCallback(platform.charChange)

	// set various window/frame size callbacks
	platform.Window.SetPosCallback(func(w *glfw.Window, xpos, ypos int) {
		// save position only if in windowed mode
		if !platform.WinDims.Fullscreen {
			platform.WinDims.X, platform.WinDims.Y = xpos, ypos
		}
	})
	platform.Window.SetSizeCallback(func(w *glfw.Window, width, height int) {
		// save size only if in windowed mode
		if !platform.WinDims.Fullscreen {
			platform.WinDims.W, platform.WinDims.H = width, height
		}
	})
	platform.Window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})
}

var glfwButtonIndexByID = map[glfw.MouseButton]int{
	glfw.MouseButton1: 0,
	glfw.MouseButton2: 1,
	glfw.MouseButton3: 2,
}

var glfwButtonIDByIndex = map[int]glfw.MouseButton{
	0: glfw.MouseButton1,
	1: glfw.MouseButton2,
	2: glfw.MouseButton3,
}

func (platform *GLFW) mouseButtonChange(window *glfw.Window, rawButton glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	buttonIndex, known := glfwButtonIndexByID[rawButton]

	if known && (action == glfw.Press) {
		platform.mouseJustPressed[buttonIndex] = true
	}
}

func (platform *GLFW) mouseScrollChange(window *glfw.Window, x, y float64) {
	platform.IO.AddMouseWheelDelta(float32(x), float32(y))
}

func (platform *GLFW) keyChange(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		platform.IO.KeyPress(int(key))
	}
	if action == glfw.Release {
		platform.IO.KeyRelease(int(key))
	}

	// Modifiers are not reliable across systems
	platform.IO.KeyCtrl(int(glfw.KeyLeftControl), int(glfw.KeyRightControl))
	platform.IO.KeyShift(int(glfw.KeyLeftShift), int(glfw.KeyRightShift))
	platform.IO.KeyAlt(int(glfw.KeyLeftAlt), int(glfw.KeyRightAlt))
	platform.IO.KeySuper(int(glfw.KeyLeftSuper), int(glfw.KeyRightSuper))
}

func (platform *GLFW) charChange(window *glfw.Window, char rune) {
	platform.IO.AddInputCharacters(string(char))
}

// ClipboardText returns the current clipboard text, if available.
func (platform *GLFW) ClipboardText() string {
	return platform.Window.GetClipboardString()
}

// SetClipboardText sets the text as the current clipboard text.
func (platform *GLFW) SetClipboardText(text string) {
	platform.Window.SetClipboardString(text)
}
