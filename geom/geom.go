// Package geom provides naive primitives for geometric algebra.

/* basic reminders
u = ae₁ + be₂
u(e₁e₂) = (ae₁ + be₂)e₁e₂
        = ae₁e₁e₂ + be₂e₁e₂
        = ae₁²e₂ + be₂e₁e₂
        = ae₂ - be₁e₂e₂
        = ae₂ - be₁e₂²
        = ae₂ - be₁
        = -be₁ + ae₂

e₁ = a + b
e₂ = c + d
 u = x + y

u(e₁e₂) = (xe₁ + ye₂)e₁e₂
        = (x(a+b) + y(c+d))(a+b)(c+d)
        = (xa + xb + yc + yd)(a + b)(c + d)
        = (xaa + xab + yac + yad + xab + xbb + ybc + ybd)(c + d)
        = (xaac + xabc + yacc + yacd + xabc + xbbc + ybcc + ybcd + xaad + xabd + yacd + yadd + xabd + xbbd + ybcd + ybdd)

// geometric product of two vectors; scalar + bivector (bivector may be weighted by second scalar)
u(e₁) = (ae₁ + be₂)e₁
      = ae₁e₁ + be₂e₁
      = ae₁² + be₂e₁
      = a - be₁e₂

// geometric product of geometric product result; vector (depending ...)
(a - be₁e₂)(e₂) = ae₂ - be₁e₂e₂
                = ae₂ - be₁
                = -be₁ + ae₂
*/
package geom

import (
	"fmt"
	"math"
	"math/bits"
	"sort"
)

// TODO rename type to something more generic?
//
// Consider ℝ², here's some examples:
// Except for scalars, the s field is being set to establish the sign, positive or negative.
// Scalar:
//   a = Blade{bitmap: 0, s: 1}
// Vector:
//   e1 = Blade{bitmap: 1, s: 1}
//   e2 = Blade{bitmap: 1<<1, s: 1}
// Bivector:
//   e1^e2 = Blade{bitmap: e1.bitmap^e2.bitmap, s: 1}
//
// The multivector is then:
//   {a, e1, e2, e1^e2}
// The outer product of the multivector is simply:
//   a + e1 + e2 + e1^e2
// This is the same as:
//   (a + e1)^(a + e2)
// This is why all scalars
type Blade struct {
	// scalar   00000000
	// e1       00000001
	// e2       00000010
	// e1^e2    00000011
	// e3       00000100
	bitmap uint8
	s      float64
}

// vectors of blade must be in canonical order as this sets sign of Blade positive.
func NewBlade(blade uint8) Blade {
	return Blade{blade, 1}
}

func (a Blade) Grade() int {
	return bits.OnesCount8(a.bitmap)
}

// Op produces a zero result if a and b are dependent, otherwise the
// geometric product is returned given e1^e2 = e1e2.
func (a Blade) Op(b Blade) Blade {
	if a.bitmap&b.bitmap != 0 {
		return Blade{}
	}
	return a.Gp(b)
}

// Gp assumes orthonormal bases and annihilates dependent vectors.
func (a Blade) Gp(b Blade) Blade {
	bitmap := a.bitmap ^ b.bitmap
	sign := signOf(a.bitmap, b.bitmap)
	return Blade{bitmap, sign * a.s * b.s}
}

func (a Blade) Norm() float64 {
	return a.s * a.Rev().s
}

func (a Blade) Angle(b Blade) float64 {
	na := math.Sqrt(a.Norm())
	nb := math.Sqrt(b.Norm())
	return a.Gp(b).s / (na * nb)
}

func (a Blade) Inverse() Blade {
	// a.s /= a.Gp(a).s
	a.s /= a.Norm()
	return a
}

func (a Blade) Rev() Blade {
	if a.Grade()%4 > 1 {
		a.s *= -1
	}
	return a
}

func (a Blade) Invol() Blade {
	if a.Grade()%2 == 1 {
		a.s *= -1
	}
	return a
}

// TODO check
func (a Blade) Conj() Blade {
	if x := a.Grade() % 4; x == 0 || x == 3 {
		a.s *= -1
	}
	return a
}

// Lc returns the left contraction of a onto b.
func (a Blade) Lc(b Blade) Blade {
	if a.Grade() <= b.Grade() && a.bitmap&b.bitmap == a.bitmap {
		return a.Gp(b)
	}
	return Blade{}
}

func (a Blade) String() string {
	return fmt.Sprintf("Blade(%08b, %v)", a.bitmap, a.s)
}

// TODO
// ScalarProduct maps a pair of k-blades to real numbers. If the blades are of
// unequal grades, the result is 0.
// func ScalarProduct(a, b Blade) []float64 {
//     if a.Grade() != b.Grade() {
//         return nil
//     }
// }

// TODO
// func RightContraction

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

func Rotor(angle float64, basis Blade) Multivector {
	return Multivector{
		Blade{s: math.Cos(angle / 2)},
		Blade{bitmap: basis.bitmap, s: -math.Sin(angle / 2)},
	}
}

type Multivector []Blade

// TODO optimize by actually defining Multivector as
func (a Multivector) Op(b Multivector) Multivector {
	// c := make(Multivector, len(a)*len(b))
	var c Multivector
	for _, b0 := range a {
		for _, b1 := range b {
			c = append(c, b0.Op(b1))
		}
	}
	// TODO simplify list of blades by adding those that are equal up to scale.
	return c
}

func (a Multivector) Gp(b Multivector) Multivector {
	// c := make(Multivector, len(a)*len(b))
	var c Multivector
	for _, b0 := range a {
		for _, b1 := range b {
			c = append(c, b0.Gp(b1))
		}
	}
	// TODO simplify list of blades by adding those that are equal up to scale.
	return simplify(c)
}

func (a Multivector) Inverse() Multivector {
	var b Multivector
	// n := a.Gp(a)[0].s
	n := a.Norm()
	for _, v := range a {
		v.s /= n
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
		m[v.bitmap] += v.s
	}
	var b Multivector
	for k, v := range m {
		if v != 0 {
			b = append(b, Blade{k, v})
		}
	}
	sort.Slice(b, func(i, j int) bool {
		return b[i].bitmap < b[j].bitmap
	})
	return b
}

func (a Multivector) Lcv(b Multivector) Multivector {
	var c Multivector
	for _, b0 := range a {
		for _, b1 := range b {
			w := b0.Lc(b1)
			if w != (Blade{}) {
				c = append(c, w)
			}
		}
	}
	// TODO simplify list of blades by adding those that are equal up to scale.
	return c
}

// argument should be a pseudoscalar ?
// pseduoscalar args should use Rev instead of Inverse ?
func (a Multivector) Lc(b Blade) Multivector {
	var c Multivector
	for _, v := range a {
		w := v.Lc(b)
		// if w.bitmap != 0 {
		if w != (Blade{}) {
			c = append(c, w)
		}
	}
	return c
}

func (a Multivector) Norm() float64 {
	var n float64
	// for _, v := range a.Gp(a) {
	for _, v := range a {
		if v.Grade()%4 > 1 {
			n -= v.s * v.s
		} else {
			n += v.s * v.s
		}
	}
	return n
	// return a.Gp(a)[0].s
}

func (a Multivector) Angle(b Multivector) float64 {
	n := math.Sqrt(a.Norm() * b.Norm())
	return math.Acos(a.Gp(b)[0].s/n) / (math.Pi / 180)
}

func (a Multivector) Add() (x, y float64) {
	const (
		e1 = 1
		e2 = 1 << 1
		I  = e1 ^ e2 // pseudoscalar
	)
	for _, v := range a {
		switch v.bitmap {
		case e1:
			x += v.s
		case e2:
			y += v.s
		case I:
			panic("can't handle bitmap for " + v.String())
		}
	}
	return x, y
}

// func (a Multivector) Dot(b Multivector) float64 {
// 	if len(a) != len(b) {
// 		panic("Dot called with vectors of unequal length")
// 	}
// 	var x float64
// 	for i := 0; i < len(a); i++ {
// 		x += a[i].s * b[i].s
// 	}
// 	return x
// }
