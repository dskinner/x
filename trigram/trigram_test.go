package trigram

import (
	"testing"
)

var big []string

func init() {
	const N = 100
	big = make([]string, N)
	for i := 0; i < N; i++ {
		var s string
		for j := i; j < i+N; j++ {
			s += string(rune(j))
		}
		big[i] = s
	}
}

func TestSet(t *testing.T) {
	var gs Set
	gs.Index([]string{"the quick", "red fox", "jumps over", "the lazy", "brown dog"}...)

	if gs.Mapping == nil || gs.Fields == nil {
		t.Fatalf("Set did not initialize correctly, have %+v", gs)
	}
	if haveks, havevs, want := len(gs.ks), len(gs.vs), 44; haveks != want || havevs != want {
		t.Fatalf("Unexpected lengths, have keys %v and values %v, want %v", haveks, havevs, want)
	}
	m, u := gs.Match("bog", 0.33)
	if len(m) != len(u) {
		t.Fatalf("Lengths for matches, %v, and unit scores, %v, don't match.", len(m), len(u))
	}
	if len(m) == 0 {
		t.Fatal("Match returned zero results.")
	}
}

func BenchmarkSet(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		var gs Set
		gs.Index(big...)
		m, u := gs.Match("abcd", 0.33)
		if len(m) != len(u) || len(m) == 0 {
			b.Fatalf("Unexpected lengths for m, %v, and u, %v.", len(m), len(u))
		}
	}
}
