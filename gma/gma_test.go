package gma

import (
	"testing"
)

var (
	e1 = Blade{1, E1}
	e2 = Blade{1, E2}
	e3 = Blade{1, E3}
)

func TestBlade(t *testing.T) {
	A := e1.Wedge(e2)
	B := e2.Wedge(e1)
	C := e3.Wedge(e1)
	D := e3

	t.Logf("grade(e1^e2) = %v", A.Grade())
	t.Logf("       e1^e2 = %08b", A.Basis)
	t.Logf("grade(e2^e1) = %v", B.Grade())
	t.Logf("grade(e3^e1) = %v", C.Grade())
	t.Logf("   grade(e3) = %v", D.Grade())
	t.Logf("          e3 = %08b", D.Basis)

	t.Logf("Op (e1^e2)^(e3^e1) = %s", A.Wedge(C))
	t.Logf("Op      (e2^e1)^e3 = %s", B.Wedge(D))

	t.Logf("Gp (e1^e2)^(e3^e1) = %s", A.Mul(C))
	t.Logf("Gp      (e2^e1)^e3 = %s", B.Mul(D))

	// the geometric product of a basis vector with itself evaluates
	// to a scalar derived from the metric:
	//  e3e3 = e3 dot e3 + e3^e3 = Q[e3, e3]
	t.Logf("Gp           e3^e3 = %s", D.Mul(D))
}

func TestMultiply(t *testing.T) {
	e11 := e1.Mul(e1)
	if e11.Basis != 0 || e11.Scalar != 1 {
		t.Errorf("expected scalar 1, have %s", e11)
	}

	e12 := e1.Mul(e2)
	if e12.Basis != 0x3 || e12.Scalar != 1 {
		t.Errorf("expected bivector, have %s", e12)
	}

	e12e12 := e12.Mul(e12)
	if e12e12.Basis != 0 || e12e12.Scalar != -1 {
		t.Errorf("expected scalar -1, have %s", e12e12)
	}

	t.Logf("    e1 = %s", e1)
	t.Logf("    e2 = %s", e2)
	t.Logf("  e1e1 = %s", e11)
	t.Logf("  e1e2 = %s", e12)
	t.Logf("e12e12 = %s", e12e12)
	t.Logf(" e12e1 = %s", e12.Mul(e1)) // multiple of e2

	// multiply
	a := Multivector{{1, E1}, {1, E2}}
	b := Multivector{{0, E1}, {1, E2}}

	t.Logf("a.NormSq() = %v", a.NormSq())
	t.Logf("b.NormSq() = %v", b.NormSq())
	t.Logf("a.Angle(b) = %v", a.Angle(b))

	ab := a.Mul(b)
	aa := a.Mul(a)
	aI := a.Inverse()
	aIa := aI.Mul(a)
	t.Logf("     a = %s", a)
	t.Logf("     b = %s", b)
	t.Logf("    ab = %s", ab)
	t.Logf("    aa = %s", aa)
	t.Logf("    aI = %s", aI)
	t.Logf("   aIa = %s", aIa)

	t.Logf("  ab/b = %s", ab.Mul(b.Inverse()))

	c := Blade{3, E1}
	cI := c.Inverse()
	t.Logf("     c = %s", c)
	t.Logf("    cI = %s", cI)
	t.Logf("   c/c = %s", cI.Mul(c))

	d := Blade{5, E1 ^ E2}
	dI := d.Inverse()
	t.Logf("     d = %s", d)
	t.Logf("    d~ = %s", d.Rev())
	t.Logf("   dd~ = %s", d.Mul(d.Rev()))
	t.Logf("    dI = %s", dI)
	t.Logf("   d/d = %s", dI.Mul(d))

	d = Blade{7, E1 ^ E2 ^ E3}
	dI = d.Inverse()
	t.Logf("     d = %s", d)
	t.Logf("    dI = %s", dI)
	t.Logf("   d/d = %s", dI.Mul(d))

	// division
	// see 6.1.4
	// x := ab.Mul(b.Invol())
	// if x != a {
	// t.Errorf("division failed, want %s, have %s", a, x)
	// }
	// t.Logf(" ab/b` = %s", x)

	// ratios of vectors as operators
	// t.Logf("    e1  = %s", e1)
	// t.Logf("    e1` = %s", e1.Invol())
	// t.Logf(" e1 e1` = %s", e1.Mul(e1.Invol()))

	// c = xb

	// a_b := a.Wedge(b)
	// t.Logf("    a^b = %s", a_b)
	// b.s *= -1
	// b_a := b.Wedge(a)
	// t.Logf("   -b^a = %s", b_a)

	// t.Logf("a(e1) = %s", a.Mul(e12))
	// t.Logf("e12e12 = %s", e12.Mul(e12))
	// t.Logf("e12e12 = %s", e12.Mul(e12))
	// t.Logf("e12e12 = %s", e12.Mul(e12))
}

func TestDual(t *testing.T) {
	t.Logf("I2 : %s", I2)
	t.Logf("I2~: %s", I2.Rev())

	t.Logf("I3 : %s", I3)
	t.Logf("I3~: %s", I3.Rev())

	a := Blade{2, E1}
	aD := a.Lc(I2.Rev())
	aDD := aD.Lc(I2.Rev())
	t.Logf("    a: %s", a)
	t.Logf("   a*: %s", aD)
	t.Logf("(a*)*: %s", aDD)

	b := Multivector{{2, E1}, {4, E2}, {8, E3}}

	bD := b.Lc(Multivector{I3.Rev()})
	for i, v := range bD {
		t.Logf("bD %v: %s", i, v)
	}

	A := e3
	B := e1
	t.Log()
	t.Log("(A^B)* = A](B*)")
	t.Logf("(A^B)* = %s", A.Wedge(B).Lc(I3.Rev()))
	t.Logf("A](B*) = %s", A.Lc(B.Lc(I3.Rev())))

	t.Log()
	t.Log("(A]B)* = A^(B*)")
	A = e1.Wedge(e2)
	B = e1.Wedge(e3).Wedge(e2)
	t.Logf("(A]B)* = %s", A.Lc(B).Lc(I3.Rev()))
	t.Logf("A^(B*) = %s", A.Wedge(B.Lc(I3.Rev())))
}

func TestContraction(t *testing.T) {
	a := Scalar(2)
	A, B, C := e1, e2, I3

	// a]B = aB
	if v0, v1 := a.Lc(B), a.Mul(B); v0 != v1 {
		t.Logf("want: a]B = aB\na]B = %s\n aB = %s", v0, v1)
	}

	// B]a = 0
	if v0 := B.Lc(a); v0 != (Blade{}) {
		t.Errorf("want: B]a = 0\n%s", v0)
	}

	u := Multivector{{2, E1}, {3, E2}}
	v := Multivector{{2, E1}, {3, E2}}
	// u]v = u dot v
	t.Logf("u]v = %s", u.Lc(v))

	// u](B^A) = (u]B)^A + (-1**grade(B))B^(u]A)
	t.Logf("    u](B^A) = %s", u.Lc(Multivector{B.Wedge(A)}))
	t.Logf("    (u]B)^A = %s", u.Lc(Multivector{B}).Wedge(Multivector{A}))
	t.Logf("-1(B^(u]A)) = %s", Multivector{B}.Wedge(u.Lc(Multivector{A})).Wedge(Multivector{Scalar(-1)}))

	// (A^B)]C = A](B]C)
	if v0, v1 := A.Wedge(B).Lc(C), A.Lc(B.Lc(C)); v0 != v1 {
		t.Errorf("want: (A^B)]C = A](B]C)\n%s\n%s", v0, v1)
	}
}

func TestVectors(t *testing.T) {
	// (a1*b1 + a2*b2) + (a1*b2 - a2*b1)e12
	//
	// a1 = 2
	// a2 = 3
	// b1 = 2
	// b2 = 3
	//
	// (2*2 + 3*3) + (2*3 - 3*2)e12
	// 13 + 0e12
	// 13

	a := Multivector{{2, E1}, {3, E2}}
	b := Multivector{{2, E1}, {3, E2}}
	p := a.Mul(b)
	t.Logf("%s", p)
}
