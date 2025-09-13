package gml

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func readAllTokens(input string) []LexerToken {
	l := NewLexer(input)
	var tokens []LexerToken
	for {
		tk := l.NextToken()
		tokens = append(tokens, tk)
		if tk.Type == TokenEOF {
			break
		}
	}
	return tokens
}

func TestLexEmptyString(t *testing.T) {
	input := ""
	want := []LexerToken{{Type: TokenEOF, Literal: ""}}
	got := readAllTokens(input)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("token mismatch (-got +want):\n%s", diff)
	}
}

func TestLexScientificNotation(t *testing.T) {
	for _, input := range []string{
		"1e-3",
		"1e+3",
		"1.0e-4",
		"1.0e+53",
	} {
		want := []LexerToken{
			{Type: TokenFloat, Literal: input},
			{Type: TokenEOF, Literal: ""},
		}
		got := readAllTokens(input)
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("token mismatch (-got +want):\n%s", diff)
		}
	}
}

func TestIllegalStringEscape(t *testing.T) {
	input := `"\a"`
	want := []LexerToken{
		{Type: TokenIllegal, Literal: `\a`},
		{Type: TokenEOF, Literal: ""},
	}

	got := readAllTokens(input)

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("token mismatch (-got +want):\n%s", diff)
	}
}

func TestLexExamples(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []LexerToken
	}{
		{
			name:  "Sphere",
			input: TestdataSphere,
			want: []LexerToken{
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenBinder, Literal: "/v"},
				{Type: TokenBinder, Literal: "/u"},
				{Type: TokenBinder, Literal: "/face"},
				{Type: TokenFloat, Literal: "0.8"},
				{Type: TokenFloat, Literal: "0.2"},
				{Type: TokenIdent, Literal: "v"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenFloat, Literal: "0.2"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenIdent, Literal: "sphere"},
				{Type: TokenBinder, Literal: "/s"},
				{Type: TokenIdent, Literal: "s"},
				{Type: TokenFloat, Literal: "-1.2"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenFloat, Literal: "3.0"},
				{Type: TokenIdent, Literal: "translate"},
				{Type: TokenIdent, Literal: "s"},
				{Type: TokenFloat, Literal: "1.2"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenFloat, Literal: "3.0"},
				{Type: TokenIdent, Literal: "translate"},
				{Type: TokenIdent, Literal: "union"},
				{Type: TokenBinder, Literal: "/scene"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenInt, Literal: "1"},
				{Type: TokenRBracket, Literal: "]"},
				{Type: TokenIdent, Literal: "scene"},
				{Type: TokenInt, Literal: "4"},
				{Type: TokenFloat, Literal: "90.0"},
				{Type: TokenInt, Literal: "320"},
				{Type: TokenInt, Literal: "240"},
				{Type: TokenString, Literal: "sphere.ppm"},
				{Type: TokenIdent, Literal: "render"},
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenRBracket, Literal: "]"},
				{Type: TokenBinder, Literal: "/ident"},
				{Type: TokenBoolean, Literal: "true"},
				{Type: TokenBoolean, Literal: "false"},
				{Type: TokenInt, Literal: "123"},
				{Type: TokenFloat, Literal: "1.23"},
				{Type: TokenString, Literal: "hello"},
				{Type: TokenEOF, Literal: ""},
			},
		},
		{
			name:  "Cube",
			input: TestdataCube,
			want: []LexerToken{
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenBinder, Literal: "/v"},
				{Type: TokenBinder, Literal: "/u"},
				{Type: TokenBinder, Literal: "/face"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenIdent, Literal: "cube"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenFloat, Literal: "-0.5"},
				{Type: TokenFloat, Literal: "4.0"},
				{Type: TokenIdent, Literal: "translate"},
				{Type: TokenFloat, Literal: "2.0"},
				{Type: TokenIdent, Literal: "uscale"},
				{Type: TokenFloat, Literal: "45.0"},
				{Type: TokenIdent, Literal: "rotatex"},
				{Type: TokenFloat, Literal: "135.0"},
				{Type: TokenIdent, Literal: "rotatey"},
				{Type: TokenBinder, Literal: "/c"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenBinder, Literal: "/white"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenBinder, Literal: "/blue"},
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenIdent, Literal: "blue"},
				{Type: TokenIdent, Literal: "white"},
				{Type: TokenRBracket, Literal: "]"},
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenIdent, Literal: "white"},
				{Type: TokenIdent, Literal: "blue"},
				{Type: TokenRBracket, Literal: "]"},
				{Type: TokenRBracket, Literal: "]"},
				{Type: TokenBinder, Literal: "/texture"},
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenBinder, Literal: "/i"},
				{Type: TokenIdent, Literal: "i"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenIdent, Literal: "lessf"},
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenIdent, Literal: "i"},
				{Type: TokenIdent, Literal: "negf"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenIdent, Literal: "addf"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenIdent, Literal: "i"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenIdent, Literal: "if"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenBinder, Literal: "/fabs"},
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenIdent, Literal: "fabs"},
				{Type: TokenIdent, Literal: "apply"},
				{Type: TokenBinder, Literal: "/v"},
				{Type: TokenIdent, Literal: "fabs"},
				{Type: TokenIdent, Literal: "apply"},
				{Type: TokenBinder, Literal: "/u"},
				{Type: TokenBinder, Literal: "/face"},
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenIdent, Literal: "frac"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenIdent, Literal: "addf"},
				{Type: TokenIdent, Literal: "floor"},
				{Type: TokenBinder, Literal: "/i"},
				{Type: TokenIdent, Literal: "i"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenBinder, Literal: "/toIntCoord"},
				{Type: TokenIdent, Literal: "texture"},
				{Type: TokenIdent, Literal: "u"},
				{Type: TokenIdent, Literal: "toIntCoord"},
				{Type: TokenIdent, Literal: "apply"},
				{Type: TokenIdent, Literal: "get"},
				{Type: TokenIdent, Literal: "v"},
				{Type: TokenIdent, Literal: "toIntCoord"},
				{Type: TokenIdent, Literal: "apply"},
				{Type: TokenIdent, Literal: "get"},
				{Type: TokenFloat, Literal: "0.3"},
				{Type: TokenFloat, Literal: "0.9"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenIdent, Literal: "plane"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenFloat, Literal: "-3.0"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenIdent, Literal: "translate"},
				{Type: TokenBinder, Literal: "/p"},
				{Type: TokenLCurly, Literal: "{"},
				{Type: TokenBinder, Literal: "/v"},
				{Type: TokenBinder, Literal: "/u"},
				{Type: TokenBinder, Literal: "/face"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenFloat, Literal: "0.5"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenFloat, Literal: "0.3"},
				{Type: TokenFloat, Literal: "0.85"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenRCurly, Literal: "}"},
				{Type: TokenIdent, Literal: "plane"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenFloat, Literal: "0.0"},
				{Type: TokenFloat, Literal: "8.0"},
				{Type: TokenIdent, Literal: "translate"},
				{Type: TokenFloat, Literal: "270.0"},
				{Type: TokenIdent, Literal: "rotatex"},
				{Type: TokenFloat, Literal: "45.0"},
				{Type: TokenIdent, Literal: "rotatez"},
				{Type: TokenBinder, Literal: "/p2"},
				{Type: TokenIdent, Literal: "c"},
				{Type: TokenIdent, Literal: "p"},
				{Type: TokenIdent, Literal: "union"},
				{Type: TokenIdent, Literal: "p2"},
				{Type: TokenIdent, Literal: "union"},
				{Type: TokenBinder, Literal: "/scene"},
				{Type: TokenInt, Literal: "-10"},
				{Type: TokenInt, Literal: "10"},
				{Type: TokenInt, Literal: "0"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenFloat, Literal: "1.0"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenIdent, Literal: "pointlight"},
				{Type: TokenBinder, Literal: "/l"},
				{Type: TokenFloat, Literal: "0.2"},
				{Type: TokenFloat, Literal: "0.2"},
				{Type: TokenFloat, Literal: "0.2"},
				{Type: TokenIdent, Literal: "point"},
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenIdent, Literal: "l"},
				{Type: TokenRBracket, Literal: "]"},
				{Type: TokenIdent, Literal: "scene"},
				{Type: TokenInt, Literal: "7"},
				{Type: TokenFloat, Literal: "90.0"},
				{Type: TokenInt, Literal: "480"},
				{Type: TokenInt, Literal: "320"},
				{Type: TokenString, Literal: "cube.ppm"},
				{Type: TokenIdent, Literal: "render"},
				{Type: TokenEOF, Literal: ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readAllTokens(tt.input)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("token mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
