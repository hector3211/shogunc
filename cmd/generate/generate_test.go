package generate_test

import (
	"os"
	"shogunc/cmd/generate"
	"testing"
)

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

func TestParseSqlFile(t *testing.T) {
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
}
