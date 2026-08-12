package raytracer

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/timdestan/go-raytracer/internal/gml"
	"github.com/timdestan/go-raytracer/internal/prim"

	_ "embed"
)

var updateFlag = flag.Bool("update", false, "If true, update goldens to current values.")

func readImage(filename string) (image.Image, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(buf))
}

func writeImage(img image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func checkImages(got image.Image, goldenFilePath string) error {
	want, err := readImage(goldenFilePath)
	if err != nil {
		return err
	}
	const minSSIM = 0.99
	ssim, err := prim.SSIM(got, want)
	if err != nil {
		return fmt.Errorf("Error in SSIM computation: %v", err)
	}
	if ssim < minSSIM {
		return fmt.Errorf("SSIM is %f, want >= %f (`go test . --update` to update goldens)", ssim, minSSIM)
	}
	// fmt.Printf("SSIM for %s is %.3f\n", goldenFilePath, ssim)
	return nil
}

func compareImages(t *testing.T, got image.Image, goldenFilePath string) {
	t.Helper()

	if *updateFlag {
		if err := writeImage(got, goldenFilePath); err != nil {
			t.Errorf("Failed to update %s", goldenFilePath)
		} else {
			t.Logf("Wrote new golden to %s", goldenFilePath)
		}
		return
	}

	if err := checkImages(got, goldenFilePath); err != nil {
		t.Fatal(err)
	}
}

func TestRenderCanned(t *testing.T) {
	got, err := ParseAndRenderGML(gml.MustReadTestdataFile("testdata/canned.gml"))
	if err != nil {
		t.Fatalf("ParseAndRenderGML: %v", err)
	}
	compareImages(t, got, "testdata/goldens/example_canned.png")
}

func TestRenderSphere(t *testing.T) {
	got, err := ParseAndRenderGML(gml.MustReadTestdataFile("testdata/sphere.gml"))
	if err != nil {
		t.Fatalf("ParseAndRenderGML: %v", err)
	}
	compareImages(t, got, "testdata/goldens/example_sphere.png")
}

func TestRenderCube(t *testing.T) {
	got, err := ParseAndRenderGML(gml.MustReadTestdataFile("testdata/cube.gml"))
	if err != nil {
		t.Fatalf("ParseAndRenderGML: %v", err)
	}
	compareImages(t, got, "testdata/goldens/example_cube.png")
}

// TestRenderCylinder renders the "front view" of the original contest
// fixture testdata/cylinder.gml, which shows a cylinder's lateral surface
// textured based on face/u/v, plus its two end caps in solid colors. The
// fixture calls render four times (one per rotated view); only the first
// ("cylinder0.ppm", the untransformed front view) is captured here, since
// the other three views are affected by a pre-existing quirk in how
// transforms compose for rotated objects and don't currently show anything.
//
// The fixture uses #include, so it's loaded from disk (via
// ParseAndEvalFile) rather than through the embedded testdata FS, so that
// relative includes resolve correctly.
func TestRenderCylinder(t *testing.T) {
	state := gml.NewEvalState()
	var got image.Image
	state.Render = func(state *gml.EvalState, args *gml.RenderArgs) error {
		if args.File != "cylinder0.ppm" {
			return nil
		}
		scene, err := ConvertRenderArgsToScene(args, state)
		if err != nil {
			return err
		}
		got = Render(scene)
		return nil
	}
	if err := state.ParseAndEvalFile("internal/gml/testdata/cylinder.gml"); err != nil {
		t.Fatalf("ParseAndEvalFile: %v", err)
	}
	if got == nil {
		t.Fatal("cylinder0.ppm was never rendered")
	}
	compareImages(t, got, "testdata/goldens/example_cylinder.png")
}

// Run benchmarks with:
// go test -run ^$ -bench . -cpuprofile=/tmp/cpu.prof
// go tool pprof -http=:8080 /tmp/cpu.prof

func BenchmarkCanned(b *testing.B) {
	for b.Loop() {
		_, err := ParseAndRenderGML(gml.MustReadTestdataFile("testdata/canned.gml"))
		if err != nil {
			b.Fatalf("BenchmarkCanned: %v", err)
		}
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

func BenchmarkCube(b *testing.B) {
	for b.Loop() {
		_, err := ParseAndRenderGML(gml.MustReadTestdataFile("testdata/cube.gml"))
		if err != nil {
			b.Fatalf("BenchmarkCube: %v", err)
		}
	}
}
