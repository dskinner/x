package glw

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"os"
	"time"

	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/gl"
)

var (
	ctx    gl.Context
	logger = log.New(os.Stderr, "glw: ", 0)
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

// TODO allow package to be used by multiple contexts in parallel.
func With(glctx gl.Context) gl.Context { ctx = glctx; return glctx }

func Context() gl.Context { return ctx }

func must(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

func MustOpen(name string) asset.File {
	f, err := asset.Open(name)
	must(err)
	return f
}

// MustReadAll reads the file named by filename from assets and returns the contents or panics on error.
func MustReadAll(filename string) []byte {
	b, err := ioutil.ReadAll(MustOpen(filename))
	must(err)
	return b
}

type Sampler struct {
	Texture
	U1i
}

func (a *Sampler) Bind() {
	a.Texture.Bind()
	a.U1i.Set(int(a.Texture.Value - 1))
}

type VertexArray struct {
	gl.Attrib
	Floats FloatBuffer
	size   int
}

func (vert *VertexArray) Create(usage gl.Enum, size int, floats []float32) {
	vert.size = size
	vert.Floats.Create(usage, floats)
}

func (vert *VertexArray) Update(floats []float32) {
	vert.Floats.Update(floats)
}

func (vert VertexArray) Delete() {
	vert.Floats.Delete()
}

func (vert *VertexArray) Bind() {
	vert.Floats.Bind()
	ctx.EnableVertexAttribArray(vert.Attrib)
	ctx.VertexAttribPointer(vert.Attrib, vert.size, gl.FLOAT, false, 0, 0)
}

func (vert VertexArray) Unbind() {
	vert.Floats.Unbind()
	ctx.DisableVertexAttribArray(vert.Attrib)
}

func (vert VertexArray) Draw(mode gl.Enum) {
	vert.Floats.Draw(mode)
}

type VertexElement struct {
	gl.Attrib
	Floats FloatBuffer
	Uints  UintBuffer
	size   int
}

func (vert *VertexElement) Create(usage gl.Enum, size int, floats []float32, uints []uint32) {
	vert.size = size
	vert.Floats.Create(usage, floats)
	vert.Uints.Create(usage, uints)
}

func (vert *VertexElement) Update(floats []float32, uints []uint32) {
	vert.Floats.Update(floats)
	vert.Uints.Update(uints)
}

func (vert VertexElement) Delete() {
	vert.Floats.Delete()
	vert.Uints.Delete()
}

func (vert *VertexElement) Bind() {
	vert.Floats.Bind()
	vert.Uints.Bind()
	ctx.EnableVertexAttribArray(vert.Attrib)
	ctx.VertexAttribPointer(vert.Attrib, vert.size, gl.FLOAT, false, 0, 0)
}

func (vert VertexElement) Unbind() {
	vert.Floats.Unbind()
	vert.Uints.Unbind()
	ctx.DisableVertexAttribArray(vert.Attrib)
}

func (vert VertexElement) Draw(mode gl.Enum) {
	vert.Uints.Draw(mode)
}

type FrameBuffer struct {
	gl.Framebuffer
	tex  Texture
	rgba *image.RGBA

	maxw, maxh int
}

func (buf *FrameBuffer) Create(options ...func(*Texture)) {
	buf.Framebuffer = ctx.CreateFramebuffer()
	buf.tex.Create(options...)
	buf.rgba = &image.RGBA{}
}

func (buf *FrameBuffer) Delete() {
	ctx.DeleteFramebuffer(buf.Framebuffer)
	buf.tex.Delete()
}

func (buf *FrameBuffer) Attach() {
	ctx.BindFramebuffer(gl.FRAMEBUFFER, buf.Framebuffer)
	buf.tex.Bind()
	ctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, buf.tex.Texture, 0)
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
	ctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{0})
	buf.tex.Unbind()
}

func (buf *FrameBuffer) RGBA() *image.RGBA {
	if buf.rgba.Rect.Empty() {
		return buf.rgba
	}
	ctx.PixelStorei(gl.PACK_ALIGNMENT, 1)
	ctx.ReadPixels(buf.rgba.Pix, 0, 0, buf.rgba.Rect.Dx(), buf.rgba.Rect.Dy(), gl.RGBA, gl.UNSIGNED_BYTE)
	return buf.rgba
}

var (
	FilterNearest = TextureFilter(gl.NEAREST, gl.NEAREST)
	FilterLinear  = TextureFilter(gl.LINEAR, gl.LINEAR)
	WrapClamp     = TextureWrap(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	WrapRepeat    = TextureWrap(gl.REPEAT, gl.REPEAT)
)

func TextureFilter(min, mag int) func(*Texture) {
	return func(tex *Texture) { tex.min, tex.mag = min, mag }
}

func TextureWrap(s, t int) func(*Texture) {
	return func(tex *Texture) { tex.s, tex.t = s, t }
}

type Texture struct {
	gl.Texture
	lod      int
	min, mag int
	s, t     int
	r        image.Rectangle
}

func (tex *Texture) Create(options ...func(*Texture)) {
	tex.min, tex.mag = gl.LINEAR, gl.LINEAR
	tex.s, tex.t = gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE
	tex.Texture = ctx.CreateTexture()

	for _, opt := range options {
		opt(tex)
	}
}

func (tex *Texture) Delete() { ctx.DeleteTexture(tex.Texture) }

func (tex Texture) Bind() {
	ctx.ActiveTexture(gl.Enum(uint32(gl.TEXTURE0) + tex.Value - 1))
	ctx.BindTexture(gl.TEXTURE_2D, tex.Texture)
	ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, tex.min)
	ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, tex.mag)
	ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, tex.s)
	ctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, tex.t)
}

func (tex Texture) Unbind() { ctx.BindTexture(gl.TEXTURE_2D, gl.Texture{0}) }

func (tex *Texture) Upload(src *image.RGBA) {
	tex.r = src.Bounds()
	ctx.TexImage2D(gl.TEXTURE_2D, tex.lod, tex.r.Dx(), tex.r.Dy(), gl.RGBA, gl.UNSIGNED_BYTE, src.Pix)
}

func (tex *Texture) DrawSrc(src *image.RGBA) {
	r := src.Bounds()
	ctx.TexSubImage2D(gl.TEXTURE_2D, tex.lod, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), gl.RGBA, gl.UNSIGNED_BYTE, src.Pix)
}

func (tex Texture) GenerateMipmap() { ctx.GenerateMipmap(gl.TEXTURE_2D) }

func (tex Texture) ColorModel() color.Model { return color.RGBAModel }
func (tex Texture) Bounds() image.Rectangle { return tex.r }
func (tex Texture) At(x, y int) color.Color {
	pix := make([]uint8, 4)
	ctx.ReadPixels(pix, x, y, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE)
	return color.RGBA{pix[0], pix[1], pix[2], pix[3]}
}
