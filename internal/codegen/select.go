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

type SelectGenerator struct {
	schemaTypes map[string]any
	queryblock  *types.QueryBlock
}

func NewSelectGenerator(types map[string]any, queryBlock *types.QueryBlock) *SelectGenerator {
	return &SelectGenerator{schemaTypes: types, queryblock: queryBlock}
}

func (g *SelectGenerator) GenerateSelectFunc(astStmt *types.SelectStatement, isMany bool) (*ast.FuncDecl, *ast.GenDecl, error) {
	paramStruct, paramTypeName, err := g.generateSelectParamStruct(astStmt)
	if err != nil {
		return nil, nil, err
	}

	returnTypeExpr := g.generateReturnType(astStmt.TableName, isMany)

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

	bodyStmts, err := g.generateFunctionBody(astStmt, isMany, paramTypeName != "")
	if err != nil {
		return nil, nil, err
	}

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
		Body: &ast.BlockStmt{List: bodyStmts},
	}

	return function, paramStruct, nil
}

func (g *SelectGenerator) generateFunctionBody(astStmt *types.SelectStatement, isMany bool, hasParams bool) ([]ast.Stmt, error) {
	var stmts []ast.Stmt

	queryStmt := g.generateSelectQuery(astStmt)
	stmts = append(stmts, queryStmt)

	resultDecl := g.generateResultDecl(astStmt, isMany)
	resultStmt := &ast.DeclStmt{Decl: resultDecl}
	stmts = append(stmts, resultStmt)

	if isMany {
		manyStmts := g.generateManyQueryStmts(astStmt, hasParams)
		stmts = append(stmts, manyStmts...)
	} else {
		dbStmts := g.generateSelectDbQuery(astStmt)
		stmts = append(stmts, dbStmts...)
	}

	returnStmt := g.generateReturnStmt()
	stmts = append(stmts, returnStmt)

	return stmts, nil
}

func (g *SelectGenerator) generateManyQueryStmts(astStmt *types.SelectStatement, hasParams bool) []ast.Stmt {
	var stmts []ast.Stmt

	// (ctx,query)...
	args := []ast.Expr{ast.NewIdent("ctx"), ast.NewIdent("query")}
	if hasParams {
		args = append(args, g.generateSelectParamArgs(astStmt)...)
	}

	// q.db.query...
	queryCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.SelectorExpr{X: ast.NewIdent("q"), Sel: ast.NewIdent("db")},
			Sel: ast.NewIdent("Query"),
		},
		Args: args,
	}

	// rows, err := q.db.query...
	queryStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("rows"), ast.NewIdent("err")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{queryCall},
	}
	stmts = append(stmts, queryStmt)

	// if err != nil...
	errCheckStmt := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{ast.NewIdent("nil"), ast.NewIdent("err")},
				},
			},
		},
	}
	stmts = append(stmts, errCheckStmt)

	// defer rows.Close()...
	deferStmt := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("rows"),
				Sel: ast.NewIdent("Close"),
			},
		},
	}
	stmts = append(stmts, deferStmt)

	// rows.Next() loop...
	loopBody := g.generateManyLoopBody(astStmt)
	forStmt := &ast.ForStmt{
		Cond: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("rows"),
				Sel: ast.NewIdent("Next"),
			},
		},
		Body: &ast.BlockStmt{List: loopBody},
	}
	stmts = append(stmts, forStmt)

	//  if err := rows.Err(); err != nil...
	finalErrCheck := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("err")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("rows"),
						Sel: ast.NewIdent("Err"),
					},
				},
			},
		},
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{ast.NewIdent("nil"), ast.NewIdent("err")},
				},
			},
		},
	}
	stmts = append(stmts, finalErrCheck)

	return stmts
}

func (g *SelectGenerator) generateManyLoopBody(astStmt *types.SelectStatement) []ast.Stmt {
	var stmts []ast.Stmt

	// Declare loop variable: var item <Type>
	singleType := g.generateReturnType(astStmt.TableName, true)
	typeName := utils.TypeToString(singleType)
	if strings.HasPrefix(typeName, "[]") {
		typeName = strings.TrimPrefix(typeName, "[]")
	}

	// var item []typeName
	itemDecl := &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent("item")},
				Type:  ast.NewIdent(typeName),
			},
		},
	}
	stmts = append(stmts, &ast.DeclStmt{Decl: itemDecl})

	// err = rows.Scan(&item.Field1, &item.Field2, ...)
	scanArgs := g.generateScanArgs(astStmt)
	for i := range scanArgs {
		// Convert &result.Field to &item.Field
		if unaryExpr, ok := scanArgs[i].(*ast.UnaryExpr); ok {
			if selectorExpr, ok := unaryExpr.X.(*ast.SelectorExpr); ok {
				scanArgs[i] = &ast.UnaryExpr{
					Op: token.AND,
					X: &ast.SelectorExpr{
						X:   ast.NewIdent("item"),
						Sel: selectorExpr.Sel,
					},
				}
			}
		}
	}

	scanCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("rows"),
			Sel: ast.NewIdent("Scan"),
		},
		Args: scanArgs,
	}

	scanStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("err")},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{scanCall},
	}
	stmts = append(stmts, scanStmt)

	// Generate error check for Scan()
	scanErrCheck := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{ast.NewIdent("nil"), ast.NewIdent("err")},
				},
			},
		},
	}
	stmts = append(stmts, scanErrCheck)

	// result = append(result, item)
	appendCall := &ast.CallExpr{
		Fun:  ast.NewIdent("append"),
		Args: []ast.Expr{ast.NewIdent("result"), ast.NewIdent("item")},
	}

	appendStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("result")},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{appendCall},
	}
	stmts = append(stmts, appendStmt)

	return stmts
}

func (g *SelectGenerator) generateReturnStmt() *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			ast.NewIdent("result"),
			ast.NewIdent("nil"),
		},
	}
}

func (g *SelectGenerator) generateSelectParamStruct(astStmt *types.SelectStatement) (*ast.GenDecl, string, error) {
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
		fieldName := utils.ToProperPascalCase(param.name) // Use proper PascalCase (no underscores)
		// Convert original param name to snake_case for JSON tag
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

func (g *SelectGenerator) generateResultDecl(astStmt *types.SelectStatement, isMany bool) *ast.GenDecl {
	resultType := g.generateReturnType(astStmt.TableName, isMany)

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

	return resultVar
}

func (g *SelectGenerator) generateSelectQuery(astStmt *types.SelectStatement) ast.Stmt {
	queryBuilder := shogun.NewSelectBuilder()
	if len(astStmt.Columns) == 1 && astStmt.Columns[0] == "*" {
		queryBuilder.Select("*")
	} else {
		queryBuilder.Select(strings.Join(astStmt.Columns, ","))
	}
	queryBuilder.From(astStmt.TableName)

	conditions := make([]string, 0)
	for _, c := range astStmt.Conditions {
		if c.ChainOp != types.Illegal {
			conditions = append(conditions, g.shoguncNextOp(c.ChainOp))
		}
		conditions = append(conditions, g.shoguncConditionalOp(c))
	}

	if len(conditions) > 0 {
		queryBuilder.Where(conditions...)
	}

	if astStmt.Limit != 0 {
		queryBuilder.Limit(astStmt.Limit)
	}

	sql := queryBuilder.Build()
	// Remove quotes if present since we'll add them with BasicLit
	if len(sql) >= 2 && sql[0] == '"' && sql[len(sql)-1] == '"' {
		sql = sql[1 : len(sql)-1]
	}

	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("query")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("\"%s\"", sql),
		}},
	}
}

func (g *SelectGenerator) generateSelectDbQuery(astStmt *types.SelectStatement) []ast.Stmt {
	var stmts []ast.Stmt

	args := []ast.Expr{ast.NewIdent("ctx"), ast.NewIdent("query")}
	args = append(args, g.generateSelectParamArgs(astStmt)...)

	queryRowCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.SelectorExpr{X: ast.NewIdent("q"), Sel: ast.NewIdent("db")},
			Sel: ast.NewIdent("QueryRow"),
		},
		Args: args,
	}

	// Create row variable: row, err := q.db.QueryRow(...)
	rowDecl := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("row"), ast.NewIdent("err")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{queryRowCall},
	}
	stmts = append(stmts, rowDecl)

	// Add error check
	errCheck := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						g.generateZeroValue(astStmt.TableName, false),
						ast.NewIdent("err"),
					},
				},
			},
		},
	}
	stmts = append(stmts, errCheck)

	// Add .Scan() call: err := row.Scan(...)
	scanArgs := g.generateScanArgs(astStmt)
	scanCall := &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: ast.NewIdent("row"), Sel: ast.NewIdent("Scan")},
		Args: scanArgs,
	}

	scanStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("err")},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{scanCall},
	}
	stmts = append(stmts, scanStmt)

	return stmts
}

func (g *SelectGenerator) generateZeroValue(tableName string, isMany bool) ast.Expr {
	if isMany {
		return &ast.CompositeLit{
			Type: &ast.ArrayType{
				Elt: ast.NewIdent("User"), // This should be dynamic based on tableName
			},
		}
	}
	return ast.NewIdent("User{}") // This should be dynamic based on tableName
}

func (g *SelectGenerator) generateSelectParamArgs(astStmt *types.SelectStatement) []ast.Expr {
	type paramInfo struct {
		name string
		typ  string
		pos  int
	}

	var paramList []paramInfo

	for _, condition := range astStmt.Conditions {
		if condition.Value.Position != 0 {
			columnNameToLower := strings.ToLower(condition.Value.Column)
			fieldMap, _ := g.inferDataType(astStmt.TableName)
			goType := fieldMap[columnNameToLower]
			paramName := utils.ToProperPascalCase(condition.Value.Column)

			paramList = append(paramList, paramInfo{
				name: paramName,
				typ:  goType,
				pos:  condition.Value.Position,
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

func (g *SelectGenerator) generateScanArgs(astStmt *types.SelectStatement) []ast.Expr {
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

func (g *SelectGenerator) generateReturnType(typeName string, isMany bool) ast.Expr {
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

func (g *SelectGenerator) shoguncNextOp(nextOp types.LogicalOp) string {
	switch nextOp {
	case types.And:
		return shogun.And()
	case types.Or:
		return shogun.Or()
	}
	return ""
}

func (g *SelectGenerator) shoguncConditionalOp(cond types.Condition) string {
	// Use lowercase column name for consistency with database schema
	columnName := strings.ToLower(cond.Column)

	if cond.Value.Position != 0 {
		return fmt.Sprintf("%s = $%d", columnName, cond.Value.Position)
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

func (g *SelectGenerator) inferDataType(typeName string) (map[string]string, error) {
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
