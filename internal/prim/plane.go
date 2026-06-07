package prim

import "fmt"

type Plane struct {
	Point  Vec3 // point on the plane
	Normal Vec3
}

func (p *Plane) String() string {
	return fmt.Sprintf("Pt: %v, Normal: %v", p.Point, p.Normal)
}

type CubeSide int

// These indexes are used directly in the surface function
// calls for cubes.
const (
	CubeFront CubeSide = iota
	CubeBack
	CubeLeft
	CubeRight
	CubeTop
	CubeBottom
)

const NUM_CUBE_SIDES CubeSide = 6

func PlanesForUnitCube() [NUM_CUBE_SIDES]Plane {
	return [NUM_CUBE_SIDES]Plane{
		CubeFront:  {Point: Vec3{Z: 0}, Normal: Vec3{Z: -1}},
		CubeBack:   {Point: Vec3{Z: 1}, Normal: Vec3{Z: +1}},
		CubeLeft:   {Point: Vec3{X: 0}, Normal: Vec3{X: -1}},
		CubeRight:  {Point: Vec3{X: 1}, Normal: Vec3{X: +1}},
		CubeTop:    {Point: Vec3{Y: 1}, Normal: Vec3{Y: +1}},
		CubeBottom: {Point: Vec3{Y: 0}, Normal: Vec3{Y: -1}},
	}
}
