package set

import (
	"sort"
	"testing"
)

const N = 1000

var uniq, dups []string

func init() {
	uniq, dups = make([]string, N), make([]string, N)
	for i := 0; i < N; i++ {
		var s string
		for j := i; j < i+N; j++ {
			s += string(rune(j))
		}
		uniq[i] = s
		if dupit(i) {
			s = uniq[i-1]
		}
		dups[i] = s
	}
}

func dupit(i int) bool { return i%2 != 0 }

func TestStringSliceInsert(t *testing.T) {
	var a StringSlice
	for i, s := range uniq {
		j, ok := a.Insert(s)
		if !ok {
			t.Fatalf("Insert(uniq[%v]) failed", i)
		}
		if i != j {
			t.Fatalf("Insert(uniq[%v]) inserted at %v", i, j)
		}
		if !sort.StringsAreSorted(a) {
			t.Fatal("sort.StringsAreSorted returned false")
		}
	}
	if have, want := len(a), len(uniq); have != want {
		t.Fatalf("Unexpected len after inserts; have %v, want %v.", have, want)
	}

	var b StringSlice
	for i, s := range dups {
		j, ok := b.Insert(s)
		if dupit(i) && ok {
			t.Fatalf("Inserted at %v when expecting dup at %v.", j, i)
		}
		if !sort.StringsAreSorted(b) {
			t.Fatal("sort.StringsAreSorted returned false")
		}
	}
	if have, want := len(b), N/2; have != want {
		t.Fatalf("Unexpected len after dup inserts; have %v, want %v.", have, want)
	}
}

func BenchmarkStringSliceFilter(b *testing.B) {
	z := make([]string, len(dups))
	copy(z, dups)
	a := StringSlice(z[:0])

	b.ReportAllocs()
	b.ResetTimer()

	for _, x := range z {
		a.Insert(x)
	}
	if len(a) != N/2 {
		b.Fail()
	}
}

func BenchmarkStringSliceInsertUniq(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		var a StringSlice
		for _, s := range uniq {
			a.Insert(s)
		}
	}
}

func BenchmarkStringSliceInsertDups(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		var a StringSlice
		for _, s := range dups {
			a.Insert(s)
		}
	}
}

func BenchmarkMapStringInsertUniq(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		m := make(map[string]struct{})
		for _, s := range uniq {
			m[s] = struct{}{}
		}
	}
}

func BenchmarkMapStringInsertDups(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		m := make(map[string]struct{})
		for _, s := range dups {
			m[s] = struct{}{}
		}
	}
}
