package set_test

import (
	"fmt"
	"sort"

	"dasa.cc/x/set"
)

func Example() {
	a := []string{
		"a", "b", "c",
		"b", "c", "d",
		"c", "d", "e",
	}

	// filter without allocating
	b := set.Slice[string](a[:0])
	for _, x := range a {
		b.Insert(x)
	}

	fmt.Println(b)
	fmt.Println("sorted", sort.StringsAreSorted(b))

	// Output:
	// [a b c d e]
	// sorted true
}
