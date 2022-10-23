package glw

import (
	"math"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

type FloatBuffer struct {
	Buffer uint32
	bin    []byte
	count  int
	usage  uint32
}

func (buf *FloatBuffer) Create(usage uint32, data []float32) {
	buf.usage = usage
	gl.GenBuffers(1, &buf.Buffer)
	buf.Bind()
	buf.Update(data)
}

func (buf FloatBuffer) Delete()          { gl.DeleteBuffers(1, &buf.Buffer) }
func (buf *FloatBuffer) Bind()           { gl.BindBuffer(gl.ARRAY_BUFFER, buf.Buffer) }
func (buf FloatBuffer) Unbind()          { gl.BindBuffer(gl.ARRAY_BUFFER, 0) }

func (buf FloatBuffer) Draw(mode uint32) {
	// TODO double check documentation for DrawArrays; last arg is number of indices, not length.
	gl.DrawArrays(mode, 0, int32(buf.count))
}

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
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(buf.bin), unsafe.Pointer(&buf.bin[0]))
	} else {
		gl.BufferData(gl.ARRAY_BUFFER, len(buf.bin), unsafe.Pointer(&buf.bin[0]), buf.usage)
	}
}

type UintBuffer struct {
	Buffer uint32
	bin    []byte
	count  int32
	usage  uint32
}

func (buf *UintBuffer) Create(usage uint32, data []uint32) {
	buf.usage = usage
	gl.GenBuffers(1, &buf.Buffer)
	buf.Bind()
	buf.Update(data)
}

func (buf UintBuffer) Delete() { gl.DeleteBuffers(1, &buf.Buffer) }
func (buf *UintBuffer) Bind()  { gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, buf.Buffer) }
func (buf UintBuffer) Unbind() { gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0) }
func (buf UintBuffer) Draw(mode uint32) {
	gl.DrawElements(mode, buf.count, gl.UNSIGNED_INT, nil)
}

func (buf *UintBuffer) Update(data []uint32) {
	buf.count = int32(len(data))
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
		gl.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, len(buf.bin), unsafe.Pointer(&buf.bin[0]))
	} else {
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(buf.bin), unsafe.Pointer(&buf.bin[0]), buf.usage)
	}
}
