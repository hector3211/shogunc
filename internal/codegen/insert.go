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

type InsertGenerator struct {
	schemaTypes map[string]any
	queryblock  *types.QueryBlock
}

func NewInsertGenerator(types map[string]any, queryBlock *types.QueryBlock) *InsertGenerator {
	return &InsertGenerator{schemaTypes: types, queryblock: queryBlock}
}

func (g InsertGenerator) GenerateInsertFunc(astStmt *types.InsertStatement) (*ast.FuncDecl, *ast.GenDecl, error) {
	paramStruct, paramTypeName, err := g.generateInsertParamStruct(astStmt)
	if err != nil {
		return nil, nil, err
	}

	// Determine return type based on RETURNING clause
	var results []*ast.Field
	hasReturning := len(astStmt.ReturningFields) > 0

	if hasReturning {
		returnType := g.generateReturnType(astStmt.TableName)
		results = []*ast.Field{
			{Type: returnType},
			{Type: ast.NewIdent("error")},
		}
	} else {
		results = []*ast.Field{
			{Type: ast.NewIdent("error")},
		}
	}

	params := []*ast.Field{
		{
			Names: []*ast.Ident{ast.NewIdent("q")},
			Type: &ast.StarExpr{
				X: ast.NewIdent("Queries"),
			},
		},
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
	bodyStmts, err := g.generateFunctionBody(astStmt, paramTypeName != "")
	if err != nil {
		return nil, nil, err
	}

	function := &ast.FuncDecl{
		Name: ast.NewIdent(g.queryblock.Name),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: params},
			Results: &ast.FieldList{List: results},
		},
		Body: &ast.BlockStmt{List: bodyStmts},
	}

	return function, paramStruct, nil
}

func (g InsertGenerator) generateInsertParamStruct(astStmt *types.InsertStatement) (*ast.GenDecl, string, error) {
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
	for _, condition := range astStmt.Values {
		if condition.Position != 0 {
			// Normalize column name to lowercase for lookup
			normalizedColumn := strings.ToLower(condition.Column)
			goType := fieldMap[normalizedColumn]
			paramName := utils.ToPascalCase(condition.Column)
			paramList = append(paramList, paramInfo{
				name: paramName,
				typ:  goType,
				pos:  condition.Position,
			})
		}
	}

	typeName := fmt.Sprintf("%sParams", g.queryblock.Name)

	if len(paramList) == 0 {
		return nil, "", nil
	}

	// Sort params by position
	for i := range len(paramList) - 1 {
		for j := i + 1; j < len(paramList); j++ {
			if paramList[i].pos > paramList[j].pos {
				paramList[i], paramList[j] = paramList[j], paramList[i]
			}
		}
	}

	var fields []*ast.Field
	for _, param := range paramList {
		fieldName := utils.ToProperPascalCase(param.name)
		jsonTag := utils.ToSnakeCase(param.name)
		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(fieldName)}, // Proper PascalCase for Go field name
			Type:  ast.NewIdent(param.typ),
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`json:\"%s\"`", jsonTag), // Use snake_case for JSON tag
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

func (g InsertGenerator) generateReturnType(typeName string) ast.Expr {
	// Generate struct name from table name
	tableName := strings.TrimSuffix(typeName, "s") // Remove plural 's'
	structName := utils.Capitalize(tableName)
	return ast.NewIdent(structName)
}

func (g InsertGenerator) generateFunctionBody(astStmt *types.InsertStatement, hasParams bool) ([]ast.Stmt, error) {
	var stmts []ast.Stmt

	queryStmt := g.generateInsertQuery(astStmt)
	stmts = append(stmts, queryStmt)

	resultDecl := g.generateResultDecl(astStmt)
	stmts = append(stmts, resultDecl)

	dbStmts := g.generateInsertDbQuery(astStmt)
	stmts = append(stmts, dbStmts...)

	returnStmt := g.generateReturnStmt(len(astStmt.ReturningFields) > 0)
	stmts = append(stmts, returnStmt)

	return stmts, nil
}

func (g InsertGenerator) generateInsertQuery(astStmt *types.InsertStatement) ast.Stmt {
	queryBuilder := shogun.NewInsertBuilder().
		Insert(astStmt.TableName).
		Columns(astStmt.Columns...)

	var values []any
	for idx, v := range astStmt.Values {
		if astStmt.Columns[idx] == v.Column {
			if v.Position != 0 {
				// Create proper bind parameter syntax ($1, $2, etc.)
				value := fmt.Sprintf("$%d", v.Position)
				values = append(values, value)
			} else {
				values = append(values, v.Value)
			}
		}
	}
	queryBuilder.Values(values...)
	sql := queryBuilder.Build()

	// Clean up any quoted bind parameters that the shogun library might add
	sql = utils.CleanBindParam(sql)

	// Todo: handle "onclonfict"
	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("query")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("\"%s\"", sql),
		}},
	}
}

func (g InsertGenerator) generateResultDecl(astStmt *types.InsertStatement) *ast.DeclStmt {
	resultType := g.generateReturnType(astStmt.TableName)

	var typeName string
	if ident, ok := resultType.(*ast.Ident); ok {
		typeName = ident.Name
	} else if arrayType, ok := resultType.(*ast.ArrayType); ok {
		if eltIdent, ok := arrayType.Elt.(*ast.Ident); ok {
			typeName = "[]" + eltIdent.Name
		}
	}

	resultVar := &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{
					ast.NewIdent("result"),
				},
				Type: ast.NewIdent(typeName),
			},
		},
	}
	resultStmt := &ast.DeclStmt{Decl: resultVar}

	return resultStmt
}

func (g InsertGenerator) generateInsertDbQuery(astStmt *types.InsertStatement) []ast.Stmt {
	var stmts []ast.Stmt

	// (ctx,query,...)
	args := []ast.Expr{ast.NewIdent("ctx"), ast.NewIdent("query")}
	// (ctx,query, params...)
	args = append(args, g.generateSelectParamArgs(astStmt)...)

	// Check if this INSERT has RETURNING clause
	hasReturning := len(astStmt.ReturningFields) > 0

	if hasReturning {
		// Use QueryRow for INSERT with RETURNING
		queryRowCall := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.SelectorExpr{X: ast.NewIdent("q"), Sel: ast.NewIdent("db")},
				Sel: ast.NewIdent("QueryRow"),
			},
			Args: args,
		}

		// Create row variable: row := q.db.QueryRow(ctx,query,params...)
		rowDecl := &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("row")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{queryRowCall},
		}
		stmts = append(stmts, rowDecl)

		//  row.Scan(...)
		scanArgs := g.generateScanArgs(astStmt)
		//  row.Scan(fields...)
		scanCall := &ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: ast.NewIdent("row"), Sel: ast.NewIdent("Scan")},
			Args: scanArgs,
		}

		// err = row.Scan(.fields...)
		scanStmt := &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("err")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{scanCall},
		}
		stmts = append(stmts, scanStmt)
	} else {
		// Use Exec for INSERT without RETURNING
		execCall := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.SelectorExpr{X: ast.NewIdent("q"), Sel: ast.NewIdent("db")},
				Sel: ast.NewIdent("Exec"),
			},
			Args: args,
		}

		// _, err := q.db.Exec(ctx,query,params...)
		execStmt := &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("_"), ast.NewIdent("err")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{execCall},
		}
		stmts = append(stmts, execStmt)
	}

	return stmts
}

func (g InsertGenerator) generateSelectParamArgs(astStmt *types.InsertStatement) []ast.Expr {
	type paramInfo struct {
		name string
		typ  string
		pos  int
	}

	var paramList []paramInfo

	for _, condition := range astStmt.Values {
		if condition.Position != 0 {
			columnNameToLower := strings.ToLower(condition.Column)
			fieldMap, _ := g.inferDataType(astStmt.TableName)
			goType := fieldMap[columnNameToLower]
			paramName := utils.ToProperPascalCase(condition.Column)

			paramList = append(paramList, paramInfo{
				name: paramName,
				typ:  goType,
				pos:  condition.Position,
			})
		}
	}

	// Sort by position
	for i := range len(paramList) - 1 {
		for j := i + 1; j < len(paramList); j++ {
			if paramList[i].pos > paramList[j].pos {
				paramList[i], paramList[j] = paramList[j],
					paramList[i]
			}
		}
	}

	var args []ast.Expr
	for _, param := range paramList {
		args = append(args, &ast.SelectorExpr{
			X:   ast.NewIdent("params"),
			Sel: ast.NewIdent(param.name),
		})
	}

	return args
}

func (g InsertGenerator) generateScanArgs(astStmt *types.InsertStatement) []ast.Expr {
	var scanArgs []ast.Expr
	var columns []string

	if len(astStmt.Columns) == 1 && astStmt.Columns[0] == "*" {
		// SELECT * case - get all columns from schema
		fieldMap, err := g.inferDataType(astStmt.TableName)
		if err != nil {
			return scanArgs
		}

		var columnNames []string
		for columnName := range fieldMap {
			columnNames = append(columnNames, columnName)
		}

		// Sort for consistent order
		for i := range len(columnNames) - 1 {
			for j := i + 1; j < len(columnNames); j++ {
				if columnNames[i] > columnNames[j] {
					columnNames[i], columnNames[j] = columnNames[j], columnNames[i]
				}
			}
		}

		columns = columnNames
	} else {
		columns = astStmt.Columns
	}

	// Generate &result.FieldName expressions for each column
	for _, column := range columns {
		fieldName := utils.ToProperPascalCase(column)

		scanArg := &ast.UnaryExpr{
			Op: token.AND,
			X: &ast.SelectorExpr{
				X:   ast.NewIdent("result"),
				Sel: ast.NewIdent(fieldName),
			},
		}

		scanArgs = append(scanArgs, scanArg)
	}

	return scanArgs
}

func (g InsertGenerator) generateReturnStmt(hasReturning bool) *ast.ReturnStmt {
	if hasReturning {
		return &ast.ReturnStmt{
			Results: []ast.Expr{
				ast.NewIdent("result"),
				ast.NewIdent("err"),
			},
		}
	} else {
		return &ast.ReturnStmt{
			Results: []ast.Expr{
				ast.NewIdent("err"),
			},
		}
	}
}

func (g InsertGenerator) inferDataType(typeName string) (map[string]string, error) {
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
