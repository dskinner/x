package main

import (
	"image"
	"image/color"
	"image/draw"

	"dasa.cc/x/glw"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/math/f32"
	"golang.org/x/image/math/fixed"
)

var (
	monobold = mustParseTTF(gomonobold.TTF)

	monobold16 = NewFace(monobold, Size(16), Hinting(font.HintingFull))
	monobold18 = NewFace(monobold, Size(18), Hinting(font.HintingFull))
	monobold20 = NewFace(monobold, Size(20), Hinting(font.HintingFull))

	monobold22 = NewFace(monobold, Size(22), Hinting(font.HintingFull))

	defaultFace = monobold16
)

func mustParseTTF(ttf []byte) *truetype.Font {
	f, err := truetype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	return f
}

func Size(x float64) func(*truetype.Options) {
	return func(a *truetype.Options) {
		a.Size = x
	}
}

func Hinting(x font.Hinting) func(*truetype.Options) {
	return func(a *truetype.Options) {
		a.Hinting = x
	}
}

func NewFace(fnt *truetype.Font, opts ...func(*truetype.Options)) font.Face {
	o := &truetype.Options{}
	for _, opt := range opts {
		opt(o)
	}
	return truetype.NewFace(fnt, o)
}

type Drawer struct {
	src  image.Image
	pos  image.Point
	face font.Face
}

func (d *Drawer) TranslateTo(pt image.Point) { d.pos = pt }
func (d *Drawer) SetColor(clr color.Color)   { d.src = image.NewUniform(clr) }
func (d *Drawer) SetFace(face font.Face)     { d.face = face }

func (d *Drawer) MeasureString(s string) image.Rectangle {
	adv := font.MeasureString(d.face, s).Ceil()
	asc := d.face.Metrics().Ascent.Ceil()
	return image.Rect(0, 0, adv, asc)
}

func (d *Drawer) DrawString(dst *image.RGBA, s string) {
	dr := font.Drawer{
		Dst:  dst,
		Src:  d.src,
		Face: d.face,
		Dot: fixed.Point26_6{
			X: fixed.I(d.pos.X),
			Y: fixed.I(d.pos.Y + d.face.Metrics().Ascent.Ceil()),
		},
	}
	dr.DrawString(s)
}

func (d *Drawer) Draw(dst *image.RGBA, r image.Rectangle, src image.Image, op draw.Op) {
	draw.Draw(dst, r.Add(d.pos), src, image.ZP, op)
}

func Fit(dst, src image.Point) glw.Transform {
	tr := glw.TransformIdent()
	var x, y float32
	ndst := float32(dst.X) / float32(dst.Y)
	nsrc := float32(src.X) / float32(src.Y)
	if nsrc > ndst {
		x, y = ndst, ndst/nsrc
	} else {
		x, y = nsrc, 1
	}
	tr.ScaleTo(f32.Vec3{x, y, 1})
	return tr
}
