package sqlparser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type Node any

type LogicalOp string

const (
	And     LogicalOp = "And"
	Or      LogicalOp = "Or"
	Illegal LogicalOp = ""
)

func toLogicOp(op string) LogicalOp {
	switch op {
	case "AND":
		return And
	case "OR":
		return Or
	default:
		return Illegal
	}
}

type ConditionOp string

const (
	EQUAL       ConditionOp = "="
	NOTEQUAL    ConditionOp = "!="
	LESSTHAN    ConditionOp = "<"
	GREATERTHAN ConditionOp = ">"
	BETWEEN     ConditionOp = "BETWEEN"
	ISNULL      ConditionOp = "IS NULL"
	NOTNULL     ConditionOp = "IS NOT NULL"
)

type Condition struct {
	Left     []byte      // Column
	Next     LogicalOp   // AND | OR | NOT
	Operator ConditionOp // = | != | >
	Right    any         // $1, $2
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
		switch a.currentToken.Type {
		case ASTERIK, IDENT:
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
		if len(cond.Next) == 0 && IsLogicalOperator(a.currentToken.Literal) {
			cond.Next = toLogicOp(a.currentToken.Literal)
			a.NextToken()
		}

		if a.currentToken.Type == IDENT {
			cond.Left = []byte(a.currentToken.Literal)
			a.NextToken()
		}

		if cond.Operator == "" && IsConditional(a.currentToken.Literal) {
			cond.Operator = ConditionOp(a.currentToken.Literal)
			a.NextToken()
		}

		switch a.currentToken.Type {
		case BINDPARAM:
			a.NextToken()
			val, err := strconv.Atoi(a.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			cond.Right = val
		case STRING:
			cond.Right = a.currentToken.Literal
		case INT:
			val, err := strconv.Atoi(a.currentToken.Literal)
			if err != nil {
				return nil, fmt.Errorf("invalid bind param: %v", err)
			}
			cond.Right = val
		}

		if cond.Left != nil && cond.Operator != "" && cond.Right != nil {
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

	a.Statements = append(a.Statements, stmt)
	return stmt, nil
}

func (a *Ast) parserInsert() (*InsertStatement, error) {
	stmt := &InsertStatement{}
	a.NextToken()

	// Parse Insert
	for a.currentToken.Type != VALUES && a.currentToken.Type != EOF {
		if a.currentToken.Type == IDENT {
			stmt.TableName = []byte(a.currentToken.Literal)
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
	if len(a.Statements) == 0 {
		return ""
	}

	switch stmt := a.Statements[0].(type) {
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
			if len(c.Next) > 0 && i > 0 {
				sb.WriteString(string(c.Next))
				sb.WriteString(" ")
			}

			sb.WriteString(string(c.Left))
			sb.WriteString(" ")
			sb.WriteString(string(c.Operator))
			sb.WriteString(" ")

			switch v := c.Right.(type) {
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
