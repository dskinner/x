package quadtree

import (
	"fmt"
	"testing"
)

func printWord(t *testing.T, pad string, title string, key uint32) {
	x, y, level := Decode(key)
	nx, ny, size := Cell(key)
	t.Logf("%s%s:\n", pad, title)
	t.Logf("%s  parent: %b\n", pad, Parent(key))
	t.Logf("%s  node: %b\n", pad, key)
	t.Logf("%s  level: %v\n", pad, level)
	t.Logf("%s  pos: %v, %v\n", pad, x, y)
	t.Logf("%s  norm: %v, %v\n", pad, nx, ny)
	t.Logf("%s  size: %v\n", pad, size)
	t.Logf("%s  UL: %v\n", pad, IsUpperLeft(key))
}

func TestTree(t *testing.T) {
	var root uint32
	_, _, lvl := Decode(root)
	if lvl != 0 {
		t.Fail()
	}
}

func TestLength(t *testing.T) {
	var nodes []uint32
	lvl := 10
	Split(0, lvl, &nodes)
	if len(nodes) != Cap(lvl) {
		t.Fatalf("Expected %v but got %v\n", Cap(lvl), len(nodes))
	}
}

func TestPrint(t *testing.T) {
	var x uint32
	// 1110 0010
	// x |= 1 << 1
	// x |= 1 << 5
	// x |= 1 << 6
	// x |= 1 << 7

	var nodes []uint32
	Split(x, 1, &nodes)
	for i, n := range nodes {
		printWord(t, "  ", fmt.Sprintf("%v", i), n)
	}
}

func BenchmarkSplit11(b *testing.B) {
	for n := 0; n < b.N; n++ {
		nodes := make([]uint32, 0, Cap(11))
		Split(0, 11, &nodes)
	}
}

func BenchmarkChildren(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _, _, _ = Children(0)
	}
}

func BenchmarkParent(b *testing.B) {
	a, _, _, _ := Children(0)
	for n := 0; n < b.N; n++ {
		_ = Parent(a)
	}
}
