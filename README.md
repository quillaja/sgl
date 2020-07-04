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

## Changelog
- 0.4.0
    - skybox
    - resizable window option at creation
    - small api changes to `Texture2D`
    - `Selecter` uitility, basically for easier imgui combo and list boxes
    - changes to Chord/ChordSet that simplfy api and fix a subtle timing bug.
    - addition to Window of "Timer" named Clock
    - additions to Window for use in rendering loop control
    - addition of "IsNthFrame()" to Timer
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