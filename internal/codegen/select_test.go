package codegen

import (
	"go/ast"
	"shogunc/internal/parser"
	"shogunc/internal/types"
	"strings"
	"testing"
)

// TestGenerateSelectFunc tests SELECT function generation with bind parameters
func TestGenerateSelectFunc(t *testing.T) {
	// Setup schema types
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{
		Name: "GetUser",
		Type: types.ONE,
	}

	generator := NewSelectGenerator(schemaTypes, queryBlock)

	// Create a SELECT statement with bind parameter
	selectStmt := &types.SelectStatement{
		TableName: "users",
		Columns:   []string{"*"},
		Conditions: []types.Condition{
			{
				Column:   "id",
				Operator: types.EQUAL,
				Value: types.Bind{
					Column:   "id",
					Position: 1,
				},
			},
		},
	}

	funcDecl, paramStruct, err := generator.GenerateSelectFunc(selectStmt, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that function declaration is generated
	if funcDecl == nil {
		t.Error("Expected function declaration to be generated")
	}
	if funcDecl.Name.Name != "GetUser" {
		t.Errorf("Expected function name 'GetUser', got '%s'", funcDecl.Name.Name)
	}

	// Check that parameter struct is generated
	if paramStruct == nil {
		t.Error("Expected parameter struct to be generated")
	}
	if paramStruct.Tok.String() != "type" {
		t.Errorf("Expected type declaration, got '%s'", paramStruct.Tok.String())
	}

	// Check that the struct has the expected name
	if typeSpec, ok := paramStruct.Specs[0].(*ast.TypeSpec); ok {
		if typeSpec.Name.Name != "GetUserParams" {
			t.Errorf("Expected struct name 'GetUserParams', got '%s'", typeSpec.Name.Name)
		}
	} else {
		t.Error("Expected TypeSpec in param struct")
	}
}

// TestGenerateSelectParamStruct tests parameter struct generation
func TestGenerateSelectParamStruct(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewSelectGenerator(schemaTypes, queryBlock)

	selectStmt := &types.SelectStatement{
		TableName: "users",
		Conditions: []types.Condition{
			{
				Column:   "id",
				Operator: types.EQUAL,
				Value: types.Bind{
					Column:   "id",
					Position: 1,
				},
			},
		},
	}

	structDecl, typeName, err := generator.generateSelectParamStruct(selectStmt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if typeName != "GetUserParams" {
		t.Errorf("Expected typeName to be 'GetUserParams', got '%s'", typeName)
	}

	if structDecl == nil {
		t.Error("Expected struct declaration to be generated")
	}

	if structDecl.Tok.String() != "type" {
		t.Errorf("Expected type declaration, got '%s'", structDecl.Tok.String())
	}

	// Check that the struct has the expected name
	if typeSpec, ok := structDecl.Specs[0].(*ast.TypeSpec); ok {
		if typeSpec.Name.Name != "GetUserParams" {
			t.Errorf("Expected struct name 'GetUserParams', got '%s'", typeSpec.Name.Name)
		}
	} else {
		t.Error("Expected TypeSpec in struct declaration")
	}
}

// TestGenerateSelectParamStruct_NoParams tests parameter struct generation with no parameters
func TestGenerateSelectParamStruct_NoParams(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}}, // Add at least one field
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "GetUsers", Type: types.MANY}
	generator := NewSelectGenerator(schemaTypes, queryBlock)

	selectStmt := &types.SelectStatement{
		TableName:  "users",
		Conditions: []types.Condition{}, // No conditions = no parameters
	}

	structDecl, typeName, err := generator.generateSelectParamStruct(selectStmt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if typeName != "" {
		t.Errorf("Expected typeName to be empty when no parameters, got '%s'", typeName)
	}

	if structDecl != nil {
		t.Error("Expected nil struct declaration for no parameters")
	}
}

// TestGenerateSelectFunc_NoParams tests SELECT function generation with no parameters
func TestGenerateSelectFunc_NoParams(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "GetUsers", Type: types.MANY}
	generator := NewSelectGenerator(schemaTypes, queryBlock)

	selectStmt := &types.SelectStatement{
		TableName:  "users",
		Columns:    []string{"*"},
		Conditions: []types.Condition{}, // No conditions = no parameters
	}

	funcDecl, paramStruct, err := generator.GenerateSelectFunc(selectStmt, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that function declaration is generated
	if funcDecl == nil {
		t.Error("Expected function declaration to be generated")
	}
	if funcDecl.Name.Name != "GetUsers" {
		t.Errorf("Expected function name 'GetUsers', got '%s'", funcDecl.Name.Name)
	}

	// Check that NO parameter struct is generated
	if paramStruct != nil {
		t.Error("Expected no parameter struct to be generated")
	}

	// Check that function has q *Queries and context parameters
	if len(funcDecl.Type.Params.List) != 2 {
		t.Errorf("Expected 2 parameters (q and ctx), got %d", len(funcDecl.Type.Params.List))
	}
	// Verify the q parameter
	qParam := funcDecl.Type.Params.List[0]
	if len(qParam.Names) == 0 || qParam.Names[0].Name != "q" {
		t.Errorf("Expected first parameter name 'q', got '%s'", qParam.Names[0].Name)
	}
	// Verify the context parameter
	ctxParam := funcDecl.Type.Params.List[1]
	if len(ctxParam.Names) == 0 || ctxParam.Names[0].Name != "ctx" {
		t.Errorf("Expected second parameter name 'ctx', got '%s'", ctxParam.Names[0].Name)
	}
}

// TestShoguncConditionalOp_BindParam tests bind parameter conditions
func TestShoguncConditionalOp_BindParam(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewSelectGenerator(schemaTypes, queryBlock)

	condition := types.Condition{
		Column:   "ID",
		Operator: types.EQUAL,
		Value: types.Bind{
			Column:   "id",
			Position: 1,
		},
	}

	result := generator.shoguncConditionalOp(condition)
	expected := "id = $1"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestShoguncConditionalOp_LiteralValue tests literal value conditions
func TestShoguncConditionalOp_LiteralValue(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewSelectGenerator(schemaTypes, queryBlock)

	value := "test@example.com"
	condition := types.Condition{
		Column:   "EMAIL",
		Operator: types.EQUAL,
		Value: types.Bind{
			Column:   "email",
			Position: 0,
			Value:    &value,
		},
	}

	result := generator.shoguncConditionalOp(condition)
	// Should use shogun.Equal which formats the result
	if !strings.Contains(result, "email") {
		t.Error("Expected result to contain email column")
	}
}

// TestShoguncNextOp tests logical operators
func TestShoguncNextOp(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewSelectGenerator(schemaTypes, queryBlock)

	andOp := generator.shoguncNextOp(types.And)
	if andOp == "" {
		t.Error("Expected non-empty AND operator")
	}

	orOp := generator.shoguncNextOp(types.Or)
	if orOp == "" {
		t.Error("Expected non-empty OR operator")
	}
}
