# sgl
simple wrapper on glfw/opengl for use in personal projects.
can use go port of imgui for gui.

# Todo
- [ ] some useful 'premade' shader programs
    - [ ] 2d image "splat" (png, jpg, image.Image,...)
    - [ ] 3d shapes with vertex colors, normals, textures, point/directional lights
    - [x] skybox
- [ ] More opengl features.
- [ ] Improve text/font rendering in `font.go`.
    - should be able to queue all strings into a single vbo with vertex coords and texture coords, then draw all in a single draw call.
- [x] rename some types and functions.
- [x] add utilites (from my gaia stars program)
- [ ] make `Texture2D` more flexible
- [ ] program->vao mapper? 
    - to do foreach program { enable program; foreach vao { draw vao } }
- [ ] change all "load" funcs that take a string path to also accept a `fs.FS` as the root
    - this will allow me to use pkg `embed` or `os.DirFS`, etc

## Changelog
- 0.6.0 todo
    - mouse and camera structs (when finalized).
    - keyboard/mouse somehow integrated into Window or other "input" manager?
    - map of pressed keys/mouse buttons so don't pass *Window to ChordSet.
- 0.5.0
    - ChordSet functionality changed so chords are executed until all matching
    chords are executed or a chord with "Stop" set to true is executed.
    - icon loading window option.
    - convienience function to get font names from FontMap for use in Selecter.
    - real fix for imgui renderer to display images in color (from v0.3.0)
    - function to check opengl errors, and error type.
- 0.4.0
    - skybox
    - resizable window option at creation
    - small api changes to `Texture2D`
    - `Selecter` uitility, basically for easier imgui combo and list boxes
    - changes to Chord/ChordSet that simplfy api and fix a subtle timing bug.
    - addition to Window of "Timer" named Clock
    - additions to Window for use in rendering loop control
    - addition of "IsNthFrame()" to Timer
    - simplify Vao to make mostly-unused params to Set*() and Draw() use defaults.
- 0.3.0
    - changed/"fixed" imgui renderer to display images in color
    - modified window creation api to allow for fonts with imgui
    - package level function to set "default" (for me) opengl settings
    - a few useful utility types (Timer, Cycler, AnimationMap...)
- 0.2.0
    - screen capture to `image.Image`
    - added simple framebuffer object.
    - renamed main data structure.
    - different api for window/platform creation.