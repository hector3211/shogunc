package codegen

import (
	"fmt"
	"go/ast"
	"go/token"
	"shogunc/internal/parser"
	"shogunc/internal/types"
	"shogunc/utils"
	"strings"
)

type GoGenerator struct {
	schemaTypes map[string]any
	queryblock  *types.QueryBlock
}

func NewGoGenerator(types map[string]any, queryBlock *types.QueryBlock) *GoGenerator {
	return &GoGenerator{schemaTypes: types, queryblock: queryBlock}
}

func (g GoGenerator) Generate(astStmt any) (*ast.FuncDecl, *ast.GenDecl, error) {
	switch g.queryblock.Type {
	case types.ONE, types.MANY:
		if selectStmt, ok := astStmt.(*types.SelectStatement); ok {
			selectGen := NewSelectGenerator(g.schemaTypes, g.queryblock)
			isMany := g.queryblock.Type == types.MANY
			return selectGen.GenerateSelectFunc(selectStmt, isMany)
		}
		return nil, nil, fmt.Errorf("no select statement found")
	case types.EXEC:
		// return g.generateExecFunc(query),nil
		return nil, nil, fmt.Errorf("EXEC not implemented yet")
	default:
		return nil, nil, fmt.Errorf("unsupported query type: %s", g.queryblock.Type)
	}
}

// Generate Types ---------------------------------------------------------------------
func GenerateEnumType(enumType *parser.Enum) (*ast.GenDecl, *ast.GenDecl, error) {
	typeDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(enumType.Name),
				Type: ast.NewIdent("string"),
			},
		},
	}

	var constSpecs []ast.Spec
	for _, v := range enumType.Values {
		value := utils.ToPascalCase(v)
		// Prefix constant name with enum type to avoid conflicts
		constantName := enumType.Name + "_" + value
		constSpec := &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent(constantName)},
			Type:  ast.NewIdent(enumType.Name),
			Values: []ast.Expr{&ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", v),
			}},
		}
		constSpecs = append(constSpecs, constSpec)
	}

	constDecl := &ast.GenDecl{
		Tok:   token.CONST,
		Specs: constSpecs,
	}

	return typeDecl, constDecl, nil
}

func GenerateTableType(tableType *parser.Table) (*ast.GenDecl, error) {
	last := strings.ToLower(tableType.Name[len(tableType.Name)-1:])
	base := tableType.Name
	if last == "s" {
		base = tableType.Name[:len(tableType.Name)-1]
	}

	typeName := utils.Capitalize(base)

	var fields []*ast.Field
	for _, f := range tableType.Fields {
		goType, err := parser.SqlToGoType(f.DataType)
		if err != nil {
			return nil, fmt.Errorf("[BUILDER] failed parsing %s %v to GO type", f.DataType.Literal, f.DataType.Type)
		}

		fieldName := utils.ToPascalCase(f.Name)
		// Convert PascalCase to snake_case for JSON tag
		jsonTag := utils.ToSnakeCase(fieldName)
		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(fieldName)}, // Keep PascalCase for Go field name
			Type:  ast.NewIdent(goType),
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`json:\"%s\" db:%q`", jsonTag, f.Name), // Add JSON tag with snake_case
			},
		}
		fields = append(fields, field)
	}

	selectTypeDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(typeName),
				Type: &ast.StructType{
					Fields: &ast.FieldList{List: fields},
				},
			},
		},
	}

	return selectTypeDecl, nil
}

func GenerateInsertableTableType(tableType *parser.Table) (*ast.GenDecl, error) {
	last := strings.ToLower(tableType.Name[len(tableType.Name)-1:])
	base := tableType.Name
	if last == "s" {
		base = tableType.Name[:len(tableType.Name)-1]
	}

	typeName := "New" + utils.Capitalize(base)

	var fields []*ast.Field
	for _, f := range tableType.Fields {
		goType, err := parser.SqlToGoType(f.DataType)
		if err != nil {
			return nil, fmt.Errorf("[BUILDER] failed parsing %s to GO type", f.DataType.Literal)
		}

		var fieldType ast.Expr
		if !f.NotNull {
			fieldType = &ast.StarExpr{
				X: ast.NewIdent(goType),
			}
		} else {
			fieldType = ast.NewIdent(goType)
		}

		fieldName := utils.ToPascalCase(f.Name)
		// Convert PascalCase to snake_case for JSON tag
		jsonTag := utils.ToSnakeCase(fieldName)
		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(fieldName)}, // Keep PascalCase for Go field name
			Type:  fieldType,
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`json:\"%s\" db:%q`", jsonTag, f.Name), // Add JSON tag with snake_case
			},
		}
		fields = append(fields, field)
	}

	insertTypeDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(typeName),
				Type: &ast.StructType{
					Fields: &ast.FieldList{List: fields},
				},
			},
		},
	}

	return insertTypeDecl, nil
}
