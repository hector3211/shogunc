package gogen

import (
	"shogunc/cmd/generate"
	"shogunc/internal/sqlparser"
	"testing"
)

func TestGenerateQueryOne_SimpleSelect(t *testing.T) {
	gen := &GoSelectFuncGenerator{}
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

	tag := generate.Query{
		Name: []byte("GetUser"),
		Type: ":one",
		SQL:  []byte("SELECT id,name FROM users WHERE name = 'john';"),
	} // stub, adjust as needed
	got := gen.GenerateQueryOne(&tag, stmt)
	want := `query := Select(id,name).From(users).Where(Equal(name, 'john')).Build()`

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateQueryOne_WithLogicalOps(t *testing.T) {
	gen := &GoSelectFuncGenerator{}
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

	tag := generate.Query{}
	got := gen.GenerateQueryOne(&tag, stmt)
	want := `query := Select(id,name).From(users).Where(And(),Equal(name, 'john'),GreaterThan(id, 10)).Build()`

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateQueryOne_SelectAll(t *testing.T) {
	gen := &GoSelectFuncGenerator{}
	stmt := &sqlparser.SelectStatement{
		TableName: []byte("products"),
		Fields:    []string{}, // SELECT *
		Conditions: []sqlparser.Condition{
			{
				Left:     []byte("price"),
				Operator: sqlparser.LESSTHAN,
				Right:    100,
			},
		},
	}

	tag := generate.Query{}
	got := gen.GenerateQueryOne(&tag, stmt)
	want := `query := Select('*').From(products).Where(LessThan(price, 100)).Build()`

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}
