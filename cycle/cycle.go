// Package cycle implements a total cyclic order relation for transforming linear orders of the form 0..N into cyclic orders.
// Beyond what a modulo provides, this allows for the cycling of a projection and an index independently.
package cycle

import (
	"fmt"
)

// R is a total cyclic order relation, maintaining a left,right projection and a tracked index, both of which may cycle independent of each other.
type R struct {
	o pro // set projection
	n int // subset length
	z int // zero index displacement; if o.l == index-z, index args of zero map to zero
}

// New instance of R with displacement, index, subset length, and parent set length.
func New(z, i, nsubset, nset int) (*R, error) {
	o := pro{i: i, n: nset}
	o.l = pmod(z, nset)
	o.r = pmod(o.l+nsubset, nset)

	if err := o.verify(); err != nil {
		return nil, err
	}

	return &R{o: o, n: nsubset, z: -z}, nil
}

// Left returns left projection index of parent set. Due to totality, left > right is possible.
func (r *R) Left() int { return r.o.l }

// Right returns right projection index of parent set. Due to totality, right < left is possible.
func (r *R) Right() int { return r.o.r }

// Index returns absolute index of parent set.
func (r *R) Index() int { return r.o.i }

// Diff returns absolute difference of index to left and right projections.
func (r *R) Diff(i int) (dl, dr int) { return r.o.diff(i) }

// Map an index for parent set to subset.
func (r *R) Map(i int) int {
	dl, _ := r.o.diff(i)
	return pmod(dl-r.z, r.n)
}

// Cycle projection and index, return offset index along stride.
func (r *R) Cycle(sp, si int) (i, s int, err error) {
	o := r.o // copy
	o.l, o.i, o.r = pmod(o.l+sp, o.n), pmod(o.i+si, o.n), pmod(o.r+sp, o.n)

	if err := o.verify(); err != nil {
		return 0, 0, err
	}

	if sp > 0 {
		i, s = r.o.r, 1
	} else if sp < 0 {
		i, s = r.o.l, -1
	}

	r.o = o   // replace
	r.z -= sp // displace
	return i, s, nil
}

// Do executes fn for each index along stride to projection end.
// If stride is zero, fn(index) is called once.
func (r *R) Do(i, s int, fn func(i int)) {
	i = pmod(i, r.o.n)
	if s == 0 {
		fn(i)
		return
	}

	il, k := r.Diff(i)
	if s < 0 {
		k = 1 + il
	}

	for ; k > 0; k, i = k-1, pmod(i+s, r.o.n) {
		fn(i)
	}
}

type pro struct{ l, i, r, n int }

// verify non-strict totality.
func (o pro) verify() error {
	if o.l < o.r && !(o.l <= o.i && o.i <= o.r) {
		return fmt.Errorf("for l < r then l < i < r but %v < %v < %v", o.l, o.i, o.r)
	}
	if o.r < o.l && !((o.l >= o.i && o.i <= o.r) || (o.l <= o.i && o.i >= o.r)) {
		return fmt.Errorf("for r < l then l > i < r or l < i > r but %v %s %v %s %v", o.l, equality(o.l, o.i), o.i, equality(o.i, o.r), o.r)
	}
	return nil
}

// diff returns absolute difference of index to left and right projections.
func (o pro) diff(i int) (dl, dr int) {
	return pmod(o.n+i-o.l, o.n), pmod(o.n-i+o.r, o.n)
}

// pmod returns positive modulo for inputs.
func pmod(x, n int) int { return (x%n + n) % n }

func equality(a, b int) string {
	if a <= b {
		return "<"
	}
	return ">"
}
