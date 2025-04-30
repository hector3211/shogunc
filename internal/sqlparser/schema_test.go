package sqlparser

import (
	"reflect"
	"testing"
)

func TestSchemaParser_TableParsing(t *testing.T) {
	parser := NewSchemaParser()

	lines := []string{
		`CREATE TABLE IF NOT EXISTS "users" (`,
		`  "id" SERIAL PRIMARY KEY NOT NULL,`,
		`  "username" TEXT NOT NULL DEFAULT 'guest',`,
		`  "email" TEXT,`,
		`  "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`)`,
	}

	for _, line := range lines {
		if err := parser.ParseLine(line); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if len(parser.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(parser.Types))
	}

	table, ok := parser.Types[0].(TableType)
	if !ok {
		t.Fatalf("expected TableType, got %T", parser.Types[0])
	}

	if string(table.Name) != "users" {
		t.Errorf("expected table name 'users', got '%s'", table.Name)
	}

	if len(table.Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(table.Fields))
	}

	tests := []struct {
		name       string
		dataType   string
		isPrimary  bool
		notNull    bool
		hasDefault bool
	}{
		{"id", "SERIAL", true, true, false},
		{"username", "TEXT", false, true, true},
		{"email", "TEXT", false, false, false},
		{"created_at", "TIMESTAMP", false, false, true},
	}

	for i, expected := range tests {
		field := table.Fields[i]
		if field.Name != expected.name {
			t.Errorf("field[%d]: expected name %s, got %s", i, expected.name, field.Name)
		}
		// if field.DataType != expected.dataType {
		// 	t.Errorf("field[%d]: expected datatype %s, got %s", i, expected.dataType, field.DataType)
		// }
		if field.IsPrimary != expected.isPrimary {
			t.Errorf("field[%d]: expected primary %v, got %v", i, expected.isPrimary, field.IsPrimary)
		}
		if field.NotNull != expected.notNull {
			t.Errorf("field[%d]: expected notNull %v, got %v", i, expected.notNull, field.NotNull)
		}
		if (field.Default != nil) != expected.hasDefault {
			t.Errorf("field[%d]: expected hasDefault %v, got %v", i, expected.hasDefault, field.Default != nil)
		}
	}
}

func TestSchemaParser_EnumParsing(t *testing.T) {
	parser := NewSchemaParser()

	lines := []string{
		`CREATE TYPE "user_role" AS ENUM (`,
		`  "admin",`,
		`  "user",`,
		`  "guest"`,
		`)`,
	}

	for _, line := range lines {
		if err := parser.ParseLine(line); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if len(parser.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(parser.Types))
	}

	enum, ok := parser.Types[0].(EnumType)
	if !ok {
		t.Fatalf("expected EnumType, got %T", parser.Types[0])
	}

	if string(enum.Name) != "user_role" {
		t.Errorf("expected enum name 'user_role', got '%s'", enum.Name)
	}

	expectedValues := []string{"admin", "user", "guest"}

	if !reflect.DeepEqual(enum.Values, expectedValues) {
		t.Errorf("expected values %v, got %v", expectedValues, enum.Values)
	}
}

func TestSchemaParser_EmptyInput(t *testing.T) {
	parser := NewSchemaParser()

	lines := []string{
		`-- this is a comment`,
		`   `,
	}

	for _, line := range lines {
		if err := parser.ParseLine(line); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if len(parser.Types) != 0 {
		t.Fatalf("expected 0 types, got %d", len(parser.Types))
	}
}

func TestSchemaParser_InvalidFieldLine(t *testing.T) {
	parser := NewSchemaParser()

	// Simulate being inside a table
	parser.CurrentTable = &TableType{Name: []byte("broken")}
	parser.InTable = true

	err := parser.ParseLine(`"onlyonecolumn"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not crash or add a field
	if len(parser.CurrentTable.Fields) != 0 {
		t.Errorf("expected 0 fields parsed, got %d", len(parser.CurrentTable.Fields))
	}
}
