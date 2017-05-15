package octree

import (
	"testing"
	"testing/quick"
)

func TestDilate8(t *testing.T) {
	f := func(x uint8) bool { return Undilate8(Dilate8(x)) == x }
	if x := uint8((1 << 8) - 1); !f(x) {
		t.Fatalf("sanity check: failed on input %0X", x)
	}
	cfg := &quick.Config{MaxCount: 1 << 8}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestInterleave8(t *testing.T) {
	f := func(x, y, z, w uint8) bool {
		a, b, c, d := Deinterleave8(Interleave8(x, y, z, w))
		return a == x && b == y && c == z && d == w
	}
	if x := uint8((1 << 8) - 1); !f(x, x, x, x) {
		t.Fatalf("sanity check: failed on input %0X", x)
	}
	cfg := &quick.Config{MaxCount: 1 << 16}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestDilate16(t *testing.T) {
	f := func(x uint16) bool { return Undilate16(Dilate16(x)) == x }
	if x := uint16((1 << 16) - 1); !f(x) {
		t.Fatalf("sanity check: failed on input %0X", x)
	}
	cfg := &quick.Config{MaxCount: 1 << 16}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestInterleave16(t *testing.T) {
	f := func(x, y, z, w uint16) bool {
		a, b, c, d := Deinterleave16(Interleave16(x, y, z, w))
		return a == x && b == y && c == z && d == w
	}
	if x := uint16((1 << 16) - 1); !f(x, x, x, x) {
		t.Fatalf("sanity check: failed on input %0X", x)
	}
	cfg := &quick.Config{MaxCount: 1 << 16}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestChildren16(t *testing.T) {
	var p []uint64
	Subdivide16(0, 2, &p)
	if want := 64; len(p) != want {
		t.Fatalf("wrong length, have %v, want %v", len(p), want)
	}
	// TODO expand testing
}

func BenchmarkDilate8(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if x := Undilate8(Dilate8(uint8(n))); x != uint8(n) {
			b.Fatalf("undilate8(dilate8(%v)) == %v", uint8(n), x)
		}
	}
}

func BenchmarkDilate16(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if x := Undilate16(Dilate16(uint16(n))); x != uint16(n) {
			b.Fatalf("undilate16(dilate16(%v)) == %v", uint16(n), x)
		}
	}
}

func BenchmarkSubdivide16(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var p []uint64
		Subdivide16(0, 7, &p)
	}
}
