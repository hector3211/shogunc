package codegen

import (
	"fmt"
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

func (g GoGenerator) Generate(astStmt any) (string, error) {
	switch g.queryblock.Type {
	case types.ONE, types.MANY:
		if selectStmt, ok := astStmt.(*types.SelectStatement); ok {
			isMany := g.queryblock.Type == types.MANY
			return g.generateSelectFunc(selectStmt, isMany)
		}
		return "", fmt.Errorf("no select statement found")
	case types.EXEC:
		// return g.generateExecFunc(query),nil
		return "", fmt.Errorf("EXEC not implemented yet")
	default:
		return "", fmt.Errorf("unsupported query type: %s", g.queryblock.Type)
	}
}

// Select Statement -----------------------------------------------------------
func (g GoGenerator) generateSelectFunc(astStmt *types.SelectStatement, isMany bool) (string, error) {
	var buffer strings.Builder

	paramStruct, paramTypeName, err := g.generateSelectParamStruct(astStmt)
	if err != nil {
		return "", err
	}

	// Add parameter struct if it has fields
	if paramStruct != "" {
		buffer.WriteString(paramStruct)
		buffer.WriteString("\n\n")
	}

	// Generate function signature
	returnType := g.generateReturnType(astStmt, isMany)
	if paramTypeName != "" {
		buffer.WriteString(fmt.Sprintf("func %s(ctx context.Context, params %s) %s {\n",
			g.queryblock.Name, paramTypeName, returnType))
	} else {
		buffer.WriteString(fmt.Sprintf("func %s(ctx context.Context) %s {\n",
			g.queryblock.Name, returnType))
	}
	query := g.generateSelectQuery(astStmt)
	buffer.WriteString(query)

	return buffer.String(), nil
}

func (g GoGenerator) generateSelectParamStruct(astStmt *types.SelectStatement) (string, string, error) {
	fieldMap, err := g.inferDataType(astStmt.TableName) // Extract data type and its column types
	if err != nil {
		return "", "", err
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
		return "", "", nil
	}

	var buffer strings.Builder
	buffer.WriteString(fmt.Sprintf("type %s struct {\n", typeName))

	for columnName, goType := range params {
		buffer.WriteString(fmt.Sprintf("\t%s %s `db:\"%s\"`\n",
			columnName, goType, strings.ToLower(columnName)))
	}

	buffer.WriteString("}\n")
	return buffer.String(), typeName, nil
}

func (g GoGenerator) generateSelectQuery(astStmt *types.SelectStatement) string {
	var buffer strings.Builder

	// Start SQL query
	buffer.WriteString("\tquery := \"")
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

	buffer.WriteString(queryStmt.Build())
	buffer.WriteString("\"\n")

	// Add basic function body
	buffer.WriteString("\t// TODO: Implement database query execution\n")
	buffer.WriteString("\treturn nil, nil\n")

	return buffer.String()
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

// Db package --------------------------------------------------------------------
func (g GoGenerator) GenerateDB(packageName string) string {
	var buffer strings.Builder

	buffer.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	buffer.WriteString(`import (
	"context"
	"database/sql"
)
`)
	buffer.WriteString(`type DBX interface {
		Exec(context.Context, string, ...any) error
		Query(context.Context, string, ...any) (*sql.Rows, error)
		QueryRow(context.Context, string, ...any) (*sql.Row, error)
}

`)

	buffer.WriteString(`type Queries struct {
		db DBX
}

`)

	buffer.WriteString(`func New(db DBX) *Queries {
		return &Queries{db: db}
}

`)
	return buffer.String()
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
		buffer.WriteString(fmt.Sprintf("\t%s %s = %q\n", value, enumType.Name, v))
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
