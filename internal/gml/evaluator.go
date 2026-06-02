package gml

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/timdestan/go-raytracer/internal/prim"
)

type RenderArgs struct {
	AmbientLight *prim.Vec3 // The intensity of ambient light (a point)
	Lights       []*PointLight
	Scene        SceneObject
	Depth        int     // The recursion depth limit
	Fov          float64 // Degrees
	Width        int     // Pixels
	Height       int     // Pixels
	File         string
}

type EvalState struct {
	CurrToken TokenGroup // The token that is currently being evaluated
	Stack     []Value
	IDMapping IDMapping
	Env       Environment
	Render    func(*EvalState, *RenderArgs) error
	Debug     bool
}

type Value interface {
	fmt.Stringer
}

type VInt int

func (v VInt) String() string {
	return fmt.Sprintf("%d", int(v))
}

type VReal float64

func (v VReal) String() string {
	return FormatFloat(float64(v))
}

type VBool bool

func (v VBool) String() string {
	return strconv.FormatBool(bool(v))
}

type VString string

func (v VString) String() string {
	return strconv.Quote(string(v))
}

type VClosure struct {
	Code TokenList
	Env  Environment
}

func (v VClosure) String() string {
	return fmt.Sprintf("Closure(%v, env=%v)", v.Code, &v.Env)
}

type VArray struct {
	Elements []Value
}

func (a VArray) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range a.Elements {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(v.String())
	}
	sb.WriteString("]")
	return sb.String()
}

type SceneObject interface {
	Value

	Transform(mat *prim.Mat4) SceneObject
}

type Sphere struct {
	Center       prim.Vec3
	Radius       float64
	SurfaceFn    VClosure
	TransformMat *prim.Mat4
}

var _ SceneObject = (*Sphere)(nil)

func (s Sphere) String() string {
	return fmt.Sprintf("Sphere(C: %v, R: %v)", s.Center, s.Radius)
}

func (s *Sphere) Transform(mat *prim.Mat4) SceneObject {
	copy := *s
	if copy.TransformMat == nil {
		copy.TransformMat = mat
	} else {
		copy.TransformMat = mat.MulMat(copy.TransformMat)
	}
	return &copy
}

type Cube struct {
	Cube         prim.Cube
	SurfaceFn    VClosure
	TransformMat *prim.Mat4
}

var _ SceneObject = (*Cube)(nil)

func (c *Cube) String() string {
	return fmt.Sprintf("Cube(%v, %v)", c.Cube.MinPoint, c.Cube.MaxPoint)
}

func (c *Cube) Transform(mat *prim.Mat4) SceneObject {
	copy := *c
	if copy.TransformMat == nil {
		copy.TransformMat = mat
	} else {
		copy.TransformMat = mat.MulMat(copy.TransformMat)
	}
	return &copy
}

type Plane struct {
	Plane        prim.Plane
	SurfaceFn    VClosure
	TransformMat *prim.Mat4
}

var _ SceneObject = (*Plane)(nil)

func (p Plane) String() string {
	return p.Plane.String()
}

func (p *Plane) Transform(mat *prim.Mat4) SceneObject {
	copy := *p
	if copy.TransformMat == nil {
		copy.TransformMat = mat
	} else {
		copy.TransformMat = mat.MulMat(copy.TransformMat)
	}
	return &copy
}

type Union struct {
	Objects []SceneObject
}

var _ SceneObject = (*Union)(nil)

func (u Union) String() string {
	return fmt.Sprintf("Union(%v)", u.Objects)
}

func (u *Union) Transform(m *prim.Mat4) SceneObject {
	v := &Union{
		Objects: make([]SceneObject, len(u.Objects)),
	}
	for i := range u.Objects {
		v.Objects[i] = u.Objects[i].Transform(m)
	}
	return v
}

type PointLight struct {
	Position prim.Vec3
	Color    prim.Vec3 // RGB
}

func (p PointLight) String() string {
	return fmt.Sprintf("PointLight(pos=%v, color=%v)", p.Position, p.Color)
}

var (
	ErrEmptyStack            = errors.New("empty stack")
	ErrUnboundIdentifier     = errors.New("unbound identifier")
	ErrNotImplemented        = errors.New("not implemented")
	ErrArrayIndexOutOfBounds = errors.New("array index out of bounds")
)

func NewEvalState() *EvalState {
	return &EvalState{
		IDMapping: *NewIDMapping(),
		Env:       newEnv(),
	}
}

func (e *EvalState) ParseAndEval(input string) error {
	p := NewParserWithIDMapping(input, &e.IDMapping)
	program, err := p.Parse()
	if err != nil {
		return err
	}

	for _, token := range program {
		if err := e.evalOneStep(token); err != nil {
			return err
		}
	}
	return nil
}

func (e *EvalState) Eval(program TokenList) error {
	for _, token := range program {
		if err := e.evalOneStep(token); err != nil {
			return err
		}
	}
	return nil
}

func (e *EvalState) evalOneStep(token TokenGroup) error {
	e.CurrToken = token
	if e.Debug {
		fmt.Printf("==============================\n")
		fmt.Printf("step: %v\nstack:\n", TokenGroupDebugString(token))
		for i, v := range e.Stack {
			fmt.Printf("  %d: %v\n", i, v)
		}
		fmt.Printf("env: %v\n", e.Env)
	}
	switch token := token.(type) {
	case *IntLiteral:
		e.Push(VInt(token.Value))
	case *FloatLiteral:
		e.Push(VReal(token.Value))
	case *BoolLiteral:
		e.Push(VBool(token.Value))
	case *StringLiteral:
		e.Push(VString(token.Value))
	case *Function:
		e.Push(VClosure{Code: token.Body, Env: e.Env.Clone()})
	case *Binder:
		v, err := e.Pop()
		if err != nil {
			return err
		}
		e.Env.Store(token.ID, v)
	case *Identifier:
		if b := builtins[token.Name]; b != nil {
			return b.Run(e)
		}
		// Else look up a variable in the environment.
		if val := e.Env.Lookup(token.ID); val != nil {
			e.Push(val)
		} else {
			return fmt.Errorf("%s%w: %s", token.Pos.prefix(), ErrUnboundIdentifier, token.Name)
		}
	case *Array:
		oldStack := e.Stack
		defer func() { e.Stack = oldStack }()
		e.Stack = nil
		err := e.Eval(token.Elements)
		if err != nil {
			return err
		}
		oldStack = append(oldStack, VArray{Elements: e.Stack})
	default:
		return fmt.Errorf("unknown token: %v", token)
	}
	return nil
}

func (e *EvalState) Push(value Value) {
	e.Stack = append(e.Stack, value)
}

func (e *EvalState) Pop() (Value, error) {
	if len(e.Stack) == 0 {
		pos := e.CurrToken.Position()
		return nil, fmt.Errorf("%s%w: token: %v", pos.prefix(), ErrEmptyStack, TokenGroupDebugString(e.CurrToken))
	}
	i := len(e.Stack) - 1
	val := e.Stack[i]
	e.Stack[i] = nil
	e.Stack = e.Stack[:i]
	return val, nil
}

// EvalClosure evaluates the code in the given closure, then restores the old environment.
func (e *EvalState) EvalClosure(closure VClosure) error {
	// BUG: Because we do not make a copy of Env here,
	// restoring the old enviroment doesn't actually do
	// anything (changes made to the environment while
	// running the closure will still be present if
	// we run it again). I don't think this observable
	// with any of the current examples, but it is a
	// bug nonetheless.
	oldEnv := e.Env
	defer func() { e.Env = oldEnv }()
	e.Env = closure.Env
	return e.Eval(closure.Code)
}

func PopValue[T Value](e *EvalState) (T, error) {
	v, err := e.Pop()
	if err != nil {
		return *new(T), err
	}
	derived, ok := v.(T)
	if !ok {
		zero := *new(T)
		pos := e.CurrToken.Position()
		return zero, fmt.Errorf("%stype mismatch (evaluating %s): expected %T, got %v (%T)", pos.prefix(), TokenGroupDebugString(e.CurrToken), zero, v, v)
	}
	return derived, nil
}

func Pop2[T Value](e *EvalState) (T, T, error) {
	var x, y T
	var err error
	if y, err = PopValue[T](e); err != nil {
		return x, y, err
	}
	if x, err = PopValue[T](e); err != nil {
		return x, y, err
	}
	return x, y, nil
}

func Pop3[T Value](e *EvalState) (T, T, T, error) {
	var x, y, z T
	var err error
	if z, err = PopValue[T](e); err != nil {
		return x, y, z, err
	}
	if y, err = PopValue[T](e); err != nil {
		return x, y, z, err
	}
	if x, err = PopValue[T](e); err != nil {
		return x, y, z, err
	}
	return x, y, z, nil
}

type stateModifier = func(*EvalState) error

type Builtin struct {
	Name string
	Func func(*EvalState) error
}

func (b Builtin) Run(e *EvalState) error {
	if b.Func == nil {
		return fmt.Errorf("%w: %s", ErrNotImplemented, b.Name)
	}
	return b.Func(e)
}

var builtins map[string]*Builtin

func init() {
	builtins = map[string]*Builtin{}

	registerBuiltin := func(name string, f stateModifier) {
		builtins[name] = &Builtin{Name: name, Func: f}
	}

	registerBuiltin("addf", add[VReal])
	registerBuiltin("addi", add[VInt])
	registerBuiltin("apply", apply)
	registerBuiltin("cube", cube)
	registerBuiltin("if", if_)
	registerBuiltin("floor", floor)
	registerBuiltin("frac", frac)
	registerBuiltin("get", get)
	registerBuiltin("lessi", less[VInt])
	registerBuiltin("lessf", less[VReal])
	registerBuiltin("negi", neg[VInt])
	registerBuiltin("negf", neg[VReal])
	registerBuiltin("plane", plane)
	registerBuiltin("point", point)
	registerBuiltin("pointlight", pointlight)
	registerBuiltin("render", render)
	registerBuiltin("rotatex", rotatex)
	registerBuiltin("rotatey", rotatey)
	registerBuiltin("rotatez", rotatez)
	registerBuiltin("sphere", sphere)
	registerBuiltin("translate", translate)
	registerBuiltin("union", union)
	registerBuiltin("uscale", uscale)
}

type numericValue interface {
	~int | ~int64 | ~float64
	Value
}

func add[VType numericValue](e *EvalState) error {
	a, err := PopValue[VType](e)
	if err != nil {
		return err
	}
	b, err := PopValue[VType](e)
	if err != nil {
		return err
	}
	e.Push(a + b)
	return nil
}

func apply(e *EvalState) error {
	closure, err := PopValue[VClosure](e)
	if err != nil {
		return err
	}
	return e.EvalClosure(closure)
}

func point(e *EvalState) error {
	x, y, z, err := Pop3[VReal](e)
	if err != nil {
		return err
	}
	e.Push(&prim.Vec3{X: float64(x), Y: float64(y), Z: float64(z)})
	return nil
}

func pointlight(e *EvalState) error {
	// pos color pointlight
	color, err := PopValue[*prim.Vec3](e)
	if err != nil {
		return err
	}
	pos, err := PopValue[*prim.Vec3](e)
	if err != nil {
		return err
	}
	e.Push(&PointLight{Position: *pos, Color: *color})
	return nil
}

// sphere creates a unit sphere at the origin
// with the surface function provided on the
// top of the stack.
func sphere(e *EvalState) error {
	surfaceFn, err := PopValue[VClosure](e)
	if err != nil {
		return err
	}
	e.Push(&Sphere{
		Center:    prim.Vec3{X: 0, Y: 0, Z: 0},
		Radius:    1.0,
		SurfaceFn: surfaceFn,
	})
	return nil
}

// cube creates a cube with corners at (0, 0, 0)
// and (1, 1, 1).
func cube(e *EvalState) error {
	surfaceFn, err := PopValue[VClosure](e)
	if err != nil {
		return err
	}
	e.Push(&Cube{
		Cube: *prim.CubeFromCorners(
			&prim.Vec3{X: 0, Y: 0, Z: 0},
			&prim.Vec3{X: 1, Y: 1, Z: 1},
		),
		SurfaceFn: surfaceFn,
	})
	return nil
}

// plane creates the half space defined by the
// equation y <= 0.
func plane(e *EvalState) error {
	surfaceFn, err := PopValue[VClosure](e)
	if err != nil {
		return err
	}
	e.Push(&Plane{
		Plane: prim.Plane{
			Point:  prim.Vec3{}, // (0, 0, 0)
			Normal: prim.Vec3{X: 0, Y: -1, Z: 0},
		},
		SurfaceFn: surfaceFn,
	})
	return nil
}

func less[VType numericValue](e *EvalState) error {
	x, y, err := Pop2[VType](e)
	if err != nil {
		return err
	}
	e.Push(VBool(x < y))
	return nil
}

func neg[VType numericValue](e *EvalState) error {
	x, err := PopValue[VType](e)
	if err != nil {
		return err
	}
	e.Push(-x)
	return nil
}

func floor(e *EvalState) error {
	x, err := PopValue[VReal](e)
	if err != nil {
		return err
	}
	e.Push(VInt(math.Floor(float64(x))))
	return nil
}

// frac returns the fractional part of a real number.
func frac(e *EvalState) error {
	x, err := PopValue[VReal](e)
	if err != nil {
		return err
	}
	realPart := float64(int(x))
	e.Push(x - VReal(realPart))
	return nil
}

// arr i get get's the i'th (zero-based) element of arr
func get(e *EvalState) error {
	i, err := PopValue[VInt](e)
	if err != nil {
		return err
	}
	arr, err := PopValue[VArray](e)
	if err != nil {
		return err
	}
	n := len(arr.Elements)
	if i < 0 || int(i) >= n {
		// We could just allow the Go bounds checking on slice access
		// to fail but this seems more user-friendly.
		return fmt.Errorf("%w: %d vs %d", ErrArrayIndexOutOfBounds, i, n)
	}
	e.Push(arr.Elements[i])
	return nil
}

// if_ implements conditional evaluation.
//
// Example:
//
//	i 0.0 lessf { i negf 0.5 addf } { i } if
func if_(e *EvalState) error {
	trueClosure, falseClosure, err := Pop2[VClosure](e)
	if err != nil {
		return err
	}
	cond, err := PopValue[VBool](e)
	if err != nil {
		return err
	}
	closure := falseClosure
	if cond {
		closure = trueClosure
	}
	return e.EvalClosure(closure)
}

func translate(e *EvalState) error {
	x, y, z, err := Pop3[VReal](e)
	if err != nil {
		return err
	}
	s, err := PopValue[SceneObject](e)
	if err != nil {
		return err
	}
	e.Push(s.Transform(prim.Mat4Translate(prim.Vec3{
		X: float64(x),
		Y: float64(y),
		Z: float64(z),
	})))
	return nil
}

func uscale(e *EvalState) error {
	scale, err := PopValue[VReal](e)
	if err != nil {
		return err
	}
	s, err := PopValue[SceneObject](e)
	if err != nil {
		return err
	}
	e.Push(s.Transform(prim.Mat4Scale(float64(scale), float64(scale), float64(scale))))
	return nil
}

// TODO: Would it be better to represent the rotations as quaternions
// since we went to all the trouble of implementing them?
func rotate(e *EvalState, f func(angle float64) *prim.Mat4) error {
	angle, err := PopValue[VReal](e)
	if err != nil {
		return err
	}
	s, err := PopValue[SceneObject](e)
	if err != nil {
		return err
	}
	e.Push(s.Transform(f(float64(angle * math.Pi / 180))))
	return nil
}

func rotatex(e *EvalState) error {
	return rotate(e, prim.Mat4RotateX)
}

func rotatey(e *EvalState) error {
	return rotate(e, prim.Mat4RotateY)
}

func rotatez(e *EvalState) error {
	return rotate(e, prim.Mat4RotateZ)
}

func union(e *EvalState) error {
	a, err := PopValue[SceneObject](e)
	if err != nil {
		return err
	}
	b, err := PopValue[SceneObject](e)
	if err != nil {
		return err
	}
	e.Push(&Union{Objects: []SceneObject{a, b}})
	return nil
}

func render(e *EvalState) error {
	// Pop the values of RenderArgs, reverse order.
	// amb lights obj depth fov wid ht file render
	file, err := PopValue[VString](e)
	if err != nil {
		return err
	}
	height, err := PopValue[VInt](e)
	if err != nil {
		return err
	}
	width, err := PopValue[VInt](e)
	if err != nil {
		return err
	}
	fov, err := PopValue[VReal](e)
	if err != nil {
		return err
	}
	depth, err := PopValue[VInt](e)
	if err != nil {
		return err
	}
	obj, err := PopValue[SceneObject](e)
	if err != nil {
		return err
	}
	lights, err := PopValue[VArray](e)
	if err != nil {
		return err
	}
	amb, err := PopValue[*prim.Vec3](e)
	if err != nil {
		return err
	}
	lightValues := make([]*PointLight, len(lights.Elements))
	for i, l := range lights.Elements {
		if l, ok := l.(*PointLight); ok {
			lightValues[i] = l
		} else {
			return fmt.Errorf("expected lights array to contain *PointLight, got %T", l)
		}
	}
	if e.Render == nil {
		return fmt.Errorf("render function not set")
	}
	return e.Render(e, &RenderArgs{
		Width:        int(width),
		Height:       int(height),
		File:         string(file),
		Fov:          float64(fov),
		Depth:        int(depth),
		Scene:        obj,
		AmbientLight: amb,
		Lights:       lightValues,
	})
}
