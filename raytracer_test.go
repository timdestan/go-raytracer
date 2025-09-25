package raytracer

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/timdestan/go-raytracer/internal/gml"
	"github.com/timdestan/go-raytracer/internal/prim"

	_ "embed"
)

func compareImages(t *testing.T, got, want image.Image) {
	t.Helper()

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
			gotVec := prim.Vec3{X: float64(gotR), Y: float64(gotG), Z: float64(gotB)}
			wantVec := prim.Vec3{X: float64(wantR), Y: float64(wantG), Z: float64(wantB)}
			similarity := gotVec.CosineSimilarity(&wantVec)
			if similarity < minCosineSimilarity {
				diffs = append(diffs, fmt.Sprintf("pixel (%d, %d): got %v, want %v (similarity = %v)", x, y, gotVec, wantVec, similarity))
			}
		}
	}
	if len(diffs) == 0 {
		return
	}
	totalDiffs := len(diffs)
	if len(diffs) > 10 {
		// Just show a few.
		diffs = diffs[:10]
	}
	t.Errorf("Render() mismatch: %d / %d diffs", totalDiffs, (bounds.Max.X-bounds.Min.X)*(bounds.Max.Y-bounds.Min.Y))
	for _, diff := range diffs {
		t.Errorf("  Diff: %s", diff)
	}
}

//go:embed testdata/goldens/example_canned.png
var goldenExampleCannedBytes []byte

func TestRenderCannedScene(t *testing.T) {
	got := Render(ExampleCannedScene(1920, 1200))

	want, err := png.Decode(bytes.NewReader(goldenExampleCannedBytes))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	compareImages(t, got, want)
}

//go:embed testdata/goldens/example_sphere.png
var goldenExampleSphereBytes []byte

func TestRenderSphere(t *testing.T) {
	got, err := ParseAndRenderGML(gml.TestdataSphere)
	if err != nil {
		t.Fatalf("ParseAndRenderGML: %v", err)
	}
	want, err := png.Decode(bytes.NewReader(goldenExampleSphereBytes))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	compareImages(t, got, want)
}

// Run benchmarks with:
// go test -run ^$ -bench . -cpuprofile=/tmp/cpu.prof
// go tool pprof -http=:8080 /tmp/cpu.prof

func BenchmarkCanned(b *testing.B) {
	for b.Loop() {
		Render(ExampleCannedScene(1920, 1200))
	}
}

func BenchmarkSphere(b *testing.B) {
	for b.Loop() {
		_, err := ParseAndRenderGML(gml.TestdataSphere)
		if err != nil {
			b.Fatalf("BenchmarkSphere: %v", err)
		}
	}
}
