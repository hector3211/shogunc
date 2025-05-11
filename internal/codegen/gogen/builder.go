package gogen

import (
	"bytes"
	"fmt"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
)

type Type string

const (
	EXEC Type = "exec"
	ONE  Type = "one"
	MANY Type = "many"
)

type TagType struct {
	Name []byte
	Type Type
}

type GoFuncGenerator struct {
	Name       []byte
	tagType    utils.Type
	ReturnType []byte
}

func NewGoFuncGenerator(statementName []byte, statementTag utils.Type) *GoFuncGenerator {
	return &GoFuncGenerator{
		Name:       statementName,
		tagType:    statementTag,
		ReturnType: []byte{},
	}
}

func (g *GoFuncGenerator) GenerateFunction(statement sqlparser.Node) string {
	var sb bytes.Buffer
	sb.WriteString(fmt.Sprintf("func %s() {\n", g.Name))
	g.NewLine()
	// f.Tab()
	switch stmt := statement.(type) {
	case *sqlparser.SelectStatement:
		sb.WriteString(generateSelectFunction(g.tagType, stmt))
	default:
		sb.WriteString("Failed parsing statement")
	}
	sb.WriteString(g.NewLine())
	sb.WriteString("}")

	// return sb.Bytes()
	return sb.String()
}

func (g *GoFuncGenerator) Tab() string {
	return "\t"
}

func (g *GoFuncGenerator) NewLine() string {
	return "\n"
}
