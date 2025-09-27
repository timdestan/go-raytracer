package raytracer

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/timdestan/go-raytracer/internal/gml"
	"github.com/timdestan/go-raytracer/internal/prim"

	_ "embed"
)

func compareImages(t *testing.T, got, want image.Image) {
	t.Helper()

	const minSSIM = 0.95
	ssim, err := prim.SSIM(got, want)
	if err != nil {
		t.Fatalf("Error in SSIM computation: %v", err)
	}
	if ssim < minSSIM {
		t.Errorf("SSIM is %f, want >= %f", ssim, minSSIM)
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
