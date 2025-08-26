package parser

type Node any

type Ast struct {
	l            *Lexer
	Statements   []Node
	currentToken Token
	peekToken    Token
}

func NewAst(l *Lexer) *Ast {
	return &Ast{
		l: l,
	}
}

func (a *Ast) NextToken() {
	a.currentToken = a.peekToken
	a.peekToken = a.l.NextToken()
}
