// +build plot

package gma

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sync"
	"testing"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func (a Multivector) MetricPlane() (x0, y0, x1, y1 float64) {
	for _, v := range a {
		switch v.Basis {
		case E1:
			x0 += v.Scalar
		case E2:
			y0 += v.Scalar
		case I2.Basis:
			// generic area, see TODO.
			x1 += v.Scalar
			y1 += v.Scalar
		}
	}
	return x0, y0, x0 + x1, y0 + y1
}

type plttr struct {
	*plot.Plot
	nlines int
}

func newplttr() *plttr {
	p := plot.New()
	p.X.Min, p.X.Max = -5, 5
	p.Y.Min, p.Y.Max = -5, 5
	p.Add(plotter.NewGrid())
	return &plttr{Plot: p}
}

func (p *plttr) addPlane(lbl string, base, u, v Multivector) {
	w := u.Add(v)
	r, err := plotter.NewPolygon(plotter.XYs{
		{base.E1(), base.E2()},
		{u.E1(), u.E2()},
		{w.E1(), w.E2()},
		{v.E1(), v.E2()},
	})
	if err != nil {
		panic(err)
	}
	r.Color = plotutil.Color(p.nlines)
	p.nlines++
	p.Add(r)
	p.Legend.Add(lbl, r)
}

func (p *plttr) addLine(lbl string, a Multivector) {
	ln, err := plotter.NewLine(plotter.XYs{
		{0, 0},
		{a.E1(), a.E2()},
	})
	if err != nil {
		panic(err)
	}

	ln.LineStyle.Width = vg.Points(1)
	ln.LineStyle.Color = plotutil.Color(p.nlines)
	p.nlines++

	p.Add(ln)
	p.Legend.Add(lbl, ln)
}

func (p *plttr) save(fname string) {
	if err := p.Save(8*vg.Inch, 8*vg.Inch, fname); err != nil {
		panic(err)
	}
}

const Degree = math.Pi / 180

// TODO for 2D, no need to sandwhich vector; see 7.3.1
func rotate(a Multivector, angle float64, t *testing.T) Multivector {
	// e1 := Blade{bitmap: 1, s: 1}
	// e2 := Blade{bitmap: 1 << 1, s: 1}
	// I := e1.Op(e2)
	// _ = I

	// A := Multivector{e1}
	// B := Multivector{e1, e2}

	// A := Multivector{Blade{e1.bitmap, 1}, Blade{e2.bitmap, 0}}
	// B := Multivector{Blade{e1.bitmap, 0}, Blade{e2.bitmap, 1}}
	// t.Logf(" A: %s", A)
	// t.Logf("AI: %s", A.Inverse())
	// t.Logf("A.Angle(B): %v", A.Angle(B))
	// R := B.Mul(A.Inverse())

	// RN := B.Mul(B).Mul(A.Mul(A).Inverse()).Norm()
	// R[0].s /= RN
	// R[1].s /= RN

	// R := Multivector{
	// Blade{s: math.Cos(angle / 2)},
	// Blade{bitmap: I.bitmap, s: -math.Sin(angle / 2)},
	// }

	R := Rotor(angle, I2.Basis)

	t.Logf("   R: %s", R)
	t.Logf(" |R|: %v", R.NormSq())
	t.Logf("  R~: %s", R.Rev())
	t.Logf("  Ri: %s", R.Inverse())
	t.Logf(" RR~: %s", R.Mul(R.Rev()))
	res := R.Mul(a).Mul(R.Rev())
	t.Logf("   a: %s", a)
	t.Logf("  Ra: %s", R.Mul(a))
	t.Logf("RaR~: %s", res)
	t.Log("---")

	return res
}

func TestPlot(t *testing.T) {
	p := newplttr()
	// p.addArea("e1^e2", 0, 0, 1, 1)

	// e1 := Blade{bitmap: 1, s: 1}
	// e2 := Blade{bitmap: 1 << 1, s: 1}
	// I := e1.Op(e2)
	// _ = I

	p.addLine("e1", Multivector{e1})
	p.addLine("e2", Multivector{e2})

	// p.addLine("e1+e2", Multivector{e1, e2})

	// b := Multivector{e1, e2.Mul(Blade{s: -0.5})}
	// p.addLine("e1+0.5*e2", b)

	u1 := e1.Mul(Blade{Scalar: 3})
	// u2 := e2.Mul(Blade{s: 0})
	u := Multivector{u1}
	p.addLine("u", u)
	// p.addLine("u1", rot(u, 1))
	// p.addLine("u1", rot(u, 2))
	v := rotate(u, 45*Degree, t)
	p.addLine("v", v)

	w := rotate(v, 45*Degree, t)
	p.addLine("w", w)

	R := v.Mul(u.Inverse())
	x := R.Mul(w).Mul(R.Rev())
	p.addLine("x", x)

	// uI := u.Mul(Multivector{I})
	// p.addLine("uI", uI)

	// uIv := u.Mul(Multivector{I.Rev()})
	// p.addLine("uIv", uIv)

	p.save("lines.png")
}

func TestPlot2(t *testing.T) {
	p := newplttr()
	// p.addArea("e1^e2", 0, 0, 1, 1)

	// e1 := Blade{bitmap: 1, s: 1}
	// e2 := Blade{bitmap: 1 << 1, s: 1}
	// I := e1.Op(e2)
	// _ = I

	// p.addLine("e1^e2", Multivector{I})

	p.addLine("e1", Multivector{e1})
	p.addLine("e2", Multivector{e2})

	// u := Multivector{e1, e2}
	// p.addLine("u = e1+e2", u)

	v := Multivector{e1, {0.5, E2}}
	p.addLine("v = e1+0.5*e2", v)

	p.addPlane("v^e2", Multivector{}, v, Multivector{e2})

	//
	// _3e1 := Blade{bitmap: 1, s: 3}
	// _3e1e2 := Multivector{_3e1.Op(e2)}

	// u_3e1e2 := u.Add(_3e1e2)
	// p.addLine("u + 3e1^e2", u_3e1e2)

	// v_3e1e2 := v.Add(_3e1e2)
	// p.addLine("v + 3e1^e2", v_3e1e2)

	r_v := rotate(v, 90*Degree, t)
	p.addLine("RvR~", r_v)

	p.addPlane("(RvR~)^e2", Multivector{}, r_v, Multivector{e2})

	// r_v_3e1e2 := rotate(v_3e1e2, 45*Degree, t)
	// p.addLine("R(v+3e1^e2)R~", r_v_3e1e2)

	// r_3e1e2 := rotate(_3e1e2, 45*Degree, t)
	// p.addLine("R(3e1^e2)R~", r_3e1e2)

	p.save("lines2.png")
}

func saveImage(m image.Image, p string) {
	out, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := png.Encode(out, m); err != nil {
		panic(err)
	}
}

func TestJuliaFractal(t *testing.T) {
	e1 := Multivector{{1, E1}}

	const width, height = 400, 300
	const zoom = 0.007

	// const width, height = 800, 600
	// const zoom = 0.0035

	const maxiter = 60

	// c := Multivector{{-0.8, E1}, {0.156, E2}}
	// c := Multivector{{-0.835, E1}, {-0.2321, E2}}
	c := Multivector{{-0.70176, E1}, {-0.3842, E2}}

	r := image.Rect(-width/2, -height/2, width/2, height/2)
	m := image.NewRGBA(r)

	var wg sync.WaitGroup

	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {

			wg.Add(1)

			go func(x, y int) {
				clr := color.RGBA{A: 255}

				p := Multivector{
					{float64(x) * zoom, E1},
					{float64(y) * zoom, E2},
				}

				for clr.G = 0; clr.G < maxiter; clr.G++ {
					p = p.Mul(e1).Mul(p).Add(c)
					if p.NormSq() > 1e4 {
						clr.R = clr.G * 2
						break
					}
				}
				clr.G *= uint8(255 / maxiter)

				// n := p.Norm()/10 + 0.5
				// if n > 255 {
				// n = 255
				// }
				// clr.B = uint8(n / 8)

				m.Set(x, y, clr)
				wg.Done()
			}(x, y)
		}
	}

	wg.Wait()
	saveImage(m, "julia.png")
}
