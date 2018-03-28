package geom

import (
	"testing"
)

var (
	e1 = Blade{bitmap: 1, s: 1}
	e2 = Blade{bitmap: 1 << 1, s: 1}
	e3 = Blade{bitmap: 1 << 2, s: 1}
	I2 = e1.Op(e2)
	I3 = e1.Op(e2).Op(e3)
)

func TestBlade(t *testing.T) {
	A := e1.Op(e2)
	B := e2.Op(e1)
	C := e3.Op(e1)
	D := e3

	t.Logf("grade(e1^e2) = %v", A.Grade())
	t.Logf("       e1^e2 = %08b", A.bitmap)
	t.Logf("grade(e2^e1) = %v", B.Grade())
	t.Logf("grade(e3^e1) = %v", C.Grade())
	t.Logf("   grade(e3) = %v", D.Grade())
	t.Logf("          e3 = %08b", D.bitmap)

	t.Logf("Op (e1^e2)^(e3^e1) = %s", A.Op(C))
	t.Logf("Op      (e2^e1)^e3 = %s", B.Op(D))

	t.Logf("Gp (e1^e2)^(e3^e1) = %s", A.Gp(C))
	t.Logf("Gp      (e2^e1)^e3 = %s", B.Gp(D))

	// the geometric product of a basis vector with itself evaluates
	// to a scalar derived from the metric:
	//  e3e3 = e3 dot e3 + e3^e3 = Q[e3, e3]
	t.Logf("Gp           e3^e3 = %s", D.Gp(D))
}

func TestMultiply(t *testing.T) {
	e11 := e1.Gp(e1)
	if e11.bitmap != 0 || e11.s != 1 {
		t.Errorf("expected scalar 1, have %s", e11)
	}

	e12 := e1.Gp(e2)
	if e12.bitmap != 0x3 || e12.s != 1 {
		t.Errorf("expected bivector, have %s", e12)
	}

	e12e12 := e12.Gp(e12)
	if e12e12.bitmap != 0 || e12e12.s != -1 {
		t.Errorf("expected scalar -1, have %s", e12e12)
	}

	t.Logf("    e1 = %s", e1)
	t.Logf("    e2 = %s", e2)
	t.Logf("  e1e1 = %s", e11)
	t.Logf("  e1e2 = %s", e12)
	t.Logf("e12e12 = %s", e12e12)
	t.Logf(" e12e1 = %s", e12.Gp(e1)) // multiple of e2

	// multiply
	a := Multivector{Blade{e1.bitmap, 1}, Blade{e2.bitmap, 1}}
	b := Multivector{Blade{e1.bitmap, 0}, Blade{e2.bitmap, 1}}
	t.Logf("a.Norm() = %v", a.Norm())
	t.Logf("b.Norm() = %v", b.Norm())
	t.Logf("a.Angle(b) = %v", a.Angle(b))
	ab := a.Gp(b)
	aa := a.Gp(a)
	aI := a.Inverse()
	// aI[0].s = 1 / aI[0].s
	// aI := Multivector{Blade{e1.bitmap, -2}, Blade{e2.bitmap, -1}}
	aIa := aI.Gp(a)
	t.Logf("     a = %s", a)
	t.Logf("     b = %s", b)
	t.Logf("    ab = %s", ab)
	t.Logf("    aa = %s", aa)
	t.Logf("    aI = %s", aI)
	t.Logf("   aIa = %s", aIa)
	t.Logf("  ab/b = %s", ab.Gp(b.Inverse()))

	c := Blade{1, 3}
	cI := c.Inverse()
	t.Logf("     c = %s", c)
	t.Logf("    cI = %s", cI)
	t.Logf("   c/c = %s", cI.Gp(c))

	d := e1.Gp(e2)
	d.s = 5
	dI := d.Inverse()
	t.Logf("     d = %s", d)
	t.Logf("    d~ = %s", d.Rev())
	t.Logf("   dd~ = %s", d.Gp(d.Rev()))
	t.Logf("    dI = %s", dI)
	t.Logf("   d/d = %s", dI.Gp(d))

	d = e1.Gp(e2).Gp(e3)
	d.s = 7
	dI = d.Inverse()
	t.Logf("     d = %s", d)
	t.Logf("    dI = %s", dI)
	t.Logf("   d/d = %s", dI.Gp(d))

	// division
	// see 6.1.4
	// x := ab.Gp(b.Invol())
	// if x != a {
	// t.Errorf("division failed, want %s, have %s", a, x)
	// }
	// t.Logf(" ab/b` = %s", x)

	// ratios of vectors as operators
	// t.Logf("    e1  = %s", e1)
	// t.Logf("    e1` = %s", e1.Invol())
	// t.Logf(" e1 e1` = %s", e1.Gp(e1.Invol()))

	// c = xb

	// a_b := a.Op(b)
	// t.Logf("    a^b = %s", a_b)
	// b.s *= -1
	// b_a := b.Op(a)
	// t.Logf("   -b^a = %s", b_a)

	// t.Logf("a(e1) = %s", a.Gp(e12))
	// t.Logf("e12e12 = %s", e12.Gp(e12))
	// t.Logf("e12e12 = %s", e12.Gp(e12))
	// t.Logf("e12e12 = %s", e12.Gp(e12))
}

func TestDual(t *testing.T) {
	t.Logf("I2 : %s", I2)
	t.Logf("I2~: %s", I2.Rev())

	t.Logf("I3 : %s", I3)
	t.Logf("I3~: %s", I3.Rev())

	a := Blade{bitmap: 1, s: 2}
	aD := a.Lc(I2.Rev())
	aDD := aD.Lc(I2.Rev())
	t.Logf("    a: %s", a)
	t.Logf("   a*: %s", aD)
	t.Logf("(a*)*: %s", aDD)

	b := Multivector{
		Blade{bitmap: 1, s: 2},
		Blade{bitmap: 1 << 1, s: 4},
		Blade{bitmap: 1 << 2, s: 8},
	}

	bD := b.Lc(I3.Rev())
	for i, v := range bD {
		t.Logf("bD %v: %s", i, v)
	}

	A := e3
	B := e1
	t.Log()
	t.Log("(A^B)* = A](B*)")
	t.Logf("(A^B)* = %s", A.Op(B).Lc(I3.Rev()))
	t.Logf("A](B*) = %s", A.Lc(B.Lc(I3.Rev())))

	t.Log()
	t.Log("(A]B)* = A^(B*)")
	A = e1.Op(e2)
	B = e1.Op(e3).Op(e2)
	t.Logf("(A]B)* = %s", A.Lc(B).Lc(I3.Rev()))
	t.Logf("A^(B*) = %s", A.Op(B.Lc(I3.Rev())))
}

func Sca(x float64) Blade {
	return Blade{s: x}
}

func TestContraction(t *testing.T) {
	a := Sca(2)
	A, B, C := e1, e2, I3

	// a]B = aB
	if v0, v1 := a.Lc(B), a.Gp(B); v0 != v1 {
		t.Logf("want: a]B = aB\na]B = %s\n aB = %s", v0, v1)
	}

	// B]a = 0
	if v0 := B.Lc(a); v0 != (Blade{}) {
		t.Errorf("want: B]a = 0\n%s", v0)
	}

	u := Multivector{e1.Gp(Sca(2)), e2.Gp(Sca(3))}
	v := Multivector{e1.Gp(Sca(2)), e2.Gp(Sca(3))}
	// u]v = u dot v
	t.Logf("u]v = %s", u.Lcv(v))

	// u](B^A) = (u]B)^A + (-1**grade(B))B^(u]A)
	t.Logf("    u](B^A) = %s", u.Lc(B.Op(A)))
	t.Logf("    (u]B)^A = %s", u.Lc(B).Op(Multivector{A}))
	t.Logf("-1(B^(u]A)) = %s", Multivector{B}.Op(u.Lc(A)).Op(Multivector{Sca(-1)}))

	// (A^B)]C = A](B]C)
	if v0, v1 := A.Op(B).Lc(C), A.Lc(B.Lc(C)); v0 != v1 {
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

	a := Multivector{
		Blade{1, 2},
		Blade{1 << 1, 3},
		// Blade{1 << 2, 4},
	}

	b := Multivector{
		Blade{1, 2},
		Blade{1 << 1, 3},
		// Blade{1 << 2, 4},
	}

	p := a.Gp(b)
	t.Logf("%s", p)
}
