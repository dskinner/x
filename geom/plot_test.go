// +build plot

package geom

import (
	"math"
	"testing"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type plttr struct {
	*plot.Plot
	nlines int
}

func newplttr() *plttr {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.X.Min, p.X.Max = -5, 5
	p.Y.Min, p.Y.Max = -5, 5
	p.Add(plotter.NewGrid())
	return &plttr{Plot: p}
}

func (p *plttr) addLine(lbl string, x, y float64) {
	xys := make(plotter.XYs, 2)
	xys[1].X = x
	xys[1].Y = y
	ln, err := plotter.NewLine(xys)
	if err != nil {
		panic(err)
	}

	ln.LineStyle.Width = vg.Points(5)
	ln.LineStyle.Color = plotutil.Color(p.nlines)
	p.nlines++

	p.Add(ln)
	p.Legend.Add(lbl, ln)
}

func (p *plttr) addArea(lbl string, x0, y0, x1, y1 float64) {
	xys := make(plotter.XYs, 4)
	xys[0].X, xys[0].Y = x0, y0
	xys[1].X, xys[1].Y = x1, y0
	xys[2].X, xys[2].Y = x1, y1
	xys[3].X, xys[3].Y = x0, y1
	r, err := plotter.NewPolygon(xys)
	if err != nil {
		panic(err)
	}
	r.Color = plotutil.Color(p.nlines)
	p.nlines++
	p.Add(r)
	p.Legend.Add(lbl, r)
}

func (p *plttr) addMultivector(lbl string, a Multivector) {
	x, y := a.Add()
	p.addLine(lbl, x, y)
}

func (p *plttr) addBlade(lbl string, a Blade) {
	p.addMultivector(lbl, Multivector{a})
}

func (p *plttr) save(fname string) {
	if err := p.Save(8*vg.Inch, 8*vg.Inch, fname); err != nil {
		panic(err)
	}
}

const Degree = math.Pi / 180

func TestPlot(t *testing.T) {
	p := newplttr()
	// p.addArea("e1^e2", 0, 0, 1, 1)

	e1 := Blade{bitmap: 1, s: 1}
	e2 := Blade{bitmap: 1 << 1, s: 1}
	I := e1.Op(e2)
	_ = I

	rot := func(a Multivector, angle float64) Multivector {
		// A := Multivector{e1}
		// B := Multivector{e1, e2}

		// A := Multivector{Blade{e1.bitmap, 1}, Blade{e2.bitmap, 0}}
		// B := Multivector{Blade{e1.bitmap, 0}, Blade{e2.bitmap, 1}}
		// t.Logf(" A: %s", A)
		// t.Logf("AI: %s", A.Inverse())
		// t.Logf("A.Angle(B): %v", A.Angle(B))
		// R := B.Gp(A.Inverse())

		// RN := B.Gp(B).Gp(A.Gp(A).Inverse()).Norm()
		// R[0].s /= RN
		// R[1].s /= RN

		// R := Multivector{
		// Blade{s: math.Cos(angle / 2)},
		// Blade{bitmap: I.bitmap, s: -math.Sin(angle / 2)},
		// }

		R := Rotor(angle, I)

		t.Logf("   R: %s", R)
		t.Logf(" |R|: %v", R.Norm())
		t.Logf("  R~: %s", R.Rev())
		t.Logf("  Ri: %s", R.Inverse())
		t.Logf(" RR~: %s", R.Gp(R.Rev()))
		res := R.Gp(a).Gp(R.Rev())
		t.Logf("   a: %s", a)
		t.Logf("  Ra: %s", R.Gp(a))
		t.Logf("RaR~: %s", res)

		return res
	}

	p.addMultivector("e1", Multivector{e1})
	p.addMultivector("e2", Multivector{e2})

	// p.addMultivector("e1+e2", Multivector{e1, e2})

	// b := Multivector{e1, e2.Gp(Blade{s: -0.5})}
	// p.addMultivector("e1+0.5*e2", b)

	u1 := e1.Gp(Blade{s: 3})
	// u2 := e2.Gp(Blade{s: 0})
	u := Multivector{u1}
	p.addMultivector("u", u)
	// p.addMultivector("u1", rot(u, 1))
	// p.addMultivector("u1", rot(u, 2))
	v := rot(u, 45*Degree)
	p.addMultivector("v", v)

	w := rot(v, 45*Degree)
	p.addMultivector("w", w)

	R := v.Gp(u.Inverse())
	x := R.Gp(w).Gp(R.Rev())
	p.addMultivector("x", x)

	// uI := u.Gp(Multivector{I})
	// p.addMultivector("uI", uI)

	// uIv := u.Gp(Multivector{I.Rev()})
	// p.addMultivector("uIv", uIv)

	p.save("lines.png")
}
