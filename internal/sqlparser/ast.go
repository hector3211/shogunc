package sqlparser

import (
	"bytes"
	"fmt"
	"strconv"
)

type Node any

type Condition struct {
	Ident     []byte // Column | Order | Group
	Condition []byte // AND | OR | =
	Value     any    // $1, $2
}

type SelectStatement struct {
	Fields     []string
	Conditions []Condition
	TableName  []byte
	Distinct   bool
	Limit      int
	Offset     int
}

type InsertStatement struct {
	TableName       string
	Columns         [][]byte
	Values          []int
	ReturningFields [][]byte
	InsertMode      []byte
}

type Ast struct {
	l            *Lexer
	Statements   []Token
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

func (a *Ast) Parse() (Node, error) {
	a.NextToken()
	a.NextToken()

	switch a.currentToken.Type {
	case SELECT:
		return a.parseSelect()
	case INSERT:
		return a.parserInsert()
	default:
		return nil, fmt.Errorf("unexpected token: %s", a.currentToken.Literal)
	}
}

func (a *Ast) parseSelect() (*SelectStatement, error) {
	stmt := &SelectStatement{}
	a.NextToken()

	// Parse SELECT
	for a.currentToken.Type != FROM && a.currentToken.Type != EOF {
		if a.currentToken.Type == ASTERIK {
			stmt.Fields = append(stmt.Fields, a.currentToken.Literal)
			break
		}
		if a.currentToken.Type == IDENT {
			stmt.Fields = append(stmt.Fields, a.currentToken.Literal)
		}
		a.NextToken()
	}

	// Parse FROM
	if a.currentToken.Type != FROM {
		return nil, fmt.Errorf("expected FROM got %s", a.currentToken.Literal)
	}
	a.NextToken()

	if a.currentToken.Type != IDENT {
		return nil, fmt.Errorf("expected table name got %s", a.currentToken.Literal)
	}
	stmt.TableName = []byte(a.currentToken.Literal)
	a.NextToken()

	// Parse WHERE
	if a.currentToken.Type != WHERE {
		return nil, fmt.Errorf("expected WHERE got %s", a.currentToken.Literal)
	}

	a.NextToken()

	var conditions []Condition
	for a.currentToken.Type != SEMICOLON && a.currentToken.Type != EOF {
		if a.currentToken.Type == LIMIT || a.currentToken.Type == OFFSET {
			break
		}
		cond := Condition{}
		if a.currentToken.Type == IDENT {
			cond.Ident = []byte(a.currentToken.Literal)
			a.NextToken()
		}

		if IsConditional(a.currentToken.Literal) {
			cond.Condition = []byte(a.currentToken.Literal)
			a.NextToken()
		}

		switch a.currentToken.Type {
		case BINDPARAM:
			a.NextToken()
			val, err := strconv.Atoi(a.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			cond.Value = val
		case STRING:
			cond.Value = a.currentToken.Literal
		}

		if cond.Ident != nil && cond.Condition != nil {
			conditions = append(conditions, cond)
		}
		a.NextToken()
	}

	// fmt.Println("before LIMIT:", a.currentToken.Type, a.currentToken.Literal)
	if a.currentToken.Type == LIMIT {
		a.NextToken()
		if a.currentToken.Type == INT {
			val, err := strconv.Atoi(a.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			stmt.Limit = val
			a.NextToken()
		}
	}

	if a.currentToken.Type == OFFSET {
		a.NextToken()
		if a.currentToken.Type == INT {
			val, err := strconv.Atoi(a.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			stmt.Offset = val
			a.NextToken()
		}
	}
	stmt.Conditions = conditions

	return stmt, nil
}

func (a *Ast) parserInsert() (*InsertStatement, error) {
	stmt := &InsertStatement{}
	a.NextToken()

	// Parse Insert
	for a.currentToken.Type != VALUES && a.currentToken.Type != EOF {
		if a.currentToken.Type == IDENT {
			stmt.TableName = a.currentToken.Literal
		}
		if a.currentToken.Type == LPAREN {
			for a.currentToken.Type != RPAREN {
				if a.currentToken.Type != COMMA {
					stmt.Columns = append(stmt.Columns, []byte(a.currentToken.Literal))
				}
				a.NextToken()
			}
		}
		a.NextToken()
	}

	// Parse Values
	for a.currentToken.Type != SEMICOLON && a.currentToken.Type != EOF && a.currentToken.Type == RETURNING {
		if a.currentToken.Type == LPAREN {
			curr := a.currentToken
			for curr.Type != RPAREN {
				if curr.Type == BINDPARAM {
					a.NextToken()
					val, err := strconv.Atoi(a.currentToken.Literal)
					if err != nil {
						return nil, fmt.Errorf("invalid bind param: %v", err)
					}
					stmt.Values = append(stmt.Values, val)
				}
				a.NextToken()
			}
		}
		a.NextToken()
	}

	if a.currentToken.Type == RETURNING {
		a.NextToken()
		for a.currentToken.Type != SEMICOLON && a.currentToken.Type != EOF {
			if a.currentToken.Type == IDENT {
				stmt.ReturningFields = append(stmt.ReturningFields, []byte(a.currentToken.Literal))
			}
		}
	}

	return stmt, nil
}

func (a *Ast) String() string {
	var out bytes.Buffer
	firstToken := a.Statements[0]

	switch firstToken.Type {
	case SELECT:
		out.WriteString(stringifySelectSatement(a.Statements))
	case INSERT:
		out.WriteString(stringifyInsertSatement(a.Statements))
	}

	return out.String()
}

func stringifySelectSatement(tokens []Token) string {
	stmt := ""
	multipleSelects := false

	for _, tok := range tokens {
		stmt += tok.Literal
		if tok.Type == LPAREN {
			multipleSelects = true
		}

		if tok.Type == RPAREN {
			multipleSelects = false
		}

		if !multipleSelects && tok.Type != IDENT && tok.Type != SEMICOLON {
			stmt += " "
		}
	}
	return stmt
}

func stringifyInsertSatement(tokens []Token) string {
	stmt := ""
	multipleInserts := false

	for _, tok := range tokens {
		stmt += tok.Literal
		if tok.Type == LPAREN {
			multipleInserts = true
		}

		if tok.Type == RPAREN {
			multipleInserts = false
		}

		if !multipleInserts && tok.Type != RPAREN && tok.Type != SEMICOLON {
			stmt += " "
		}

	}
	return stmt
}
