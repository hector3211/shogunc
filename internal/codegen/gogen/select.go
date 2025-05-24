package gogen

import (
	"fmt"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
	"strings"
)

func generateSelectFunction(tagType utils.Type, query *sqlparser.SelectStatement) string {
	switch tagType {
	case utils.ONE:
		return generateSelectOne(query)
	case utils.MANY:
		return ""
	case utils.EXEC:
		return ""
	default:
		return ""
	}
}

func generateSelectOne(query *sqlparser.SelectStatement) string {
	var sb strings.Builder
	sb.WriteString("query := ")

	if len(query.Fields) == 1 && query.Fields[0] == "*" {
		sb.WriteString("Select('*')")
	} else {
		sb.WriteString("Select(")
		for idx, f := range query.Fields {
			// Note: Field names come from the lexer capitalized
			sb.WriteString(fmt.Sprintf("%q", strings.ToLower(f)))

			if idx < len(query.Fields)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString(")")
	}

	sb.WriteString(fmt.Sprintf(".From(%q)", query.TableName))

	sb.WriteString(".Where(")
	for idx, c := range query.Conditions {
		if c.Next != sqlparser.Illegal {
			sb.WriteString(fmt.Sprintf("%s,", shoguncNextOp(c.Next)))
		}
		sb.WriteString(shoguncConditionalOp(c))

		if idx < len(query.Conditions)-1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString(")")

	sb.WriteString(".Build()")
	return sb.String()
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
	strB.WriteString(fmt.Sprintf("Equal(%q, %v)", string(cond.Left), formatType(cond.Right)))
	return strB.String()
}

func shoguncNotEqualOp(cond sqlparser.Condition) string {
	strB := strings.Builder{}
	strB.WriteString(fmt.Sprintf("NotEqual(%q, %v)", string(cond.Left), formatType(cond.Right)))
	return strB.String()
}

func shoguncLessThanOp(cond sqlparser.Condition) string {
	strB := strings.Builder{}
	strB.WriteString(fmt.Sprintf("LessThan(%q, %v)", string(cond.Left), formatType(cond.Right)))
	return strB.String()
}

func shoguncGreaterThanOp(cond sqlparser.Condition) string {
	strB := strings.Builder{}
	strB.WriteString(fmt.Sprintf("GreaterThan(%q, %s)", string(cond.Left), formatType(cond.Right)))
	return strB.String()
}

func formatType(v any) string {
	switch v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
