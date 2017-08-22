// Package set provides primitives for inserting distinct values into ordered sets.
package set

import (
	"reflect"
	"sort"
)

// StringSlice must be sorted in ascending order.
type StringSlice []string

// Insert x in place if not exists; returns x index and true if inserted.
// The slice must be sorted in ascending order.
func (a *StringSlice) Insert(x string) (i int, ok bool) {
	i = sort.SearchStrings(*a, x)
	if ok = i == len(*a) || (*a)[i] != x; ok {
		*a = stringUpsert(*a, x, i, ok)
	}
	return
}

func (a StringSlice) Has(x string) bool {
	i := sort.SearchStrings(a, x)
	return !(i == len(a) || (a)[i] != x)
}

func (a *StringSlice) filter() {
	sort.Strings(*a)
	b := StringSlice((*a)[:0])
	for i, x := range *a {
		// b.Insert(x)
		if i > 0 && (*a)[i-1] == x {
			continue
		}
		b = append(b, x)
	}
	*a = b
}

// StringSimple is always strictly ordered by its indices, given as [0 .. N-1].
type StringSimple []string

// Upsert inserts x at i if ok; otherwise, updates i to x.
func (a *StringSimple) Upsert(x string, i int, ok bool) { *a = stringUpsert(*a, x, i, ok) }

// StringChain is always strictly ordered by its indices, given as [0 .. N-1].
type StringChain []StringSlice

// Upsert inserts slice{x} at i if ok, and returns 0 and true.
// Otherwise, slice at i attempts insert of distinct x, and returns x index and true if inserted.
func (a *StringChain) Upsert(x string, i int, ok bool) (int, bool) {
	if ok {
		*a = append(*a, nil)
		copy((*a)[i+1:], (*a)[i:])
		(*a)[i] = StringSlice{x}
		return 0, true
	} else {
		return (*a)[i].Insert(x)
	}
}

func stringUpsert(a []string, x string, i int, ok bool) []string {
	if ok {
		a = append(a, "")
		copy(a[i+1:], a[i:])
	}
	a[i] = x
	return a
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

// IntSimple is always strictly ordered by its indices, given as [0 .. N-1].
type IntSimple []int

// Upsert inserts x at i if ok; otherwise, updates i to x.
func (a *IntSimple) Upsert(x int, i int, ok bool) { *a = intUpsert(*a, x, i, ok) }

func intUpsert(a []int, x int, i int, ok bool) []int {
	if ok {
		a = append(a, 0)
		copy(a[i+1:], a[i:])
	}
	a[i] = x
	return a
}

// IntChain is always strictly ordered by its indices, given as [0 .. N-1].
type IntChain []IntSlice

// Upsert inserts slice{x} at i if ok, and returns 0 and true.
// Otherwise, slice at i attempts insert of distinct x, and returns x index and true if inserted.
func (a *IntChain) Upsert(x int, i int, ok bool) (int, bool) {
	if ok {
		*a = append(*a, nil)
		copy((*a)[i+1:], (*a)[i:])
		(*a)[i] = IntSlice{x}
		return 0, true
	} else {
		return (*a)[i].Insert(x)
	}
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

// Float64Simple is always strictly ordered by its indices, given as [0 .. N-1].
type Float64Simple []float64

// Upsert inserts x at i if ok; otherwise, updates i to x.
func (a *Float64Simple) Upsert(x float64, i int, ok bool) { *a = float64Upsert(*a, x, i, ok) }

func float64Upsert(a []float64, x float64, i int, ok bool) []float64 {
	if ok {
		a = append(a, 0)
		copy(a[i+1:], a[i:])
	}
	a[i] = x
	return a
}

// Float64Chain is always strictly ordered by its indices, given as [0 .. N-1].
type Float64Chain []Float64Slice

// Upsert inserts slice{x} at i if ok, and returns 0 and true.
// Otherwise, slice at i attempts insert of distinct x, and returns x index and true if inserted.
func (a *Float64Chain) Upsert(x float64, i int, ok bool) (int, bool) {
	if ok {
		*a = append(*a, nil)
		copy((*a)[i+1:], (*a)[i:])
		(*a)[i] = Float64Slice{x}
		return 0, true
	} else {
		return (*a)[i].Insert(x)
	}
}

// Insert x in place if not exists; returns x index and true if inserted.
// Function f is called as described by sort.Search but the slice must already
// be sorted in the order specified by f.
//
// The function panics if slice is not addressable or does not point to a slice of type x.
func Insert(slice interface{}, x interface{}, f func(int) bool) (i int, ok bool) {
	sval := reflect.ValueOf(slice).Elem()
	xval := reflect.ValueOf(x)
	i = sort.Search(sval.Len(), f)
	if ok = i == sval.Len() || !reflect.DeepEqual(sval.Index(i).Interface(), x); ok {
		sval.Set(reflect.Append(sval, reflect.Zero(xval.Type())))
		reflect.Copy(sval.Slice(i+1, sval.Len()), sval.Slice(i, sval.Len()))
		sval.Index(i).Set(xval)
	}
	return
}
