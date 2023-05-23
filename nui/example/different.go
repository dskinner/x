//go:build ignore

package main

import (
	"fmt"

	"dasa.cc/x/nui"
)

var events = make(chan nui.Event)

func draw() {
	fmt.Println("TODO draw something")
}

func main() {
	// nui.Open(events, nui.Surface|nui.Size)
	// <-events // wait for first event

	// ready! add new quirk to same channel
	// by casting draw func below as a Kink.
	// type Kink implements Quirk because
	// a quirk is a kink is a quirk.
	nui.Open(events, nui.EventFunc(draw))

	for ev := range events {
		fmt.Println(ev)
	}
}
