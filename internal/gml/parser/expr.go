package parser

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
	Value string
}

func (i *Identifier) Type() TokenGroupType {
	return TGIdentifier
}

type Array struct {
	Elements []TokenGroup
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

// TODO: Binders should probably not be treated specially.

type Function struct {
	Binders []*Binder
	Body    TokenList
}

func (f *Function) Type() TokenGroupType {
	return TGFunction
}
