package cycle_test

import (
	"bytes"
	"fmt"
	"log"

	"dasa.cc/x/cycle"
)

func Example() {
	// load represents some finite set to cycle that can't be held in memory.
	load := func(i int) []uint8 { return [][]uint8{{1}, {2}, {3}, {4}, {5}}[i] }

	// p is what will be stored in memory. With a little more effort, this example
	// could reuse allocated memory in p.
	p := make([][]uint8, 3)
	// initial index of zero, also keep one item to the left and to the right loaded.
	r, err := cycle.New(-1, 0, 3, 5)
	if err != nil {
		log.Fatal(err)
	}

	// load indices from zero towards right and minus-one towards left;
	// these could be ran in parallel or a single load from left to right
	// could be ran instead.
	r.Do(0, 1, func(i int) {
		p[r.Map(i)] = load(i)
	})
	r.Do(-1, -1, func(i int) {
		p[r.Map(i)] = load(i)
	})

	// print start values from left to right.
	buf := new(bytes.Buffer)
	r.Do(r.Left(), 1, func(i int) {
		fmt.Fprintf(buf, "%+v", p[r.Map(i)])
	})
	fmt.Println("start", buf.String())

	// cycle projection and index to the right.
	wi, ws, err := r.Cycle(1, 1)
	if err != nil {
		log.Fatal(err)
	}
	// update stale entries in p.
	r.Do(wi, ws, func(i int) {
		p[r.Map(i)] = load(i)
	})

	// print values after cycle.
	buf.Reset()
	r.Do(r.Left(), 1, func(i int) {
		fmt.Fprintf(buf, "%+v", p[r.Map(i)])
	})
	fmt.Println("cycle", buf.String())
	// Output:
	// start [5][1][2]
	// cycle [1][2][3]
}
