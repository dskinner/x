package trigram_test

import (
	"fmt"

	"dasa.cc/x/trigram"
)

func Example() {
	terms := []string{"the quick", "red fox", "jumps over", "the lazy", "brown dog"}

	var gs trigram.Set
	gs.Index(terms...)

	var m []string
	var u []float64

	m, u = gs.Match("bog", 0.33) // mispelled "dog"
	for i, s := range m {
		fmt.Printf("%.2f: %q\n", u[i], s)
	}

	m, u = gs.Match("the", 0.33)
	for i, s := range m {
		fmt.Printf("%.2f: %q\n", u[i], s)
	}

	// Output:
	// 0.50: "brown dog"
	// 1.00: "the lazy"
	// 1.00: "the quick"
}
