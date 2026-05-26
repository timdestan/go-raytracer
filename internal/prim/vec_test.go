package prim

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var approxOpts = cmpopts.EquateApprox(1e-7, 0.0)

var vec4Approx = cmp.Comparer(func(x, y Vec4) bool {
	return math.Abs(x.X-y.X) < 1e-7 &&
		math.Abs(x.Y-y.Y) < 1e-7 &&
		math.Abs(x.Z-y.Z) < 1e-7 &&
		math.Abs(x.W-y.W) < 1e-7
})

func TestNormalizeSimple(t *testing.T) {
	tests := []struct {
		v    Vec3
		want Vec3
	}{
		{v: Vec3{X: 2, Y: 0, Z: 0}, want: Vec3{X: 1, Y: 0, Z: 0}},
		{v: Vec3{X: 0, Y: -12, Z: 5}, want: Vec3{X: 0, Y: -12.0 / 13, Z: 5.0 / 13}},
		{v: Vec3{X: 3, Y: 4, Z: 0}, want: Vec3{X: 3.0 / 5.0, Y: 4.0 / 5.0, Z: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.v.String(), func(t *testing.T) {
			got := tt.v.Normalize()
			if diff := cmp.Diff(got, &tt.want, approxOpts); diff != "" {
				t.Errorf("Vec3.Normalize() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestNormalizeIsUnitLength(t *testing.T) {
	tests := []struct {
		v Vec3
	}{
		{v: Vec3{X: 2, Y: 0, Z: 0}},
		{v: Vec3{X: 12, Y: 14, Z: 23}},
		{v: Vec3{X: 0, Y: 83, Z: 0.32}},
	}
	for _, tt := range tests {
		t.Run(tt.v.String(), func(t *testing.T) {
			normed := tt.v.Normalize()
			want := 1.0
			got := normed.Length()
			if diff := cmp.Diff(got, want, approxOpts); diff != "" {
				t.Errorf("Vec3.Length() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestVec4Rotation(t *testing.T) {
	type testcase struct {
		name  string
		vec   Vec4
		want  Vec4
		axis  Vec3
		angle float64
	}

	tests := []testcase{
		{
			name:  "rotate around perpendicular axis",
			vec:   Vec4{X: 1, Y: 0, Z: 0, W: 0},
			want:  Vec4{X: 1, Y: 0, Z: 0, W: 0},
			axis:  Vec3{X: 1, Y: 0, Z: 0},
			angle: math.Pi,
		},
		{
			name:  "rotate around parallel axis",
			vec:   Vec4{X: 0, Y: 1, Z: 0, W: 0},
			want:  Vec4{X: 0, Y: 1, Z: 0, W: 0},
			axis:  Vec3{X: 0, Y: 1, Z: 0},
			angle: math.Pi,
		},
		{
			name:  "rotate 90 degrees around axis",
			vec:   Vec4{X: 1, Y: 1, Z: 0, W: 0},
			want:  Vec4{X: 1, Y: -1, Z: 0, W: 0},
			axis:  Vec3{X: 0, Y: 0, Z: 1},
			angle: math.Pi / 2.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.vec.Rotate(&tt.axis, tt.angle)
			if diff := cmp.Diff(got, &tt.want, vec4Approx); diff != "" {
				t.Errorf("Vec4 mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRotationIdentities(t *testing.T) {
	xAxis := Vec3{X: 1, Y: 0, Z: 0}
	yAxis := Vec3{X: 0, Y: 1, Z: 0}
	zAxis := Vec3{X: 0, Y: 0, Z: 1}

	rotateN := func(n int, axis *Vec3, angle float64) *Vec4 {
		rot1 := Rotation(axis, angle)
		rot := rot1
		for i := 1; i < n; i++ {
			rot = QMul(rot1, rot)
		}
		return rot
	}

	testVec := Vec3{X: 1, Y: 2, Z: 3}

	type testcase struct {
		name     string
		init     *Vec3
		rotation *Vec4
		want     *Vec3
	}
	testCases := []testcase{
		{
			name:     "pi * 2 around x axis",
			init:     &testVec,
			rotation: rotateN(2, &xAxis, math.Pi),
			want:     &testVec,
		},
		{
			name:     "pi * 2 around y axis",
			init:     &testVec,
			rotation: rotateN(2, &yAxis, math.Pi),
			want:     &testVec,
		},
		{
			name:     "pi * 2 around z axis",
			init:     &testVec,
			rotation: rotateN(2, &zAxis, math.Pi),
			want:     &testVec,
		},
		{
			name:     "pi/2 * 4 * 4 around z axis",
			init:     &testVec,
			rotation: rotateN(4*4, &zAxis, math.Pi/2),
			want:     &testVec,
		},
		{
			name: "2 * pi/2 around y axis",
			init: &Vec3{
				X: 1,
				Y: 2,
				Z: 3,
			},
			rotation: rotateN(2, &yAxis, math.Pi/2),
			want: &Vec3{
				X: -1,
				Y: 2,
				Z: -3,
			},
		},
		{
			name: "pi/2 around z axis",
			init: &Vec3{
				X: 1,
				Y: 2,
				Z: 3,
			},
			rotation: rotateN(1, &zAxis, math.Pi/2),
			want: &Vec3{
				X: 2,
				Y: -1,
				Z: 3,
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := QMul(QMul(tt.rotation, tt.init.ToVec4()), tt.rotation.QInv())
			if diff := cmp.Diff(got.ToVec3(), tt.want, approxOpts); diff != "" {
				t.Errorf("Rotation(%s) failed: (-want +got):\n%s", tt.name, diff)
			}
		})
	}
}

func TestMulMat(t *testing.T) {
	type testcase struct {
		name string
		a    Mat4
		b    Mat4
		want Mat4
	}
	tests := []testcase{
		{
			name: "Simple test",
			a: Mat4{
				{1, 2, 3, 4},
				{5, 6, 7, 8},
				{9, 10, 11, 12},
				{13, 14, 15, 16},
			},
			b: Mat4{
				{1, 2, 3, 4},
				{5, 6, 7, 8},
				{9, 10, 11, 12},
				{13, 14, 15, 16},
			},
			want: Mat4{
				{90, 100, 110, 120},
				{202, 228, 254, 280},
				{314, 356, 398, 440},
				{426, 484, 542, 600},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.MulMat(&tt.b)
			if diff := cmp.Diff(tt.want, *got, approxOpts); diff != "" {
				t.Errorf("MulMat() mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
