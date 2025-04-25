package generate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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

type GeneratorBuilder struct {
	QueryPath  []byte
	Queries    []Query
	SchemaPath []byte
	Driver     Driver
}

func NewGenerator() *GeneratorBuilder {
	return &GeneratorBuilder{
		QueryPath:  []byte{},
		Queries:    []Query{},
		SchemaPath: []byte{},
		Driver:     "",
	}
}

// Generate code
func (g *GeneratorBuilder) Execute() error {
	err := g.LoadConfig()
	if err != nil {
		return err
	}

	return nil
}

// Read shogunc config file
func (g GeneratorBuilder) HasConfig() bool {
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
func (g *GeneratorBuilder) ParseConfig(fileContents []byte) error {
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

func (g *GeneratorBuilder) LoadConfig() error {
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
func (g *GeneratorBuilder) LoadSqlFiles() error {
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

func (g *GeneratorBuilder) ParseSqlFile(file *os.File) error {
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

func (g GeneratorBuilder) listSqlQueries() {
	for _, stmt := range g.Queries {
		fmt.Printf("%s\n", string(stmt.SQL))
	}
}
