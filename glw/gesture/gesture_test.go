package gesture

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/touch"
)

const ms = time.Millisecond

func init() {
	setTime(time.Now())
}

func (t Type) forMouse() mouse.Direction {
	switch t {
	case TypeBegin:
		return mouse.DirPress
	case TypeMove:
		return mouse.DirNone
	case TypeEnd, TypeFinal, TypeInvalid:
		return mouse.DirRelease
	default:
		panic(fmt.Errorf("unknown Type(%v)", int(t)))
	}
}

func (t Type) forTouch() touch.Type {
	switch t {
	case TypeBegin:
		return touch.TypeBegin
	case TypeMove:
		return touch.TypeMove
	case TypeEnd, TypeFinal, TypeInvalid:
		return touch.TypeEnd
	default:
		panic(fmt.Errorf("unknown Type(%v)", int(t)))
	}
}

// setTime replaces the global var func used by EventFilter to return a static time.
func setTime(t time.Time) { now = func() time.Time { return t } }

// genEvent is a generic testing event for building touch mechanics and activities over time.
type genEvent struct {
	x, y float32
	typ  Type
}

// Touch returns touch.Event represenation of genEvent.
func (e genEvent) Touch() touch.Event {
	return touch.Event{X: e.x, Y: e.y, Type: e.typ.forTouch()}
}

// Mouse returns mouse.Event representation of genEvent.
func (e genEvent) Mouse() mouse.Event {
	return mouse.Event{X: e.x, Y: e.y, Direction: e.typ.forMouse(), Button: mouse.ButtonLeft}
}

// testEvent wraps an event with a scheduled time to occur.
type testEvent struct {
	event interface{}
	sched time.Time
}

// testQueue is a priority queue for scheduling testEvent.
type testQueue []testEvent

// push schedules testEvent in testQueue.
func (q *testQueue) push(a testEvent) {
	*q = append(*q, a)
	sort.SliceStable(*q, func(i, j int) bool {
		return (*q)[i].sched.Before((*q)[j].sched)
	})
}

// pop next event scheduled and update static clock; returns nil if empty.
func (q *testQueue) pop() interface{} {
	if len(*q) == 0 {
		return nil
	}
	a := (*q)[0]
	*q = (*q)[1:]
	setTime(a.sched)
	return a.event
}

// after schedules an event to occur from current static clock value.
func (q *testQueue) after(dt time.Duration, e interface{}) {
	q.push(testEvent{event: e, sched: now().Add(dt)})
}

func (q *testQueue) gen(dt time.Duration, x, y float32, typ Type) {
	q.after(dt, genEvent{x: x, y: y, typ: typ})
}

func (q *testQueue) new(dt time.Duration, x, y float32) { q.gen(dt, x, y, TypeBegin) }
func (q *testQueue) mov(dt time.Duration, x, y float32) { q.gen(dt, x, y, TypeMove) }
func (q *testQueue) end(dt time.Duration, x, y float32) { q.gen(dt, x, y, TypeEnd) }

// makeMouse returns copy of testQueue replacing instances of genEvent with mouse.Event instead.
func (q testQueue) makeMouse() testQueue {
	var p testQueue
	for _, a := range q {
		if gen, ok := a.event.(genEvent); ok {
			a.event = gen.Mouse()
		}
		p = append(p, a)
	}
	return p
}

// filter runs EventFilter over queue; returns events on queue and generated gesture events.
func (q testQueue) filter() testResult {
	sendAfter = func(_ func(interface{}), dur time.Duration, e interface{}) {
		q.after(dur, e)
	}
	var r testResult
	f := &EventFilter{Send: func(e interface{}) { r.gestures = append(r.gestures, e) }}
	for len(q) != 0 {
		r.events = append(r.events, f.Filter(q.pop()))
	}
	return r
}

type testResult struct{ events, gestures []interface{} }

func (a testResult) Empty() bool { return len(a.gestures) == 0 }

func (a testResult) Last() interface{} { return a.gestures[len(a.gestures)-1] }

func (a testResult) Log(t *testing.T) {
	for _, e := range a.events {
		t.Logf("%[1]T%+[1]v\n", e)
	}
	t.Log("---")
	for _, e := range a.gestures {
		t.Logf("%#v\n", e)
	}
	if a.Empty() {
		t.FailNow()
	}
}

func TestTouch(t *testing.T) {
	var q testQueue
	q.new(0*ms, 0, 0)
	q.end(50*ms, 0, 0)
	r := q.makeMouse().filter()
	r.Log(t)
	if _, ok := r.Last().(Touch); !ok {
		t.FailNow()
	}
}

func TestDrag(t *testing.T) {
	var q testQueue
	q.new(0, 0, 0)
	q.mov(50*ms, 50, 50)
	q.end(90*ms, 90, 90)
	r := q.makeMouse().filter()
	r.Log(t)
	if _, ok := r.Last().(Drag); !ok {
		t.FailNow()
	}
}

func TestLongPress(t *testing.T) {
	var q testQueue
	q.new(0*ms, 0, 0)
	q.end(220*ms, 0, 0)
	r := q.makeMouse().filter()
	r.Log(t)
	if _, ok := r.Last().(LongPress); !ok {
		t.FailNow()
	}
}

func TestLongPressDrag(t *testing.T) {
	var q testQueue
	q.new(0*ms, 0, 0)
	q.mov(220*ms, 50, 50)
	q.end(300*ms, 90, 90)
	r := q.makeMouse().filter()
	r.Log(t)
	if _, ok := r.Last().(LongPressDrag); !ok {
		t.FailNow()
	}
}

func TestDoubleTouch(t *testing.T) {
	var q testQueue
	q.new(0*ms, 0, 0)
	q.end(50*ms, 0, 0)
	q.new(100*ms, 0, 0)
	q.end(150*ms, 0, 0)
	r := q.makeMouse().filter()
	r.Log(t)
	if _, ok := r.Last().(DoubleTouch); !ok {
		t.FailNow()
	}
}

func TestDoubleTouchDrag(t *testing.T) {
	var q testQueue
	q.new(0*ms, 0, 0)
	q.end(50*ms, 0, 0)
	q.new(100*ms, 0, 0)
	q.mov(150*ms, 50, 50)
	q.end(200*ms, 90, 90)
	r := q.makeMouse().filter()
	r.Log(t)
	if _, ok := r.Last().(DoubleTouchDrag); !ok {
		t.FailNow()
	}
}

// func TestDualTouch(t *testing.T) {
// 	t.Fail()
// }

// func TestDualDrag(t *testing.T) {
// 	t.Fail()
// }

// func TestDualLongPress(t *testing.T) {
// 	t.Fail()
// }

// func TestDualLongPressDrag(t *testing.T) {
// 	t.Fail()
// }

// func TestDualDoubleTouch(t *testing.T) {
// 	t.Fail()
// }

// func TestPinchOpen(t *testing.T) {
// 	t.Fail()
// }

// func TestPinchClosed(t *testing.T) {
// 	t.Fail()
// }

// func TestRotate(t *testing.T) {
// 	t.Fail()
// }

func (t Type) Bin() string {
	return fmt.Sprintf("%08b", t)
}

func TestType(t *testing.T) {
	fc := cond(TypeEnd, TypeBegin|TypeEnd)

	c0 := cond(TypeEnd, TypeBegin)
	c1 := cond(TypeEnd, TypeEnd)

	t.Log("fc", fc.Bin())
	t.Log(" &", (c0 & fc).Bin())
	t.Log(" &", (fc & c0).Bin())
	t.Log("c0", c0.Bin(), fc.Has(c0))
	t.Log("c1", c1.Bin(), fc.Has(c1))

	t.Log("TypeEnd|TypeFinal", (TypeEnd | TypeFinal).Has(TypeEnd))
}
