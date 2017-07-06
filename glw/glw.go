package glw

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"golang.org/x/image/math/f32"
	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/gl"
)

var (
	ctx    gl.Context
	logger = log.New(os.Stderr, "glw: ", 0)
)

func RGBA(c color.Color) (r, g, b, a float32) {
	ur, ug, ub, ua := c.RGBA()
	return float32(uint8(ur)) / 255, float32(uint8(ug)) / 255, float32(uint8(ub)) / 255, float32(uint8(ua)) / 255
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

func must(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

func mustOpen(name string) asset.File {
	f, err := asset.Open(name)
	must(err)
	return f
}

// mustReadAll reads from assets.
func mustReadAll(name string) []byte {
	b, err := ioutil.ReadAll(mustOpen(name))
	must(err)
	return b
}

// caller returns first file and line number outside of this package for calling
// goroutine's stack, prefixed with defaultName which may be overridden based on
// stack frames.
func caller(defaultName string) string {
	pc := make([]uintptr, 10)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])

	var (
		frame runtime.Frame
		more  bool
		name  = defaultName
		inpkg = func(s string) bool { return strings.HasPrefix(s, "dasa.cc/x/glw") }
	)

	for frame, more = frames.Next(); more && inpkg(frame.Function); frame, more = frames.Next() {
		switch frame.Function {
		case "dasa.cc/x/glw.VertSrc.Compile":
			name = "VertexShader"
		case "dasa.cc/x/glw.FragSrc.Compile":
			name = "FragmentShader"
		}
	}

	return fmt.Sprintf("%s %s:%v", name, frame.File, frame.Line)
}

func compile(typ gl.Enum, src string) (gl.Shader, error) {
	shd := ctx.CreateShader(typ)
	ctx.ShaderSource(shd, src)
	ctx.CompileShader(shd)
	if ctx.GetShaderi(shd, gl.COMPILE_STATUS) == 0 {
		return shd, fmt.Errorf("%s\n%s", caller("CompileShader"), ctx.GetShaderInfoLog(shd))
	}
	return shd, nil
}

type VertSrc string

func (src VertSrc) Compile() (gl.Shader, error) { return compile(gl.VERTEX_SHADER, string(src)) }

type FragSrc string

func (src FragSrc) Compile() (gl.Shader, error) { return compile(gl.FRAGMENT_SHADER, string(src)) }

type VertAsset string

func (name VertAsset) Source() VertSrc { return VertSrc(mustReadAll(string(name))) }

type FragAsset string

func (name FragAsset) Source() FragSrc { return FragSrc(mustReadAll(string(name))) }

type Program struct{ gl.Program }

func (prg Program) Use()                           { ctx.UseProgram(prg.Program) }
func (prg Program) Uniform(name string) gl.Uniform { return ctx.GetUniformLocation(prg.Program, name) }
func (prg Program) Attrib(name string) gl.Attrib   { return ctx.GetAttribLocation(prg.Program, name) }
func (prg Program) Delete()                        { ctx.DeleteProgram(prg.Program) }

func (prg *Program) MustBuild(vsrc VertSrc, fsrc FragSrc)           { must(prg.Build(vsrc, fsrc)) }
func (prg *Program) MustBuildAssets(vtag VertAsset, ftag FragAsset) { must(prg.BuildAssets(vtag, ftag)) }

func (prg *Program) BuildAssets(vtag VertAsset, ftag FragAsset) error {
	return prg.Build(vtag.Source(), ftag.Source())
}

func (prg *Program) Build(vsrc VertSrc, fsrc FragSrc) error {
	prg.Program = ctx.CreateProgram()

	vshd, err := vsrc.Compile()
	if err != nil {
		return err
	}
	ctx.AttachShader(prg.Program, vshd)
	defer ctx.DeleteShader(vshd)

	fshd, err := fsrc.Compile()
	if err != nil {
		return err
	}
	ctx.AttachShader(prg.Program, fshd)
	defer ctx.DeleteShader(fshd)

	ctx.LinkProgram(prg.Program)
	if ctx.GetProgrami(prg.Program, gl.LINK_STATUS) == 0 {
		return fmt.Errorf("%s\n%s", caller("LinkProgram"), ctx.GetProgramInfoLog(prg.Program))
	}

	return nil
}

func (prg Program) SetLocations(dst interface{}) {
	val := reflect.ValueOf(dst).Elem()
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if f := val.Field(i); f.CanSet() {
			name := strings.ToLower(typ.Field(i).Name)
			switch f.Interface().(type) {
			case A2fv:
				f.Set(reflect.ValueOf(A2fv(prg.Attrib(name))))
			case A3fv:
				f.Set(reflect.ValueOf(A3fv(prg.Attrib(name))))
			case A4fv:
				f.Set(reflect.ValueOf(A4fv(prg.Attrib(name))))
			case U1i:
				f.Set(reflect.ValueOf(U1i(prg.Uniform(name))))
			case U2i:
				f.Set(reflect.ValueOf(U2i(prg.Uniform(name))))
			case U3i:
				f.Set(reflect.ValueOf(U3i(prg.Uniform(name))))
			case U4i:
				f.Set(reflect.ValueOf(U4i(prg.Uniform(name))))
			case U1f:
				f.Set(reflect.ValueOf(U1f(prg.Uniform(name))))
			case U2fv:
				f.Set(reflect.ValueOf(U2fv{prg.Uniform(name), f32.Vec2{}}))
			case U3fv:
				f.Set(reflect.ValueOf(U3fv{prg.Uniform(name), f32.Vec3{}}))
			case U4fv:
				f.Set(reflect.ValueOf(U4fv{prg.Uniform(name), f32.Vec4{}, nil}))
			case U9fv:
				f.Set(reflect.ValueOf(U9fv(prg.Uniform(name))))
			case U16fv:
				f.Set(reflect.ValueOf(U16fv{prg.Uniform(name), ident16fv(), nil}))
			}
		}
	}
}

type U1i gl.Uniform

func (u U1i) Set(v int) { ctx.Uniform1i(gl.Uniform(u), v) }

type U2i gl.Uniform

func (u U2i) Set(v0, v1 int) { ctx.Uniform2i(gl.Uniform(u), v0, v1) }

type U3i gl.Uniform

func (u U3i) Set(v0, v1, v2 int32) { ctx.Uniform3i(gl.Uniform(u), v0, v1, v2) }

type U4i gl.Uniform

func (u U4i) Set(v0, v1, v2, v3 int32) { ctx.Uniform4i(gl.Uniform(u), v0, v1, v2, v3) }

type U1f gl.Uniform

func (u U1f) Set(v float32) { ctx.Uniform1f(gl.Uniform(u), v) }

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
	v f32.Vec4
	a *animator
}

func (u U4fv) Update() {
	if u.a != nil {
		u.v = u.a.pt.eval4fv()
	}
	ctx.Uniform4fv(u.Uniform, u.v[:])
}

func (u *U4fv) Set(v f32.Vec4) {
	if u.a != nil {
		u.a.pt.translate = v
	}
	u.v = v
	ctx.Uniform4fv(u.Uniform, u.v[:])
}

func (u *U4fv) Animator(options ...func(Animator)) Animator {
	if u.a == nil {
		u.a = newanimator()
	}
	u.a.apply(options...)
	return u.a
}

func (u *U4fv) Transform(transforms ...func(Transformer)) { u.Animator().Start(transforms...) }

func (u *U4fv) Stage(epoch time.Time, values ...func(Transformer)) { u.a.Stage(epoch, values...) }
func (u *U4fv) Step(now time.Time) {
	if !u.a.Step(now) {
		u.a.Cancel()
	}
	u.Update()
}

type U9fv gl.Uniform

func (u U9fv) Set(m f32.Mat3) { ctx.UniformMatrix4fv(gl.Uniform(u), m[:]) }

type U16fv struct {
	gl.Uniform
	m f32.Mat4
	a *animator
}

func (u U16fv) Inv2f(nx, ny float32) (float32, float32) {
	m := inv16fv(u.m)
	return nx*m[0] + ny*m[1], nx*m[4] + ny*m[5]
}

func (u U16fv) Update() {
	if u.a != nil {
		u.m = u.a.pt.eval16fv()
	}
	ctx.UniformMatrix4fv(u.Uniform, u.m[:])
}

func (u *U16fv) Set(m f32.Mat4) {
	u.m = m
	ctx.UniformMatrix4fv(u.Uniform, u.m[:])
}

func (u *U16fv) Ortho(l, r float32, b, t float32, n, f float32) {
	u.m = Ortho(l, r, b, t, n, f)
	u.Update()
}

func (u *U16fv) Animator(options ...func(Animator)) Animator {
	if u.a == nil {
		u.a = newanimator()
	}
	u.a.apply(options...)
	return u.a
}

func (u *U16fv) Transform(transforms ...func(Transformer)) { u.Animator().Start(transforms...) }

func (u *U16fv) Stage(epoch time.Time, values ...func(Transformer)) { u.a.Stage(epoch, values...) }
func (u *U16fv) Step(now time.Time) {
	if !u.a.Step(now) {
		u.a.Cancel()
	}
	u.Update()
}

func (u U16fv) String() string { return string16fv(u.m) }

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
func (buf FloatBuffer) Unbind()           { ctx.BindBuffer(gl.ARRAY_BUFFER, gl.Buffer{0}) }
func (buf FloatBuffer) Draw(mode gl.Enum) { ctx.DrawArrays(mode, 0, buf.count) }

func (buf *FloatBuffer) Update(data []float32) {
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
func (buf UintBuffer) Unbind()           { ctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, gl.Buffer{0}) }
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

func (buf *FrameBuffer) Attach(width, height int) {
	ctx.BindFramebuffer(gl.FRAMEBUFFER, buf.Framebuffer)
	buf.tex.Bind()
	buf.Update(width, height)
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
		buf.tex.Update(&image.RGBA{Stride: buf.maxw * 4, Rect: image.Rect(0, 0, buf.maxw, buf.maxh)})
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

func (tex *Texture) Update(src *image.RGBA) {
	r := src.Bounds()
	switch {
	case r.In(tex.r) && len(src.Pix) > 0:
		ctx.TexSubImage2D(gl.TEXTURE_2D, tex.lod, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), gl.RGBA, gl.UNSIGNED_BYTE, src.Pix)
	default:
		ctx.TexImage2D(gl.TEXTURE_2D, tex.lod, r.Dx(), r.Dy(), gl.RGBA, gl.UNSIGNED_BYTE, src.Pix)
		tex.r = r
	}
}

func (tex Texture) GenerateMipmap() { ctx.GenerateMipmap(gl.TEXTURE_2D) }
