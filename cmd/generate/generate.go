package generate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"shogunc/internal/codegen/gogen"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
	"strings"
)

type Driver string

const (
	SQLITE   Driver = "sqlite3"
	POSTGRES Driver = "postgres"
)

type QueryBlock struct {
	Name     string // -name: GetUser
	Type     utils.Type
	SQL      string // sql query statement
	Filename string // for debug or error reporting
}

type Generator struct {
	QueryPath  []byte
	SchemaPath []byte
	Driver     Driver
	Types      map[string]any
	Imports    []string
}

func NewGenerator() *Generator {
	return &Generator{
		QueryPath:  []byte{},
		SchemaPath: []byte{},
		Driver:     "",
		Imports:    []string{"context"},
	}
}

// Generate code
func (g *Generator) Execute() error {
	if !g.HasConfig() {
		return fmt.Errorf("shogunc config file does not exists")
	}
	err := g.LoadConfig()
	if err != nil {
		return err
	}

	return nil
}

// Read shogunc config file
func (g Generator) HasConfig() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	_, err = os.ReadFile(fmt.Sprintf("%s/shogunc.yml", cwd))

	return err == nil
}

func (g *Generator) LoadConfig() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	configFile, err := os.ReadFile(fmt.Sprintf("%s/shogunc.yml", cwd))
	if err != nil {
		return err
	}

	g.ParseConfig(configFile)
	return nil
}

// parse for
// schema
// queries
// driver
func (g *Generator) ParseConfig(fileContents []byte) error {
	// NOTE: Go has yaml parser package
	lines := strings.Split(string(fileContents), "\n")
	matchers := map[string]struct{}{
		"queries": {},
		"schema":  {},
		"driver":  {},
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if _, ok := matchers[key]; !ok {
			continue
		}

		switch key {
		case "queries":
			g.QueryPath = []byte(val)
		case "schema":
			g.SchemaPath = []byte(val)
		case "driver":
			g.Driver = Driver(val)
		}
	}

	if len(g.QueryPath) == 0 {
		return fmt.Errorf("failed reading queries path")
	}
	if len(g.SchemaPath) == 0 {
		return fmt.Errorf("failed reading schema path")
	}
	if g.Driver == "" {
		return fmt.Errorf("failed reading sql driver")
	}

	g.LoadSchema()
	g.LoadSqlFiles()

	// fmt.Printf("queries: %s\n", g.QueryPath)
	// fmt.Printf("schema: %s\n", g.SchemaPath)
	// fmt.Printf("driver: %s\n", g.Driver)

	return nil
}

func (g *Generator) LoadSchema() error {
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }
	// directory := filepath.Join(cwd, string(g.QueryPath))
	file := fmt.Sprintf("../../%s", string(g.SchemaPath))
	fileContents, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	lexer := sqlparser.NewLexer(string(fileContents))
	ast := sqlparser.NewAst(lexer)
	if err := ast.Parse(); err != nil {
		return err
	}

	for _, datatype := range ast.Statements {
		switch t := datatype.(type) {
		case *sqlparser.TableType:
			if _, ok := g.Types[t.Name]; !ok {
				g.Types[t.Name] = t
			}
		case *sqlparser.EnumType:
			if _, ok := g.Types[t.Name]; !ok {
				g.Types[t.Name] = t
			}
		}
	}
	return nil
}

// Read sql file / files
// Look for special tag: --name: GetUserById :one
func (g Generator) LoadSqlFiles() error {
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }
	// directory := filepath.Join(cwd, string(g.QueryPath))
	directory := fmt.Sprintf("../../%s", string(g.QueryPath))
	dirEntries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue // Skip directories
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		fullPath := filepath.Join(directory, entry.Name())
		file, err := os.Open(fullPath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %v", fullPath, err)
		}
		defer file.Close()

		_, err = g.ParseSqlFile(file)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %v", fullPath, err)
		}
	}
	return nil
}

func (g Generator) ParseSqlFile(file *os.File) (string, error) {
	var bytes strings.Builder
	queryBlocks, err := g.extractSqlBlocks(file, file.Name())
	if err != nil {
		return "", err
	}

	for _, qb := range queryBlocks {
		lexer := sqlparser.NewLexer(qb.SQL)
		ast := sqlparser.NewAst(lexer)
		if err := ast.Parse(); err != nil {
			return "", fmt.Errorf("[GENERATE] failed parsing %s: %w", qb.Name, err)
		}

		dataType := g.inferType(qb.Name)
		if dataType == nil {
			return "", fmt.Errorf("[GENERATE] failed infering type for %s", qb.Name)
		}

		funcGen := gogen.NewFuncGenerator([]byte(qb.Name), qb.Type, dataType)
		for _, stmt := range ast.Statements {
			code, err := funcGen.GenerateFunction(stmt)
			if err != nil {
				return "", err
			}
			bytes.WriteString(code + "\n")
		}
	}
	return bytes.String(), nil
}

func (g Generator) extractSqlBlocks(file *os.File, fileName string) ([]QueryBlock, error) {
	scanner := bufio.NewScanner(file)
	tagReg := regexp.MustCompile(`--\s*name:\s*(\w+)\s*:(\w+)`) // -- name: GetUserById :one
	var blocks []QueryBlock

	var current *QueryBlock
	var sqlBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		// Match on shogunc tag
		if matches := tagReg.FindStringSubmatch(line); matches != nil {
			if current != nil {
				current.SQL = sqlBuilder.String()
				blocks = append(blocks, *current)
				sqlBuilder.Reset()
			}
			// Initialize Tag with name & type
			current = &QueryBlock{
				Name:     strings.ToUpper(matches[1][:1]) + matches[1][1:],
				Type:     utils.Type(matches[2]),
				Filename: fileName,
			}
			continue
		}

		if current != nil {
			sqlBuilder.WriteString(line)
			sqlBuilder.WriteRune('\n')
		}
	}

	// Last SQL statement
	if current != nil {
		current.SQL = sqlBuilder.String()
		blocks = append(blocks, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (g Generator) inferType(name string) any {
	if datType, ok := g.Types[name]; ok {
		return datType
	}
	return nil
}
