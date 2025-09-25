// Package prim implements primitives for 3D graphics.
package prim

import (
	"fmt"
	"math"
)

type Vec3 struct {
	X, Y, Z float64
}

func (v *Vec3) String() string {
	return fmt.Sprintf("Vec3(%.4f, %.4f, %.4f)", v.X, v.Y, v.Z)
}

// RGB is a convenience function to construct a vector
// from normalized RGB values [0.0, 1.0].
func RGB(r, g, b float64) Vec3 {
	return Vec3{X: r, Y: g, Z: b}
}

func (v *Vec3) Add(other *Vec3) *Vec3 {
	return &Vec3{
		X: v.X + other.X,
		Y: v.Y + other.Y,
		Z: v.Z + other.Z,
	}
}

// AddI is an in-place version of Add
func (v *Vec3) AddI(other *Vec3) *Vec3 {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
	return v
}

func (v *Vec3) Sub(other *Vec3) *Vec3 {
	return &Vec3{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

// Mul multiples two vectors pointwise.
func (v *Vec3) Mul(other *Vec3) *Vec3 {
	return &Vec3{
		X: v.X * other.X,
		Y: v.Y * other.Y,
		Z: v.Z * other.Z,
	}
}

func (v *Vec3) Dot(other *Vec3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v *Vec3) CosineSimilarity(other *Vec3) float64 {
	return v.Dot(other) / (v.Length() * other.Length())
}

func (v *Vec3) LerpI(other *Vec3, t float64) *Vec3 {
	v.X += (other.X - v.X) * t
	v.Y += (other.Y - v.Y) * t
	v.Z += (other.Z - v.Z) * t
	return v
}

func (v *Vec3) Scale(s float64) *Vec3 {
	return &Vec3{
		X: v.X * s,
		Y: v.Y * s,
		Z: v.Z * s,
	}
}

func (v *Vec3) Normalize() *Vec3 {
	magnitude := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	return &Vec3{
		X: v.X / magnitude,
		Y: v.Y / magnitude,
		Z: v.Z / magnitude,
	}
}

func (v *Vec3) Neg() *Vec3 {
	return &Vec3{
		X: -v.X,
		Y: -v.Y,
		Z: -v.Z,
	}
}

func (v *Vec3) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func (v *Vec3) IsZero() bool {
	return v.X == 0.0 && v.Y == 0.0 && v.Z == 0.0
}

// RGBA implements the image.Color interface
func (v *Vec3) RGBA() (r, g, b, a uint32) {
	const max = 0xffff
	return uint32(v.X * max), uint32(v.Y * max), uint32(v.Z * max), max
}

// ClampI clamps the X, Y, and Z values between 0 and 1, in place.
func (c *Vec3) ClampI() *Vec3 {
	c.X = clamp(0, 1, c.X)
	c.Y = clamp(0, 1, c.Y)
	c.Z = clamp(0, 1, c.Z)
	return c
}

// Reflect reflects this vector around the given axis vector.
func (c *Vec3) Reflect(axis *Vec3) *Vec3 {
	return axis.Scale(2 * axis.Dot(c)).Sub(c)
}

// clamp limits x between min and max
func clamp(min, max, x float64) float64 {
	return math.Min(math.Max(x, min), max)
}
