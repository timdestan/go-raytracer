package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	rt "raytracer"
)

var (
	filename = flag.String("filename", "", "png filename to write")

	bgStart = rt.Vec3{X: 0.0, Y: 0.0, Z: 0.0}
	bgEnd   = rt.Vec3{X: 0.5, Y: 0.7, Z: 1.0}
)

const (
	WIDTH_PX  = 1900
	HEIGHT_PX = 1200

	// Distance from camera to screen
	CAMERA_DISTANCE = 4.0
)

func writeImage(img image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	flag.Parse()

	if len(*filename) == 0 {
		log.Fatal("--filename is required")
	}

	img := rt.Render(&rt.RenderOptions{
		WidthPx:        WIDTH_PX,
		HeightPx:       HEIGHT_PX,
		CameraPosition: rt.Vec3{X: 0, Y: 0, Z: 0},
		CameraDistance: CAMERA_DISTANCE,
		Spheres: []*rt.Sphere{
			// Glass sphere with metallic sheen
			{Center: rt.Vec3{X: 0, Y: 0, Z: -5}, Radius: 1.0, Material: rt.Material{Color: rt.RGB(0.8, 0.2, 0.2), Reflectivity: 0.9, RefractiveIndex: 1.5}},
			// Dull, fuzzy surface with some reflection
			{Center: rt.Vec3{X: 2, Y: 0, Z: -8}, Radius: 1.0, Material: rt.Material{Color: rt.RGB(0.2, 0.2, 0.8), Reflectivity: 0.2, Fuzziness: 0.5}},
			// Original reflective green sphere
			{Center: rt.Vec3{X: -2, Y: 0, Z: -6}, Radius: 1.0, Material: rt.Material{Color: rt.RGB(0.2, 0.8, 0.2), Reflectivity: 0.8}},
			// Ground plane
			{Center: rt.Vec3{X: 0, Y: -1001, Z: -5}, Radius: 1000.0, Material: rt.Material{Color: rt.RGB(0.8, 0.8, 0.8), Reflectivity: 0.0}},
		},
		Lights: []*rt.Light{
			{Position: rt.Vec3{X: 5, Y: 5, Z: 0}, Color: rt.Vec3{X: 1, Y: 1, Z: 1}},
		},
		BgColorStart: bgStart,
		BgColorEnd:   bgEnd,
	})

	err := writeImage(img, *filename)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s\n", *filename)
}
