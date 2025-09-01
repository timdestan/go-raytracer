package raytracer

func ExampleScene1(width, height int) *Scene {
	return &Scene{
		WidthPx:        width,
		HeightPx:       height,
		CameraDistance: 4.0,
		Spheres: []*Sphere{
			// Glass sphere with metallic sheen
			{Center: Vec3{X: 0, Y: 0, Z: -5},
				Radius:   1.0,
				Material: Material{Color: RGB(0.8, 0.2, 0.2), Reflectivity: 0.9, Transparency: 0.9, RefractiveIndex: 1.5}},
			// Dull, fuzzy surface with some reflection
			{Center: Vec3{X: 2, Y: 0, Z: -8},
				Radius:   1.0,
				Material: Material{Color: RGB(0.2, 0.2, 0.8), Reflectivity: 0.2, Fuzziness: 0.5}},
			// Original reflective green sphere
			{Center: Vec3{X: -2, Y: 0, Z: -6},
				Radius:   1.0,
				Material: Material{Color: RGB(0.2, 0.8, 0.2), Reflectivity: 0.8}},
			// Ground plane
			{Center: Vec3{X: 0, Y: -1001, Z: -5},
				Radius:   1000.0,
				Material: Material{Color: RGB(0.8, 0.8, 0.8)}},
		},
		Lights: []*Light{
			{Position: Vec3{X: 5, Y: 5, Z: 0}, Color: RGB(1, 1, 1)},
		},
		BgColorStart: Vec3{X: 0.0, Y: 0.0, Z: 0.0},
		BgColorEnd:   Vec3{X: 0.5, Y: 0.7, Z: 1.0},
	}
}
