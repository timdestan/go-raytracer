package gml

import (
	"fmt"
	"strconv"
	"strings"
)

type TokenGroupType int

const (
	TGIdentifier TokenGroupType = iota
	TGArray
	TGIntLiteral
	TGFloatLiteral
	TGBoolLiteral
	TGStringLiteral
	TGBinder
	TGFunction
)

type TokenList []TokenGroup

type TokenGroup interface {
	Type() TokenGroupType
}

type Identifier struct {
	Name string
}

func (i *Identifier) Type() TokenGroupType {
	return TGIdentifier
}

type Array struct {
	Elements TokenList
}

func (a *Array) Type() TokenGroupType {
	return TGArray
}

type IntLiteral struct {
	Value int64
}

func (i *IntLiteral) Type() TokenGroupType {
	return TGIntLiteral
}

type FloatLiteral struct {
	Value float64
}

func (f *FloatLiteral) Type() TokenGroupType {
	return TGFloatLiteral
}

type BoolLiteral struct {
	Value bool
}

func (b *BoolLiteral) Type() TokenGroupType {
	return TGBoolLiteral
}

type StringLiteral struct {
	Value string
}

func (s *StringLiteral) Type() TokenGroupType {
	return TGStringLiteral
}

type Binder struct {
	Name string
}

func (b *Binder) Type() TokenGroupType {
	return TGBinder
}

type Function struct {
	Body TokenList
}

func (f *Function) Type() TokenGroupType {
	return TGFunction
}

func TokenGroupDebugString(g TokenGroup) string {
	switch g := g.(type) {
	case *IntLiteral:
		return strconv.FormatInt(g.Value, 10)
	case *FloatLiteral:
		str := strconv.FormatFloat(g.Value, 'g', -1, 64)
		if strings.Contains(str, ".") || strings.ContainsAny(str, "eE") {
			return str
		}
		// Show trailing .0 even for integers to make it obvious the result is
		// a float.
		return str + ".0"
	case *BoolLiteral:
		return strconv.FormatBool(g.Value)
	case *StringLiteral:
		return strconv.Quote(g.Value)
	case *Identifier:
		return g.Name
	case *Binder:
		return "/" + g.Name
	case *Function:
		return "{ " + g.Body.String() + " }"
	case *Array:
		return "[ " + g.Elements.String() + " ]"
	default:
		// All TokenGroup types should be defined in this file.
		panic(fmt.Sprintf("unknown token group: %v", g))
	}
}

func (l TokenList) String() string {
	body := make([]string, len(l))
	for i, token := range l {
		body[i] = TokenGroupDebugString(token)
	}
	return strings.Join(body, " ")
}
