package main

import (
	"time"

	"dasa.cc/x/glw"
	"dasa.cc/x/glw/gesture"
	"golang.org/x/image/math/f32"
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"
)

const vsrc = `#version 100
uniform mat4 proj;
uniform mat4 model;
attribute vec2 vert;

void main() {
	gl_Position = proj * model * vec4(vert.x, vert.y, -1.0f, 1.0f);
}`

const fsrc = `#version 100
precision mediump float;

void main() {
	gl_FragColor = vec4(1.0f, 0.5f, 0.2f, 1.0f);
}`

type Shape interface {
	Create()
	Draw(time.Time)
	Delete()
}

type Env struct {
	prg  glw.Program
	Proj glw.U16fv

	shapes []Shape
	index  int
}

func (env *Env) Create() {
	env.prg.MustBuild(vsrc, fsrc)
	env.prg.Unmarshal(env)
	env.prg.Use()
	for _, shape := range env.shapes {
		env.prg.Unmarshal(shape)
		shape.Create()
	}
}

func (env *Env) Layout(ev size.Event) {
	if ev.HeightPx != 0 {
		ar := float32(ev.WidthPx) / float32(ev.HeightPx)
		env.Proj.Ortho(-ar, ar, -1, 1, 0.1, 10.0)
	}
}

func (env *Env) Draw(now time.Time) {
	env.shapes[env.index].Draw(now)
}

func (env *Env) Delete() {
	for _, shape := range env.shapes {
		shape.Delete()
	}
	env.prg.Delete()
}

type Triangle struct {
	Model glw.U16fv
	Vert  glw.VertexArray
}

func (tri *Triangle) Create() {
	tri.Vert.Floats.Create(gl.STATIC_DRAW, []float32{
		-0.5, -0.5,
		+0.5, -0.5,
		+0.0, +0.5,
	})
	tri.Vert.StepSize(2, 0, 0)
	tri.Vert.Bind()

	tri.Model.Animator(glw.Duration(1 * time.Second))
}

func (tri *Triangle) Draw(now time.Time) {
	if !tri.Model.Step(now) {
		tri.Model.Transform(glw.RotateBy(180, f32.Vec3{0, 0, 1}))
	}
	tri.Model.Update()
	tri.Vert.Bind()
	tri.Vert.Draw(gl.TRIANGLES)
}

func (tri *Triangle) Delete() {
	tri.Vert.Delete()
}

type Square struct {
	Model glw.U16fv
	Vert  glw.VertexElement
}

func (sqr *Square) Create() {
	sqr.Vert.Floats.Create(gl.STATIC_DRAW, []float32{
		-0.5, -0.5,
		+0.5, -0.5,
		+0.5, +0.5,
		-0.5, +0.5,
	})
	sqr.Vert.Uints.Create(gl.STATIC_DRAW, []uint32{0, 1, 2, 0, 2, 3})
	sqr.Vert.StepSize(2, 0, 0)
	sqr.Vert.Bind()
}

func (sqr *Square) Draw(now time.Time) {
	sqr.Model.Step(now)
	sqr.Model.Update()
	sqr.Vert.Bind()
	sqr.Vert.Draw(gl.TRIANGLES)
}

func (sqr *Square) Delete() {
	sqr.Vert.Delete()
}

func main() {
	app.Main(func(a app.App) {
		var glctx gl.Context
		env := &Env{shapes: []Shape{new(Triangle), new(Square)}}

		gef := gesture.EventFilter{}
		gef.Send = func(e interface{}) {
			if e = gef.Filter(e); e == nil {
				return
			}
			switch e := e.(type) {
			case gesture.DoubleTouch:
				if e[len(e)-1].Final() {
					env.index = (env.index + 1) % len(env.shapes)
				}
			}
		}

		for ev := range a.Events() {
			switch ev := a.Filter(ev).(type) {
			case lifecycle.Event:
				switch ev.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glctx = glw.With(ev.DrawContext.(gl.Context))
					env.Create()
				case lifecycle.CrossOff:
					env.Delete()
					glctx = glw.With(nil)
				}
			case size.Event:
				if glctx == nil {
					a.Send(ev)
				} else {
					env.Layout(ev)
					glctx.Viewport(0, 0, ev.WidthPx, ev.HeightPx)
				}
			case paint.Event:
				if glctx != nil {
					now := time.Now()
					glctx.Clear(gl.COLOR_BUFFER_BIT)
					env.Draw(now)
					a.Publish()
					a.Send(paint.Event{})
				}
			case touch.Event:
				gef.Send(ev)
			}
		}
	})
}
