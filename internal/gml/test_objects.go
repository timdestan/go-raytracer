package gml

import (
	_ "embed"
)

var (
	//go:embed testdata/cube.gml
	TestdataCube string
	//go:embed testdata/sphere.gml
	TestdataSphere string
)
