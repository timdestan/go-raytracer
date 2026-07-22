package gml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTestFile(%s): %v", name, err)
	}
	return path
}

func readAllTokensFrom(l *Lexer) []LexerToken {
	var tokens []LexerToken
	for {
		tk := l.NextToken()
		tokens = append(tokens, tk)
		if tk.Type == TokenEOF || tk.Type == TokenError {
			break
		}
	}
	return tokens
}

func findToken(tokens []LexerToken, typ LexemeType, literal string) (LexerToken, bool) {
	for _, tk := range tokens {
		if tk.Type == typ && tk.Literal == literal {
			return tk, true
		}
	}
	return LexerToken{}, false
}

func TestLexIncludeBasic(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "lib.gml", "1 2\n")
	mainPath := writeTestFile(t, dir, "main.gml", "foo\n#include \"lib.gml\"\nbar\n")

	l, err := NewFileLexer(mainPath)
	if err != nil {
		t.Fatalf("NewFileLexer: %v", err)
	}
	tokens := readAllTokensFrom(l)

	want := []struct {
		typ     LexemeType
		literal string
		line    int
		col     int
	}{
		{TokenIdent, "foo", 1, 1},
		{TokenInt, "1", 1, 1}, // from lib.gml, independently numbered from line 1
		{TokenInt, "2", 1, 3},
		{TokenIdent, "bar", 3, 1}, // back in main.gml, its own true physical line
		{TokenEOF, "", 4, 1},
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(tokens), len(want), tokens)
	}
	for i, w := range want {
		got := tokens[i]
		if got.Type != w.typ || got.Literal != w.literal || got.Line != w.line || got.Col != w.col {
			t.Errorf("token %d: got %+v, want {%v %q %d %d}", i, got, w.typ, w.literal, w.line, w.col)
		}
	}
}

func TestLexIncludeMissingFile(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeTestFile(t, dir, "main.gml", `#include "nope.gml"`)

	l, err := NewFileLexer(mainPath)
	if err != nil {
		t.Fatalf("NewFileLexer: %v", err)
	}
	tk := l.NextToken()
	if tk.Type != TokenError {
		t.Fatalf("got token type %v, want TokenError: %+v", tk.Type, tk)
	}
	if !strings.Contains(tk.Literal, "nope.gml") {
		t.Errorf("error message %q does not mention missing file", tk.Literal)
	}
}

func TestLexIncludeCycle(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.gml", `#include "b.gml"`)
	bPath := writeTestFile(t, dir, "b.gml", `#include "a.gml"`)

	l, err := NewFileLexer(bPath)
	if err != nil {
		t.Fatalf("NewFileLexer: %v", err)
	}
	tk := l.NextToken()
	if tk.Type != TokenError {
		t.Fatalf("got token type %v, want TokenError: %+v", tk.Type, tk)
	}
	if !strings.Contains(tk.Literal, "cycle") {
		t.Errorf("error message %q does not mention a cycle", tk.Literal)
	}
}

func TestLexBlockComment(t *testing.T) {
	input := "foo /* comment\nspanning\nlines */ bar"
	tokens := readAllTokensFrom(NewLexer(input))

	fooTok, ok := findToken(tokens, TokenIdent, "foo")
	if !ok || fooTok.Line != 1 {
		t.Errorf("foo token: got %+v", fooTok)
	}
	barTok, ok := findToken(tokens, TokenIdent, "bar")
	if !ok || barTok.Line != 3 {
		t.Errorf("bar token: got %+v, want line 3", barTok)
	}
	for _, tk := range tokens {
		if tk.Type == TokenIllegal || tk.Type == TokenError {
			t.Errorf("unexpected %v token in output: %+v", tk.Type, tk)
		}
	}
}

func TestLexIncludeGuardDiamond(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "lib.gml", "#ifndef _LIB_\n#define _LIB_\n42\n#endif\n")
	writeTestFile(t, dir, "mid.gml", `#include "lib.gml"`+"\n")
	mainPath := writeTestFile(t, dir, "main.gml", "#include \"lib.gml\"\n#include \"mid.gml\"\n")

	l, err := NewFileLexer(mainPath)
	if err != nil {
		t.Fatalf("NewFileLexer: %v", err)
	}
	tokens := readAllTokensFrom(l)

	count := 0
	for _, tk := range tokens {
		if tk.Type == TokenInt && tk.Literal == "42" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("got %d occurrences of the guarded content, want exactly 1 (guard should suppress the second #include): %+v", count, tokens)
	}
}

func TestLexUnmatchedEndif(t *testing.T) {
	tk := NewLexer("#endif").NextToken()
	if tk.Type != TokenError {
		t.Fatalf("got token type %v, want TokenError: %+v", tk.Type, tk)
	}
}

func TestLexUnknownDirective(t *testing.T) {
	tk := NewLexer("#foo").NextToken()
	if tk.Type != TokenError {
		t.Fatalf("got token type %v, want TokenError: %+v", tk.Type, tk)
	}
}

// realFixturesUsingInclude are testdata/*.gml files known to use #include,
// carried over from the original GML contest test suite. Before #include
// support existed, none of these could be parsed.
var realFixturesUsingInclude = []string{
	"spheres.gml", "cube2.gml", "pipe.gml", "cone-fractal.gml", "fractal.gml",
	"large.gml", "cone.gml", "cylinder.gml", "rotate.gml", "intercyl.gml",
	"house.gml", "holes.gml", "ellipsoid.gml", "fov.gml",
}

func TestParseRealIncludeFixtures(t *testing.T) {
	for _, name := range realFixturesUsingInclude {
		t.Run(name, func(t *testing.T) {
			p, err := NewParserFromFile(filepath.Join("testdata", name))
			if err != nil {
				t.Fatalf("NewParserFromFile: %v", err)
			}
			if _, err := p.Parse(); err != nil {
				t.Fatalf("Parse: %v", err)
			}
		})
	}
}
