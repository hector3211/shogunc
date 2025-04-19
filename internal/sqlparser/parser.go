package sqlparser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

const (
	ERROR_OBJ = "ERROR"
)

type Error struct {
	Messsage []byte
	Line     int
}

func NewError(message string, line int) *Error {
	return &Error{
		Messsage: []byte(message),
		Line:     line,
	}
}

func (e *Error) Inspect() string {
	err := fmt.Sprintf("ERROR: %s LINE: %d", e.Messsage, e.Line)
	return err
}

type Condition struct {
	Ident     []byte // Column | Order | Group
	Logical   []byte // AND | OR | NOT
	Condition []byte // = | != | >
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
	TableName       []byte
	Columns         [][]byte
	Values          []int
	ReturningFields [][]byte
	InsertMode      []byte
}

type Node any

type Parser struct {
	l            *Lexer
	Statements   []Node
	currentToken Token
	peekToken    Token
}

func NewParser(l *Lexer) *Parser {
	return &Parser{
		l:          l,
		Statements: []Node{},
	}
}

func (p *Parser) NextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Parse() (Node, error) {
	p.NextToken()
	p.NextToken()

	switch p.currentToken.Type {
	case SELECT:
		return p.parseSelect()
	case INSERT:
		return p.parserInsert()
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.currentToken.Literal)
	}
}

func (p *Parser) parseSelect() (*SelectStatement, error) {
	stmt := &SelectStatement{}
	p.NextToken()

	// Parse SELECT
	for p.currentToken.Type != FROM && p.currentToken.Type != EOF {
		switch p.currentToken.Type {
		case ASTERIK, IDENT:
			stmt.Fields = append(stmt.Fields, p.currentToken.Literal)
		}
		p.NextToken()
	}

	// Parse FROM
	if p.currentToken.Type != FROM {
		return nil, fmt.Errorf("expected FROM got %s", p.currentToken.Literal)
	}
	p.NextToken()

	if p.currentToken.Type != IDENT {
		return nil, fmt.Errorf("expected table name got %s", p.currentToken.Literal)
	}
	stmt.TableName = []byte(p.currentToken.Literal)
	p.NextToken()

	// Parse WHERE
	if p.currentToken.Type != WHERE {
		return nil, fmt.Errorf("expected WHERE got %s", p.currentToken.Literal)
	}
	p.NextToken()

	var conditions []Condition
	for p.currentToken.Type != SEMICOLON && p.currentToken.Type != EOF {
		if p.currentToken.Type == LIMIT || p.currentToken.Type == OFFSET {
			break
		}
		cond := Condition{}
		if len(cond.Logical) == 0 && IsLogicalOperator(p.currentToken.Literal) {
			cond.Logical = []byte(p.currentToken.Literal)
			p.NextToken()
		}

		if p.currentToken.Type == IDENT {
			cond.Ident = []byte(p.currentToken.Literal)
			p.NextToken()
		}

		if len(cond.Condition) == 0 && IsConditional(p.currentToken.Literal) {
			cond.Condition = []byte(p.currentToken.Literal)
			p.NextToken()
		}

		switch p.currentToken.Type {
		case BINDPARAM:
			p.NextToken()
			val, err := strconv.Atoi(p.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			cond.Value = val
		case STRING:
			cond.Value = p.currentToken.Literal
		case INT:
			val, err := strconv.Atoi(p.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			cond.Value = val
		}

		if cond.Ident != nil && cond.Condition != nil && cond.Value != nil {
			conditions = append(conditions, cond)
		}
		p.NextToken()
	}

	// fmt.Println("before LIMIT:", a.currentToken.Type, a.currentToken.Literal)
	if p.currentToken.Type == LIMIT {
		p.NextToken()
		if p.currentToken.Type == INT {
			val, err := strconv.Atoi(p.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			stmt.Limit = val
			p.NextToken()
		}
	}

	if p.currentToken.Type == OFFSET {
		p.NextToken()
		if p.currentToken.Type == INT {
			val, err := strconv.Atoi(p.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			stmt.Offset = val
			p.NextToken()
		}
	}
	stmt.Conditions = conditions

	p.Statements = append(p.Statements, stmt)
	return stmt, nil
}

func (p *Parser) parserInsert() (*InsertStatement, error) {
	stmt := &InsertStatement{}
	p.NextToken()

	// Parse Insert
	for p.currentToken.Type != VALUES && p.currentToken.Type != EOF {
		if p.currentToken.Type == IDENT {
			stmt.TableName = []byte(p.currentToken.Literal)
		}
		if p.currentToken.Type == LPAREN {
			for p.currentToken.Type != RPAREN {
				if p.currentToken.Type != COMMA {
					stmt.Columns = append(stmt.Columns, []byte(p.currentToken.Literal))
				}
				p.NextToken()
			}
		}
		p.NextToken()
	}

	// Parse Values
	for p.currentToken.Type != SEMICOLON && p.currentToken.Type != EOF && p.currentToken.Type == RETURNING {
		if p.currentToken.Type == LPAREN {
			curr := p.currentToken
			for curr.Type != RPAREN {
				if curr.Type == BINDPARAM {
					p.NextToken()
					val, err := strconv.Atoi(p.currentToken.Literal)
					if err != nil {
						return nil, fmt.Errorf("invalid bind param: %v", err)
					}
					stmt.Values = append(stmt.Values, val)
				}
				p.NextToken()
			}
		}
		p.NextToken()
	}

	if p.currentToken.Type == RETURNING {
		p.NextToken()
		for p.currentToken.Type != SEMICOLON && p.currentToken.Type != EOF {
			if p.currentToken.Type == IDENT {
				stmt.ReturningFields = append(stmt.ReturningFields, []byte(p.currentToken.Literal))
			}
		}
	}

	return stmt, nil
}

func (p *Parser) String() string {
	var out bytes.Buffer
	if len(p.Statements) == 0 {
		return ""
	}

	switch stmt := p.Statements[0].(type) {
	case *SelectStatement:
		out.WriteString(stringifySelectStatement(stmt))
	case *InsertStatement:
		out.WriteString(stringifyInsertStatement(stmt))
	}

	return out.String()
}

func stringifySelectStatement(stmt *SelectStatement) string {
	var sb strings.Builder

	sb.WriteString("SELECT ")
	if stmt.Distinct {
		sb.WriteString("DISTINCT ")
	}

	if len(stmt.Fields) == 1 && stmt.Fields[0] == "*" {
		sb.WriteString("* ")
	} else {
		for i, f := range stmt.Fields {
			sb.WriteString(f)
			if i < len(stmt.Fields)-1 {
				sb.WriteString(", ")
			} else {
				sb.WriteString(" ")
			}
		}
	}

	sb.WriteString("FROM ")
	sb.WriteString(string(stmt.TableName))
	sb.WriteString(" ")

	if len(stmt.Conditions) > 0 {
		sb.WriteString("WHERE ")
		for i, c := range stmt.Conditions {
			if len(c.Logical) > 0 && i > 0 {
				sb.WriteString(string(c.Logical))
				sb.WriteString(" ")
			}

			sb.WriteString(string(c.Ident))
			sb.WriteString(" ")
			sb.WriteString(string(c.Condition))
			sb.WriteString(" ")

			switch v := c.Value.(type) {
			case string:
				sb.WriteString(fmt.Sprintf("'%s'", v))
			case int:
				sb.WriteString(fmt.Sprintf("%d", v))
			default:
				sb.WriteString(fmt.Sprintf("%v", v))
			}

			if i < len(stmt.Conditions)-1 {
				sb.WriteString(" ")
			}
		}
		sb.WriteString(" ")
	}

	if stmt.Limit > 0 {
		sb.WriteString(fmt.Sprintf("LIMIT %d ", stmt.Limit))
	}

	if stmt.Offset > 0 {
		sb.WriteString(fmt.Sprintf("OFFSET %d ", stmt.Offset))
	}

	return strings.TrimSpace(sb.String()) + ";"
}

func stringifyInsertStatement(stmt *InsertStatement) string {
	var sb strings.Builder

	sb.WriteString("INSERT ")
	if len(stmt.InsertMode) > 0 {
		sb.WriteString(string(stmt.InsertMode))
		sb.WriteString(" ")
	}
	sb.WriteString("INTO ")
	sb.WriteString(string(stmt.TableName))
	sb.WriteString(" ")

	if len(stmt.Columns) > 0 {
		sb.WriteString("(")
		for i, col := range stmt.Columns {
			sb.WriteString(string(col))
			if i < len(stmt.Columns)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString(") ")
	}

	sb.WriteString("VALUES (")
	for i, val := range stmt.Values {
		sb.WriteString(fmt.Sprintf("%v", val))
		if i < len(stmt.Values)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(") ")

	if len(stmt.ReturningFields) > 0 {
		sb.WriteString("RETURNING ")
		for i, field := range stmt.ReturningFields {
			sb.WriteString(string(field))
			if i < len(stmt.ReturningFields)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString(" ")
	}

	return strings.TrimSpace(sb.String()) + ";"
}
