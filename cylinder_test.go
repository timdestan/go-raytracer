package raytracer

import (
	"math"
	"testing"

	"github.com/timdestan/go-raytracer/internal/gml"
	"github.com/timdestan/go-raytracer/internal/prim"
)

func newIdentityCylinder() *Cylinder {
	identity := prim.IdentityMatrix()
	return &Cylinder{
		SurfaceFn:     gml.VSurfaceFn{Material: &gml.Material{}},
		ObjectToWorld: identity,
		WorldToObject: identity,
		NormalMat:     identity,
	}
}

func TestCylinderIntersectSide(t *testing.T) {
	c := newIdentityCylinder()
	ray := Ray{Origin: prim.Vec3{X: -2, Y: 0.5, Z: 0}, Direction: prim.Vec3{X: 1, Y: 0, Z: 0}}

	hit := c.Intersect(ray)
	if hit == nil {
		t.Fatal("expected a hit, got nil")
	}
	if hit.Face != CylinderSide {
		t.Errorf("Face = %d, want CylinderSide", hit.Face)
	}
	if math.Abs(hit.T-1.0) > 1e-9 {
		t.Errorf("T = %v, want 1.0", hit.T)
	}
	wantPoint := prim.Vec3{X: -1, Y: 0.5, Z: 0}
	if hit.PointObj.Sub(wantPoint).Length() > 1e-9 {
		t.Errorf("PointObj = %v, want %v", hit.PointObj, wantPoint)
	}
}

func TestCylinderIntersectTopCap(t *testing.T) {
	c := newIdentityCylinder()
	ray := Ray{Origin: prim.Vec3{X: 0, Y: 2, Z: 0}, Direction: prim.Vec3{X: 0, Y: -1, Z: 0}}

	hit := c.Intersect(ray)
	if hit == nil {
		t.Fatal("expected a hit, got nil")
	}
	if hit.Face != CylinderTop {
		t.Errorf("Face = %d, want CylinderTop", hit.Face)
	}
	if math.Abs(hit.T-1.0) > 1e-9 {
		t.Errorf("T = %v, want 1.0", hit.T)
	}
}

func TestCylinderIntersectBottomCap(t *testing.T) {
	c := newIdentityCylinder()
	ray := Ray{Origin: prim.Vec3{X: 0, Y: -2, Z: 0}, Direction: prim.Vec3{X: 0, Y: 1, Z: 0}}

	hit := c.Intersect(ray)
	if hit == nil {
		t.Fatal("expected a hit, got nil")
	}
	if hit.Face != CylinderBottom {
		t.Errorf("Face = %d, want CylinderBottom", hit.Face)
	}
	if math.Abs(hit.T-2.0) > 1e-9 {
		t.Errorf("T = %v, want 2.0", hit.T)
	}
}

func TestCylinderIntersectFromInsideHitsNearestCap(t *testing.T) {
	c := newIdentityCylinder()
	ray := Ray{Origin: prim.Vec3{X: 0, Y: 0.5, Z: 0}, Direction: prim.Vec3{X: 0, Y: 1, Z: 0}}

	hit := c.Intersect(ray)
	if hit == nil {
		t.Fatal("expected a hit, got nil")
	}
	if hit.Face != CylinderTop {
		t.Errorf("Face = %d, want CylinderTop", hit.Face)
	}
	if math.Abs(hit.T-0.5) > 1e-9 {
		t.Errorf("T = %v, want 0.5", hit.T)
	}
}

func TestCylinderIntersectMiss(t *testing.T) {
	c := newIdentityCylinder()
	// A ray well outside the radius, travelling parallel to the axis.
	ray := Ray{Origin: prim.Vec3{X: 5, Y: -1, Z: 0}, Direction: prim.Vec3{X: 0, Y: 1, Z: 0}}

	if hit := c.Intersect(ray); hit != nil {
		t.Errorf("expected no hit, got %+v", hit)
	}

	// A ray that would hit the infinite lateral surface, but outside [0, 1]
	// in height, and pointed away from both caps.
	missRay := Ray{Origin: prim.Vec3{X: -2, Y: 5, Z: 0}, Direction: prim.Vec3{X: 1, Y: 0, Z: 0}}
	if hit := c.Intersect(missRay); hit != nil {
		t.Errorf("expected no hit, got %+v", hit)
	}
}

func TestCylinderIntersectBehindRay(t *testing.T) {
	c := newIdentityCylinder()
	// Cylinder is entirely behind the ray origin.
	ray := Ray{Origin: prim.Vec3{X: 2, Y: 0.5, Z: 0}, Direction: prim.Vec3{X: 1, Y: 0, Z: 0}}

	if hit := c.Intersect(ray); hit != nil {
		t.Errorf("expected no hit, got %+v", hit)
	}
}

func TestCylinderComputeSurfacePropsSide(t *testing.T) {
	c := newIdentityCylinder()
	hit := Hit{
		Object:   c,
		Face:     CylinderSide,
		PointObj: prim.Vec3{X: 1, Y: 0.5, Z: 0},
	}

	hitEx, err := c.ComputeSurfaceProps(hit)
	if err != nil {
		t.Fatalf("ComputeSurfaceProps: %v", err)
	}
	wantNormal := prim.Vec3{X: 1, Y: 0, Z: 0}
	if hitEx.NormalWorld.Sub(wantNormal).Length() > 1e-9 {
		t.Errorf("NormalWorld = %v, want %v", hitEx.NormalWorld, wantNormal)
	}
}

func TestCylinderComputeSurfacePropsCaps(t *testing.T) {
	c := newIdentityCylinder()

	topHit := Hit{Object: c, Face: CylinderTop, PointObj: prim.Vec3{X: 0.2, Y: 1, Z: 0.3}}
	topEx, err := c.ComputeSurfaceProps(topHit)
	if err != nil {
		t.Fatalf("ComputeSurfaceProps(top): %v", err)
	}
	wantTopNormal := prim.Vec3{X: 0, Y: 1, Z: 0}
	if topEx.NormalWorld.Sub(wantTopNormal).Length() > 1e-9 {
		t.Errorf("top NormalWorld = %v, want %v", topEx.NormalWorld, wantTopNormal)
	}

	bottomHit := Hit{Object: c, Face: CylinderBottom, PointObj: prim.Vec3{X: 0.2, Y: 0, Z: 0.3}}
	bottomEx, err := c.ComputeSurfaceProps(bottomHit)
	if err != nil {
		t.Fatalf("ComputeSurfaceProps(bottom): %v", err)
	}
	wantBottomNormal := prim.Vec3{X: 0, Y: -1, Z: 0}
	if bottomEx.NormalWorld.Sub(wantBottomNormal).Length() > 1e-9 {
		t.Errorf("bottom NormalWorld = %v, want %v", bottomEx.NormalWorld, wantBottomNormal)
	}
}

func TestCylinderComputeSurfacePropsInvalidFace(t *testing.T) {
	c := newIdentityCylinder()
	hit := Hit{Object: c, Face: 99, PointObj: prim.Vec3{}}

	if _, err := c.ComputeSurfaceProps(hit); err == nil {
		t.Error("expected an error for an invalid face index, got nil")
	}
}
