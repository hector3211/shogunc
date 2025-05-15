package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	configContents, err := os.ReadFile("../../shogunc.yml")
	if err != nil {
		t.Fatal(err)
	}

	gen := NewGenerator()
	if err := gen.parseConfig(configContents); err != nil {
		t.Fatal(err)
	}

	if gen.Config.Sql.Driver == "" {
		t.Fatalf("Expected driver entry [ 'sqlite', 'postgres' ] Got: %s", gen.Config.Sql.Driver)
	}

	if gen.Config.Sql.Queries == "" {
		t.Fatalf("Expected queries entry Got: %s", gen.Config.Sql.Queries)
	}

	if gen.Config.Sql.Schema == "" {
		t.Fatalf("Expected schema entry Got: %s", gen.Config.Sql.Schema)
	}
}

func TestParseSqlFile(t *testing.T) {
	tmpDir := t.TempDir()

	schemaContent := `
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

CREATE TABLE IF NOT EXISTS "lockers" (
    "id"          UUID PRIMARY KEY,
    "access_code" VARCHAR,
    "in_use"      BOOLEAN NOT NULL DEFAULT FALSE,
    "user_id"     BIGINT
);

	`
	schemaPath := filepath.Join(tmpDir, "schema.sql")
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	if err != nil {
		t.Fatalf("failed writing schema: %v", err)
	}

	sqlContent := `-- name: GetUserById :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users;

`
	sqlDir := filepath.Join(tmpDir, "queries")
	os.Mkdir(sqlDir, 0755)
	sqlFile := filepath.Join(sqlDir, "query.sql")
	err = os.WriteFile(sqlFile, []byte(sqlContent), 0644)
	if err != nil {
		t.Fatalf("failed writing sql file: %v", err)
	}

	gen := NewGenerator()
	gen.Config.Sql.Schema = filepath.Join("schema.sql")
	gen.Config.Sql.Queries = filepath.Join("queries")
	gen.Config.Sql.Driver = SQLITE

	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(tmpDir)

	err = gen.LoadSchema()
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	if _, ok := gen.Types["users"]; !ok {
		t.Fatalf("[GENERATE_TEST] expected type Users to be loaded from schema")
	}

	file, err := os.Open(sqlFile)
	if err != nil {
		t.Fatalf("[GENERATE_TEST] failed to open sql file: %v", err)
	}
	defer file.Close()

	out, err := gen.parseSqlFile(file)
	if err != nil {
		t.Fatalf("[GENERATE_TEST] ParseSqlFile failed: %v", err)
	}
	// t.Logf("[OUTPUT]: %s\n", out)

	if !strings.Contains(out, "func GetUserById(ctx context.Context)") {
		t.Errorf("[GENERATE_TEST] expected output to contain 'func GetUserById', got: %s", out)
	}
	if !strings.Contains(out, "func ListUsers(ctx context.Context)") {
		t.Errorf("[GENERATE_TEST] expected output to contain 'func ListUsers', got: %s", out)
	}
	// if !strings.Contains(out, "func GetUserByClerkId(ctx context.Context)") {
	// 	t.Errorf("[GENERATE_TEST] expected output to contain 'func ListUsers', got: %s", out)
	// }
}

func TestHasConfig(t *testing.T) {
	tmpDir := t.TempDir()
	yml := `
sql:
    queries: queries
    schema: schema.sql
    driver: sqlite3
`
	_ = createTempFile(t, tmpDir, "shogunc.yml", yml)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	gen := NewGenerator()
	if !gen.hasConfig() {
		t.Fatal("Expected HasConfig to return true")
	}
}

func TestParseConfig(t *testing.T) {
	gen := NewGenerator()
	yml := `
sql:
    queries: queries
    schema: schema.sql
    driver: sqlite3
`
	err := gen.parseConfig([]byte(yml))
	if err != nil {
		t.Fatalf("Expected ParseConfig to succeed: %v", err)
	}
	if string(gen.Config.Sql.Queries) != "queries" {
		t.Errorf("Expected query path 'sql', got '%s'", gen.Config.Sql.Queries)
	}
	if string(gen.Config.Sql.Schema) != "schema.sql" {
		t.Errorf("Expected schema path 'schema.sql', got '%s'", gen.Config.Sql.Schema)
	}
	if gen.Config.Sql.Driver != SQLITE {
		t.Errorf("Expected driver sqlite3, got %s", gen.Config.Sql.Driver)
	}
}

func TestExtractSqlBlocks(t *testing.T) {
	sql := `-- name: GetUserById :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users;
`
	file := filepath.Join(t.TempDir(), "test.sql")
	os.WriteFile(file, []byte(sql), 0644)
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gen := NewGenerator()
	blocks, err := gen.extractSqlBlocks(f, "test.sql")
	if err != nil {
		t.Fatalf("extractSqlBlocks failed: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("Expected 2 query blocks, got %d", len(blocks))
	}
	if blocks[0].Name != "GetUserById" {
		t.Errorf("Expected first block name 'GetUserById', got '%s'", blocks[0].Name)
	}
	if !strings.Contains(blocks[0].SQL, "SELECT * FROM users") {
		t.Errorf("Unexpected SQL in first block")
	}
}

func createTempFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(contents), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	return path
}
