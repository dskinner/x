package gesture

// https://android.googlesource.com/platform/frameworks/base/+/master/core/java/android/view/GestureDetector.java
// https://android.googlesource.com/platform/frameworks/base/+/master/core/java/android/view/ViewConfiguration.java

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/touch"
)

/*
how to promote; plus(+) means state of parent, not potential promotion.

touch
	- drag // or swipe or fling
  - longPress
    - longPressDrag
  - doubleTouch
    - doubleTouchDrag
  - dualTouch
    - dualDrag
    - dualLongPress
      - dualLongPressDrag
      + pinchOpen
      + pinchClosed
      + rotate
    - dualDoubleTouch
*/

type Type uint8

func (t Type) Has(x Type) bool { return t&x == x }
func (t Type) Any(x Type) bool { return t&x != 0 }

const (
	TypeBegin Type = 1 << iota
	TypeMove
	TypeEnd
	TypeFinal

	TypeInvalid Type = 0
)

func typeFor(t interface{}) Type {
	switch t {
	case touch.TypeBegin, mouse.DirPress:
		return TypeBegin
	case touch.TypeEnd, mouse.DirRelease, mouse.DirStep:
		return TypeEnd
	case touch.TypeMove, mouse.DirNone:
		return TypeMove
	default:
		panic(fmt.Errorf("no Type for %v", t))
	}
}

func cond(b, a Type) Type      { return (b << 4) | a }
func conds(c Type) (b, a Type) { return c >> 4, c & 0xf }

const (
	touchMargin = 20

	// duration-factor of 2 works better for mins, consider leaving max alone, maybe even doubleTouchMin alone too.
	dfac = 2

	longPressMin   = dfac * 100 * time.Millisecond
	longPressMax   = dfac * 500 * time.Millisecond
	doubleTouchMin = dfac * 40 * time.Millisecond  // between first up and second down
	doubleTouchMax = dfac * 300 * time.Millisecond // after second up
)

// ZE is the zero Event.
var ZE Event

type Event struct {
	X, Y float32
	Type Type
	Time time.Time
}

// withType returns a copy of Event with new Type set.
func (e Event) withType(t Type) Event { e.Type = t; return e }

// withTime returns a copy of Event with new Time set.
func (e Event) withTime(t time.Time) Event { e.Time = t; return e }

func (e Event) Final() bool { return e.Type.Has(TypeFinal) }

func (e Event) GoString() string {
	return fmt.Sprintf("%T{X:%2v Y:%2v Type:%-5v Time:%s}", e, e.X, e.Y, e.Type, e.Time.Format("15:04:05.000"))
}

type Events []Event

func (es Events) Last() Event {
	if len(es) == 0 {
		return ZE
	}
	return es[len(es)-1]
}

type handler interface {
	handle(Event) handler
	final() bool
	last() Event
	condense() handler
}

var (
	touchCond     = cond(TypeBegin, 0)
	dualTouchCond = cond(0, 0)

	// conditions for promoting Touch; assumes TypeMove events within touch-margin are discarded.
	longPressCond   = cond(TypeBegin, TypeBegin)
	dragCond        = cond(TypeMove, TypeBegin)
	doubleTouchCond = cond(TypeBegin, TypeEnd)

	// endCond   = cond(TypeEnd, TypeBegin)
	// finalCond = cond(TypeEnd, TypeEnd)

	finalCond = cond(TypeEnd, TypeBegin|TypeEnd)
)

// TODO endless events is nice for history, but no more than ?three? events are ever actually needed.
type Touch []Event

func (a Touch) final() bool { return len(a) != 0 && a[len(a)-1].Type.Has(TypeFinal) }
func (a Touch) last() Event { return a.Last() }
func (a Touch) condense() handler {
	if len(a) <= 3 {
		return a
	}
	return Touch{a[0], a[len(a)-2], a[len(a)-1]}
}

func (a Touch) Last() Event {
	if len(a) == 0 {
		return ZE
	}
	return a[len(a)-1]
}

// promote candidates are drag, longPress, doubleTouch, dualTouch
func (a Touch) handle(e Event) handler {
	dt := e.Time.Sub(a[0].Time)
	cn := cond(e.Type, a.Last().Type)
	switch {
	case longPressCond.Has(cn):
		if longPressMin <= dt && dt <= longPressMax {
			return LongPress(a)
		}
	case dragCond.Has(cn):
		if abs(a[0].X-e.X) > touchMargin || abs(a[0].Y-e.Y) > touchMargin {
			return Drag(append(a, e))
		}
	case doubleTouchCond.Has(cn):
		if doubleTouchMin <= dt && dt <= doubleTouchMax {
			return DoubleTouch(append(a, e))
		}
	case finalCond.Has(cn):
		// if dt <= longPressMin {
		// 	return append(a, e) // begin | end
		// }
		// a[1].Type = TypeFinal // end | final

		if dt <= longPressMin {
			return append(a, e)
		} else if len(a) == 1 {
			return append(a, e.withType(TypeFinal))
		} else {
			a[len(a)-1].Type = TypeFinal
			return a
		}
	}
	return a
}

// TODO or could consider
// type handler func(Event) handler
// func handleDrag(e Event) handler { ... }

type Drag []Event

func (a Drag) final() bool { return len(a) != 0 && a[len(a)-1].Type.Has(TypeFinal) }
func (a Drag) last() Event { return Touch(a).Last() }
func (a Drag) condense() handler {
	if len(a) <= 3 {
		return a
	}
	return Drag{a[0], a[len(a)-2], a[len(a)-1]}
}

func (a Drag) handle(e Event) handler {
	if e.Type.Has(TypeEnd) {
		e.Type = TypeFinal
	}
	if e.Type.Any(TypeMove | TypeFinal) {
		return append(a, e)
	}
	return nil
}

func (a Drag) GoString() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%T len(%v)", a, len(a))
	for _, e := range a {
		fmt.Fprintf(&buf, "\n%#v", e)
	}
	return buf.String()
}

type LongPress []Event

func (a LongPress) final() bool { return len(a) != 0 && a[len(a)-1].Type.Has(TypeFinal) }
func (a LongPress) last() Event { return Touch(a).Last() }
func (a LongPress) condense() handler {
	if len(a) <= 3 {
		return a
	}
	return LongPress{a[0], a[len(a)-2], a[len(a)-1]}
}

func (a LongPress) handle(e Event) handler {
	last := Touch(a).Last()
	switch e.Type {
	case TypeBegin:
		// last.Type must be TypeBegin
		return LongPressDrag(append(a, e))
	case TypeMove:
		if abs(last.X-e.X) > touchMargin || abs(last.Y-e.Y) > touchMargin {
			return LongPressDrag(append(a, e))
		}
	case TypeEnd: // TODO possibly not right
		return append(a, e.withType(TypeFinal))
	}
	return a // or nil ???
}

type LongPressDrag []Event

func (a LongPressDrag) final() bool { return len(a) != 0 && a[len(a)-1].Type.Has(TypeFinal) }
func (a LongPressDrag) last() Event { return Touch(a).Last() }
func (a LongPressDrag) condense() handler {
	if len(a) <= 3 {
		return a
	}
	return LongPressDrag{a[0], a[len(a)-2], a[len(a)-1]}
}

func (a LongPressDrag) handle(e Event) handler {
	if e.Type == TypeEnd {
		e.Type = TypeFinal
	}
	switch e.Type {
	case TypeMove, TypeFinal:
		a = append(a, e) // TODO like Drag? return here or return nil to cancel later ?due to stale events?
	}
	return a
}

func (a LongPressDrag) GoString() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%T len(%v)", a, len(a))
	for _, e := range a {
		fmt.Fprintf(&buf, "\n%#v", e)
	}
	return buf.String()
}

type DoubleTouch []Event

func (a DoubleTouch) final() bool { return len(a) != 0 && a[len(a)-1].Type.Has(TypeFinal) }
func (a DoubleTouch) last() Event { return Touch(a).Last() }
func (a DoubleTouch) condense() handler {
	if len(a) <= 3 {
		return a
	}
	return DoubleTouch{a[0], a[len(a)-2], a[len(a)-1]}
}

func (a DoubleTouch) handle(e Event) handler {
	// TODO could just call a[1].handle(b) and have things like
	// DoubleTouchDrag, DoubleTouchLongPress, DoubleTouchLongPressDrag
	// but then if receiving a new DoubleTouch, it'd be a TripleTouch
	// and so on and that might not be desireable.
	last := Touch(a).Last()
	dt := e.Time.Sub(last.Time)
	switch e.Type {
	case TypeMove:
		if dt > longPressMin {
			// TODO cancel makes a double-touch drag almost impossible to execute
			// as there's no leniency in time for a user to start drag after second down.

			// material doesn't define double-touch long-press; cancel event.
			// return nil

			// TODO combine if-condition with below if this works out
			return DoubleTouchDrag(append(a, e))
		}
		if abs(last.X-e.X) > touchMargin || abs(last.Y-e.Y) > touchMargin {
			return DoubleTouchDrag(append(a, e))
		}
	case TypeEnd:
		if dt <= longPressMin {
			return append(a, e.withType(TypeFinal)) // TODO yuck, really ???
		}
	}
	return a
}

type DoubleTouchDrag []Event

func (a DoubleTouchDrag) final() bool { return len(a) != 0 && a[len(a)-1].Type.Has(TypeFinal) }
func (a DoubleTouchDrag) last() Event { return Touch(a).Last() }
func (a DoubleTouchDrag) condense() handler {
	if len(a) <= 3 {
		return a
	}
	return DoubleTouchDrag{a[0], a[len(a)-2], a[len(a)-1]}
}

func (a DoubleTouchDrag) handle(e Event) handler {
	if e.Type == TypeEnd {
		e.Type = TypeFinal
	}

	switch e.Type {
	case TypeMove, TypeEnd, TypeFinal:
		a = append(a, e) // TODO like Drag ???
	}
	return a
}

// type DualTouch struct {
// 	A, B Touch
// }

var now = time.Now

var sendAfter = func(send func(interface{}), dur time.Duration, e interface{}) {
	go func() {
		time.Sleep(dur)
		send(e)
	}()
}

type touchTimeout struct {
	event Event
}

// TODO embed struct with for default timeouts; if zero-struct, autoinit values during Filter.
type EventFilter struct {
	Send func(interface{})

	tracking handler
	last     Event
}

func (f *EventFilter) Filter(e interface{}) interface{} {
	var t Event
	switch e := e.(type) {
	case mouse.Event:
		t = Event{X: e.X, Y: e.Y, Time: now(), Type: typeFor(e.Direction)}
	case touch.Event:
		t = Event{X: e.X, Y: e.Y, Time: now(), Type: typeFor(e.Type)}
	case touchTimeout:
		if f.tracking == nil {
			return e
		}
		if g, ok := f.tracking.(Touch); ok && g.Last().Type == e.event.Type {
			t = e.event.withTime(now())
		}
	}

	if t == ZE {
		return e
	}

	if f.tracking == nil {
		// assure begin; is stale event of cancelled gesture if tracking is nil and otherwise.
		if t.Type == TypeBegin {
			f.tracking = Touch{t}
			sendAfter(f.Send, longPressMin, touchTimeout{t})
		}
		f.Send(f.tracking)
		return e
	}

	// TODO this is where the work happens, handle is called which
	// returns another handler with different methods and so on.
	// Given a completed []Event, consider writing each Event.Type bits
	// from right to left into an ever extending int.
	// Consider if extreneous Events are dropped making path to gesture
	// deterministic, and that ever extending int a key.
	// In that case, instead of the whackery below, we write t.Event.Type
	// to the int, check state and/or cancel or append t to history
	// and/or finalize etc etc etc
	f.tracking = f.tracking.handle(t)

	if f.tracking == nil {
		return e // gesture was cancelled
	}

	if f.tracking.final() {
		f.Send(f.tracking.condense())
		f.tracking = nil
		return e
	}

	if g, ok := f.tracking.(Touch); ok && g.Last().Type == TypeEnd {
		// sendAfter(f.Send, doubleTouchMax, touchTimeout{t}) // doubleTouchMin
		sendAfter(f.Send, doubleTouchMin, touchTimeout{t})
	}

	if last := f.tracking.last(); last != f.last {
		f.Send(f.tracking.condense())
		f.last = last
	}

	return e
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	} else if x == 0 {
		return 0
	}
	return x
}
