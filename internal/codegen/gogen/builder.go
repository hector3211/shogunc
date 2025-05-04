package gogen

import (
	"fmt"
	"shogunc/cmd/generate"
	"shogunc/internal/sqlparser"
	"strings"
)

type GoFuncGenerator struct {
	funcName []byte
	tagType  generate.Type
	sb       strings.Builder
}

func NewGoFuncGenerator(query generate.Query) *GoFuncGenerator {
	return &GoFuncGenerator{
		funcName: query.Name,
		tagType:  query.Type,
	}
}

func (g *GoFuncGenerator) GenerateFunction(statement sqlparser.Node) string {
	g.sb.WriteString(fmt.Sprintf("func %s() {", g.funcName))
	g.NewLine()
	// f.Tab()
	switch stmt := statement.(type) {
	case *sqlparser.SelectStatement:
		g.sb.WriteString(GenerateSelectFunction(g.tagType, stmt))
	default:
		g.sb.WriteString("Failed parsing statement")
	}
	g.NewLine()
	g.sb.WriteString("}")

	return g.sb.String()
}

func (g *GoFuncGenerator) Tab() {
	g.sb.WriteString("\t")
}

func (g *GoFuncGenerator) NewLine() {
	g.sb.WriteString("\n")
}
