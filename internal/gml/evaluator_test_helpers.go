package gml

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/timdestan/go-raytracer/internal/prim"
)

// SplitLines splits a string by lines, supporting LF and CRLF
func SplitLines(line string) []string {
	return strings.Split(strings.ReplaceAll(line, "\r\n", "\n"), "\n")
}

func RenderArgsToLines(args *RenderArgs) []string {
	var lines []string
	indent := 0
	add := func(s string) {
		lines = append(lines, strings.Repeat("  ", indent)+s)
	}

	fmtFloat := func(f float64) string {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	fmt3 := func(v *prim.Vec3) string {
		return fmt.Sprintf("%s %s %s", fmtFloat(v.X), fmtFloat(v.Y), fmtFloat(v.Z))
	}

	add(fmt.Sprintf("render %d %d %s", args.Width, args.Height, args.File))
	indent++
	add(fmt.Sprintf("fov: %s", fmtFloat(args.Fov)))
	add(fmt.Sprintf("depth: %d", args.Depth))
	add("ambient: " + fmt3(args.AmbientLight))
	for _, l := range args.Lights {
		add("light:")
		indent++
		add("position: " + fmt3(&l.Position))
		add("color: " + fmt3(&l.Color))
		indent--
	}
	addSurfaceFn := func(fn VClosure) {
		add("surface:")
		indent++
		defer func() { indent-- }()
		add("code: " + fn.Code.String())
		if len(fn.Env) == 0 {
			return
		}
		add("env:")
		var keys []string
		for k := range fn.Env {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		indent++
		for _, k := range keys {
			add(k + ": " + fn.Env[k].String())
		}
		indent--
	}

	var addSceneObj func(obj SceneObject)
	addSceneObj = func(obj SceneObject) {
		// TODO: Render transformation matrix?
		switch obj := obj.(type) {
		case *Sphere:
			add("sphere:")
			indent++
			add("center: " + fmt3(&obj.Center))
			add("radius: " + fmtFloat(obj.Radius))
			addSurfaceFn(obj.SurfaceFn)
			indent--
		case *Cube:
			add("cube:")
			indent++
			add("minpoint: " + fmt3(&obj.Cube.MinPoint))
			add("maxpoint: " + fmt3(&obj.Cube.MaxPoint))
			addSurfaceFn(obj.SurfaceFn)
			indent--
		case *Plane:
			add("plane:")
			indent++
			add("point: " + fmt3(&obj.Plane.Point))
			add("normal: " + fmt3(&obj.Plane.Normal))
			addSurfaceFn(obj.SurfaceFn)
			indent--
		case *Union:
			add("union:")
			indent++
			for _, o := range obj.Objects {
				addSceneObj(o)
			}
			indent--
		default:
			panic("unknown scene object type")
		}
	}
	addSceneObj(args.Scene)
	indent--

	return lines
}
