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
	"sync"
	"time"

	_ "image/jpeg"

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

	Proj  glw.U16fv
	Model glw.U16fv

	Vertex  Vertex
	Texture Texture

	animating uint32
	evg       gesture.Event
	state     state
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
			// TODO factor out frame buffer into type like widget.Sheet that contains a GLWidget
			w.buf.Create()
			w.buf.Attach()
		}

		w.prg.MustBuild(vsrc, fsrc)
		w.prg.SetLocations(w)
		w.prg.Use()
		w.Texture.Bind()
		w.Vertex.Bind()

		w.Texture.Upload(mustDecodeAsset("fancygopher.jpg"))

		w.Model.Animator(
			glw.Duration(250*time.Millisecond),
			glw.Notify(&w.animating),
		)
	case lifecycle.CrossOff:
		if w.useFrameBuffer {
			w.buf.Detach()
			w.buf.Delete()
		}
		w.Texture.Delete()
		w.Vertex.Delete()
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

	if size := w.Rect.Size(); size != w.MeasuredSize && size != image.ZP {
		ar := float32(size.X) / float32(size.Y)
		w.Proj.Ortho(-ar, ar, -1, 1, 1.0, 10.0)
		if w.useFrameBuffer {
			w.buf.Attach()
			w.buf.Update(size.X, size.Y)
		}
		w.ctx.Viewport(0, 0, size.X, size.Y)
		w.Mark(node.MarkNeedsPaintBase)
	}

	w.Model.Update()
}

func (w *GLWidget) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	now := time.Now()
	w.Marks.UnmarkNeedsPaintBase()
	if w.animating != 0 {
		w.Model.Step(now)
		w.Mark(node.MarkNeedsPaintBase)
	}

	if w.useFrameBuffer {
		w.buf.Attach()
	}

	w.ctx.Clear(gl.COLOR_BUFFER_BIT)

	w.Texture.Bind()
	w.Vertex.Bind()
	w.Vertex.Draw(gl.TRIANGLES)

	if w.useFrameBuffer {
		// TODO support for gomobile example
		draw.Draw(ctx.Dst, w.Rect.Add(origin), w.buf.RGBA(), image.ZP, draw.Over)
	}

	if w.evg != (gesture.Event{}) {
		x, y := cton(w.evg.CurrentPos.X, w.evg.CurrentPos.Y, w.Rect)
		x, y = w.Proj.Inv2f(x, y)
		x, y = w.Model.Inv2f(x, y)
		w.Model.Stage(now, glw.TranslateTo(f32.Vec4{x, y, 0, 0}))
		w.evg = gesture.Event{}
		w.Mark(node.MarkNeedsPaintBase)
	}

	if w.state.active() {
		w.Model.Stage(now, w.state.transforms()...)
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
		active := ev.Direction != key.DirRelease
		switch ev.Code {
		case key.CodeA:
			w.state.panLeft = active
		case key.CodeD:
			w.state.panRight = active
		case key.CodeW:
			w.state.panUp = active
		case key.CodeS:
			w.state.panDown = active
		case key.CodeQ:
			w.state.rotateLeft = active
		case key.CodeE:
			w.state.rotateRight = active
		case key.CodeZ:
			w.state.scaleUp = active
		case key.CodeX:
			w.state.scaleDown = active
		}
		w.Mark(node.MarkNeedsPaintBase)
		return node.Handled
	default:
		return node.NotHandled
	}
}

type state struct {
	panLeft, panRight, panUp, panDown bool
	rotateLeft, rotateRight           bool
	scaleUp, scaleDown                bool
}

func (t state) active() bool {
	return t.panLeft || t.panRight || t.panUp || t.panDown || t.rotateLeft || t.rotateRight || t.scaleUp || t.scaleDown
}

func (t state) transforms() []func(glw.Transformer) {
	const x = 0.1
	var p []func(glw.Transformer)
	if t.panLeft {
		p = append(p, glw.TranslateBy(f32.Vec4{-x, 0, 0, 0}))
	}
	if t.panRight {
		p = append(p, glw.TranslateBy(f32.Vec4{+x, 0, 0, 0}))
	}
	if t.panUp {
		p = append(p, glw.TranslateBy(f32.Vec4{0, -x, 0, 0}))
	}
	if t.panDown {
		p = append(p, glw.TranslateBy(f32.Vec4{0, +x, 0, 0}))
	}
	if t.rotateLeft {
		p = append(p, glw.RotateBy(-x, f32.Vec3{0, 0, 1}))
	}
	if t.rotateRight {
		p = append(p, glw.RotateBy(+x, f32.Vec3{0, 0, 1}))
	}
	if t.scaleUp {
		p = append(p, glw.ScaleBy(f32.Vec4{+x, +x, 0, 0}))
	}
	if t.scaleDown {
		p = append(p, glw.ScaleBy(f32.Vec4{-x, -x, 0, 0}))
	}
	return p
}

type Texture struct {
	glw.Texture
	Tex         glw.U1i
	Texcoord    glw.A2fv
	TexcoordBuf glw.FloatBuffer
	once        sync.Once
}

func (t *Texture) Bind() {
	t.once.Do(func() {
		t.Texture.Create()
		t.TexcoordBuf.Create(gl.STATIC_DRAW, []float32{
			-0, -0,
			-0, +1,
			+1, +1,
			+1, -0})
	})
	t.Texture.Bind()
	t.Tex.Set(int(t.Value - 1))
	t.TexcoordBuf.Bind()
	t.Texcoord.Pointer()
}

func (t *Texture) Delete() {
	t.Texture.Delete()
	t.TexcoordBuf.Delete()
	t.once = sync.Once{}
}

type Vertex struct {
	Vertex    glw.A3fv
	VertexBuf glw.FloatBuffer
	VertexInd glw.UintBuffer
	once      sync.Once
}

func (v *Vertex) Bind() {
	v.once.Do(func() {
		v.VertexBuf.Create(gl.STATIC_DRAW, []float32{
			-1, -1, 0,
			-1, +1, 0,
			+1, +1, 0,
			+1, -1, 0})
		v.VertexInd.Create(gl.STATIC_DRAW, []uint32{0, 1, 2, 0, 2, 3})
	})
	v.VertexBuf.Bind()
	v.VertexInd.Bind()
	v.Vertex.Pointer()
}

func (v *Vertex) Delete() {
	v.VertexInd.Delete()
	v.VertexBuf.Delete()
	v.once = sync.Once{}
}

func (v Vertex) Draw(mode gl.Enum) { v.VertexInd.Draw(mode) }

func mustDecodeAsset(name string) *image.RGBA {
	src, _, err := image.Decode(glw.MustOpen(name))
	if err != nil {
		panic(err)
	}
	if v, ok := src.(*image.RGBA); ok {
		return v
	}
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, src.Bounds(), src, image.ZP, draw.Src)
	return dst
}

const (
	vsrc = `#version 100
uniform mat4 proj;
uniform mat4 model;
attribute vec4 vertex;
attribute vec2 texcoord;
varying vec2 vtexcoord;
void main() {
  gl_Position = proj*model*vertex;
  vtexcoord = texcoord;
}`

	fsrc = `#version 100
precision mediump float;
uniform vec4 color;
uniform sampler2D tex;
varying vec2 vtexcoord;
void main() {
  //gl_FragColor = color;
  gl_FragColor = texture2D(tex, vtexcoord.xy);
}`
)
