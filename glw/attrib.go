package glw

import "github.com/go-gl/gl/v4.1-core/gl"

type AttribLocation struct {
	Value uint32
}

type A2fv AttribLocation

func (a A2fv) Enable()  { gl.EnableVertexAttribArray(a.Value) }
func (a A2fv) Disable() { gl.DisableVertexAttribArray(a.Value) }

func (a A2fv) Pointer() {
	a.Enable()
	gl.VertexAttribPointer(a.Value, 2, gl.FLOAT, false, 2*4, nil)
}

func (a A2fv) PointerWithOffset(offset int) {
	a.Enable()
	gl.VertexAttribPointerWithOffset(a.Value, 2, gl.FLOAT, false, 2*4, uintptr(offset*4))
}

type A3fv AttribLocation

func (a A3fv) Enable()  { gl.EnableVertexAttribArray(a.Value) }
func (a A3fv) Disable() { gl.DisableVertexAttribArray(a.Value) }

func (a A3fv) Pointer() {
	a.Enable()
	gl.VertexAttribPointer(a.Value, 3, gl.FLOAT, false, 3*4, nil)
}

func (a A3fv) PointerWithOffset(offset int) {
	a.Enable()
	gl.VertexAttribPointerWithOffset(a.Value, 3, gl.FLOAT, false, 3*4, uintptr(offset*4))
}

type A4fv AttribLocation

func (a A4fv) Enable()  { gl.EnableVertexAttribArray(a.Value) }
func (a A4fv) Disable() { gl.DisableVertexAttribArray(a.Value) }

func (a A4fv) Pointer() {
	a.Enable()
	gl.VertexAttribPointer(a.Value, 4, gl.FLOAT, false, 4*4, nil)
}

func (a A4fv) PointerWithOffset(offset int) {
	a.Enable()
	gl.VertexAttribPointerWithOffset(a.Value, 4, gl.FLOAT, false, 4*4, uintptr(offset*4))
}