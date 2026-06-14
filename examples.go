package raytracer

import (
	"github.com/timdestan/go-raytracer/internal/gml"
	"github.com/timdestan/go-raytracer/internal/prim"
)

func ExampleCannedScene(width, height int) *Scene {
	// TODO: Extend the operations in GML so this scene
	// can be represented with a GML program.

	objects := []SceneObject{
		// Glass sphere with metallic sheen
		&Sphere{Center: prim.Vec3{X: 0, Y: 0, Z: 5},
			Radius: 1.0,
			SurfaceFn: gml.VSurfaceFn{
				Material: &gml.Material{
					Color:            prim.RGB(0.8, 0.2, 0.2),
					Ks:               0.8,
					Kd:               1.0,
					SpecularExponent: 50.0,
					Transparency:     0.9,
					RefractiveIndex:  1.5},
			},
			ObjectToWorld: prim.IdentityMatrix(),
			WorldToObject: prim.IdentityMatrix(),
			NormalMat:     prim.IdentityMatrix(),
		},
		// Dull, fuzzy surface with some reflection
		&Sphere{Center: prim.Vec3{X: 2, Y: 0, Z: 8},
			Radius: 1.0,
			SurfaceFn: gml.VSurfaceFn{
				Material: &gml.Material{
					Color:        prim.RGB(0.2, 0.2, 0.8),
					Kd:           1.0,
					Reflectivity: 0.2,
					Fuzziness:    0.5},
			},
			ObjectToWorld: prim.IdentityMatrix(),
			WorldToObject: prim.IdentityMatrix(),
			NormalMat:     prim.IdentityMatrix(),
		},
		// Reflective green sphere
		&Sphere{Center: prim.Vec3{X: -2, Y: 0, Z: 6},
			Radius: 1.0,
			SurfaceFn: gml.VSurfaceFn{
				Material: &gml.Material{
					Color:        prim.RGB(0.2, 0.8, 0.2),
					Kd:           1.0,
					Reflectivity: 0.8,
				},
			},
			ObjectToWorld: prim.IdentityMatrix(),
			WorldToObject: prim.IdentityMatrix(),
			NormalMat:     prim.IdentityMatrix(),
		},
		// Ground plane
		&Sphere{Center: prim.Vec3{X: 0, Y: -1001, Z: 5},
			Radius: 1000.0,
			SurfaceFn: gml.VSurfaceFn{
				Material: &gml.Material{
					Color: prim.RGB(0.8, 0.8, 0.8),
					Kd:    1.0,
				},
			},
			ObjectToWorld: prim.IdentityMatrix(),
			WorldToObject: prim.IdentityMatrix(),
			NormalMat:     prim.IdentityMatrix(),
		},
	}

	scene := &Scene{
		WidthPx:  width,
		HeightPx: height,
		Fov:      120.0,
		Lights: []*gml.PointLight{
			{Position: prim.Vec3{X: 5, Y: 5, Z: 0}, Color: prim.RGB(1, 1, 1)},
		},
		AmbientLight: prim.Vec3{X: 0.1, Y: 0.1, Z: 0.1},
		BgColorStart: prim.Vec3{X: 0.0, Y: 0.0, Z: 0.0},
		BgColorEnd:   prim.Vec3{X: 0.5, Y: 0.7, Z: 1.0},
	}

	// Sharing copies should be fine since we're never calling back into the
	// GML environment (in fact the EvalStates are all nil). Sigh...
	scene.PerThreadStates = make([]SceneThreadState, 8)
	for i := range scene.PerThreadStates {
		scene.PerThreadStates[i].Objects = objects
	}

	return scene
}
