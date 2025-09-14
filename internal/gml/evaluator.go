package gml

import (
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
)

type RenderArgs struct {
	AmbientLight *Point // The intensity of ambient light (a point)
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
	Env       map[string]Value
	Render    func(*RenderArgs)
	// Optional for debugging, can be nil
	Tracer func(string)
}

type Value interface {
	fmt.Stringer
	value()
}

type VInt int

func (VInt) value() {}

func (v VInt) String() string {
	return fmt.Sprintf("%d", int(v))
}

type VReal float64

func (VReal) value() {}

func (v VReal) String() string {
	return FormatFloat(float64(v))
}

type VBool bool

func (VBool) value() {}

func (v VBool) String() string {
	return strconv.FormatBool(bool(v))
}

type VString string

func (VString) value() {}

func (v VString) String() string {
	return strconv.Quote(string(v))
}

type VClosure struct {
	Code TokenList
	Env  map[string]Value
}

func (VClosure) value() {}

func formatMap[V fmt.Stringer](m map[string]V) string {
	var sb strings.Builder
	sb.WriteString("{")
	for k, v := range m {
		if sb.Len() > 1 {
			sb.WriteString(", ")
		}
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(v.String())
	}
	sb.WriteString("}")
	return sb.String()
}

func (v VClosure) String() string {
	return fmt.Sprintf("Closure(%v, env=%v)", v.Code, formatMap(v.Env))
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

func (VArray) value() {}

type Point struct {
	X, Y, Z VReal
}

func (Point) value() {}

func (p Point) String() string {
	return fmt.Sprintf("Point(%v, %v, %v)", p.X, p.Y, p.Z)
}

type SceneObject interface {
	Value

	// Translate produces a new SceneObject by applying the translation.
	Translate(x, y, z VReal) SceneObject
}

type Sphere struct {
	Center    Point
	Radius    VReal
	SurfaceFn VClosure
}

func (Sphere) value() {}

func (s Sphere) String() string {
	return fmt.Sprintf("Sphere(C: %v, R: %v)", s.Center, s.Radius)
}

func (s *Sphere) Translate(x, y, z VReal) SceneObject {
	return &Sphere{
		Center: Point{
			X: s.Center.X + x,
			Y: s.Center.Y + y,
			Z: s.Center.Z + z,
		},
		Radius:    s.Radius,
		SurfaceFn: s.SurfaceFn,
	}
}

type Union struct {
	Objects []SceneObject
}

func (Union) value() {}

func (u Union) String() string {
	return fmt.Sprintf("Union(%v)", u.Objects)
}

func (u *Union) Translate(x, y, z VReal) SceneObject {
	v := &Union{
		Objects: make([]SceneObject, len(u.Objects)),
	}
	for i := range u.Objects {
		v.Objects[i] = u.Objects[i].Translate(x, y, z)
	}
	return v
}

type PointLight struct {
	Position Point
	Color    Point // RGB
}

func (PointLight) value() {}

func (p PointLight) String() string {
	return fmt.Sprintf("PointLight(pos=%v, color=%v)", p.Position, p.Color)
}

func NewEvalState() *EvalState {
	return &EvalState{
		Env: make(map[string]Value),
	}
}

func (e *EvalState) tracef(format string, args ...any) {
	if e.Tracer != nil {
		e.Tracer(fmt.Sprintf(format, args...))
	}
}

var ErrEmptyStack = errors.New("empty stack")
var ErrUnboundIdentifier = errors.New("unbound identifier")

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
	if e.Tracer != nil {
		e.tracef("==============================\n")
		e.tracef("step: %v\nstack:\n", TokenGroupDebugString(token))
		for i, v := range e.Stack {
			e.tracef("  %d: %v\n", i, v)
		}
		e.tracef("env:\n")
		for k, v := range e.Env {
			e.tracef("  %s: %v\n", k, v)
		}
	}
	switch token := token.(type) {
	case *IntLiteral:
		e.push(VInt(token.Value))
	case *FloatLiteral:
		e.push(VReal(token.Value))
	case *BoolLiteral:
		e.push(VBool(token.Value))
	case *StringLiteral:
		e.push(VString(token.Value))
	case *Function:
		e.push(VClosure{Code: token.Body, Env: maps.Clone(e.Env)})
	case *Binder:
		v, err := e.pop()
		if err != nil {
			return err
		}
		e.Env[token.Name] = v
	case *Identifier:
		if b := builtins[token.Name]; b != nil {
			return b.Run(e)
		}
		// Else look up a variable in the environment.
		if val, ok := e.Env[token.Name]; ok {
			e.push(val)
		} else {
			return fmt.Errorf("%w: %s", ErrUnboundIdentifier, token.Name)
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

func (e *EvalState) push(value Value) {
	e.Stack = append(e.Stack, value)
}

func (e *EvalState) pop() (Value, error) {
	if len(e.Stack) == 0 {
		return nil, fmt.Errorf("%w: token: %v", ErrEmptyStack, TokenGroupDebugString(e.CurrToken))
	}
	val := e.Stack[len(e.Stack)-1]
	e.Stack = e.Stack[:len(e.Stack)-1]
	return val, nil
}

func popValue[T Value](e *EvalState) (T, error) {
	v, err := e.pop()
	if err != nil {
		return *new(T), err
	}
	derived, ok := v.(T)
	if !ok {
		zero := *new(T)
		return zero, fmt.Errorf("type mismatch (evaluating %s): expected %T, got %v (%T)", TokenGroupDebugString(e.CurrToken), zero, v, v)
	}
	return derived, nil
}

func pop3[T Value](e *EvalState) (T, T, T, error) {
	var x, y, z T
	var err error
	if z, err = popValue[T](e); err != nil {
		return x, y, z, err
	}
	if y, err = popValue[T](e); err != nil {
		return x, y, z, err
	}
	if x, err = popValue[T](e); err != nil {
		return x, y, z, err
	}
	return x, y, z, nil
}

type stateModifier = func(*EvalState) error

type Builtin struct {
	Name string
	Func func(*EvalState) error
}

var errNotImplemented = errors.New("not implemented")

func (b Builtin) Run(e *EvalState) error {
	if b.Func == nil {
		return fmt.Errorf("%w: %s", errNotImplemented, b.Name)
	}
	return b.Func(e)
}

var builtins map[string]*Builtin

func init() {
	builtins = map[string]*Builtin{}

	registerBuiltin := func(name string, f stateModifier) {
		builtins[name] = &Builtin{Name: name, Func: f}
	}

	registerBuiltin("addi", addi)
	registerBuiltin("apply", apply)
	registerBuiltin("point", nil)
	registerBuiltin("cube", nil)
	registerBuiltin("sphere", sphere)
	registerBuiltin("plane", nil)
	registerBuiltin("point", point)
	registerBuiltin("pointlight", pointlight)
	registerBuiltin("translate", translate)
	registerBuiltin("uscale", nil)
	registerBuiltin("rotatex", nil)
	registerBuiltin("rotatey", nil)
	registerBuiltin("rotatez", nil)
	registerBuiltin("union", union)
	registerBuiltin("render", render)
}

func addi(e *EvalState) error {
	a, err := popValue[VInt](e)
	if err != nil {
		return err
	}
	b, err := popValue[VInt](e)
	if err != nil {
		return err
	}
	e.push(a + b)
	return nil
}

func apply(e *EvalState) error {
	closure, err := popValue[VClosure](e)
	if err != nil {
		return err
	}
	oldEnv := e.Env
	defer func() { e.Env = oldEnv }()
	e.Env = closure.Env
	return e.Eval(closure.Code)
}

func point(e *EvalState) error {
	x, y, z, err := pop3[VReal](e)
	if err != nil {
		return err
	}
	e.push(Point{X: x, Y: y, Z: z})
	return nil
}

func pointlight(e *EvalState) error {
	// pos color pointlight
	color, err := popValue[Point](e)
	if err != nil {
		return err
	}
	pos, err := popValue[Point](e)
	if err != nil {
		return err
	}
	e.push(&PointLight{Position: pos, Color: color})
	return nil
}

// sphere creates a unit sphere at the origin
// with the surface function provided on the
// top of the stack.
func sphere(e *EvalState) error {
	surfaceFn, err := popValue[VClosure](e)
	if err != nil {
		return err
	}
	e.push(&Sphere{
		Center:    Point{X: 0, Y: 0, Z: 0},
		Radius:    1.0,
		SurfaceFn: surfaceFn,
	})
	return nil
}

func translate(e *EvalState) error {
	x, y, z, err := pop3[VReal](e)
	if err != nil {
		return err
	}
	s, err := popValue[SceneObject](e)
	if err != nil {
		return err
	}
	e.push(s.Translate(x, y, z))
	return nil
}

func union(e *EvalState) error {
	a, err := popValue[SceneObject](e)
	if err != nil {
		return err
	}
	b, err := popValue[SceneObject](e)
	if err != nil {
		return err
	}
	e.push(&Union{Objects: []SceneObject{a, b}})
	return nil
}

func render(e *EvalState) error {
	// Pop the values of RenderArgs, reverse order.
	// amb lights obj depth fov wid ht file render
	file, err := popValue[VString](e)
	if err != nil {
		return err
	}
	height, err := popValue[VInt](e)
	if err != nil {
		return err
	}
	width, err := popValue[VInt](e)
	if err != nil {
		return err
	}
	fov, err := popValue[VReal](e)
	if err != nil {
		return err
	}
	depth, err := popValue[VInt](e)
	if err != nil {
		return err
	}
	obj, err := popValue[SceneObject](e)
	if err != nil {
		return err
	}
	lights, err := popValue[VArray](e)
	if err != nil {
		return err
	}
	amb, err := popValue[Point](e)
	if err != nil {
		return err
	}
	// Lights should contain int values
	lightInts := make([]*PointLight, len(lights.Elements))
	for i, l := range lights.Elements {
		if l, ok := l.(*PointLight); ok {
			lightInts[i] = l
		} else {
			return fmt.Errorf("expected lights array to contain *PointLight, got %T", l)
		}
	}
	if e.Render == nil {
		return fmt.Errorf("render function not set")
	}
	e.Render(&RenderArgs{
		Width:        int(width),
		Height:       int(height),
		File:         string(file),
		Fov:          float64(fov),
		Depth:        int(depth),
		Scene:        obj,
		AmbientLight: &amb,
		Lights:       lightInts,
	})
	return nil

}
