package gml

// TODO: The error handling / reporting is not great (or existing at all).
//
// We avoid the name "Token" in some of the types here because this also refers
// to some of the types in the BNF grammar for the parser and overloading it
// to refer to the lexer tokens can be confusing.

import (
	"errors"
	"strings"
)

type LexemeType int

const (
	TokenUnknown LexemeType = iota
	TokenEOF
	TokenIllegal
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

type Lexer struct {
	input   string
	pos     int
	readPos int
	ch      byte
	line    int
	col     int
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
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

var errIllegalEscape = errors.New("illegal escape sequence")

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
	l.readChar() // closing quote
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
