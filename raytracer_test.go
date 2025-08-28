package raytracer

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "embed"
)

var approxOpts = cmpopts.EquateApprox(1e-7, 0.0)

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

//go:embed testdata/goldens/example1.png
var goldenExample1Bytes []byte

func TestRenderGolden(t *testing.T) {
	img := Render(&RenderOptions{
		WidthPx:        1000,
		HeightPx:       1000,
		CameraPosition: Vec3{X: 0, Y: 0, Z: 0},
		CameraDistance: 0.5,
		Spheres: []*Sphere{
			{Center: Vec3{X: -0.3, Y: 0.2, Z: 2.0}, Radius: 0.2, Color: RGB(1, 0, 0)},
			{Center: Vec3{X: 0.5, Y: -0.2, Z: 3.0}, Radius: 1.0, Color: RGB(0, 0, 1)},
		},
		Lights: []*Light{
			{Position: Vec3{X: 0.5, Y: 0.5, Z: 0}, Color: Vec3{X: 1, Y: 1, Z: 1}},
		},
		BgColorStart: Vec3{X: 0.0, Y: 0.0, Z: 0.0},
		BgColorEnd:   Vec3{X: 0.5, Y: 0.7, Z: 1.0},
	})

	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	got := b.Bytes()
	want := goldenExample1Bytes
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Render() mismatch (-got +want):\n%s", diff)
	}
}
