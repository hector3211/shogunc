package utils

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
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

func ToProperPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// First convert to PascalCase (which may include underscores)
	pascal := ToPascalCase(s)

	// Then remove underscores and capitalize the next letter
	var result strings.Builder
	for i, r := range pascal {
		if r == '_' && i+1 < len(pascal) {
			// Skip underscore and capitalize next character
			continue
		} else if i > 0 && pascal[i-1] == '_' {
			// This character follows an underscore, capitalize it
			result.WriteRune(r - 'a' + 'A')
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

// GenerateTestFiles creates test queries and config files for development
func GenerateTestFiles() error {
	if err := generateTestQueries(); err != nil {
		return fmt.Errorf("failed to generate test queries: %w", err)
	}
	if err := generateTestConfig(); err != nil {
		return fmt.Errorf("failed to generate test config: %w", err)
	}
	return nil
}

func generateTestQueries() error {
	queriesDir := "queries"
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		return err
	}

	userSQL := `-- name: GetUser :one
SELECT id, first_name, last_name, email, phone, role, created_at
FROM users
WHERE id = $1;

-- name: GetUserByClerkId :one
SELECT id, clerk_id, first_name, last_name, email, phone, role, status, created_at
FROM users
WHERE first_name = $1
LIMIT 1;

-- name: GetUserByClerkIdTwo :one
SELECT id, clerk_id, first_name, created_at
FROM users
WHERE clerk_id = $1 AND first_name = $2
LIMIT 1 OFFSET 20;

-- name: GetAllUsers :many
SELECT id, first_name, last_name, email, role
FROM users
WHERE status = $1;
`

	return os.WriteFile(filepath.Join(queriesDir, "user.sql"), []byte(userSQL), 0644)
}

func generateTestConfig() error {
	config := `sql:
  schema: schema.sql
  queries: queries
  driver: sqlite3
  output: /tmp/internal/db/generated
#   gen:
#     go:
#       package: "authors"
#       out: "postgresql"
#   database:
#     managed: true
#   rules:
#     - sqlc/db-prepare
# - schema: "mysql/schema.sql"
#   queries: "mysql/query.sql"
#   engine: "mysql"
#   gen:
#     go:
#       package: "authors"
#       out: "mysql"
`

	return os.WriteFile("shogunc.yml", []byte(config), 0644)
}
