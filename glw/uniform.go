package glw

import (
	"time"

	"golang.org/x/image/math/f32"
	"golang.org/x/mobile/gl"
)

type Uniform interface {
	// Animator allows setting additional options.
	Animator(options ...func(*Animator)) *Animator

	// Transform applies given transforms instantly.
	Transform(transforms ...func(Transformer))

	// Stage transforms to occur the next time Step is called.
	// Previously staged transforms or animations in progress
	// are cancelled with each call.
	Stage(epoch time.Time, transforms ...func(Transformer))

	// Step through staged transforms, returning true if work
	// was performed. The first false result returned performs
	// work to animate to final destination; subsequent calls
	// do not perform any work.
	Step(now time.Time) (ok bool)
}

type uniform struct {
	gl.Uniform
	animator *Animator
}

func newuniform(a gl.Uniform) *uniform {
	return &uniform{Uniform: a, animator: NewAnimator()}
}

func (u *uniform) Animator(options ...func(*Animator)) *Animator {
	u.animator.Apply(options...)
	return u.animator
}

func (u *uniform) Transform(transforms ...func(Transformer)) {
	for _, tr := range transforms {
		tr(u.animator)
	}
	u.animator.Done()
}

func (u *uniform) Stage(epoch time.Time, transforms ...func(Transformer)) {
	u.animator.Stage(epoch, transforms...)
}

func (u *uniform) Step(now time.Time) (ok bool) { return u.animator.Step(now) }

type U1i gl.Uniform

func (u U1i) Set(v int) { ctx.Uniform1i(gl.Uniform(u), v) }

type U2i gl.Uniform

func (u U2i) Set(v0, v1 int) { ctx.Uniform2i(gl.Uniform(u), v0, v1) }

type U3i gl.Uniform

func (u U3i) Set(v0, v1, v2 int32) { ctx.Uniform3i(gl.Uniform(u), v0, v1, v2) }

type U4i gl.Uniform

func (u U4i) Set(v0, v1, v2, v3 int32) { ctx.Uniform4i(gl.Uniform(u), v0, v1, v2, v3) }

type U1f struct{ *uniform }

// TODO
// func (u *U1f) Update() { ctx.Uniform1f(u.Uniform, u.animator.pt.eval1f()) }

type U2fv struct {
	gl.Uniform
	v f32.Vec2
}

func (u U2fv) Update() { ctx.Uniform2fv(u.Uniform, u.v[:]) }

func (u *U2fv) Set(v f32.Vec2) { u.v = v }

type U3fv struct {
	gl.Uniform
	v f32.Vec3
}

func (u U3fv) Update() { ctx.Uniform3fv(u.Uniform, u.v[:]) }

func (u U3fv) Set(v f32.Vec3) { u.v = v }

type U4fv struct {
	gl.Uniform
	animating uint32

	v f32.Vec4
	// a *animator
}

func (u U4fv) Update() {
	// if u.a != nil {
	// panic("not implemented")
	// u.v = u.a.pt.eval4fv()
	// }
	ctx.Uniform4fv(u.Uniform, u.v[:])
}

// func (u *U4fv) Set(v f32.Vec4) {
// 	if u.a != nil {
// 		u.a.pt.Translate = v
// 	}
// 	u.v = v
// 	gl.Uniform4fv(u.Value, 1, &u.v[0])
// }

func (u *U4fv) Animator(options ...func(*Animator)) *Animator {
	panic("TODO")
	// if u.a == nil {
	// 	u.a = newanimator(Notify(&u.animating))
	// }
	// u.a.Apply(options...)
	// return u.a
}

func (u *U4fv) Transform(transforms ...func(Transformer)) {
	panic("TODO")
	// u.Animator().Start(transforms...)
}

func (u *U4fv) Stage(epoch time.Time, transforms ...func(Transformer)) {
	// u.a.Stage(epoch, transforms...)
}

// func (u *U4fv) Step(now time.Time) (ok bool) {
// 	if ok = u.animating != 0; ok {
// 		if ok = u.a.Step(now); !ok {
// 			u.a.Cancel()
// 		}
// 		u.Update()
// 	}
// 	return ok
// }

type U9fv gl.Uniform

func (u U9fv) Set(m f32.Mat3) { ctx.UniformMatrix4fv(gl.Uniform(u), m[:]) }

type U16fv struct{ *uniform }

func (u *U16fv) Update() {
	m := u.animator.pt.Eval16fv()
	ctx.UniformMatrix4fv(u.Uniform, m[:])
}

func (u *U16fv) Inv2f(nx, ny float32) (float32, float32) {
	m := u.animator.pt.Eval16fv()
	return nx*(1/m[0]) + ny*m[1], nx*m[4] + ny*(1/m[5])

	// m := inv16fv(u.animator.pt.eval16fv())
	// return nx*m[0] + ny*m[1], nx*m[4] + ny*m[5]
}

func (u *U16fv) Ortho(l, r float32, b, t float32, n, f float32) {
	u.animator.to.TranslateTo(f32.Vec3{
		-(r + l) / (r - l),
		-(t + b) / (t - b),
		-(f + n) / (f - n),
	})
	u.animator.to.ScaleTo(f32.Vec3{
		+2 / (r - l),
		+2 / (t - b),
		-2 / (f - n),
	})
	u.animator.at = u.animator.to
	u.animator.pt = u.animator.to
	u.Update()
}

func (u U16fv) String() string {
	return string16fv(u.animator.pt.Eval16fv())
}
