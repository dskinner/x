package set

import (
	"math/rand"
	"sort"
	"testing"
)

const N, L = 100000, 100

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
}

func TestStringSliceInsert(t *testing.T) {
	var a StringSlice
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

	var b StringSlice
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

func TestReflectInsert(t *testing.T) {
	var a []string
	for i, s := range uniq {
		_, ok := Insert(&a, s, func(i int) bool { return a[i] >= s })
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

	type w struct {
		s string
		f func(float64) float64
	}
	var b []w
	for _, s := range dups {
		_, _ = Insert(&b, w{s: s}, func(i int) bool { return b[i].s >= s })
		if !sort.SliceIsSorted(b, func(i, j int) bool { return b[i].s < b[j].s }) {
			t.Fatal("sort.SliceIsSorted returned false")
		}
	}
	if have, want := len(b), N/2; have != want {
		t.Fatalf("Unexpected len after dup inserts; have %v, want %v.", have, want)
	}
}

func BenchmarkUniq_String_Slice(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var a StringSlice
		for _, s := range uniq {
			a.Insert(s)
		}
	}
}

func BenchmarkDups_String_Slice(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var a StringSlice
		for _, s := range dups {
			a.Insert(s)
		}
	}
}

func BenchmarkUniq_String_Wacky(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var a StringSlice
		for _, s := range uniq {
			a = append(a, s)
		}
		a.Filter()
	}
}

func BenchmarkDups_String_Wacky(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var a StringSlice
		for _, s := range dups {
			a = append(a, s)
		}
		a.Filter()
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

func BenchmarkUniq_Reflect(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var a []string
		for _, s := range uniq {
			Insert(&a, s, func(i int) bool { return a[i] >= s })
		}
	}
}

func BenchmarkDups_Reflect(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var a []string
		for _, s := range dups {
			Insert(&a, s, func(i int) bool { return a[i] >= s })
		}
	}
}
