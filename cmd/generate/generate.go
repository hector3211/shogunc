package generate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SqlType any

type Field struct {
	Name      string  // "description"
	DataType  string  // "TEXT"
	NotNull   bool    // true if NOT NULL, false if nullable
	Default   *string // optional default value
	IsPrimary bool    // true if PRIMARY KEY
	IsUnique  bool    // true if UNIQUE
	Comment   *string // optional comment
}

type TableType struct {
	Name  []byte
	Field []Field
}

type EnumType struct {
	name   []byte
	values []string
}

type Driver string

const (
	SQLITE   Driver = "sqlite3"
	POSTGRES Driver = "postgres"
)

type Type string

const (
	EXEC Type = "exec"
	ONE  Type = "one"
	MANY Type = "many"
)

type Query struct {
	Name []byte
	Type Type
	SQL  []byte
}

type Generator struct {
	QueryPath  []byte
	Queries    []Query
	Types      []SqlType
	SchemaPath []byte
	Driver     Driver
}

func NewGenerator() *Generator {
	return &Generator{
		QueryPath:  []byte{},
		Queries:    []Query{},
		SchemaPath: []byte{},
		Driver:     "",
	}
}

// Generate code
func (g *Generator) Execute() error {
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

// parse for
// schema
// queries
// driver
func (g *Generator) ParseConfig(fileContents []byte) error {
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

	// fmt.Printf("queries: %s\n", g.QueryPath)
	// fmt.Printf("schema: %s\n", g.SchemaPath)
	// fmt.Printf("driver: %s\n", g.Driver)

	return nil
}

func (g *Generator) LoadConfig() error {
	if !g.HasConfig() {
		return fmt.Errorf("shogunc config file does not exists")
	}

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

// Read sql file / files
// Look for special tag: --name: GetUserById :one
func (g *Generator) LoadSqlFiles() error {
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }
	// directory := filepath.Join(cwd, string(g.QueryPath))
	// TODO: remove this after testing
	directory := fmt.Sprintf("../../%s", string(g.QueryPath))
	dirEntries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	// fmt.Printf("DIR: %s\n", directory)

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

		if err := g.ParseSqlFile(file); err != nil {
			return fmt.Errorf("failed to parse %s: %v", fullPath, err)
		}
	}

	// g.listSqlQueries()

	return nil
}

func (g *Generator) ParseSqlFile(file *os.File) error {
	var queries []Query
	scanner := bufio.NewScanner(file)

	// Shogunc Tag: -- name: GetUserById :one
	re := regexp.MustCompile(`--\s*name:\s*(\w+)\s*:(\w+)`)
	var current *Query
	var sqlBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Match on shogunc tag
		if matches := re.FindStringSubmatch(line); matches != nil {
			if current != nil {
				current.SQL = []byte(strings.TrimSpace(sqlBuilder.String()))
				queries = append(queries, *current)
				sqlBuilder.Reset()
			}

			// Initialize Tag with name & type
			current = &Query{
				Name: []byte(matches[1]),
				Type: Type(matches[2]),
			}
			// Jump to next line (SQL statement)
			continue
		}

		if current != nil {
			sqlBuilder.WriteString(line)
			sqlBuilder.WriteRune('\n')
		}
	}

	// Last SQL statement
	if current != nil {
		current.SQL = []byte(strings.TrimSpace(sqlBuilder.String()))
		queries = append(queries, *current)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	g.Queries = append(g.Queries, queries...)

	return nil
}

func (g Generator) LoadSchema() error {
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }
	// directory := filepath.Join(cwd, string(g.QueryPath))
	// TODO: remove this after testing
	directory := fmt.Sprintf("../../%s", string(g.SchemaPath))
	dirEntries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	// fmt.Printf("DIR: %s\n", directory)

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

		if err := g.ParseSqlFile(file); err != nil {
			return fmt.Errorf("failed to parse %s: %v", fullPath, err)
		}
	}

	return nil
}

func (g *Generator) ParseSchemaFile(file *os.File) error {
	var types []TableType
	scanner := bufio.NewScanner(file)

	// Shogunc Tag: -- name: GetUserById :one
	tablePattern := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+"([a-zA-Z_][a-zA-Z0-9_]*)"`)
	// enumPattern := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+"([a-zA-Z_][a-zA-Z0-9_]*)"\s+AS\s+ENUM\s*\(`)
	// foreignKeyPattern := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(([^)]+)\)\s+REFERENCES\s+"([a-zA-Z_][a-zA-Z0-9_]*)"\s*\(([a-zA-Z_][a-zA-Z0-9_]*)\)`)
	// indexPattern := regexp.MustCompile(`(?i)CREATE\s+INDEX\s+"([a-zA-Z_][a-zA-Z0-9_]*)"\s+ON\s+"([a-zA-Z_][a-zA-Z0-9_]*)"`)

	var current *TableType
	inTable := false

	for scanner.Scan() {
		line := scanner.Text()
		if matches := tablePattern.FindStringSubmatch(line); matches != nil {
			if len(matches) > 1 {
				current = &TableType{
					Name: []byte(matches[1]),
				}
				inTable = true
			}
			continue
		}

		if !inTable {
			continue
		}

		if strings.Contains(line, ")") {
			types = append(types, *current)
			current = nil
			inTable = false
			continue
		}

		field, ok := parseFieldLine(line)
		if ok {
			current.Field = append(current.Field, *field)
		}
	}

	g.Types = append(g.Types, types)

	return nil
}

func parseFieldLine(line string) (*Field, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "--") {
		return nil, false
	}

	line = strings.TrimSuffix(line, ",")

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, false
	}

	name := strings.Trim(parts[0], `"`)
	dataType := parts[1]

	notNull := false
	var defaultVal *string

	for i := 2; i < len(parts); i++ {
		p := strings.ToUpper(parts[i])

		if p == "NOT" && i+1 < len(parts) && strings.ToUpper(parts[i+1]) == "NULL" {
			notNull = true
			i++
		}

		if p == "DEFAULT" && i+1 < len(parts) {
			val := parts[i+1]
			defaultVal = &val
			i++
		}
	}

	return &Field{
		Name:     name,
		DataType: dataType,
		NotNull:  notNull,
		Default:  defaultVal,
	}, true
}

func (g Generator) listSqlQueries() {
	for _, stmt := range g.Queries {
		fmt.Printf("%s\n", string(stmt.SQL))
	}
}

func NewTableType(name []byte, fields []Field) *TableType {
	return &TableType{
		Name:  name,
		Field: fields,
	}
}

func NewEnumType(name []byte, values []string) *EnumType {
	return &EnumType{
		name:   name,
		values: values,
	}
}
