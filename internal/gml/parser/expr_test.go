package parser

import (
	"testing"
)

func TestDebugString(t *testing.T) {
	type testCase struct {
		name string
		expr TokenGroup
		want string
	}
	for _, tt := range []testCase{
		{
			name: "int literal",
			expr: &IntLiteral{Value: 1},
			want: "1",
		},
		{
			name: "float literal",
			expr: &FloatLiteral{Value: 1.0},
			want: "1.0",
		},
		{
			name: "float literal more decimals",
			expr: &FloatLiteral{Value: 1.2343},
			want: "1.2343",
		},
		{
			name: "float literal large exponent",
			expr: &FloatLiteral{Value: 1e23},
			want: "1e+23",
		},
		{
			name: "string literal",
			expr: &StringLiteral{Value: "foo"},
			want: "\"foo\"",
		},
		{
			name: "array",
			expr: &Array{Elements: []TokenGroup{
				&IntLiteral{Value: 1},
				&IntLiteral{Value: 2},
				&Function{
					Body: []TokenGroup{
						&Identifier{Name: "foo"},
						&Binder{Name: "bar"},
					},
				},
				&Identifier{Name: "foo"},
				&Binder{Name: "bar"},
			}},
			want: "[ 1 2 { foo /bar } foo /bar ]",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := TokenGroupDebugString(tt.expr)
			if got != tt.want {
				t.Errorf("TokenGroupDebugString() = %v, want %v", got, tt.want)
			}
		})
	}
}
