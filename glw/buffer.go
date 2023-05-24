package glw

import (
	"math"

	"golang.org/x/mobile/gl"
)

type FloatBuffer struct {
	gl.Buffer
	bin   []byte
	count int
	usage gl.Enum
}

func (buf *FloatBuffer) Create(usage gl.Enum, data []float32) {
	buf.usage = usage
	buf.Buffer = ctx.CreateBuffer()
	buf.Bind()
	buf.Update(data)
}

func (buf FloatBuffer) Delete()           { ctx.DeleteBuffer(buf.Buffer) }
func (buf *FloatBuffer) Bind()            { ctx.BindBuffer(gl.ARRAY_BUFFER, buf.Buffer) }
func (buf FloatBuffer) Unbind()           { ctx.BindBuffer(gl.ARRAY_BUFFER, gl.Buffer{Value: 0}) }
func (buf FloatBuffer) Draw(mode gl.Enum) { ctx.DrawArrays(mode, 0, buf.count) }

func (buf *FloatBuffer) Update(data []float32) {
	// TODO see gl.Ptr: gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// TODO seems like this could be simplified
	// also look at UintBuffer
	buf.count = len(data)
	subok := len(buf.bin) > 0 && len(data)*4 <= len(buf.bin)
	if !subok {
		buf.bin = make([]byte, len(data)*4)
	}
	for i, x := range data {
		u := math.Float32bits(x)
		buf.bin[4*i+0] = byte(u >> 0)
		buf.bin[4*i+1] = byte(u >> 8)
		buf.bin[4*i+2] = byte(u >> 16)
		buf.bin[4*i+3] = byte(u >> 24)
	}
	if subok {
		ctx.BufferSubData(gl.ARRAY_BUFFER, 0, buf.bin)
	} else {
		ctx.BufferData(gl.ARRAY_BUFFER, buf.bin, buf.usage)
	}
}

type UintBuffer struct {
	gl.Buffer
	bin   []byte
	count int
	usage gl.Enum
}

func (buf *UintBuffer) Create(usage gl.Enum, data []uint32) {
	buf.usage = usage
	buf.Buffer = ctx.CreateBuffer()
	buf.Bind()
	buf.Update(data)
}

func (buf UintBuffer) Delete()           { ctx.DeleteBuffer(buf.Buffer) }
func (buf *UintBuffer) Bind()            { ctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, buf.Buffer) }
func (buf UintBuffer) Unbind()           { ctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, gl.Buffer{Value: 0}) }
func (buf UintBuffer) Draw(mode gl.Enum) { ctx.DrawElements(mode, buf.count, gl.UNSIGNED_INT, 0) }

func (buf *UintBuffer) Update(data []uint32) {
	buf.count = len(data)
	subok := len(buf.bin) > 0 && len(data)*4 <= len(buf.bin)
	if !subok {
		buf.bin = make([]byte, len(data)*4)
	}
	for i, u := range data {
		buf.bin[4*i+0] = byte(u >> 0)
		buf.bin[4*i+1] = byte(u >> 8)
		buf.bin[4*i+2] = byte(u >> 16)
		buf.bin[4*i+3] = byte(u >> 24)
	}
	if subok {
		ctx.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, buf.bin)
	} else {
		ctx.BufferData(gl.ELEMENT_ARRAY_BUFFER, buf.bin, buf.usage)
	}
}
