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

func TestGenerateReturnType(t *testing.T) {
	schemaTypes := make(map[string]any)
	queryBlock := &types.QueryBlock{Name: "GetUser", Type: types.ONE}
	generator := NewSelectGenerator(schemaTypes, queryBlock)

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
	generator := NewSelectGenerator(schemaTypes, queryBlock)

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
	generator := NewSelectGenerator(schemaTypes, queryBlock)

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
	generator := NewSelectGenerator(schemaTypes, queryBlock)

	_, err := generator.inferDataType("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent table")
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
			if field1.Tag == nil || field1.Tag.Value != "`json:\"id\" db:\"id\"`" {
				t.Errorf("Expected first field tag to be '`json:\"id\" db:\"id\"`', got '%s'", field1.Tag.Value)
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

	expectedError := "[GO_GENERATOR][EXEC] no statement found"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}
