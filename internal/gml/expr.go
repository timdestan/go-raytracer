package gml

import (
	"fmt"
	"slices"
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

// Pos is a source position (1-based line and column).
type Pos struct {
	Line int
	Col  int
}

func (p Pos) String() string {
	if p.Line == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

// prefix returns "line:col: " for use at the start of error messages,
// or "" when the position is unknown (zero value).
func (p Pos) prefix() string {
	if p.Line == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d: ", p.Line, p.Col)
}

type TokenList []TokenGroup

type TokenGroup interface {
	Type() TokenGroupType
	Position() Pos
}

type Identifier struct {
	Name string
	// Id is a unique id for the symbol. Symbols with
	// the same names in different scopes will share
	// the same id.
	ID int
	Pos
}

func (i *Identifier) Type() TokenGroupType { return TGIdentifier }
func (i *Identifier) Position() Pos        { return i.Pos }

type Array struct {
	Elements TokenList
	Pos
}

func (a *Array) Type() TokenGroupType { return TGArray }
func (a *Array) Position() Pos        { return a.Pos }

type IntLiteral struct {
	Value int64
	Pos
}

func (i *IntLiteral) Type() TokenGroupType { return TGIntLiteral }
func (i *IntLiteral) Position() Pos        { return i.Pos }

type FloatLiteral struct {
	Value float64
	Pos
}

func (f *FloatLiteral) Type() TokenGroupType { return TGFloatLiteral }
func (f *FloatLiteral) Position() Pos        { return f.Pos }

type BoolLiteral struct {
	Value bool
	Pos
}

func (b *BoolLiteral) Type() TokenGroupType { return TGBoolLiteral }
func (b *BoolLiteral) Position() Pos        { return b.Pos }

type StringLiteral struct {
	Value string
	Pos
}

func (s *StringLiteral) Type() TokenGroupType { return TGStringLiteral }
func (s *StringLiteral) Position() Pos        { return s.Pos }

type Binder struct {
	Name string
	ID   int
	Pos
}

func (b *Binder) Type() TokenGroupType { return TGBinder }
func (b *Binder) Position() Pos        { return b.Pos }

type Function struct {
	Body TokenList
	Pos
}

func (f *Function) Type() TokenGroupType { return TGFunction }
func (f *Function) Position() Pos        { return f.Pos }

func FormatFloat(f float64) string {
	str := strconv.FormatFloat(f, 'g', -1, 64)
	if strings.ContainsAny(str, ".eE") {
		return str
	}
	// Show trailing .0 even for integers to make it obvious the result is
	// a float.
	return str + ".0"
}

func TokenGroupDebugString(g TokenGroup) string {
	switch g := g.(type) {
	case *IntLiteral:
		return strconv.FormatInt(g.Value, 10)
	case *FloatLiteral:
		return FormatFloat(g.Value)
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

func formatMap[V fmt.Stringer](m map[string]V) string {
	var sb strings.Builder
	sb.WriteString("{")
	var sortedKeys []string
	for k := range m {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)
	for _, k := range sortedKeys {
		if sb.Len() > 1 {
			sb.WriteString(", ")
		}
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(m[k].String())
	}
	sb.WriteString("}")
	return sb.String()
}
