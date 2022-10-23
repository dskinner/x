package glw

import (
	"time"

	"golang.org/x/image/math/f32"
)

type Animator2 struct {
	at, pt, to Transform

	epoch  time.Time
	dur    time.Duration
	interp func(float64) float64
}

func (a *Animator2) Duration(d time.Duration)        { a.dur = d }
func (a *Animator2) Interp(fn func(float64) float64) { a.interp = fn }

func (a *Animator2) Pt() Transform { return a.pt }

func (a *Animator2) TranslateBy(v f32.Vec3) { a.to.TranslateBy(v) }
func (a *Animator2) TranslateTo(v f32.Vec3) { a.to.TranslateTo(v) }
func (a *Animator2) TranslateAt(v f32.Vec3) {
	a.at.TranslateTo(v)
	a.pt.TranslateTo(v)
	a.to.TranslateTo(v)
}

func (a *Animator2) ShearBy(v f32.Vec3) { a.to.ShearBy(v) }
func (a *Animator2) ShearTo(v f32.Vec3) { a.to.ShearTo(v) }
func (a *Animator2) ShearAt(v f32.Vec3) {
	a.at.ShearTo(v)
	a.pt.ShearTo(v)
	a.to.ShearTo(v)
}

func (a *Animator2) ScaleBy(v f32.Vec3) { a.to.ScaleBy(v) }
func (a *Animator2) ScaleTo(v f32.Vec3) { a.to.ScaleTo(v) }
func (a *Animator2) ScaleAt(v f32.Vec3) {
	a.at.ScaleTo(v)
	a.pt.ScaleTo(v)
	a.to.ScaleTo(v)
}

func (a *Animator2) RotateBy(angle float32, axis f32.Vec3) { a.to.RotateBy(angle, axis) }
func (a *Animator2) RotateTo(angle float32, axis f32.Vec3) { a.to.RotateTo(angle, axis) }
func (a *Animator2) RotateAt(angle float32, axis f32.Vec3) {
	a.at.RotateTo(angle, axis)
	a.pt.RotateTo(angle, axis)
	a.to.RotateTo(angle, axis)
}

// Epoch sets Animator start time for duration and sets all internal transforms to last resolved.
func (a *Animator2) Epoch(now time.Time) {
	if now == a.epoch {
		return
	}
	a.epoch = now
	a.at = a.pt
	a.to = a.pt
}

func (a *Animator2) Step(now time.Time) (ok bool) {
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

func (a *Animator2) Step2(dt float32) {
	a.pt = a.at.Lerp(a.to, dt)
	a.at = a.pt
	// TODO if a.pt is almost equal a.to, set it so
}

func (a *Animator2) Step3() {
	a.at = a.to
	a.pt = a.to
}