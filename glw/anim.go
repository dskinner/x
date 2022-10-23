package glw

import (
	"math"
	"sync/atomic"
	"time"

	"golang.org/x/image/math/f32"
)

func ExpDecay(t float64) float64 {
	return math.Exp(2*math.Pi*-t)
}

func ExpDrive(t float64) float64 {
	return 1 - math.Exp(2*math.Pi*-t)
}

type Transformer interface {
	To(Transform)
	TranslateAt(f32.Vec3)
	TranslateBy(f32.Vec3)
	TranslateTo(f32.Vec3)
	ShearAt(f32.Vec3)
	ShearBy(f32.Vec3)
	ShearTo(f32.Vec3)
	ScaleAt(f32.Vec3)
	ScaleBy(f32.Vec3)
	ScaleTo(f32.Vec3)
	RotateAt(angle float32, axis f32.Vec3)
	RotateBy(angle float32, axis f32.Vec3)
	RotateTo(angle float32, axis f32.Vec3)

	// TODO ReflectionAt, ReflectionBy, ReflectionTo
}

func To(t Transform) func(Transformer) { return func(a Transformer) { a.To(t) } }

func TranslateAt(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.TranslateAt(v) } }
func TranslateBy(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.TranslateBy(v) } }
func TranslateTo(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.TranslateTo(v) } }

func ShearAt(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ShearAt(v) } }
func ShearBy(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ShearBy(v) } }
func ShearTo(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ShearTo(v) } }

func ScaleAt(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ScaleAt(v) } }
func ScaleBy(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ScaleBy(v) } }
func ScaleTo(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ScaleTo(v) } }

func RotateAt(angle float32, axis f32.Vec3) func(Transformer) {
	return func(a Transformer) { a.RotateAt(angle, axis) }
}

func RotateBy(angle float32, axis f32.Vec3) func(Transformer) {
	return func(a Transformer) { a.RotateBy(angle, axis) }
}

func RotateTo(angle float32, axis f32.Vec3) func(Transformer) {
	return func(a Transformer) { a.RotateTo(angle, axis) }
}

// TODO actually implement Transformer interface.
// var _ Transformer = (*Transform)(nil)

// TODO this is essentially type Sheet, but should vectors be tucked behind Transform type ???
type Transform struct {
	Translate f32.Vec3
	Shear     f32.Vec3
	Scale     f32.Vec3
	Rotate    f32.Vec4
}

func TransformIdent() (a Transform) {
	return Transform{
		Translate: f32.Vec3{0, 0, 0},
		Shear:     f32.Vec3{0, 0, 0},
		Scale:     f32.Vec3{1, 1, 1},
		Rotate:    f32.Vec4{1, 0, 0, 0},
	}
}

func (a *Transform) TranslateBy(v f32.Vec3) { a.Translate = add3fv(a.Translate, v) }
func (a *Transform) TranslateTo(v f32.Vec3) { a.Translate = v }

func (a *Transform) ShearBy(v f32.Vec3) { a.Shear = add3fv(a.Shear, v) }
func (a *Transform) ShearTo(v f32.Vec3) { a.Shear = v }

func (a *Transform) ScaleBy(v f32.Vec3) { a.Scale = mul3fv(a.Scale, v) }
func (a *Transform) ScaleTo(v f32.Vec3) { a.Scale = v }

func (a *Transform) RotateBy(angle float32, axis f32.Vec3) {
	a.Rotate = norm4fv(QuatMul(a.Rotate, Quat(angle, axis)))
}

func (a *Transform) RotateTo(angle float32, axis f32.Vec3) {
	a.Rotate = norm4fv(Quat(angle, axis))
}

func (a Transform) Lerp(b Transform, t float32) Transform {
	return Transform{
		Translate: lerp3fv(a.Translate, b.Translate, t),
		Shear:     lerp3fv(a.Shear, b.Shear, t),
		Scale:     lerp3fv(a.Scale, b.Scale, t),
		Rotate:    norm4fv(lerp4fv(a.Rotate, b.Rotate, t)),
	}
}

func (a Transform) Eval16fv() (m f32.Mat4) {
	t := translationIdent(a.Translate, a.Shear)
	r := quat16fv(a.Rotate)
	s := scaleIdent(a.Scale)
	_, _, _ = t, r, s

	// m = mul16fv(r, s)
	// m = mul16fv(m, t)

	m = mul16fv(s, r)
	m = mul16fv(m, t)
	// m = mul16fv(s, t)
	// m = mul16fv(m, r)

	// m = mul16fv(r, t)
	// m = mul16fv(m, s)

	return m
}

// TODO confirm a sensible order of operations
func (a Transform) eval4fv() f32.Vec4 {
	// return quatmul(a.Rotate, mul4fv(a.Translate, a.Scale))
	// return mul4fv(a.Scale, QuatMul(a.Rotate, a.Translate))
	panic("eval4fv not implemented")
}

// TODO ...
func (a Transform) eval3fv() f32.Vec3 {
	v := a.eval4fv()
	return f32.Vec3{v[0], v[1], v[2]}
}

// TODO ... ...
func (a Transform) eval2fv() f32.Vec2 {
	v := a.eval4fv()
	return f32.Vec2{v[0], v[1]}
}

// TODO ... ... ... ugh
func (a Transform) eval1f() float32 {
	return a.eval4fv()[0]
}

type Animator interface {
	// Apply sets default options on animator. Any animation in progress is cancelled
	// first unless the length of options is zero, then this call has no effect.
	Apply(options ...func(Animator))

	Tick(time.Duration)

	Duration(time.Duration)

	// Interp defaults to exponential decay.
	Interp(func(float64) float64)

	Notify(*uint32)

	At() Transform

	Start(transforms ...func(Transformer))

	Stage(epoch time.Time, transforms ...func(Transformer))

	Step(now time.Time) bool

	Cancel()
}

func Tick(d time.Duration) func(Animator) { return func(a Animator) { a.Tick(d) } }

func Duration(d time.Duration) func(Animator) { return func(a Animator) { a.Duration(d) } }

func Interp(fn func(float64) float64) func(Animator) { return func(a Animator) { a.Interp(fn) } }

func Notify(p *uint32) func(Animator) { return func(a Animator) { a.Notify(p) } }

type animator struct {
	at, pt, to Transform

	epoch  time.Time
	dur    time.Duration
	interp func(float64) float64

	notify    *uint32
	tick      time.Duration
	die, done chan struct{}
}

func newanimator(options ...func(Animator)) *animator {
	a := &animator{
		at:     TransformIdent(),
		pt:     TransformIdent(),
		to:     TransformIdent(),
		tick:   16 * time.Millisecond,
		interp: ExpDecay,
		die:    make(chan struct{}),
		done:   make(chan struct{}),
	}
	close(a.die)
	close(a.done)
	a.Apply(options...)
	return a
}

func (a *animator) Apply(options ...func(Animator)) {
	if len(options) == 0 {
		return
	}
	a.Cancel()
	for _, opt := range options {
		opt(a)
	}
}

func (a *animator) Tick(d time.Duration)            { a.tick = d }
func (a *animator) Duration(d time.Duration)        { a.dur = d }
func (a *animator) Interp(fn func(float64) float64) { a.interp = fn }

func (a *animator) Notify(p *uint32) { a.notify = p }

func (a *animator) At() Transform  { return a.to }
func (a *animator) To(t Transform) { a.to = t }

func (a *animator) TranslateAt(v f32.Vec3) { a.at.TranslateTo(v) }
func (a *animator) TranslateBy(v f32.Vec3) { a.to.TranslateBy(v) }
func (a *animator) TranslateTo(v f32.Vec3) { a.to.TranslateTo(v) }

func (a *animator) ShearAt(v f32.Vec3) { a.at.ShearTo(v) }
func (a *animator) ShearBy(v f32.Vec3) { a.to.ShearBy(v) }
func (a *animator) ShearTo(v f32.Vec3) { a.to.ShearTo(v) }

func (a *animator) ScaleAt(v f32.Vec3) { a.at.ScaleTo(v) }
func (a *animator) ScaleBy(v f32.Vec3) { a.to.ScaleBy(v) }
func (a *animator) ScaleTo(v f32.Vec3) { a.to.ScaleTo(v) }

func (a *animator) RotateAt(angle float32, axis f32.Vec3) { a.at.RotateTo(angle, axis) }
func (a *animator) RotateBy(angle float32, axis f32.Vec3) { a.to.RotateBy(angle, axis) }
func (a *animator) RotateTo(angle float32, axis f32.Vec3) { a.to.RotateTo(angle, axis) }

func (a *animator) Cancel() {
	select {
	case <-a.done:
	default:
		close(a.die)
		<-a.done
	}
}

func (a *animator) start() {
	if a.dur == 0 {
		a.at = a.to
		a.pt = a.to
		a.end()
		close(a.done)
		return
	}
	ticker := time.NewTicker(a.tick)
	for {
		select {
		case now := <-ticker.C:
			if !a.Step(now) {
				ticker.Stop()
				a.end()
				close(a.done)
				return
			}
		case <-a.die:
			ticker.Stop()
			a.end()
			close(a.done)
			return
		}
	}
}

func (a *animator) Step(now time.Time) (ok bool) {
	if a.epoch == (time.Time{}) {
		a.epoch = now
		return true
	}
	since := now.Sub(a.epoch)
	if ok = since < a.dur; ok {
		delta := float32(a.interp(float64(since) / float64(a.dur)))
		a.pt = a.at.Lerp(a.to, delta)
	} else {
		a.at = a.to
		a.pt = a.to
	}
	return ok
}

func (a *animator) end() {
	if a.notify != nil {
		atomic.AddUint32(a.notify, ^uint32(0))
	}
}

func (a *animator) listen() {
	select {
	case <-a.die:
		a.end()
		close(a.done)
	}
}

func (a *animator) stage(epoch time.Time, transforms ...func(Transformer)) {
	if a.notify != nil {
		atomic.AddUint32(a.notify, 1)
	}
	a.Cancel()
	a.die = make(chan struct{})
	a.done = make(chan struct{})

	a.epoch = epoch
	a.at = a.pt
	a.to = a.pt

	for _, opt := range transforms {
		opt(a)
	}
}

func (a *animator) Stage(epoch time.Time, transforms ...func(Transformer)) {
	a.stage(epoch, transforms...)
	go a.listen()
}

func (a *animator) Start(transforms ...func(Transformer)) {
	a.stage(time.Now(), transforms...)
	go a.start()
}
