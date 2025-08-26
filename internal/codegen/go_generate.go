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

	returnType := g.generateReturnType(astStmt, isMany)
	params := make([]*ast.Field, 0)
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
					{Type: ast.NewIdent(returnType)},
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

	params := make(map[string]string)
	for _, condition := range astStmt.Conditions {
		if condition.Value.Position != 0 {
			// Normalize column name to lowercase for lookup
			normalizedColumn := strings.ToLower(condition.Value.Column)
			goType := fieldMap[normalizedColumn]
			paramName := utils.ToPascalCase(condition.Value.Column)
			params[paramName] = goType
		}
	}

	typeName := fmt.Sprintf("%sParams", g.queryblock.Name)

	// If no parameters, return empty struct name
	if len(params) == 0 {
		return nil, "", nil
	}

	var fields []*ast.Field
	for columnName, goType := range params {
		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(columnName)},
			Type:  ast.NewIdent(goType),
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`json:\"%s\"`", columnName),
			},
		}

		fields = append(fields, field)
	}

	paramStructDel := &ast.GenDecl{
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

	return paramStructDel, typeName, nil
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

func (g GoGenerator) generateReturnType(astStmt *types.SelectStatement, isMany bool) string {
	// Generate struct name from table name
	tableName := strings.TrimSuffix(astStmt.TableName, "s") // Remove plural 's'
	structName := utils.Capitalize(tableName)

	if isMany {
		return fmt.Sprintf("([]%s, error)", structName)
	}
	return fmt.Sprintf("(%s, error)", structName)
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
		// For bind parameters, create the condition manually to avoid shogun adding quotes
		paramPlaceholder := fmt.Sprintf("$%d", cond.Value.Position)
		switch cond.Operator {
		case types.EQUAL:
			return fmt.Sprintf("%s = %s", columnName, paramPlaceholder)
		case types.NOTEQUAL:
			return fmt.Sprintf("%s != %s", columnName, paramPlaceholder)
		case types.LESSTHAN:
			return fmt.Sprintf("%s < %s", columnName, paramPlaceholder)
		case types.GREATERTHAN:
			return fmt.Sprintf("%s > %s", columnName, paramPlaceholder)
		}
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

func (g GoGenerator) inferDataType(table string) (map[string]string, error) {
	dataMap := make(map[string]string)

	for k, v := range g.schemaTypes {
		if k == table {
			switch inferredType := v.(type) {
			case *parser.Table:
				// Extract field types from table schema
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
			break // Found the table, no need to continue
		}
	}

	// If no table was found, return an error
	if len(dataMap) == 0 {
		return nil, fmt.Errorf("table '%s' not found in schema", table)
	}

	return dataMap, nil
}

// Generate Types ---------------------------------------------------------------------
func GenerateEnumType(enumType *parser.Enum) (string, error) {
	var buffer strings.Builder

	buffer.WriteString(fmt.Sprintf("type %s string\n\n", enumType.Name))
	buffer.WriteString("const (\n")

	for _, v := range enumType.Values {
		value := utils.ToPascalCase(v)
		// Prefix constant name with enum type to avoid conflicts
		constantName := enumType.Name + "_" + value
		buffer.WriteString(fmt.Sprintf("\t%s %s = %q\n", constantName, enumType.Name, v))
	}

	buffer.WriteString(")\n")
	return buffer.String(), nil
}

func GenerateTableType(tableType *parser.Table) (string, error) {
	var buffer strings.Builder

	// Generate selectable type
	last := strings.ToLower(tableType.Name[len(tableType.Name)-1:])
	base := tableType.Name
	if last == "s" {
		base = tableType.Name[:len(tableType.Name)-1]
	}

	typeName := utils.Capitalize(base)

	// Selectable type
	buffer.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
	for _, f := range tableType.Fields {
		goDataType, err := parser.SqlToGoType(f.DataType)
		if err != nil {
			return "", fmt.Errorf("[BUILDER] failed parsing %s %v to GO type", f.DataType.Literal, f.DataType.Type)
		}

		fieldName := utils.ToPascalCase(f.Name)
		jsonTag := fmt.Sprintf("`db:%q`", f.Name)

		buffer.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, goDataType, jsonTag))
	}
	buffer.WriteString("}\n\n")

	// Insertable type
	buffer.WriteString(fmt.Sprintf("type New%s struct {\n", typeName))
	for _, f := range tableType.Fields {
		goDataType, err := parser.SqlToGoType(f.DataType)
		if err != nil {
			return "", fmt.Errorf("[BUILDER] failed parsing %s to GO type", f.DataType.Literal)
		}

		fieldType := goDataType
		if !f.NotNull {
			fieldType = "*" + goDataType
		}

		fieldName := utils.ToPascalCase(f.Name)
		jsonTag := fmt.Sprintf("`db:%q`", f.Name)

		buffer.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag))
	}
	buffer.WriteString("}\n")

	return buffer.String(), nil
}
