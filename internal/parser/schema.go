package parser

import (
	"fmt"
	"strings"
)

type Field struct {
	Name      string  // "description"
	DataType  Token   // "TEXT"
	NotNull   bool    // true if NOT NULL, false if nullable
	Default   *string // optional default value
	IsPrimary bool    // true if PRIMARY KEY
	IsUnique  bool    // true if UNIQUE
}

type Table struct {
	Name   string  // Table name
	Fields []Field // table Fields
}

type Enum struct {
	Name   string
	Values []string
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
			} else {
				return fmt.Errorf("[SCHEMA_PARSER] unexpected token: %s", a.currentToken.Literal)
			}
		case SEMICOLON:
			a.NextToken()
		default:
			return fmt.Errorf("unexpected token: %s", a.currentToken.Literal)
		}
	}

	return nil
}

func (a *Ast) parseTable() error {
	stmt := &Table{}

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
	idx := 0 // debugging
	for a.currentToken.Type != SEMICOLON && a.currentToken.Type != EOF {
		field, err := a.parseTableField(idx)
		if err != nil {
			return err
		}
		idx = idx + 1
		fields = append(fields, *field)
		a.NextToken()
	}

	stmt.Fields = fields
	a.Statements = append(a.Statements, stmt)
	return nil
}

func (a *Ast) parseTableField(idx int) (*Field, error) {
	var field Field
	if a.currentToken.Type != STRING {
		return nil, fmt.Errorf("[PARSER_TABLE] unexpected token wanted columns name STRING got: %s field_idx: %d", a.currentToken.Literal, idx)
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
		return nil, fmt.Errorf("[PARSER_TABLE] expected datatype (IDENT or STRING), got: %s field_idx: %d", a.currentToken.Literal, idx)
	}

	for a.currentToken.Type != COMMA && a.currentToken.Type != RPAREN && a.currentToken.Type != EOF {
		switch a.currentToken.Type {
		case PRIMARY:
			if a.peekToken.Type == KEY {
				field.IsPrimary = true
				a.NextToken() // consume PRIMARY
				a.NextToken() // consume KEY
			} else {
				return nil, fmt.Errorf("[PARSER_TABLE] expected KEY got: %s field_idx: %d", a.currentToken.Literal, idx)
			}
		case NOT:
			if a.peekToken.Type == NULL {
				field.NotNull = true
				a.NextToken() // consume NOT
				a.NextToken() // consume NULL
			} else {
				return nil, fmt.Errorf("[PARSER_TABLE] expected NULL got: %s field_idx: %d", a.currentToken.Literal, idx)
			}
		case NULL:
			a.NextToken()
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
				continue
			}

			// Handle literals like: "Status" 'open'
			if (a.currentToken.Type == IDENT || a.currentToken.Type == STRING) &&
				a.isPrimitiveLiteral(a.peekToken.Type) {
				a.NextToken() // Skip the prefix "Status"
				val := a.currentToken.Literal
				field.Default = &val
				a.NextToken()
				continue
			}

			// Handle plain literals (STRING, INT, TRUE, FALSE)
			if a.isPrimitiveLiteral(a.currentToken.Type) {
				val := a.currentToken.Literal
				field.Default = &val
				a.NextToken()
				continue
			}

			return nil, fmt.Errorf(
				"[PARSER_TABLE] invalid DEFAULT value: %s TYPE: %v field_idx: %d",
				a.currentToken.Literal, a.currentToken.Type, idx,
			)
		default:
			return nil, fmt.Errorf("[PARSER_TABLE] unexpected token in field definition: %s TYPE: %v field_idx: %d", a.currentToken.Literal, a.currentToken.Type, idx)
		}
	}
	return &field, nil
}

func (a Ast) isPrimitiveLiteral(t TokenType) bool {
	return t == STRING || t == INT || t == TRUE || t == FALSE
}

func (a *Ast) parseType() error {
	stmt := &Enum{}
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

func (a *Ast) parseIndex() error {
	stmt := &Enum{}
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

func stringifyTableType(t *Table) string {
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

func stringifyEnumType(e *Enum) string {
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
