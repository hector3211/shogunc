package utils

import (
	"fmt"
	"go/ast"
	"strings"
)

func StrPtr(s string) *string {
	return &s
}

func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func FormatType(v any) string {
	switch v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func TypeToString(resultType ast.Expr) string {
	var typeName string
	if ident, ok := resultType.(*ast.Ident); ok {
		typeName = ident.Name
	} else if arrayType, ok := resultType.(*ast.ArrayType); ok {
		if eltIdent, ok := arrayType.Elt.(*ast.Ident); ok {
			typeName = "[]" + eltIdent.Name
		}
	}
	return typeName
}
