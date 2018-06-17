package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dasa.cc/x/glw"

	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/exp/shiny/driver/gldriver"
	"golang.org/x/exp/shiny/materialdesign/colornames"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/exp/shiny/widget/theme"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/math/f32"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/gl"
)

const (
	Kilobyte ByteUnit = 1e3
	Megabyte ByteUnit = 1e6
	Gigabyte ByteUnit = 1e9
)

type ByteUnit int

func (b ByteUnit) String() string {
	switch {
	case b < Kilobyte:
		return fmt.Sprintf("%vB", b)
	case b < Megabyte:
		return fmt.Sprintf("%.1fkB", float64(b)/float64(Kilobyte))
	case b < Gigabyte:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(Megabyte))
	default:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(Gigabyte))
	}
}

var (
	viewport image.Point
	store    = NewStore()
	title    = "giv"

	flagOne     = flag.Bool("one", false, "operates one singular instance, passing args to one if already running")
	flagFavs    = flag.String("favs", "", "folder to copy file to on keypress B")
	flagDur     = flag.String("dur", "", "only load images after now+dur, such as -12h")
	flagSort    = flag.Bool("sort", false, "sort by mod time")
	flagRev     = flag.Bool("rev", false, "when used with sort, reverse results")
	flagLogfile = flag.String("logfile", "", "specify a file path to write log output to")
	flagIndex   = flag.Int("index", 0, "initial index of image to view")
)

var ErrForfeit = errors.New("forfeited args to running instance")

func init() {
	flag.Parse()
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}
	log.SetPrefix(title + ": ")
	// log.SetFlags(0)

	theme.Default = &theme.Theme{
		Palette: &theme.Palette{
			theme.Light:      image.Uniform{colornames.BlueGrey100},
			theme.Neutral:    image.Uniform{colornames.BlueGrey500},
			theme.Dark:       image.Uniform{colornames.BlueGrey900},
			theme.Accent:     image.Uniform{colornames.DeepOrangeA200},
			theme.Foreground: image.Uniform{colornames.BlueGrey900},
			theme.Background: image.Uniform{colornames.BlueGrey500},
		},
	}
}

func AbsArgs() []string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Getwd failed:", err)
	}
	args := flag.Args()
	for i, p := range args {
		if !filepath.IsAbs(p) {
			args[i] = filepath.Join(wd, p)
		}
	}
	return args
}

func isFav(x string) bool {
	if *flagFavs == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(*flagFavs, filepath.Base(x)))
	return err == nil
}

func main() {
	flag.Parse()
	if *flagOne {
		if err := tryListenAndServe(); err != nil {
			data := make(url.Values)
			data["args"] = AbsArgs()
			if _, err := http.PostForm("http://localhost:6177/", data); err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}
	}

	if err := store.Walk(*flagIndex, flag.Args()...); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	if *flagLogfile != "" {
		logfile, err := os.OpenFile(*flagLogfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening logfile: %v", err)
		}
		defer logfile.Close()
		log.SetOutput(logfile)
	}

	gldriver.Main(shinyMain)
}

type sampler struct {
	glw.Program
	Proj     glw.U16fv
	Model    glw.U16fv
	Vertex   glw.VertexElement
	Texcoord glw.VertexArray
	Sampler  glw.Sampler
}

func (a *sampler) init() {
	a.Install(vsrc, fsrc)
	a.Unmarshal(a)
	a.Vertex.Create(gl.STATIC_DRAW, 3, box3fv, box3iv)
	a.Texcoord.Create(gl.STATIC_DRAW, 2, tex2fv)
	a.Sampler.Create()
}

func (a *sampler) ortho(l, r, b, t, n, f float32) {
	a.Use()
	a.Proj.Ortho(l, r, b, t, n, f)
}

// don't forget to call sampler.Use first!
func (a *sampler) draw() {
	a.Sampler.Bind()
	a.Texcoord.Bind()
	a.Vertex.Bind()
	a.Vertex.Draw(gl.TRIANGLES)
}

var overlay struct {
	sampler
	dst *image.RGBA
	m   image.Rectangle
}

var gallery struct {
	sampler
	evm  mouse.Event
	view View
}

func glinit(ctx gl.Context) {
	ctx = glw.With(ctx)

	gallery.init()
	gallery.Model.Animator(glw.Duration(250 * time.Millisecond))

	overlay.init()
	overlay.ortho(-1, 1, -1, 1, 0, 10)
	overlay.Model.Update()
}

func glresize(width, height int) {
	viewport = image.Pt(width, height)
	ctx := glw.Context()
	ctx.Disable(gl.CULL_FACE)
	ctx.Disable(gl.DEPTH_TEST)
	ctx.Enable(gl.BLEND)
	ctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	ctx.ClearColor(glw.RGBA(theme.Default.Palette.Background()))
	ctx.Viewport(0, 0, width, height)
	if height != 0 {
		ar := float32(width) / float32(height)
		gallery.ortho(-ar, ar, -1, 1, 0.0, 10.0)

		overlay.Use()
		overlay.dst = image.NewRGBA(image.Rectangle{Max: viewport})
		overlay.Sampler.Bind()
		overlay.Sampler.Upload(overlay.dst)
	}
}

func gldraw() {
	if viewport == image.ZP {
		return
	}

	now := time.Now()
	ctx := glw.Context()
	ctx.Clear(gl.COLOR_BUFFER_BIT)

	gallery.Use()
	if !cycler.done {
		cycleView()
	}
	if gallery.Model.Step(now) {
		glwidget.Mark(node.MarkNeedsPaintBase)
	}
	gallery.draw()

	overlay.Use()
	draw.Draw(overlay.dst, overlay.m, image.Transparent, image.ZP, draw.Src)
	overlay.m = drawLabel(overlay.dst)
	overlay.Sampler.Bind()
	overlay.Sampler.Upload(overlay.dst)
	overlay.draw()

	// post draw
	if gesturer.pending {
		handleGesture(now)
	}
	if stateActive() {
		gallery.Model.Stage(now, transforms()...)
		glwidget.Mark(node.MarkNeedsPaintBase)
	}
}

func glinput(e interface{}) {
	switch e := e.(type) {
	case mouse.Event:
		// fmt.Printf("%#v\n", e)
		mousestate[e.Button] = e.Direction
		if mousestate[mouse.ButtonLeft] == mouse.DirPress {
			gesturer.emouse = e
			gesturer.pending = true
			glwidget.Mark(node.MarkNeedsPaintBase)
		}
	case key.Event:
		state.mods = e.Modifiers
		if p, ok := keymap[e.Code]; ok {
			if p.Cond == nil || p.Cond(e) {
				switch fn := p.Func.(type) {
				case func():
					fn()
				case func(key.Event):
					fn(e)
				}
				glwidget.Mark(node.MarkNeedsPaintBase)
			}
		}
	}
}

var gesturer struct {
	emouse  mouse.Event
	pending bool
}

func handleGesture(now time.Time) {
	e := gesturer.emouse
	x, y := glw.Uton(e.X/float32(viewport.X)), glw.Uton(e.Y/float32(viewport.Y))
	x, y = gallery.Proj.Inv2f(x, y)
	gallery.Model.Stage(now, glw.TranslateTo(f32.Vec4{x, y, 0, 0}))
	glwidget.Mark(node.MarkNeedsPaintBase)
	gesturer.pending = false
}

var cycler struct {
	view   View
	stride int
	done   bool
}

func Cycle(stride int) {
	if cycler.done {
		cycler.view = gallery.view
		cycler.stride = stride
		cycler.done = false
	}
}

func cycleView() {
	// TODO gallery.Mutex instead of storing view on cycler ???
	view := cycler.view
	view.transform = gallery.Model.Animator().At()
	view = store.Cycle(view, cycler.stride)
	if view.err != nil {
		// TODO upload uniform color to Texture and return early ???
		fmt.Printf("%v: %s\n", view.err, view.name)
	}

	if view.name != cycler.view.name {
		pix := store.Pix()
		rgba := &image.RGBA{pix, 4 * view.config.Width, image.Rect(0, 0, view.config.Width, view.config.Height)}
		gallery.Sampler.Bind()
		gallery.Sampler.Upload(rgba)
	}

	if view.transform == (glw.Transform{}) {
		tr := Fit(viewport, gallery.Sampler.Bounds().Size())
		gallery.Model.Stage(time.Time{}, glw.To(tr))
	} else {
		gallery.Model.Stage(time.Time{}, glw.To(view.transform))
	}

	gallery.view = view
	cycler.done = true
}

var labelState int

func drawLabel(dst *image.RGBA) image.Rectangle {
	lbl := ""
	switch labelState {
	case 1:
		lbl = fmt.Sprintf("%s %s", title, store)
	case 2:
		lbl = fmt.Sprintf("%s %s %s", title, store, filepath.Base(gallery.view.name))
	}
	if isFav(gallery.view.name) {
		lbl += " ♥"
	}

	lbl = strings.TrimSpace(lbl)
	if lbl == "" {
		return image.ZR
	}

	drw := &Drawer{}
	drw.SetFace(monobold16)
	r := drw.MeasureString(lbl).Inset(-7)
	drw.TranslateTo(image.Pt(14, 14))
	drw.Draw(dst, r, image.NewUniform(color.NRGBA{0, 0, 0, 0x7f}), draw.Src)
	drw.SetColor(color.NRGBA{0xff, 0xff, 0xff, 0xd8})
	drw.DrawString(dst, lbl)
	return r.Add(drw.pos)
}

var state struct {
	panLeft, panRight, panUp, panDown bool
	rotateLeft, rotateRight           bool
	scaleUp, scaleDown                bool

	mods key.Modifiers
	invf float32
}

func stateActive() bool {
	return state.panLeft || state.panRight || state.panUp || state.panDown ||
		state.rotateLeft || state.rotateRight ||
		state.scaleUp || state.scaleDown
}

func transforms() []func(glw.Transformer) {
	var p []func(glw.Transformer)
	x := float32(0.1)
	if state.mods&key.ModShift != 0 {
		x = 0.04
	}
	if state.invf == 0 {
		state.invf = 1
	}
	if state.panLeft {
		p = append(p, glw.TranslateBy(f32.Vec4{-x * state.invf, 0, 0, 0}))
	}
	if state.panRight {
		p = append(p, glw.TranslateBy(f32.Vec4{+x * state.invf, 0, 0, 0}))
	}
	if state.panUp {
		p = append(p, glw.TranslateBy(f32.Vec4{0, -x * state.invf, 0, 0}))
	}
	if state.panDown {
		p = append(p, glw.TranslateBy(f32.Vec4{0, +x * state.invf, 0, 0}))
	}
	if state.rotateLeft {
		p = append(p, glw.RotateBy(-x, f32.Vec3{0, 0, 1}))
	}
	if state.rotateRight {
		p = append(p, glw.RotateBy(+x, f32.Vec3{0, 0, 1}))
	}
	if state.scaleUp {
		p = append(p, glw.ScaleBy(f32.Vec4{1 + x, 1 + x, 1, 1}))
	}
	if state.scaleDown {
		p = append(p, glw.ScaleBy(f32.Vec4{1 - x, 1 - x, 1, 1}))
	}
	return p
}

var (
	tex2fv = []float32{
		-0, -0,
		-0, +1,
		+1, +1,
		+1, -0,
	}
	box3fv = []float32{
		-1, -1, 0,
		-1, +1, 0,
		+1, +1, 0,
		+1, -1, 0,
	}
	box3iv = []uint32{0, 1, 2, 0, 2, 3}
)

const (
	vsrc = `#version 100
uniform mat4 proj;
uniform mat4 model;
attribute vec4 vertex;
attribute vec2 texcoord;
varying vec2 vtexcoord;
void main() {
  gl_Position = proj*model*vertex;
  vtexcoord = texcoord;
}`

	fsrc = `#version 100
precision mediump float;
uniform vec4 color;
uniform sampler2D sampler;
varying vec2 vtexcoord;
void main() {
  //gl_FragColor = color;
  gl_FragColor = texture2D(sampler, vtexcoord.xy);
}`
)
