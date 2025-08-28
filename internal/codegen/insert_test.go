package codegen

import (
	"go/ast"
	"go/token"
	"reflect"
	"shogunc/internal/parser"
	"shogunc/internal/types"
	"strings"
	"testing"
)

// TestNewInsertGenerator tests the constructor
func TestNewInsertGenerator(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{
		Name: "CreateUser",
		Type: types.EXEC,
	}

	generator := NewInsertGenerator(schemaTypes, queryBlock)

	if generator.schemaTypes == nil {
		t.Error("Expected schemaTypes to be set")
	}
	if generator.queryblock != queryBlock {
		t.Error("Expected queryblock to be set")
	}
}

// TestGenerateInsertFunc tests INSERT function generation with bind parameters
func TestGenerateInsertFunc(t *testing.T) {
	// Setup schema types
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
				{Name: "name", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{
		Name: "CreateUser",
		Type: types.EXEC,
	}

	generator := NewInsertGenerator(schemaTypes, queryBlock)

	// Create an INSERT statement with bind parameters
	insertStmt := &types.InsertStatement{
		TableName: "users",
		Columns:   []string{"email", "name"},
		Values: []types.Bind{
			{
				Column:   "email",
				Position: 1,
			},
			{
				Column:   "name",
				Position: 2,
			},
		},
	}

	funcDecl, paramStruct, err := generator.GenerateInsertFunc(insertStmt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that function declaration is generated
	if funcDecl == nil {
		t.Error("Expected function declaration to be generated")
	}
	if funcDecl.Name.Name != "CreateUser" {
		t.Errorf("Expected function name 'CreateUser', got '%s'", funcDecl.Name.Name)
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
		if typeSpec.Name.Name != "CreateUserParams" {
			t.Errorf("Expected struct name 'CreateUserParams', got '%s'", typeSpec.Name.Name)
		}
	} else {
		t.Error("Expected TypeSpec in param struct")
	}
}

// TestGenerateInsertFunc_NoParams tests INSERT function generation with no parameters
func TestGenerateInsertFunc_NoParams(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	// INSERT with literal values (no bind parameters)
	emailValue := "test@example.com"
	insertStmt := &types.InsertStatement{
		TableName: "users",
		Columns:   []string{"email"},
		Values: []types.Bind{
			{
				Column:   "email",
				Position: 0, // Position 0 means literal value
				Value:    &emailValue,
			},
		},
	}

	funcDecl, paramStruct, err := generator.GenerateInsertFunc(insertStmt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that function declaration is generated
	if funcDecl == nil {
		t.Error("Expected function declaration to be generated")
	}
	if funcDecl.Name.Name != "CreateUser" {
		t.Errorf("Expected function name 'CreateUser', got '%s'", funcDecl.Name.Name)
	}

	// Check that NO parameter struct is generated for literal values
	if paramStruct != nil {
		t.Error("Expected no parameter struct to be generated for literal values")
	}

	// Check that function has q *Queries and context parameters only
	if len(funcDecl.Type.Params.List) != 2 {
		t.Errorf("Expected 2 parameters (q and ctx), got %d", len(funcDecl.Type.Params.List))
	}
}

// TestGenerateInsertParamStruct tests parameter struct generation
func TestGenerateInsertParamStruct(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
				{Name: "name", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	insertStmt := &types.InsertStatement{
		TableName: "users",
		Columns:   []string{"email", "name"},
		Values: []types.Bind{
			{
				Column:   "email",
				Position: 1,
			},
			{
				Column:   "name",
				Position: 2,
			},
		},
	}

	structDecl, typeName, err := generator.generateInsertParamStruct(insertStmt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if typeName != "CreateUserParams" {
		t.Errorf("Expected typeName to be 'CreateUserParams', got '%s'", typeName)
	}

	if structDecl == nil {
		t.Error("Expected struct declaration to be generated")
	}

	if structDecl.Tok.String() != "type" {
		t.Errorf("Expected type declaration, got '%s'", structDecl.Tok.String())
	}

	// Check that the struct has the expected name
	if typeSpec, ok := structDecl.Specs[0].(*ast.TypeSpec); ok {
		if typeSpec.Name.Name != "CreateUserParams" {
			t.Errorf("Expected struct name 'CreateUserParams', got '%s'", typeSpec.Name.Name)
		}
	} else {
		t.Error("Expected TypeSpec in struct declaration")
	}
}

// TestGenerateInsertParamStruct_NoParams tests parameter struct generation with no parameters
func TestGenerateInsertParamStruct_NoParams(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	// No bind parameters (all literal values)
	emailValue := "test@example.com"
	insertStmt := &types.InsertStatement{
		TableName: "users",
		Columns:   []string{"email"},
		Values: []types.Bind{
			{
				Column:   "email",
				Position: 0,
				Value:    &emailValue,
			},
		},
	}

	structDecl, typeName, err := generator.generateInsertParamStruct(insertStmt)
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

// TestInsertGenerateReturnType tests return type generation for INSERT
func TestInsertGenerateReturnType(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	// Test with users table
	returnType := generator.generateReturnType("users")
	if ident, ok := returnType.(*ast.Ident); !ok || ident.Name != "User" {
		t.Errorf("Expected ast.Ident with name 'User', got %T with value %v", returnType, returnType)
	}

	// Test with posts table
	returnType = generator.generateReturnType("posts")
	if ident, ok := returnType.(*ast.Ident); !ok || ident.Name != "Post" {
		t.Errorf("Expected ast.Ident with name 'Post', got %T with value %v", returnType, returnType)
	}
}

// TestInsertInferDataType tests data type inference for INSERT
func TestInsertInferDataType(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
				{Name: "active", DataType: parser.Token{Literal: "BOOLEAN"}},
				{Name: "age", DataType: parser.Token{Literal: "INT"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	fieldMap, err := generator.inferDataType("users")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := map[string]string{
		"id":     "string",
		"email":  "string",
		"active": "bool",
		"age":    "int",
	}

	if !reflect.DeepEqual(fieldMap, expected) {
		t.Errorf("Expected %v, got %v", expected, fieldMap)
	}
}

// TestInsertInferDataType_Enum tests enum data type inference for INSERT
func TestInsertInferDataType_Enum(t *testing.T) {
	schemaTypes := map[string]any{
		"role": &parser.Enum{
			Name:   "role",
			Values: []string{"admin", "user"},
		},
	}

	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	fieldMap, err := generator.inferDataType("role")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := map[string]string{
		"role": "string",
	}

	if !reflect.DeepEqual(fieldMap, expected) {
		t.Errorf("Expected %v, got %v", expected, fieldMap)
	}
}

// TestInsertInferDataType_TableNotFound tests error handling for nonexistent table in INSERT
func TestInsertInferDataType_TableNotFound(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	_, err := generator.inferDataType("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent table")
	}

	expectedError := "table 'nonexistent' not found in schema"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

// TestGenerateInsertQuery tests SQL query generation
func TestGenerateInsertQuery(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	insertStmt := &types.InsertStatement{
		TableName: "users",
		Columns:   []string{"email", "name"},
		Values: []types.Bind{
			{
				Column:   "email",
				Position: 1,
			},
			{
				Column:   "name",
				Position: 2,
			},
		},
	}

	stmt := generator.generateInsertQuery(insertStmt)
	if stmt == nil {
		t.Error("Expected statement to be generated")
	}

	// Check that it's an assignment statement
	assignStmt, ok := stmt.(*ast.AssignStmt)
	if !ok {
		t.Errorf("Expected *ast.AssignStmt, got %T", stmt)
	}

	// Check LHS is 'query'
	if len(assignStmt.Lhs) != 1 {
		t.Errorf("Expected 1 LHS expression, got %d", len(assignStmt.Lhs))
	}
	if ident, ok := assignStmt.Lhs[0].(*ast.Ident); !ok || ident.Name != "query" {
		t.Errorf("Expected LHS to be 'query', got %v", assignStmt.Lhs[0])
	}

	// Check RHS is a string literal
	if len(assignStmt.Rhs) != 1 {
		t.Errorf("Expected 1 RHS expression, got %d", len(assignStmt.Rhs))
	}
	if basicLit, ok := assignStmt.Rhs[0].(*ast.BasicLit); !ok || basicLit.Kind != token.STRING {
		t.Errorf("Expected RHS to be string literal, got %v", assignStmt.Rhs[0])
	}
}

// TestGenerateResultDecl tests result variable declaration
func TestGenerateResultDecl(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	insertStmt := &types.InsertStatement{
		TableName: "users",
	}

	declStmt := generator.generateResultDecl(insertStmt)
	if declStmt == nil {
		t.Error("Expected statement to be generated")
	}

	// Check that it's a declaration statement
	if genDecl, ok := declStmt.Decl.(*ast.GenDecl); ok {
		if genDecl.Tok.String() != "var" {
			t.Errorf("Expected var declaration, got %s", genDecl.Tok.String())
		}
	} else {
		t.Errorf("Expected *ast.GenDecl, got %T", declStmt.Decl)
	}
}

// TestGenerateScanArgs tests scan arguments generation
func TestGenerateScanArgs(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	insertStmt := &types.InsertStatement{
		TableName: "users",
		Columns:   []string{"id", "email"},
	}

	args := generator.generateScanArgs(insertStmt)
	if len(args) != 2 {
		t.Errorf("Expected 2 scan arguments, got %d", len(args))
	}

	// Check first argument: &result.Id
	if unaryExpr, ok := args[0].(*ast.UnaryExpr); ok {
		if unaryExpr.Op != token.AND {
			t.Errorf("Expected & operator, got %v", unaryExpr.Op)
		}
		if selectorExpr, ok := unaryExpr.X.(*ast.SelectorExpr); ok {
			if selectorExpr.Sel.Name != "Id" {
				t.Errorf("Expected field name 'Id', got '%s'", selectorExpr.Sel.Name)
			}
		} else {
			t.Errorf("Expected selector expression, got %T", unaryExpr.X)
		}
	} else {
		t.Errorf("Expected unary expression, got %T", args[0])
	}
}

// TestGenerateScanArgs_SelectAll tests scan arguments generation with SELECT *
func TestGenerateScanArgs_SelectAll(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
				{Name: "name", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	insertStmt := &types.InsertStatement{
		TableName: "users",
		Columns:   []string{"*"},
	}

	args := generator.generateScanArgs(insertStmt)
	if len(args) != 3 {
		t.Errorf("Expected 3 scan arguments for SELECT *, got %d", len(args))
	}
}

// TestGenerateReturnStmt tests return statement generation
func TestGenerateReturnStmt(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	returnStmt := generator.generateReturnStmt(true) // Test with RETURNING
	if returnStmt == nil {
		t.Error("Expected return statement to be generated")
	}

	if len(returnStmt.Results) != 2 {
		t.Errorf("Expected 2 return values, got %d", len(returnStmt.Results))
	}

	// Check first result: result
	if ident, ok := returnStmt.Results[0].(*ast.Ident); !ok || ident.Name != "result" {
		t.Errorf("Expected first result to be 'result', got %v", returnStmt.Results[0])
	}

	// Check second result: err
	if ident, ok := returnStmt.Results[1].(*ast.Ident); !ok || ident.Name != "err" {
		t.Errorf("Expected second result to be 'err', got %v", returnStmt.Results[1])
	}
}

// TestGenerateInsertFunc_WithReturning tests INSERT with RETURNING clause
func TestGenerateInsertFunc_WithReturning(t *testing.T) {
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
		Name: "CreateUser",
		Type: types.ONE, // RETURNING should return ONE result
	}

	generator := NewInsertGenerator(schemaTypes, queryBlock)

	insertStmt := &types.InsertStatement{
		TableName:       "users",
		Columns:         []string{"email"},
		ReturningFields: []string{"id", "email"},
		Values: []types.Bind{
			{
				Column:   "email",
				Position: 1,
			},
		},
	}

	funcDecl, paramStruct, err := generator.GenerateInsertFunc(insertStmt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that function declaration is generated
	if funcDecl == nil {
		t.Error("Expected function declaration to be generated")
	}

	// Check return type - should be User (not error)
	if len(funcDecl.Type.Results.List) != 2 {
		t.Errorf("Expected 2 return values, got %d", len(funcDecl.Type.Results.List))
	}

	// Check that parameter struct is generated
	if paramStruct == nil {
		t.Error("Expected parameter struct to be generated")
	}
}

// TestGenerateInsertFunc_ErrorHandling tests error handling in various scenarios
func TestGenerateInsertFunc_ErrorHandling(t *testing.T) {
	// Test with nonexistent table
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "CreateUser", Type: types.EXEC}
	generator := NewInsertGenerator(schemaTypes, queryBlock)

	insertStmt := &types.InsertStatement{
		TableName: "nonexistent",
		Columns:   []string{"email"},
		Values: []types.Bind{
			{
				Column:   "email",
				Position: 1,
			},
		},
	}

	_, _, err := generator.GenerateInsertFunc(insertStmt)
	if err == nil {
		t.Error("Expected error for nonexistent table")
	}
}
