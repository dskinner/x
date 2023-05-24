package glw

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unicode"

	"golang.org/x/image/math/f32"
	"golang.org/x/mobile/gl"
)

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

// VertSrc is vertex shader source code.
type VertSrc string

// Compile returns the compiled shader of src and error if any.
func (src VertSrc) Compile() (gl.Shader, error) { return compile(gl.VERTEX_SHADER, string(src)) }

// FragSrc is fragment shader source code.
type FragSrc string

// Compile returns the compiled shader of src and error if any.
func (src FragSrc) Compile() (gl.Shader, error) { return compile(gl.FRAGMENT_SHADER, string(src)) }

// VertAsset is a filename in assets containing vertex shader source code.
type VertAsset string

// Source returns the contents of the named file in assets or panics on error.
func (name VertAsset) Source() VertSrc { return VertSrc(MustReadAll(string(name))) }

// FragAsset is a filename in assets containing fragment shader source code.
type FragAsset string

// Source returns the contents of the named file in assets or panics on error.
func (name FragAsset) Source() FragSrc { return FragSrc(MustReadAll(string(name))) }

// Program identifies a compiled shader program. The bool Program.Init can be used to check if valid.
type Program struct{ gl.Program }

// Use installs program as part of current rendering state.
func (prg Program) Use() { ctx.UseProgram(prg.Program) }

// Uniform returns uniform location by name in program.
func (prg Program) Uniform(name string) gl.Uniform { return ctx.GetUniformLocation(prg.Program, name) }

// Attrib returns attribute location by name in program.
func (prg Program) Attrib(name string) gl.Attrib { return ctx.GetAttribLocation(prg.Program, name) }

// Delete frees the memory and invalidates the name associated with the program.
func (prg Program) Delete() { ctx.DeleteProgram(prg.Program) }

// MustBuild is a helper that wraps Program.Build and panics on error.
func (prg *Program) MustBuild(vsrc VertSrc, fsrc FragSrc) { must(prg.Build(vsrc, fsrc)) }

// MustBuildAssets is a helper that wraps Program.BuildAssets and panics on error.
func (prg *Program) MustBuildAssets(vtag VertAsset, ftag FragAsset) {
	must(prg.BuildAssets(vtag, ftag))
}

// BuildAssets is a helper that wraps Program.Build with asset file contents of filenames vtag and ftag.
func (prg *Program) BuildAssets(vtag VertAsset, ftag FragAsset) error {
	return prg.Build(vtag.Source(), ftag.Source())
}

// Build compiles shaders and links program.
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

// Install is a helper that wraps Program.Build and Program.Use.
func (prg *Program) Install(vsrc VertSrc, fsrc FragSrc) error {
	if err := prg.Build(vsrc, fsrc); err != nil {
		return err
	}
	prg.Use()
	return nil
}

// Unmarshal recursively sets fields of dst for recognized types.
func (prg Program) Unmarshal(dst interface{}) {
	var val reflect.Value
	if v, ok := dst.(reflect.Value); ok {
		val = v
	} else {
		val = reflect.ValueOf(dst).Elem()
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if f := val.Field(i); f.CanSet() {
			p := []rune(typ.Field(i).Name)
			p[0] = unicode.ToLower(p[0])
			name := string(p)
			switch f.Interface().(type) {
			case gl.Attrib:
				f.Set(reflect.ValueOf(prg.Attrib(name)))
			case VertexArray:
				f.Field(0).Set(reflect.ValueOf(prg.Attrib(name)))
			case VertexElement:
				f.Field(0).Set(reflect.ValueOf(prg.Attrib(name)))
			case Sampler:
				f.Field(1).Set(reflect.ValueOf(U1i(prg.Uniform(name))))
			case A2fv:
				f.Set(reflect.ValueOf(A2fv(prg.Attrib(name))))
			case A3fv:
				f.Set(reflect.ValueOf(A3fv(prg.Attrib(name))))
			case A4fv:
				f.Set(reflect.ValueOf(A4fv(prg.Attrib(name))))
			case gl.Uniform:
				f.Set(reflect.ValueOf(prg.Uniform(name)))
			case U1i:
				f.Set(reflect.ValueOf(U1i(prg.Uniform(name))))
			case U2i:
				f.Set(reflect.ValueOf(U2i(prg.Uniform(name))))
			case U3i:
				f.Set(reflect.ValueOf(U3i(prg.Uniform(name))))
			case U4i:
				f.Set(reflect.ValueOf(U4i(prg.Uniform(name))))
			case U1f:
				u := U1f{}
				u.uniform = newuniform(prg.Uniform(name), u.Update)
				f.Set(reflect.ValueOf(u))
			case U2fv:
				f.Set(reflect.ValueOf(U2fv{prg.Uniform(name), f32.Vec2{}}))
			case U3fv:
				f.Set(reflect.ValueOf(U3fv{prg.Uniform(name), f32.Vec3{}}))
			case U4fv:
				u := U4fv{prg.Uniform(name), 0, f32.Vec4{}, nil}
				f.Set(reflect.ValueOf(u))
			case U9fv:
				f.Set(reflect.ValueOf(U9fv(prg.Uniform(name))))
			case U16fv:
				u := U16fv{}
				u.uniform = newuniform(prg.Uniform(name), u.Update)
				f.Set(reflect.ValueOf(u))
			default:
				if f.Kind() == reflect.Struct {
					prg.Unmarshal(f)
				}
			}
		}
	}
}
