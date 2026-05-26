package prim

import (
	"image"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"
)

var seed1, seed2 uint64
var rng *rand.Rand

func getSeed(key string) (uint64, error) {
	v := os.Getenv(key)
	if v == "" {
		return rand.Uint64(), nil
	}
	return strconv.ParseUint(v, 10, 64)
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func init() {
	resetRNG()
}

func resetRNG() {
	seed1 = must(getSeed("SEED1"))
	seed2 = must(getSeed("SEED2"))
	rng = rand.New(rand.NewPCG(seed1, seed2))
}

func TestSSIMSameImage(t *testing.T) {
	resetRNG()

	image := makeRandomImage(100, 100)
	ssim, err := SSIM(image, image)
	if err != nil {
		t.Fatal(err)
	}
	if ssim < 0.999 {
		t.Errorf("SSIM is %f, want ~1.0, SEED1=%d SEED2=%d", ssim, seed1, seed2)
	}
}

func TestSSIMDifferentImages(t *testing.T) {
	resetRNG()

	image1 := makeRandomImage(100, 100)
	image2 := makeRandomImage(100, 100)
	ssim, err := SSIM(image1, image2)
	if err != nil {
		t.Fatal(err)
	}
	if ssim > 0.999 {
		t.Errorf("SSIM is %f, want some number < 1.0 SEED1=%d SEED2=%d", ssim, seed1, seed2)
	}
}

func makeRandomImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := range width {
		for y := range height {
			img.Set(x, y, &Vec3{
				X: rng.Float64(),
				Y: rng.Float64(),
				Z: rng.Float64(),
			})
		}
	}
	return img
}

// Run benchmarks with:
// go test ./internal/prim -run ^$ -bench . -cpuprofile=/tmp/cpu.prof
// go tool pprof -http=:8080 /tmp/cpu.prof

func BenchmarkSSIM(b *testing.B) {
	const width = 1000
	const height = 1000

	img1 := makeRandomImage(width, height)
	img2 := makeRandomImage(width, height)

	for b.Loop() {
		SSIM(img1, img2)
	}
}
