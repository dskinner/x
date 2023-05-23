package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"dasa.cc/x/gma"
)

var (
	flagI      = flag.Int("i", 0, "number to start iterator; used in output filename")
	flagN      = flag.Int("n", 1, "number of frames to output")
	flagWidth  = flag.Int("w", 650, "frame width")
	flagHeight = flag.Int("h", 500, "frame height")
	// flagPanX   = flag.Int("panx", 0, "pan along x coord")
	// flagPanY   = flag.Int("pany", 0, "pan along y coord")
)

func init() {
	flag.Parse()
}

/*

def f(step, frames):
  x = 0.007
	for a in range(frames):
		x *= step
	return x

*/

func main() {
	var (
		zoom = 0.007
		// zoom = 0.0001
		// zoom = 0.00001
		// zoom = 0.0000001
		// zoom = 0.000000001
		// zoom = 0.0000000001
		// step = 0.995 // for n = 600 target zoom 0.00035
		step = 0.972 // for n = 600 and target zoom 2e10
		i, n = *flagI, *flagI + *flagN
	)

	if i != 0 {
		for j := 0; j < i; j++ {
			zoom *= step
		}
	}

	progress := monitor(uint64(n - i))

	for ; i < n; i++ {
		fname := fmt.Sprintf("output/%04v_julia.png", i)
		julia(*flagWidth, *flagHeight, zoom, fname)
		fmt.Printf("[%v/%v] [zoom=%.9f]\n", i+1, n, zoom)
		zoom *= step

		atomic.AddUint64(progress, 1)
	}
}

func monitor(total uint64) *uint64 {
	progress := uint64(0)
	epoch := time.Now()
	go func() {
		for range time.Tick(1 * time.Second) {
			complete := float64(atomic.LoadUint64(&progress)) / float64(total)
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
	return &progress
}

func julia(width, height int, zoom float64, fname string) {
	e1 := gma.Multivector{{Scalar: 1, Basis: gma.E1}}

	// const width, height = 200, 200
	// const width, height = 400, 400
	// const width, height = 800, 600
	// const width, height = 1000, 1000

	// const width, height = 2560, 1440

	// const zoom = 0.0005
	const maxiter = 90

	bounds := image.Rect(-width/2, -height/2, width/2, height/2)

	panw := (int)(2.368821 * (0.007 / zoom))
	pan := image.Pt(-panw, panw)
	bounds = bounds.Add(pan)

	m := image.NewRGBA(bounds)

	// c := Multivector{{-0.8, E1}, {0.156, E2}}
	// c := Multivector{{-0.835, E1}, {-0.2321, E2}}
	// c := Multivector{{-0.70176, E1}, {-0.3842, E2}}

	c := gma.Multivector{{Scalar: -1.1, Basis: gma.E1}, {Scalar: -0.27, Basis: gma.E2}}
	// c := gma.Multivector{{Scalar: -1.1003, Basis: gma.E1}, {Scalar: -0.27003, Basis: gma.E2}}
	// c := gma.Multivector{{Scalar: -1.1003, Basis: gma.E1}, {Scalar: -0.27, Basis: gma.E2}}

	var (
		wg       sync.WaitGroup
		progress uint64
	)

	// monitor
	// go func() {
	// 	total := float64(bounds.Dx() * bounds.Dy())
	// 	epoch := time.Now()
	// 	for range time.Tick(1 * time.Second) {
	// 		complete := float64(atomic.LoadUint64(&progress)) / total
	// 		since := time.Since(epoch)
	// 		estimate := time.Duration(1 / complete * float64(since))
	// 		remaining := estimate - since
	// 		fmt.Printf("%.0f%% complete; time remaining %s\n", complete*100, remaining)
	// 		if complete == 1 {
	// 			fmt.Printf("completed in %s\n", since)
	// 			break
	// 		}
	// 	}
	// }()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {

			wg.Add(1)

			go func(x, y int) {
				clr := color.RGBA{A: 255}

				p := gma.Multivector{
					{Scalar: float64(x) * zoom, Basis: gma.E1},
					{Scalar: float64(y) * zoom, Basis: gma.E2},
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
							// clr.G = uint8((1e12 / nsq / 1e4) * 100)

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

	// additive(m)
	reduceNoise(m, 7)
	// markMiddle(m, 21)
	saveImage(m, fname)
}

func markMiddle(m *image.RGBA, size int) {
	if size < 3 || size%2 == 0 {
		panic("size must be >= 3 and odd")
	}

	bounds := m.Bounds()
	mx := bounds.Min.X + bounds.Dx()/2 - size/2
	my := bounds.Min.Y + bounds.Dy()/2 - size/2

	subm := m.SubImage(image.Rect(mx, my, mx+size, my+size)).(*image.RGBA)

	clr := color.RGBA{R: 255, A: 255}
	r := subm.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			subm.Set(x, y, clr)
		}
	}
}

func additive(m *image.RGBA) {
	bounds := m.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			clr := m.At(x, y).(color.RGBA)
			clr.R *= 2
			clr.G *= 2
			clr.B *= 2
			m.Set(x, y, clr)
		}
	}
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
