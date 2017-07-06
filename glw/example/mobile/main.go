package main

import (
	"image"
	"log"
	"os"
	"time"

	"dasa.cc/x/glw"

	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/image/math/f32"
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"
)

var (
	fps       int
	now       = time.Now()
	lastpaint = now
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("mobile: ")
}

func main() {
	app.Main(func(a app.App) {
		w := &GLWidget{}
		go func() {
			for range time.Tick(500 * time.Millisecond) {
				log.Printf("%v FPS\n", fps)
			}
		}()
		var lastTouch touch.Event
		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				w.OnLifecycleEvent(e)
				a.Send(paint.Event{})
			case size.Event:
				w.OnSizeEvent(e)
			case paint.Event:
				if w.ctx == nil || e.External {
					continue
				}
				now = time.Now()
				w.Paint()
				a.Publish()
				a.Send(paint.Event{})
				fps = int(time.Second / now.Sub(lastpaint))
				lastpaint = now
			case touch.Event, key.Event:
				switch e := e.(type) {
				case touch.Event:
					e.X = float32(int(e.X))
					e.Y = float32(int(e.Y))
					if e != lastTouch {
						// w.OnInputEvent(e)
						w.lastTouch = e
					}
					lastTouch = e
				default:
					w.OnInputEvent(e)
				}
			}
		}
	})
}

type GLWidget struct {
	ctx  gl.Context
	size image.Point

	prg glw.Program

	Proj      glw.U16fv
	Model     glw.U16fv
	Color     glw.U4fv
	Vertex    glw.A3fv
	VertexBuf glw.FloatBuffer
	VertexInd glw.UintBuffer

	animating uint32
	lastTouch touch.Event
}

func (w *GLWidget) OnLifecycleEvent(e lifecycle.Event) {
	switch e.Crosses(lifecycle.StageVisible) {
	case lifecycle.CrossOn:
		glctx, _ := e.DrawContext.(gl.Context)
		w.ctx = glw.With(glctx)
		w.prg.MustBuild(vsrc, fsrc)
		w.prg.SetLocations(w)
		w.prg.Use()

		w.VertexBuf.Create(gl.STATIC_DRAW, []float32{-0.5, -0.5, 0, -0.5, +0.5, 0, +0.5, +0.5, 0, +0.5, -0.5, 0})
		w.VertexInd.Create(gl.STATIC_DRAW, []uint32{0, 1, 2, 0, 2, 3})
		w.Vertex.Pointer()

		opts := []func(glw.Animator){
			glw.Duration(750 * time.Millisecond),
			glw.Notify(&w.animating),
		}
		w.Color.Animator(opts...)
		w.Model.Animator(opts...)
	case lifecycle.CrossOff:
		w.VertexInd.Delete()
		w.VertexBuf.Delete()
		w.prg.Delete()
		w.ctx = nil
	}
}

func (w *GLWidget) OnSizeEvent(e size.Event) {
	w.size = e.Size()
	if w.ctx == nil {
		return
	}

	w.ctx.Disable(gl.CULL_FACE)
	w.ctx.Disable(gl.DEPTH_TEST)
	w.ctx.Enable(gl.BLEND)
	w.ctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	w.ctx.ClearColor(glw.RGBA(colornames.BlueGrey500))
	w.Color.Set(glw.Vec4(glw.RGBA(colornames.DeepOrangeA200)))

	ar := float32(w.size.X) / float32(w.size.Y)
	w.Proj.Ortho(-ar, ar, -1, 1, 1.0, 10.0)
	w.ctx.Viewport(0, 0, w.size.X, w.size.Y)
}

func (w *GLWidget) Paint() {
	w.ctx.Clear(gl.COLOR_BUFFER_BIT)
	w.Model.Step(now)
	w.Model.Update()
	w.Color.Step(now)
	w.Color.Update()
	w.VertexInd.Draw(gl.TRIANGLES)

	var e touch.Event
	e, w.lastTouch = w.lastTouch, touch.Event{}
	if e != (touch.Event{}) {
		w.OnInputEvent(e)
	}
}

func (w *GLWidget) InvCoords(ex, ey float32) (x, y float32) {
	return w.Proj.Inv2f(glw.Uton(ex/float32(w.size.X)), glw.Uton(1-(ey/float32(w.size.Y))))
}

func (w *GLWidget) OnInputEvent(ev interface{}) {
	switch ev := ev.(type) {
	case touch.Event:
		x, y := w.InvCoords(ev.X, ev.Y)
		w.Model.Stage(now, glw.TranslateTo(f32.Vec4{x, y, 0, 0}))
		g := float32(uint8(ev.X/10)) / 255
		w.Color.Stage(now, glw.TranslateTo(f32.Vec4{1, g, 0, 1}))
	case key.Event:
		if ev.Code == key.CodeEscape {
			os.Exit(0)
		}
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
