package main

import (
	"image"
	"image/draw"
	"log"

	"dasa.cc/x/glw"
	"golang.org/x/exp/shiny/driver/gldriver"
	"golang.org/x/exp/shiny/gesture"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget"
	"golang.org/x/exp/shiny/widget/flex"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/gl"
)

var (
	glwidget *GLWidget
	deque    screen.EventDeque
)

type GLWidget struct {
	node.LeafEmbed
	fbuf   glw.FrameBuffer
	draw   func()
	resize func(int, int)
	input  func(interface{})
}

func NewGLWidget(ctx gl.Context, draw func(), resize func(int, int), input func(interface{})) *GLWidget {
	// TODO
	// ctx = glw.With(ctx)
	w := &GLWidget{draw: draw, resize: resize, input: input}
	w.Wrapper = w
	w.fbuf.Create()
	w.fbuf.Attach()
	return w
}

func (w *GLWidget) Measure(*theme.Theme, int, int) { w.MeasuredSize = w.Rect.Size() }

func (w *GLWidget) Layout(t *theme.Theme) {
	if size := w.Rect.Size(); size != w.MeasuredSize && size != image.ZP {
		w.resize(size.X, size.Y)
		w.fbuf.Attach()
		w.fbuf.Update(size.X, size.Y)
		w.Mark(node.MarkNeedsPaintBase)
	}
}

func (w *GLWidget) PaintBase(ctx *node.PaintBaseContext, origin image.Point) error {
	w.Marks.UnmarkNeedsPaintBase()
	w.fbuf.Attach()
	w.draw()
	draw.Draw(ctx.Dst, w.Rect.Add(origin), w.fbuf.RGBA(), image.ZP, draw.Over)
	return nil
}

func (w *GLWidget) OnInputEvent(ev interface{}, origin image.Point) node.EventHandled {
	w.input(ev)
	return node.Handled
}

func shinyMain(s screen.Screen) {
	ctx, err := gldriver.NewContext()
	if err != nil {
		panic(err)
	}
	glwidget = NewGLWidget(ctx, gldraw, glresize, glinput)
	glinit(ctx)
	root := widget.NewSheet(
		flex.NewFlex(
			widget.WithLayoutData(glwidget, flex.LayoutData{Grow: 1, Align: flex.AlignItemStretch}),
		),
	)

	opts := widget.RunWindowOptions{}
	opts.NewWindowOptions.Title = title
	opts.Theme = *theme.Default

	if err := RunWindow(s, root, &opts); err != nil {
		log.Fatal(err)
	}
}

// RunWindow differs from widget.RunWindow by propagating lifecycle events and
// forwarding key events to OnInputEvent.
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

	deque = w

	// paintPending batches up multiple NeedsPaint observations so that we
	// paint only once (which can be relatively expensive) even when there are
	// multiple input events in the queue, such as from a rapidly moving mouse
	// or from the user typing many keys.
	//
	// TODO: determine somehow if there's an external paint event in the queue,
	// not just internal paint events?
	//
	// TODO: if every package that uses package screen should basically
	// throttle like this, should it be provided at a lower level?
	paintPending := false

	// gef := gesture.EventFilter{EventDeque: w}
	for {
		e := w.NextEvent()

		// if e = gef.Filter(e); e == nil {
		// 	continue
		// }

		switch e := e.(type) {
		case lifecycle.Event:
			root.OnLifecycleEvent(e)

			// switch e.Crosses(lifecycle.StageVisible) {
			// case lifecycle.CrossOn:
			// 	glinit(e.DrawContext.(gl.Context))
			// case lifecycle.CrossOff:
			// 	gldestroy()
			// }

			if e.To == lifecycle.StageDead {
				return nil
			}

		// TODO reconsider doing this
		case gesture.Event, mouse.Event, key.Event:
			root.OnInputEvent(e, image.Point{})
		case paint.Event:
			if e.External {
				glwidget.Mark(node.MarkNeedsPaintBase)
				break
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
			if dpi := float64(e.PixelsPerPt) * unit.PointsPerInch; dpi != t.GetDPI() {
				newT := new(theme.Theme)
				if t != nil {
					*newT = *t
				}
				newT.DPI = dpi
				t = newT
			}

			size := e.Size()
			root.Measure(t, size.X, size.Y)
			root.Wrappee().Rect = e.Bounds()
			root.Layout(t)
		case error:
			return e
		}

		if !paintPending && root.Wrappee().Marks.NeedsPaint() {
			paintPending = true
			w.Send(paint.Event{})
		}
	}
}
