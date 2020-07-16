package sgl

import (
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type GlError map[uint32]string

func (e GlError) Error() string {
	msgs := make([]string, 0, len(e))
	for _, m := range e {
		msgs = append(msgs, m)
	}
	return strings.Join(msgs, ", ")
}

func CheckError() error {
	errorCode := gl.GetError()
	if errorCode == gl.NO_ERROR {
		return nil
	}

	err := make(GlError)
	for ; errorCode != gl.NO_ERROR; errorCode = gl.GetError() {
		var errorMsg string
		switch errorCode {
		case gl.INVALID_ENUM:
			errorMsg = "INVALID_ENUM"
		case gl.INVALID_VALUE:
			errorMsg = "INVALID_VALUE"
		case gl.INVALID_OPERATION:
			errorMsg = "INVALID_OPERATION"
		case gl.STACK_OVERFLOW:
			errorMsg = "STACK_OVERFLOW"
		case gl.STACK_UNDERFLOW:
			errorMsg = "STACK_UNDERFLOW"
		case gl.OUT_OF_MEMORY:
			errorMsg = "OUT_OF_MEMORY"
		case gl.INVALID_FRAMEBUFFER_OPERATION:
			errorMsg = "INVALID_FRAMEBUFFER_OPERATION"
		}

		err[errorCode] = errorMsg
	}

	return err
}
