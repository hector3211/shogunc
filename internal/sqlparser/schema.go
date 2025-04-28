package sqlparser

import (
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

type SchemaParser struct {
	Types        []SqlType
	CurrentTable *TableType
	InTable      bool
	CurrentEnum  *EnumType
	InEnum       bool
}

func NewSchemaParser() *SchemaParser {
	return &SchemaParser{}
}

func (p *SchemaParser) ParseLine(line string) error {
	if p.matchTableStart(line) {
		return nil
	}
	if p.InTable {
		return p.parseTableBody(line)
	}
	// Future: handle enums, indexes, etc.
	return nil
}

func (p *SchemaParser) matchTableStart(line string) bool {
	tableReg := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+"([a-zA-Z_][a-zA-Z0-9_]*)"`)
	if matches := tableReg.FindStringSubmatch(line); matches != nil {
		p.CurrentTable = &TableType{
			Name: []byte(matches[1]),
		}
		p.InTable = true
		return true
	}
	return false
}

func (p *SchemaParser) parseTableBody(line string) error {
	if strings.Contains(line, ")") {
		p.Types = append(p.Types, *p.CurrentTable)
		p.CurrentTable = nil
		p.InTable = false
		return nil
	}

	field, ok := parseFieldLine(line)
	if ok {
		p.CurrentTable.Field = append(p.CurrentTable.Field, *field)
	}
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
