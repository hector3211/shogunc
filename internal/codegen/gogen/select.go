package gogen

import (
	"shogunc/internal/sqlparser"
	"strings"
)

type GoSelectFuncGenerator struct {
	b strings.Builder
}

func (g *GoSelectFuncGenerator) GenerateFunc(query *sqlparser.SelectStatement) string {
	return ""
}
