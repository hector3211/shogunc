package sqlparser

import (
	"strings"
	"unicode"
)

type Lexer struct {
	input        string
	position     int // Current position
	readPosition int // Next Position
	ch           byte
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.ReadChar()
	return l
}

func (l *Lexer) ReadChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition += 1
}

func (l *Lexer) NextToken() Token {
	var tok Token
	l.skipWhiteSpaces()

	switch l.ch {
	case '=':
		tok = CreateToken(ASSIGN, '=')
	case ',':
		tok = CreateToken(COMMA, ',')
	case ';':
		tok = CreateToken(SEMICOLON, ';')
	case '(':
		tok = CreateToken(LPAREN, '(')
	case ')':
		tok = CreateToken(RPAREN, ')')
	case '*':
		tok = CreateToken(ASTERIK, '*')
	case '$':
		tok = CreateToken(BINDPARAM, '$')
	case 0:
		tok.Literal = ""
		tok.Type = EOF
	default:
		if isString(l.ch) {
			tok.Type = STRING
			tok.Literal = l.readString()
			// fmt.Printf("Token STRING: %+v\n", tok)
			return tok
		} else if isLetter(l.ch) {
			// tok.Literal = l.readIdentifer()
			tok.Literal = strings.ToUpper(l.readIdentifer())
			tok.Type = LookupIdent(tok.Literal)
			// fmt.Printf("Token CHAR: %+v\n", tok)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = INT
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = CreateToken(ILLEGAL, l.ch)
		}
	}
	l.ReadChar()
	return tok
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.ReadChar()
	}

	return l.input[position:l.position]
}

func (l *Lexer) readIdentifer() string {
	position := l.position
	for isLetter(l.ch) || l.ch == '_' {
		l.ReadChar()
	}

	return l.input[position:l.position]
}

func (l *Lexer) skipWhiteSpaces() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.ReadChar()
	}
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch))
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}

func isString(ch byte) bool {
	return ch == '\'' || ch == '"'
}

func (l *Lexer) readString() string {
	quote := l.ch
	l.ReadChar() // skip opening '
	start := l.position
	for l.ch != quote && l.ch != 0 {
		l.ReadChar()
	}
	str := l.input[start:l.position]
	l.ReadChar() // skip closing '
	return str
}
