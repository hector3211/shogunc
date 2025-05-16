package gogen

import (
	"fmt"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
	"testing"
)

func TestGenerateQuerySimpleSelect(t *testing.T) {
	table := sqlparser.TableType{
		Name:   "User",
		Fields: []sqlparser.Field{},
	}
	tag := utils.TagType{
		Name: []byte("GetUser"),
		Type: "one",
	}
	gen := NewFuncGenerator(tag.Name, tag.Type, &table)
	stmt := &sqlparser.SelectStatement{
		TableName: "users",
		Fields:    []string{"id", "name"},
		Conditions: []sqlparser.Condition{
			{
				Left:     []byte("name"),
				Operator: sqlparser.EQUAL,
				Right:    "john",
			},
		},
	}

	got, err := gen.GenerateFunction(stmt)
	if err != nil {
		t.Error(err)
	}
	want := fmt.Sprintf(`func %s(ctx context.Context) User {
query := Select("id","name").From("users").Where(Equal("name", "john")).Build()
}`, tag.Name)

	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateQueryWithLogicalOps(t *testing.T) {
	table := sqlparser.TableType{
		Name:   "User",
		Fields: []sqlparser.Field{},
	}
	tag := utils.TagType{
		Name: []byte("GetUser"),
		Type: "one",
	}
	gen := NewFuncGenerator(tag.Name, tag.Type, &table)
	stmt := &sqlparser.SelectStatement{
		TableName: "users",
		Fields:    []string{"id", "name"},
		Conditions: []sqlparser.Condition{
			{
				Left:     []byte("name"),
				Operator: sqlparser.EQUAL,
				Right:    "john",
				Next:     sqlparser.And,
			},
			{
				Left:     []byte("id"),
				Operator: sqlparser.GREATERTHAN,
				Right:    10,
			},
		},
	}

	got, err := gen.GenerateFunction(stmt)
	if err != nil {
		t.Error(err)
	}
	want := fmt.Sprintf(`func %s(ctx context.Context) User {
query := Select("id","name").From("users").Where(And(),Equal("name", "john"),GreaterThan("id", 10)).Build()
}`, tag.Name)

	if got != want {
		t.Errorf("\nexpected:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateQuerySelectAll(t *testing.T) {
	table := sqlparser.TableType{
		Name:   "Lockers",
		Fields: []sqlparser.Field{},
	}
	tag := utils.TagType{
		Name: []byte("GetLocker"),
		Type: "one",
	}
	gen := NewFuncGenerator(tag.Name, tag.Type, &table)
	stmt := &sqlparser.SelectStatement{
		TableName: "lockers",
		Fields:    []string{"*"}, // SELECT *
		Conditions: []sqlparser.Condition{
			{
				Left:     []byte("access_code"),
				Operator: sqlparser.EQUAL,
				Right:    "k1jk21",
			},
		},
	}

	got, err := gen.GenerateFunction(stmt)
	if err != nil {
		t.Error(err)
	}
	want := fmt.Sprintf(`func %s(ctx context.Context) Lockers {
query := Select('*').From("lockers").Where(Equal("access_code", "k1jk21")).Build()
}`, tag.Name)

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
	Id string ` + "`db:\"id\"`" + `
	Email string ` + "`db:\"email\"`" + `
	Status *UserStatus ` + "`db:\"status\"`" + `
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
