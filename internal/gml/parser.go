package gml

import (
	"fmt"
	"strconv"
	"strings"
)

type Parser struct {
	lexer *Lexer
	curr  LexerToken
}

func NewParser(input string) *Parser {
	return &Parser{lexer: NewLexer(input)}
}

func (p *Parser) readAndAdvanceToken() LexerToken {
	token := p.curr
	p.curr = p.lexer.NextToken()
	return token
}

func (p *Parser) consume(tokenType LexemeType) error {
	if p.curr.Type != tokenType {
		return fmt.Errorf("expected %s, got %s", tokenType, p.curr.Type)
	}
	p.readAndAdvanceToken()
	return nil
}

func (p *Parser) currToken() LexerToken {
	return p.curr
}

func (p *Parser) Parse() (TokenList, error) {
	p.readAndAdvanceToken()
	l, err := p.parseTokenList()
	if err != nil {
		return nil, err
	}
	if p.curr.Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token: %s, expected end of input", p.curr.Type)
	}
	return l, nil
}

// TokenList
//
//	::= 	TokenGroup*
func (p *Parser) parseTokenList() (TokenList, error) {
	var l TokenList
	for startsTokenGroup(p.currToken().Type) {
		group, err := p.parseTokenGroup()
		if err != nil {
			return nil, err
		}
		l = append(l, group)
	}
	return l, nil
}

func startsTokenGroup(tokenType LexemeType) bool {
	switch tokenType {
	// "Compound" tokens:
	case TokenLBracket, TokenLCurly:
		return true
	// Single tokens:
	case TokenIdent, TokenInt, TokenFloat, TokenString, TokenBinder, TokenBoolean:
		return true
	default:
		return false
	}
}

// TokenGroup
//
//	::= 	Token
//	| 	{ TokenList }
//	| 	[ TokenList ]
func (p *Parser) parseTokenGroup() (TokenGroup, error) {
	switch p.curr.Type {
	case TokenLBracket:
		return p.parseArray()
	case TokenLCurly:
		return p.parseFunction()
	default:
		return p.parseSingleToken()
	}
}

// Token
//
//	::= 	Operator
//	| 	Identifier
//	| 	Binder
//	| 	Boolean
//	| 	Integer
//	| 	Float
//	| 	String
func (p *Parser) parseSingleToken() (TokenGroup, error) {
	switch p.currToken().Type {
	case TokenIdent:
		return &Identifier{Name: p.readAndAdvanceToken().Literal}, nil
	case TokenInt:
		return p.parseIntLiteral()
	case TokenFloat:
		return p.parseFloatLiteral()
	case TokenString:
		return &StringLiteral{Value: p.readAndAdvanceToken().Literal}, nil
	case TokenBinder:
		return p.parseBinder()
	case TokenBoolean:
		return p.parseBooleanLiteral()
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.currToken().Type)
	}
}

func (p *Parser) parseBinder() (*Binder, error) {
	token := p.readAndAdvanceToken()
	name := token.Literal
	if !strings.HasPrefix(name, "/") {
		return nil, fmt.Errorf("binder must start with /, got %s", token.Type)
	}
	return &Binder{Name: name[1:]}, nil
}

func (p *Parser) parseFloatLiteral() (TokenGroup, error) {
	token := p.readAndAdvanceToken()
	val, err := strconv.ParseFloat(token.Literal, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse number: %s", token.Literal)
	}
	return &FloatLiteral{Value: val}, nil
}

func (p *Parser) parseIntLiteral() (TokenGroup, error) {
	token := p.readAndAdvanceToken()
	val, err := strconv.ParseInt(token.Literal, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse number: %s", token.Literal)
	}
	return &IntLiteral{Value: val}, nil
}

func (p *Parser) parseBooleanLiteral() (TokenGroup, error) {
	token := p.readAndAdvanceToken()
	val, err := strconv.ParseBool(token.Literal)
	if err != nil {
		return nil, fmt.Errorf("could not parse boolean: %s", token.Literal)
	}
	return &BoolLiteral{Value: val}, nil
}

func (p *Parser) parseArray() (TokenGroup, error) {
	if err := p.consume(TokenLBracket); err != nil {
		return nil, err
	}
	l, err := p.parseTokenList()
	if err != nil {
		return nil, err
	}
	if err := p.consume(TokenRBracket); err != nil {
		return nil, err
	}
	return &Array{Elements: l}, nil
}

func (p *Parser) parseFunction() (TokenGroup, error) {
	if err := p.consume(TokenLCurly); err != nil {
		return nil, err
	}
	l, err := p.parseTokenList()
	if err != nil {
		return nil, err
	}
	if err := p.consume(TokenRCurly); err != nil {
		return nil, err
	}
	return &Function{Body: l}, nil
}
