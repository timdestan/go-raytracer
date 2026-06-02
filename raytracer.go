package raytracer

import (
	"errors"
	"fmt"
	"image"
	"math"
	"math/rand"

	"github.com/timdestan/go-raytracer/internal/gml"
	"github.com/timdestan/go-raytracer/internal/prim"
)

type Ray struct {
	Origin    prim.Vec3
	Direction prim.Vec3
}

func (r *Ray) String() string {
	return fmt.Sprintf("Ray(Origin: %v, Direction: %v)", r.Origin, r.Direction)
}

type Material struct {
	Color           prim.Vec3
	Reflectivity    float64 // 0 for diffuse, 1 for perfect mirror reflection
	Fuzziness       float64 // For fuzzy reflections (0 = no fuzz, 1 = max fuzz)
	Transparency    float64 // 0.0 (opaque) to 1.0 (fully transparent)
	RefractiveIndex float64 // For transparent materials (1.0 = air, 1.5 = glass)

	// Phong parameters
	Kd               float64 // diffuse reflection coefficient
	Ks               float64 // specular reflection coefficient
	SpecularExponent float64
}

type Hit struct {
	Object   SceneObject
	T        float64
	Point    prim.Vec3
	Normal   prim.Vec3
	Material *Material
}

type SceneObject interface {
	Intersect(ray Ray) *Hit
}

type Sphere struct {
	Center        prim.Vec3
	Radius        float64
	Material      Material
	SurfaceFn     *gml.VClosure
	EvalState     *gml.EvalState
	WorldToObject prim.Mat4
	NormalMat     prim.Mat4
}

func rayToObjectSpace(ray Ray, worldToObject *prim.Mat4) Ray {
	var localRay Ray
	localRay.Origin = worldToObject.MulPoint(ray.Origin)
	localRay.Direction = worldToObject.MulDir(ray.Direction)
	return localRay
}

func evalSurfaceFn(face int, u, v float64, state *gml.EvalState, closure gml.VClosure) (*Material, error) {
	state.Push(gml.VInt(face))
	state.Push(gml.VReal(u))
	state.Push(gml.VReal(v))

	err := state.EvalClosure(closure)

	if err != nil {
		return nil, err
	}

	// x y z point        % surface color
	// 1.0 0.2 1.0		  % kd ks n

	kd, ks, n, err := gml.Pop3[gml.VReal](state)
	if err != nil {
		return nil, err
	}
	surfaceColor, err := gml.PopValue[*prim.Vec3](state)
	if err != nil {
		return nil, err
	}
	m := &Material{
		Color:            *surfaceColor,
		Kd:               float64(kd),
		Ks:               float64(ks),
		SpecularExponent: float64(n),
		// Reflectivity:     0.0,
		// Transparency:     0.0,
	}
	return m, nil
}

func (sphere *Sphere) Intersect(ray Ray) *Hit {
	ray = rayToObjectSpace(ray, &sphere.WorldToObject)

	L := sphere.Center.Sub(ray.Origin)
	t_ca := L.Dot(ray.Direction)
	if t_ca < 0.0 {
		// Center of the sphere is behind the screen.
		return nil
	}
	t_hc := math.Sqrt(square(sphere.Radius) - (L.Dot(L) - square(t_ca)))
	t0 := t_ca - t_hc
	if t0 > 0.0 {
		hitPoint := ray.Origin.Add(ray.Direction.Scale(t0))
		material, err := computeSphereSurfaceMaterial(sphere, hitPoint)
		if err != nil {
			// TODO: Render operation should be able to propagate an error.
			fmt.Printf("Sphere surfaceFn evaluation failed with error: %v\n", err)
			return nil
		}
		normalDir := sphere.NormalMat.MulDir(hitPoint.Sub(sphere.Center))

		return &Hit{
			Object:   sphere,
			T:        t0,
			Point:    hitPoint,
			Normal:   normalDir.Normalize(),
			Material: material,
		}
	}
	// TODO: Should we include these far hits?
	// t1 := t_ca + t_hc
	// if t1 > 0.0 {
	// 	return t1, true
	// }
	return nil
}

func computeSphereSurfaceMaterial(sphere *Sphere, point prim.Vec3) (*Material, error) {
	if sphere.SurfaceFn == nil {
		return &sphere.Material, nil
	}
	if sphere.EvalState == nil {
		return nil, fmt.Errorf("sphere has no eval state")
	}
	// Need to pass the face (always 0) and u and v coordinates on the stack.
	//
	// (0, u, v)
	//
	// x = sqrt(1 - y^2) * sin(2*pi u)
	// y = 2 v - 1
	// z = sqrt(1 - y^2) * cos(2*pi u)
	//
	// v = (y + 1.0) / 2.0
	// u = acos (z / sqrt (1.0 - y * y)) / (2.0 * pi)

	// TODO: How do we know the sqrt will not go negative?
	// This implies point.Y <= 1, but it's not totally clear if this
	// is guaranteed (point is computed from a hit on the sphere)
	if math.Abs(point.Y) > 1 {
		return nil, fmt.Errorf("expected |pt.Y| <= 1 in sphere surface, got %v", point)
	}

	v := (point.Y + 1.0) / 2.0
	u := math.Acos(point.Z/math.Sqrt(1.0-point.Y*point.Y)) / (2.0 * math.Pi)

	return evalSurfaceFn(0, u, v, sphere.EvalState, *sphere.SurfaceFn)
}

func (v *Sphere) String() string {
	return fmt.Sprintf("Sphere(Center: %v, Radius: %v)", v.Center, v.Radius)
}

type Plane struct {
	Normal        prim.Vec3
	D             float64
	NormalWorld   prim.Vec3
	SurfaceFn     *gml.VClosure
	EvalState     *gml.EvalState
	WorldToObject prim.Mat4
	// We don't need NormalMat since we precompute NormalWorld
}

func (p *Plane) Intersect(ray Ray) *Hit {
	ray = rayToObjectSpace(ray, &p.WorldToObject)

	denom := p.Normal.Dot(ray.Direction)
	if math.Abs(denom) < 1e-6 {
		return nil
	}
	t := (-p.D - p.Normal.Dot(ray.Origin)) / denom
	if t <= 0.0 {
		return nil
	}
	hitPoint := ray.Origin.Add(ray.Direction.Scale(t))
	material, err := computePlaneSurfaceMaterial(p, hitPoint)
	if err != nil {
		// TODO: Render operation should be able to propagate an error.
		fmt.Printf("Plane surfaceFn evaluation failed with error: %v\n", err)
		return nil
	}
	return &Hit{
		Object:   p,
		T:        t,
		Point:    ray.Origin.Add(ray.Direction.Scale(t)),
		Normal:   p.NormalWorld,
		Material: material,
	}
}

func computePlaneSurfaceMaterial(plane *Plane, point prim.Vec3) (*Material, error) {
	if plane.SurfaceFn == nil {
		return nil, fmt.Errorf("plane has no SurfaceFn")
	}
	if plane.EvalState == nil {
		return nil, fmt.Errorf("plane has no eval state")
	}
	// Need to pass the face (always 0) and u and v coordinates on the stack.
	//
	// (0, u, v) <=> (u, 0, v)

	u := point.X
	v := point.Z

	return evalSurfaceFn(0, u, v, plane.EvalState, *plane.SurfaceFn)
}

func (p *Plane) String() string {
	return fmt.Sprintf("Plane(Normal: %v, D: %v)", p.Normal, p.D)
}

type Cube struct {
	Faces [6]Plane
	// SurfaceFn and EvalState are duplicated in each face.
	// Material not supported
}

func (c *Cube) Intersect(ray Ray) *Hit {
	// TODO:
	// To handle cases where the cube is not axis aligned, we need
	// to apply the inverse transform of the cube (inverse rotation + translation)
	// to the ray origin and direction to project the ray in the cube's local space.

	var hits [6]*Hit
	for i := range c.Faces {
		hits[i] = c.Faces[i].Intersect(ray)
	}
	// We need to find a hit that is within the bounds of the cube.
	// A hit should point into the cube.
	var minHit *Hit
	// for i, hit := range hits {
	// 	if hit == nil || hit.T < 0.0 {
	// 		continue
	// 	}
	// 	for j, otherHit := range hits {
	// 		if i == j {
	// 			continue
	// 		}
	// 		if minHit == nil || hit.T < minHit.T {
	// 			minHit = hit
	// 		}
	// 	}
	// }
	return minHit

}

func computeLighting(hit *Hit, scene *Scene, ray Ray) prim.Vec3 {
	V := ray.Direction.Neg() // view vector = opposite of ray

	mat := hit.Material
	result := mat.Color.Mul(&scene.AmbientLight).Scale(mat.Kd)

	for _, light := range scene.Lights {
		lightToHit := light.Position.Sub(hit.Point)
		distToLight := lightToHit.Length()
		lightDir := lightToHit.Normalize()

		if inShadow(hit, scene, lightDir, distToLight, ray) {
			continue
		}

		// Diffuse term
		diff := math.Max(0, hit.Normal.Dot(lightDir)) * mat.Kd
		diffuse := mat.Color.Mul(&light.Color).Scale(diff)

		// Specular term (Blinn-Phong reflection)
		H := V.Add(lightDir).Normalize()
		spec := math.Max(0, hit.Normal.Dot(H))
		specular := light.Color.Scale(mat.Ks * math.Pow(spec, mat.SpecularExponent))

		result = result.Add(diffuse).Add(specular)
	}

	return result
}

// inShadow checks if the point hit by the ray is in the shadow of the light
// source, by tracing a ray from the hit point to the light and checking if
// there are any intersections with other spheres.
//
// The ray is offset by a small amount in the direction of the normal so that
// the intersection with the current sphere is not counted.
//
// lightDir is assumed to be a normal vector.
func inShadow(hit *Hit, scene *Scene, lightDir prim.Vec3, distToLight float64, ray Ray) bool {
	const epsilon = 1e-4
	shadowOrigin := hit.Point.Add(hit.Normal.Scale(epsilon))
	shadowRay := Ray{Origin: shadowOrigin, Direction: lightDir}
	for _, obj := range scene.Objects {
		if obj == hit.Object {
			continue
		}
		shadowHit := obj.Intersect(shadowRay)
		if shadowHit == nil {
			continue
		}
		// Check if the intersection is between the hit point and the light.
		if shadowHit.T*ray.Direction.Length() < distToLight {
			return true
		}
	}
	return false
}

// refract computes the direction of a refracted ray.
// `incident` is the incident vector (the direction of the incoming ray).
// `normal` is the normal vector of the surface at the hit point.
// `n1` is the refractive index of the medium the ray is leaving.
// `n2` is the refractive index of the medium the ray is entering.
//
// The function returns the refracted direction or empty vector if no refraction occurs.
func refract(incident, normal prim.Vec3, n1, n2 float64) prim.Vec3 {
	ratio := n1 / n2
	cosI := -normal.Dot(incident)
	sinT2 := ratio * ratio * (1.0 - cosI*cosI)

	// Check for total internal reflection
	if sinT2 > 1.0 {
		return prim.Vec3{}
	}

	cosT := math.Sqrt(1.0 - sinT2)
	return incident.Scale(ratio).Add(normal.Scale(ratio*cosI - cosT))
}

// fresnel computes the reflection coefficient (Kr) using Schlick's approximation.
// normal: surface normal (unit vector)
// incident: incoming ray direction (unit vector, pointing INTO the surface)
// ior: index of refraction of the material
func fresnel(normal, incident prim.Vec3, ior float64) float64 {
	// cosi := clamp(-1, 1, incident.Dot(normal))
	cosi := incident.CosineSimilarity(normal)
	etai, etat := 1.0, ior // assume ray is coming from air (n=1)

	// Compute R0
	r0 := (etai - etat) / (etai + etat)
	r0 = r0 * r0

	cost := math.Abs(cosi)
	return r0 + (1-r0)*math.Pow(1-cost, 5) // Schlick's approximation
}

func closestHit(scene *Scene, ray Ray) *Hit {
	var minHit *Hit
	for _, obj := range scene.Objects {
		hit := obj.Intersect(ray)
		if hit == nil {
			continue
		}
		if minHit == nil || hit.T < minHit.T {
			minHit = hit
		}
	}
	return minHit
}

// traceRay returns the color of the closest sphere hit by the ray, or nil
// if no sphere is hit.
func traceRay(scene *Scene, ray Ray, depth int) prim.Vec3 {
	if depth <= 0 {
		// Recursion limit
		return prim.Vec3{}
	}
	hit := closestHit(scene, ray)
	if hit == nil {
		// Calculate background color (linear gradient).
		t := 0.5 * (ray.Direction.Y + 1.0)
		return scene.BgColorStart.Lerp(scene.BgColorEnd, t)
	}

	surfaceColor := computeLighting(hit, scene, ray)

	mat := hit.Material
	if mat.Reflectivity == 0 && mat.Transparency == 0 {
		return surfaceColor.Clamp()
	}

	// Handle reflection and transparency based on material properties
	reflectedColor := prim.Vec3{}
	if mat.Reflectivity > 0 {
		// For fuzzy reflections, add a random component to the reflection direction.
		fuzz := mat.Fuzziness
		reflectedDir := ray.Direction.Sub(hit.Normal.Scale(2.0 * ray.Direction.Dot(hit.Normal)))
		// "random" vector
		randomVector := prim.Vec3{
			X: math.Cos(fuzz) * math.Cos(fuzz),
			Y: math.Sin(fuzz) * math.Sin(fuzz),
			Z: 0,
		}
		reflectionRay := Ray{
			Origin:    hit.Point.Add(hit.Normal.Scale(1e-4)),
			Direction: reflectedDir.Add(randomVector.Scale(fuzz)).Normalize(),
		}
		reflectedColor = traceRay(scene, reflectionRay, depth-1)
	}

	refractedColor := prim.Vec3{}
	if mat.Transparency > 0 {
		// This assumes the outer medium is air.
		n1 := 1.0
		n2 := mat.RefractiveIndex

		// If the dot product of the ray direction and the normal is positive,
		// then the ray is inside the object and trying to exit.
		// In this case, must swap the refractive indices.
		normal := hit.Normal
		if ray.Direction.Dot(normal) > 0.0 {
			n1, n2 = n2, n1
			// We also need to invert the normal
			normal = normal.Scale(-1.0)
		}

		refractedDir := refract(ray.Direction, normal, n1, n2)

		if !refractedDir.IsZero() {
			// Create the refracted ray. We offset the origin slightly to avoid self-intersection.
			refractedRay := Ray{Origin: hit.Point.Sub(normal.Scale(1e-4)), Direction: refractedDir}

			// Recursively trace the refracted ray
			refractedColor = traceRay(scene, refractedRay, depth-1)
		}
	}
	kr := fresnel(hit.Normal, ray.Direction, mat.RefractiveIndex)
	return surfaceColor.Scale(1.0 - mat.Transparency).Add(reflectedColor.Scale(kr).Add(refractedColor.Scale(1.0 - kr))).Clamp()
}

func square(x float64) float64 {
	return x * x
}

type Scene struct {
	WidthPx, HeightPx int

	// Fov is the camera field of view in degrees
	Fov float64

	RecursionDepth int

	Objects []SceneObject
	Lights  []*gml.PointLight

	AmbientLight prim.Vec3

	// BgColorStart and BgColorEnd define the 2 ends of the gradient
	// background color.
	BgColorStart, BgColorEnd prim.Vec3
}

func Render(scene *Scene) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, scene.WidthPx, scene.HeightPx))

	var recursionLimit = scene.RecursionDepth
	if recursionLimit <= 0 {
		recursionLimit = 3
	}

	if scene.Fov <= 0.0 {
		fmt.Printf("warning: fov not specified, using default of 90 degrees\n")
		scene.Fov = 90.0
	}
	fovRadians := scene.Fov * math.Pi / 180.0
	viewportWidth := 2.0 / math.Tan(fovRadians/2.0)
	viewportHeight := viewportWidth * (float64(scene.HeightPx) / float64(scene.WidthPx))

	eyePosition := prim.Vec3{
		X: 0.0,
		Y: 0.0,
		Z: -1.0,
	}

	for x := range scene.WidthPx {
		for y := range scene.HeightPx {
			// Subsample for antialiasing
			totalColor := prim.Vec3{}
			const numSamples = 4
			for range numSamples {
				// Map pixel coordinates to world coordinates.
				du := rand.Float64() - 0.5
				dv := rand.Float64() - 0.5
				u := (float64(x)+du)/float64(scene.WidthPx-1)*viewportWidth - viewportWidth/2.0
				v := (float64(y)+dv)/float64(scene.HeightPx-1)*viewportHeight - viewportHeight/2.0

				var ray Ray
				ray.Origin = prim.Vec3{X: u, Y: -v, Z: 0.0} // screen point
				ray.Direction = ray.Origin.Sub(eyePosition).Normalize()

				totalColor = totalColor.Add(traceRay(scene, ray, recursionLimit))
			}
			img.Set(x, y, totalColor.Scale(1.0/float64(numSamples)))
		}
	}
	return img
}

func ParseAndRenderGML(programText string) (image.Image, error) {
	state := gml.NewEvalState()

	images := make(map[string]image.Image)
	state.Render = func(state *gml.EvalState, args *gml.RenderArgs) error {
		scene, err := ConvertRenderArgsToScene(args, state)
		if err != nil {
			return err
		}
		images[args.File] = Render(scene)
		return nil
	}

	err := state.ParseAndEval(programText)
	if err != nil {
		return nil, err
	}
	if len(images) > 1 {
		// We could easily support this if we wanted to.
		return nil, errors.New("multiple images were rendered by the GML program")
	}
	// Return first (only) image.
	for _, img := range images {
		return img, nil
	}
	return nil, errors.New("no image was rendered by the GML program")
}

func ConvertRenderArgsToScene(args *gml.RenderArgs, state *gml.EvalState) (*Scene, error) {
	convertedObjects, err := convertGMLSceneObjects([]gml.SceneObject{args.Scene}, state)
	if err != nil {
		return nil, err
	}
	return &Scene{
		WidthPx:        args.Width,
		HeightPx:       args.Height,
		Fov:            args.Fov,
		RecursionDepth: args.Depth,
		Objects:        convertedObjects,
		Lights:         args.Lights,
		AmbientLight:   *args.AmbientLight,
	}, nil
}

func convertGMLSceneObjects(sceneObjects []gml.SceneObject, evalState *gml.EvalState) ([]SceneObject, error) {
	toVisit := sceneObjects
	var results []SceneObject
	for len(toVisit) > 0 {
		sceneObject := toVisit[0]
		toVisit = toVisit[1:]
		switch typedObject := sceneObject.(type) {
		case *gml.Sphere:
			var worldToObject prim.Mat4

			if typedObject.TransformMat != nil {
				worldToObject = *typedObject.TransformMat.Inverse()
			} else {
				worldToObject = prim.IdentityMatrix()
			}

			results = append(results, &Sphere{
				Center:        typedObject.Center,
				Radius:        float64(typedObject.Radius),
				SurfaceFn:     &typedObject.SurfaceFn,
				EvalState:     evalState,
				WorldToObject: worldToObject,
				NormalMat:     *worldToObject.Transpose(),

				// Material : nil
			})
		case *gml.Cube:
			fmt.Println("WARN: cube skipped in rendering")
			// cube := &Cube{}
			// TODO: We don't represent the GML cube as a slice of faces anymore.
			// Instead we just have 2 opposite corners and an XForm struct (translation, rotation, scale).
			// for i, p := range typedObject.Faces {
			// 	cube.Faces[i] = Plane{
			// 		Normal:    p.Normal,
			// 		D:         -p.Normal.Dot(&p.Point),
			// 		SurfaceFn: &typedObject.SurfaceFn,
			// 		EvalState: evalState,
			// 	}
			// }
			// result = append(result, cube)
		case *gml.Plane:
			var worldToObject prim.Mat4

			if typedObject.TransformMat != nil {
				worldToObject = *typedObject.TransformMat.Inverse()
			} else {
				worldToObject = prim.IdentityMatrix()
			}

			results = append(results, &Plane{
				Normal:        typedObject.Plane.Normal,
				NormalWorld:   worldToObject.Transpose().MulDir(typedObject.Plane.Normal),
				D:             -typedObject.Plane.Normal.Dot(typedObject.Plane.Point),
				SurfaceFn:     &typedObject.SurfaceFn,
				EvalState:     evalState,
				WorldToObject: worldToObject,
			})
		case *gml.Union:
			// Right now we just flatten everything...
			toVisit = append(toVisit, typedObject.Objects...)
		default:
			return nil, fmt.Errorf("unknown scene object type %T", sceneObject)
		}
	}
	return results, nil
}
