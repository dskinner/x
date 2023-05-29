package glw

import (
	"math"
	"time"

	"golang.org/x/image/math/f32"
)

func ExpDecay(t float64) float64 {
	return 1 - math.Exp(2*math.Pi*-t)
}

func ExpDrive(t float64) float64 {
	return math.Exp(2 * math.Pi * -t)
}

type Transformer interface {
	To(Transform)
	TranslateBy(f32.Vec3)
	TranslateTo(f32.Vec3)
	ShearBy(f32.Vec3)
	ShearTo(f32.Vec3)
	ScaleBy(f32.Vec3)
	ScaleTo(f32.Vec3)
	RotateBy(angle float32, axis f32.Vec3)
	RotateTo(angle float32, axis f32.Vec3)

	// TODO ReflectionBy, ReflectionTo
}

func To(t Transform) func(Transformer) { return func(a Transformer) { a.To(t) } }

func TranslateBy(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.TranslateBy(v) } }
func TranslateTo(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.TranslateTo(v) } }

func ShearBy(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ShearBy(v) } }
func ShearTo(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ShearTo(v) } }

func ScaleBy(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ScaleBy(v) } }
func ScaleTo(v f32.Vec3) func(Transformer) { return func(a Transformer) { a.ScaleTo(v) } }

func RotateBy(angle float32, axis f32.Vec3) func(Transformer) {
	return func(a Transformer) { a.RotateBy(angle, axis) }
}

func RotateTo(angle float32, axis f32.Vec3) func(Transformer) {
	return func(a Transformer) { a.RotateTo(angle, axis) }
}

var _ Transformer = (*Transform)(nil)

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

func (a *Transform) To(b Transform) { *a = b }

func (a *Transform) TranslateBy(v f32.Vec3) { a.Translate = add3fv(a.Translate, v) }
func (a *Transform) TranslateTo(v f32.Vec3) { a.Translate = v }

func (a *Transform) ShearBy(v f32.Vec3) { a.Shear = add3fv(a.Shear, v) }
func (a *Transform) ShearTo(v f32.Vec3) { a.Shear = v }

func (a *Transform) ScaleBy(v f32.Vec3) { a.Scale = mul3fv(a.Scale, v) }
func (a *Transform) ScaleTo(v f32.Vec3) { a.Scale = v }

func (a *Transform) RotateBy(angle float32, axis f32.Vec3) {
	a.Rotate = QuatMul(a.Rotate, Quat(angle, axis))
}

func (a *Transform) RotateTo(angle float32, axis f32.Vec3) {
	a.Rotate = Quat(angle, axis)
}

func (a Transform) Lerp(b Transform, t float32) Transform {
	return Transform{
		Translate: lerp3fv(a.Translate, b.Translate, t),
		Shear:     lerp3fv(a.Shear, b.Shear, t),
		Scale:     lerp3fv(a.Scale, b.Scale, t),
		Rotate:    lerp4fv(a.Rotate, b.Rotate, t),
	}
}

func (a Transform) Eval16fv() (m f32.Mat4) {
	t := translationIdent(a.Translate, a.Shear)
	r := quat16fv(a.Rotate)
	s := scaleIdent(a.Scale)

	// m = mul16fv(r, s)
	// m = mul16fv(m, t)

	m = mul16fv(s, r)
	m = mul16fv(m, t)

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

func Duration(d time.Duration) func(*Animator) { return func(a *Animator) { a.Duration(d) } }

func Interp(fn func(float64) float64) func(*Animator) { return func(a *Animator) { a.Interp(fn) } }

var DefaultAnimatorDuration = 1000 * time.Millisecond

type Animator struct {
	// animations lerp (at) -> (to) for dur, storing result in pt
	at, pt, to Transform

	// TODO epochs for each transform type (translate, rotate, etc) so each operates independently
	epoch time.Time

	dur    time.Duration
	interp func(float64) float64
}

func NewAnimator(options ...func(*Animator)) *Animator {
	a := &Animator{
		at:     TransformIdent(),
		pt:     TransformIdent(),
		to:     TransformIdent(),
		dur:    DefaultAnimatorDuration,
		interp: ExpDecay,
	}
	a.Apply(options...)
	return a
}

func (a *Animator) Apply(options ...func(*Animator)) {
	if len(options) == 0 {
		return
	}
	a.Cancel()
	for _, opt := range options {
		opt(a)
	}
}

func (a *Animator) Duration(d time.Duration)        { a.dur = d }
func (a *Animator) Interp(fn func(float64) float64) { a.interp = fn }

func (a *Animator) Pt() Transform  { return a.pt }
func (a *Animator) To(t Transform) { a.to = t } // TODO questionable if this is needed

func (a *Animator) TranslateBy(v f32.Vec3) { a.to.TranslateBy(v) }
func (a *Animator) TranslateTo(v f32.Vec3) { a.to.TranslateTo(v) }

func (a *Animator) ShearBy(v f32.Vec3) { a.to.ShearBy(v) }
func (a *Animator) ShearTo(v f32.Vec3) { a.to.ShearTo(v) }

func (a *Animator) ScaleBy(v f32.Vec3) { a.to.ScaleBy(v) }
func (a *Animator) ScaleTo(v f32.Vec3) { a.to.ScaleTo(v) }

func (a *Animator) RotateBy(angle float32, axis f32.Vec3) { a.to.RotateBy(angle, axis) }
func (a *Animator) RotateTo(angle float32, axis f32.Vec3) { a.to.RotateTo(angle, axis) }

// TODO this cancels anything in progress but doesn't allow independent transforms to continue
// e.g. trigger 1s rotate, 0.5 seconds elapse, trigger 1s translate,
// 0.5 second elapse and rotate finishes, 0.5 seconds elapse and translate finishes
// what's actually desireable here? I think I'd want things to finish independently
// which suggests a different kind of layout
func (a *Animator) Stage(epoch time.Time, transforms ...func(Transformer)) {
	a.Cancel()
	a.epoch = epoch
	for _, tr := range transforms {
		tr(a)
	}
}

// TODO Epoch, Step, Cancel, Done allow building out custom pipeline but is that
// actually desireable ???

// Epoch sets Animator start time for duration and sets all internal transforms to last resolved.
//
// TODO its like Cancel but for starting a new animation which is confusing
// if actually desirable just make Cancel call Epoch(time.Time{}) ???
// func (a *Animator) Epoch(now time.Time) {
// 	if now == a.epoch {
// 		return
// 	}
// 	a.epoch = now
// 	// a.at = a.pt
// 	// a.to = a.pt
// }

func (a *Animator) Step(now time.Time) (ok bool) {
	if a.epoch == (time.Time{}) {
		return false
	}
	since := now.Sub(a.epoch)
	if ok = since < a.dur; ok {
		delta := float32(a.interp(float64(since) / float64(a.dur)))
		a.pt = a.at.Lerp(a.to, delta)
	} else {
		a.Done()
	}
	return ok
}

func (a *Animator) Cancel() {
	// a.pt is last lerp of (at) -> (to) so set accordingly
	a.at = a.pt
	a.to = a.pt
	a.epoch = time.Time{}
}

func (a *Animator) Done() {
	a.at = a.to
	a.pt = a.to
	a.epoch = time.Time{}
}
