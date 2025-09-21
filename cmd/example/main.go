package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	rt "github.com/timdestan/go-raytracer"
)

var (
	gmlFile = flag.String("gml_file", "", "gml filename to run")

	outFile = flag.String("out_file", "", "png filename to write")
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

func renderCannedScene() image.Image {
	return rt.Render(rt.ExampleScene1(WIDTH_PX, HEIGHT_PX))
}

func renderFromGMLFile(filename string) (image.Image, error) {
	prog, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return rt.ParseAndRenderGML(string(prog))
}

func main() {
	flag.Parse()
	if len(*outFile) == 0 {
		log.Fatal("--out_file is required")
	}

	var img image.Image
	var err error
	if len(*gmlFile) == 0 {
		log.Print("--gml_file not specified, using canned scene.")
		img = renderCannedScene()
	} else {
		img, err = renderFromGMLFile(*gmlFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err = writeImage(img, *outFile); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s\n", *outFile)
}
