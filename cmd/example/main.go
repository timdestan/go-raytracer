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

func writeImage(img image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
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

	if len(*gmlFile) == 0 {
		log.Fatal("--gml_file is required")
	}
	if len(*outFile) == 0 {
		log.Fatal("--out_file is required")
	}

	img, err := renderFromGMLFile(*gmlFile)
	if err != nil {
		log.Fatal(err)
	}

	if err = writeImage(img, *outFile); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s\n", *outFile)
}
