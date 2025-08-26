package codegen

import (
	"fmt"
	"go/ast"
	"go/token"
	"shogunc/internal/parser"
	"shogunc/internal/types"
	"shogunc/utils"
	"strings"

	"github.com/hector3211/shogun"
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
			isMany := g.queryblock.Type == types.MANY
			return g.generateSelectFunc(selectStmt, isMany)
		}
		return nil, nil, fmt.Errorf("no select statement found")
	case types.EXEC:
		// return g.generateExecFunc(query),nil
		return nil, nil, fmt.Errorf("EXEC not implemented yet")
	default:
		return nil, nil, fmt.Errorf("unsupported query type: %s", g.queryblock.Type)
	}
}

// Select Statement -----------------------------------------------------------
func (g GoGenerator) generateSelectFunc(astStmt *types.SelectStatement, isMany bool) (*ast.FuncDecl, *ast.GenDecl, error) {
	paramStruct, paramTypeName, err := g.generateSelectParamStruct(astStmt)
	if err != nil {
		return nil, nil, err
	}

	returnTypeExpr := g.generateReturnType(astStmt.TableName, isMany)

	params := []*ast.Field{
		{
			Names: []*ast.Ident{ast.NewIdent("ctx")},
			Type: &ast.SelectorExpr{
				X:   ast.NewIdent("context"),
				Sel: ast.NewIdent("Context"),
			},
		},
	}
	if paramTypeName != "" {
		params = append(params, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent("params")},
			Type:  ast.NewIdent(paramTypeName),
		})
	}

	query := g.generateSelectQuery(astStmt)

	function := &ast.FuncDecl{
		Name: ast.NewIdent(g.queryblock.Name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: params},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: returnTypeExpr},
					{Type: ast.NewIdent("error")},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				query,
			},
		},
	}

	return function, paramStruct, nil
}

func (g GoGenerator) generateSelectParamStruct(astStmt *types.SelectStatement) (*ast.GenDecl, string, error) {
	fieldMap, err := g.inferDataType(astStmt.TableName) // Extract data type and its column types
	if err != nil {
		return nil, "", err
	}

	type paramInfo struct {
		name string
		typ  string
		pos  int
	}

	var paramList []paramInfo
	for _, condition := range astStmt.Conditions {
		if condition.Value.Position != 0 {
			// Normalize column name to lowercase for lookup
			normalizedColumn := strings.ToLower(condition.Value.Column)
			goType := fieldMap[normalizedColumn]
			paramName := utils.ToPascalCase(condition.Value.Column)
			paramList = append(paramList, paramInfo{
				name: paramName,
				typ:  goType,
				pos:  condition.Value.Position,
			})
		}
	}

	typeName := fmt.Sprintf("%sParams", g.queryblock.Name)

	// If no parameters, return empty struct name
	if len(paramList) == 0 {
		return nil, "", nil
	}

	// Sort parameters by position
	for i := 0; i < len(paramList)-1; i++ {
		for j := i + 1; j < len(paramList); j++ {
			if paramList[i].pos > paramList[j].pos {
				paramList[i], paramList[j] = paramList[j], paramList[i]
			}
		}
	}

	var fields []*ast.Field
	for _, param := range paramList {
		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(param.name)},
			Type:  ast.NewIdent(param.typ),
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`json:\"%s\"`", param.name),
			},
		}

		fields = append(fields, field)
	}

	paramStructDecl := &ast.GenDecl{
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

	return paramStructDecl, typeName, nil
}

func (g GoGenerator) generateSelectQuery(astStmt *types.SelectStatement) ast.Stmt {
	// Start SQL query
	queryStmt := shogun.NewSelectBuilder()
	if len(astStmt.Columns) == 1 && astStmt.Columns[0] == "*" {
		queryStmt.Select("*")
	} else {
		queryStmt.Select(strings.Join(astStmt.Columns, ","))
	}
	queryStmt.From(astStmt.TableName)

	conditions := make([]string, 0)
	for _, c := range astStmt.Conditions {
		if c.ChainOp != types.Illegal {
			conditions = append(conditions, g.shoguncNextOp(c.ChainOp))
			continue
		}
		conditions = append(conditions, g.shoguncConditionalOp(c))
	}

	// Only add WHERE clause if there are conditions
	if len(conditions) > 0 {
		queryStmt.Where(conditions...)
	}

	if astStmt.Limit != 0 {
		queryStmt.Limit(astStmt.Limit)
	}

	sql := queryStmt.Build()

	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("query")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.BasicLit{
			Kind:  token.STRING,
			Value: sql,
		}},
	}
}

func (g GoGenerator) generateReturnType(typeName string, isMany bool) ast.Expr {
	// Generate struct name from table name
	tableName := strings.TrimSuffix(typeName, "s") // Remove plural 's'
	structName := utils.Capitalize(tableName)

	if isMany {
		return &ast.ArrayType{
			Elt: ast.NewIdent(structName),
		}
	}
	return ast.NewIdent(structName)
}

func (g GoGenerator) shoguncNextOp(nextOp types.LogicalOp) string {
	switch nextOp {
	case types.And:
		return shogun.And()
	case types.Or:
		return shogun.Or()
	}
	return ""
}

func (g GoGenerator) shoguncConditionalOp(cond types.Condition) string {
	// Use lowercase column name for consistency with database schema
	columnName := strings.ToLower(cond.Column)

	if cond.Value.Position != 0 {
		fieldName := utils.ToPascalCase(cond.Value.Column)
		return fmt.Sprintf("%s = params.%s", columnName, fieldName)
	} else if cond.Value.Value != nil {
		// For literal values, use shogun functions
		switch cond.Operator {
		case types.EQUAL:
			return shogun.Equal(columnName, *cond.Value.Value)
		case types.NOTEQUAL:
			return shogun.NotEqual(columnName, *cond.Value.Value)
		case types.LESSTHAN:
			return shogun.LessThan(columnName, *cond.Value.Value)
		case types.GREATERTHAN:
			return shogun.GreaterThan(columnName, *cond.Value.Value)
		}
	} else {
		// Fallback for NULL values
		switch cond.Operator {
		case types.EQUAL:
			return fmt.Sprintf("%s IS NULL", columnName)
		case types.NOTEQUAL:
			return fmt.Sprintf("%s IS NOT NULL", columnName)
		}
	}

	return ""
}

func (g *GoGenerator) setSchemaTypes(types map[string]any) {
	g.schemaTypes = types
}

func (g GoGenerator) inferDataType(typeName string) (map[string]string, error) {
	dataMap := make(map[string]string)

	for k, v := range g.schemaTypes {
		if k == typeName {
			switch inferredType := v.(type) {
			case *parser.Table:
				for _, field := range inferredType.Fields {
					goType, err := parser.SqlToGoType(field.DataType)
					if err != nil || goType == "" {
						return map[string]string{}, fmt.Errorf("failed to convert field %s: %v", field.Name, err)
					}
					dataMap[field.Name] = goType
				}
			case *parser.Enum:
				dataMap[inferredType.Name] = "string"
			}
			break // Found the type
		}
	}

	// If no table was found, return an error
	if len(dataMap) == 0 {
		return nil, fmt.Errorf("table '%s' not found in schema", typeName)
	}

	return dataMap, nil
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
		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(fieldName)},
			Type:  ast.NewIdent(goType),
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`db:%q`", f.Name),
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
		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(fieldName)},
			Type:  fieldType,
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`db:%q`", f.Name),
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
