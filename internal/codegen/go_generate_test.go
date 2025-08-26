package codegen

import (
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

	result, err := generator.generateSelectFunc(selectStmt, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that parameter struct is generated
	if !strings.Contains(result, "type GetUserParams struct") {
		t.Error("Expected parameter struct to be generated")
	}

	// Check that function signature is correct
	if !strings.Contains(result, "func GetUser(ctx context.Context, params GetUserParams) (User, error)") {
		t.Error("Expected correct function signature")
	}

	// Check that query is generated
	if !strings.Contains(result, `query := "SELECT * FROM users WHERE id = $1;"`) {
		t.Error("Expected correct SQL query")
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

	structDef, typeName, err := generator.generateSelectParamStruct(selectStmt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if typeName != "GetUserParams" {
		t.Errorf("Expected typeName to be 'GetUserParams', got '%s'", typeName)
	}

	if !strings.Contains(structDef, "type GetUserParams struct") {
		t.Error("Expected struct definition")
	}

	if !strings.Contains(structDef, "Id string `db:\"id\"`") {
		t.Error("Expected Id field with correct type and tag")
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

	structDef, typeName, err := generator.generateSelectParamStruct(selectStmt)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if typeName != "" {
		t.Errorf("Expected typeName to be empty when no parameters, got '%s'", typeName)
	}

	if structDef != "" {
		t.Error("Expected empty struct definition for no parameters")
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

	result, err := generator.generateSelectFunc(selectStmt, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that NO parameter struct is generated
	if strings.Contains(result, "type GetUsersParams struct") {
		t.Error("Expected no parameter struct to be generated")
	}

	// Check that function signature has NO params parameter
	if !strings.Contains(result, "func GetUsers(ctx context.Context) ([]User, error)") {
		t.Errorf("Expected function signature without params parameter, got: %s", result)
	}

	// Check that query is generated
	if !strings.Contains(result, `query := "SELECT * FROM users;"`) {
		t.Error("Expected correct SQL query")
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

	_, err := generator.Generate(nil)
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

	_, err := generator.Generate(nil)
	if err == nil {
		t.Error("Expected error for EXEC type")
	}

	expectedError := "EXEC not implemented yet"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}
