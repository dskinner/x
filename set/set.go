// Package set provides primitives for inserting distinct values into sorted slices.
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
