package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"dasa.cc/x/glw"
	"golang.org/x/exp/shiny/widget/node"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

// type KeyHandler interface {
// 	Handle(key.Event)
// }

// type KeyHandleFunc func()

// func (fn KeyHandleFunc) Handle(key.Event) { fn() }

type KeyProc struct {
	Func interface{}
	Cond func(key.Event) bool
}

func KeyPressed(ev key.Event) bool  { return ev.Direction != key.DirRelease }
func KeyReleased(ev key.Event) bool { return ev.Direction == key.DirRelease }

func keymapPanLeft(e key.Event)     { state.panLeft = KeyPressed(e) }
func keymapPanRight(e key.Event)    { state.panRight = KeyPressed(e) }
func keymapPanUp(e key.Event)       { state.panUp = KeyPressed(e) }
func keymapPanDown(e key.Event)     { state.panDown = KeyPressed(e) }
func keymapRotateLeft(e key.Event)  { state.rotateLeft = KeyPressed(e) }
func keymapRotateRight(e key.Event) { state.rotateRight = KeyPressed(e) }
func keymapScaleUp(e key.Event)     { state.scaleUp = KeyPressed(e) }
func keymapScaleDown(e key.Event)   { state.scaleDown = KeyPressed(e) }

var (
	mousestate = make(map[mouse.Button]mouse.Direction)

	// TODO maybe could use x/set package to map multiple keys to same func ???
	keymap = map[key.Code]KeyProc{
		key.CodeA: {Func: keymapPanLeft},
		key.CodeD: {Func: keymapPanRight},
		key.CodeW: {Func: keymapPanUp},
		key.CodeS: {Func: keymapPanDown},
		key.CodeQ: {Func: keymapRotateLeft},
		key.CodeE: {Func: keymapRotateRight},
		key.CodeZ: {Func: keymapScaleUp},
		key.CodeX: {Func: keymapScaleDown},

		key.CodeLeftArrow:          {Func: keymapPanLeft},
		key.CodeRightArrow:         {Func: keymapPanRight},
		key.CodeUpArrow:            {Func: keymapPanUp},
		key.CodeDownArrow:          {Func: keymapPanDown},
		key.CodeLeftSquareBracket:  {Func: keymapRotateLeft},
		key.CodeRightSquareBracket: {Func: keymapRotateRight},
		key.CodeEqualSign:          {Func: keymapScaleUp},
		key.CodeHyphenMinus:        {Func: keymapScaleDown},

		key.CodeEscape: {
			Func: func() { os.Exit(0) },
			Cond: KeyReleased,
		},
		key.Code1: {
			Func: func() { Cycle(-1) },
			Cond: KeyPressed,
		},
		key.Code2: {
			Func: func() { Cycle(1) },
			Cond: KeyPressed,
		},
		key.Code3: {
			Func: func() { Cycle(-5) },
			Cond: KeyPressed,
		},
		key.Code4: {
			Func: func() { Cycle(5) },
			Cond: KeyPressed,
		},
		key.CodeI: {
			Func: func() { state.invf *= -1 },
			Cond: KeyReleased,
		},
		key.CodeR: {
			Func: func() {
				gallery.view.transform = Fit(viewport, gallery.view.Bounds().Size())
				gallery.Model.Stage(time.Time{}, glw.To(gallery.view.transform))
				Cycle(0)
			},
			Cond: KeyReleased,
		},

		key.CodeL: {
			Func: func() {
				labelState = (labelState + 1) % 3
				glwidget.Mark(node.MarkNeedsPaintBase)
			},
			Cond: KeyReleased,
		},
		key.CodeO: {
			Func: func() {
				cmd := exec.Command("giv", gallery.view.name)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Start(); err != nil {
					log.Println(err)
				}
			},
			Cond: KeyReleased,
		},

		key.CodeB: {
			Func: func() {
				if *flagFavs == "" {
					log.Println("favs folder not configured")
				} else {
					dst, err := os.Create(filepath.Join(*flagFavs, filepath.Base(gallery.view.name)))
					defer dst.Close()
					if err != nil {
						log.Println("dst error:", err)
						return
					}
					src, err := os.Open(gallery.view.name)
					defer src.Close()
					if err != nil {
						log.Println("src error:", err)
						return
					}
					if _, err := io.Copy(dst, src); err != nil {
						log.Println("copy error:", err)
						return
					}
				}
			},
			Cond: KeyReleased,
		},

		key.CodeDeleteForward: {
			Func: func() {
				ioutil.WriteFile(gallery.view.name, make([]byte, 24576), 0666)
				if err := store.Drop(ZV); err != nil {
					log.Println(err)
				} else {
					gallery.view = ZV
					Cycle(0)
				}
			},
			Cond: KeyReleased,
		},
	}
)
