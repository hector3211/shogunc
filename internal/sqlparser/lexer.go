package sqlparser

import (
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
	l.readChar()
	return l
}

func (l *Lexer) NextToken() Token {
	var tok Token
	l.skipWhiteSpaces()

	switch l.ch {
	case '=':
		tok = CreateToken(ASSIGN, l.ch)
	case ',':
		tok = CreateToken(COMMA, l.ch)
	case ';':
		tok = CreateToken(SEMICOLON, l.ch)
	case '(':
		tok = CreateToken(LPAREN, l.ch)
	case ')':
		tok = CreateToken(RPAREN, l.ch)
	// TODO: Parse arrays
	case '[':
		tok = CreateToken(LBRACKET, l.ch)
	case ']':
		tok = CreateToken(RBRACKET, l.ch)
	case '*':
		tok = CreateToken(ASTERIK, l.ch)
	case '$':
		tok = CreateToken(BINDPARAM, l.ch)
	// NOTE: Parsing comments before AST
	case '-':
		tok = CreateToken(MINUS, l.ch)
		// if l.peekChar() == '-' {
		// 	tok.Type = COMMENT
		// 	tok.Literal = l.readComment()
		// 	return tok
		// } else {
		// }
	case '+':
		tok = CreateToken(PLUS, l.ch)
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
			tok.Literal = l.readIdentifer()
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
	l.readChar()
	return tok
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition += 1
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	} else {
		return l.input[l.readPosition]
	}
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}

	return l.input[position:l.position]
}

func (l *Lexer) readIdentifer() string {
	position := l.position
	for isLetter(l.ch) || l.ch == '_' {
		l.readChar()
	}

	return l.input[position:l.position]
}

func (l *Lexer) skipWhiteSpaces() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readString() string {
	l.readChar() // skip opening '
	position := l.position
	for l.ch != '\'' && l.ch != 0 {
		l.readChar()
	}
	str := l.input[position:l.position]
	l.readChar() // skip closing '
	return str
}

// func (l *Lexer) readComment() string {
// 	l.readChar()
// 	positition := l.position
// 	for l.ch != 0 {
// 		l.readChar()
// 	}
//
// 	str := l.input[positition:l.position]
// 	l.readChar()
// 	return str
// }

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch))
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}

func isString(ch byte) bool {
	return ch == '\''
}
