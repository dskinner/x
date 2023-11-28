package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"dasa.cc/x/glw"
	"dasa.cc/x/glw/gesture"
	"golang.org/x/image/math/f32"
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/gl"
)

// https://github.com/dskinner/x/blob/e7bfeb25a399a8fbb2b342636f4b142bc141d04c/glw/example/glwidget/glwidget.go

const vsrc = `#version 100
uniform mat4 proj;
uniform mat4 model;
attribute vec2 vert;

void main() {
	gl_Position = proj * model * vec4(vert.x, vert.y, 0.0, 1.0);
}`

const fsrc = `#version 100
precision mediump float;

void main() {
	gl_FragColor = vec4(1.0, 0.5, 0.2, 1.0);
}`

func export(path string, img *image.RGBA) {
	out, err := os.Create(path)
	defer out.Close()
	if err != nil {
		fmt.Printf("Failed to create %s: %v\n", path, err)
	}
	png.Encode(out, img.SubImage(img.Rect))
}

type Shape interface {
	Create()
	Draw(time.Time)
	Delete()
	Gesture(any)
}

var env = &Env{
	shapes:         []Shape{new(Triangle), new(Square), new(Pentagon)},
	index:          1,
	useFrameBuffer: true,
}

type Env struct {
	glw.Program
	Proj glw.U16fv
	Rect image.Rectangle

	useFrameBuffer bool
	buf            glw.FrameBuffer

	shapes []Shape
	index  int

	ge any
}

func (env *Env) Create(ctx gl.Context3) {
	if env.useFrameBuffer {
		env.buf.Create()
		env.buf.Attach()
	}

	env.MustBuild(vsrc, fsrc)
	env.Unmarshal(env)
	env.Use()

	for _, shape := range env.shapes {
		env.Unmarshal(shape)
		shape.Create()
	}

	ctx.LineWidth(4)
}

func (env *Env) Layout(ctx gl.Context3, ev size.Event) {
	if ev.HeightPx != 0 {
		ar := float32(ev.WidthPx) / float32(ev.HeightPx)
		env.Proj.Ortho(-ar, ar, -1, 1, 0.1, 10.0)
		if env.useFrameBuffer {
			env.buf.Attach()
			env.buf.Update(ev.WidthPx, ev.HeightPx)
		}
		ctx.Viewport(0, 0, ev.WidthPx, ev.HeightPx)
		env.Rect = ev.Bounds()
	}
}

func (env *Env) Draw(ctx gl.Context3) {
	now := time.Now()
	if env.useFrameBuffer {
		env.buf.Attach()
	}
	ctx.Clear(gl.COLOR_BUFFER_BIT)
	env.Use()
	env.shapes[env.index].Draw(now)

	if env.useFrameBuffer {
		env.buf.Blit(env.Rect.Size())
	}

	var ge any
	env.ge, ge = nil, env.ge
	if ge != nil {
		env.Gesture(ge)
	}
}

func (env *Env) Delete() {
	for _, shape := range env.shapes {
		shape.Delete()
	}
	if env.useFrameBuffer {
		env.buf.Detach()
		env.buf.Delete()
	}
	env.Program.Delete()
}

func (env *Env) Gesture(e any) {
	switch e := e.(type) {
	case gesture.DoubleTouch:
		if e.Final() {
			// export("export.png", env.buf.RGBA())
			env.index = (env.index + 1) % len(env.shapes)
		}
	default:
		env.shapes[env.index].Gesture(e)
	}
}

func (env *Env) Unproject(x, y float32) (float32, float32) {
	nx, ny := glw.Uton(x/float32(env.Rect.Dx())), glw.Uton(1-y/float32(env.Rect.Dy()))
	return env.Proj.Inv2f(nx, ny)
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

	tri.Model.Transform(glw.TranslateTo(f32.Vec3{0, 0, -1}))
}

func (tri *Triangle) Draw(now time.Time) {
	tri.Model.Transform(glw.RotateBy(1, f32.Vec3{0, 0, 1}))
	tri.Model.Update()
	tri.Vert.Bind()
	tri.Vert.Draw(gl.TRIANGLES)
}

func (tri *Triangle) Delete() {
	tri.Vert.Delete()
}

func (tri *Triangle) Gesture(e any) {}

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

	sqr.Model.Transform(glw.TranslateTo(f32.Vec3{0, 0, -1}))
}

func (sqr *Square) Draw(now time.Time) {
	if !sqr.Model.Step(now) {
		sqr.Model.Stage(now, glw.RotateBy(90, f32.Vec3{0, 0, 1}))
	}
	sqr.Model.Update()
	sqr.Vert.Bind()
	sqr.Vert.Draw(gl.TRIANGLES)
}

func (sqr *Square) Delete() {
	sqr.Vert.Delete()
}

func (sqr *Square) Gesture(e any) {
	switch e := e.(type) {
	case gesture.LongPress:
		if e.Final() {
			sqr.Model.Stage(time.Now(), glw.RotateBy(90, f32.Vec3{0, 0, 1}))
		}
	case gesture.Touch:
		sqr.TranslateTo(e.Last().X, e.Last().Y)
	case gesture.Drag:
		sqr.TranslateTo(e.Last().X, e.Last().Y)
	case gesture.LongPressDrag:
		sqr.TranslateTo(e.Last().X, e.Last().Y)
	}
}

func (sqr *Square) TranslateTo(x, y float32) {
	x, y = env.Unproject(x, y)
	sqr.Model.Stage(time.Now(), glw.TranslateTo(f32.Vec3{x, y, -1}))
}

type Pentagon struct {
	Model glw.U16fv
	Vert  glw.VertexElement
}

func (pnt *Pentagon) Create() {
	pnt.Vert.Floats.Create(gl.STATIC_DRAW, []float32{
		+0.000, +0.500,
		+0.475, +0.154,
		+0.293, -0.404,
		-0.293, -0.404,
		-0.475, +0.154,
	})
	pnt.Vert.Uints.Create(gl.STATIC_DRAW, []uint32{0, 1, 1, 2, 2, 3, 3, 4, 4, 0})
	pnt.Vert.StepSize(2, 0, 0)
	pnt.Vert.Bind()
	pnt.Model.Transform(glw.TranslateTo(f32.Vec3{0, 0, -1}))
}

func (pnt *Pentagon) Draw(now time.Time) {
	pnt.Model.Update()
	pnt.Vert.Bind()
	pnt.Vert.Draw(gl.LINES)
}

func (pnt *Pentagon) Delete() {
	pnt.Vert.Delete()
}

func (pnt *Pentagon) Gesture(e any) {}

func main() {
	app.Main(func(a app.App) {
		var glctx gl.Context3
		gef := gesture.EventFilter{Send: a.Send}
		// a.RegisterFilter(gef.Filter)

		for e := range a.Events() {
			if e = gef.Filter(e); e == nil {
				continue
			}
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glctx = glw.With(e.DrawContext.(gl.Context3))
					env.Create(glctx)
				case lifecycle.CrossOff:
					env.Delete()
					glctx = glw.With(nil)
				}
			case size.Event:
				if glctx == nil {
					a.Send(e)
				} else {
					env.Layout(glctx, e)
				}
			case paint.Event:
				if glctx != nil {
					env.Draw(glctx)
					a.Publish()
					a.Send(paint.Event{})
				}
			case gesture.Touch, gesture.Drag, gesture.LongPress, gesture.LongPressDrag, gesture.DoubleTouch, gesture.DoubleTouchDrag:
				env.ge = e
			}
		}
	})
}
