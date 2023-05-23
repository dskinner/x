package glw

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"os"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

var (
	logger = log.New(os.Stderr, "glw: ", 0)

	Assets embed.FS
)

type FPS struct{ time.Time }

func (fps *FPS) Update(a time.Time) {
	dt := a.Sub(fps.Time)
	fmt.Printf("fps=%v dt=%s\n", int(time.Second/dt), dt)
	fps.Time = a
}

func RGBA(c color.Color) (r, g, b, a float32) {
	// ur, ug, ub, ua := c.RGBA()
	// return float32(uint8(ur)) / 255, float32(uint8(ug)) / 255, float32(uint8(ub)) / 255, float32(uint8(ua)) / 255
	const z = math.MaxUint16
	cr, cg, cb, ca := c.RGBA()
	return float32(cr) / z, float32(cg) / z, float32(cb) / z, float32(ca) / z
}

func genHeightmap(r image.Rectangle) (vertices []float32, indices []uint32) {
	for y := r.Min.Y; y <= r.Max.Y; y++ {
		vy := float32(y) / float32(r.Max.Y)
		for x := r.Min.X; x <= r.Max.X; x++ {
			vx := float32(x) / float32(r.Max.X)
			vertices = append(vertices, vx, vy, 0)
		}
	}
	for j := 0; j < r.Dy(); j++ {
		for i := 0; i < r.Dx(); i++ {
			k := uint32(i + (j * (r.Dx() + 1)))
			l := uint32(i + ((j + 1) * (r.Dx() + 1)))
			indices = append(indices, k, l, l+1, k, l+1, k+1)
		}
	}
	return vertices, indices
}

func must(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

func MustOpen(name string) fs.File {
	f, err := Assets.Open(name)
	must(err)
	return f
}

// MustReadAll reads the file named by filename from assets and returns the contents or panics on error.
func MustReadAll(filename string) []byte {
	b, err := ioutil.ReadAll(MustOpen(filename))
	must(err)
	return b
}

// EnableDefaultVertexArrayObject generates and binds a single VAO as required by OpenGL3.3+.
func EnableDefaultVertexArrayObject() {
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
}

type Sampler struct {
	Texture
	U1i
}

func (a *Sampler) Bind() {
	a.Texture.Bind()
	a.U1i.Set(int32(a.Texture.Texture - 1))
}

type VertexArray struct {
	AttribLocation
	Floats FloatBuffer
	Size   int
	Stride int
	Offset int
}

// TODO deprecate
/*
should change uses as follows basically:
	// x, y, z, u, v
	obj.Vert.Floats.Create(gl.STATIC_DRAW, []float32{
		-1, -1, -1, 1, 1,
		-1, +1, -1, 1, 0,
		+1, +1, -1, 0, 0,
		+1, -1, -1, 0, 1,
	})
	obj.Vert.Uints.Create(gl.STATIC_DRAW, []uint32{0, 1, 2, 0, 2, 3})
	obj.Vert.StepSize(3, 5, 0)
	obj.Vert.Bind()

	obj.TexCoord.Floats = obj.Vert.Floats
	obj.TexCoord.StepSize(2, 5, 3)
	obj.TexCoord.Bind()
*/
func (vert *VertexArray) Create(usage uint32, size int, floats []float32) {
	vert.Size = size
	vert.Floats.Create(usage, floats)
}

// TODO deprecate
func (vert *VertexArray) Update(floats []float32) {
	vert.Floats.Update(floats)
}

func (vert *VertexArray) StepSize(size, stride, offset int) {
	vert.Size = size
	vert.Stride = stride
	vert.Offset = offset
}

func (vert VertexArray) Delete() {
	vert.Floats.Delete()
}

func (vert *VertexArray) Bind() {
	vert.Floats.Bind()
	gl.EnableVertexAttribArray(vert.Value)
	gl.VertexAttribPointerWithOffset(vert.Value, int32(vert.Size), gl.FLOAT, false, int32(vert.Stride)*4, uintptr(vert.Offset*4))
}

func (vert VertexArray) Unbind() {
	vert.Floats.Unbind()
	gl.DisableVertexAttribArray(vert.Value)
}

func (vert VertexArray) Draw(mode uint32) {
	vert.Floats.Draw(mode)
}

type VertexElement struct {
	AttribLocation
	Floats FloatBuffer
	Uints  UintBuffer
	Size   int
	Stride int
	Offset int
}

// TODO deprecate
func (vert *VertexElement) Create(usage uint32, size int, offset int, floats []float32, uints []uint32) {
	vert.Size = size
	vert.Offset = offset
	vert.Floats.Create(usage, floats)
	vert.Uints.Create(usage, uints)
}

// TODO deprecate
func (vert *VertexElement) Update(floats []float32, uints []uint32) {
	vert.Floats.Update(floats)
	vert.Uints.Update(uints)
}

func (vert *VertexElement) StepSize(size, stride, offset int) {
	vert.Size = size
	vert.Stride = stride
	vert.Offset = offset
}

func (vert VertexElement) Delete() {
	vert.Floats.Delete()
	vert.Uints.Delete()
}

func (vert *VertexElement) Bind() {
	vert.Floats.Bind()
	vert.Uints.Bind()
	gl.EnableVertexAttribArray(vert.Value)
	gl.VertexAttribPointerWithOffset(vert.Value, int32(vert.Size), gl.FLOAT, false, int32(vert.Stride)*4, uintptr(vert.Offset*4))
}

func (vert VertexElement) Unbind() {
	vert.Floats.Unbind()
	vert.Uints.Unbind()
	gl.DisableVertexAttribArray(vert.Value)
}

func (vert VertexElement) Draw(mode uint32) {
	vert.Uints.Draw(mode)
}

type FrameBuffer struct {
	Framebuffer uint32
	tex         Texture
	rgba        *image.RGBA

	maxw, maxh int
}

func (buf *FrameBuffer) Create(options ...func(*Texture)) {
	gl.CreateFramebuffers(1, &buf.Framebuffer)
	buf.tex.Create(options...)
	buf.rgba = &image.RGBA{}
}

func (buf *FrameBuffer) Delete() {
	gl.DeleteFramebuffers(1, &buf.Framebuffer)
	buf.tex.Delete()
}

func (buf *FrameBuffer) Attach() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, buf.Framebuffer)
	buf.tex.Bind()
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, buf.tex.Texture, 0)
}

func (buf *FrameBuffer) Update(width, height int) {
	n := 4 * width * height
	a := cap(buf.rgba.Pix)
	if a > n {
		buf.rgba.Pix = buf.rgba.Pix[:n]
	} else {
		buf.rgba.Pix = append(buf.rgba.Pix[:a], make([]uint8, n-a)...)
	}
	buf.rgba.Stride = 4 * width
	buf.rgba.Rect = image.Rect(0, 0, width, height)

	var grow, groh bool
	if grow = buf.maxw < width; grow {
		buf.maxw = width
	}
	if groh = buf.maxh < height; groh {
		buf.maxh = height
	}

	if grow || groh {
		src := &image.RGBA{Stride: buf.maxw * 4, Rect: image.Rect(0, 0, buf.maxw, buf.maxh)}
		if src.Bounds().In(buf.tex.Bounds()) && len(src.Pix) > 0 {
			buf.tex.DrawSrc(src)
		} else {
			buf.tex.Upload(src)
		}
	}
}

func (buf *FrameBuffer) Detach() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	buf.tex.Unbind()
}

func (buf *FrameBuffer) RGBA() *image.RGBA {
	if buf.rgba.Rect.Empty() {
		return buf.rgba
	}
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadPixels(0, 0, int32(buf.rgba.Rect.Dx()), int32(buf.rgba.Rect.Dy()), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&buf.rgba.Pix[0]))
	return buf.rgba
}

var (
	FilterNearest = TextureFilter(gl.NEAREST, gl.NEAREST)
	FilterLinear  = TextureFilter(gl.LINEAR, gl.LINEAR)
	WrapClamp     = TextureWrap(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	WrapRepeat    = TextureWrap(gl.REPEAT, gl.REPEAT)
)

func TextureFilter(min, mag int32) func(*Texture) {
	return func(tex *Texture) { tex.min, tex.mag = min, mag }
}

func TextureWrap(s, t int32) func(*Texture) {
	return func(tex *Texture) { tex.s, tex.t = s, t }
}

type Texture struct {
	Texture  uint32
	lod      int32
	min, mag int32
	s, t     int32
	r        image.Rectangle
}

func (tex *Texture) Create(options ...func(*Texture)) {
	tex.min, tex.mag = gl.LINEAR, gl.LINEAR
	tex.s, tex.t = gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE
	gl.GenTextures(1, &tex.Texture)

	for _, opt := range options {
		opt(tex)
	}
}

func (tex *Texture) Delete() { gl.DeleteTextures(1, &tex.Texture) }

func (tex Texture) Bind() {
	gl.ActiveTexture(uint32(uint32(gl.TEXTURE0) + tex.Texture - 1))
	gl.BindTexture(gl.TEXTURE_2D, tex.Texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, tex.min)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, tex.mag)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, tex.s)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, tex.t)
}

func (tex Texture) Unbind() { gl.BindTexture(gl.TEXTURE_2D, 0) }

func (tex *Texture) Upload(src *image.RGBA) {
	tex.r = src.Bounds()
	gl.TexImage2D(gl.TEXTURE_2D, tex.lod, gl.RGBA, int32(tex.r.Dx()), int32(tex.r.Dy()), 0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&src.Pix[0]))
}

func (tex *Texture) DrawSrc(src *image.RGBA) {
	r := src.Bounds()
	gl.TexSubImage2D(gl.TEXTURE_2D, tex.lod, int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy()), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&src.Pix[0]))
}

func (tex Texture) GenerateMipmap() { gl.GenerateMipmap(gl.TEXTURE_2D) }

func (tex Texture) ColorModel() color.Model { return color.RGBAModel }
func (tex Texture) Bounds() image.Rectangle { return tex.r }
func (tex Texture) At(x, y int) color.Color {
	pix := make([]uint8, 4)
	gl.ReadPixels(int32(x), int32(y), 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&pix[0]))
	return color.RGBA{pix[0], pix[1], pix[2], pix[3]}
}
