package gogen

import (
	"fmt"
	"shogunc/cmd/generate"
	"shogunc/internal/sqlparser"
	"testing"
)

func TestGenerateQueryOne_SimpleSelect(t *testing.T) {
	tag := generate.Query{
		Name: []byte("GetUser"),
		Type: "one",
		SQL:  []byte("SELECT id,name FROM users WHERE name = 'john';"),
	}
	gen := NewGoFuncGenerator(tag.Name, tag.Type)
	stmt := &sqlparser.SelectStatement{
		TableName: []byte("users"),
		Fields:    []string{"id", "name"},
		Conditions: []sqlparser.Condition{
			{
				Left:     []byte("name"),
				Operator: sqlparser.EQUAL,
				Right:    "'john'",
			},
		},
	}

	got := gen.GenerateFunction(stmt)
	want := fmt.Sprintf(`func %s() {
query := Select(id,name).From(users).Where(Equal(name, 'john')).Build()
}`, tag.Name)

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateQueryOne_WithLogicalOps(t *testing.T) {
	tag := generate.Query{
		Name: []byte("GetUser"),
		Type: "one",
		SQL:  []byte("SELECT id,name FROM users WHERE name = 'john' AND id > 10;"),
	}
	gen := NewGoFuncGenerator(tag.Name, tag.Type)
	stmt := &sqlparser.SelectStatement{
		TableName: []byte("users"),
		Fields:    []string{"id", "name"},
		Conditions: []sqlparser.Condition{
			{
				Left:     []byte("name"),
				Operator: sqlparser.EQUAL,
				Right:    "'john'",
				Next:     sqlparser.And,
			},
			{
				Left:     []byte("id"),
				Operator: sqlparser.GREATERTHAN,
				Right:    10,
			},
		},
	}

	got := gen.GenerateFunction(stmt)
	want := fmt.Sprintf(`func %s() {
query := Select(id,name).From(users).Where(And(),Equal(name, 'john'),GreaterThan(id, 10)).Build()
}`, tag.Name)

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateQueryOne_SelectAll(t *testing.T) {
	tag := generate.Query{
		Name: []byte("GetProducts"),
		Type: "one",
		SQL:  []byte("SELECT * FROM products WHERE price < 100;"),
	}
	gen := NewGoFuncGenerator(tag.Name, tag.Type)
	stmt := &sqlparser.SelectStatement{
		TableName: []byte("products"),
		Fields:    []string{"*"}, // SELECT *
		Conditions: []sqlparser.Condition{
			{
				Left:     []byte("price"),
				Operator: sqlparser.LESSTHAN,
				Right:    100,
			},
		},
	}

	got := gen.GenerateFunction(stmt)
	want := fmt.Sprintf(`func %s() {
query := Select('*').From(products).Where(LessThan(price, 100)).Build()
}`, tag.Name)

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}
