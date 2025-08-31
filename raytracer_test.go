package raytracer

import (
	"bytes"
	"fmt"
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
	got := Render(ExampleScene1Opts(1900, 1200))

	want, err := png.Decode(bytes.NewReader(goldenExample1Bytes))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	if diff := cmp.Diff(got.Bounds(), want.Bounds()); diff != "" {
		t.Errorf("Render() bounds mismatch (-got +want):\n%s", diff)
	}
	bounds := want.Bounds()

	// TODO: This sucks. I'm sure there's a better way to do this.
	const minCosineSimilarity = 0.75
	var diffs []string
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			gotR, gotG, gotB, _ := got.At(x, y).RGBA()
			wantR, wantG, wantB, _ := want.At(x, y).RGBA()
			gotVec := Vec3{X: float64(gotR), Y: float64(gotG), Z: float64(gotB)}
			wantVec := Vec3{X: float64(wantR), Y: float64(wantG), Z: float64(wantB)}
			similarity := gotVec.CosineSimilarity(&wantVec)
			if similarity < minCosineSimilarity {
				diffs = append(diffs, fmt.Sprintf("pixel (%d, %d): got %v, want %v (similarity = %v)", x, y, gotVec, wantVec, similarity))
			}
		}
	}
	totalDiffs := len(diffs)
	if len(diffs) > 10 {
		// Just show a few.
		diffs = diffs[:10]
	}
	if len(diffs) == 0 {
		return
	}
	t.Errorf("Render() mismatch: %d / %d diffs", totalDiffs, (bounds.Max.X-bounds.Min.X)*(bounds.Max.Y-bounds.Min.Y))
	for _, diff := range diffs {
		t.Errorf("  Diff: %s", diff)
	}
}
