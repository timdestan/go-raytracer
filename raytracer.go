package raytracer

import (
	"errors"
	"fmt"
	"image"
	"math"
	"math/rand"

	"github.com/timdestan/go-raytracer/internal/gml"
)

type Vec3 struct {
	X, Y, Z float64
}

func (v *Vec3) String() string {
	return fmt.Sprintf("Vec3(%.4f, %.4f, %.4f)", v.X, v.Y, v.Z)
}

// RGB is a convenience function to construct a vector
// from normalized RGB values [0.0, 1.0].
func RGB(r, g, b float64) Vec3 {
	return Vec3{X: r, Y: g, Z: b}
}

func (v *Vec3) Add(other *Vec3) *Vec3 {
	return &Vec3{
		X: v.X + other.X,
		Y: v.Y + other.Y,
		Z: v.Z + other.Z,
	}
}

// AddI is an in-place version of Add
func (v *Vec3) AddI(other *Vec3) *Vec3 {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
	return v
}

func (v *Vec3) Sub(other *Vec3) *Vec3 {
	return &Vec3{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

// Mul multiples two vectors pointwise.
func (v *Vec3) Mul(other *Vec3) *Vec3 {
	return &Vec3{
		X: v.X * other.X,
		Y: v.Y * other.Y,
		Z: v.Z * other.Z,
	}
}

func (v *Vec3) Dot(other *Vec3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v *Vec3) CosineSimilarity(other *Vec3) float64 {
	return v.Dot(other) / (v.Length() * other.Length())
}

func (v *Vec3) LerpI(other *Vec3, t float64) *Vec3 {
	v.X += (other.X - v.X) * t
	v.Y += (other.Y - v.Y) * t
	v.Z += (other.Z - v.Z) * t
	return v
}

func (v *Vec3) Scale(s float64) *Vec3 {
	return &Vec3{
		X: v.X * s,
		Y: v.Y * s,
		Z: v.Z * s,
	}
}

func (v *Vec3) Normalize() *Vec3 {
	magnitude := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	return &Vec3{
		X: v.X / magnitude,
		Y: v.Y / magnitude,
		Z: v.Z / magnitude,
	}
}

func (v *Vec3) Neg() *Vec3 {
	return &Vec3{
		X: -v.X,
		Y: -v.Y,
		Z: -v.Z,
	}
}

func (v *Vec3) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func (v *Vec3) IsZero() bool {
	return v.X == 0.0 && v.Y == 0.0 && v.Z == 0.0
}

// RGBA implements the image.Color interface
func (v *Vec3) RGBA() (r, g, b, a uint32) {
	const max = 0xffff
	return uint32(v.X * max), uint32(v.Y * max), uint32(v.Z * max), max
}

// ClampI clamps the X, Y, and Z values between 0 and 1, in place.
func (c *Vec3) ClampI() *Vec3 {
	c.X = clamp(0, 1, c.X)
	c.Y = clamp(0, 1, c.Y)
	c.Z = clamp(0, 1, c.Z)
	return c
}

// Reflect reflects this vector around the given axis vector.
func (c *Vec3) Reflect(axis *Vec3) *Vec3 {
	return axis.Scale(2 * axis.Dot(c)).Sub(c)
}

type Ray struct {
	Origin    *Vec3
	Direction *Vec3
}

func (r *Ray) String() string {
	return fmt.Sprintf("Ray(Origin: %v, Direction: %v)", r.Origin, r.Direction)
}

type Material struct {
	Color           Vec3
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
	Point    *Vec3
	Normal   *Vec3
	Material *Material
}

type SceneObject interface {
	Intersect(ray *Ray) *Hit
}

type Sphere struct {
	Center    Vec3
	Radius    float64
	Material  Material
	SurfaceFn *gml.VClosure
	EvalState *gml.EvalState
}

func (sphere *Sphere) Intersect(ray *Ray) *Hit {
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
		material, err := computeSphereSurface(sphere, hitPoint)
		if err != nil {
			// TODO: Render operation should be able to propagate an error.
			fmt.Printf("Sphere surfaceFn evaluation failed with error: %v\n", err)
			return nil
		}
		return &Hit{
			Object:   sphere,
			T:        t0,
			Point:    hitPoint,
			Normal:   hitPoint.Sub(&sphere.Center).Normalize(),
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

func computeSphereSurface(sphere *Sphere, point *Vec3) (*Material, error) {
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
	// x = sqrt(1 - y^2)sin(2*pi u)
	// y = 2 v - 1
	// z = sqrt(1 - y^2)cos(2*pi u)
	//
	// v = (y + 1.0) / 2.0
	// u = acos (z / sqrt (1.0 - y * y)) / (2.0 * pi)

	// TODO: How do we know the sqrt will not go negative?
	v := (point.Y + 1.0) / 2.0
	u := math.Acos(point.Z/math.Sqrt(1.0-point.Y*point.Y)) / (2.0 * math.Pi)

	// TODO: Might be simpler to just construct the token list:
	//
	// 0 u v sphere.EvalState.code apply
	//
	// And evaluate that

	sphere.EvalState.Push(gml.VInt(0))
	sphere.EvalState.Push(gml.VReal(u))
	sphere.EvalState.Push(gml.VReal(v))

	oldEnv := sphere.EvalState.Env
	defer func() { sphere.EvalState.Env = oldEnv }()
	sphere.EvalState.Env = sphere.SurfaceFn.Env
	err := sphere.EvalState.Eval(sphere.SurfaceFn.Code)
	if err != nil {
		return nil, err
	}

	// x y z point        % surface color
	// 1.0 0.2 1.0		  % kd ks n

	kd, ks, n, err := gml.Pop3[gml.VReal](sphere.EvalState)
	if err != nil {
		return nil, err
	}
	surfaceColor, err := gml.PopValue[gml.Point](sphere.EvalState)
	if err != nil {
		return nil, err
	}
	return &Material{
		Color:            pointToVec3(surfaceColor),
		Kd:               float64(kd),
		Ks:               float64(ks),
		SpecularExponent: float64(n),
		// Reflectivity:     0.0,
		// Transparency:     0.0,
	}, nil
}

func (v *Sphere) String() string {
	// Doesn't include color
	return fmt.Sprintf("Sphere(Center: %v, Radius: %v)", v.Center, v.Radius)
}

// Light represents a point light source.
type Light struct {
	Position Vec3
	Color    Vec3
}

var Magenta = RGB(1, 0, 1)

func (l *Light) String() string {
	return fmt.Sprintf("Light(Position: %v, Color: %v)", l.Position, l.Color)
}

func computeLighting(hit *Hit, scene *Scene, ray *Ray) *Vec3 {
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

		result.AddI(diffuse).AddI(specular)
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
func inShadow(hit *Hit, scene *Scene, lightDir *Vec3, distToLight float64, ray *Ray) bool {
	const epsilon = 1e-4
	shadowOrigin := hit.Point.Add(hit.Normal.Scale(epsilon))
	shadowRay := &Ray{Origin: shadowOrigin, Direction: lightDir}
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
// The function returns the refracted direction or nil if no refraction occurs.
func refract(incident, normal *Vec3, n1, n2 float64) *Vec3 {
	ratio := n1 / n2
	cosI := -normal.Dot(incident)
	sinT2 := ratio * ratio * (1.0 - cosI*cosI)

	// Check for total internal reflection
	if sinT2 > 1.0 {
		return nil
	}

	cosT := math.Sqrt(1.0 - sinT2)
	return incident.Scale(ratio).Add(normal.Scale(ratio*cosI - cosT))
}

// fresnel computes the reflection coefficient (Kr) using Schlick's approximation.
// normal: surface normal (unit vector)
// incident: incoming ray direction (unit vector, pointing INTO the surface)
// ior: index of refraction of the material
func fresnel(normal, incident *Vec3, ior float64) float64 {
	// cosi := clamp(-1, 1, incident.Dot(normal))
	cosi := incident.CosineSimilarity(normal)
	etai, etat := 1.0, ior // assume ray is coming from air (n=1)

	// Compute R0
	r0 := (etai - etat) / (etai + etat)
	r0 = r0 * r0

	cost := math.Abs(cosi)
	return r0 + (1-r0)*math.Pow(1-cost, 5) // Schlick's approximation
}

// clamp limits x between min and max
func clamp(min, max, x float64) float64 {
	return math.Min(math.Max(x, min), max)
}

func closestHit(scene *Scene, ray *Ray) *Hit {
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
func traceRay(scene *Scene, ray *Ray, depth int) *Vec3 {
	if depth <= 0 {
		// Recursion limit
		return &Vec3{}
	}
	hit := closestHit(scene, ray)
	if hit == nil {
		// Calculate background color (linear gradient).
		t := 0.5 * (ray.Direction.Y + 1.0)
		return scene.BgColorStart.LerpI(&scene.BgColorEnd, t)
	}

	surfaceColor := computeLighting(hit, scene, ray)

	mat := hit.Material
	if mat.Reflectivity == 0 && mat.Transparency == 0 {
		return surfaceColor.ClampI()
	}

	// Handle reflection and transparency based on material properties
	reflectedColor := &Vec3{}
	if mat.Reflectivity > 0 {
		// For fuzzy reflections, add a random component to the reflection direction.
		fuzz := mat.Fuzziness
		reflectedDir := ray.Direction.Sub(hit.Normal.Scale(2.0 * ray.Direction.Dot(hit.Normal)))
		// "random" vector
		randomVector := Vec3{math.Cos(fuzz) * math.Cos(fuzz), math.Sin(fuzz) * math.Sin(fuzz), 0}
		reflectionRay := Ray{
			Origin:    hit.Point.Add(hit.Normal.Scale(1e-4)),
			Direction: reflectedDir.Add(randomVector.Scale(fuzz)).Normalize(),
		}
		reflectedColor = traceRay(scene, &reflectionRay, depth-1)
	}

	refractedColor := &Vec3{}
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

		if refractedDir != nil {
			// Create the refracted ray. We offset the origin slightly to avoid self-intersection.
			refractedRay := Ray{Origin: hit.Point.Sub(normal.Scale(1e-4)), Direction: refractedDir}

			// Recursively trace the refracted ray
			refractedColor = traceRay(scene, &refractedRay, depth-1)
		}
	}
	kr := fresnel(hit.Normal, ray.Direction, mat.RefractiveIndex)
	return surfaceColor.Scale(1.0 - mat.Transparency).AddI(reflectedColor.Scale(kr).AddI(refractedColor.Scale(1.0 - kr))).ClampI()
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
	Lights  []*Light

	AmbientLight Vec3

	// BgColorStart and BgColorEnd define the 2 ends of the gradient
	// background color.
	BgColorStart, BgColorEnd Vec3
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
	fmt.Printf("viewport size: %f x %f\n", viewportWidth, viewportHeight)

	eyePosition := &Vec3{
		X: 0.0,
		Y: 0.0,
		Z: -1.0,
	}

	for x := range scene.WidthPx {
		for y := range scene.HeightPx {
			// Subsample for antialiasing
			totalColor := &Vec3{}
			const numSamples = 4
			for range numSamples {
				// Map pixel coordinates to world coordinates.
				du := rand.Float64() - 0.5
				dv := rand.Float64() - 0.5
				u := (float64(x)+du)/float64(scene.WidthPx-1)*viewportWidth - viewportWidth/2.0
				v := (float64(y)+dv)/float64(scene.HeightPx-1)*viewportHeight - viewportHeight/2.0
				screenPoint := &Vec3{
					X: u,
					Y: -v,
					Z: 0.0,
				}
				ray := Ray{
					Origin:    screenPoint,
					Direction: screenPoint.Sub(eyePosition).Normalize(),
				}
				color := traceRay(scene, &ray, recursionLimit)
				totalColor.AddI(color)
			}
			img.Set(x, y, totalColor.Scale(1.0/float64(numSamples)))
		}
	}
	return img
}

func ParseAndRenderGML(programText string) (image.Image, error) {
	token, err := gml.Parse(programText)
	if err != nil {
		return nil, err
	}
	state := gml.NewEvalState()

	// TODO: At the moment we ignore any filename requested and always write
	// to one image. All example programs at the moment only render once.
	var renderedImage image.Image
	state.Render = func(state *gml.EvalState, args *gml.RenderArgs) error {
		// Create a scene object from the render args.

		convertedObjects, err := convertGMLSceneObjects([]gml.SceneObject{args.Scene}, state)
		if err != nil {
			return err
		}
		scene := &Scene{
			WidthPx:  args.Width,
			HeightPx: args.Height,

			Fov:            args.Fov,
			RecursionDepth: args.Depth,

			Objects: convertedObjects,
			Lights:  convertGMLLights(args.Lights),

			AmbientLight: pointToVec3(*args.AmbientLight),
		}
		renderedImage = Render(scene)
		return nil
	}

	err = state.Eval(token)
	if err != nil {
		return nil, err
	}
	if renderedImage == nil || renderedImage.Bounds().Empty() {
		return nil, errors.New("no image was rendered by the GML program")
	}
	return renderedImage, nil
}

func convertGMLSceneObjects(sceneObjects []gml.SceneObject, evalState *gml.EvalState) ([]SceneObject, error) {
	toVisit := sceneObjects
	var result []SceneObject
	for len(toVisit) > 0 {
		sceneObject := toVisit[0]
		toVisit = toVisit[1:]
		switch typedObject := sceneObject.(type) {
		case *gml.Sphere:
			result = append(result, &Sphere{
				Center: pointToVec3(typedObject.Center),
				Radius: float64(typedObject.Radius),
				// Material: nil,
				SurfaceFn: &typedObject.SurfaceFn,
				EvalState: evalState,
			})
		case *gml.Union:
			toVisit = append(toVisit, typedObject.Objects...)
		default:
			return nil, fmt.Errorf("unknown scene object type %T", sceneObject)
		}
	}
	return result, nil
}

func convertGMLLights(lights []*gml.PointLight) []*Light {
	var result []*Light
	for _, light := range lights {
		result = append(result, &Light{
			Position: pointToVec3(light.Position),
			Color:    pointToVec3(light.Color),
		})
	}
	return result
}

func pointToVec3(point gml.Point) Vec3 {
	return Vec3{
		X: float64(point.X),
		Y: float64(point.Y),
		Z: float64(point.Z),
	}
}
