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
	Output  string `yaml:"output,omitempty"`
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
				Output:  "../../tmp/generated.sql.go",
			},
		},
		Types:   make(map[string]any),
		Imports: []string{},
	}
}

func (g *Generator) Execute() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if !g.hasConfig() {
		return fmt.Errorf("shogunc config file does not exists CWD: %s", cwd)
	}
	if err := g.loadConfig(); err != nil {
		return err
	}
	return nil
}

func (g Generator) hasConfig() bool {
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	return false
	// }

	// _, err = os.ReadFile(fmt.Sprintf("%s/shogunc.yml", cwd))
	_, err := os.ReadFile("../../shogunc.yml")

	return err == nil
}

func (g *Generator) loadConfig() error {
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }
	path := filepath.Join("../../", "shogunc.yml")

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

	g.Config.Sql.Queries = config.Sql.Queries
	g.Config.Sql.Schema = config.Sql.Schema
	g.Config.Sql.Driver = config.Sql.Driver
	g.Config.Sql.Output = config.Sql.Output

	return nil
}

func (g *Generator) LoadSchema() error {
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }
	// file := filepath.Join("../..", string(g.Config.Sql.Schema))
	fileContents, err := os.ReadFile(g.Config.Sql.Schema)
	if err != nil {
		return err
	}

	lexer := sqlparser.NewLexer(string(fileContents))
	ast := sqlparser.NewAst(lexer)
	if err := ast.ParseSchema(); err != nil {
		return err
	}

	var genContent strings.Builder
	// TODO: package name needed
	genContent.WriteString(`import (\n`)
	for _, pkg := range g.Imports {
		genContent.WriteString(fmt.Sprintf("\t%q", pkg))
		genContent.WriteString("\n")
	}
	genContent.WriteString(")")

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

			genContent.WriteString(content)
			genContent.WriteString("\n")

		case *sqlparser.EnumType:
			if _, ok := g.Types[t.Name]; !ok {
				g.Types[t.Name] = t
			}
			content, err := gogen.GenerateEnumType(t)
			if err != nil {
				return fmt.Errorf("[GENERATE] failed generating enum type %v", err)
			}

			genContent.WriteString(content)
			genContent.WriteString("\n")
		default:
			return errors.New("[GENERATE] load schema failed with invalid type")
		}
	}

	// fmt.Printf("contents being written: %s", &genContent)

	if genContent.String() == "" {
		return errors.New("[GENERATE] failed geenrating SQL types")
	}

	if err = os.WriteFile(g.Config.Sql.Output, []byte(genContent.String()), 0666); err != nil {
		return fmt.Errorf("[GENERATE] failed writing to file; path: %s error: %v", g.Config.Sql.Output, err)
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

		err = g.parseSqlFile(file)
		if err != nil {
			return fmt.Errorf("[GENERATE] failed to parse %s: %v", fullPath, err)
		}
	}
	return nil
}

func (g Generator) parseSqlFile(file *os.File) error {
	queryBlocks, err := g.extractSqlBlocks(file, file.Name())
	if err != nil {
		return err
	}

	var genContent strings.Builder
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

		// fmt.Printf("[GENERATE] data type in ParseSqlFile : %v\n\n", dataType)

		funcGen := gogen.NewFuncGenerator([]byte(qb.Name), qb.Type, dataType)
		for _, stmt := range ast.Statements {
			code, err := funcGen.GenerateFunction(stmt)
			if err != nil {
				return err
			}
			// fmt.Printf("statement: %s\n", code)
			genContent.WriteString(code + "\n")
		}
	}
	if genContent.String() == "" {
		return errors.New("[GENERATE] failed generating code")
	}

	if err = os.WriteFile(g.Config.Sql.Output, []byte(genContent.String()+"\n"), 0666); err != nil {
		return errors.New("[GENERATE] failed writing gogen code to file")
	}
	return nil
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

func (g *Generator) addImport(pkg string) error {
	g.Imports = append(g.Imports, pkg)
	fileBytes, err := os.ReadFile(g.Config.Sql.Output)
	if err != nil {
		return err
	}

	lines := strings.Split(string(fileBytes), "\n")
	var (
		newLines        []string
		inImportBlock   bool
		alreadyImported bool
	)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "import") {
			inImportBlock = true
			newLines = append(newLines, line)
			continue
		}

		if inImportBlock && trimmed == ")" {
			if !alreadyImported {
				newLines = append(newLines, fmt.Sprintf("\t%q", pkg))
			}
			inImportBlock = false
			newLines = append(newLines, line)
		}

		if inImportBlock && trimmed == fmt.Sprintf("%q", pkg) {
			alreadyImported = true
		}

		newLines = append(newLines, trimmed)
	}

	ctnt := strings.Join(newLines, "\n")

	if err := os.WriteFile(g.Config.Sql.Output, []byte(ctnt), 0666); err != nil {
		return err
	}

	return nil
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
