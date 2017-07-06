// +build shiny gomobile

// Glwidget app for shiny and gomobile.
//
// Run on desktop:
//  go run shiny.go glwidget.go
//  go run gomobile.go glwidget.go
//
// Install on device:
//  gomobile install -tags gomobile
package main

import (
	"image"
	"image/draw"
	"os"
	"time"

	"dasa.cc/x/glw"
	"golang.org/x/exp/shiny/gesture"
	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/math/f32"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/gl"
)

func init() {
	theme.Default = &theme.Theme{
		Palette: &theme.Palette{
			theme.Light:      image.Uniform{colornames.BlueGrey100},
			theme.Neutral:    image.Uniform{colornames.BlueGrey500},
			theme.Dark:       image.Uniform{colornames.BlueGrey900},
			theme.Accent:     image.Uniform{colornames.DeepOrangeA200},
			theme.Foreground: image.Uniform{colornames.Black},
			theme.Background: image.Uniform{colornames.BlueGrey500},
		},
	}
}

type GLWidget struct {
	node.LeafEmbed
	ctx gl.Context
	buf glw.FrameBuffer

	useFrameBuffer bool

	prg glw.Program

	Proj      glw.U16fv
	Model     glw.U16fv
	Color     glw.U4fv
	Vertex    glw.A3fv
	VertexBuf glw.FloatBuffer
	VertexInd glw.UintBuffer

	animating uint32
	evg       gesture.Event
}

func NewGLWidget(ctx gl.Context, useFrameBuffer bool) *GLWidget {
	w := &GLWidget{
		ctx:            glw.With(ctx),
		useFrameBuffer: useFrameBuffer,
	}
	w.Wrapper = w
	return w
}

func (w *GLWidget) OnLifecycleEvent(e lifecycle.Event) {
	switch e.Crosses(lifecycle.StageVisible) {
	case lifecycle.CrossOn:
		if w.useFrameBuffer {
			w.buf.Create()
			w.buf.Attach(0, 0)
		}
		w.prg.MustBuild(vsrc, fsrc)
		w.prg.SetLocations(w)
		w.prg.Use()

		w.VertexBuf.Create(gl.STATIC_DRAW, []float32{
			-0.2, -0.2, 0,
			-0.2, +0.2, 0,
			+0.2, +0.2, 0,
			+0.2, -0.2, 0})
		w.VertexInd.Create(gl.STATIC_DRAW, []uint32{0, 1, 2, 0, 2, 3})
		w.Vertex.Pointer()

		opts := []func(glw.Animator){
			glw.Duration(750 * time.Millisecond),
			glw.Notify(&w.animating),
		}
		w.Color.Animator(opts...)
		w.Model.Animator(opts...)
	case lifecycle.CrossOff:
		if w.useFrameBuffer {
			w.buf.Detach()
			w.buf.Delete()
		}
		w.VertexInd.Delete()
		w.VertexBuf.Delete()
		w.prg.Delete()
		w.ctx = nil
	}
}

func (w *GLWidget) Measure(*theme.Theme, int, int) { w.MeasuredSize = w.Rect.Size() }

func (w *GLWidget) Layout(t *theme.Theme) {
	w.ctx.Disable(gl.CULL_FACE)
	w.ctx.Disable(gl.DEPTH_TEST)
	w.ctx.Enable(gl.BLEND)
	w.ctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	w.ctx.ClearColor(glw.RGBA(t.Palette.Background()))
	w.Color.Set(glw.Vec4(glw.RGBA(t.Palette.Accent())))

	if size := w.Rect.Size(); size != w.MeasuredSize && size != image.ZP {
		ar := float32(size.X) / float32(size.Y)
		w.Proj.Ortho(-ar, ar, -1, 1, 1.0, 10.0)
		if w.useFrameBuffer {
			w.buf.Update(size.X, size.Y)
		}
		w.ctx.Viewport(0, 0, size.X, size.Y)
		w.Mark(node.MarkNeedsPaintBase)
	}
}

func (w *GLWidget) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	now := time.Now()
	if w.animating != 0 {
		w.Model.Step(now)
		w.Color.Step(now)
		w.Mark(node.MarkNeedsPaintBase)
	}

	w.ctx.Clear(gl.COLOR_BUFFER_BIT)
	w.Model.Update()
	w.Color.Update()
	w.VertexInd.Draw(gl.TRIANGLES)

	if w.useFrameBuffer {
		// TODO support for gomobile example
		draw.Draw(ctx.Dst, w.Rect.Add(origin), w.buf.RGBA(), image.ZP, draw.Over)
	}

	if w.evg != (gesture.Event{}) {
		x, y := w.Proj.Inv2f(cton(w.evg.CurrentPos.X, w.evg.CurrentPos.Y, w.Rect))
		w.Model.Stage(now, glw.TranslateTo(f32.Vec4{x, y, 0, 0}))
		g := float32(uint8(w.evg.CurrentPos.X/10)) / 255
		b := float32(uint8(w.evg.CurrentPos.Y/10)) / 255
		w.Color.Stage(now, glw.TranslateTo(f32.Vec4{0, g, b, 1}))
		w.evg = gesture.Event{}
		w.Mark(node.MarkNeedsPaintBase)
	}

	return nil
}

func (w *GLWidget) OnInputEvent(ev interface{}, origin image.Point) node.EventHandled {
	switch ev := ev.(type) {
	case gesture.Event:
		w.evg = ev
		w.Mark(node.MarkNeedsPaintBase)
		return node.Handled
	case key.Event:
		if ev.Code == key.CodeEscape {
			os.Exit(0)
		}
		return node.NotHandled
	default:
		return node.NotHandled
	}
}

const (
	vsrc = `#version 100
uniform mat4 proj;
uniform mat4 model;
attribute vec4 vertex;
void main() {
	gl_Position = proj*model*vertex;
}`

	fsrc = `#version 100
precision mediump float;
uniform vec4 color;
void main() {
	gl_FragColor = color;
}`
)
