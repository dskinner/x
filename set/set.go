// Package set provides primitives for inserting distinct values into sorted slices.
package set

import "sort"

// StringSlice must be sorted in ascending order.
type StringSlice []string

// Insert x in place if not exists; returns x index and true if inserted.
// The slice must be sorted in ascending order.
func (a *StringSlice) Insert(x string) (i int, ok bool) {
	i = sort.SearchStrings(*a, x)
	if ok = i == len(*a) || (*a)[i] != x; ok {
		*a = append(*a, "")
		copy((*a)[i+1:], (*a)[i:])
		(*a)[i] = x
	}
	return
}

// IntSlice must be sorted in ascending order.
type IntSlice []int

// Insert x in place if not exists; returns x index and true if inserted.
// The slice must be sorted in ascending order.
func (a *IntSlice) Insert(x int) (i int, ok bool) {
	i = sort.SearchInts(*a, x)
	if ok = i == len(*a) || (*a)[i] != x; ok {
		*a = append(*a, 0)
		copy((*a)[i+1:], (*a)[i:])
		(*a)[i] = x
	}
	return
}

// Float64Slice must be sorted in ascending order.
type Float64Slice []float64

// Insert x in place if not exists; returns x index and true if inserted.
// The slice must be sorted in ascending order.
func (a *Float64Slice) Insert(x float64) (i int, ok bool) {
	i = sort.SearchFloat64s(*a, x)
	if ok = i == len(*a) || (*a)[i] != x; ok {
		*a = append(*a, 0)
		copy((*a)[i+1:], (*a)[i:])
		(*a)[i] = x
	}
	return
}
