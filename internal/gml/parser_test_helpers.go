package gml

import "fmt"

// Helpers for building parse tree expectations

func sym(name string) *Identifier {
	return &Identifier{Name: name}
}

func binder(name string) *Binder {
	return &Binder{Name: name}
}

func array(ts ...any) *Array {
	return &Array{Elements: tokens(ts...)}
}

func function(ts ...any) *Function {
	return &Function{Body: tokens(ts...)}
}

func tokens(tokens ...any) TokenList {
	l := make(TokenList, len(tokens))
	for i, token := range tokens {
		switch token := (token).(type) {
		case TokenGroup:
			l[i] = token
		case string:
			l[i] = &StringLiteral{Value: token}
		case int:
			l[i] = &IntLiteral{Value: int64(token)}
		case float64:
			l[i] = &FloatLiteral{Value: token}
		case bool:
			l[i] = &BoolLiteral{Value: token}
		case TokenList:
			panic(fmt.Sprintf("unexpected TokenList in tokens() (nested call?): %v", token))
		default:
			panic(fmt.Sprintf("unknown token arg in tokens(): %v", token))
		}
	}
	return l
}
