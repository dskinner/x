// Package trigram provides string indexing and matching.
package trigram

import (
	"sort"
	"strings"
	"unicode"

	"dasa.cc/x/set"
)

// Set indexer; zero value is valid.
type Set struct {
	// Mapping function used when calling Parse
	// or defaults to IsSpaceDigitLetterToLower if not set.
	Mapping func(rune) rune

	// Fields function used when calling Parse
	// or defaults to unicode.IsSpace if not set.
	Fields func(rune) bool

	ks set.StringSlice
	vs set.StringChain
}

// Index parses and stores trigrams for each s in xs. Panics if nil.
func (a *Set) Index(xs ...string) {
	if a.Mapping == nil {
		a.Mapping = IsSpaceDigitLetterToLower
	}
	if a.Fields == nil {
		a.Fields = unicode.IsSpace
	}
	for _, s := range xs {
		for _, t := range Parse(s, a.Mapping, a.Fields) {
			i, ok := a.ks.Insert(t)
			a.vs.Upsert(s, i, ok)
		}
	}
}

// Match indexed values for x that meet min threshold; returns matches and unit scores.
func (a Set) Match(x string, min float64) ([]string, []float64) {
	var p set.StringSlice
	var u []float64

	q := Parse(x, a.Mapping, a.Fields)
	for _, s := range q {
		if i := sort.SearchStrings(a.ks, s); i < len(a.ks) && a.ks[i] == s {
			for _, t := range a.vs[i] {
				j, ok := 0, false
				if j, ok = p.Insert(t); ok {
					u = append(u, 0)
					copy(u[j+1:], u[j:])
					u[j] = 0
				}
				u[j]++
			}
		}
	}

	fp, fu := p[:0], u[:0]
	for i, s := range p {
		if w := u[i] / float64(len(q)); min <= w {
			fp, fu = append(fp, s), append(fu, w)
		}
	}
	return fp, fu
}

// Parse returns a slice of trigrams for s after modifying characters according to
// the mapping function followed by word splitting according to fields function.
func Parse(s string, mapping func(rune) rune, fields func(rune) bool) []string {
	var p set.StringSlice
	for _, t := range strings.FieldsFunc(strings.Map(mapping, s), fields) {
		t = "\x00\x00" + t + "\x00"
		for i := 0; i <= len(t)-3; i++ {
			p.Insert(t[i : i+3])
		}
	}
	return p
}

// IsSpaceDigitLetterToLower reports whether rune is a letter, digit, or space character
// as defined by Unicode's White Space property and maps rune to lower case.
func IsSpaceDigitLetterToLower(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
		return unicode.ToLower(r)
	}
	return -1
}

// NoFields always returns false; preserves white space in Parse results.
func NoFields(rune) bool { return false }
