package glw

import (
	"fmt"
	"math"

	"golang.org/x/image/math/f32"
)

// Uton converts unit to norm.
func Uton(u float32) float32 { return 2*u - 1 }

// Ntou converts norm to unit.
func Ntou(n float32) float32 { return (n + 1) / 2 }

func Vec2(v0, v1 float32) f32.Vec2         { return f32.Vec2{v0, v1} }
func Vec3(v0, v1, v2 float32) f32.Vec3     { return f32.Vec3{v0, v1, v2} }
func Vec4(v0, v1, v2, v3 float32) f32.Vec4 { return f32.Vec4{v0, v1, v2, v3} }

// Ortho returns orthographic projection.
func Ortho(l, r float32, b, t float32, n, f float32) f32.Mat4 {
	w, h, d := 2/(r-l), 2/(t-b), 2/(f-n)
	x, y, z := -((l + r) / 2), -((t + b) / 2), (f+n)/2
	return f32.Mat4{
		w, 0, 0, x * w,
		0, h, 0, y * h,
		0, 0, d, z * d,
		0, 0, 0, 1,
	}
}

// ident16fv returns identity matrix.
func ident16fv() f32.Mat4 {
	return f32.Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func rotate16fv(quat f32.Vec4, m f32.Mat4) f32.Mat4 {
	return mul16fv(quat16fv(quat), m)
}

func translate16fv(a f32.Vec4, m f32.Mat4) f32.Mat4 {
	x, y, z := a[0], a[1], a[2]
	return mul16fv(f32.Mat4{
		1, 0, 0, x,
		0, 1, 0, y,
		0, 0, 1, z,
		0, 0, 0, 1,
	}, m)
}

func scale16fv(a f32.Vec3, m f32.Mat4) f32.Mat4 {
	w, h, d := a[0], a[1], a[2]
	return mul16fv(f32.Mat4{
		w, 0, 0, 0,
		0, h, 0, 0,
		0, 0, d, 0,
		0, 0, 0, 1,
	}, m)
}

func lerp3fv(a, b f32.Vec3, t float32) f32.Vec3 {
	return f32.Vec3{
		a[0] + t*(b[0]-a[0]),
		a[1] + t*(b[1]-a[1]),
		a[2] + t*(b[2]-a[2]),
	}
}

func lerp4fv(a, b f32.Vec4, t float32) f32.Vec4 {
	return f32.Vec4{
		a[0] + t*(b[0]-a[0]),
		a[1] + t*(b[1]-a[1]),
		a[2] + t*(b[2]-a[2]),
		a[3] + t*(b[3]-a[3]),
	}
}

func add4fv(a, b f32.Vec4) f32.Vec4 {
	return f32.Vec4{a[0] + b[0], a[1] + b[1], a[2] + b[2], a[3] + b[3]}
}

func mul3fv(a, b f32.Vec3) f32.Vec3 {
	return f32.Vec3{a[0] * b[0], a[1] * b[1], a[2] * b[2]}
}

func quat(angle float32, axis f32.Vec3) f32.Vec4 {
	c, s := float32(math.Cos(float64(angle/2))), float32(math.Sin(float64(angle/2)))
	return f32.Vec4{c, axis[0] * s, axis[1] * s, axis[2] * s}
}

func quatmul(a, b f32.Vec4) f32.Vec4 {
	return f32.Vec4{
		a[0]*b[0] - a[1]*b[1] - a[2]*b[2] - a[3]*b[3],
		a[2]*b[3] - b[2]*a[3] + a[0]*b[1] + b[0]*a[1],
		b[1]*a[3] - a[1]*b[3] + a[0]*b[2] + b[0]*a[2],
		a[1]*b[2] - b[1]*a[2] + a[0]*b[3] + b[0]*a[3],
	}
}

func quat16fv(q f32.Vec4) f32.Mat4 {
	w, x, y, z := q[0], q[1], q[2], q[3]
	return mul16fv(
		f32.Mat4{
			+w, +z, -y, -x,
			-z, +w, +x, -y,
			+y, -x, +w, -z,
			+x, +y, +z, +w,
		},
		f32.Mat4{
			+w, -z, +y, -x,
			+z, +w, -x, -y,
			-y, +x, +w, -z,
			+x, +y, +z, +w,
		})
}

func mul9fv(a, b f32.Mat3) (m f32.Mat3) {
	m[0] = a[0]*b[0] + a[3]*b[3] + a[6]*b[6]
	m[1] = a[0]*b[1] + a[3]*b[4] + a[6]*b[7]
	m[2] = a[0]*b[2] + a[3]*b[5] + a[6]*b[8]

	m[3] = a[1]*b[0] + a[4]*b[3] + a[7]*b[6]
	m[4] = a[1]*b[1] + a[4]*b[4] + a[7]*b[7]
	m[5] = a[1]*b[2] + a[4]*b[5] + a[7]*b[8]

	m[6] = a[2]*b[0] + a[5]*b[3] + a[8]*b[6]
	m[7] = a[2]*b[1] + a[5]*b[4] + a[8]*b[7]
	m[8] = a[2]*b[2] + a[5]*b[5] + a[8]*b[8]
	return m
}

func mul16fv(a, b f32.Mat4) (m f32.Mat4) {
	m[+0] = a[0]*b[0] + a[4]*b[4] + a[+8]*b[+8] + a[12]*b[12]
	m[+1] = a[0]*b[1] + a[4]*b[5] + a[+8]*b[+9] + a[12]*b[13]
	m[+2] = a[0]*b[2] + a[4]*b[6] + a[+8]*b[10] + a[12]*b[14]
	m[+3] = a[0]*b[3] + a[4]*b[7] + a[+8]*b[11] + a[12]*b[15]

	m[+4] = a[1]*b[0] + a[5]*b[4] + a[+9]*b[+8] + a[13]*b[12]
	m[+5] = a[1]*b[1] + a[5]*b[5] + a[+9]*b[+9] + a[13]*b[13]
	m[+6] = a[1]*b[2] + a[5]*b[6] + a[+9]*b[10] + a[13]*b[14]
	m[+7] = a[1]*b[3] + a[5]*b[7] + a[+9]*b[11] + a[13]*b[15]

	m[+8] = a[2]*b[0] + a[6]*b[4] + a[10]*b[+8] + a[14]*b[12]
	m[+9] = a[2]*b[1] + a[6]*b[5] + a[10]*b[+9] + a[14]*b[13]
	m[10] = a[2]*b[2] + a[6]*b[6] + a[10]*b[10] + a[14]*b[14]
	m[11] = a[2]*b[3] + a[6]*b[7] + a[10]*b[11] + a[14]*b[15]

	m[12] = a[3]*b[0] + a[7]*b[4] + a[11]*b[+8] + a[15]*b[12]
	m[13] = a[3]*b[1] + a[7]*b[5] + a[11]*b[+9] + a[15]*b[13]
	m[14] = a[3]*b[2] + a[7]*b[6] + a[11]*b[10] + a[15]*b[14]
	m[15] = a[3]*b[3] + a[7]*b[7] + a[11]*b[11] + a[15]*b[15]
	return m
}

func det16fv(m f32.Mat4) float32 {
	return m[0]*(m[5]*(m[10]*m[15]-m[11]*m[14])-
		m[6]*(m[9]*m[15]-m[11]*m[13])+
		m[7]*(m[9]*m[14]-m[10]*m[13])) -
		m[1]*(m[4]*(m[10]*m[15]-m[11]*m[14])-
			m[6]*(m[8]*m[15]-m[11]*m[12])+
			m[7]*(m[8]*m[14]-m[10]*m[12])) +
		m[2]*(m[4]*(m[9]*m[15]-m[11]*m[13])-
			m[5]*(m[8]*m[15]-m[11]*m[12])+
			m[7]*(m[8]*m[13]-m[9]*m[12])) -
		m[3]*(m[4]*(m[9]*m[14]-m[10]*m[13])-
			m[5]*(m[8]*m[14]-m[10]*m[12])+
			m[6]*(m[8]*m[13]-m[9]*m[12]))
}

func inv16fv(m f32.Mat4) f32.Mat4 {
	det := det16fv(m)
	if equals(det, 0) {
		return f32.Mat4{}
	}
	r := 1 / det
	return f32.Mat4{
		r * (-m[7]*m[10]*m[13] + m[6]*m[11]*m[13] + m[7]*m[9]*m[14] - m[5]*m[11]*m[14] - m[6]*m[9]*m[15] + m[5]*m[10]*m[15]),
		r * (+m[3]*m[10]*m[13] - m[2]*m[11]*m[13] - m[3]*m[9]*m[14] + m[1]*m[11]*m[14] + m[2]*m[9]*m[15] - m[1]*m[10]*m[15]),
		r * (-m[3]*m[+6]*m[13] + m[2]*m[+7]*m[13] + m[3]*m[5]*m[14] - m[1]*m[+7]*m[14] - m[2]*m[5]*m[15] + m[1]*m[+6]*m[15]),
		r * (+m[3]*m[+6]*m[+9] - m[2]*m[+7]*m[+9] - m[3]*m[5]*m[10] + m[1]*m[+7]*m[10] + m[2]*m[5]*m[11] - m[1]*m[+6]*m[11]),
		r * (+m[7]*m[10]*m[12] - m[6]*m[11]*m[12] - m[7]*m[8]*m[14] + m[4]*m[11]*m[14] + m[6]*m[8]*m[15] - m[4]*m[10]*m[15]),
		r * (-m[3]*m[10]*m[12] + m[2]*m[11]*m[12] + m[3]*m[8]*m[14] - m[0]*m[11]*m[14] - m[2]*m[8]*m[15] + m[0]*m[10]*m[15]),
		r * (+m[3]*m[+6]*m[12] - m[2]*m[+7]*m[12] - m[3]*m[4]*m[14] + m[0]*m[+7]*m[14] + m[2]*m[4]*m[15] - m[0]*m[+6]*m[15]),
		r * (-m[3]*m[+6]*m[+8] + m[2]*m[+7]*m[+8] + m[3]*m[4]*m[10] - m[0]*m[+7]*m[10] - m[2]*m[4]*m[11] + m[0]*m[+6]*m[11]),
		r * (-m[7]*m[+9]*m[12] + m[5]*m[11]*m[12] + m[7]*m[8]*m[13] - m[4]*m[11]*m[13] - m[5]*m[8]*m[15] + m[4]*m[+9]*m[15]),
		r * (+m[3]*m[+9]*m[12] - m[1]*m[11]*m[12] - m[3]*m[8]*m[13] + m[0]*m[11]*m[13] + m[1]*m[8]*m[15] - m[0]*m[+9]*m[15]),
		r * (-m[3]*m[+5]*m[12] + m[1]*m[+7]*m[12] + m[3]*m[4]*m[13] - m[0]*m[+7]*m[13] - m[1]*m[4]*m[15] + m[0]*m[+5]*m[15]),
		r * (+m[3]*m[+5]*m[+8] - m[1]*m[+7]*m[+8] - m[3]*m[4]*m[+9] + m[0]*m[+7]*m[+9] + m[1]*m[4]*m[11] - m[0]*m[+5]*m[11]),
		r * (+m[6]*m[+9]*m[12] - m[5]*m[10]*m[12] - m[6]*m[8]*m[13] + m[4]*m[10]*m[13] + m[5]*m[8]*m[14] - m[4]*m[+9]*m[14]),
		r * (-m[2]*m[+9]*m[12] + m[1]*m[10]*m[12] + m[2]*m[8]*m[13] - m[0]*m[10]*m[13] - m[1]*m[8]*m[14] + m[0]*m[+9]*m[14]),
		r * (+m[2]*m[+5]*m[12] - m[1]*m[+6]*m[12] - m[2]*m[4]*m[13] + m[0]*m[+6]*m[13] + m[1]*m[4]*m[14] - m[0]*m[+5]*m[14]),
		r * (-m[2]*m[+5]*m[+8] + m[1]*m[+6]*m[+8] + m[2]*m[4]*m[+9] - m[0]*m[+6]*m[+9] - m[1]*m[4]*m[10] + m[0]*m[+5]*m[10]),
	}
}

func string16fv(a f32.Mat4) string {
	return fmt.Sprintf("%+.2f %+.2f %+.2f %+.2f\n%+.2f %+.2f %+.2f %+.2f\n%+.2f %+.2f %+.2f %+.2f\n%+.2f %+.2f %+.2f %+.2f",
		a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[8], a[9], a[10], a[11], a[12], a[13], a[14], a[15])
}

const epsilon = 0.0001

func equals(a, b float32) bool {
	return equaleps(a, b, epsilon)
}

func equaleps(a, b float32, eps float32) bool {
	return (a-b) < eps && (b-a) < eps
}