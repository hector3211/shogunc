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

func generateTestSchema() error {
	schemaSQL := `CREATE TYPE "Complaint_Category" AS ENUM (
    'maintenance',
    'noise',
    'security',
    'parking',
    'neighbor',
    'trash',
    'internet',
    'lease',
    'natural_disaster',
    'other'
);
CREATE TYPE "Status" AS ENUM (
    'open',
    'in_progress',
    'resolved',
    'closed'
);
CREATE TYPE "Type" AS ENUM (
    'lease_agreement',
    'amendment',
    'extension',
    'termination',
    'addendum'
);
CREATE TYPE "Lease_Status" AS ENUM (
    'draft',
    'pending_approval',
    'active',
    'expired',
    'terminated',
    'renewed'
);
CREATE TYPE "Compliance_Status" AS ENUM (
    'pending_review',
    'compliant',
    'non_compliant',
    'exempted'
);
CREATE TYPE "Work_Category" AS ENUM (
    'plumbing',
    'electric',
    'carpentry',
    'hvac',
    'other'
);

CREATE TYPE "Role" AS ENUM (
    'tenant',
    'landlord',
    'admin',
    'staff'
);

CREATE TYPE "Account_Status" AS ENUM (
    'active',
    'inactive',
    'suspended',
    'pending'
);

CREATE TABLE IF NOT EXISTS "users" (
    "id"          UUID PRIMARY KEY,
    "clerk_id"    TEXT UNIQUE                    NOT NULL,
    "first_name"  VARCHAR                        NOT NULL,
    "last_name"   VARCHAR                        NOT NULL,
    "email"       VARCHAR                        NOT NULL,
    "phone"       VARCHAR                        NULL,
    "unit_number" SMALLINT                       NULL,
    "role"        "Role"                         NOT NULL DEFAULT "Role" 'tenant',
    "status"      "Account_Status"               NOT NULL DEFAULT "Account_Status" 'active',
    "last_login"  TIMESTAMP NOT NULL,
    "updated_at"  TIMESTAMP          DEFAULT now(),
    "created_at"  TIMESTAMP          DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "parking_permits" (
    "id"            UUID NOT NULL PRIMARY KEY,
    "permit_number" BIGINT NOT NULL,
    "created_by"    SMALLINT NOT NULL,
    "updated_at"    TIMESTAMP DEFAULT now(),
    "expires_at"    TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "lockers" (
    "id"          UUID PRIMARY KEY,
    "access_code" VARCHAR,
    "in_use"      BOOLEAN NOT NULL DEFAULT false,
    "user_id"     BIGINT
);
`

	return os.WriteFile("schema.sql", []byte(schemaSQL), 0644)
}

// GenerateTestFiles creates test queries, config, and schema files for development
func GenerateTestFiles() error {
	if err := generateTestSchema(); err != nil {
		return fmt.Errorf("failed to generate test schema: %w", err)
	}
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
