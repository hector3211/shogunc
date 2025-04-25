package sqlparser

import (
	"fmt"
	"os"
	"shogunc/cmd/generate"
	"testing"
)

func setUpGenerator(t *testing.T) *generate.GeneratorBuilder {
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

func TestAstParse(t *testing.T) {
	gen := setUpGenerator(t)

	for _, file := range gen.Queries {
		t.Run(fmt.Sprintf("Parsing: %s", file.Name), func(t *testing.T) {
			lexer := NewLexer(string(file.SQL))
			parser := NewAst(lexer)

			node, err := parser.Parse()
			if err != nil {
				t.Errorf("parse error: %v", err)
				return
			}
			if node == nil {
				t.Errorf("parsed node is nil")
				return
			}

			switch stmt := node.(type) {
			case *SelectStatement:
				t.Logf("SELECT - table: %s, fields: %v", stmt.TableName, stmt.Fields)
				for _, c := range stmt.Conditions {
					t.Logf("Condition - Ident: %s, Operator: %s, Value: %v\n", c.Left, c.Operator, c.Right)
				}
				t.Logf("LIMIT: %d, OFFSET: %d", stmt.Limit, stmt.Offset)

			case *InsertStatement:
				t.Logf("INSERT - table: %s, values: %d", stmt.TableName, stmt.Values)
				for _, col := range stmt.Columns {
					t.Logf("Column: %s", string(col))
				}

			default:
				t.Errorf("unknown AST node type: %T", node)
			}
		})
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
		t.Run(fmt.Sprintf("SelectStatement(%v)", tt.stmt), func(t *testing.T) {
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
		t.Run(fmt.Sprintf("InsertStatement(%v)", tt.stmt), func(t *testing.T) {
			got := stringifyInsertStatement(tt.stmt)
			if got != tt.want {
				t.Errorf("got = %v want %v", got, tt.want)
			}
		})
	}
}
