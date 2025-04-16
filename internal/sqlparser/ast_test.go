package sqlparser

import (
	"fmt"
	"os"
	"shogunc/cmd/generate"
	"testing"
)

func TestAstOne(t *testing.T) {
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

func TestAstTwo(t *testing.T) {
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

	for _, file := range gen.Queries {
		lexer := NewLexer(string(file.SQL))
		parser := NewAst(lexer)

		node, err := parser.Parse()
		if err != nil || node == nil {
			t.Errorf("Failed to parse in %s: %v", file.Name, err)
			continue
		}
		// t.Logf("Parsed node: %#v", node)

		switch stmt := node.(type) {
		case *SelectStatement:
			t.Logf("Parsed SELECT: table=%s, fields=%v\n", stmt.TableName, stmt.Fields)
			for _, c := range stmt.Conditions {
				fmt.Printf("Ident:%s Condition:%s Bind:%v\n", c.Ident, c.Condition, c.Value)
			}
			t.Logf("LIMIT: %d", stmt.Limit)
			t.Logf("OFFSET: %d", stmt.Offset)
		case *InsertStatement:
			t.Logf("Parsed INSERT: table=%s, values=%d", stmt.TableName, stmt.Values)
			for _, c := range stmt.Columns {
				t.Logf("%s,", string(c))
			}
		default:
			t.Errorf("Unknown AST node type for query: %s", file.Name)
		}
	}
}
