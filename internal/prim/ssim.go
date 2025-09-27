package prim

import (
	"errors"
	"image"
	"math/rand"
)

const (
	kernelSize = 11

	k1 = 0.01
	k2 = 0.03

	c1 = (k1 * k1)
	c2 = (k2 * k2)
)

// SSIM computes a structured similarity index (SSIM) between two images.
//
// See https://www.cns.nyu.edu/pub/eero/wang03-reprint.pdf
//
// This has not been carefully validated and I'll bet it has bugs in it.
func SSIM(img1, img2 image.Image) (float64, error) {
	if img1.Bounds() != img2.Bounds() {
		return 0.0, errors.New("images are not the same size")
	}
	if img1.Bounds().Dx() < kernelSize || img1.Bounds().Dy() < kernelSize {
		return 0.0, errors.New("images are too small")
	}
	rgbImg1 := convertImageToRGB(img1)
	rgbImg2 := convertImageToRGB(img2)

	kernel := makeGaussianKernel()

	n := 0
	sum := 0.0
	for x := 0; x < len(rgbImg1)-kernelSize; x++ {
		for y := 0; y < len(rgbImg1[x])-kernelSize; y++ {
			sum += computeSSIMOnWindow(rgbImg1, rgbImg2, x, y, kernel)
			n++
		}
	}
	return sum / float64(n), nil
}

func computeSSIMOnWindow(img1, img2 [][]rgb, xstart, ystart int, kernel []float64) float64 {
	var r1_sum, r2_sum, g1_sum, g2_sum, b1_sum, b2_sum float64
	n := float64(kernelSize * kernelSize)

	// TODO: I think we're supposed to add padding, so that we can apply the kernel on the edges of the image.
	for k1 := range kernelSize {
		for k2 := range kernelSize {
			x := xstart + k1
			y := ystart + k2
			w := kernel[k1*kernelSize+k2]

			i1 := img1[x][y]
			i2 := img2[x][y]

			r1_sum += float64(i1.r) * w
			g1_sum += float64(i1.g) * w
			b1_sum += float64(i1.b) * w

			r2_sum += float64(i2.r) * w
			g2_sum += float64(i2.g) * w
			b2_sum += float64(i2.b) * w
		}
	}

	r1_avg := r1_sum / n
	g1_avg := g1_sum / n
	b1_avg := b1_sum / n

	r2_avg := r2_sum / n
	g2_avg := g2_sum / n
	b2_avg := b2_sum / n

	var r1_var, g1_var, b1_var, r2_var, g2_var, b2_var, r12_var, g12_var, b12_var float64

	for k1 := range kernelSize {
		for k2 := range kernelSize {
			x := xstart + k1
			y := ystart + k2
			w := kernel[k1*kernelSize+k2]

			i1 := img1[x][y]
			i2 := img2[x][y]

			r1_var += w * square(float64(i1.r)-r1_avg)
			g1_var += w * square(float64(i1.g)-g1_avg)
			b1_var += w * square(float64(i1.b)-b1_avg)

			r2_var += w * square(float64(i2.r)-r2_avg)
			g2_var += w * square(float64(i2.g)-g2_avg)
			b2_var += w * square(float64(i2.b)-b2_avg)

			r12_var += w * (float64(i1.r) - r1_avg) * (float64(i2.r) - r2_avg)
			g12_var += w * (float64(i1.g) - g1_avg) * (float64(i2.g) - g2_avg)
			b12_var += w * (float64(i1.b) - b1_avg) * (float64(i2.b) - b2_avg)
		}
	}

	r1_var /= (n - 1)
	g1_var /= (n - 1)
	b1_var /= (n - 1)

	r2_var /= (n - 1)
	g2_var /= (n - 1)
	b2_var /= (n - 1)

	r12_var /= (n - 1)
	g12_var /= (n - 1)
	b12_var /= (n - 1)

	compute_ssim := func(avg1, avg2, var1, var2, covar float64) float64 {
		numerator := (2*avg1*avg2 + c1) * (2*covar + c2)
		denominator := (avg1*avg1 + avg2*avg2 + c1) * (var1 + var2 + c2)
		return numerator / denominator
	}

	red_ssim := compute_ssim(r1_avg, r2_avg, r1_var, r2_var, r12_var)
	green_ssim := compute_ssim(g1_avg, g2_avg, g1_var, g2_var, g12_var)
	blue_ssim := compute_ssim(b1_avg, b2_avg, b1_var, b2_var, b12_var)

	// Average over RGB
	return (red_ssim + green_ssim + blue_ssim) / 3.0
}

func makeGaussianKernel() []float64 {
	window := make([]float64, kernelSize*kernelSize)
	const stddev = 1.5
	total := 0.0
	for i := range window {
		window[i] = rand.NormFloat64() * stddev
		total += window[i]
	}
	// Normalize so it sums to 1
	for i := range window {
		window[i] /= total
	}
	return window
}

func square(x float64) float64 { return x * x }

type rgb struct {
	r, g, b uint32
}

func convertImageToRGB(img image.Image) [][]rgb {
	rgbs := make([][]rgb, img.Bounds().Dx())
	for x := 0; x < img.Bounds().Dx(); x++ {
		rgbs[x] = make([]rgb, img.Bounds().Dy())
		for y := 0; y < img.Bounds().Dy(); y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			rgbs[x][y] = rgb{r, g, b}
		}
	}
	return rgbs
}
