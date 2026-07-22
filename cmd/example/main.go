package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	return rt.ParseAndRenderGMLFile(filename)
}

func main() {
	flag.Parse()

	if len(*gmlFile) == 0 {
		log.Fatal("--gml_file is required")
	}
	if len(*outFile) == 0 {
		base := filepath.Base(*gmlFile)
		base, found := strings.CutSuffix(base, ".gml")
		if !found {
			log.Fatal("Could not derive --out_file, please specify it.")
		}
		*outFile = fmt.Sprintf("output/%s.png", base)
		log.Printf("Using derived output path: %s", *outFile)
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
