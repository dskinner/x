//go:build plot

package gma

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	p, _ := plot.New()
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

	// const width, height = 200, 200
	// const width, height = 400, 400
	// const width, height = 800, 600
	// const width, height = 1000, 1000
	// const width, height = 1280, 720
	const width, height = 2560, 1440

	const zoom = 0.0005
	const maxiter = 90

	bounds := image.Rect(-width/2, -height/2, width/2, height/2)
	m := image.NewRGBA(bounds)

	// c := Multivector{{-0.8, E1}, {0.156, E2}}
	// c := Multivector{{-0.835, E1}, {-0.2321, E2}}
	// c := Multivector{{-0.70176, E1}, {-0.3842, E2}}
	c := Multivector{{-1.1, E1}, {-0.27, E2}}

	var (
		wg       sync.WaitGroup
		progress uint64
	)

	go func() {
		total := float64(bounds.Dx() * bounds.Dy())
		epoch := time.Now()
		for range time.Tick(1 * time.Second) {
			complete := float64(atomic.LoadUint64(&progress)) / total

			since := time.Since(epoch)
			estimate := time.Duration(1 / complete * float64(since))
			remaining := estimate - since

			fmt.Printf("%.0f%% complete; time remaining %s\n", complete*100, remaining)

			if complete == 1 {
				fmt.Printf("completed in %s\n", since)
				break
			}
		}
	}()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {

			wg.Add(1)

			go func(x, y int) {
				clr := color.RGBA{A: 255}

				p := Multivector{
					{float64(x) * zoom, E1},
					{float64(y) * zoom, E2},
				}

				for clr.G = 0; clr.G < maxiter; clr.G++ {
					// if clr.G%3 == 0 {
					// clr.B += 2
					// }

					p = p.Mul(e1).Mul(p).Add(c)

					if nsq := p.NormSq(); nsq > 1e6 {
						if 1e8 < nsq && nsq < 1e12 {
							cf := (1e12 / nsq) / 1e4
							u8 := uint8(cf * 255)

							clr.R = u8

							// darken out background
							clr.G = uint8((1e12 / nsq / 1e4) * 100)

							// brighten edges up
							// if nsq > 1e8 {
							// clr.R += u8
							// }

							// shift edge color from red to orange
							if nsq < 1e9 {
								clr.B += u8
							}
						}

						break
					}
				}

				m.Set(x, y, clr)
				atomic.AddUint64(&progress, 1)
				wg.Done()
			}(x, y)
		}
	}

	wg.Wait()

	reduceNoise(m, 7)
	saveImage(m, "julia.png")
}

// reduceNoise filters m by given window size with median filter; panics if size is less than 3 or even.
// m will be inset by size/2.
func reduceNoise(m *image.RGBA, size int) {
	if size < 3 || size%2 == 0 {
		panic("size must be >= 3 and odd")
	}

	var (
		n  = size*size - 1
		rs = make(Uint8Slice, 0, n)
		gs = make(Uint8Slice, 0, n)
		bs = make(Uint8Slice, 0, n)
	)

	apply := func(window *image.RGBA) {
		bounds := window.Bounds()
		if sz := bounds.Size(); sz.X != size || sz.Y != size {
			return // edge detected
		}
		pt := bounds.Min.Add(bounds.Size().Div(2))

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				if pt.X == x && pt.Y == y {
					continue
				}
				clr := m.At(x, y).(color.RGBA)
				rs = append(rs, clr.R)
				gs = append(gs, clr.B) // NOTE channel swap
				bs = append(bs, clr.G)
			}
		}

		window.Set(pt.X, pt.Y, color.RGBA{
			R: rs.Median(),
			G: gs.Median(),
			B: bs.Median(),
			A: 255,
		})

		rs, gs, bs = rs[:0], gs[:0], bs[:0]
	}

	bounds := m.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			apply(m.SubImage(image.Rect(x, y, x+size, y+size)).(*image.RGBA))
		}
	}

	inset := m.SubImage(m.Bounds().Inset(size / 2)).(*image.RGBA)
	*m = *inset
}

type Uint8Slice []uint8

func (x Uint8Slice) Len() int           { return len(x) }
func (x Uint8Slice) Less(i, j int) bool { return x[i] < x[j] }
func (x Uint8Slice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x Uint8Slice) Sort()              { sort.Sort(x) }

// Median sorts the receiver and returns median of values.
func (x Uint8Slice) Median() uint8 {
	x.Sort()
	n := len(x)
	if n == 0 {
		return 0
	}
	d := n / 2
	if n%2 == 0 {
		return (x[d] + x[d-1]) / 2
	}
	return x[d]
}
