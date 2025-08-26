package codegen

import (
	"go/ast"
	"reflect"
	"strings"
	"testing"

	"shogunc/internal/parser"
	"shogunc/internal/types"
)

func TestNewGoGenerator(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{
		Name: "GetUser",
		Type: types.ONE,
	}

	generator := NewGoGenerator(schemaTypes, queryBlock)

	if generator.schemaTypes == nil {
		t.Error("Expected schemaTypes to be set")
	}
	if generator.queryblock != queryBlock {
		t.Error("Expected queryblock to be set")
	}
}

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

	generator := NewGoGenerator(schemaTypes, queryBlock)

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

	funcDecl, paramStruct, err := generator.generateSelectFunc(selectStmt, false)
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
	generator := NewGoGenerator(schemaTypes, queryBlock)

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
	generator := NewGoGenerator(schemaTypes, queryBlock)

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
	generator := NewGoGenerator(schemaTypes, queryBlock)

	selectStmt := &types.SelectStatement{
		TableName:  "users",
		Columns:    []string{"*"},
		Conditions: []types.Condition{}, // No conditions = no parameters
	}

	funcDecl, paramStruct, err := generator.generateSelectFunc(selectStmt, true)
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

	// Check that function has only context parameter (no params)
	if len(funcDecl.Type.Params.List) != 1 {
		t.Errorf("Expected 1 parameter (context only), got %d", len(funcDecl.Type.Params.List))
	}
}

func TestGenerateReturnType(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

	selectStmt := &types.SelectStatement{TableName: "users"}

	// Test ONE query
	returnType := generator.generateReturnType(selectStmt, false)
	expected := "(User, error)"
	if returnType != expected {
		t.Errorf("Expected '%s', got '%s'", expected, returnType)
	}

	// Test MANY query
	returnType = generator.generateReturnType(selectStmt, true)
	expected = "([]User, error)"
	if returnType != expected {
		t.Errorf("Expected '%s', got '%s'", expected, returnType)
	}
}

func TestInferDataType(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
				{Name: "active", DataType: parser.Token{Literal: "BOOLEAN"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

	fieldMap, err := generator.inferDataType("users")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := map[string]string{
		"id":     "string",
		"email":  "string",
		"active": "bool",
	}

	if !reflect.DeepEqual(fieldMap, expected) {
		t.Errorf("Expected %v, got %v", expected, fieldMap)
	}
}

func TestInferDataType_Enum(t *testing.T) {
	schemaTypes := map[string]any{
		"role": &parser.Enum{
			Name:   "role",
			Values: []string{"admin", "user"},
		},
	}

	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

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

func TestInferDataType_TableNotFound(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

	_, err := generator.inferDataType("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent table")
	}
}

func TestShoguncConditionalOp_BindParam(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

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

func TestShoguncConditionalOp_LiteralValue(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

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

func TestShoguncNextOp(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

	andOp := generator.shoguncNextOp(types.And)
	if andOp == "" {
		t.Error("Expected non-empty AND operator")
	}

	orOp := generator.shoguncNextOp(types.Or)
	if orOp == "" {
		t.Error("Expected non-empty OR operator")
	}
}

func TestGenerateDB(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

	result := generator.GenerateDB("queries")

	if !strings.Contains(result, "package queries") {
		t.Error("Expected package declaration")
	}

	if !strings.Contains(result, "type DBX interface") {
		t.Error("Expected DBX interface")
	}

	if !strings.Contains(result, "type Queries struct") {
		t.Error("Expected Queries struct")
	}

	if !strings.Contains(result, "func New(db DBX) *Queries") {
		t.Error("Expected New function")
	}
}

func TestGenerateEnumType(t *testing.T) {
	enumType := &parser.Enum{
		Name:   "Role",
		Values: []string{"admin", "user", "moderator"},
	}

	result, err := GenerateEnumType(enumType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(result, "type Role string") {
		t.Error("Expected enum type definition")
	}

	if !strings.Contains(result, "Admin Role = \"admin\"") {
		t.Error("Expected Admin constant")
	}

	if !strings.Contains(result, "User Role = \"user\"") {
		t.Error("Expected User constant")
	}
}

func TestGenerateTableType(t *testing.T) {
	tableType := &parser.Table{
		Name: "users",
		Fields: []parser.Field{
			{Name: "id", DataType: parser.Token{Literal: "UUID"}},
			{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
		},
	}

	result, err := GenerateTableType(tableType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(result, "type User struct") {
		t.Error("Expected User struct")
	}

	if !strings.Contains(result, "Id string `db:\"id\"`") {
		t.Error("Expected Id field")
	}

	if !strings.Contains(result, "Email string `db:\"email\"`") {
		t.Error("Expected Email field")
	}

	if !strings.Contains(result, "type NewUser struct") {
		t.Error("Expected NewUser struct")
	}
}

func TestGenerate_UnsupportedType(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{
		Name: "GetUser",
		Type: types.Type("UNSUPPORTED"),
	}
	generator := NewGoGenerator(schemaTypes, queryBlock)

	_, _, err := generator.Generate(nil)
	if err == nil {
		t.Error("Expected error for unsupported query type")
	}
}

func TestGenerate_ExecNotImplemented(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{
		Name: "CreateUser",
		Type: types.EXEC,
	}
	generator := NewGoGenerator(schemaTypes, queryBlock)

	_, _, err := generator.Generate(nil)
	if err == nil {
		t.Error("Expected error for EXEC type")
	}

	expectedError := "EXEC not implemented yet"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGenerateSelectParamStruct_WithStructTags(t *testing.T) {
	schemaTypes := map[string]any{
		"users": &parser.Table{
			Name: "users",
			Fields: []parser.Field{
				{Name: "id", DataType: parser.Token{Literal: "UUID"}},
				{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}},
				{Name: "first_name", DataType: parser.Token{Literal: "VARCHAR"}},
			},
		},
	}

	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

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
			{
				Column:   "email",
				Operator: types.EQUAL,
				Value: types.Bind{
					Column:   "email",
					Position: 2,
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

	// Check that the struct has the correct number of fields
	if typeSpec, ok := structDecl.Specs[0].(*ast.TypeSpec); ok {
		if structType, ok := typeSpec.Type.(*ast.StructType); ok {
			if len(structType.Fields.List) != 2 {
				t.Errorf("Expected 2 fields, got %d", len(structType.Fields.List))
			}

			// Check first field (Id)
			field1 := structType.Fields.List[0]
			if len(field1.Names) == 0 || field1.Names[0].Name != "Id" {
				t.Errorf("Expected first field name to be 'Id', got '%s'", field1.Names[0].Name)
			}
			if field1.Tag == nil {
				t.Error("Expected first field to have a struct tag")
			} else if field1.Tag.Value != "`json:\"Id\"`" {
				t.Errorf("Expected first field tag to be '`json:\"Id\"`', got '%s'", field1.Tag.Value)
			}

			// Check second field (Email)
			field2 := structType.Fields.List[1]
			if len(field2.Names) == 0 || field2.Names[0].Name != "Email" {
				t.Errorf("Expected second field name to be 'Email', got '%s'", field2.Names[0].Name)
			}
			if field2.Tag == nil {
				t.Error("Expected second field to have a struct tag")
			} else if field2.Tag.Value != "`json:\"Email\"`" {
				t.Errorf("Expected second field tag to be '`json:\"Email\"`', got '%s'", field2.Tag.Value)
			}
		} else {
			t.Error("Expected StructType in struct declaration")
		}
	} else {
		t.Error("Expected TypeSpec in struct declaration")
	}
}
