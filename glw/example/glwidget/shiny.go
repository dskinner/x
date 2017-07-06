// +build shiny,!gomobile

package main

import (
	"image"
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
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("glwidget: ")
}

func cton(x, y float32, r image.Rectangle) (nx, ny float32) {
	return glw.Uton(x / float32(r.Dx())), glw.Uton(y / float32(r.Dy()))
}

func main() {
	gldriver.Main(func(s screen.Screen) {
		ctx, err := gldriver.NewContext()
		if err != nil {
			panic(err)
		}
		root := widget.NewSheet(
			flex.NewFlex(
				widget.WithLayoutData(NewGLWidget(ctx, true), flex.LayoutData{Grow: 1, Align: flex.AlignItemStretch}),
			),
		)

		opts := widget.RunWindowOptions{}
		opts.NewWindowOptions.Title = "GLWidget"
		opts.Theme = *theme.Default

		if err := RunWindow(s, root, &opts); err != nil {
			log.Fatal(err)
		}
	})
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

	gef := gesture.EventFilter{EventDeque: w}
	for {
		e := w.NextEvent()

		if e = gef.Filter(e); e == nil {
			continue
		}

		switch e := e.(type) {
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
			if e.External {
				root.Mark(node.MarkNeedsPaint)
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
			// TODO: call Mark(node.MarkNeedsPaint)?

		case error:
			return e
		}

		if !paintPending && root.Wrappee().Marks.NeedsPaint() {
			paintPending = true
			w.Send(paint.Event{})
		}
	}
}
