package main

import (
	"math"
	"math/rand"

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
attribute vec3 vert;

varying vec3 pos;

void main() {
	gl_Position = proj * model * vec4(vert.x, vert.y, vert.z, 1.0);
	pos = gl_Position.xyz;
}`

const fsrc = `#version 100
precision mediump float;

varying vec3 pos;

void main() {
	gl_FragColor = vec4(1.0-pos.x, pos.y, 1.0-pos.z, 0.8);
}`

type Chaos struct {
	prg   glw.Program
	Proj  glw.U16fv
	Model glw.U16fv
	Vert  glw.VertexArray
}

// func (chs *Chaos) Create(ctx gl.Context) {
// 	chs.prg.MustBuild(vsrc, fsrc)
// 	chs.prg.Unmarshal(chs)
// 	chs.prg.Use()

// 	const (
// 		r = math.Pi / 180
// 		m = 0.5
// 	)
// 	rot := func(angle float64) (float32, float32) {
// 		c, s := math.Cos(r*angle), math.Sin(r*angle)
// 		return float32(m * c), float32(m * s)
// 	}

// 	x1, y1 := rot(90 - 72)
// 	x2, y2 := rot(18 - 72)

// 	pnt := []f32.Vec2{
// 		{0, m},
// 		{x1, y1},
// 		{x2, y2},
// 		{-x2, y2},
// 		{-x1, y1},
// 	}
// 	pts := []f32.Vec2{
// 		{0, m},
// 		{x1, y1},
// 		{x2, y2},
// 		{-x2, y2},
// 		{-x1, y1},
// 	}

// 	midpoint := func(a, b f32.Vec2) f32.Vec2 {
// 		return f32.Vec2{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2}
// 	}

// 	midpoint3fv := func(a, b f32.Vec3) f32.Vec3 {
// 		return f32.Vec3{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2, (a[2] + b[2]) / 2}
// 	}

// 	game := func(sx, sy float32, n int) {
// 		at := f32.Vec2{sx, sy}
// 		pts = append(pts, at)
// 		for i := 0; i < n; i++ {
// 			at = midpoint(at, pnt[rand.Intn(len(pnt))])
// 			pts = append(pts, at)
// 		}
// 	}
// 	game(0, 0, 20000)

// 	const nverts = 3
// 	verts := make([]float32, len(pts)*nverts)
// 	for i, pt := range pts {
// 		verts[nverts*i] = pt[0]
// 		verts[nverts*i+1] = pt[1]
// 		verts[nverts*i+2] = (float32(rand.Intn(100)) / float32(100)) - 0.5
// 	}

// 	// indices := make([]uint32, len(verts))
// 	// for i := range indices {
// 	// 	indices[i] = uint32((i + 1) / 2)
// 	// }
// 	// indices[len(indices)-1] = indices[len(indices)-2]

// 	chs.Vert.Floats.Create(gl.STREAM_DRAW, verts)
// 	// chs.Vert.Uints.Create(gl.STATIC_DRAW, indices)
// 	chs.Vert.StepSize(nverts, 0, 0)
// 	chs.Vert.Bind()

// 	chs.Model.Transform(glw.TranslateTo(f32.Vec3{0, 0, -1}))

// 	ctx.Enable(gl.BLEND)
// 	ctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
// }

func (chs *Chaos) Create(ctx gl.Context) {
	chs.prg.MustBuild(vsrc, fsrc)
	chs.prg.Unmarshal(chs)
	chs.prg.Use()

	const (
		r = math.Pi / 180
		m = 0.5
	)
	rot := func(angle float64) (float32, float32) {
		c, s := math.Cos(r*angle), math.Sin(r*angle)
		return float32(m * c), float32(m * s)
	}

	x1, y1 := rot(90 - 72)
	x2, y2 := rot(18 - 72)

	pnt := []f32.Vec3{
		{0, m, 0},
		{x1, y1, 0},
		{x2, y2, 0},
		{-x2, y2, 0},
		{-x1, y1, 0},

		{0, m, -0.1},
		{x1, y1, -0.1},
		{x2, y2, -0.1},
		{-x2, y2, -0.1},
		{-x1, y1, -0.1},
	}
	pts := []f32.Vec3{
		{0, m},
		{x1, y1},
		{x2, y2},
		{-x2, y2},
		{-x1, y1},
	}

	midpoint := func(a, b f32.Vec3) f32.Vec3 {
		return f32.Vec3{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2, (a[2] + b[2]) / 2}
	}

	game := func(sx, sy float32, n int) {
		at := f32.Vec3{sx, sy}
		pts = append(pts, at)
		for i := 0; i < n; i++ {
			at = midpoint(at, pnt[rand.Intn(len(pnt))])
			pts = append(pts, at)
		}
	}
	game(0, 0, 20000)

	const nverts = 3
	verts := make([]float32, len(pts)*nverts)
	for i, pt := range pts {
		verts[nverts*i] = pt[0]
		verts[nverts*i+1] = pt[1]
		verts[nverts*i+2] = pt[2]
		// verts[nverts*i+2] = (float32(rand.Intn(100)) / float32(100)) - 0.5
	}

	// indices := make([]uint32, len(verts))
	// for i := range indices {
	// 	indices[i] = uint32((i + 1) / 2)
	// }
	// indices[len(indices)-1] = indices[len(indices)-2]

	chs.Vert.Floats.Create(gl.STREAM_DRAW, verts)
	// chs.Vert.Uints.Create(gl.STATIC_DRAW, indices)
	chs.Vert.StepSize(nverts, 0, 0)
	chs.Vert.Bind()

	chs.Model.Transform(glw.TranslateTo(f32.Vec3{0, 0, -1}))

	ctx.Enable(gl.BLEND)
	ctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
}

func (chs *Chaos) Layout(ev size.Event) {
	if ev.HeightPx != 0 {
		ar := float32(ev.WidthPx) / float32(ev.HeightPx)
		chs.Proj.Ortho(-ar, ar, -1, 1, 0.1, 10.0)
	}
}

func (chs *Chaos) Draw(ctx gl.Context) {
	ctx.Clear(gl.COLOR_BUFFER_BIT)
	chs.Model.Transform(glw.RotateBy(1, f32.Vec3{1, 0, 0}))
	chs.Model.Update()
	chs.Vert.Bind()
	chs.Vert.Draw(gl.POINTS)
}

func (chs *Chaos) Delete() {
	chs.Vert.Delete()
}

func main() {
	app.Main(func(a app.App) {
		var glctx gl.Context
		chs := new(Chaos)

		gef := gesture.EventFilter{}
		gef.Send = func(e interface{}) {
			if e = gef.Filter(e); e == nil {
				return
			}
			switch e := e.(type) {
			case gesture.DoubleTouch:
				_ = e
			}
		}

		for ev := range a.Events() {
			switch ev := a.Filter(ev).(type) {
			case lifecycle.Event:
				switch ev.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glctx = glw.With(ev.DrawContext.(gl.Context3))
					chs.Create(glctx)
				case lifecycle.CrossOff:
					chs.Delete()
					glctx = glw.With(nil)
				}
			case size.Event:
				if glctx == nil {
					a.Send(ev)
				} else {
					chs.Layout(ev)
					glctx.Viewport(0, 0, ev.WidthPx, ev.HeightPx)
				}
			case paint.Event:
				if glctx != nil {
					chs.Draw(glctx)
					a.Publish()
					a.Send(paint.Event{})
				}
			case touch.Event:
				gef.Send(ev)
			}
		}
	})
}
