package main

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/mobile/event/paint"

	"dasa.cc/x/cycle"
	"dasa.cc/x/glw"
)

const maxsize = 20

var walked = make(map[string]bool)

func decodeConfig(name string) (image.Config, error) {
	f, err := os.Open(name)
	if err != nil {
		return image.Config{}, err
	}
	defer f.Close()
	config, _, err := image.DecodeConfig(f)
	return config, err
}

// ZV is the zero View.
var ZV View

type View struct {
	name      string
	config    image.Config
	info      os.FileInfo
	err       error
	transform glw.Transform
}

func (v View) Bytes() int { return 4 * v.config.Width * v.config.Height }

func (v View) Bounds() image.Rectangle { return image.Rect(0, 0, v.config.Width, v.config.Height) }

func (v View) Fit(width, height int) float32 {
	fbw, fbh := float32(width), float32(height)
	iw, ih := float32(v.config.Width), float32(v.config.Height)

	// image height fills framebuffer height when zoom == 1
	// and f is image's width factor.
	f := fbh / ih
	p := f * iw

	// if product > framebuffer width, reduce scale to equal.
	if p > fbw {
		return fbw / p
	}
	return 1
}

type byMod []View

func (a byMod) Len() int { return len(a) }

func (a byMod) Less(i, j int) bool {
	return a[i].info.ModTime().Before(a[j].info.ModTime())
}

func (a byMod) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type pixl struct {
	sync.Mutex
	sync.Cond
	name string
	bin  []uint8
}

func newpixl() *pixl {
	p := &pixl{}
	p.L = &p.Mutex
	return p
}

type Store struct {
	r     *cycle.R
	views []View
	pool  []*pixl
	ops   uint64

	signal    chan struct{}
	die, done chan struct{}
}

func NewStore() *Store {
	st := &Store{
		signal: make(chan struct{}, 1),
		die:    make(chan struct{}),
		done:   make(chan struct{}),
	}
	// for first call to walk
	go func() {
		select {
		case <-st.die:
			close(st.done)
		}
	}()
	return st
}

func (st *Store) String() string {
	busy := ""
	if st.ops > 0 {
		busy = ""
	}
	i := 0
	if st.r != nil {
		i = st.r.Index() + 1
	}
	return fmt.Sprintf("%v/%v %s", i, len(st.views), busy)
}

// Pix return pixel data for current index or blocks until it becomes available.
func (st *Store) Pix() []uint8 {
	i := st.r.Index()
	p := st.pool[st.r.Map(i)]
	p.Lock()
	defer p.Unlock()

	for {
		if st.views[i].err != nil {
			return nil
		}
		if len(p.bin) != 0 && st.views[i].name == p.name {
			return p.bin
		}
		p.Wait()
	}
}

// Cancel monitoring of background data loading.
func (st *Store) Cancel() (ok bool) {
	select {
	case <-st.done:
		return false
	default:
		close(st.die)
		<-st.done
		return true
	}
}

// Monitor for background data loading.
func (st *Store) Monitor() {
	st.die = make(chan struct{})
	st.done = make(chan struct{})
	go monitor(st.r, st.Load, st.signal, st.die, st.done)
}

// Load image data into pool.
func (st *Store) Load(i int) {
	idx := st.r.Map(i)
	p := st.pool[idx]
	p.Lock()
	defer p.Unlock()

	view := st.views[i]
	if len(p.bin) > 0 && p.name == view.name {
		return
	}
	p.name = view.name

	atomic.AddUint64(&st.ops, 1)
	defer func() {
		if atomic.LoadUint64(&st.ops) == 0 && deque != nil {
			glwidget.Mark(node.MarkNeedsPaintBase)
			deque.Send(paint.Event{})
		}
	}()
	defer atomic.AddUint64(&st.ops, ^uint64(0))

	f, err := os.Open(view.name)
	if err != nil {
		st.views[i].err = fmt.Errorf("os.Open: %s", err)
		return
	}
	defer f.Close()

	// TODO image.Decode calls into registered driver which in turn calls into
	// image package, e.g. image.NewRGBA(bounds)
	// This function for example is always making []uint8 slices but if I vendored
	// image package, I could maybe override this to allow reuse of already allocated
	// slice memory. I just might have to manually register each supported image format
	// though as side effect by import probably wouldn't work.
	//
	// Decode times have ranged 10ms - 80ms with test images.
	m, _, err := image.Decode(f)
	if err != nil {
		st.views[i].err = fmt.Errorf("image.Decode: %s", err)
		return
	}

	// format conversion, reuse previously allocated memory if available.
	var rgba *image.RGBA
	if pix := p.bin; pix != nil {
		r := m.Bounds()
		w, h := r.Dx(), r.Dy()
		n := 4 * w * h

		a := cap(pix)
		if a > n {
			pix = pix[:n]
		} else {
			pix = append(pix[:a], make([]uint8, n-a)...)
		}

		rgba = &image.RGBA{pix, 4 * w, r}
	} else {
		rgba = image.NewRGBA(m.Bounds())
	}

	// TODO have seen cases where image is corrupt and available pix data
	// doesn't match expected length from decoded image config width and height.
	// should do some kind of check and support loading partial, like feh.

	if rgba.Stride != rgba.Rect.Size().X*4 {
		panic("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), m, image.ZP, draw.Src)

	p.bin = rgba.Pix
	p.Broadcast()
}

// Cycle index by stride. Notifies monitor to reload data if projection shifts.
func (st *Store) Cycle(v View, stride int) View {
	if v.name != "" {
		st.views[st.r.Index()] = v
	}

	// TODO make seperate method
	if stride == 0 { // fastpath reload
		i := st.r.Index()
		p := st.pool[st.r.Map(i)]
		p.bin = p.bin[:0]
		st.Load(i)
		return st.views[i]
	}

	var shift int
	il, ir := st.r.Diff(st.r.Index() + stride)
	if il/2 > ir {
		shift = il / 2
	} else if ir/2 > il {
		shift = -ir / 2
	}

	if shift != 0 {
		st.Cancel()
	}

	if shift > 0 {
		st.r.Do(st.r.Left()+shift, -1, func(i int) {
			p := st.pool[st.r.Map(i)]
			p.bin = p.bin[:0]
		})
	} else if shift < 0 {
		st.r.Do(st.r.Right()+shift, 1, func(i int) {
			p := st.pool[st.r.Map(i)]
			p.bin = p.bin[:0]
		})
	}

	if _, _, err := st.r.Cycle(shift, stride); err != nil {
		log.Fatal("Store cycle failed:", err)
	}

	if shift != 0 {
		st.Monitor()
		// select {
		// case st.signal <- struct{}{}:
		// default:
		// }
	}

	return st.views[st.r.Index()]
}

// Drop view from displaying.
func (st *Store) Drop(view View) error {
	if len(st.views) == 1 {
		os.Exit(1)
	}

	if !st.Cancel() {
		return fmt.Errorf("Drop failed, store is busy")
	}
	defer st.Monitor()

	i := st.r.Index()
	st.views = append(st.views[:i], st.views[i+1:]...)

	st.pool[st.r.Map(i)].bin = nil

	st.r.Do(i+1, 1, func(i int) {
		p0, p1 := st.pool[st.r.Map(i-1)], st.pool[st.r.Map(i)]
		p0.bin, p1.bin = p1.bin, nil
	})

	return st.Init(i)
}

func mustParseDuration(x string) time.Duration {
	dur, err := time.ParseDuration(x)
	if err != nil {
		panic(err)
	}
	return dur
}

// Walk paths to locate images for display.
func (st *Store) Walk(i int, paths ...string) error {
	if !st.Cancel() {
		return fmt.Errorf("Walk failed, store is busy")
	}
	defer st.Monitor()

	var dur time.Duration
	if *flagDur != "" {
		dur = mustParseDuration(*flagDur)
	}

	var changed bool
	for _, path := range paths {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if walked[path] {
				return nil
			}
			walked[path] = true

			if dur != 0 && info.ModTime().Before(time.Now().Add(dur)) {
				return nil
			}

			if config, err := decodeConfig(path); err == nil {
				st.views = append(st.views, View{name: path, config: config, info: info})
				changed = true
			}
			return nil
		})
	}

	if !changed {
		return fmt.Errorf("No images to display")
	}

	if *flagSort {
		if *flagRev {
			sort.Stable(sort.Reverse(byMod(st.views)))
		} else {
			sort.Stable(byMod(st.views))
		}
	}

	return st.Init(i)
}

// Init cycle relation and pool.
func (st *Store) Init(i int) (err error) {
	i = max(0, min(i, len(st.views)-1))
	n := min(maxsize, len(st.views))
	st.r, err = cycle.New(i-n/2, i, n, len(st.views))
	if err != nil {
		return fmt.Errorf("failed to create cyclic relation:", err)
	}

	if st.pool == nil {
		st.pool = make([]*pixl, n)
		for i := range st.pool {
			st.pool[i] = newpixl()
		}
	} else if len(st.pool) != cap(st.pool) {
		st.pool = st.pool[:min(n, cap(st.pool))]
		for i := range st.pool {
			if st.pool[i] == nil {
				st.pool[i] = newpixl()
			}
		}
	}

	if len(st.pool) < n {
		diff := n - len(st.pool)
		for i := 0; i < diff; i++ {
			st.pool = append(st.pool, newpixl())
		}
	}

	return nil
}

func monitor(r *cycle.R, load func(int), signal, die, done chan struct{}) {
	var dead bool
	loader := func(i int) {
		select {
		case <-die:
			dead = true
		default:
		}
		if dead {
			return
		}
		load(i)
	}

	i := r.Index()
	r.Do(i, 1, loader)
	r.Do(i-1, -1, loader)

	for {
		if dead {
			close(done)
			return
		}

		select {
		case <-signal:
			i := r.Index()
			r.Do(i, 1, loader)
			r.Do(i-1, -1, loader)
		case <-die:
			close(done)
			return
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
