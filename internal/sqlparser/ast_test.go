package sqlparser

import (
	"fmt"
	"os"
	"shogunc/cmd/generate"
	"testing"
	"time"
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

func TestAstLoadTokens(t *testing.T) {
	gen := setUpGenerator(t)

	for _, file := range gen.Queries {
		lexer := NewLexer(string(file.SQL))
		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}

			// t.Logf("Token Type: %+v\n", token.Type)
			// t.Logf("Token Literal: %s\n", token.Literal)
		}
	}
}

func TestAstStatementParsing(t *testing.T) {
	gen := setUpGenerator(t)

	for _, file := range gen.Queries {
		t.Run(fmt.Sprintf("Parsing: %s", file.Name), func(t *testing.T) {
			lexer := NewLexer(string(file.SQL))
			parser := NewAst(lexer)

			err := parser.Parse()
			if err != nil {
				t.Errorf("parse error: %v", err)
				return
			}

			for _, n := range parser.Statements {
				switch stmt := n.(type) {
				case *SelectStatement:
					t.Logf("SELECT - table: %s, fields: %v", stmt.TableName, stmt.Fields)
					// for _, c := range stmt.Conditions {
					// 	t.Logf("Condition - Ident: %s, Operator: %s, Value: %v\n", c.Left, c.Operator, c.Right)
					// }
					t.Logf("LIMIT: %d, OFFSET: %d", stmt.Limit, stmt.Offset)

				case *InsertStatement:
					t.Logf("INSERT - table: %s, values: %d", stmt.TableName, stmt.Values)
					// for _, col := range stmt.Columns {
					// 	t.Logf("Column: %s", string(col))
					// }

				default:
					t.Errorf("unknown AST node type: %T", n)
				}
			}
		})
	}
}

func TestSchemaParse(t *testing.T) {
	lexer := NewLexer(string(setUpSchema(t)))
	parser := NewAst(lexer)
	if err := parser.ParseSchema(); err != nil {
		t.Errorf("[PARSER_TEST] parse schema error: %v", err)
		return
	}

	if len(parser.Statements) <= 1 {
		t.Fatalf("[PARSER_TEST] statements, got %d", len(parser.Statements))
	}

	// Tracking for expected items
	expectedEnums := map[string]int{
		"Complaint_Category": 10,
		"Status":             4,
		"Type":               5,
		"Lease_Status":       6,
		"Compliance_Status":  4,
		"Work_Category":      5,
		"Account_Status":     3,
		"Role":               2,
	}
	expectedTables := map[string]int{
		"parking_permits": 5,
		"complaints":      9,
		"work_orders":     9,
		"users":           12,
		"apartments":      10,
		"lockers":         4,
	}

	for _, n := range parser.Statements {
		switch s := n.(type) {
		case *TableType:
			name := s.Name
			if expected, ok := expectedTables[name]; ok {
				if len(s.Fields) != expected {
					t.Errorf("table %s: expected %d fields, got %d\n", name, expected, len(s.Fields))
				}
			}
			t.Logf("Parsed table: %s (%d columns)\n", name, len(s.Fields))

		case *EnumType:
			name := s.Name
			if expected, ok := expectedEnums[name]; ok {
				if len(s.Values) != expected {
					t.Errorf("enum %s: expected %d values, got %d", name, expected, len(s.Values))
				}
			}
			t.Logf("Parsed enum: %s (%d values)", name, len(s.Values))

		default:
			t.Errorf("unknown statement type: %T", s)
		}
	}
}

func TestSchemaTypes(t *testing.T) {
	lexer := NewLexer(string(setUpSchema(t)))
	parser := NewAst(lexer)
	if err := parser.ParseSchema(); err != nil {
		t.Errorf("[PARSER_TEST] parse schema error: %v", err)
		return
	}

	if len(parser.Statements) <= 1 {
		t.Fatalf("[PARSER_TEST] statements, got %d", len(parser.Statements))
	}

	nowOne := time.Now().Format("2006-01-02 15:04:05")
	inUseFalse := fmt.Sprintf("%v", false)
	expectedTypes := map[string][]Field{
		"parking_permits": {
			{
				Name: "id",
				DataType: Token{
					Type:    IDENT,
					Literal: "UUID",
				},
				NotNull:   true,
				Default:   nil,
				IsPrimary: true,
				IsUnique:  false,
			},
			{
				Name: "permit_number",
				DataType: Token{
					Type:    IDENT,
					Literal: "BIGINT",
				},
				NotNull:   true,
				Default:   nil,
				IsPrimary: false,
				IsUnique:  false,
			},
			{
				Name: "created_by",
				DataType: Token{
					Type:    IDENT,
					Literal: "SMALLINT",
				},
				NotNull:   true,
				Default:   nil,
				IsPrimary: false,
				IsUnique:  false,
			},
			{
				Name: "updated_at",
				DataType: Token{
					Type:    IDENT,
					Literal: "TIMESTAMP",
				},
				NotNull:   false,
				Default:   &nowOne,
				IsPrimary: false,
				IsUnique:  false,
			},
			{
				Name: "expires_at",
				DataType: Token{
					Type:    IDENT,
					Literal: "TIMESTAMP",
				},
				NotNull:   true,
				Default:   nil,
				IsPrimary: false,
				IsUnique:  false,
			},
		},
		"lockers": {
			{
				Name: "id",
				DataType: Token{
					Type:    IDENT,
					Literal: "UUID",
				},
				NotNull:   false,
				Default:   nil,
				IsPrimary: true,
				IsUnique:  false,
			},
			{
				Name: "access_code",
				DataType: Token{
					Type:    IDENT,
					Literal: "VARCHAR",
				},
				NotNull:   false,
				Default:   nil,
				IsPrimary: false,
				IsUnique:  false,
			},
			{
				Name: "in_use",
				DataType: Token{
					Type:    IDENT,
					Literal: "BOOLEAN",
				},
				NotNull:   true,
				Default:   &inUseFalse,
				IsPrimary: false,
				IsUnique:  false,
			},
			{
				Name: "user_id",
				DataType: Token{
					Type:    IDENT,
					Literal: "BIGINT",
				},
				NotNull:   false,
				Default:   nil,
				IsPrimary: false,
				IsUnique:  false,
			},
		},
	}

	for _, n := range parser.Statements {
		switch s := n.(type) {
		case *TableType:
			name := s.Name
			fields, ok := expectedTypes[name]
			if !ok {
				t.Errorf("unexpected table name: %s", name)
				continue
			}

			if len(fields) != len(s.Fields) {
				t.Errorf("table %s: expected %d fields, got %d", name, len(fields), len(s.Fields))
				continue
			}

			for idx, f := range s.Fields {
				if fields[idx].Name != f.Name {
					t.Errorf("table %s: expected field name '%s', got '%s'", name, fields[idx].Name, f.Name)
				}
				if fields[idx].DataType != f.DataType {
					t.Errorf("table %s: expected field '%s' type %v, got type %v", name, fields[idx].Name, fields[idx].DataType, f.DataType)
				}
			}
		}
	}
}

func TestStringifySelectStatement(t *testing.T) {
	tests := []struct {
		stmt *SelectStatement
		want string
	}{
		{
			stmt: &SelectStatement{
				Fields:    []string{"id", "name"},
				TableName: []byte("users"),
				Conditions: []Condition{
					{Left: []byte("age"), Operator: ">", Right: 30},
				},
				Limit:  10,
				Offset: 5,
			},
			want: "SELECT id, name FROM users WHERE age > 30 LIMIT 10 OFFSET 5;",
		},
		{
			stmt: &SelectStatement{
				Fields:     []string{"*"},
				TableName:  []byte("orders"),
				Conditions: nil,
				Limit:      0,
				Offset:     0,
			},
			want: "SELECT * FROM orders;",
		},
		{
			stmt: &SelectStatement{
				Fields:    []string{"id", "name"},
				TableName: []byte("products"),
				Conditions: []Condition{
					{Left: []byte("price"), Operator: ">=", Right: 100},
				},
				Distinct: true,
				Limit:    0,
				Offset:   0,
			},
			want: "SELECT DISTINCT id, name FROM products WHERE price >= 100;",
		},
	}

	for _, tt := range tests {
		t.Run("SelectStatement", func(t *testing.T) {
			got := stringifySelectStatement(tt.stmt)
			if got != tt.want {
				t.Errorf("stringifySelectStatement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringifyInsertStatement(t *testing.T) {
	tests := []struct {
		stmt *InsertStatement
		want string
	}{
		{
			stmt: &InsertStatement{
				TableName:       []byte("users"),
				Columns:         [][]byte{[]byte("name"), []byte("age")},
				Values:          []int{30, 25},
				ReturningFields: [][]byte{[]byte("id")},
				InsertMode:      []byte("OR REPLACE"),
			},
			want: "INSERT OR REPLACE INTO users (name, age) VALUES (30, 25) RETURNING id;",
		},
		{
			stmt: &InsertStatement{
				TableName:       []byte("products"),
				Columns:         [][]byte{[]byte("id"), []byte("name"), []byte("price")},
				Values:          []int{1, 100, 25},
				ReturningFields: nil,
				InsertMode:      nil,
			},
			want: "INSERT INTO products (id, name, price) VALUES (1, 100, 25);",
		},
		{
			stmt: &InsertStatement{
				TableName:       []byte("orders"),
				Columns:         [][]byte{[]byte("id"), []byte("quantity")},
				Values:          []int{10, 2},
				ReturningFields: [][]byte{[]byte("order_id")},
				InsertMode:      nil,
			},
			want: "INSERT INTO orders (id, quantity) VALUES (10, 2) RETURNING order_id;",
		},
		{
			stmt: &InsertStatement{
				TableName:       []byte("customers"),
				Columns:         nil,
				Values:          nil,
				ReturningFields: nil,
				InsertMode:      nil,
			},
			want: "INSERT INTO customers VALUES ();",
		},
	}

	for _, tt := range tests {
		t.Run("InsertStatement", func(t *testing.T) {
			got := stringifyInsertStatement(tt.stmt)
			if got != tt.want {
				t.Errorf("got = %v want %v", got, tt.want)
			}
		})
	}
}

func TestStringifyTableType(t *testing.T) {
	table := &TableType{
		Name: "users",
		Fields: []Field{
			{
				Name:      "id",
				DataType:  Token{Type: UUID, Literal: "UUID"},
				IsPrimary: true,
				NotNull:   true,
			},
			{
				Name:     "email",
				DataType: Token{Type: TEXT, Literal: "TEXT"},
				NotNull:  true,
				IsUnique: true,
			},
			{
				Name:     "status",
				DataType: Token{Type: ENUM, Literal: "UserStatus"},
				Default:  strPtr("active"),
			},
		},
	}

	expected := `CREATE TABLE IF NOT EXISTS "users" (
  "id" UUID NOT NULL PRIMARY KEY,
  "email" TEXT NOT NULL UNIQUE,
  "status" "UserStatus" DEFAULT 'active'
);`

	got := stringifyTableType(table)
	if got != expected {
		t.Errorf("unexpected table SQL:\nGot:\n%s\n\nExpected:\n%s", got, expected)
	}
}

func TestStringifyEnumType(t *testing.T) {
	enum := &EnumType{
		Name:   "UserStatus",
		Values: []string{"active", "inactive", "banned"},
	}

	expected := `CREATE TYPE "UserStatus" AS ENUM ('active','inactive','banned');`

	got := stringifyEnumType(enum)
	if got != expected {
		t.Errorf("unexpected enum SQL:\nGot:\n%s\n\nExpected:\n%s", got, expected)
	}
}

func strPtr(s string) *string {
	return &s
}
