package gogen

import (
	"fmt"
	"shogunc/cmd/generate"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
	"testing"
)

func TestGenerateQuerySimpleSelect(t *testing.T) {
	query := generate.Query{
		Name: []byte("GetUser"),
		Type: "one",
		SQL:  []byte("SELECT id,name FROM users WHERE name = 'john';"),
	}
	gen := NewGoFuncGenerator(query)
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
}`, query.Name)

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateQueryWithLogicalOps(t *testing.T) {
	query := generate.Query{
		Name: []byte("GetUser"),
		Type: "one",
		SQL:  []byte("SELECT id,name FROM users WHERE name = 'john' AND id > 10;"),
	}
	gen := NewGoFuncGenerator(query)
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
}`, query.Name)

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateQuerySelectAll(t *testing.T) {
	query := generate.Query{
		Name: []byte("GetProducts"),
		Type: "one",
		SQL:  []byte("SELECT * FROM products WHERE price < 100;"),
	}
	gen := NewGoFuncGenerator(query)
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
}`, query.Name)

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestTableTypeGenerator(t *testing.T) {
	table := &sqlparser.TableType{
		Name: "Users",
		Fields: []sqlparser.Field{
			{
				Name:      "id",
				DataType:  sqlparser.Token{Type: sqlparser.UUID, Literal: "UUID"},
				IsPrimary: true,
				NotNull:   true,
			},
			{
				Name:     "email",
				DataType: sqlparser.Token{Type: sqlparser.TEXT, Literal: "TEXT"},
				NotNull:  true,
				IsUnique: true,
			},
			{
				Name:     "status",
				DataType: sqlparser.Token{Type: sqlparser.ENUM, Literal: "UserStatus"},
				Default:  utils.StrPtr("active"),
			},
		},
	}

	got, err := GenerateTableType(*table)
	if err != nil {
		t.Fatalf("generating table type failed: %v", err)
	}

	want := `type Users struct {
	Id string ` + "`json:\"id\"`" + `
	Email string ` + "`json:\"email\"`" + `
	Status *UserStatus ` + "`json:\"status\"`" + `
}
`

	if got != want {
		t.Errorf("unexpected output:\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestEnumTypeGenerator(t *testing.T) {
	enum := &sqlparser.EnumType{
		Name:   "UserStatus",
		Values: []string{"active", "inactive", "banned"},
	}

	got, err := GenerateEnumType(*enum)
	if err != nil {
		t.Fatalf("generating enum type failed: %v", err)
	}

	want := `type UserStatus string

const (
	Active UserStatus = "active"
	Inactive UserStatus = "inactive"
	Banned UserStatus = "banned"
)
`

	if got != want {
		t.Errorf("unexpected output:\nGot:\n%s\nWant:\n%s", got, want)
	}
}
