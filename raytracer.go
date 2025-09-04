package raytracer

import (
	"fmt"
	"image"
	"math"
	"math/rand"
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
	AbsorptionColor Vec3    // Beer’s law tint (optional, only for transmissive)
}

type Sphere struct {
	Center   Vec3
	Radius   float64
	Material Material
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

func intersectSphere(sphere *Sphere, ray *Ray) (float64, bool) {
	L := sphere.Center.Sub(ray.Origin)
	t_ca := L.Dot(ray.Direction)
	if t_ca < 0.0 {
		// Center of the sphere is behind the screen.
		return 0.0, false
	}
	t_hc := math.Sqrt(square(sphere.Radius) - (L.Dot(L) - square(t_ca)))
	t0 := t_ca - t_hc
	if t0 > 0.0 {
		return t0, true
	}
	// TODO: Should we include these far hits?
	// t1 := t_ca + t_hc
	// if t1 > 0.0 {
	// 	return t1, true
	// }
	return 0.0, false
}

func computeLighting(hit *Hit, scene *Scene, ray *Ray) *Vec3 {
	V := ray.Direction.Neg() // view vector = opposite of ray

	const ambientTerm = 0.1 // Constant ambient term
	mat := &hit.Sphere.Material
	result := mat.Color.Scale(ambientTerm)

	for _, light := range scene.Lights {
		lightToHit := light.Position.Sub(hit.Point)
		distToLight := lightToHit.Length()
		lightDir := lightToHit.Normalize()

		if inShadow(hit, scene, lightDir, distToLight, ray) {
			continue
		}

		// Diffuse term
		diff := math.Max(0, hit.Normal.Dot(lightDir))
		diffuse := mat.Color.Mul(&light.Color).Scale(diff)

		// Specular term
		R := lightDir.Neg().Reflect(hit.Normal) // reflect light direction about normal
		spec := math.Max(0, R.Dot(V))
		specular := light.Color.Scale(mat.Reflectivity * math.Pow(spec, 50)) // 50 = fixed shininess

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
	for _, s := range scene.Spheres {
		if s == hit.Sphere {
			continue
		}
		t, ok := intersectSphere(s, shadowRay)
		if !ok {
			continue
		}
		// Check if the intersection is between the hit point and the light.
		if t*ray.Direction.Length() < distToLight {
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

func applyBeersLaw(color *Vec3, absorptionColor *Vec3, distance float64) *Vec3 {
	return &Vec3{
		X: color.X * math.Exp(-absorptionColor.X*distance),
		Y: color.Y * math.Exp(-absorptionColor.Y*distance),
		Z: color.Z * math.Exp(-absorptionColor.Z*distance),
	}
}

type Hit struct {
	Sphere *Sphere
	T      float64
	Point  *Vec3
	Normal *Vec3
}

// traceRay returns the color of the closest sphere hit by the ray, or nil
// if no sphere is hit.
func traceRay(scene *Scene, ray *Ray, depth int) *Vec3 {
	if depth <= 0 {
		// Recursion limit
		return &Vec3{}
	}

	minT := math.MaxFloat64
	var hitSphere *Sphere
	for _, sphere := range scene.Spheres {
		t, ok := intersectSphere(sphere, ray)
		if !ok {
			continue
		}
		if t < minT {
			minT = t
			hitSphere = sphere
		}
	}
	if hitSphere == nil {
		// Calculate background color (linear gradient).
		t := 0.5 * (ray.Direction.Y + 1.0)
		return scene.BgColorStart.LerpI(&scene.BgColorEnd, t)
	}
	hit := &Hit{
		Sphere: hitSphere,
		T:      minT,
		Point:  ray.Origin.Add(ray.Direction.Scale(minT)),
		Normal: nil,
	}
	hit.Normal = hit.Point.Sub(&hit.Sphere.Center).Normalize()

	surfaceColor := computeLighting(hit, scene, ray)

	mat := &hit.Sphere.Material
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
		reflectionRay := Ray{Origin: hit.Point.Add(hit.Normal.Scale(1e-4)), Direction: reflectedDir.Add(randomVector.Scale(fuzz)).Normalize()}
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

	// Distance from camera to screen. The viewport plane is at
	// Z=-CameraDistance with both Y and Z going from -0.5 to 0.5.
	CameraDistance float64

	Spheres []*Sphere
	Lights  []*Light

	// BgColorStart and BgColorEnd define the 2 ends of the gradient
	// background color.
	BgColorStart, BgColorEnd Vec3
}

func Render(scene *Scene) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, scene.WidthPx, scene.HeightPx))

	viewportWidth := 4.0
	viewportHeight := viewportWidth * (float64(scene.HeightPx) / float64(scene.WidthPx))

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

				origin := &Vec3{
					X: u,
					Y: -v,
					Z: -scene.CameraDistance,
				}
				// The ray vector goes from the origin to the screen pixel
				ray := Ray{
					Origin:    origin,
					Direction: origin.Normalize(),
				}

				const recursionLimit = 3
				color := traceRay(scene, &ray, recursionLimit)
				totalColor.AddI(color)
			}
			img.Set(x, y, totalColor.Scale(1.0/float64(numSamples)))
		}
	}
	return img
}
