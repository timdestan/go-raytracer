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

func (v *Vec3) Lerp(other Vec3, t float64) *Vec3 {
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

func (v *Vec3) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func (v *Vec3) RGBA() (r, g, b, a uint32) {
	const max = 0xffff
	return uint32(v.X * max), uint32(v.Y * max), uint32(v.Z * max), max
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
		// Calculate background color.
		t := 0.5 * (ray.Direction.Y + 1.0)
		return opts.BgColorStart.Lerp(opts.BgColorEnd, t)
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

	shadedColor := ambientColor.Add(hitSphere.Material.Color.Scale(diffuseIntensity).Scale(light.Color.X))

	if inShadow {
		shadedColor = ambientColor
	}

	// Handle reflection and transparency based on material properties
	if hitSphere.Material.Reflectivity > 0 {
		// For fuzzy reflections, add a random component to the reflection direction.
		fuzz := hitSphere.Material.Fuzziness
		reflectedDir := ray.Direction.Sub(normal.Scale(2.0 * ray.Direction.Dot(normal)))
		// "random" vector
		randomVector := Vec3{math.Cos(fuzz) * math.Cos(fuzz), math.Sin(fuzz) * math.Sin(fuzz), 0}
		reflectionRay := Ray{Origin: hitPoint.Add(normal.Scale(0.001)), Direction: reflectedDir.Add(randomVector.Scale(fuzz)).Normalize()}
		reflectedColor := traceRay(opts, &reflectionRay, depth-1)
		return reflectedColor.Scale(hitSphere.Material.Reflectivity).Add(shadedColor.Scale(1.0 - hitSphere.Material.Reflectivity))
	}
	return shadedColor
}

func square(x float64) float64 {
	return x * x
}

type RenderOptions struct {
	WidthPx, HeightPx int

	// CameraPosition is the position behind the screen of the point
	// from which all the rays originate. Defaults to (0,0,0).
	CameraPosition Vec3

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
				u := (float64(x)+rand.Float64())/float64(opts.WidthPx-1)*viewportWidth - viewportWidth/2.0
				v := (float64(y)+rand.Float64())/float64(opts.HeightPx-1)*viewportHeight - viewportHeight/2.0

				origin := &Vec3{
					X: u,
					Y: -v,
					Z: -opts.CameraDistance,
				}
				// The ray vector goes from the origin to the screen pixel
				ray := Ray{
					Origin:    origin,
					Direction: origin.Sub(&opts.CameraPosition).Normalize(),
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
