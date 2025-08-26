package codegen

import (
	"go/ast"
	"reflect"
	"shogunc/internal/parser"
	"shogunc/internal/types"
	"strings"
	"testing"
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

	// Check that function has context parameter (no params)
	if len(funcDecl.Type.Params.List) != 1 {
		t.Errorf("Expected 1 parameter (context only), got %d", len(funcDecl.Type.Params.List))
	}
	// Verify the context parameter
	ctxParam := funcDecl.Type.Params.List[0]
	if len(ctxParam.Names) == 0 || ctxParam.Names[0].Name != "ctx" {
		t.Errorf("Expected first parameter name 'ctx', got '%s'", ctxParam.Names[0].Name)
	}
}

func TestGenerateReturnType(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewGoGenerator(schemaTypes, queryBlock)

	selectStmt := &types.SelectStatement{TableName: "users"}

	// Test ONE query
	returnType := generator.generateReturnType(selectStmt.TableName, false)
	if ident, ok := returnType.(*ast.Ident); !ok || ident.Name != "User" {
		t.Errorf("Expected ast.Ident with name 'User', got %T with value %v", returnType, returnType)
	}

	// Test MANY query
	returnType = generator.generateReturnType(selectStmt.TableName, true)
	if arrayType, ok := returnType.(*ast.ArrayType); !ok {
		t.Errorf("Expected ast.ArrayType, got %T", returnType)
	} else if ident, ok := arrayType.Elt.(*ast.Ident); !ok || ident.Name != "User" {
		t.Errorf("Expected array element type to be ast.Ident with name 'User', got %T with value %v", arrayType.Elt, arrayType.Elt)
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
	expected := "id = params.Id"
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

func TestGenerateEnumType(t *testing.T) {
	enumType := &parser.Enum{
		Name:   "Role",
		Values: []string{"admin", "user", "moderator"},
	}

	typeDecl, constDecl, err := GenerateEnumType(enumType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check type declaration
	if typeDecl == nil {
		t.Error("Expected type declaration")
	}
	if typeDecl.Tok.String() != "type" {
		t.Errorf("Expected type declaration, got %s", typeDecl.Tok.String())
	}
	if typeSpec, ok := typeDecl.Specs[0].(*ast.TypeSpec); ok {
		if typeSpec.Name.Name != "Role" {
			t.Errorf("Expected type name 'Role', got '%s'", typeSpec.Name.Name)
		}
		if ident, ok := typeSpec.Type.(*ast.Ident); !ok || ident.Name != "string" {
			t.Error("Expected type to be string")
		}
	} else {
		t.Error("Expected TypeSpec")
	}

	// Check const declaration
	if constDecl == nil {
		t.Error("Expected const declaration")
	}
	if constDecl.Tok.String() != "const" {
		t.Errorf("Expected const declaration, got %s", constDecl.Tok.String())
	}
	if len(constDecl.Specs) != 3 {
		t.Errorf("Expected 3 const specs, got %d", len(constDecl.Specs))
	}

	// Check first constant (Admin)
	if valueSpec, ok := constDecl.Specs[0].(*ast.ValueSpec); ok {
		if len(valueSpec.Names) == 0 || valueSpec.Names[0].Name != "Role_Admin" {
			t.Errorf("Expected first constant name 'Role_Admin', got '%s'", valueSpec.Names[0].Name)
		}
		if valueSpec.Type == nil || valueSpec.Type.(*ast.Ident).Name != "Role" {
			t.Error("Expected constant type to be Role")
		}
		if len(valueSpec.Values) == 0 || valueSpec.Values[0].(*ast.BasicLit).Value != "\"admin\"" {
			t.Error("Expected constant value to be \"admin\"")
		}
	} else {
		t.Error("Expected ValueSpec")
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

	selectableType, err := GenerateTableType(tableType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check selectable type
	if selectableType == nil {
		t.Error("Expected selectable type declaration")
	}
	if selectableType.Tok.String() != "type" {
		t.Errorf("Expected type declaration, got %s", selectableType.Tok.String())
	}
	if typeSpec, ok := selectableType.Specs[0].(*ast.TypeSpec); ok {
		if typeSpec.Name.Name != "User" {
			t.Errorf("Expected type name 'User', got '%s'", typeSpec.Name.Name)
		}
		if structType, ok := typeSpec.Type.(*ast.StructType); ok {
			if len(structType.Fields.List) != 2 {
				t.Errorf("Expected 2 fields, got %d", len(structType.Fields.List))
			}
			// Check first field (Id)
			field1 := structType.Fields.List[0]
			if len(field1.Names) == 0 || field1.Names[0].Name != "Id" {
				t.Errorf("Expected first field name 'Id', got '%s'", field1.Names[0].Name)
			}
			if field1.Type.(*ast.Ident).Name != "string" {
				t.Error("Expected first field type to be string")
			}
			if field1.Tag == nil || field1.Tag.Value != "`db:\"id\"`" {
				t.Errorf("Expected first field tag to be '`db:\"id\"`', got '%s'", field1.Tag.Value)
			}
		} else {
			t.Error("Expected StructType")
		}
	} else {
		t.Error("Expected TypeSpec")
	}
}

func TestGenerateInsertableTableType(t *testing.T) {
	tableType := &parser.Table{
		Name: "users",
		Fields: []parser.Field{
			{Name: "id", DataType: parser.Token{Literal: "UUID"}, NotNull: true},
			{Name: "email", DataType: parser.Token{Literal: "VARCHAR"}, NotNull: false},
		},
	}

	insertableType, err := GenerateInsertableTableType(tableType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check insertable type
	if insertableType == nil {
		t.Error("Expected insertable type declaration")
	}
	if insertableType.Tok.String() != "type" {
		t.Errorf("Expected type declaration, got %s", insertableType.Tok.String())
	}
	if typeSpec, ok := insertableType.Specs[0].(*ast.TypeSpec); ok {
		if typeSpec.Name.Name != "NewUser" {
			t.Errorf("Expected type name 'NewUser', got '%s'", typeSpec.Name.Name)
		}
		if structType, ok := typeSpec.Type.(*ast.StructType); ok {
			if len(structType.Fields.List) != 2 {
				t.Errorf("Expected 2 fields, got %d", len(structType.Fields.List))
			}
			// Check first field (Id) - should be non-pointer since NotNull: true
			field1 := structType.Fields.List[0]
			if len(field1.Names) == 0 || field1.Names[0].Name != "Id" {
				t.Errorf("Expected first field name 'Id', got '%s'", field1.Names[0].Name)
			}
			if field1.Type.(*ast.Ident).Name != "string" {
				t.Error("Expected first field type to be string (non-pointer)")
			}
			// Check second field (Email) - should be pointer since NotNull: false
			field2 := structType.Fields.List[1]
			if len(field2.Names) == 0 || field2.Names[0].Name != "Email" {
				t.Errorf("Expected second field name 'Email', got '%s'", field2.Names[0].Name)
			}
			if starExpr, ok := field2.Type.(*ast.StarExpr); ok {
				if starExpr.X.(*ast.Ident).Name != "string" {
					t.Error("Expected second field type to be *string")
				}
			} else {
				t.Error("Expected second field type to be pointer")
			}
		} else {
			t.Error("Expected StructType")
		}
	} else {
		t.Error("Expected TypeSpec")
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
