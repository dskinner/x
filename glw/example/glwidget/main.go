// +build ignore

package main

import (
	"image"
	"image/draw"
	"log"
	"os"
	"sync"
	"time"

	"dasa.cc/x/glw"

	"golang.org/x/exp/shiny/driver/gldriver"
	"golang.org/x/exp/shiny/gesture"
	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget"
	"golang.org/x/exp/shiny/widget/flex"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/math/f32"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/gl"
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("glwidget: ")
}

func main() {
	gldriver.Main(func(s screen.Screen) {
		root := widget.NewSheet(
			flex.NewFlex(
				widget.WithLayoutData(NewGLWidget(), flex.LayoutData{Grow: 1, Align: flex.AlignItemStretch}),
			),
		)

		opts := widget.RunWindowOptions{}
		opts.NewWindowOptions.Title = "GLWidget"
		opts.Theme.Palette = &theme.Palette{
			theme.Light:      image.Uniform{colornames.BlueGrey100},
			theme.Neutral:    image.Uniform{colornames.BlueGrey500},
			theme.Dark:       image.Uniform{colornames.BlueGrey900},
			theme.Accent:     image.Uniform{colornames.DeepOrangeA200},
			theme.Foreground: image.Uniform{colornames.Black},
			theme.Background: image.Uniform{colornames.BlueGrey500},
		}

		if err := RunWindow(s, root, &opts); err != nil {
			log.Fatal(err)
		}
	})
}

var (
	now       = time.Now()
	lastpaint = now
)

type GLWidget struct {
	node.LeafEmbed
	ctx gl.Context
	buf glw.FrameBuffer

	prg glw.Program

	Proj      glw.U16fv
	Model     glw.U16fv
	Color     glw.U4fv
	Vertex    glw.A3fv
	VertexBuf glw.FloatBuffer
	VertexInd glw.UintBuffer

	animating uint32
	egesture  gesture.Event
}

func NewGLWidget() *GLWidget {
	ctx, err := gldriver.NewContext()
	if err != nil {
		panic(err)
	}
	w := &GLWidget{ctx: glw.With(ctx)}
	w.Wrapper = w
	w.buf.Create()
	w.buf.Attach(0, 0)
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

	return w
}

func (w *GLWidget) OnLifecycleEvent(e lifecycle.Event) {
	if e.To == lifecycle.StageDead {
		w.buf.Detach()
		w.buf.Delete()
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
		w.buf.Update(size.X, size.Y)
		w.ctx.Viewport(0, 0, size.X, size.Y)
	}

	w.Mark(node.MarkNeedsPaintBase)
}

func (w *GLWidget) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()

	// handle paint
	now = time.Now()

	w.ctx.Clear(gl.COLOR_BUFFER_BIT)
	w.Model.Step(now)
	w.Model.Update()
	w.Color.Step(now)
	w.Color.Update()
	w.VertexInd.Draw(gl.TRIANGLES)
	draw.Draw(ctx.Dst, w.Rect.Add(origin), w.buf.RGBA(), image.ZP, draw.Over)

	lastpaint = now

	// handle gesture
	if w.animating != 0 { // check before Stage in-case last frame
		w.Mark(node.MarkNeedsPaintBase)
	}

	if w.egesture != (gesture.Event{}) {
		var ev gesture.Event
		ev, w.egesture = w.egesture, gesture.Event{}
		x, y := w.InvCoords(ev.CurrentPos.X, ev.CurrentPos.Y)
		w.Model.Stage(now, glw.TranslateTo(f32.Vec4{x, y, 0, 0}))
		g := float32(uint8(ev.CurrentPos.X)) / 255
		w.Color.Stage(now, glw.TranslateTo(f32.Vec4{1, g, 0, 1}))
	}

	if w.animating != 0 { // check after Stage in-case first frame
		w.Mark(node.MarkNeedsPaintBase)
	}

	return nil
}

func (w *GLWidget) InvCoords(ex, ey float32) (x, y float32) {
	return w.Proj.Inv2f(glw.Uton(ex/float32(w.Rect.Dx())), glw.Uton(ey/float32(w.Rect.Dy())))
}

func (w *GLWidget) OnInputEvent(ev interface{}, origin image.Point) node.EventHandled {
	switch ev := ev.(type) {
	case gesture.Event:
		w.egesture = ev
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

// RunWindow mainly differs from widget.RunWindow in that event queue is exhausted
// in goroutine to provide only the latest size and paint events. This isn't done
// at great effort as these events only fire when queue is empty.
//
// Also propagates lifecycle events to root's first child and enables key events.
func RunWindow(s screen.Screen, root node.Node, opts *widget.RunWindowOptions) error {
	var (
		nwo *screen.NewWindowOptions
		t   *theme.Theme
	)
	if opts != nil {
		nwo = &opts.NewWindowOptions
		t = &opts.Theme
	}
	w, err := s.NewWindow(nwo)
	if err != nil {
		return err
	}
	defer w.Release()

	var que struct {
		sync.Mutex
		sync.Cond
		epaint *paint.Event
		es     []interface{}
	}
	que.L = &que.Mutex

	nextEvent := func() (e interface{}) {
		que.Lock()
		defer que.Unlock()
		for {
			if n := len(que.es); n > 0 {
				e, que.es = que.es[0], que.es[1:]
				return e
			}
			if que.epaint != nil {
				e, que.epaint = *que.epaint, nil
				return e
			}
			que.Wait()
		}
	}

	go func() {
		gef := gesture.EventFilter{EventDeque: w}
		for {
			e := gef.Filter(w.NextEvent())
			if e == nil {
				continue
			}
			que.Lock()
			switch e := e.(type) {
			case paint.Event:
				que.epaint = &e
			default:
				que.es = append(que.es, e)
			}
			que.Signal()
			que.Unlock()
		}
	}()

	var esize size.Event
	paintPending := false
	for {
		switch e := nextEvent().(type) {
		case lifecycle.Event:
			root.OnLifecycleEvent(e)
			if c := root.Wrappee().FirstChild; c != nil {
				c.Wrapper.OnLifecycleEvent(e)
			}
			if e.To == lifecycle.StageDead {
				return nil
			}

		case gesture.Event, mouse.Event, key.Event:
			root.OnInputEvent(e, image.Point{})

		case paint.Event:
			if esize != (size.Event{}) {
				var ev size.Event
				ev, esize = esize, size.Event{}
				if dpi := float64(ev.PixelsPerPt) * unit.PointsPerInch; dpi != t.GetDPI() {
					newT := new(theme.Theme)
					if t != nil {
						*newT = *t
					}
					newT.DPI = dpi
					t = newT
				}

				size := ev.Size()
				root.Measure(t, size.X, size.Y)
				root.Wrappee().Rect = ev.Bounds()
				root.Layout(t)
			}

			ctx := &node.PaintContext{
				Theme:  t,
				Screen: s,
				Drawer: w,
				Src2Dst: f64.Aff3{
					1, 0, 0,
					0, 1, 0,
				},
			}
			if err := root.Paint(ctx, image.Point{}); err != nil {
				return err
			}
			w.Publish()
			paintPending = false

		case size.Event:
			esize = e
			w.Send(paint.Event{})

		case error:
			return e
		}

		if !paintPending && root.Wrappee().Marks.NeedsPaint() {
			paintPending = true
			w.Send(paint.Event{})
		}
	}
}
