package gml

import (
	"embed"
)

var (
	//go:embed testdata/*
	testdataFS embed.FS
)

func MustReadTestdataFile(name string) string {
	b, err := testdataFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}
