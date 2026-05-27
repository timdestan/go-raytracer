// Package prim implements primitives for 3D graphics.
package prim

import (
	"fmt"
	"math"
)

type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) String() string {
	return fmt.Sprintf("[%v, %v, %v]", v.X, v.Y, v.Z)
}

// RGB is a convenience function to construct a vector
// from normalized RGB values [0.0, 1.0].
func RGB(r, g, b float64) Vec3 {
	return Vec3{X: r, Y: g, Z: b}
}

func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{
		X: v.X + other.X,
		Y: v.Y + other.Y,
		Z: v.Z + other.Z,
	}
}

func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

// Mul multiples two vectors pointwise.
func (v Vec3) Mul(other *Vec3) *Vec3 {
	return &Vec3{
		X: v.X * other.X,
		Y: v.Y * other.Y,
		Z: v.Z * other.Z,
	}
}

func (v Vec3) Dot(other Vec3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v Vec3) CosineSimilarity(other Vec3) float64 {
	return v.Dot(other) / (v.Length() * other.Length())
}

func (v Vec3) Lerp(other Vec3, t float64) Vec3 {
	return Vec3{
		X: v.X + (other.X-v.X)*t,
		Y: v.Y + (other.Y-v.Y)*t,
		Z: v.Z + (other.Z-v.Z)*t,
	}
}

func (v *Vec3) LerpI(other Vec3, t float64) {
	v.X += (other.X - v.X) * t
	v.Y += (other.Y - v.Y) * t
	v.Z += (other.Z - v.Z) * t
}

func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{
		X: v.X * s,
		Y: v.Y * s,
		Z: v.Z * s,
	}
}

func (v Vec3) Normalize() Vec3 {
	magnitude := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	return Vec3{
		X: v.X / magnitude,
		Y: v.Y / magnitude,
		Z: v.Z / magnitude,
	}
}

func (v Vec3) Neg() *Vec3 {
	return &Vec3{
		X: -v.X,
		Y: -v.Y,
		Z: -v.Z,
	}
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func (v Vec3) IsZero() bool {
	return v.X == 0.0 && v.Y == 0.0 && v.Z == 0.0
}

// RGBA implements the image.Color interface
func (v Vec3) RGBA() (r, g, b, a uint32) {
	const max = 0xffff
	return uint32(v.X * max), uint32(v.Y * max), uint32(v.Z * max), max
}

// ClampI clamps the X, Y, and Z values between 0 and 1, in place.
func (c Vec3) Clamp() Vec3 {
	return Vec3{
		X: clamp(0, 1, c.X),
		Y: clamp(0, 1, c.Y),
		Z: clamp(0, 1, c.Z),
	}
}

// Reflect reflects this vector around the given axis vector.
func (c Vec3) Reflect(axis Vec3) Vec3 {
	return axis.Scale(2 * axis.Dot(c)).Sub(c)
}

func (c *Vec3) Rotate(axis *Vec3, angle float64) *Vec3 {
	return c.ToVec4().Rotate(axis, angle).ToVec3()
}

func (c *Vec3) ToVec4() *Vec4 {
	// TODO: Whether W is 0 or 1 depends on what this represents.
	return &Vec4{X: c.X, Y: c.Y, Z: c.Z, W: 1}
}

type Vec4 struct {
	X, Y, Z, W float64
}

func (v *Vec4) String() string {
	return fmt.Sprintf("Vec4(%.4f, %.4f, %.4f, %.4f)", v.X, v.Y, v.Z, v.W)
}

func (v *Vec4) Normalize() *Vec4 {
	magnitude := math.Sqrt(v.W*v.X + v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	return &Vec4{
		W: v.Y / magnitude,
		X: v.X / magnitude,
		Y: v.Y / magnitude,
		Z: v.Z / magnitude,
	}
}

// Rotate rotates this vector around the given axis by the given angle in
// radians.
func (v *Vec4) Rotate(axis *Vec3, angle float64) *Vec4 {
	rot := Rotation(axis, angle)
	return QMul(QMul(rot, v), rot.QInv())
}

func Rotation(axis *Vec3, angle float64) *Vec4 {
	cos := math.Cos(angle / 2.0)
	sin := math.Sin(angle / 2.0)
	return &Vec4{
		X: axis.X * sin,
		Y: axis.Y * sin,
		Z: axis.Z * sin,
		W: cos,
	}
}

func QIdentity() *Vec4 {
	return &Vec4{X: 0, Y: 0, Z: 0, W: 1}
}

func QMul(q1, q2 *Vec4) *Vec4 {
	x1, y1, z1, w1 := q1.X, q1.Y, q1.Z, q1.W
	x2, y2, z2, w2 := q2.X, q2.Y, q2.Z, q2.W

	return &Vec4{
		W: w2*w1 - x2*x1 - y2*y1 - z2*z1,
		X: w2*x1 + x2*w1 + y2*z1 - z2*y1,
		Y: w2*y1 - x2*z1 + y2*w1 + z2*x1,
		Z: w2*z1 + x2*y1 - y2*x1 + z2*w1,
	}
}

func QuatToMat(q *Vec4) *Mat4 {
	q = q.Normalize()
	x, y, z, w := q.X, q.Y, q.Z, q.W

	xx, yy, zz := x*x, y*y, z*z
	xy, xz, yz := x*y, x*z, y*z
	wx, wy, wz := w*x, w*y, w*z

	return &Mat4{
		[4]float64{1 - 2*(yy+zz), 2 * (xy + wz), 2 * (xz - wy), 0},
		[4]float64{2 * (xy - wz), 1 - 2*(xx+zz), 2 * (yz + wx), 0},
		[4]float64{2 * (xz + wy), 2 * (yz - wx), 1 - 2*(xx+yy), 0},
		[4]float64{0, 0, 0, 1},
	}
}

func (q *Vec4) QInv() *Vec4 {
	return &Vec4{
		W: q.W,
		X: -q.X,
		Y: -q.Y,
		Z: -q.Z,
	}
}

func (v *Vec4) ToVec3() *Vec3 {
	return &Vec3{
		X: v.X,
		Y: v.Y,
		Z: v.Z,
	}
}

// clamp limits x between min and max
func clamp(min, max, x float64) float64 {
	return math.Min(math.Max(x, min), max)
}

type XForm struct {
	Translation Vec3
	Rotation    Vec4
	Scale       Vec3
}

func IdentityXForm() *XForm {
	return &XForm{
		Translation: Vec3{},
		Rotation:    *QIdentity(),
		Scale:       Vec3{X: 1, Y: 1, Z: 1},
	}
}

func (t *XForm) Invert() *XForm {
	return &XForm{
		Translation: t.Translation.Scale(-1),
		Rotation:    *t.Rotation.QInv(),
		Scale: Vec3{
			X: 1.0 / t.Scale.X,
			Y: 1.0 / t.Scale.Y,
			Z: 1.0 / t.Scale.Z,
		},
	}
}

func (t *XForm) ToMat4() *Mat4 {
	// TODO: A more efficient implementation is possible.
	S := Mat4Scale(t.Scale.X, t.Scale.Y, t.Scale.Z)
	R := QuatToMat(&t.Rotation)
	T := Mat4Translate(t.Translation)
	return T.MulMat(R.MulMat(S))
}

type Mat4 [4][4]float64

func (m *Mat4) MulMat(n *Mat4) *Mat4 {
	var product Mat4
	for i := range 4 {
		for j := range 4 {
			for k := range 4 {
				product[i][j] += m[i][k] * n[k][j]
			}
		}
	}
	return &product
}

func (m *Mat4) MulVec(v *Vec4) *Vec4 {
	return &Vec4{
		m[0][0]*v.X + m[0][1]*v.Y + m[0][2]*v.Z + m[0][3]*v.W,
		m[1][0]*v.X + m[1][1]*v.Y + m[1][2]*v.Z + m[1][3]*v.W,
		m[2][0]*v.X + m[2][1]*v.Y + m[2][2]*v.Z + m[2][3]*v.W,
		m[3][0]*v.X + m[3][1]*v.Y + m[3][2]*v.Z + m[3][3]*v.W,
	}
}

func (m *Mat4) MulScalar(s float64) *Mat4 {
	return &Mat4{
		{m[0][0] * s, m[0][1] * s, m[0][2] * s, m[0][3] * s},
		{m[1][0] * s, m[1][1] * s, m[1][2] * s, m[1][3] * s},
		{m[2][0] * s, m[2][1] * s, m[2][2] * s, m[2][3] * s},
		{m[3][0] * s, m[3][1] * s, m[3][2] * s, m[3][3] * s},
	}
}

func (m *Mat4) Transpose() *Mat4 {
	return &Mat4{
		{m[0][0], m[1][0], m[2][0], m[3][0]},
		{m[0][1], m[1][1], m[2][1], m[3][1]},
		{m[0][2], m[1][2], m[2][2], m[3][2]},
		{m[0][3], m[1][3], m[2][3], m[3][3]},
	}
}

// Multiply matrix by point (w=1, includes translation)
func (m *Mat4) MulPoint(v Vec3) Vec3 {
	return Vec3{
		X: m[0][0]*v.X + m[0][1]*v.Y + m[0][2]*v.Z + m[0][3],
		Y: m[1][0]*v.X + m[1][1]*v.Y + m[1][2]*v.Z + m[1][3],
		Z: m[2][0]*v.X + m[2][1]*v.Y + m[2][2]*v.Z + m[2][3],
	}
}

// Multiply matrix by direction (w=0, ignores translation)
func (m *Mat4) MulDir(v Vec3) Vec3 {
	return Vec3{
		X: m[0][0]*v.X + m[0][1]*v.Y + m[0][2]*v.Z,
		Y: m[1][0]*v.X + m[1][1]*v.Y + m[1][2]*v.Z,
		Z: m[2][0]*v.X + m[2][1]*v.Y + m[2][2]*v.Z,
	}
}

// Inverse computes the inverse of the matrix.
//
// This currently assumes that the matrix is an affine transformation,
// and will not work for general purpose matrices.
func (m *Mat4) Inverse() *Mat4 {
	// An affine transformation matrix should have this form:

	// | L T |
	// | 0 1 |

	// where L is a 3x3 matrix and T is a 3-vector.

	// The inverse is then:

	// | L⁻¹   -L⁻¹ T |
	// |  0       1   |

	a := m[0][0]
	b := m[0][1]
	c := m[0][2]
	d := m[1][0]
	e := m[1][1]
	f := m[1][2]
	g := m[2][0]
	h := m[2][1]
	i := m[2][2]

	det := a*(e*i-f*h) - b*(d*i-f*g) + c*(d*h-e*g)
	if det == 0.0 {
		// Not sure if this is possible for the matrices we expect
		// to generate.
		return nil
	}

	// Compute the transpose of the cofactor matrix
	inv := &Mat4{
		{(e*i - f*h) / det, (c*h - b*i) / det, (b*f - c*e) / det, 0.0},
		{(f*g - d*i) / det, (a*i - c*g) / det, (c*d - a*f) / det, 0.0},
		{(d*h - e*g) / det, (b*g - a*h) / det, (a*e - b*d) / det, 0.0},
		{0.0, 0.0, 0.0, 1.0},
	}

	// We've filled in the upper left 3x3 part L⁻¹.
	// We need to fill in the right column: -L⁻¹ T.

	inv[0][3] = -(inv[0][0]*m[0][3] + inv[0][1]*m[1][3] + inv[0][2]*m[2][3])
	inv[1][3] = -(inv[1][0]*m[0][3] + inv[1][1]*m[1][3] + inv[1][2]*m[2][3])
	inv[2][3] = -(inv[2][0]*m[0][3] + inv[2][1]*m[1][3] + inv[2][2]*m[2][3])

	return inv
}

func IdentityMatrix() Mat4 {
	return Mat4{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

func Mat4Translate(v Vec3) *Mat4 {
	return &Mat4{
		{1, 0, 0, v.X},
		{0, 1, 0, v.Y},
		{0, 0, 1, v.Z},
		{0, 0, 0, 1},
	}
}

func Mat4Scale(x, y, z float64) *Mat4 {
	return &Mat4{
		{x, 0, 0, 0},
		{0, y, 0, 0},
		{0, 0, z, 0},
		{0, 0, 0, 1},
	}
}

func Mat4RotateX(angle float64) *Mat4 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return &Mat4{
		{1, 0, 0, 0},
		{0, cos, -sin, 0},
		{0, sin, cos, 0},
		{0, 0, 0, 1},
	}
}

func Mat4RotateY(angle float64) *Mat4 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return &Mat4{
		{cos, 0, sin, 0},
		{0, 1, 0, 0},
		{-sin, 0, cos, 0},
		{0, 0, 0, 1},
	}
}

func Mat4RotateZ(angle float64) *Mat4 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return &Mat4{
		{cos, -sin, 0, 0},
		{sin, cos, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}
