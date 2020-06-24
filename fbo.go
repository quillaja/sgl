package sgl

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// UseDefaultFramebuffer unbinds other FBOs and binds the default framebuffer.
func UseDefaultFramebuffer() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

// Fbo is a very simple Frame Buffer Object with a texture
// bound as a color attachment and renderbuffer for depth and stencil attachments.
type Fbo struct {
	ID              uint32
	Width, Height   int32
	depthStencilRbo uint32
	ColorBufferTex  *Texture2D
}

// NewFbo creates a FBO of the given dimensions.
func NewFbo(width, height int) (*Fbo, error) {

	var fbo Fbo
	fbo.Width, fbo.Height = int32(width), int32(height)
	gl.GenFramebuffers(1, &fbo.ID)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo.ID)

	// generate texture and attach it to as a color buffer for this fbo
	fbo.ColorBufferTex = new(Texture2D)
	fbo.ColorBufferTex.Width, fbo.ColorBufferTex.Height = fbo.Width, fbo.Height
	gl.GenTextures(1, &fbo.ColorBufferTex.ID)
	gl.BindTexture(gl.TEXTURE_2D, fbo.ColorBufferTex.ID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB, fbo.Width, fbo.Height, 0, gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(nil))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.BindTexture(gl.TEXTURE_2D, 0)                                                                       // unbind texture
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fbo.ColorBufferTex.ID, 0) // attach

	// generate and attach render buffer object as depth and stencil buffers for this fbo.
	gl.GenRenderbuffers(1, &fbo.depthStencilRbo)
	gl.BindRenderbuffer(gl.RENDERBUFFER, fbo.depthStencilRbo)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, fbo.Width, fbo.Height)
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)                                                                       // unbind rbo
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.RENDERBUFFER, fbo.depthStencilRbo) // attach

	// check that fbo is complete
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		fbo.Delete()
		return nil, fmt.Errorf("framebuffer is not complete")
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	return &fbo, nil
}

// Delete resources associated with the FBO.
func (fbo *Fbo) Delete() {
	fbo.ColorBufferTex.Delete()
	gl.DeleteRenderbuffers(1, &fbo.depthStencilRbo)
	gl.DeleteFramebuffers(1, &fbo.ID)
}

// Use binds the FBO for use.
func (fbo *Fbo) Use() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo.ID)
}
