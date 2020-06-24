# sgl
simple wrapper on glfw/opengl for use in personal projects.
can use go port of imgui for gui.

# Todo
- [ ] some useful 'premade' shader programs
    - [ ] 2d image "splat" (png, jpg, image.Image,...)
    - [ ] 3d shapes with vertex colors, normals, textures, point/directional lights
- [ ] More opengl features.
- [ ] Improve text/font rendering in `font.go`.
- [x] rename some types and functions.
- [ ] add utilites (from my gaia stars program)
- [ ] make `Texture2D` more flexible

## Changelog

- Next
    - screen capture to `image.Image`
    - added simple framebuffer object.
    - renamed main data structure.
    - different api for window/platform creation.