package generate_test

import (
	"os"
	"shogunc/cmd/generate"
	"strings"
	"testing"
)

func setUpGenerator(t *testing.T) *generate.Generator {
	t.Helper()
	configContents, err := os.ReadFile("../../shogunc.yml")
	if err != nil {
		t.Fatal(err)
	}

	gen := generate.NewGenerator()
	if err := gen.ParseConfig(configContents); err != nil {
		t.Fatal(err)
	}

	if err := gen.LoadSqlFiles(); err != nil {
		t.Fatalf("Error loading sql files: %v", err)
	}

	return gen
}

func setUpSchema(t *testing.T) []byte {
	t.Helper()

	configContents, err := os.ReadFile("../../shogunc.yml")
	if err != nil {
		t.Fatal(err)
	}

	gen := generate.NewGenerator()
	if err := gen.ParseConfig(configContents); err != nil {
		t.Fatal(err)
	}

	schema, err := gen.LoadSchema()
	if err != nil {
		t.Fatalf("Error loading sql files: %v", err)
	}

	return schema
}

func TestLoadConfig(t *testing.T) {
	configContents, err := os.ReadFile("../../shogunc.yml")
	if err != nil {
		t.Fatal(err)
	}

	gen := generate.NewGenerator()
	if err := gen.ParseConfig(configContents); err != nil {
		t.Fatal(err)
	}

	if gen.Driver == "" {
		t.Fatalf("Expected driver entry [ 'sqlite', 'postgres' ] Got: %s", gen.Driver)
	}

	if len(gen.QueryPath) == 0 {
		t.Fatalf("Expected queries entry Got: %d", len(gen.QueryPath))
	}

	if len(gen.SchemaPath) == 0 {
		t.Fatalf("Expected schema entry Got: %d", len(gen.SchemaPath))
	}
}

// func TestParseSqlFile(t *testing.T) {
// 	configContents, err := os.ReadFile("../../shogunc.yml")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	gen := generate.NewGenerator()
// 	if err := gen.ParseConfig(configContents); err != nil {
// 		t.Fatal(err)
// 	}
//
// 	if err := gen.LoadSqlFiles(); err != nil {
// 		t.Fatalf("Error loading sql files: %v", err)
// 	}
// }

// helper to create temp .sql file
func createTempSqlFile(t *testing.T, content string) *os.File {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "*.sql")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		t.Fatalf("failed to rewind temp file: %v", err)
	}
	return tmpFile
}

func TestParseSqlFile(t *testing.T) {
	sql := `-- name: getUserById :one
SELECT * FROM users WHERE id = $1;

-- name: listUsers :many
SELECT * FROM users;

-- name: GetUserByClerkId :one
SELECT id, clerk_id, first_name, last_name, email, phone, role, status, created_at
FROM users
WHERE first_name = 'hector'
LIMIT 1;

	`

	tmpFile := createTempSqlFile(t, sql)
	defer os.Remove(tmpFile.Name())

	gen := generate.NewGenerator()
	out, err := gen.ParseSqlFile(tmpFile)
	if err != nil {
		t.Fatalf("[GENERATE_TEST] error: %v", err)
	}

	t.Logf("[OUTPUT]: %s\n", out)

	if !strings.Contains(out, "func GetUserById()") {
		t.Errorf("[GENERATE_TEST] expected output to contain 'func GetUserById', got: %s", out)
	}
	if !strings.Contains(out, "func ListUsers()") {
		t.Errorf("[GENERATE_TEST] expected output to contain 'func ListUsers', got: %s", out)
	}
	if !strings.Contains(out, "func GetUserByClerkId()") {
		t.Errorf("[GENERATE_TEST] expected output to contain 'func ListUsers', got: %s", out)
	}
}
