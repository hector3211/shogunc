package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	gen := NewGenerator()
	if err := gen.Execute(); err != nil {
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
	if gen.Config.Sql.Output == "" {
		t.Fatalf("Expected ouput entry Got: %s", gen.Config.Sql.Output)
	}
}

func TestParseSqlFile(t *testing.T) {
	outputPath := filepath.Join("/tmp", "generated.sql.go")

	gen := NewGenerator()
	gen.Config.Sql.Schema = "../../schema.sql"
	gen.Config.Sql.Queries = "../../queries"
	gen.Config.Sql.Driver = SQLITE
	gen.Config.Sql.Output = outputPath

	err := gen.LoadSchema()
	if err != nil {
		t.Fatalf("[GENERATE_TEST] LoadSchema error: %v", err)
	}

	if _, ok := gen.Types["users"]; !ok {
		t.Fatalf("[GENERATE_TEST] expected type Users to be loaded from schema")
	}

	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(contents), "type Users") {
		t.Fatalf("[GENERATE_TEST] generated file has no type User\n contents: %s\n\npath: %s", contents, outputPath)
	}

	file, err := os.Open("../../queries/user.sql")
	if err != nil {
		t.Fatalf("[GENERATE_TEST] ParseSqlFile failed: %v", err)
	}
	defer file.Close()

	err = gen.parseSqlFile(file)
	if err != nil {
		t.Fatalf("[GENERATE_TEST] ParseSqlFile failed: %v", err)
	}

	contentsGenFuncs, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(contentsGenFuncs), "func GetUser(ctx context.Context) Users") {
		t.Fatalf("[GENERATE_TEST] generated file has no func\n contents: %s", contents)
	}
}

func TestParseConfig(t *testing.T) {
	gen := NewGenerator()
	err := gen.Execute()
	if err != nil {
		t.Fatalf("Expected ParseConfig to succeed: %v", err)
	}
	if string(gen.Config.Sql.Queries) != "queries" {
		t.Errorf("Expected query path 'queries', got '%s'", gen.Config.Sql.Queries)
	}
	if string(gen.Config.Sql.Schema) != "schema.sql" {
		t.Errorf("Expected schema path 'schema.sql', got '%s'", gen.Config.Sql.Schema)
	}
	if string(gen.Config.Sql.Output) != "/tmp/generated.sql.go" {
		t.Errorf("Expected output path '/tmp/generated.sql.go', got '%s'", gen.Config.Sql.Output)
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
