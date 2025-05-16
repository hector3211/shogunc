package generate

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"shogunc/internal/codegen/gogen"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
	"strings"

	"gopkg.in/yaml.v3"
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

type SqlConfig struct {
	Queries string `yaml:"queries"`
	Schema  string `yaml:"schema"`
	Driver  Driver `yaml:"driver"`
}

type ShogunConfig struct {
	Sql SqlConfig `yaml:"sql"`
}

type Generator struct {
	Config  ShogunConfig
	Types   map[string]any
	Imports []string
}

func NewGenerator() *Generator {
	return &Generator{
		Config: ShogunConfig{
			Sql: SqlConfig{
				Queries: "",
				Schema:  "",
				Driver:  "",
			},
		},
		Types:   make(map[string]any),
		Imports: []string{"context"},
	}
}

func (g *Generator) Execute() error {
	if !g.hasConfig() {
		return fmt.Errorf("shogunc config file does not exists")
	}
	err := g.loadConfig()
	if err != nil {
		return err
	}

	return nil
}

func (g Generator) hasConfig() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	_, err = os.ReadFile(fmt.Sprintf("%s/shogunc.yml", cwd))

	return err == nil
}

func (g *Generator) loadConfig() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(cwd, "shogunc.yml")

	configFile, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("[GENERATE] failed to read config file at %s: %w", path, err)
	}

	return g.parseConfig(configFile)
}

func (g *Generator) parseConfig(fileContents []byte) error {
	var config ShogunConfig
	if err := yaml.Unmarshal(fileContents, &config); err != nil {
		return fmt.Errorf("[GENERATE] invalid config: %v", err)
	}

	if config.Sql.Queries == "" {
		return fmt.Errorf("[GENERATE] failed reading queries path")
	}
	if config.Sql.Schema == "" {
		return fmt.Errorf("[GENERATE] failed reading schema path")
	}
	if config.Sql.Driver == "" {
		return fmt.Errorf("[GENERATE] failed reading sql driver")
	}

	g.Config.Sql.Queries = config.Sql.Queries
	g.Config.Sql.Schema = config.Sql.Schema
	g.Config.Sql.Driver = config.Sql.Driver

	return nil
}

func (g *Generator) LoadSchema() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	file := filepath.Join(cwd, string(g.Config.Sql.Schema))
	fileContents, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	lexer := sqlparser.NewLexer(string(fileContents))
	ast := sqlparser.NewAst(lexer)
	if err := ast.ParseSchema(); err != nil {
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
		default:
			return errors.New("[GENERATE] load schema failed with invalid type")
		}
	}
	return nil
}

// Read sql file / files
// Look for special tag: --name: GetUserById :one
func (g Generator) LoadSqlFiles() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	directory := filepath.Join(cwd, string(g.Config.Sql.Queries))
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
			return fmt.Errorf("[GENERATE] failed to open %s: %v", fullPath, err)
		}
		defer file.Close()

		_, err = g.parseSqlFile(file)
		if err != nil {
			return fmt.Errorf("[GENERATE] failed to parse %s: %v", fullPath, err)
		}
	}
	return nil
}

func (g Generator) parseSqlFile(file *os.File) (string, error) {
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

		dataType := g.inferType(qb.SQL)
		if dataType == nil {
			return "", fmt.Errorf("[GENERATE] failed infering type for %s\n SQL: %s", qb.Name, qb.SQL)
		}

		// fmt.Printf("[GENERATE] data type in ParseSqlFile : %v\n\n", dataType)

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

	// TODO: clean this logic up
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

func (g Generator) inferType(sql string) any {
	tokens := strings.Fields(sql)
	for _, k := range tokens {
		key := strings.Trim(k, ";,()")
		if datType, ok := g.Types[key]; ok {
			return datType
		}
	}
	return nil
}
