package gogen

import (
	"bytes"
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
	var buffer bytes.Buffer
	buffer.WriteString("query := ")

	if len(query.Fields) == 1 && query.Fields[0] == "*" {
		buffer.WriteString("Select('*')")
	} else {
		buffer.WriteString("Select(")
		for idx, f := range query.Fields {
			// Note: Field names come from the lexer capitalized
			buffer.WriteString(fmt.Sprintf("%q", strings.ToLower(f)))

			if idx < len(query.Fields)-1 {
				buffer.WriteString(",")
			}
		}
		buffer.WriteString(")")
	}

	buffer.WriteString(fmt.Sprintf(".From(%q)", query.TableName))

	buffer.WriteString(".Where(")
	for idx, c := range query.Conditions {
		if c.Next != sqlparser.Illegal {
			buffer.WriteString(fmt.Sprintf("%s,", shoguncNextOp(c.Next)))
		}
		buffer.WriteString(shoguncConditionalOp(c))

		if idx < len(query.Conditions)-1 {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString(")")

	buffer.WriteString(".Build()")
	return buffer.String()
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
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Equal(%q, %v)", string(cond.Left), formatType(cond.Right)))
	return buffer.String()
}

func shoguncNotEqualOp(cond sqlparser.Condition) string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("NotEqual(%q, %v)", string(cond.Left), formatType(cond.Right)))
	return buffer.String()
}

func shoguncLessThanOp(cond sqlparser.Condition) string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("LessThan(%q, %v)", string(cond.Left), formatType(cond.Right)))
	return buffer.String()
}

func shoguncGreaterThanOp(cond sqlparser.Condition) string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("GreaterThan(%q, %s)", string(cond.Left), formatType(cond.Right)))
	return buffer.String()
}

func formatType(v any) string {
	switch v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
