package raytracer

import (
	"errors"
	"fmt"
	"image"
	"log"
	"math"
	"math/rand/v2"

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

type Hit struct {
	Object   SceneObject
	T        float64
	PointObj prim.Vec3
	// Face indicates which face was hit, only relevant for objects
	// with multiple faces.
	Face int
}

// HitEx extends hit with additional computed surface properties.
type HitEx struct {
	Hit
	PointWorld  prim.Vec3
	NormalWorld prim.Vec3
	Material    *gml.Material
}

type SceneObject interface {
	Intersect(ray Ray) *Hit
	ComputeSurfaceProps(hit Hit) (HitEx, error)
}

type Sphere struct {
	Center        prim.Vec3
	Radius        float64
	SurfaceFn     gml.VSurfaceFn
	EvalState     *gml.EvalState
	ObjectToWorld prim.Mat4
	WorldToObject prim.Mat4
	NormalMat     prim.Mat4
}

func rayToObjectSpace(ray Ray, worldToObject *prim.Mat4) Ray {
	var localRay Ray
	localRay.Origin = worldToObject.MulPoint(ray.Origin)
	localRay.Direction = worldToObject.MulDir(ray.Direction)
	return localRay
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
		return &Hit{
			Object:   sphere,
			T:        t0,
			PointObj: ray.Origin.Add(ray.Direction.Scale(t0)),
		}
	}
	// TODO: Should we include these far hits?
	// t1 := t_ca + t_hc
	// if t1 > 0.0 {
	// 	 ...
	// }
	return nil
}

func (sphere *Sphere) ComputeSurfaceProps(hit Hit) (HitEx, error) {
	material, err := computeSphereSurfaceMaterial(sphere, hit.PointObj)
	if err != nil {
		return HitEx{}, err
	}
	normalDir := sphere.NormalMat.MulDir(hit.PointObj.Sub(sphere.Center))
	return HitEx{
		Hit:         hit,
		PointWorld:  sphere.ObjectToWorld.MulPoint(hit.PointObj),
		NormalWorld: normalDir.Normalize(),
		Material:    material,
	}, nil
}

func computeSphereSurfaceMaterial(sphere *Sphere, point prim.Vec3) (*gml.Material, error) {
	if sphere.SurfaceFn.Material != nil {
		return sphere.SurfaceFn.Material, nil
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

	// GML spheres should always be unit spheres (the transformation matrix may
	// scale and move them). The spheres from example scene break this rule,
	// but these set Material directly so should not reach this point.
	if math.Abs(point.Y) > 1 {
		return nil, fmt.Errorf("expected |pt.Y| <= 1 in sphere surface, got %v", point)
	}

	v := (point.Y + 1.0) / 2.0
	u := math.Acos(point.Z/math.Sqrt(1.0-point.Y*point.Y)) / (2.0 * math.Pi)

	return gml.EvalSurfaceFn(0, u, v, sphere.EvalState, &sphere.SurfaceFn)
}

func (v *Sphere) String() string {
	return fmt.Sprintf("Sphere(Center: %v, Radius: %v)", v.Center, v.Radius)
}

type Plane struct {
	Side          prim.CubeSide // always 0 if not part of cube
	Normal        prim.Vec3
	D             float64
	NormalWorld   prim.Vec3
	SurfaceFn     gml.VSurfaceFn
	EvalState     *gml.EvalState
	ObjectToWorld prim.Mat4
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
	return &Hit{
		Object:   p,
		T:        t,
		PointObj: ray.Origin.Add(ray.Direction.Scale(t)),
	}
}

func (p *Plane) ComputeSurfaceProps(hit Hit) (HitEx, error) {
	material, err := computePlaneSurfaceMaterial(p, hit.PointObj)
	if err != nil {
		return HitEx{}, err
	}

	return HitEx{
		Hit:         hit,
		PointWorld:  p.ObjectToWorld.MulPoint(hit.PointObj),
		NormalWorld: p.NormalWorld,
		Material:    material,
	}, nil
}

func computePlaneSurfaceMaterial(plane *Plane, point prim.Vec3) (*gml.Material, error) {
	// Need to pass the face (always 0) and u and v coordinates on the stack.
	//
	// (0, u, v) <=> (u, 0, v)

	u := point.X
	v := point.Z

	return gml.EvalSurfaceFn(int(plane.Side), u, v, plane.EvalState, &plane.SurfaceFn)
}

func (p *Plane) String() string {
	return fmt.Sprintf("Plane(Normal: %v, D: %v)", p.Normal, p.D)
}

type Cube struct {
	Faces         [prim.NUM_CUBE_SIDES]Plane
	ObjectToWorld prim.Mat4
	WorldToObject prim.Mat4
	// A lot of data is duplicated in each Plane.
}

func (c *Cube) Intersect(ray Ray) *Hit {
	// ray = rayToObjectSpace(ray, &c.WorldToObject)

	var minHit *Hit
	for faceIndex, face := range c.Faces {
		hit := face.Intersect(ray)
		if hit == nil || hit.T < 0.0 {
			continue
		}

		pt := hit.PointObj
		// Is the hit within the bounds of the cube?
		// We may want to not check the dimension of our side to avoid floating
		// point precision issues.
		if pt.X < 0 || pt.X > 1 || pt.Y < 0 || pt.Y > 1 || pt.Z < 0 || pt.Z > 1 {
			continue
		}

		hit.Object = c
		hit.Face = faceIndex

		if minHit == nil || hit.T < minHit.T {
			minHit = hit
		}
	}
	return minHit
}

func (c *Cube) ComputeSurfaceProps(hit Hit) (HitEx, error) {
	if hit.Face < 0 || hit.Face >= int(prim.NUM_CUBE_SIDES) {
		return HitEx{}, fmt.Errorf("face index out of range: %d", hit.Face)
	}

	face := c.Faces[hit.Face]

	material, err := computePlaneSurfaceMaterial(&face, hit.PointObj)
	if err != nil {
		return HitEx{}, err
	}

	return HitEx{
		Hit:         hit,
		PointWorld:  face.ObjectToWorld.MulPoint(hit.PointObj),
		NormalWorld: face.NormalWorld,
		Material:    material,
	}, nil
}

func computeLighting(hit *HitEx, scene *Scene, ray Ray) prim.Vec3 {
	V := ray.Direction.Neg() // view vector = opposite of ray

	mat := hit.Material
	result := scene.AmbientLight.Scale(mat.Kd)

	for _, light := range scene.Lights {
		lightToHit := light.Position.Sub(hit.PointWorld)
		distToLight := lightToHit.Length()
		lightDir := lightToHit.Normalize()

		if inShadow(hit, scene, lightDir, distToLight, ray) {
			continue
		}

		// Diffuse term
		nDotL := hit.NormalWorld.Dot(lightDir)
		// TODO: Should we be skipping the contribution of lights if this dot product
		// is not positive?
		diffuse := light.Color.Scale(nDotL * mat.Kd)

		// Specular term (Blinn-Phong reflection)
		H := V.Add(lightDir).Normalize()
		spec := math.Max(0, hit.NormalWorld.Dot(H))
		specular := light.Color.Scale(mat.Ks * math.Pow(spec, mat.SpecularExponent))

		result = result.Add(diffuse).Add(specular)
	}

	return result
}

// inShadow checks if the point hit by the ray is in the shadow of the light
// source, by tracing a ray from the hit point to the light and checking if
// there are any intersections with other objects.
//
// The ray is offset by a small amount in the direction of the normal so that
// the intersection with the current object is not counted.
//
// lightDir is assumed to be a normal vector.
func inShadow(hit *HitEx, scene *Scene, lightDir prim.Vec3, distToLight float64, ray Ray) bool {
	const epsilon = 1e-4
	shadowOrigin := hit.PointWorld.Add(hit.NormalWorld.Scale(epsilon))
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
	// PERF: We currently compute the surface function as part of Intersect,
	// but we really only need to do this for the closest hit.
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

// traceRay returns the color of the closest object hit by the ray, or nil
// if no object is hit.
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
	hitEx, err := hit.Object.ComputeSurfaceProps(*hit)
	if err != nil {
		panic(fmt.Errorf("error computing hit properties of %+v: %w", hit, err))
	}

	lighting := computeLighting(&hitEx, scene, ray)

	mat := hitEx.Material
	if mat.Reflectivity == 0 && mat.Transparency == 0 {
		return lighting.Mul(&mat.Color).Clamp()
	}

	// Handle reflection and transparency based on material properties
	reflectedColor := prim.Vec3{}
	if mat.Reflectivity > 0 {
		reflectedDir := ray.Direction.Sub(hitEx.NormalWorld.Scale(2.0 * ray.Direction.Dot(hitEx.NormalWorld)))

		// For fuzzy reflections, add a random component to the reflection direction.
		if fuzz := mat.Fuzziness; fuzz >= 0 {
			reflectedDir = reflectedDir.Add(prim.Vec3{
				X: fuzz * math.Cos(fuzz) * math.Cos(fuzz),
				Y: fuzz * math.Sin(fuzz) * math.Sin(fuzz),
				Z: 0,
			})
		}

		reflectionRay := Ray{
			Origin:    hitEx.PointWorld.Add(hitEx.NormalWorld.Scale(1e-4)),
			Direction: reflectedDir.Normalize(),
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
		normal := hitEx.NormalWorld
		if ray.Direction.Dot(normal) > 0.0 {
			n1, n2 = n2, n1
			// We also need to invert the normal
			normal = normal.Scale(-1.0)
		}

		refractedDir := refract(ray.Direction, normal, n1, n2)

		if !refractedDir.IsZero() {
			// Create the refracted ray. We offset the origin slightly to avoid self-intersection.
			refractedRay := Ray{Origin: hitEx.PointWorld.Sub(normal.Scale(1e-4)), Direction: refractedDir}

			// Recursively trace the refracted ray
			refractedColor = traceRay(scene, refractedRay, depth-1)
		}
	}
	if mat.Transparency == 0 {
		return lighting.Add(reflectedColor.Scale(mat.Reflectivity)).Mul(&mat.Color).Clamp()
	}
	kr := fresnel(hitEx.NormalWorld, ray.Direction, mat.RefractiveIndex)
	return lighting.Scale(1.0 - mat.Transparency).Add(reflectedColor.Scale(kr).Add(refractedColor.Scale(1.0 - kr))).Mul(&mat.Color).Clamp()
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
	rng := rand.New(rand.NewPCG(0xDEAD, 0xBEEF))

	var recursionLimit = scene.RecursionDepth
	if recursionLimit <= 0 {
		recursionLimit = 3
	}

	if scene.Fov <= 0.0 {
		log.Printf("WARN: fov not specified, using default of 90 degrees\n")
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
				dx := rng.Float64() - 0.5
				dy := rng.Float64() - 0.5
				u := (float64(x)+dx)/float64(scene.WidthPx-1)*viewportWidth - viewportWidth/2.0
				v := (float64(y)+dy)/float64(scene.HeightPx-1)*viewportHeight - viewportHeight/2.0

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
	createMatrices := func(xform *prim.Mat4) (objectToWorld, worldToObject prim.Mat4) {
		if xform == nil {
			return prim.IdentityMatrix(), prim.IdentityMatrix()
		}
		return *xform, *xform.Inverse()
	}

	createPlane := func(point prim.Vec3, normal prim.Vec3, objectToWorld, worldToObject prim.Mat4, surfaceFn gml.VSurfaceFn) Plane {
		return Plane{
			Normal:        normal,
			NormalWorld:   worldToObject.Transpose().MulDir(normal).Normalize(),
			D:             -normal.Dot(point),
			SurfaceFn:     surfaceFn,
			EvalState:     evalState,
			ObjectToWorld: objectToWorld,
			WorldToObject: worldToObject,
		}
	}

	toVisit := sceneObjects
	var results []SceneObject
	for len(toVisit) > 0 {
		sceneObject := toVisit[0]
		toVisit = toVisit[1:]
		switch typedObject := sceneObject.(type) {
		case *gml.Sphere:
			objectToWorld, worldToObject := createMatrices(typedObject.TransformMat)

			results = append(results, &Sphere{
				Center:        typedObject.Center,
				Radius:        float64(typedObject.Radius),
				SurfaceFn:     typedObject.SurfaceFn,
				EvalState:     evalState,
				ObjectToWorld: objectToWorld,
				WorldToObject: worldToObject,
				NormalMat:     *worldToObject.Transpose(),
			})
		case *gml.Cube:
			objectToWorld, worldToObject := createMatrices(typedObject.TransformMat)

			cube := &Cube{
				ObjectToWorld: objectToWorld,
				WorldToObject: worldToObject,
			}
			for i, side := range prim.PlanesForUnitCube() {
				plane := createPlane(side.Point, side.Normal, objectToWorld, worldToObject, typedObject.SurfaceFn)
				plane.Side = prim.CubeSide(i)
				cube.Faces[i] = plane
			}

			results = append(results, cube)
		case *gml.Plane:
			objectToWorld, worldToObject := createMatrices(typedObject.TransformMat)

			plane := createPlane(typedObject.Plane.Point, typedObject.Plane.Normal, objectToWorld, worldToObject, typedObject.SurfaceFn)

			results = append(results, &plane)
		case *gml.Union:
			// Right now we just flatten everything...
			toVisit = append(toVisit, typedObject.Objects...)
		default:
			return nil, fmt.Errorf("unknown scene object type %T", sceneObject)
		}
	}
	return results, nil
}
