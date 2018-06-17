// +build gomobile,!shiny

package main

import (
	"image"
	"log"

	"dasa.cc/x/glw"
	"dasa.cc/x/glw/gesture"

	"golang.org/x/exp/shiny/unit"
	"golang.org/x/exp/shiny/widget/theme"
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("glwidget: ")
}

func cton(x, y float32, r image.Rectangle) (nx, ny float32) {
	return glw.Uton(x / float32(r.Dx())), glw.Uton(1 - y/float32(r.Dy()))
}

func main() {
	app.Main(func(a app.App) {
		t := theme.Default
		root := NewGLWidget(nil, false)
		gef := gesture.EventFilter{Send: a.Send}
		for e := range a.Events() {
			if e = gef.Filter(e); e == nil {
				continue
			}
			switch e := e.(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glctx, _ := e.DrawContext.(gl.Context)
					root.ctx = glw.With(glctx)
					a.Send(paint.Event{})
				}
				root.OnLifecycleEvent(e)
			case size.Event:
				if root.ctx == nil {
					a.Send(e)
					break
				}
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
			case paint.Event:
				if e.External || root.ctx == nil {
					continue
				}
				root.PaintBase(nil, image.ZP)
				a.Publish()
				a.Send(paint.Event{})
			case touch.Event, key.Event:
				root.OnInputEvent(e, image.ZP)
			case gesture.Touch, gesture.Drag, gesture.LongPress, gesture.LongPressDrag, gesture.DoubleTouch, gesture.DoubleTouchDrag:
				root.OnInputEvent(e, image.ZP)
			}
		}
	})
}
