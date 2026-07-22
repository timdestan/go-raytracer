package gml

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Binding struct {
	ID    int
	Value Value
}

func (b *Binding) String() string {
	return fmt.Sprintf("%d: %v", b.ID, b.Value)
}

func (b *Binding) DebugStringCtx(ctx DebugStringContext) string {
	symbol, ok := ctx.idMapping.IDNameMap[b.ID]
	if !ok {
		symbol = fmt.Sprintf("%d (?)", b.ID)
	}
	return fmt.Sprintf("%s: %s", symbol, DebugString(b.Value, ctx))
}

type Environment struct {
	bindings map[int]Value
	// TODO: Should environment just have a pointer to the IDMapping? This
	// would simplify the need to pass it around when constructing the debug
	// string.
}

func newEnv() Environment {
	return Environment{
		bindings: make(map[int]Value),
	}
}

// Bindings returns a slice of the bindings that
// have been set in the environment.
func (env *Environment) Bindings() []Binding {
	bs := make([]Binding, 0, len(env.bindings))
	for id, v := range env.bindings {
		bs = append(bs, Binding{id, v})
	}
	slices.SortFunc(bs, func(a, b Binding) int { return a.ID - b.ID })
	return bs
}

func (env *Environment) Clone() Environment {
	return Environment{
		bindings: maps.Clone(env.bindings),
	}
}

func (env *Environment) String() string {
	return env.mkString(func(b Binding) string {
		return b.String()
	})
}

func (env *Environment) DebugStringCtx(ctx DebugStringContext) string {
	return env.mkString(func(b Binding) string {
		return b.DebugStringCtx(ctx)
	})
}

func (env *Environment) mkString(fmtBinding func(Binding) string) string {
	var sb strings.Builder
	sb.WriteByte('{')
	for i, b := range env.Bindings() {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmtBinding(b))
	}
	sb.WriteByte('}')
	return sb.String()
}

func (env *Environment) Store(id int, value Value) {
	env.bindings[id] = value
}

func (env *Environment) Lookup(id int) Value {
	return env.bindings[id]
}

type IDMapping struct {
	NameIDMap map[string]int
	IDNameMap map[int]string
	MaxId     int
}

func NewIDMapping() *IDMapping {
	return &IDMapping{
		NameIDMap: make(map[string]int),
		IDNameMap: make(map[int]string),
	}
}

func (f *IDMapping) GetOrCreateId(name string) int {
	if x, ok := f.NameIDMap[name]; ok {
		return x
	}
	f.MaxId++
	newId := f.MaxId
	f.NameIDMap[name] = newId
	f.IDNameMap[newId] = name
	return newId
}

func (m *IDMapping) Clone() *IDMapping {
	return &IDMapping{
		NameIDMap: maps.Clone(m.NameIDMap),
		IDNameMap: maps.Clone(m.IDNameMap),
		MaxId:     m.MaxId,
	}
}

type DebugStringContext struct {
	idMapping *IDMapping
}

type DebugStringer interface {
	DebugStringCtx(ctx DebugStringContext) string
}

func DebugString(x any, ctx DebugStringContext) string {
	switch v := x.(type) {
	case DebugStringer:
		return v.DebugStringCtx(ctx)
	default:
		return fmt.Sprintf("%v", x)
	}
}
