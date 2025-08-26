package parser

import (
	"errors"
	"fmt"
	"shogunc/internal/types"
	"strconv"
	"strings"
)

func (a *Ast) Parse() error {
	a.NextToken()
	a.NextToken()

	switch a.currentToken.Type {
	case SELECT:
		return a.parseSelect()
	case INSERT:
		return a.parseInsert()
	default:
		return fmt.Errorf("unexpected token: %s", a.currentToken.Literal)
	}
}

func (a *Ast) parseSelect() error {
	stmt := &types.SelectStatement{}
	bindPositionCounter := 1 // Track automatic position assignment
	a.NextToken()

	// Parse SELECT
	for a.currentToken.Type != FROM && a.currentToken.Type != EOF { // Extracting columns
		switch a.currentToken.Type {
		case ASTERIK, IDENT:
			stmt.Columns = append(stmt.Columns, a.currentToken.Literal)
		}
		a.NextToken()
	}

	// Parse FROM
	a.advanceAndExpect(FROM)

	// Parse table name
	if a.currentToken.Type != IDENT {
		return fmt.Errorf("expected table name (IDENT), got %s", a.currentToken.Type)
	}
	stmt.TableName = strings.ToLower(a.currentToken.Literal)
	a.NextToken() // Advance past the table name

	// Parse WHERE
	var conditions []types.Condition
	if a.currentToken.Type == WHERE {
		for a.currentToken.Type != SEMICOLON && a.currentToken.Type != EOF {
			if a.currentToken.Type == LIMIT || a.currentToken.Type == OFFSET {
				break
			}
			cond := types.Condition{}
			if len(cond.ChainOp) == 0 && IsLogicalOperator(a.currentToken.Literal) {
				cond.ChainOp = types.ToLogicOp(a.currentToken.Literal)
				a.NextToken()
			}

			if a.currentToken.Type == IDENT {
				cond.Column = a.currentToken.Literal
				a.NextToken()
			}

			if cond.Operator == "" && IsConditional(a.currentToken.Literal) {
				cond.Operator = types.ConditionOp(a.currentToken.Literal)
				a.NextToken()
			}

			switch a.currentToken.Type {
			case BINDPARAM:
				a.NextToken()
				var position int // Bind position
				if a.currentToken.Literal != "" {
					// Explicit position like $1, $2
					var err error
					position, err = strconv.Atoi(a.currentToken.Literal)
					if err != nil {
						return fmt.Errorf("invalid bind param: %v", err)
					}
				} else {
					// Automatic position assignment for ? placeholders
					position = bindPositionCounter
					bindPositionCounter++
				}
				bind, err := a.parseBindParam(cond.Column, position, nil)
				if err != nil {
					return err
				}
				cond.Value = bind
			case STRING:
				bind, err := a.parseBindParam(cond.Column, 0, &a.currentToken.Literal)
				if err != nil {
					return err
				}
				cond.Value = bind
			case INT:
				bind, err := a.parseBindParam(cond.Column, 0, &a.currentToken.Literal)
				if err != nil {
					return err
				}
				cond.Value = bind
			}

			if cond.Column != "" && cond.Operator != "" && (cond.Value.Position != 0 || cond.Value.Value != nil) {
				conditions = append(conditions, cond)
			}
			a.NextToken()
		}
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

func (a *Ast) parseInsert() error {
	stmt := &types.InsertStatement{}
	bindPositionCounter := 1 // Track automatic position assignment
	columnIndex := 0         // Track which column we're processing
	a.NextToken()

	// Parse Insert
	for a.currentToken.Type != VALUES && a.currentToken.Type != EOF {
		if a.currentToken.Type == IDENT {
			stmt.TableName = a.currentToken.Literal
		}
		if a.currentToken.Type == LPAREN {
			for a.currentToken.Type != RPAREN {
				if a.currentToken.Type == IDENT {
					stmt.Columns = append(stmt.Columns, a.currentToken.Literal)
				}
				a.NextToken()
			}
		}
		a.NextToken()
	}

	// Parse VALUES
	a.advanceAndExpect(VALUES)
	// if a.currentToken.Type == VALUES {
	// 	a.NextToken()
	// }

	// Parse Values
	for a.currentToken.Type != SEMICOLON && a.currentToken.Type != EOF && a.currentToken.Type != RETURNING {
		if a.currentToken.Type == LPAREN {
			curr := a.currentToken
			for curr.Type != RPAREN {
				if curr.Type == BINDPARAM {
					a.NextToken()
					var position int
					if a.currentToken.Literal != "" {
						// Explicit position like $1, $2
						var err error
						position, err = strconv.Atoi(a.currentToken.Literal)
						if err != nil {
							return fmt.Errorf("invalid bind param: %v", err)
						}
					} else {
						// Automatic position assignment for ? placeholders
						position = bindPositionCounter
						bindPositionCounter++
					}

					// Get column name for this value
					var columnName string
					if columnIndex < len(stmt.Columns) {
						columnName = stmt.Columns[columnIndex]
					}

					// Create Bind struct with column association
					bindValue := types.Bind{
						Column:   columnName,
						Position: position,
					}
					stmt.Values = append(stmt.Values, bindValue)
					columnIndex++
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
				stmt.ReturningFields = append(stmt.ReturningFields, a.currentToken.Literal)
			}
		}
	}

	a.Statements = append(a.Statements, stmt)
	return nil
}

func (a *Ast) parseBindParam(columnName string, positionCounter int, defaultValue *string) (types.Bind, error) {
	if columnName == "" {
		return types.Bind{}, errors.New("invalid bind params, no column name provided")
	}
	return types.Bind{
		Column:   columnName,
		Position: positionCounter,
		Value:    defaultValue,
	}, nil
}

func (a *Ast) advanceAndExpect(expected TokenType) error {
	a.NextToken()
	if a.currentToken.Type != expected {
		return fmt.Errorf("expected %s, got %s", expected, a.currentToken.Type)
	}

	return nil
}

func (a *Ast) String() string {
	var out strings.Builder
	if len(a.Statements) == 0 {
		return ""
	}

	switch stmt := a.Statements[0].(type) {
	case *types.SelectStatement:
		out.WriteString(stringifySelectStatement(stmt))
	case *types.InsertStatement:
		out.WriteString(stringifyInsertStatement(stmt))
	}

	return out.String()
}

func stringifySelectStatement(stmt *types.SelectStatement) string {
	var sb strings.Builder

	sb.WriteString("SELECT ")
	if stmt.Distinct {
		sb.WriteString("DISTINCT ")
	}

	if len(stmt.Columns) == 1 && stmt.Columns[0] == "*" {
		sb.WriteString("* ")
	} else {
		for i, f := range stmt.Columns {
			sb.WriteString(f)
			if i < len(stmt.Columns)-1 {
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
			if len(c.ChainOp) > 0 && i > 0 {
				sb.WriteString(string(c.ChainOp))
				sb.WriteString(" ")
			}

			sb.WriteString(string(c.Column))
			sb.WriteString(" ")
			sb.WriteString(string(c.Operator))
			sb.WriteString(" ")

			if c.Value.Position != 0 {
				sb.WriteString(fmt.Sprintf("$%d", c.Value.Position))
			} else if c.Value.Value != nil {
				if _, err := strconv.Atoi(*c.Value.Value); err == nil {
					sb.WriteString(*c.Value.Value)
				} else {
					sb.WriteString(fmt.Sprintf("'%s'", *c.Value.Value))
				}
			} else {
				sb.WriteString("NULL")
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

func stringifyInsertStatement(stmt *types.InsertStatement) string {
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
	for i, bind := range stmt.Values {
		if bind.Position != 0 {
			sb.WriteString(fmt.Sprintf("$%d", bind.Position))
		} else if bind.Value != nil {
			if _, err := strconv.Atoi(*bind.Value); err == nil {
				sb.WriteString(*bind.Value)
			} else {
				sb.WriteString(fmt.Sprintf("'%s'", *bind.Value))
			}
		} else {
			sb.WriteString("NULL")
		}
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
