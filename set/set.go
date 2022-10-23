// Package set provides primitives for inserting distinct values into ordered sets.
package set

import (
	"sort"

	"golang.org/x/exp/constraints"
)

// Slice must be sorted in ascending order.
type Slice[T constraints.Ordered] []T

// Insert x in place if not exists; returns x index and true if inserted.
// The slice must be sorted in ascending order.
func (a *Slice[T]) Insert(x T) (i int, ok bool) {
	i = sort.Search(len(*a), func(i int) bool { return (*a)[i] >= x })
	if ok = i == len(*a) || (*a)[i] != x; ok {
		*a = upsert(*a, x, i, ok)
	}
	return
}

func (a Slice[T]) Has(x T) bool {
	i := sort.Search(len(a), func(i int) bool { return a[i] >= x })
	return !(i == len(a) || a[i] != x)
}

// Simple is always strictly ordered by its indices, given as [0 .. N-1].
type Simple[T constraints.Ordered] []T

// Upsert inserts x at i if ok; otherwise, updates i to x.
func (a *Simple[T]) Upsert(x T, i int, ok bool) { *a = upsert(*a, x, i, ok) }

// Chain is always strictly ordered by its indices, given as [0 .. N-1].
type Chain[T constraints.Ordered] []Slice[T]

// Upsert inserts slice{x} at i if ok, and returns 0 and true.
// Otherwise, slice at i attempts insert of distinct x, and returns x index and true if inserted.
func (a *Chain[T]) Upsert(x T, i int, ok bool) (int, bool) {
	if ok {
		*a = append(*a, nil)
		copy((*a)[i+1:], (*a)[i:])
		(*a)[i] = Slice[T]{x}
		return 0, true
	} else {
		return (*a)[i].Insert(x)
	}
}

func upsert[T constraints.Ordered](a []T, x T, i int, ok bool) []T {
	if ok {
		a = append(a, *new(T))
		copy(a[i+1:], a[i:])
	}
	a[i] = x
	return a
}

// Filter without allocating.
func Filter[T constraints.Ordered](a *[]T) {
	b := Slice[T]((*a)[:0])
	for _, x := range (*a) {
		b.Insert(x)
	}
	*a = b
}