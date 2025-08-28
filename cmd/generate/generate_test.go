package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerator_Execute(t *testing.T) {
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalCwd)

	tmp := t.TempDir()

	// Write schema.sql
	schema := `CREATE TABLE IF NOT EXISTS "users" (
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
`
	if err := os.WriteFile(filepath.Join(tmp, "schema.sql"), []byte(schema), 0644); err != nil {
		t.Fatal(err)
	}

	// Make queries dir & write user.sql
	queriesDir := filepath.Join(tmp, "queries")
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		t.Fatal(err)
	}

	query := `-- name: GetUser :one
SELECT * FROM users WHERE id = $1;`
	if err := os.WriteFile(filepath.Join(queriesDir, "user.sql"), []byte(query), 0644); err != nil {
		t.Fatal(err)
	}

	// Write config file
	config := fmt.Sprintf(`
  sql:
    schema: schema.sql
    queries: queries
    driver: sqlite3
    output: %s/output
  `, tmp)
	if err := os.WriteFile(filepath.Join(tmp, "shogunc.yml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	// Change into temp directory
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	gen := NewGenerator()
	if err := gen.Execute(tmp); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Asserts
	expectedOutput := filepath.Join(tmp, "output")
	if gen.Config.Sql.Driver != SQLITE {
		t.Errorf("Expected driver 'sqlite3', got '%s'", gen.Config.Sql.Driver)
	}
	if gen.Config.Sql.Queries != "queries" {
		t.Errorf("Expected queries path 'queries', got '%s'", gen.Config.Sql.Queries)
	}
	if gen.Config.Sql.Schema != "schema.sql" {
		t.Errorf("Expected schema path 'schema.sql', got '%s'", gen.Config.Sql.Schema)
	}
	if gen.Config.Sql.Output != expectedOutput {
		t.Errorf("Expected output path '%s', got '%s'", expectedOutput, gen.Config.Sql.Output)
	}

	// Check that output directory exists
	if _, err := os.Stat(expectedOutput); os.IsNotExist(err) {
		t.Fatalf("Expected output directory to exist, got error: %v", err)
	}

	// Check that schema.sql.go exists and contains expected content
	schemaFile := filepath.Join(expectedOutput, "schema.sql.go")
	out, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatalf("Expected schema.sql.go file to exist, got error: %v", err)
	}

	fmt.Println("schema.sql.go content:")
	fmt.Println(string(out))

	if !strings.Contains(string(out), "type User") {
		t.Errorf("Expected generated schema output to contain 'type User'\nOutput: %s", string(out))
	}

	// Check that user.sql.go exists and contains expected content
	userFile := filepath.Join(expectedOutput, "user.sql.go")
	userOut, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatalf("Expected user.sql.go file to exist, got error: %v", err)
	}

	fmt.Println("user.sql.go content:")
	fmt.Println(string(userOut))

	if !strings.Contains(string(userOut), "func GetUser") {
		t.Errorf("Expected generated user output to contain 'func GetUser'\nOutput: %s", string(userOut))
	}
}
