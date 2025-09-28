package prim

import (
	"errors"
	"image"
	"math/rand"
	"sync"
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

	type workitem struct {
		ssim float64
		n    int
	}

	ch := make(chan workitem)

	go func() {
		defer close(ch)
		var wg sync.WaitGroup
		for x := 0; x < len(rgbImg1)-kernelSize; x++ {
			wg.Add(1)
			go func() {
				sum := 0.0
				n := 0
				for y := 0; y < len(rgbImg1[x])-kernelSize; y++ {
					sum += computeSSIMOnWindow(rgbImg1, rgbImg2, x, y, kernel)
					n++
				}
				ch <- workitem{
					ssim: sum,
					n:    n,
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}()

	for item := range ch {
		sum += item.ssim
		n += item.n
	}

	return sum / float64(n), nil
}

func computeSSIMOnWindow(img1, img2 [][]rgb, xstart, ystart int, kernel []float64) float64 {
	var r1Sum, r2Sum, g1Sum, g2Sum, b1Sum, b2Sum float64
	n := float64(kernelSize * kernelSize)

	// TODO: I think we're supposed to add padding, so that we can apply the kernel on the edges of the image.
	for k1 := range kernelSize {
		for k2 := range kernelSize {
			x := xstart + k1
			y := ystart + k2
			w := kernel[k1*kernelSize+k2]

			i1 := img1[x][y]
			i2 := img2[x][y]

			r1Sum += float64(i1.r) * w
			g1Sum += float64(i1.g) * w
			b1Sum += float64(i1.b) * w

			r2Sum += float64(i2.r) * w
			g2Sum += float64(i2.g) * w
			b2Sum += float64(i2.b) * w
		}
	}

	r1Avg := r1Sum / n
	g1Avg := g1Sum / n
	b1Avg := b1Sum / n

	r2Avg := r2Sum / n
	g2Avg := g2Sum / n
	b2Avg := b2Sum / n

	var r1Var, g1Var, b1Var, r2Var, g2Var, b2Var, r12Var, g12Var, b12Var float64

	for k1 := range kernelSize {
		for k2 := range kernelSize {
			x := xstart + k1
			y := ystart + k2
			w := kernel[k1*kernelSize+k2]

			i1 := img1[x][y]
			i2 := img2[x][y]

			r1Var += w * square(float64(i1.r)-r1Avg)
			g1Var += w * square(float64(i1.g)-g1Avg)
			b1Var += w * square(float64(i1.b)-b1Avg)

			r2Var += w * square(float64(i2.r)-r2Avg)
			g2Var += w * square(float64(i2.g)-g2Avg)
			b2Var += w * square(float64(i2.b)-b2Avg)

			r12Var += w * (float64(i1.r) - r1Avg) * (float64(i2.r) - r2Avg)
			g12Var += w * (float64(i1.g) - g1Avg) * (float64(i2.g) - g2Avg)
			b12Var += w * (float64(i1.b) - b1Avg) * (float64(i2.b) - b2Avg)
		}
	}

	r1Var /= (n - 1)
	g1Var /= (n - 1)
	b1Var /= (n - 1)

	r2Var /= (n - 1)
	g2Var /= (n - 1)
	b2Var /= (n - 1)

	r12Var /= (n - 1)
	g12Var /= (n - 1)
	b12Var /= (n - 1)

	computeSSIM := func(avg1, avg2, var1, var2, covar float64) float64 {
		numerator := (2*avg1*avg2 + c1) * (2*covar + c2)
		denominator := (avg1*avg1 + avg2*avg2 + c1) * (var1 + var2 + c2)
		return numerator / denominator
	}

	redSSIM := computeSSIM(r1Avg, r2Avg, r1Var, r2Var, r12Var)
	greenSSIM := computeSSIM(g1Avg, g2Avg, g1Var, g2Var, g12Var)
	blueSSIM := computeSSIM(b1Avg, b2Avg, b1Var, b2Var, b12Var)

	// Average over RGB
	return (redSSIM + greenSSIM + blueSSIM) / 3.0
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
