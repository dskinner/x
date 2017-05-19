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
	ScaleAt(f32.Vec3)
	ScaleBy(f32.Vec3)
	ScaleTo(f32.Vec3)
	TranslateAt(f32.Vec4)
	TranslateBy(f32.Vec4)
	TranslateTo(f32.Vec4)
	RotateAt(angle float32, axis f32.Vec3)
	RotateBy(angle float32, axis f32.Vec3)
	RotateTo(angle float32, axis f32.Vec3)
}

func ScaleAt(v f32.Vec3) func(Transformer) {
	return func(a Transformer) { a.ScaleAt(v) }
}

func ScaleBy(v f32.Vec3) func(Transformer) {
	return func(a Transformer) { a.ScaleBy(v) }
}

func ScaleTo(v f32.Vec3) func(Transformer) {
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

type transform struct {
	scale     f32.Vec3
	translate f32.Vec4
	rotate    f32.Vec4
}

func transformIdent() transform {
	return transform{scale: f32.Vec3{1, 1, 1}, rotate: f32.Vec4{1, 0, 0, 0}}
}

func (a transform) lerp(b transform, t float32) transform {
	return transform{
		scale:     lerp3fv(a.scale, b.scale, t),
		translate: lerp4fv(a.translate, b.translate, t),
		rotate:    lerp4fv(a.rotate, b.rotate, t),
	}
}

func (a transform) eval16fv() f32.Mat4 {
	m := rotate16fv(a.rotate, ident16fv())
	m = translate16fv(a.translate, m)
	m = scale16fv(a.scale, m)
	return m
}

func (a transform) eval4fv() f32.Vec4 {
	// TODO scale and rotate?
	return a.translate
}

func (a transform) eval3fv() f32.Vec3 {
	// TODO scale and rotate? drop w?
	v := a.eval4fv()
	return f32.Vec3{v[0], v[1], v[2]}
}

type Animator interface {
	Tick(time.Duration)

	Duration(time.Duration)

	// Interp defaults to exponential decay.
	Interp(func(float64) float64)

	Notify(*uint32)

	Transform(...func(Transformer))

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
	u16fv  *U16fv
	at, to transform
	delta  float32

	tick, dur time.Duration
	interp    func(float64) float64

	notify *uint32

	eval func(transform)

	die, done chan struct{}
}

func newanimator(eval func(transform)) *animator {
	return &animator{
		eval:   eval,
		at:     transformIdent(),
		tick:   16 * time.Millisecond,
		interp: ExpDecay,
	}
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

func (a *animator) ScaleAt(v f32.Vec3) { a.at.scale = v }
func (a *animator) ScaleBy(v f32.Vec3) { a.to.scale = mul3fv(a.to.scale, v) }
func (a *animator) ScaleTo(v f32.Vec3) { a.to.scale = v }

func (a *animator) TranslateAt(v f32.Vec4) { a.at.translate = v }
func (a *animator) TranslateBy(v f32.Vec4) { a.to.translate = add4fv(a.to.translate, v) }
func (a *animator) TranslateTo(v f32.Vec4) { a.to.translate = v }

func (a *animator) RotateAt(angle float32, axis f32.Vec3) { a.at.rotate = quat(angle, axis) }
func (a *animator) RotateBy(angle float32, axis f32.Vec3) {
	a.to.rotate = quatmul(a.to.rotate, quat(angle, axis))
}
func (a *animator) RotateTo(angle float32, axis f32.Vec3) { a.to.rotate = quat(angle, axis) }

func (a *animator) Cancel() {
	if a.die != nil {
		close(a.die)
		<-a.done
		a.die = nil
		a.done = nil
	}
}

func (a *animator) start() {
	if a.dur == 0 {
		a.end(a.to)
		close(a.done)
		return
	}

	start := time.Now()
	ticker := time.NewTicker(a.tick)
	for {
		select {
		case now := <-ticker.C:
			since := now.Sub(start)
			if since < a.dur {
				a.delta = float32(a.interp(float64(since) / float64(a.dur)))
				a.update(a.at.lerp(a.to, a.delta))
			} else {
				a.end(a.to)
				ticker.Stop()
				close(a.done)
				return
			}
		case <-a.die:
			a.end(a.at.lerp(a.to, a.delta))
			ticker.Stop()
			close(a.done)
			return
		}
	}
}

func (a *animator) update(t transform) {
	if a.eval != nil {
		a.eval(t)
	}
}

func (a *animator) end(t transform) {
	a.update(t)
	a.at = t
	if a.notify != nil {
		atomic.AddUint32(a.notify, ^uint32(0))
	}
}

func (a *animator) Transform(values ...func(Transformer)) {
	if a.notify != nil {
		atomic.AddUint32(a.notify, 1)
	}
	a.Cancel()
	a.die = make(chan struct{})
	a.done = make(chan struct{})
	a.delta = 0
	a.to = a.at
	for _, opt := range values {
		opt(a)
	}
	go a.start()
}
