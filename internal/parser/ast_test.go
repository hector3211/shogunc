package parser

import (
	"fmt"
	"shogunc/internal/types"
	"shogunc/utils"
	"testing"
	"time"
)

type mockQueryFile struct {
	Name string
	SQL  []byte
}

var gen = struct {
	Queries []mockQueryFile
}{
	Queries: []mockQueryFile{
		{
			Name: "select_users",
			SQL:  []byte("SELECT id, name FROM users WHERE age > 30 LIMIT 10 OFFSET 5;"),
		},
		{
			Name: "select_with_bind_params",
			SQL:  []byte("SELECT * FROM users WHERE id = $1 AND active = $2;"),
		},
		{
			Name: "insert_user",
			SQL:  []byte("INSERT INTO users (name, age) VALUES ('John', 25) RETURNING id;"),
		},
		{
			Name: "insert_with_bind_params",
			SQL:  []byte("INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id;"),
		},
	},
}

func setUpSchema() []byte {
	return []byte(`
CREATE TYPE "Complaint_Category" AS ENUM (
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
`)
}

func TestAstLoadTokens(t *testing.T) {
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
				case *types.SelectStatement:
					t.Logf("SELECT - table: %s, fields: %v", stmt.TableName, stmt.Columns)
					for _, c := range stmt.Conditions {
						t.Logf("Condition - Left: %s, Operator: %s, Right: %+v", c.Column, c.Operator, c.Value)
					}
					t.Logf("LIMIT: %d, OFFSET: %d", stmt.Limit, stmt.Offset)

				case *types.InsertStatement:
					t.Logf("INSERT - table: %s, columns: %v", stmt.TableName, stmt.Columns)
					for i, bind := range stmt.Values {
						t.Logf("Value[%d] - Column: %s, Position: %d, Value: %+v", i, bind.Column, bind.Position, bind.Value)
					}
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

func TestBindFunctionality(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []types.Condition
	}{
		{
			name: "SELECT with explicit bind parameters",
			sql:  "SELECT * FROM users WHERE id = $1 AND name = $2;",
			expected: []types.Condition{
				{Column: "id", Operator: types.ConditionOp("="), Value: types.Bind{Column: "id", Position: 1}},
				{Column: "name", Operator: types.ConditionOp("="), Value: types.Bind{Column: "name", Position: 2}},
			},
		},
		{
			name: "SELECT with string literals",
			sql:  "SELECT * FROM users WHERE active = 'true' AND role = 'admin';",
			expected: []types.Condition{
				{Column: "active", Operator: types.ConditionOp("="), Value: types.Bind{Column: "active", Value: &[]string{"true"}[0]}},
				{Column: "role", Operator: types.ConditionOp("="), Value: types.Bind{Column: "role", Value: &[]string{"admin"}[0]}},
			},
		},
		{
			name: "INSERT with bind parameters",
			sql:  "INSERT INTO users (name, email) VALUES ($1, $2);",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.sql)
			parser := NewAst(lexer)

			err := parser.Parse()
			if err != nil {
				t.Errorf("parse error: %v", err)
				return
			}

			if len(parser.Statements) == 0 {
				t.Errorf("no statements parsed")
				return
			}

			switch stmt := parser.Statements[0].(type) {
			case *types.SelectStatement:
				if tt.expected != nil {
					if len(stmt.Conditions) != len(tt.expected) {
						t.Errorf("expected %d conditions, got %d", len(tt.expected), len(stmt.Conditions))
						return
					}

					for i, expected := range tt.expected {
						actual := stmt.Conditions[i]
						if actual.Column != expected.Column {
							t.Errorf("condition %d: expected Left %s, got %s", i, expected.Column, actual.Column)
						}
						if actual.Operator != expected.Operator {
							t.Errorf("condition %d: expected Operator %s, got %s", i, expected.Operator, actual.Operator)
						}
						// Check Bind struct
						if actual.Value.Column != expected.Value.Column {
							t.Errorf("condition %d: expected Column %s, got %s", i, expected.Value.Column, actual.Value.Column)
						}
						if actual.Value.Position != expected.Value.Position {
							t.Errorf("condition %d: expected Position %d, got %d", i, expected.Value.Position, actual.Value.Position)
						}
					}
				}

			case *types.InsertStatement:
				// Check that INSERT statement has proper Bind structs
				for i, bind := range stmt.Values {
					if bind.Column != stmt.Columns[i] {
						t.Errorf("value %d: expected Column %s, got %s", i, stmt.Columns[i], bind.Column)
					}
					if bind.Position == 0 && bind.Value == nil {
						t.Errorf("value %d: Bind struct is empty", i)
					}
				}
			}
		})
	}
}

func TestSchemaParse(t *testing.T) {
	lexer := NewLexer(string(setUpSchema()))
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
		"lockers":         4,
	}

	for _, n := range parser.Statements {
		switch s := n.(type) {
		case *Table:
			name := s.Name
			if expected, ok := expectedTables[name]; ok {
				if len(s.Fields) != expected {
					t.Errorf("table %s: expected %d fields, got %d\n", name, expected, len(s.Fields))
				}
			}
			t.Logf("Parsed table: %s (%d columns)\n", name, len(s.Fields))

		case *Enum:
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
	lexer := NewLexer(string(setUpSchema()))
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
		case *Table:
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
		stmt *types.SelectStatement
		want string
	}{
		{
			stmt: &types.SelectStatement{
				Columns:   []string{"id", "name"},
				TableName: "users",
				Conditions: []types.Condition{
					{Column: "age", Operator: types.ConditionOp(">"), Value: types.Bind{Value: &[]string{"30"}[0]}},
				},
				Limit:  10,
				Offset: 5,
			},
			want: "SELECT id, name FROM users WHERE age > '30' LIMIT 10 OFFSET 5;",
		},
		{
			stmt: &types.SelectStatement{
				Columns:    []string{"*"},
				TableName:  "orders",
				Conditions: nil,
				Limit:      0,
				Offset:     0,
			},
			want: "SELECT * FROM orders;",
		},
		{
			stmt: &types.SelectStatement{
				Columns:   []string{"id", "name"},
				TableName: "products",
				Conditions: []types.Condition{
					{Column: "price", Operator: types.ConditionOp(">="), Value: types.Bind{Value: &[]string{"100"}[0]}},
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
		stmt *types.InsertStatement
		want string
	}{
		{
			stmt: &types.InsertStatement{
				TableName: "users",
				Columns:   []string{"name", "age"},
				Values: []types.Bind{
					{Column: "name", Position: 0, Value: &[]string{"John"}[0]},
					{Column: "age", Position: 0, Value: &[]string{"25"}[0]},
				},
				ReturningFields: []string{"id"},
				InsertMode:      []byte("OR REPLACE"),
			},
			want: "INSERT OR REPLACE INTO users (name, age) VALUES ('John', '25') RETURNING id;",
		},
		{
			stmt: &types.InsertStatement{
				TableName: "products",
				Columns:   []string{"id", "name", "price"},
				Values: []types.Bind{
					{Column: "id", Position: 1},
					{Column: "name", Position: 2},
					{Column: "price", Position: 3},
				},
				ReturningFields: nil,
				InsertMode:      nil,
			},
			want: "INSERT INTO products (id, name, price) VALUES (1, 100, 25);",
		},
		{
			stmt: &types.InsertStatement{
				TableName: "orders",
				Columns:   []string{"id", "quantity"},
				Values: []types.Bind{
					{Column: "id", Position: 1},
					{Column: "quantity", Position: 2},
				},
				ReturningFields: []string{"order_id"},
				InsertMode:      nil,
			},
			want: "INSERT INTO orders (id, quantity) VALUES ($1, $2) RETURNING order_id;",
		},
		{
			stmt: &types.InsertStatement{
				TableName:       "customers",
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
	table := &Table{
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
				Default:  utils.StrPtr("active"),
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
	enum := &Enum{
		Name:   "UserStatus",
		Values: []string{"active", "inactive", "banned"},
	}

	expected := `CREATE TYPE "UserStatus" AS ENUM ('active','inactive','banned');`

	got := stringifyEnumType(enum)
	if got != expected {
		t.Errorf("unexpected enum SQL:\nGot:\n%s\n\nExpected:\n%s", got, expected)
	}
}
