package main

import (
	"fmt"

	"dasa.cc/x/nui"
)

var evs = make(chan nui.Event)

func init() {
	go monitor()
}

func monitor() {
	<-evs // wait for first event

	// ready! add new quirk to same channel
	// by casting draw func below as a Kink.
	// type Kink implements Quirk because
	// a quirk is a kink is a quirk.
	go nui.Open(evs, nui.EventFunc(draw))

	for ev := range evs {
		fmt.Println(ev)
	}
}

func draw() {
	fmt.Println("TODO draw something")
}

func main() {
	nui.Open(evs, nui.Surface|nui.Size)
}
