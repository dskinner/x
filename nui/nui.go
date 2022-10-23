//go:generate stringer -type=Event

// Package nui aims to be unremarkable in aiding windowing.
package nui

import (
	"fmt"
	"sync"
)

type event interface {
	String() string

	// Event is as EventFunc is an Event.
	Event()
}

type Event int

func (a Event) Event() {}

// EventFunc is a func that implements Event.
type EventFunc func()

func (a EventFunc) String() string { return fmt.Sprintf("%T", a) }
func (a EventFunc) Event()         { a() } // a quirk is a kink is a quirk

// TODO nui.Stable Quirk ??? so if something changes, nui will try
// and put things right again, such as resizing GL viewport, redrawing
// dirty window, etc.
const (
	Surface  Event = 1 << iota // Surface quirks includes things like windowing, rendering, etc
	Position                   // Position quirks include things like window and mouse coordinates, etc
	Size                       // Size quirks include things like window and framebuffer size and resize, etc
	View                       // View quirks include things like visibility, focusing, floating, maximizing, etc
	Touch                      // Touch quirks include things like button and key presses, physicality, etc
	Nada                       // Nada donned alone terminates the system and closes provided channel if not nil
)

var (
	// events are those little differences in a chugging-along system.
	// events = make(chan Event)

	// eventFuncs are those little things prone to panicking a system.
	eventFuncs = make(chan func())

	// aid requests give meaning to all that is to come.
	aid sync.Map
)

// unremarkable is the aim in spite of kinks/eventFuncs.
func unremarkable() {
	window, terminate := surface()
	defer terminate()

	for eventFunc := range eventFuncs {
		eventFunc()

		aid.Range(func(k, v interface{}) bool {
			// this could concievably change in the future but I'd rather not,
			// keep kinks feeding into the global, not something stored in the map.
			// ... though that'd be handy for redrawing until Doffing.
			// ... so, switch to a type switch.
			on, ok := k.(chan<- Event)
			if !ok {
				return true
			}

			evs := v.([]event)
			for _, ev := range evs {
				switch v := ev.(type) {
				case Event:
					if v&Surface == Surface {
						on <- Surface
					}
				case EventFunc:
					v()
				}
			}

			return true
		})

		if window.ShouldClose() {
			close(eventFuncs)
			return
		}

		window.SwapBuffers()
		go leer(waitEvents) // TODO not what I had in mind, maybe do another "once" and just keep a goroutine working on that.
	}
}

// leer at eventFunc until gone.
//
// eventFunc is wrapped to supply a channel to wait on, then passed into eventFuncs channel
// for evaluation on main thread. leer returns once eventFunc has executed and returned.
func leer(eventFunc func()) {
	done := make(chan struct{}, 1)
	eventFuncs <- func() { eventFunc(); done <- struct{}{} }
	<-done
}

/*
func store(key chan<- Event, events ...event) func() {
	panic("DO NOT CALL; for reference only.")
	return func() { aid.Store(key, events) }
}
*/

func delete(key chan<- Event) func() {
	return func() { aid.Delete(key) }
}

func loadOrStore(key chan<- Event, events ...event) func() {
	return func() {
		if actual, loaded := aid.LoadOrStore(key, events); loaded {
			value := actual.([]event)
			// []event; some are ints and some are funcs so append
			aid.Store(key, append(value, events...))
		}
	}
}

var initOnce sync.Once

// Don these quirks and leer.
//
// First call made to Don should be from main thread.
func Open(on chan<- Event, events ...event) {
	eventFunc := loadOrStore(on, events...)
	go leer(eventFunc)

	initOnce.Do(unremarkable)
}

// Doff and leer.
func Close(off chan<- Event) { eventFunc := delete(off); leer(eventFunc) }

// expose the peculiar.
// func expose(quirk Quirk) func() {
	// switch {
		// case quirk^Nada == 0:
		// return func() { close(kinks) }
		// default:
		// return nil
		// }
		// }

		// TODO https://godoc.org/github.com/go-gl/glfw/v3.2/glfw
		// don't forget to protect all those main-only funcs

		// TODO, alternate, figure out what works best
		// const (
			// Surface Quirk = 1 << iota
			// Position
			// Size
			// View
			// Touch
			// Nada
			// )
