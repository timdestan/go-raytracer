package prim

import "fmt"

type Plane struct {
	Point  Vec3 // point on the plane
	Normal Vec3
}

func (p *Plane) String() string {
	return fmt.Sprintf("Pt: %v, Normal: %v", p.Point, p.Normal)
}

type Cube struct {
	MinPoint, MaxPoint Vec3
	Rotation           Vec4
}

func (c *Cube) Rotate(axis *Vec3, angle float64) *Cube {
	return &Cube{
		MinPoint: c.MinPoint,
		MaxPoint: c.MaxPoint,
		Rotation: *c.Rotation.Rotate(axis, angle),
	}
}

func (c *Cube) String() string {
	return fmt.Sprintf("Cube(%v, %v, Rotation: %v)", c.MinPoint, c.MaxPoint, c.Rotation)
}

// CubeFromCorners returns a cube defined by the given corners.
// One corner *must* have coordinates strictly larger than the other in all
// dimensions.
func CubeFromCorners(corner1, corner2 *Vec3) *Cube {
	if corner1.X > corner2.X {
		corner1, corner2 = corner2, corner1
	}
	if corner1.X >= corner2.X || corner1.Y >= corner2.Y || corner1.Z >= corner2.Z {
		panic("invalid corners: " + fmt.Sprintf("%v %v", corner1, corner2))
	}
	return &Cube{
		MinPoint: *corner1,
		MaxPoint: *corner2,
		Rotation: *QIdentity(),
	}
}
