package sqlparser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type Node any

type Field struct {
	Name      string  // "description"
	DataType  Token   // "TEXT"
	NotNull   bool    // true if NOT NULL, false if nullable
	Default   *string // optional default value
	IsPrimary bool    // true if PRIMARY KEY
	IsUnique  bool    // true if UNIQUE
}

type TableType struct {
	Name   string
	Fields []Field
}

type EnumType struct {
	Name   string
	Values []string
}

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

func (a *Ast) Parse() error {
	a.NextToken()
	a.NextToken()

	switch a.currentToken.Type {
	case SELECT:
		return a.parseSelect()
	case INSERT:
		return a.parserInsert()
	default:
		return fmt.Errorf("unexpected token: %s", a.currentToken.Literal)
	}
}

func (a *Ast) ParseSchema() error {
	a.NextToken()
	a.NextToken()

	for a.currentToken.Type != EOF {
		switch a.currentToken.Type {
		case CREATE:
			a.NextToken()
			if a.currentToken.Type == TABLE {
				if err := a.parseTable(); err != nil {
					return err
				}
				a.NextToken()
			} else if a.currentToken.Type == TYPE {
				if err := a.parseType(); err != nil {
					return err
				}
				a.NextToken()
			}
		case SEMICOLON:
			a.NextToken()
		default:
			return fmt.Errorf("unexpected token: %s", a.currentToken.Literal)
		}
	}

	return nil
}

func (a *Ast) parseSelect() error {
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
		return fmt.Errorf("expected FROM got %s", a.currentToken.Literal)
	}
	a.NextToken()

	if a.currentToken.Type != IDENT {
		return fmt.Errorf("expected table name got %s", a.currentToken.Literal)
	}
	stmt.TableName = []byte(a.currentToken.Literal)
	a.NextToken()

	// Parse WHERE
	if a.currentToken.Type != WHERE {
		return fmt.Errorf("expected WHERE got %s", a.currentToken.Literal)
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
				return fmt.Errorf("invalid bind param: %v", err)
			}
			cond.Right = val
		case STRING:
			cond.Right = a.currentToken.Literal
		case INT:
			val, err := strconv.Atoi(a.currentToken.Literal)
			if err != nil {
				return fmt.Errorf("invalid bind param: %v", err)
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
				return fmt.Errorf("invalid bind param: %v", err)
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
				return fmt.Errorf("invalid bind param: %v", err)
			}
			stmt.Offset = val
			a.NextToken()
		}
	}
	stmt.Conditions = conditions

	a.Statements = append(a.Statements, stmt)
	return nil
}

func (a *Ast) parserInsert() error {
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
						return fmt.Errorf("invalid bind param: %v", err)
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

	a.Statements = append(a.Statements, stmt)
	return nil
}

func (a *Ast) parseTable() error {
	stmt := &TableType{}

	for a.currentToken.Type != STRING && a.currentToken.Type != EOF {
		a.NextToken()
	}

	if a.currentToken.Type != STRING {
		return fmt.Errorf("[PARSER_TABLE] unexpected token: %s wanted STRING", a.currentToken.Literal)
	}
	stmt.Name = a.currentToken.Literal
	a.NextToken()

	if a.currentToken.Type != LPAREN {
		return fmt.Errorf("[PARSER_TABLE] unexpected token: %s wanted LPAREN", a.currentToken.Literal)
	}
	a.NextToken()

	var fields []Field
	// debug
	idx := 0
	for a.currentToken.Type != SEMICOLON && a.currentToken.Type != EOF {
		var field Field
		if a.currentToken.Type != STRING {
			return fmt.Errorf("[PARSER_TABLE] unexpected token wanted columns name STRING got: %s field_idx: %d", a.currentToken.Literal, idx)
		}
		field.Name = a.currentToken.Literal
		a.NextToken()

		if a.currentToken.Type == IDENT && IsDatabaseType(a.currentToken.Literal) {
			// TODO add datatype validation
			field.DataType = a.currentToken
			a.NextToken()
		} else if a.currentToken.Type == STRING {
			// ENUM type
			field.DataType = Token{Type: ENUM, Literal: a.currentToken.Literal}
			a.NextToken()
		} else {
			return fmt.Errorf("[PARSER_TABLE] expected datatype (IDENT or STRING), got: %s field_idx: %d", a.currentToken.Literal, idx)
		}

		for a.currentToken.Type != COMMA && a.currentToken.Type != RPAREN && a.currentToken.Type != EOF {
			switch a.currentToken.Type {
			case PRIMARY:
				if a.peekToken.Type == KEY {
					field.IsPrimary = true
					a.NextToken() // consume PRIMARY
					a.NextToken() // consume KEY
				} else {
					return fmt.Errorf("[PARSER_TABLE] expected KEY got: %s field_idx: %d", a.currentToken.Literal, idx)
				}
			case NOT:
				if a.peekToken.Type == NULL {
					field.NotNull = true
					a.NextToken() // consume NOT
					a.NextToken() // consume NULL
				} else {
					return fmt.Errorf("[PARSER_TABLE] expected NULL got: %s field_idx: %d", a.currentToken.Literal, idx)
				}

			case UNIQUE:
				field.IsUnique = true
				a.NextToken()
			case DEFAULT:
				a.NextToken()
				if IsNowCompatible(field.DataType) && a.currentToken.Type == IDENT && a.peekToken.Type == LPAREN {
					val := SqlNow(a.currentToken)
					field.Default = &val
					a.NextToken() // Consume now
					a.NextToken() // Consume (
					a.NextToken() // Consume )
				} else if a.currentToken.Type == STRING || a.currentToken.Type == INT || a.currentToken.Type == TRUE || a.currentToken.Type == FALSE {
					val := a.currentToken.Literal
					field.Default = &val
					a.NextToken()
				} else {
					return fmt.Errorf("[PARSER_TABLE] expected literal (STRING, INT, TRUE, FALSE, or now()), got: %s field_idx: %d", a.currentToken.Literal, idx)
				}
			default:
				return fmt.Errorf("[PARSER_TABLE] unexpected token in field definition: %s field_idx: %d", a.currentToken.Literal, idx)
			}
		}
		idx = idx + 1
		fields = append(fields, field)
		a.NextToken()
	}

	stmt.Fields = fields
	a.Statements = append(a.Statements, stmt)
	return nil
}

func (a *Ast) parseType() error {
	stmt := &EnumType{}
	a.NextToken()

	if a.currentToken.Type != STRING {
		return fmt.Errorf("unexpected token: %s WANTED STRING", a.currentToken.Literal)
	}
	stmt.Name = a.currentToken.Literal
	a.NextToken()

	if a.currentToken.Type != AS && a.peekToken.Type != ENUM {
		return fmt.Errorf("failed parsing ENUM invalid got: %s", a.currentToken.Literal)
	}
	a.NextToken()

	for a.currentToken.Type != RPAREN {
		if a.currentToken.Type == STRING {
			stmt.Values = append(stmt.Values, a.currentToken.Literal)
		}
		a.NextToken()
	}

	a.Statements = append(a.Statements, stmt)
	return nil
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

func stringifyTableType(t *TableType) string {
	var sb strings.Builder
	sb.WriteString("CREATE TABLE IF NOT EXISTS ")
	sb.WriteString(fmt.Sprintf("\"%s\"", t.Name))
	sb.WriteString(" (\n")

	for i, field := range t.Fields {
		if field.DataType.Type == ENUM {
			sb.WriteString(fmt.Sprintf("  \"%s\" \"%s\"", field.Name, field.DataType.Literal))
		} else {
			sb.WriteString(fmt.Sprintf("  \"%s\" %v", field.Name, field.DataType.Literal))
		}

		if field.NotNull {
			sb.WriteString(" NOT NULL")
		}
		if field.IsUnique {
			sb.WriteString(" UNIQUE")
		}
		if field.IsPrimary {
			sb.WriteString(" PRIMARY KEY")
		}
		if field.Default != nil {
			sb.WriteString(" DEFAULT ")
			sb.WriteString(fmt.Sprintf("'%s'", *field.Default))
		}

		if i < len(t.Fields)-1 {
			sb.WriteString(",\n")
		} else {
			sb.WriteString("\n")
		}
	}

	sb.WriteString(");")
	return sb.String()
}

func stringifyEnumType(e *EnumType) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TYPE \"%s\" AS ENUM (", e.Name))

	for i, val := range e.Values {
		sb.WriteString(fmt.Sprintf("'%s'", val))
		if i < len(e.Values)-1 {
			sb.WriteString(",")
		}
	}

	sb.WriteString(");")
	return sb.String()
}
