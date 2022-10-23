package glw

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unicode"

	"github.com/go-gl/gl/v4.1-core/gl"
	"golang.org/x/image/math/f32"
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

func compile(typ uint32, src string) (uint32, error) {
	shd := gl.CreateShader(typ)
	csrc, free := gl.Strs(src)
	gl.ShaderSource(shd, 1, csrc, nil)
	free()
	gl.CompileShader(shd)

	var status int32
	gl.GetShaderiv(shd, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetShaderiv(shd, gl.INFO_LOG_LENGTH, &n)
		msg := strings.Repeat("\x00", int(n+1))
		gl.GetShaderInfoLog(shd, n, nil, gl.Str(msg))
		return shd, fmt.Errorf("%s\n%s", caller("CompileShader"), msg)
	}
	return shd, nil
}

// VertSrc is vertex shader source code.
type VertSrc string

// Compile returns the compiled shader of src and error if any.
func (src VertSrc) Compile() (uint32, error) { return compile(gl.VERTEX_SHADER, string(src)) }

// FragSrc is fragment shader source code.
type FragSrc string

// Compile returns the compiled shader of src and error if any.
func (src FragSrc) Compile() (uint32, error) { return compile(gl.FRAGMENT_SHADER, string(src)) }

// VertAsset is a filename in assets containing vertex shader source code.
type VertAsset string

// Source returns the contents of the named file in assets or panics on error.
func (name VertAsset) Source() VertSrc { return VertSrc(MustReadAll(string(name))) }

// FragAsset is a filename in assets containing fragment shader source code.
type FragAsset string

// Source returns the contents of the named file in assets or panics on error.
func (name FragAsset) Source() FragSrc { return FragSrc(MustReadAll(string(name))) }

// Program identifies a compiled shader program. The bool Program.Init can be used to check if valid.
type Program struct{ Program uint32 }

// Use installs program as part of current rendering state.
func (prg Program) Use() { gl.UseProgram(prg.Program) }

// Uniform returns uniform location by name in program.
func (prg Program) Uniform(name string) UniformLocation {
	return UniformLocation{Value:gl.GetUniformLocation(prg.Program, gl.Str(name + "\x00"))}
}

// Attrib returns attribute location by name in program.
func (prg Program) Attrib(name string) AttribLocation {
	return AttribLocation{Value:uint32(gl.GetAttribLocation(prg.Program, gl.Str(name + "\x00")))}
}

// Delete frees the memory and invalidates the name associated with the program.
func (prg Program) Delete() { gl.DeleteProgram(prg.Program) }

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
	prg.Program = gl.CreateProgram()

	vshd, err := vsrc.Compile()
	if err != nil {
		return err
	}
	gl.AttachShader(prg.Program, vshd)
	defer gl.DeleteShader(vshd)

	fshd, err := fsrc.Compile()
	if err != nil {
		return err
	}
	gl.AttachShader(prg.Program, fshd)
	defer gl.DeleteShader(fshd)

	gl.LinkProgram(prg.Program)

	var status int32
	gl.GetProgramiv(prg.Program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetProgramiv(prg.Program, gl.INFO_LOG_LENGTH, &n)

		msg := strings.Repeat("\x00", int(n+1))
		gl.GetProgramInfoLog(prg.Program, n, nil, gl.Str(msg))
		return fmt.Errorf("%s\n%s", caller("LinkProgram"), msg)
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
			case AttribLocation:
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
			case UniformLocation:
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
