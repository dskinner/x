// +build shiny gomobile

// Glwidget app for shiny and gomobile.
//
// Build tags are done so to allow running gomobile on desktop for the sake of Windows.
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

	_ "image/png"

	"dasa.cc/x/glw"
	"dasa.cc/x/glw/gesture"
	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/math/f32"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/gl"
)

const (
	fast    = 250 * time.Millisecond
	medium  = 350 * time.Millisecond
	slow    = 750 * time.Millisecond
	tedious = 2500 * time.Millisecond
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

	Color  glw.U4fv
	Radius glw.U1f

	Vertex  Vertex
	Texture Texture

	animating uint32
	isCircle  bool

	ge interface{}
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

		w.prg.MustBuildAssets("env-vert.glsl", "env-frag.glsl")
		w.prg.Unmarshal(w)
		w.prg.Use()
		w.Texture.Bind()
		w.Vertex.Bind()

		w.Texture.Upload(mustDecodeAsset("fancygopher.png"))

		w.Radius.Animator(glw.Duration(medium))
		w.Model.Animator(glw.Duration(medium))
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
		w.Proj.Ortho(-ar, ar, -1, 1, 0.0, 10.0)
		if w.useFrameBuffer {
			w.buf.Attach()
			w.buf.Update(size.X, size.Y)
		}
		w.ctx.Viewport(0, 0, size.X, size.Y)
		w.Mark(node.MarkNeedsPaintBase)
	}

	w.Model.Update()
}

func (w *GLWidget) TranslateTo(now time.Time, a gesture.Event) {
	x, y := cton(a.X, a.Y, w.Rect)
	x, y = w.Proj.Inv2f(x, y)
	w.Model.Stage(now, glw.TranslateTo(f32.Vec4{x, y, 0, 1}))
}

func (w *GLWidget) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	now := time.Now()
	w.Marks.UnmarkNeedsPaintBase()

	if w.Model.Step(now) {
		w.Mark(node.MarkNeedsPaintBase)
	}
	if w.Radius.Step(now) {
		w.Mark(node.MarkNeedsPaintBase)
	}

	if w.useFrameBuffer {
		w.buf.Attach()
	}

	w.ctx.Clear(gl.COLOR_BUFFER_BIT)

	w.Color.Set(glw.Vec4(glw.RGBA(theme.Default.Palette.Accent())))

	w.Texture.Bind()
	w.Vertex.Bind()
	w.Vertex.Draw(gl.TRIANGLES)

	if w.useFrameBuffer {
		// TODO support for gomobile example
		draw.Draw(ctx.Dst, w.Rect.Add(origin), w.buf.RGBA(), image.ZP, draw.Over)
	}

	var ge interface{}
	if ge, w.ge = w.ge, nil; ge != nil {
		switch e := ge.(type) {
		case gesture.Touch:
			if last := e[len(e)-1]; last.Type.Has(gesture.TypeFinal) {
				w.TranslateTo(now, last)
			}
		case gesture.Drag:
			w.TranslateTo(now, e[len(e)-1])
		case gesture.LongPress:
			if e[len(e)-1].Type.Has(gesture.TypeFinal) {
				var x float32
				if w.isCircle = !w.isCircle; w.isCircle {
					x = 1
				}
				w.Radius.Stage(now, glw.TranslateTo(f32.Vec4{x, 0, 0, 0}))
			}
		case gesture.LongPressDrag:
			ev0, ev1 := e[len(e)-1], e[len(e)-2]

			// x0, _ := cton(ev0.X, ev0.Y, w.Rect)
			// x1, _ := cton(ev1.X, ev1.Y, w.Rect)
			// angle := 10 * ((1 + x1) - (1 + x0))
			// w.Model.Stage(now, glw.RotateBy(-angle, f32.Vec3{0, 0, 1}))

			x0, y0 := cton(ev0.X, ev0.Y, w.Rect)
			x1, y1 := cton(ev1.X, ev1.Y, w.Rect)
			w.Model.Stage(now, glw.ShearBy(f32.Vec4{5 * (x0 - x1), 5 * (y0 - y1), 0, 1}))
		case gesture.DoubleTouch:
			if e[len(e)-1].Type.Has(gesture.TypeFinal) {
				w.Model.Stage(now, glw.RotateBy(-10, f32.Vec3{0, 0, 1}))
			}
		case gesture.DoubleTouchDrag:
			ev0, ev1 := e[len(e)-1], e[len(e)-2]
			x0, y0 := cton(ev0.X, ev0.Y, w.Rect)
			x0, y0 = w.Proj.Inv2f(x0, y0)
			x1, y1 := cton(ev1.X, ev1.Y, w.Rect)
			x1, y1 = w.Proj.Inv2f(x1, y1)
			dx := x0 - x1
			dy := y0 - y1
			w.Model.Stage(now, glw.ScaleBy(f32.Vec4{1 + 2*dx, 1 + 2*dy, 1, 1}))
		}
		w.Mark(node.MarkNeedsPaintBase)
	}

	return nil
}

func (w *GLWidget) OnInputEvent(ev interface{}, origin image.Point) node.EventHandled {
	switch ev := ev.(type) {
	case gesture.Touch, gesture.Drag, gesture.LongPress, gesture.LongPressDrag, gesture.DoubleTouch, gesture.DoubleTouchDrag:
		w.ge = ev
		w.Mark(node.MarkNeedsPaintBase)
		return node.Handled
	case key.Event:
		if ev.Code == key.CodeEscape {
			os.Exit(0)
			return node.Handled
		}
		return node.NotHandled
	default:
		return node.NotHandled
	}
}

type Texture struct {
	glw.Texture
	Content         glw.U1i
	ContentCoord    glw.A2fv
	ContentCoordBuf glw.FloatBuffer
	once            sync.Once
}

func (t *Texture) Bind() {
	t.once.Do(func() {
		t.Texture.Create()
		t.ContentCoordBuf.Create(gl.STATIC_DRAW, []float32{
			-0, -0,
			-0, +1,
			+1, +1,
			+1, -0})
	})
	t.Texture.Bind()
	t.Content.Set(int(t.Value - 1))
	t.ContentCoordBuf.Bind()
	t.ContentCoord.Pointer()
}

func (t *Texture) Delete() {
	t.Texture.Delete()
	t.ContentCoordBuf.Delete()
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
