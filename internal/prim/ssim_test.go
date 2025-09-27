package prim

import (
	"image"
	"math/rand"
	"testing"
)

func TestSSIMSameImage(t *testing.T) {
	image := makeRandomImage(100, 100)
	ssim, err := SSIM(image, image)
	if err != nil {
		t.Fatal(err)
	}
	if ssim < 0.999 {
		t.Errorf("SSIM is %f, want ~1.0", ssim)
	}
}

func TestSSIMDifferentImages(t *testing.T) {
	image1 := makeRandomImage(100, 100)
	image2 := makeRandomImage(100, 100)
	ssim, err := SSIM(image1, image2)
	if err != nil {
		t.Fatal(err)
	}
	if ssim > 0.999 {
		t.Errorf("SSIM is %f, want some number < 1.0", ssim)
	}
}

func makeRandomImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, &Vec3{
				X: rand.Float64(),
				Y: rand.Float64(),
				Z: rand.Float64(),
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
