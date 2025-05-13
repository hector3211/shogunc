package utils

import "strings"

type Type string // exec | one | many

const (
	EXEC Type = "exec"
	ONE  Type = "one"
	MANY Type = "many"
)

type TagType struct {
	Name []byte
	Type Type
}

func StrPtr(s string) *string {
	return &s
}

func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
