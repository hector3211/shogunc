package gogen

import (
	"bytes"
	"errors"
	"fmt"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
)

type FuncGenerator struct {
	Name       []byte
	tagType    utils.Type
	ReturnType any
}

func NewFuncGenerator(statementName []byte, statementTag utils.Type, statementReturnType any) *FuncGenerator {
	return &FuncGenerator{
		Name:       statementName,
		tagType:    statementTag,
		ReturnType: statementReturnType,
	}
}

func (g FuncGenerator) GenerateFunction(statement sqlparser.Node) (string, error) {
	var sb bytes.Buffer
	var returnType string
	if g.tagType == utils.MANY {
		returnType = "[]"
	}

	// fmt.Printf("[BUILDER] data type GenerateFunction: %v\n\n", g.ReturnType)
	if g.tagType != utils.EXEC {
		switch t := g.ReturnType.(type) {
		case *sqlparser.TableType:
			returnType += utils.Capitalize(t.Name)
		default:
			return "", errors.New("[BUILDER] failed infering type")
		}
	}

	sb.WriteString(fmt.Sprintf("func %s(ctx context.Context) %s {\n", g.Name, returnType))

	switch stmt := statement.(type) {
	case *sqlparser.SelectStatement:
		sb.WriteString(generateSelectFunction(g.tagType, stmt))
	default:
		return "", errors.New("[BUILDER] fialed parsing SQL statement")
	}

	sb.WriteString("\n}")
	return sb.String(), nil
}

// func (g FuncGenerator) tab() string {
// 	return "\t"
// }

// func (g FuncGenerator) newLine() string {
// 	return "\n"
// }
