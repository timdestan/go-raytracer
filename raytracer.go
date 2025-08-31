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

func (v *Vec3) Sub(other *Vec3) *Vec3 {
	return &Vec3{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

func (v *Vec3) Dot(other *Vec3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v *Vec3) CosineSimilarity(other *Vec3) float64 {
	return v.Dot(other) / (v.Length() * other.Length())
}

func (v *Vec3) Lerp(other *Vec3, t float64) *Vec3 {
	return &Vec3{
		X: v.X + (other.X-v.X)*t,
		Y: v.Y + (other.Y-v.Y)*t,
		Z: v.Z + (other.Z-v.Z)*t,
	}
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

// Clamp clamps the X, Y, and Z values between 0 and 1.
func (c *Vec3) Clamp() *Vec3 {
	return &Vec3{
		X: clamp(0, 1, c.X),
		Y: clamp(0, 1, c.Y),
		Z: clamp(0, 1, c.Z),
	}
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
	t1 := t_ca + t_hc
	if t1 > 0.0 {
		return t1, true
	}
	return 0.0, false
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
	n := normal

	if cosi > 0 { // we are inside the object, swap
		etai, etat = etat, etai
		n = n.Neg() // flip normal
	}

	// Compute R0
	r0 := (etai - etat) / (etai + etat)
	r0 = r0 * r0

	cost := math.Abs(cosi)
	return r0 + (1-r0)*math.Pow(1-cost, 5) // Schlick's approximation
}

// clamp limits x between min and max
func clamp(min, max, x float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

func applyBeersLaw(color *Vec3, absorptionColor *Vec3, distance float64) *Vec3 {
	return &Vec3{
		X: color.X * math.Exp(-absorptionColor.X*distance),
		Y: color.Y * math.Exp(-absorptionColor.Y*distance),
		Z: color.Z * math.Exp(-absorptionColor.Z*distance),
	}
}

// traceRay returns the color of the closest sphere hit by the ray, or nil
// if no sphere is hit.
func traceRay(opts *RenderOptions, ray *Ray, depth int) *Vec3 {
	if depth <= 0 {
		// Recursion limit
		return &Vec3{}
	}

	minT := math.MaxFloat64
	var hitSphereIndex = -1
	for i, sphere := range opts.Spheres {
		t, ok := intersectSphere(sphere, ray)
		if !ok {
			continue
		}
		if t < minT {
			minT = t
			hitSphereIndex = i
		}
	}
	if hitSphereIndex == -1 {
		// Calculate background color (linear gradient).
		t := 0.5 * (ray.Direction.Y + 1.0)
		return opts.BgColorStart.Lerp(&opts.BgColorEnd, t)
	}
	hitSphere := opts.Spheres[hitSphereIndex]
	hitPoint := ray.Origin.Add(ray.Direction.Scale(minT))
	normal := hitPoint.Sub(&hitSphere.Center).Normalize()

	// TODO: Handle multiple lights.
	if len(opts.Lights) != 1 {
		panic("expected exactly 1 light")
	}
	light := opts.Lights[0]
	lightDirection := light.Position.Sub(hitPoint).Normalize()

	// Check for shadows.
	shadowRay := &Ray{Origin: hitPoint, Direction: lightDirection}
	inShadow := false
	for i, s := range opts.Spheres {
		if i == hitSphereIndex {
			continue
		}
		t, ok := intersectSphere(s, shadowRay)
		if !ok {
			continue
		}
		// Check if the intersection is between the hit point and the light.
		if t*lightDirection.Length() < light.Position.Sub(hitPoint).Length() {
			inShadow = true
			break
		}
	}

	// Calculate diffuse intensity
	diffuseIntensity := math.Max(0, normal.Dot(lightDirection))
	ambientColor := hitSphere.Material.Color.Scale(0.1)
	surfaceColor := ambientColor.Add(hitSphere.Material.Color.Scale(diffuseIntensity).Scale(light.Color.X))

	if inShadow {
		surfaceColor = ambientColor
	}

	mat := &hitSphere.Material
	if mat.Reflectivity == 0 && mat.Transparency == 0 {
		return surfaceColor
	}

	// Handle reflection and transparency based on material properties
	reflectedColor := &Vec3{}
	if mat.Reflectivity > 0 {
		// For fuzzy reflections, add a random component to the reflection direction.
		fuzz := mat.Fuzziness
		reflectedDir := ray.Direction.Sub(normal.Scale(2.0 * ray.Direction.Dot(normal)))
		// "random" vector
		randomVector := Vec3{math.Cos(fuzz) * math.Cos(fuzz), math.Sin(fuzz) * math.Sin(fuzz), 0}
		reflectionRay := Ray{Origin: hitPoint.Add(normal.Scale(1e-4)), Direction: reflectedDir.Add(randomVector.Scale(fuzz)).Normalize()}
		reflectedColor = traceRay(opts, &reflectionRay, depth-1)
	}

	refractedColor := &Vec3{}
	if mat.Transparency > 0 {
		// This assumes the outer medium is air.
		n1 := 1.0
		n2 := mat.RefractiveIndex

		// If the dot product of the ray direction and the normal is positive,
		// then the ray is inside the object and trying to exit.
		// In this case, must swap the refractive indices.
		if ray.Direction.Dot(normal) > 0.0 {
			n1, n2 = n2, n1
			// We also need to invert the normal
			normal = normal.Scale(-1.0)
		}

		refractedDir := refract(ray.Direction, normal, n1, n2)

		if refractedDir != nil {
			// Create the refracted ray. We offset the origin slightly to avoid self-intersection.
			refractedRay := Ray{Origin: hitPoint.Sub(normal.Scale(1e-4)), Direction: refractedDir}

			// Recursively trace the refracted ray
			refractedColor = traceRay(opts, &refractedRay, depth-1)
		}
	}
	kr := fresnel(normal, ray.Direction, mat.RefractiveIndex)
	return surfaceColor.Scale(1.0 - mat.Transparency).Add(reflectedColor.Scale(kr).Add(refractedColor.Scale(1.0 - kr))).Clamp()
}

func square(x float64) float64 {
	return x * x
}

type RenderOptions struct {
	WidthPx, HeightPx int

	// Distance from camera to screen. The viewport plane is at
	// Z=CameraDistance with both Y and Z going from -0.5 to 0.5.
	CameraDistance float64

	Spheres []*Sphere
	Lights  []*Light

	// BgColorStart and BgColorEnd define the 2 ends of the gradient
	// background color.
	BgColorStart, BgColorEnd Vec3
}

func Render(opts *RenderOptions) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, opts.WidthPx, opts.HeightPx))

	viewportWidth := 4.0
	viewportHeight := viewportWidth * (float64(opts.HeightPx) / float64(opts.WidthPx))

	for x := range opts.WidthPx {
		for y := range opts.HeightPx {
			// Subsample for antialiasing
			totalColor := &Vec3{}
			const numSamples = 4
			for range numSamples {
				// Map pixel coordinates to world coordinates.
				du := rand.Float64() - 0.5
				dv := rand.Float64() - 0.5
				u := (float64(x)+du)/float64(opts.WidthPx-1)*viewportWidth - viewportWidth/2.0
				v := (float64(y)+dv)/float64(opts.HeightPx-1)*viewportHeight - viewportHeight/2.0

				origin := &Vec3{
					X: u,
					Y: -v,
					Z: -opts.CameraDistance,
				}
				// The ray vector goes from the origin to the screen pixel
				ray := Ray{
					Origin:    origin,
					Direction: origin.Normalize(),
				}

				const recursionLimit = 3
				color := traceRay(opts, &ray, recursionLimit)
				totalColor = totalColor.Add(color)
			}
			img.Set(x, y, totalColor.Scale(1.0/float64(numSamples)))
		}
	}
	return img
}
