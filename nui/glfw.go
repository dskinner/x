package nui

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// TODO pack this into a window wrapper that surface returns
// ... and rename surface.
var (
	pollEvents        = glfw.PollEvents
	waitEvents        = glfw.WaitEvents
	waitEventsTimeout = glfw.WaitEventsTimeout
)

func surface() (window *glfw.Window, terminate func()) {
	var err error

	err = glfw.Init()
	if err != nil {
		panic(err)
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	window, err = glfw.CreateWindow(640, 480, "Testing", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		panic(err)
	}

	// TODO get this out of here was actual data is being
	// passed on to client, then they can wrap their setup
	// func to have signature func() and don it.
	setup(window.GetFramebufferSize())

	return window, glfw.Terminate
}

// TODO start translating things like two ints into common like image.Point.
func setup(width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.ClearColor(0, 0, 0, 0)

	// gl.MatrixMode(gl.PROJECTION)
	// gl.LoadIdentity()

	// aspectRatio := float64(width) / float64(height)
	// gl.Ortho(-aspectRatio, aspectRatio, -1, 1, 1.0, 10.0)

	// gl.MatrixMode(gl.MODELVIEW)
	// gl.LoadIdentity()
}
