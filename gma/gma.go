// Package gma provides naive primitives for geometric algebra.
package gma

import (
	"fmt"
	"math"
	"math/bits"
)

/*

Geometric Algebra optimization problems

https://arxiv.org/pdf/1305.5663.pdf
https://www.researchgate.net/publication/232654491_2D_Geometric_Constraint_Solving_An_Overview
https://arxiv.org/pdf/1608.03450.pdf
http://citeseerx.ist.psu.edu/viewdoc/download?doi=10.1.1.42.4508&rep=rep1&type=pdf
GACS p346
http://geocalc.clas.asu.edu/GA_Primer/GA_Primer/standard-algebraic-tools/parametric-equations/solution-linear-constraints.html




TODO just embed multiplication table for geometric product instead of the looping in signOf

===============

 NEXT

* clean up source
* types for 2D and 3D, keep general Multivector around too
* Review chapter 10.+
* Rotation Interpolation, 10.3.4, 10.7.1
* vector space model (chapter 10) is all about directions.
  Would a directional vector be enough to establish constraints for layout purposes ???

         ^
  A ---->|
         B

  So, to say "B is to the right of A" is to say that A's direction is as follows:

    (a) points right
    (b) not parallel with B
    (c) some kind of magnitude, obviously, that affects B or vice-versa depending on layout config

  Have been using magnitude as "length" for drawing, but maybe it could play interim role
  for balancing constraints, or using the norm and then adjusting magnitude/scale.
*/

var (
	E1 = uint8(1)
	E2 = uint8(1 << 1)
	E3 = uint8(1 << 2)
	I2 = Blade{1, E1 ^ E2}
	I3 = Blade{1, E1 ^ E2 ^ E3}

	ZB = Blade{}
)

// Scalar returns a 0-grade Blade.
func Scalar(x float64) Blade { return Blade{Scalar: x} }

// TODO for 2D, no need to sandwhich vector; see 7.3.1
func Rotor(angle float64, basis uint8) Multivector {
	return Multivector{
		Scalar(math.Cos(angle / 2)),
		{-math.Sin(angle / 2), basis},
	}
}

type Blade struct {
	Scalar float64

	// Basis is a bitmap of independent vectors, if any; vectors must be in
	// canonical ordering so account for sign changes of Scalar when specifying.
	Basis uint8
}

// Grade returns the number of independent vectors of Blade.
func (a Blade) Grade() int {
	return bits.OnesCount8(a.Basis)
}

// Wedge returns the outer product of a^b; a zero product if a and b are
// dependent, otherwise the geometric product.
func (a Blade) Wedge(b Blade) Blade {
	if a.Basis&b.Basis != 0 {
		return ZB
	}
	return a.Mul(b)
}

// Mul returns the geometric product of ab; assumes orthonormal bases and
// annihilates dependent vectors.
func (a Blade) Mul(b Blade) Blade {
	return Blade{signOf(a.Basis, b.Basis) * a.Scalar * b.Scalar, a.Basis ^ b.Basis}
}

func (a Blade) Angle(b Blade) float64 {
	return a.Mul(b).Scalar / (a.Norm() * b.Norm())
}

func (a Blade) Norm() float64 {
	return math.Sqrt(a.NormSq())
}

func (a Blade) NormSq() float64 {
	return a.Scalar * a.Rev().Scalar
}

func (a Blade) Inverse() Blade {
	a.Scalar /= a.NormSq()
	return a
}

func (a Blade) Rev() Blade {
	if a.Grade()%4 > 1 {
		a.Scalar *= -1
	}
	return a
}

func (a Blade) Invol() Blade {
	if a.Grade()%2 == 1 {
		a.Scalar *= -1
	}
	return a
}

// TODO check
func (a Blade) Conj() Blade {
	if x := a.Grade() % 4; x == 0 || x == 3 {
		a.Scalar *= -1
	}
	return a
}

// Lc returns the left contraction of a onto b.
func (a Blade) Lc(b Blade) Blade {
	if a.Grade() <= b.Grade() && a.Basis&b.Basis == a.Basis {
		return a.Mul(b)
	}
	return ZB
}

// TODO func (a Blade) RightContraction

// TODO return instead something like: 0.8*e1^e2
func (a Blade) String() string {
	return fmt.Sprintf("Blade(%v, %08b)", a.Scalar, a.Basis)
}

// TODO func ScalarProduct(a, b Blade) []float64

func signOf(a, b uint8) float64 {
	a = a >> 1
	n := 0
	for a != 0 {
		n += bits.OnesCount8(a & b)
		a = a >> 1
	}
	if n&1 == 0 {
		return 1
	}
	return -1
}

type Multivector []Blade

func (a Multivector) Wedge(b Multivector) Multivector {
	var c Multivector
	for _, b0 := range a {
		for _, b1 := range b {
			c = append(c, b0.Wedge(b1))
		}
	}
	return simplify(c)
}

func (a Multivector) Mul(b Multivector) Multivector {
	var c Multivector
	for _, b0 := range a {
		for _, b1 := range b {
			c = append(c, b0.Mul(b1))
		}
	}
	return simplify(c)
}

func (a Multivector) Add(b Multivector) Multivector {
	c := make(Multivector, len(a))
	copy(c, a)
	return simplify(append(c, b...))
}

func (a Multivector) Inverse() Multivector {
	var b Multivector
	n := a.NormSq()
	for _, v := range a {
		v.Scalar /= n
		b = append(b, v)
	}
	return b
}

func (a Multivector) Rev() Multivector {
	var b Multivector
	for _, v := range a {
		b = append(b, v.Rev())
	}
	return b
}

func simplify(a Multivector) Multivector {
	m := make(map[uint8]float64)
	for _, v := range a {
		m[v.Basis] += v.Scalar
	}

	var b Multivector
	for k, v := range m {
		if v != 0 {
			b = append(b, Blade{Scalar: v, Basis: k})
		}
	}

	// sort.Slice(b, func(i, j int) bool {
	// 	return b[i].Basis < b[j].Basis
	// })
	return b
}

func (a Multivector) Lc(b Multivector) Multivector {
	var c Multivector
	for _, b0 := range a {
		for _, b1 := range b {
			w := b0.Lc(b1)
			if w != ZB {
				c = append(c, w)
			}
		}
	}
	return simplify(c)
}

func (a Multivector) ScalarProduct(b Multivector) float64 {
	return a.Lc(b).Scalar()
}

func (a Multivector) NormE() float64 {
	s := a.ScalarProduct(a.Rev())
	if s < 0 {
		return 0
	}
	return math.Sqrt(s)
}

func (a Multivector) NormE2() float64 {
	s := a.ScalarProduct(a.Rev())
	if s < 0 {
		return 0
	}
	return s
}

func (a Multivector) Norm() float64 {
	return math.Sqrt(a.NormSq())
}

func (a Multivector) NormSq() float64 {
	var n float64
	for _, v := range a {
		if v.Grade()%4 > 1 {
			n -= v.Scalar * v.Scalar
		} else {
			n += v.Scalar * v.Scalar
		}
	}
	return n
}

func (a Multivector) Angle(b Multivector) float64 {
	const fac = 1 / (math.Pi / 180)
	return fac * math.Acos(a.Mul(b).Scalar()/(a.Norm()*b.Norm()))
}

func (a Multivector) ScalarOf(basis uint8) float64 {
	for _, v := range a {
		if v.Basis == basis {
			return v.Scalar
		}
	}
	return 0
}

func (a Multivector) Scalar() float64 { return a.ScalarOf(0) }

func (a Multivector) E1() float64 { return a.ScalarOf(E1) }

func (a Multivector) E2() float64 { return a.ScalarOf(E2) }

func (a Multivector) E3() float64 { return a.ScalarOf(E3) }
