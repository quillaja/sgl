package sgl

import (
	"fmt"
	"image"
	"math"
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/inkyblackness/imgui-go/v2"
)

// Init should be called once to initalize GLFW along with a
// deferred call to Destroy.
func Init() error {
	runtime.LockOSThread()
	err := glfw.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize glfw: %w", err)
	}
	return nil
}

// Destroy calls glfw.Terminate().
func Destroy() {
	glfw.Terminate()
}

// SetGLDefaults sets a few opengl options that I commonly use.
func SetGLDefaults() {
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
}

// Window implements a window, opengl contenxt based on github.com/go-gl/glfw (v3.3),
// and (optionally) imgui context. It also has additional helpful features.
type Window struct {
	// Allows direct access to some of the imgui data.
	Gui *imguiData

	// Allows direct access to the glfw window.
	GlfwWindow *glfw.Window

	// Basically 'read only' info about the dimensions of the window.
	Dimensions WindowMetric

	time             float64
	mouseJustPressed [3]bool
}

// FontMap associates a friendly name (key) with info about a font loaded
// for use with imgui. The Filename and Size fields are used during
// initializtion of imgui.
type FontMap map[string]struct {
	Filename string
	Size     float32
	Font     imgui.Font
}

// a convenient struct to hold data related to imgui.
type imguiData struct {
	IO       imgui.IO
	imguiCtx *imgui.Context
	renderer *openGL3
	Fonts    FontMap
}

// Font returns a font from the FontMap with the given name key.
// Returns the imgui default font if not found.
func (gui *imguiData) Font(name string) imgui.Font {
	if font, ok := gui.Fonts[name]; ok {
		return font.Font
	}
	return imgui.DefaultFont
}

// Destroy releases resources.
func (gui *imguiData) Destroy() {
	gui.renderer.Dispose()
	gui.imguiCtx.Destroy()
}

// WindowMetric contains info on the window position (X, Y),
// size (W, H), and windowed/fullscreen status.
// The window position and size are only valid while the window is in windowed
// mode (ie W and H are not the resolution when fullscreen).
type WindowMetric struct {
	X, Y       int
	W, H       int
	Fullscreen bool
	Resizable  bool
}

// WindowOption sets a option during window creation.
type WindowOption func(*Window) error

// NewWindow attempts to initialize a GLFW context/window/imgui etc.
func NewWindow(title string, size WindowMetric, options ...WindowOption) (*Window, error) {
	var platform *Window

	// i always just use these, so just set them here to simplify window creation
	if !size.Resizable {
		glfw.WindowHint(glfw.Resizable, glfw.False)
	}
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

	err = gl.Init() // must be called after glfw & MakeContextCurrent and before other gl functions
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenGL: %w", err)
	}

	window.SetPos(size.X, size.Y)
	defer func() {
		if window != nil {
			if size.Fullscreen {
				platform.Fullscreen(true, 0, 0)
			}
			window.Show()
		}
	}()

	platform = &Window{
		GlfwWindow: window,
	}

	// save initial window position and size
	platform.Dimensions.X, platform.Dimensions.Y = platform.GlfwWindow.GetPos()
	platform.Dimensions.W, platform.Dimensions.H = platform.GlfwWindow.GetSize()

	platform.installWindowDimensionsCallbacks()

	for i, option := range options {
		optErr := option(platform)
		if optErr != nil {
			return nil, fmt.Errorf("option %d had an error: %w", i, optErr)
		}
	}

	return platform, nil
}

// UseImgui is an option to setup additional bits so the window can be used
// with Imgui to create a user interface. Provide a key (for later reference)
// and the `Filename` and `Size` fields to load fonts for use with imgui.
// Pass nil to just use the default font. Imgui ini file disabled by default.
func UseImgui(fonts FontMap) WindowOption {
	return func(platform *Window) error {
		// imgui initialization things
		imgctx := imgui.CreateContext(nil)
		io := imgui.CurrentIO()

		io.SetIniFilename("") // default to no ini file. can be set later to enable one.

		// add fonts
		// default font would be added if the fontmap was empty, but this lets
		// imgui.DefaultFont reference the built-in font rather than the "first"
		// font added via the fontmap.
		io.Fonts().AddFontDefault()
		for name, font := range fonts {
			font.Font = io.Fonts().AddFontFromFileTTF(font.Filename, font.Size)
			fonts[name] = font
		}

		// the renderer creates a texture font atlas so fonts have
		// to be added to the "io" before this call.
		glrenderer, err := newOpenGL3(io)
		if err != nil {
			return fmt.Errorf("couldn't create imgui's opengl renderer: %w", err)
		}

		gui := imguiData{
			IO:       io,
			imguiCtx: imgctx,
			renderer: glrenderer,
			Fonts:    fonts,
		}

		platform.Gui = &gui
		platform.setImguiKeyMapping()
		platform.installImguiCallbacks()

		return nil
	}
}

// MakeContextCurrent calls Window's MakeContextCurrent() to activate the
// opengl context for use.
func (platform *Window) MakeContextCurrent() {
	platform.GlfwWindow.MakeContextCurrent()
}

// Fullscreen toggles windowed and fullscreen modes. Parameters width and height
// will set screen resolution only for fullscreen mode, and values of 0 will
// use the current resolution.
func (platform *Window) Fullscreen(full bool, width, height int) (setWidth, setHeight int) {
	if full {
		m := glfw.GetPrimaryMonitor()
		if width <= 0 {
			width = m.GetVideoMode().Width
		}
		if height <= 0 {
			height = m.GetVideoMode().Height
		}
		platform.GlfwWindow.SetMonitor(m, 0, 0, width, height, glfw.DontCare)
		platform.Dimensions.Fullscreen = true
		return width, height
	}

	d := platform.Dimensions
	platform.GlfwWindow.SetMonitor(nil, d.X, d.Y, d.W, d.H, glfw.DontCare)
	platform.Dimensions.Fullscreen = false
	return d.W, d.H
}

// Dispose cleans up the resources.
func (platform *Window) Dispose() {
	platform.GlfwWindow.Destroy()
	if platform.Gui != nil {
		platform.Gui.Destroy()
	}
}

// ShouldClose returns true if the window is to be closed.
func (platform *Window) ShouldClose() bool {
	return platform.GlfwWindow.ShouldClose()
}

// PollEvents handles all pending window events.
func (platform *Window) PollEvents() {
	glfw.PollEvents()
}

// SwapBuffers performs a buffer swap.
func (platform *Window) SwapBuffers() {
	platform.GlfwWindow.SwapBuffers()
}

// ClearBuffers clears color buffer and depth buffer.
func (platform *Window) ClearBuffers() {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

// RenderImgui will perform the beginning and ending steps of rendering
// the imgui constructed by calls to the imgui pkg in the 'gui' function.
func (platform *Window) RenderImgui(gui func()) {
	// start 'frame'
	platform.forwardStateToImgui()
	imgui.NewFrame()

	gui()

	// end 'frame'
	imgui.Render()

	// render gui
	drawdata := imgui.RenderedDrawData()
	platform.Gui.renderer.Render(platform.DisplaySize(), platform.FramebufferSize(), drawdata)
}

// Aspect returns aspect ratio.
func (platform *Window) Aspect() float32 {
	size := platform.DisplaySize()
	return size[0] / size[1]
}

// DisplaySize returns the dimension of the display.
func (platform *Window) DisplaySize() [2]float32 {
	w, h := platform.GlfwWindow.GetSize()
	return [2]float32{float32(w), float32(h)}
}

// FramebufferSize returns the dimension of the framebuffer.
func (platform *Window) FramebufferSize() [2]float32 {
	w, h := platform.GlfwWindow.GetFramebufferSize()
	return [2]float32{float32(w), float32(h)}
}

// ScreenCapture saves a copy of the opengl front buffer and saves it into
// an image.Image.
func (platform *Window) ScreenCapture() image.Image {
	w, h := platform.GlfwWindow.GetFramebufferSize()
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	gl.ReadBuffer(gl.FRONT)
	gl.ReadPixels(0, 0, int32(w), int32(h), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))

	// flip image vertically
	temp := make([]byte, 4*rgba.Stride)
	for y := 0; y < rgba.Bounds().Dy()/2; y++ {
		top := rgba.Pix[y*rgba.Stride : (y+1)*rgba.Stride]
		bottom := rgba.Pix[(rgba.Bounds().Dy()-1-y)*rgba.Stride : (rgba.Bounds().Dy()-y)*rgba.Stride]
		copy(temp, top)
		copy(top, bottom)
		copy(bottom, temp)
	}
	return rgba
}

// ClipboardText returns the current clipboard text, if available.
func (platform *Window) ClipboardText() string {
	return platform.GlfwWindow.GetClipboardString()
}

// SetClipboardText sets the text as the current clipboard text.
func (platform *Window) SetClipboardText(text string) {
	platform.GlfwWindow.SetClipboardString(text)
}

// installWindowDimensionsCallbacks set various window/frame size callbacks
func (platform *Window) installWindowDimensionsCallbacks() {
	platform.GlfwWindow.SetPosCallback(func(w *glfw.Window, xpos, ypos int) {
		// save position only if in windowed mode
		if !platform.Dimensions.Fullscreen {
			platform.Dimensions.X, platform.Dimensions.Y = xpos, ypos
		}
	})
	platform.GlfwWindow.SetSizeCallback(func(w *glfw.Window, width, height int) {
		// save size only if in windowed mode
		if !platform.Dimensions.Fullscreen {
			platform.Dimensions.W, platform.Dimensions.H = width, height
		}
	})
	platform.GlfwWindow.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})
}

///////////////////////////////
// imgui hooks and things
///////////////////////////////

// CapturesKeyboard returns true if Imgui is capturing keyboard input.
func (platform *Window) CapturesKeyboard() bool {
	return platform.Gui != nil && platform.Gui.IO.WantCaptureKeyboard()
}

// CapturesMouse returns true if Imgui is capturing mouse input.
func (platform *Window) CapturesMouse() bool {
	return platform.Gui != nil && platform.Gui.IO.WantCaptureMouse()
}

// forwardStateToImgui marks the begin of a render pass. It forwards all current state to imgui IO.
func (platform *Window) forwardStateToImgui() {
	// Setup display size (every frame to accommodate for window resizing)
	displaySize := platform.DisplaySize()
	platform.Gui.IO.SetDisplaySize(imgui.Vec2{X: displaySize[0], Y: displaySize[1]})

	// Setup time step
	currentTime := glfw.GetTime()
	platform.Gui.IO.SetDeltaTime(float32(currentTime - platform.time))
	platform.time = currentTime

	// Setup inputs
	if platform.GlfwWindow.GetAttrib(glfw.Focused) != 0 {
		x, y := platform.GlfwWindow.GetCursorPos()
		platform.Gui.IO.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
	} else {
		platform.Gui.IO.SetMousePosition(imgui.Vec2{X: -math.MaxFloat32, Y: -math.MaxFloat32})
	}

	for i := 0; i < len(platform.mouseJustPressed); i++ {
		down := platform.mouseJustPressed[i] || (platform.GlfwWindow.GetMouseButton(glfwButtonIDByIndex[i]) == glfw.Press)
		platform.Gui.IO.SetMouseButtonDown(i, down)
		platform.mouseJustPressed[i] = false
	}
}

func (platform *Window) setImguiKeyMapping() {
	// Keyboard mapping. ImGui will use those indices to peek into the io.KeysDown[] array.
	platform.Gui.IO.KeyMap(imgui.KeyTab, int(glfw.KeyTab))
	platform.Gui.IO.KeyMap(imgui.KeyLeftArrow, int(glfw.KeyLeft))
	platform.Gui.IO.KeyMap(imgui.KeyRightArrow, int(glfw.KeyRight))
	platform.Gui.IO.KeyMap(imgui.KeyUpArrow, int(glfw.KeyUp))
	platform.Gui.IO.KeyMap(imgui.KeyDownArrow, int(glfw.KeyDown))
	platform.Gui.IO.KeyMap(imgui.KeyPageUp, int(glfw.KeyPageUp))
	platform.Gui.IO.KeyMap(imgui.KeyPageDown, int(glfw.KeyPageDown))
	platform.Gui.IO.KeyMap(imgui.KeyHome, int(glfw.KeyHome))
	platform.Gui.IO.KeyMap(imgui.KeyEnd, int(glfw.KeyEnd))
	platform.Gui.IO.KeyMap(imgui.KeyInsert, int(glfw.KeyInsert))
	platform.Gui.IO.KeyMap(imgui.KeyDelete, int(glfw.KeyDelete))
	platform.Gui.IO.KeyMap(imgui.KeyBackspace, int(glfw.KeyBackspace))
	platform.Gui.IO.KeyMap(imgui.KeySpace, int(glfw.KeySpace))
	platform.Gui.IO.KeyMap(imgui.KeyEnter, int(glfw.KeyEnter))
	platform.Gui.IO.KeyMap(imgui.KeyEscape, int(glfw.KeyEscape))
	platform.Gui.IO.KeyMap(imgui.KeyA, int(glfw.KeyA))
	platform.Gui.IO.KeyMap(imgui.KeyC, int(glfw.KeyC))
	platform.Gui.IO.KeyMap(imgui.KeyV, int(glfw.KeyV))
	platform.Gui.IO.KeyMap(imgui.KeyX, int(glfw.KeyX))
	platform.Gui.IO.KeyMap(imgui.KeyY, int(glfw.KeyY))
	platform.Gui.IO.KeyMap(imgui.KeyZ, int(glfw.KeyZ))
}

func (platform *Window) installImguiCallbacks() {
	platform.GlfwWindow.SetMouseButtonCallback(platform.mouseButtonChange)
	platform.GlfwWindow.SetScrollCallback(platform.mouseScrollChange)
	platform.GlfwWindow.SetKeyCallback(platform.keyChange)
	platform.GlfwWindow.SetCharCallback(platform.charChange)
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

func (platform *Window) mouseButtonChange(window *glfw.Window, rawButton glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	buttonIndex, known := glfwButtonIndexByID[rawButton]

	if known && (action == glfw.Press) {
		platform.mouseJustPressed[buttonIndex] = true
	}
}

func (platform *Window) mouseScrollChange(window *glfw.Window, x, y float64) {
	platform.Gui.IO.AddMouseWheelDelta(float32(x), float32(y))
}

func (platform *Window) keyChange(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		platform.Gui.IO.KeyPress(int(key))
	}
	if action == glfw.Release {
		platform.Gui.IO.KeyRelease(int(key))
	}

	// Modifiers are not reliable across systems
	platform.Gui.IO.KeyCtrl(int(glfw.KeyLeftControl), int(glfw.KeyRightControl))
	platform.Gui.IO.KeyShift(int(glfw.KeyLeftShift), int(glfw.KeyRightShift))
	platform.Gui.IO.KeyAlt(int(glfw.KeyLeftAlt), int(glfw.KeyRightAlt))
	platform.Gui.IO.KeySuper(int(glfw.KeyLeftSuper), int(glfw.KeyRightSuper))
}

func (platform *Window) charChange(window *glfw.Window, char rune) {
	platform.Gui.IO.AddInputCharacters(string(char))
}
