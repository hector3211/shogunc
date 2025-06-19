package generate

import (
	"bufio"
	"bytes"
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

// NOTE: Debug feature
type ErrMsg string

type ErrorLogger struct {
	ErrMsg   ErrMsg
	Position int
}

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

// schema: schema.sql
// queries: queries
// driver: sqlite3
// output: /tmp/generated.sql.go

type SqlConfig struct {
	Schema  string `yaml:"schema"`
	Queries string `yaml:"queries"`
	Driver  Driver `yaml:"driver"`
	Output  string `yaml:"output,omitempty"`
}

type ShogunConfig struct {
	Sql SqlConfig `yaml:"sql"`
}

type Generator struct {
	Config      ShogunConfig
	Types       map[string]any
	Imports     []string
	tagRegex    *regexp.Regexp
	outputCache *bytes.Buffer
}

func NewGenerator() *Generator {
	return &Generator{
		Config: ShogunConfig{
			Sql: SqlConfig{
				Queries: "",
				Schema:  "",
				Driver:  "",
				// Output:  fmt.Sprintf("%s/generated.sql.go",cwd),
				Output: "../../tmp/generated.sql.go",
			},
		},
		Types:       make(map[string]any),
		Imports:     []string{"context"},
		tagRegex:    regexp.MustCompile(`--\s*name:\s*(\w+)\s*:(\w+)`),
		outputCache: &bytes.Buffer{},
	}
}

func (g *Generator) Execute(cwd string) error {
	if !g.hasConfig(cwd) {
		return fmt.Errorf("no shogunc.yml exists CWD: %s", cwd)
	}

	if err := g.loadConfig(cwd); err != nil {
		return err
	}

	if err := g.LoadSchema(); err != nil {
		return err
	}

	if err := g.LoadSqlFiles(); err != nil {
		return err
	}

	return g.writeOutput()
}

func (g Generator) hasConfig(cwd string) bool {
	_, err := os.ReadFile(filepath.Join(cwd, "shogunc.yml"))
	return err == nil
}

func (g *Generator) loadConfig(cwd string) error {
	path := filepath.Join(cwd, "shogunc.yml")

	configFile, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("[GENERATE] failed to read config file at %s: %w", path, err)
	}

	var config ShogunConfig
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return fmt.Errorf("[GENERATE] invalid config: %v", err)
	}

	if config.Sql.Queries == "" {
		return errors.New("[GENERATE] failed reading queries path")
	}
	if config.Sql.Schema == "" {
		return errors.New("[GENERATE] failed reading schema path")
	}
	if config.Sql.Driver == "" {
		return errors.New("[GENERATE] failed reading sql driver")
	}
	if config.Sql.Output == "" {
		return errors.New("[GENERATE] failed reading sql config output")
	}

	g.Config.Sql = config.Sql
	return nil
}

func (g *Generator) LoadSchema() error {
	fileContents, err := os.ReadFile(g.Config.Sql.Schema)
	if err != nil {
		return err
	}

	lexer := sqlparser.NewLexer(string(fileContents))
	ast := sqlparser.NewAst(lexer)
	if err := ast.ParseSchema(); err != nil {
		return err
	}

	var genContent bytes.Buffer
	genContent.WriteString(`import (\n`)
	for _, pkg := range g.Imports {
		genContent.WriteString(fmt.Sprintf("\t%q\n", pkg))
	}
	genContent.WriteString(")\n")

	for _, datatype := range ast.Statements {
		switch t := datatype.(type) {
		case *sqlparser.TableType:
			if _, ok := g.Types[t.Name]; !ok {
				g.Types[t.Name] = t
			}

			content, err := gogen.GenerateTableType(t)
			if err != nil {
				return fmt.Errorf("[GENERATE] failed generating table type %v", err)
			}

			genContent.WriteString(content + "\n")

		case *sqlparser.EnumType:
			if _, ok := g.Types[t.Name]; !ok {
				g.Types[t.Name] = t
			}
			content, err := gogen.GenerateEnumType(t)
			if err != nil {
				return fmt.Errorf("[GENERATE] failed generating enum type %v", err)
			}

			genContent.WriteString(content + "\n")
		default:
			return errors.New("[GENERATE] load schema failed with invalid type")
		}
	}

	if genContent.Len() == 0 {
		return errors.New("[GENERATE] failed generating SQL types")
	}

	g.outputCache.Write(genContent.Bytes())
	return nil
}

func (g *Generator) LoadSqlFiles() error {
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

		err = g.parseSqlFile(file)
		if err != nil {
			return fmt.Errorf("[GENERATE] failed to parse %s: %v", fullPath, err)
		}
	}
	return nil
}

func (g *Generator) parseSqlFile(file *os.File) error {
	queryBlocks, err := g.extractSqlBlocks(file, file.Name())
	if err != nil {
		return err
	}

	var genContent bytes.Buffer
	for _, qb := range queryBlocks {
		lexer := sqlparser.NewLexer(qb.SQL)
		ast := sqlparser.NewAst(lexer)
		if err := ast.Parse(); err != nil {
			return fmt.Errorf("[GENERATE] failed parsing %s: %w", qb.Name, err)
		}

		dataType := g.inferType(qb.SQL)
		if dataType == nil {
			return fmt.Errorf("[GENERATE] failed infering type for %s\n SQL: %s", qb.Name, qb.SQL)
		}

		funcGen := gogen.NewFuncGenerator([]byte(qb.Name), qb.Type, dataType)
		for _, stmt := range ast.Statements {
			code, err := funcGen.GenerateFunction(stmt)
			if err != nil {
				return err
			}
			genContent.WriteString(code + "\n")
		}
	}
	if genContent.String() == "" {
		return errors.New("[GENERATE] failed generating code")
	}

	g.outputCache.WriteString(genContent.String())
	return nil
}

func (g *Generator) extractSqlBlocks(file *os.File, fileName string) ([]QueryBlock, error) {
	scanner := bufio.NewScanner(file)
	var blocks []QueryBlock

	var current *QueryBlock
	var sqlBuilder bytes.Buffer

	// TODO: clean this logic up
	for scanner.Scan() {
		line := scanner.Text()
		// Match on shogunc tag
		if matches := g.tagRegex.FindStringSubmatch(line); matches != nil {
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

func (g Generator) writeOutput() error {
	if g.outputCache.Len() == 0 {
		return errors.New("[GENERATE] no content to write")
	}

	return os.WriteFile(g.Config.Sql.Output, g.outputCache.Bytes(), 0666)
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
