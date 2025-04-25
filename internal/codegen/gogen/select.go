package gogen

import (
	"fmt"
	"shogunc/cmd/generate"
	"shogunc/internal/sqlparser"
	"strings"
)

type GoSelectFuncGenerator struct {
	b        strings.Builder
	funcName string
	tagType  generate.Type
}

func NewGoSelectFuncGenerator(funcName string, tagType generate.Type) *GoSelectFuncGenerator {
	return &GoSelectFuncGenerator{
		funcName: funcName,
		tagType:  tagType,
	}
}

// TODO: finish
func (g *GoSelectFuncGenerator) GenerateFunc(body string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("func %s() ", g.funcName))
	switch g.tagType {
	case generate.ONE, generate.MANY:
		sb.WriteString("")
	case generate.EXEC:
		sb.WriteString("")
	default:
		sb.WriteString("")
	}

	return sb.String()
}

func (g *GoSelectFuncGenerator) GenerateQueryOne(tag *generate.Query, query *sqlparser.SelectStatement) string {
	g.b.WriteString("query := ")

	if len(query.Fields) == 0 {
		g.b.WriteString("Select('*')")
	} else {
		g.b.WriteString("Select(")
		for idx, f := range query.Fields {
			g.b.WriteString(f)

			if idx < len(query.Fields)-1 {
				g.b.WriteString(",")
			}
		}
		g.b.WriteString(")")
	}

	g.b.WriteString(fmt.Sprintf(".From(%s)", query.TableName))

	g.b.WriteString(".Where(")
	for idx, c := range query.Conditions {
		if c.Next != sqlparser.Illegal {
			g.b.WriteString(fmt.Sprintf("%s,", shoguncNextOp(c.Next)))
		}
		g.b.WriteString(shoguncConditionalOp(c))

		if idx < len(query.Conditions)-1 {
			g.b.WriteString(",")
		}
	}
	g.b.WriteString(")")

	g.b.WriteString(".Build()")
	return g.b.String()
}

func shoguncNextOp(nextOp sqlparser.LogicalOp) string {
	switch nextOp {
	case sqlparser.And:
		return "And()"
	case sqlparser.Or:
		return "Or()"
	}
	return ""
}

func shoguncConditionalOp(cond sqlparser.Condition) string {
	switch cond.Operator {
	case sqlparser.EQUAL:
		return shoguncEqualOp(cond)
	case sqlparser.NOTEQUAL:
		return shoguncNotEqualOp(cond)
	case sqlparser.LESSTHAN:
		return shoguncLessThanOp(cond)
	case sqlparser.GREATERTHAN:
		return shoguncGreaterThanOp(cond)
	}

	return ""
}

func shoguncEqualOp(cond sqlparser.Condition) string {
	strB := strings.Builder{}
	strB.WriteString(fmt.Sprintf("Equal(%s, %v)", string(cond.Left), cond.Right))
	return strB.String()
}

func shoguncNotEqualOp(cond sqlparser.Condition) string {
	strB := strings.Builder{}
	strB.WriteString(fmt.Sprintf("NotEqual(%s, %v)", string(cond.Left), cond.Right))
	return strB.String()
}

func shoguncLessThanOp(cond sqlparser.Condition) string {
	strB := strings.Builder{}
	strB.WriteString(fmt.Sprintf("LessThan(%s, %v)", string(cond.Left), cond.Right))
	return strB.String()
}

func shoguncGreaterThanOp(cond sqlparser.Condition) string {
	strB := strings.Builder{}
	strB.WriteString(fmt.Sprintf("GreaterThan(%s, %v)", string(cond.Left), cond.Right))
	return strB.String()
}
