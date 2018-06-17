package glw

import "golang.org/x/mobile/gl"

type A2fv gl.Attrib

func (a A2fv) Enable()  { ctx.EnableVertexAttribArray(gl.Attrib(a)) }
func (a A2fv) Disable() { ctx.DisableVertexAttribArray(gl.Attrib(a)) }
func (a A2fv) Pointer() {
	a.Enable()
	ctx.VertexAttribPointer(gl.Attrib(a), 2, gl.FLOAT, false, 0, 0)
}

type A3fv gl.Attrib

func (a A3fv) Enable()  { ctx.EnableVertexAttribArray(gl.Attrib(a)) }
func (a A3fv) Disable() { ctx.DisableVertexAttribArray(gl.Attrib(a)) }
func (a A3fv) Pointer() {
	a.Enable()
	ctx.VertexAttribPointer(gl.Attrib(a), 3, gl.FLOAT, false, 0, 0)
}

type A4fv gl.Attrib

func (a A4fv) Enable()  { ctx.EnableVertexAttribArray(gl.Attrib(a)) }
func (a A4fv) Disable() { ctx.DisableVertexAttribArray(gl.Attrib(a)) }
func (a A4fv) Pointer() {
	a.Enable()
	ctx.VertexAttribPointer(gl.Attrib(a), 4, gl.FLOAT, false, 0, 0)
}
