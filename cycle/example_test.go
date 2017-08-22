package cycle_test

import (
	"bytes"
	"fmt"
	"log"

	"dasa.cc/x/cycle"
)

func Example() {
	// ids of entire set, held in memory.
	ids := []int{0, 1, 2, 3, 4}

	// load data by id; don't want entire set's data in memory.
	load := func(id int) ([]byte, error) {
		if id < 0 || id > 4 {
			return nil, fmt.Errorf("Unknown id %v", id)
		}
		return []byte{uint8(id)}, nil
	}

	// pool holds data loaded by id for a given number of items held in memory.
	pool := make([][]byte, 3)

	// initial index of zero, also keep one item to the left and to the right loaded.
	r, err := cycle.New(-1, 0, len(pool), len(ids))
	if err != nil {
		log.Fatal(err)
	}

	// mustLoad populates pool with data; argument i is an absolute position
	// in entire set to load data for; r.Map(i) maps to a relative position
	// in pool to store data.
	mustLoad := func(i int) {
		var err error
		if pool[r.Map(i)], err = load(i); err != nil {
			panic(err)
		}
	}

	// load data from left to right.
	r.Do(r.Left(), 1, mustLoad)

	// or, load data from index to right and index-1 to left; could ran in parallel.
	r.Do(r.Index(), 1, mustLoad)
	r.Do(r.Index()-1, -1, mustLoad)

	// print data from left to right.
	buf := new(bytes.Buffer)
	r.Do(r.Left(), 1, func(i int) {
		fmt.Fprintf(buf, "%+v", pool[r.Map(i)])
	})
	fmt.Println("index", r.Index())
	fmt.Println("start", buf.String())
	fmt.Println()

	// cycle the projection and index to right.
	wi, ws, err := r.Cycle(1, 1)
	if err != nil {
		log.Fatal(err)
	}

	// update stale entries in p.
	r.Do(wi, ws, mustLoad)

	// print values after cycle.
	buf.Reset()
	r.Do(r.Left(), 1, func(i int) {
		fmt.Fprintf(buf, "%+v", pool[r.Map(i)])
	})
	fmt.Println("index", r.Index())
	fmt.Println("cycle", buf.String())
	// Output:
	// index 0
	// start [4][0][1]
	//
	// index 1
	// cycle [0][1][2]
}
