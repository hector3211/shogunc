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

func (f *GoFuncGenerator) GenerateFunction(statement sqlparser.Node) string {
	f.sb.WriteString(fmt.Sprintf("func %s() {", f.funcName))
	f.NewLine()
	switch stmt := statement.(type) {
	case *sqlparser.SelectStatement:
		f.sb.WriteString(GenerateSelectFunction(f.tagType, stmt))
	default:
		f.sb.WriteString("Failed parsing statement")
	}
	f.NewLine()
	f.sb.WriteString("}")

	return f.sb.String()
}

func (f *GoFuncGenerator) NewLine() {
	f.sb.WriteString("\n")
}
