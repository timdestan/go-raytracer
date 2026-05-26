package raytracer

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/timdestan/go-raytracer/internal/gml"
	"github.com/timdestan/go-raytracer/internal/prim"

	_ "embed"
)

var updateFlag = flag.Bool("update_goldens", false, "If true, update goldens to current values.")

func writeImage(img image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func compareImages(t *testing.T, got, want image.Image, goldenFilePath string) {
	t.Helper()

	if *updateFlag {
		if err := writeImage(got, goldenFilePath); err != nil {
			t.Errorf("Failed to update %s", goldenFilePath)
		} else {
			t.Logf("Wrote new golden to %s", goldenFilePath)
		}
		return
	}

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
	compareImages(t, got, want, "testdata/goldens/example_canned.png")
}

//go:embed testdata/goldens/example_sphere.png
var goldenExampleSphereBytes []byte

// TODO: Embed this
var goldenExampleCubeBytes []byte

func TestRenderSphere(t *testing.T) {
	got, err := ParseAndRenderGML(gml.MustReadTestdataFile("testdata/sphere.gml"))
	if err != nil {
		t.Fatalf("ParseAndRenderGML: %v", err)
	}
	want, err := png.Decode(bytes.NewReader(goldenExampleSphereBytes))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	compareImages(t, got, want, "testdata/goldens/example_sphere.png")
}

func TestRenderCube(t *testing.T) {
	t.Skip("Cubes are WIP")
	got, err := ParseAndRenderGML(gml.MustReadTestdataFile("testdata/cube.gml"))
	if err != nil {
		t.Fatalf("ParseAndRenderGML: %v", err)
	}
	want, err := png.Decode(bytes.NewReader(goldenExampleCubeBytes))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	compareImages(t, got, want, "testdata/goldens/example_cube.png")
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
		_, err := ParseAndRenderGML(gml.MustReadTestdataFile("testdata/sphere.gml"))
		if err != nil {
			b.Fatalf("BenchmarkSphere: %v", err)
		}
	}
}
