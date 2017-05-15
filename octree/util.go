package octree

import "sort"

type u32slice []uint32

func (a u32slice) Len() int           { return len(a) }
func (a u32slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a u32slice) Less(i, j int) bool { return a[i] < a[j] }

// SortU32s is a convenience method for sorting a slice of uint32s.
func SortU32s(a []uint32) { sort.Sort(u32slice(a)) }

type u64slice []uint64

func (a u64slice) Len() int           { return len(a) }
func (a u64slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a u64slice) Less(i, j int) bool { return a[i] < a[j] }

// SortU64s is a convenience method for sorting a slice of uint64s.
func SortU64s(a []uint64) { sort.Sort(u64slice(a)) }

// Subdivide16 treats u as a node in a point tree and recursively
// subdivides u up to lvl, collecting results at lvl into p.
func Subdivide16(u uint64, lvl int, p *[]uint64) {
	_, _, _, w := Deinterleave16(u)
	if int(w) == lvl {
		*p = append(*p, u)
	} else {
		a, b, c, d, e, f, g, h := Children16(u)
		Subdivide16(a, lvl, p)
		Subdivide16(b, lvl, p)
		Subdivide16(c, lvl, p)
		Subdivide16(d, lvl, p)
		Subdivide16(e, lvl, p)
		Subdivide16(f, lvl, p)
		Subdivide16(g, lvl, p)
		Subdivide16(h, lvl, p)
	}
}
