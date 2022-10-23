package set

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

const N, L = 100, 100

var uniq, dups []string

func init() {
	uniq, dups = make([]string, N), make([]string, N)
	for i := 0; i < N; i++ {
		var s string
		for j := i; j < i+L; j++ {
			s += string(rune(j))
		}
		uniq[i] = s
		if i%2 != 0 {
			s = uniq[i-1]
		}
		dups[i] = s
	}

	for i, j := range rand.Perm(N) {
		uniq[i], uniq[j] = uniq[j], uniq[i]
		dups[i], dups[j] = dups[j], dups[i]
	}

	for i, a := range uniq {
		for j, b := range uniq {
			if i != j && a == b {
				panic("init failed to produce unique string values")
			}
		}
	}
}

func TestGenericSliceInsert(t *testing.T) {
	var a Slice[string]
	for i, s := range uniq {
		_, ok := a.Insert(s)
		if !ok {
			t.Fatalf("Insert(uniq[%v]) failed", i)
		}
		if !sort.StringsAreSorted(a) {
			t.Fatal("sort.StringsAreSorted returned false")
		}
	}
	if have, want := len(a), len(uniq); have != want {
		t.Fatalf("Unexpected len after inserts; have %v, want %v.", have, want)
	}

	var b Slice[string]
	for _, s := range dups {
		_, _ = b.Insert(s)
		if !sort.StringsAreSorted(b) {
			t.Fatal("sort.StringsAreSorted returned false")
		}
	}
	if have, want := len(b), N/2; have != want {
		t.Fatalf("Unexpected len after dup inserts; have %v, want %v.", have, want)
	}
}

func TestFilter(t *testing.T) {
	a := []string{
		"a", "b", "c",
		"b", "c", "d",
		"c", "d", "e",
	}
	Filter(&a)
	have := fmt.Sprintf("%v", a)
	want := "[a b c d e]"
	if have != want {
		t.Fatalf("have %v, want %v.", have, want)
	}
}

func BenchmarkFilter(b *testing.B) {
	b.ReportAllocs()
	p := []string{
		"a", "b", "c",
		"b", "c", "d",
		"c", "d", "e",
	}

	for i := 0; i < b.N; i++ {
		q := make([]string, len(p))
		copy(q, p)
		Filter(&q)
	}
}

func BenchmarkUniq_Generic_Slice(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var a Slice[string]
		for _, s := range uniq {
			a.Insert(s)
		}
	}
}

func BenchmarkDups_Generic_Slice(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var a Slice[string]
		for _, s := range dups {
			a.Insert(s)
		}
	}
}

func BenchmarkUniq_String_Map(b *testing.B) {
	for n := 0; n < b.N; n++ {
		m := make(map[string]struct{})
		for _, s := range uniq {
			m[s] = struct{}{}
		}
	}
}

func BenchmarkDups_String_Map(b *testing.B) {
	for n := 0; n < b.N; n++ {
		m := make(map[string]struct{})
		for _, s := range dups {
			m[s] = struct{}{}
		}
	}
}
