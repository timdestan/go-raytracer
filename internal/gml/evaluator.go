package gml

import (
	"errors"
	"fmt"
	"maps"
)

type EvalState struct {
	Stack  []Value
	Env    map[string]Value
	Tracer func(string)
}

type Value interface {
	value()
}

type VInt int

func (VInt) value() {}

type VReal float64

func (VReal) value() {}

type VBool bool

func (VBool) value() {}

type VString string

func (VString) value() {}

type VClosure struct {
	Code TokenList
	Env  map[string]Value
}

func (VClosure) value() {}

type VArray struct {
	Elements []Value
}

func (VArray) value() {}

// TODO: Point
// TODO: Object
// TODO: Light

func NewEvalState() *EvalState {
	return &EvalState{
		Env: make(map[string]Value),
	}
}

func (e *EvalState) trace(msg string) {
	if e.Tracer != nil {
		e.Tracer(msg)
	}
}

func (e *EvalState) tracef(msg string, args ...any) {
	if e.Tracer != nil {
		e.Tracer(fmt.Sprintf(msg, args...))
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

var ErrNotImplemented = errors.New("not implemented")

func (e *EvalState) evalOneStep(token TokenGroup) error {
	if e.Tracer != nil {
		defer func() { e.tracef("  token: %v, stack: %v, env: %v", TokenGroupDebugString(token), e.Stack, e.Env) }()
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
		if builtins[token.Name] != nil {
			return builtins[token.Name](e)
		}
		// Else look up a variable in the environment.
		if val, ok := e.Env[token.Name]; ok {
			e.push(val)
		} else {
			return fmt.Errorf("%w: %s", ErrUnboundIdentifier, token.Name)
		}
	case *Array:
		// e.push(VArray{Elements: token.Elements})
		return ErrNotImplemented
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
		return nil, ErrEmptyStack
	}
	val := e.Stack[len(e.Stack)-1]
	e.Stack = e.Stack[:len(e.Stack)-1]
	return val, nil
}

func popValue[T Value](e *EvalState) (*T, error) {
	v, err := e.pop()
	if err != nil {
		return nil, err
	}
	derived, ok := v.(T)
	if !ok {
		return &derived, fmt.Errorf("type mismatch: expected %T, got %v (%T)", derived, v, v)
	}
	return &derived, nil
}

type stateModifier = func(*EvalState) error

var builtins map[string]stateModifier

func init() {
	builtins = map[string]stateModifier{
		"addi":      addi,
		"apply":     apply,
		"point":     point,
		"cube":      cube,
		"sphere":    sphere,
		"plane":     plane,
		"translate": translate,
		"uscale":    uscale,
		"rotatex":   rotatex,
		"rotatey":   rotatey,
		"rotatez":   rotatez,
		"union":     union,
		"render":    render,
	}
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
	e.push(*a + *b)
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
	return nil
}

func cube(e *EvalState) error {
	return nil
}

func sphere(e *EvalState) error {
	return nil
}

func plane(e *EvalState) error {
	return nil
}

func translate(e *EvalState) error {
	return nil
}

func uscale(e *EvalState) error {
	return nil
}

func rotatex(e *EvalState) error {
	return nil
}

func rotatey(e *EvalState) error {
	return nil
}

func rotatez(e *EvalState) error {
	return nil
}

func union(e *EvalState) error {
	return nil
}

func render(e *EvalState) error {
	return nil
}
