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
)

const (
	WIDTH_PX  = 1900
	HEIGHT_PX = 1200
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

	img := rt.Render(rt.ExampleScene1Opts(WIDTH_PX, HEIGHT_PX))
	err := writeImage(img, *filename)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s\n", *filename)
}
