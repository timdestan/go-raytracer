package gml

// TODO: The error handling / reporting is not great (or existing at all).
//
// We avoid the name "Token" in some of the types here because this also refers
// to some of the types in the BNF grammar for the parser and overloading it
// to refer to the lexer tokens can be confusing.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LexemeType int

const (
	TokenUnknown LexemeType = iota
	TokenEOF
	TokenIllegal
	// TokenError signals a lexer/preprocessor error (e.g. a bad #include)
	// whose Literal holds a human-readable message, as opposed to
	// TokenIllegal, whose Literal is raw partial lexical text.
	TokenError
	TokenIdent
	TokenBinder
	TokenBoolean
	TokenInt
	TokenFloat
	TokenString
	TokenLCurly
	TokenRCurly
	TokenLBracket
	TokenRBracket
)

var lexemeNames = [...]string{
	TokenUnknown:  "Unknown",
	TokenEOF:      "EOF",
	TokenIllegal:  "Illegal",
	TokenError:    "Error",
	TokenIdent:    "Ident",
	TokenBinder:   "Binder",
	TokenBoolean:  "Boolean",
	TokenInt:      "Integer",
	TokenFloat:    "Float",
	TokenString:   "String",
	TokenLCurly:   "LCurly",
	TokenRCurly:   "RCurly",
	TokenLBracket: "LBracket",
	TokenRBracket: "RBracket",
}

func (t LexemeType) String() string {
	return lexemeNames[t]
}

type LexerToken struct {
	Type    LexemeType
	Literal string
	Line    int
	Col     int
}

// lexerFrame holds the position-tracking state for a single source: either
// the raw string passed to NewLexer, or one file in a #include chain.
type lexerFrame struct {
	input   string
	pos     int
	readPos int
	ch      byte
	line    int
	col     int
	// file is the absolute path of the file that produced input, used to
	// resolve #include directives relative to it. Empty for a raw-string
	// frame with no file context (e.g. NewLexer input, or the REPL).
	file string
}

type Lexer struct {
	lexerFrame
	// stack holds suspended parent frames while lexing an #include chain.
	stack []lexerFrame
	// active holds the absolute paths of files currently open (this frame
	// plus all its ancestors), used to detect #include cycles.
	active map[string]bool
	// defined holds names set by #define, used to evaluate #ifndef guards.
	defined map[string]bool
	// condDepth counts #ifndef blocks that are currently open (condition
	// was true) and awaiting a matching #endif.
	condDepth int
}

func NewLexer(input string) *Lexer {
	l := &Lexer{lexerFrame: lexerFrame{input: input, line: 1}}
	l.readChar()
	return l
}

// NewFileLexer creates a Lexer reading from the file at path, so that any
// #include directives it contains resolve relative to path's directory.
func NewFileLexer(path string) (*Lexer, error) {
	abs, content, err := readFileAbs(path)
	if err != nil {
		return nil, err
	}
	l := &Lexer{
		lexerFrame: lexerFrame{input: content, line: 1, file: abs},
		active:     map[string]bool{abs: true},
	}
	l.readChar()
	return l, nil
}

func readFileAbs(path string) (abs string, content string, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	abs, err = filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	return abs, string(b), nil
}

func (l *Lexer) readChar() {
	if l.ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	if l.readPos >= len(l.input) && l.popFrame() {
		return
	}
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

// popFrame restores the most recently suspended parent frame (resuming
// right after the #include directive that pushed it) and reports whether a
// frame was available to pop.
func (l *Lexer) popFrame() bool {
	if len(l.stack) == 0 {
		return false
	}
	delete(l.active, l.file)
	n := len(l.stack)
	l.lexerFrame = l.stack[n-1]
	l.stack = l.stack[:n-1]
	return true
}

// newToken returns a single byte token with the current
// character and advances the lexer.
func (l *Lexer) newToken(tokenType LexemeType, line, col int) LexerToken {
	tk := LexerToken{Type: tokenType, Literal: string(l.ch), Line: line, Col: col}
	l.readChar()
	return tk
}

func (l *Lexer) NextToken() LexerToken {
	l.skipWhitespace()
	line, col := l.line, l.col

	switch l.ch {
	case '{':
		return l.newToken(TokenLCurly, line, col)
	case '}':
		return l.newToken(TokenRCurly, line, col)
	case '[':
		return l.newToken(TokenLBracket, line, col)
	case ']':
		return l.newToken(TokenRBracket, line, col)
	case '/':
		if isLetter(l.peekChar()) {
			l.readChar()
			literal := l.readIdentifier()
			return LexerToken{Type: TokenBinder, Literal: "/" + literal, Line: line, Col: col}
		} else if l.peekChar() == '*' {
			if err := l.skipBlockComment(); err != nil {
				return LexerToken{Type: TokenError, Literal: err.Error(), Line: line, Col: col}
			}
			return l.NextToken()
		} else {
			return l.newToken(TokenIllegal, line, col)
		}
	case '"':
		literal, err := l.readString()
		typ := TokenString
		if err != nil {
			typ = TokenIllegal
		}
		return LexerToken{Type: typ, Literal: literal, Line: line, Col: col}
	case '%':
		l.skipComment()
		return l.NextToken()
	case '#':
		if err := l.handleDirective(); err != nil {
			return LexerToken{Type: TokenError, Literal: err.Error(), Line: line, Col: col}
		}
		return l.NextToken()
	case 0:
		return LexerToken{Type: TokenEOF, Literal: "", Line: line, Col: col}
	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			var tokType LexemeType
			if literal == "true" || literal == "false" {
				tokType = TokenBoolean
			} else {
				tokType = TokenIdent
			}
			return LexerToken{Type: tokType, Literal: literal, Line: line, Col: col}
		} else if isDigit(l.ch) || l.ch == '-' {
			literal, typ := l.readNumber()
			return LexerToken{Type: typ, Literal: literal, Line: line, Col: col}
		} else {
			return l.newToken(TokenIllegal, line, col)
		}
	}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) skipInlineSpace() {
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}
}

// skipBlockComment consumes a C-style /* ... */ comment. l.ch must be '/'
// with a peeked '*' when called.
func (l *Lexer) skipBlockComment() error {
	l.readChar() // consume '/'
	l.readChar() // consume '*'
	for {
		if l.ch == 0 {
			return errors.New("unterminated block comment")
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			return nil
		}
		l.readChar()
	}
}

// handleDirective consumes a preprocessor directive. l.ch must be '#' when
// called. It supports a minimal subset: #include, and the classic
// #ifndef/#define/#endif header-guard idiom.
func (l *Lexer) handleDirective() error {
	l.readChar() // consume '#'
	l.skipInlineSpace()
	word := l.readIdentifier()
	switch word {
	case "include":
		return l.handleInclude()
	case "ifndef":
		return l.handleIfndef()
	case "define":
		return l.handleDefine()
	case "endif":
		return l.handleEndif()
	default:
		return fmt.Errorf("unsupported preprocessor directive: #%s", word)
	}
}

func (l *Lexer) handleInclude() error {
	l.skipInlineSpace()
	if l.ch != '"' {
		return errors.New("expected quoted filename after #include")
	}
	name, err := l.readString()
	if err != nil {
		return fmt.Errorf("invalid #include filename: %w", err)
	}
	return l.pushInclude(name)
}

// pushInclude resolves name relative to the including file's directory
// (or the current directory, for a raw-string top-level input), then
// suspends the current frame and switches the lexer to read from it.
func (l *Lexer) pushInclude(name string) error {
	dir := "."
	if l.file != "" {
		dir = filepath.Dir(l.file)
	}
	path := filepath.Join(dir, name)
	abs, content, err := readFileAbs(path)
	if err != nil {
		return fmt.Errorf("#include %q: %w", name, err)
	}
	if l.active[abs] {
		return fmt.Errorf("#include %q: include cycle detected", name)
	}
	if l.active == nil {
		l.active = make(map[string]bool)
	}
	l.active[abs] = true
	l.stack = append(l.stack, l.lexerFrame)
	l.lexerFrame = lexerFrame{input: content, line: 1, file: abs}
	l.readChar()
	return nil
}

func (l *Lexer) handleIfndef() error {
	l.skipInlineSpace()
	name := l.readIdentifier()
	if name == "" {
		return errors.New("expected identifier after #ifndef")
	}
	if l.defined[name] {
		return l.skipConditional()
	}
	l.condDepth++
	return nil
}

func (l *Lexer) handleDefine() error {
	l.skipInlineSpace()
	name := l.readIdentifier()
	if name == "" {
		return errors.New("expected identifier after #define")
	}
	if l.defined == nil {
		l.defined = make(map[string]bool)
	}
	l.defined[name] = true
	return nil
}

func (l *Lexer) handleEndif() error {
	if l.condDepth == 0 {
		return errors.New("#endif without matching #ifndef")
	}
	l.condDepth--
	return nil
}

// skipConditional discards the body of an #ifndef block whose condition was
// false, up to and including its matching #endif. It scans raw characters
// rather than tokens (dead code need not be lexically valid GML), tracking
// nested #ifndef/#endif directives to find the correct matching #endif.
// Everything else, including nested #include and #define, is ignored.
func (l *Lexer) skipConditional() error {
	depth := 1
	for depth > 0 {
		if l.ch == 0 {
			return errors.New("unterminated #ifndef: missing #endif")
		}
		if l.ch == '#' {
			l.readChar()
			l.skipInlineSpace()
			switch l.readIdentifier() {
			case "ifndef":
				depth++
			case "endif":
				depth--
			}
			continue
		}
		l.readChar()
	}
	return nil
}

func (l *Lexer) readIdentifier() string {
	pos := l.pos
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '-' || l.ch == '_' {
		l.readChar()
	}
	return l.input[pos:l.pos]
}

func (l *Lexer) readNumber() (string, LexemeType) {
	pos := l.pos
	typ := TokenInt
	if l.ch == '-' {
		l.readChar()
	}
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' {
		typ = TokenFloat
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	if l.ch == 'e' || l.ch == 'E' {
		typ = TokenFloat
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[pos:l.pos], typ
}

var (
	errIllegalEscape  = errors.New("illegal escape sequence")
	errUnclosedString = errors.New("unclosed string literal")
)

func (l *Lexer) readString() (string, error) {
	var sb strings.Builder
	var err error
	l.readChar()
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			default:
				err = errIllegalEscape
				sb.WriteByte('\\')
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
		}
		l.readChar()
	}
	if l.ch == '"' {
		l.readChar()
	} else if err == nil {
		err = errUnclosedString
	}
	return sb.String(), err
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
