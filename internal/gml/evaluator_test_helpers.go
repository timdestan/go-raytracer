package gml

import (
	"fmt"
	"strings"

	"github.com/timdestan/go-raytracer/internal/prim"
)

// SplitLines splits a string by lines, supporting LF and CRLF
func SplitLines(line string) []string {
	return strings.Split(strings.ReplaceAll(line, "\r\n", "\n"), "\n")
}

func RenderArgsToLines(args *RenderArgs, idMapping *IDMapping) []string {
	assert := func(cond bool, msg string) {
		if !cond {
			panic(msg)
		}
	}

	debugStringCtx := DebugStringContext{idMapping: idMapping}

	var lines []string
	indent := 0
	add := func(s string) {
		lines = append(lines, strings.Repeat("    ", indent)+s)
	}

	fmtFloat := func(x float64) string {
		// This leaves trailing whitespace on the last float in a line.
		return fmt.Sprintf("%+-10.2f", x)
	}
	fmt3 := func(v *prim.Vec3) string {
		return fmt.Sprintf("%s %s %s", fmtFloat(v.X), fmtFloat(v.Y), fmtFloat(v.Z))
	}
	fmtSlice := func(v []float64) string {
		var sb strings.Builder
		for _, x := range v {
			sb.WriteString(fmtFloat(x))
		}
		return sb.String()
	}

	add(fmt.Sprintf("render %d %d %s", args.Width, args.Height, args.File))
	indent++
	add(fmt.Sprintf("fov: %s", fmtFloat(args.Fov)))
	add(fmt.Sprintf("depth: %d", args.Depth))
	if !args.BgColorStart.IsZero() || !args.BgColorEnd.IsZero() {
		add("background-gradient:")
		indent++
		add("p1: " + fmt3(&args.BgColorStart))
		add("p2: " + fmt3(&args.BgColorEnd))
		indent--
	}
	add("ambient: " + fmt3(args.AmbientLight))
	for _, l := range args.Lights {
		add("light:")
		indent++
		add("position: " + fmt3(&l.Position))
		add("color: " + fmt3(&l.Color))
		indent--
	}

	addSurfaceFn := func(fn VSurfaceFn) {
		add("surface:")
		indent++
		defer func() { indent-- }()

		if fn.Closure != nil {
			add("code: " + fn.Closure.Code.String())
			bindings := fn.Closure.Env.Bindings()
			if len(bindings) == 0 {
				return
			}
			// TODO: Complex variables from the environment
			// (including bindings to other closures) are squashed
			// into a single line here. We could do a lot better here
			// although it might require passing the indentation
			// level in the debug string context.
			add("env:")
			indent++
			for _, b := range bindings {
				add(b.DebugStringCtx(debugStringCtx))
			}
			indent--
		} else {
			assert(fn.Material != nil, "invalid state for VSurfaceFn")
			add("color: " + fmt3(&fn.Material.Color))
			add("reflectivity: " + fmtFloat(fn.Material.Reflectivity))
			add("fuzz: " + fmtFloat(fn.Material.Fuzziness))
			add("transparency: " + fmtFloat(fn.Material.Transparency))
			add("refractiveIndex: " + fmtFloat(fn.Material.RefractiveIndex))
			add("kd: " + fmtFloat(fn.Material.Kd))
			add("ks: " + fmtFloat(fn.Material.Ks))
			add("n: " + fmtFloat(fn.Material.SpecularExponent))
		}
	}

	addXform := func(m4 prim.Mat4) {
		add("xform:")
		indent++
		for _, row := range m4 {
			add(fmtSlice(row[:]))
		}
		indent--
	}

	var addSceneObj func(obj SceneObject)
	addSceneObj = func(obj SceneObject) {
		switch obj := obj.(type) {
		case *Sphere:
			add("sphere:")
			indent++
			addXform(*obj.TransformMat)
			addSurfaceFn(obj.SurfaceFn)
			indent--
		case *Cube:
			add("cube:")
			indent++
			addXform(*obj.TransformMat)
			addSurfaceFn(obj.SurfaceFn)
			indent--
		case *Plane:
			add("plane:")
			indent++
			addXform(*obj.TransformMat)
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
