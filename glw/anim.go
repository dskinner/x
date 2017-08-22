package glw

import (
	"math"
	"sync/atomic"
	"time"

	"golang.org/x/image/math/f32"
)

func ExpDecay(t float64) float64 {
	return 1 - math.Exp(2*math.Pi*-t)
}

type Transformer interface {
	To(Transform)
	ScaleAt(f32.Vec4)
	ScaleBy(f32.Vec4)
	ScaleTo(f32.Vec4)
	TranslateAt(f32.Vec4)
	TranslateBy(f32.Vec4)
	TranslateTo(f32.Vec4)
	RotateAt(angle float32, axis f32.Vec3)
	RotateBy(angle float32, axis f32.Vec3)
	RotateTo(angle float32, axis f32.Vec3)
}

func To(t Transform) func(Transformer) {
	return func(a Transformer) { a.To(t) }
}

func ScaleAt(v f32.Vec4) func(Transformer) {
	return func(a Transformer) { a.ScaleAt(v) }
}

func ScaleBy(v f32.Vec4) func(Transformer) {
	return func(a Transformer) { a.ScaleBy(v) }
}

func ScaleTo(v f32.Vec4) func(Transformer) {
	return func(a Transformer) { a.ScaleTo(v) }
}

func TranslateAt(v f32.Vec4) func(Transformer) {
	return func(a Transformer) { a.TranslateAt(v) }
}

func TranslateBy(v f32.Vec4) func(Transformer) {
	return func(a Transformer) { a.TranslateBy(v) }
}

func TranslateTo(v f32.Vec4) func(Transformer) {
	return func(a Transformer) { a.TranslateTo(v) }
}

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
	Translate f32.Vec4
	Rotate    f32.Vec4
	Scale     f32.Vec4
}

func TransformIdent() (a Transform) {
	a.Ident()
	return
}

func (a *Transform) Ident() {
	a.Scale = f32.Vec4{1, 1, 1, 1}
	a.Translate = f32.Vec4{0, 0, 0, 0}
	a.Rotate = f32.Vec4{1, 0, 0, 0}
}

func (a *Transform) ScaleBy(v f32.Vec4) { a.Scale = mul4fv(a.Scale, v) }
func (a *Transform) ScaleTo(v f32.Vec4) { a.Scale = v }

func (a *Transform) TranslateBy(v f32.Vec4) { a.Translate = add4fv(a.Translate, v) }
func (a *Transform) TranslateTo(v f32.Vec4) { a.Translate = v }

func (a *Transform) RotateBy(angle float32, axis f32.Vec3) {
	a.Rotate = mulquat(a.Rotate, quat(angle, axis))
}
func (a *Transform) RotateTo(angle float32, axis f32.Vec3) { a.Rotate = quat(angle, axis) }

func (a Transform) lerp(b Transform, t float32) Transform {
	return Transform{
		Scale:     lerp4fv(a.Scale, b.Scale, t),
		Translate: lerp4fv(a.Translate, b.Translate, t),
		Rotate:    lerp4fv(a.Rotate, b.Rotate, t),
	}
}

func (a Transform) eval16fv() f32.Mat4 {
	m := translate16fv(a.Translate, ident16fv())
	m = rotate16fv(a.Rotate, m)
	m = scale16fv(a.Scale, m)
	return m
}

func (a Transform) eval4fv() f32.Vec4 {
	// return quatmul(a.Rotate, mul4fv(a.Translate, a.Scale))
	return mul4fv(a.Scale, mulquat(a.Rotate, a.Translate))
}

func (a Transform) eval3fv() f32.Vec3 {
	v := a.eval4fv()
	return f32.Vec3{v[0], v[1], v[2]}
}

type Animator interface {
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

func Tick(d time.Duration) func(Animator) {
	return func(a Animator) { a.Tick(d) }
}

func Duration(d time.Duration) func(Animator) {
	return func(a Animator) { a.Duration(d) }
}

func Interp(fn func(float64) float64) func(Animator) {
	return func(a Animator) { a.Interp(fn) }
}

func Notify(p *uint32) func(Animator) {
	return func(a Animator) { a.Notify(p) }
}

type animator struct {
	at, pt, to Transform

	epoch  time.Time
	dur    time.Duration
	interp func(float64) float64

	notify    *uint32
	tick      time.Duration
	die, done chan struct{}
}

func newanimator() *animator {
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
	return a
}

func (a *animator) apply(options ...func(Animator)) {
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

func (a *animator) ScaleAt(v f32.Vec4) { a.at.Scale = v }
func (a *animator) ScaleBy(v f32.Vec4) { a.to.Scale = mul4fv(a.to.Scale, v) }
func (a *animator) ScaleTo(v f32.Vec4) { a.to.Scale = v }

func (a *animator) TranslateAt(v f32.Vec4) { a.at.Translate = v }
func (a *animator) TranslateBy(v f32.Vec4) { a.to.Translate = add4fv(a.to.Translate, v) }
func (a *animator) TranslateTo(v f32.Vec4) { a.to.Translate = v }

func (a *animator) RotateAt(angle float32, axis f32.Vec3) { a.at.Rotate = quat(angle, axis) }
func (a *animator) RotateBy(angle float32, axis f32.Vec3) {
	a.to.Rotate = mulquat(a.to.Rotate, quat(angle, axis))
}
func (a *animator) RotateTo(angle float32, axis f32.Vec3) { a.to.Rotate = quat(angle, axis) }

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
		a.pt = a.at.lerp(a.to, delta)
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
