package parser

import (
	"fmt"
	"strings"
	"time"
)

type TokenType string

const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"
	STRING  TokenType = "STRING"
	COMMENT TokenType = "COMMENT"

	// Identifiers + Literals
	IDENT     TokenType = "IDENT" // foobar
	DEFAULT   TokenType = "DEFAULT"
	UUID      TokenType = "UUID"
	INT       TokenType = "INT" // 12345
	BIGINT    TokenType = "BIGINT"
	SMALLINT  TokenType = "SMALLINT"
	DECIMAL   TokenType = "DECIMAL"
	VARCHAR   TokenType = "VARCAHR"
	TEXT      TokenType = "TEXT"
	BOOLEAN   TokenType = "BOOLEAN"
	TIMESTAMP TokenType = "TIMESTAMP"
	DATE      TokenType = "DATE"
	// Input Identifier
	BINDPARAM   TokenType = "$"
	PLACEHOLDER TokenType = "PLACEHOLDER"

	// Operators
	ASSIGN  TokenType = "="
	ASTERIK TokenType = "*"

	// Delimiters
	COMMA     TokenType = ","
	SEMICOLON TokenType = ";"
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"

	// Data Query
	SELECT   TokenType = "SELECT"
	FROM     TokenType = "FROM"
	WHERE    TokenType = "WHERE"
	JOIN     TokenType = "JOIN"
	LIMIT    TokenType = "LIMIT"
	OFFSET   TokenType = "OFFSET"
	DISTINCT TokenType = "DISTINCT"
	GROUP    TokenType = "GROUP"
	ORDER    TokenType = "ORDER"
	BY       TokenType = "BY"

	// Data Maniplulation
	INSERT    TokenType = "INSERT"
	INTO      TokenType = "INTO"
	VALUES    TokenType = "VALUES"
	UPDATE    TokenType = "UPDATE"
	SET       TokenType = "SET"
	DELETE    TokenType = "DELETE"
	RETURNING TokenType = "RETURNING"

	// Data Definition
	CREATE   TokenType = "CREATE"
	TABLE    TokenType = "TABLE"
	TYPE     TokenType = "TYPE"
	PRIMARY  TokenType = "PRIMARY"
	KEY      TokenType = "KEY"
	DATABASE TokenType = "DATABASE"
	INDEX    TokenType = "INDEX"
	VIEW     TokenType = "VIEW"
	DROP     TokenType = "DROP"
	ALTER    TokenType = "ALTER"
	TRUNCATE TokenType = "TRUNCATE"
	ENUM     TokenType = "ENUM"
	UNIQUE   TokenType = "UNIQUE"

	// Condition & Logical
	AND    TokenType = "AND"
	OR     TokenType = "OR"
	NOT    TokenType = "NOT"
	NULL   TokenType = "NULL"
	ASC    TokenType = "ASC"
	DESC   TokenType = "DESC"
	HAVING TokenType = "HAVING"
	INNER  TokenType = "INNER"
	LEFT   TokenType = "LEFT"
	RIGHT  TokenType = "RIGHT"
	ON     TokenType = "ON"
	AS     TokenType = "AS"
	IN     TokenType = "IN"
	IS     TokenType = "IS"
	TRUE   TokenType = "TRUE"
	FALSE  TokenType = "FALSE"
	UNION  TokenType = "UNION"
	ALL    TokenType = "ALL"
	EXISTS TokenType = "EXISTS"
	CASE   TokenType = "CASE"
	WHEN   TokenType = "WHEN"
	THEN   TokenType = "THEN"
	ELSE   TokenType = "ELSE"
	END    TokenType = "END"
	ADD    TokenType = "ADD"
)

type Token struct {
	Type    TokenType
	Literal string
}

func CreateToken(token TokenType, char byte) Token {
	return Token{
		Type:    token,
		Literal: string(char),
	}
}

var keyWords = map[string]TokenType{
	"SELECT":    SELECT,
	"FROM":      FROM,
	"WHERE":     WHERE,
	"INSERT":    INSERT,
	"INTO":      INTO,
	"VALUES":    VALUES,
	"UPDATE":    UPDATE,
	"SET":       SET,
	"DELETE":    DELETE,
	"CREATE":    CREATE,
	"PRIMARY":   PRIMARY,
	"UNIQUE":    UNIQUE,
	"KEY":       KEY,
	"TABLE":     TABLE,
	"TYPE":      TYPE,
	"DROP":      DROP,
	"ALTER":     ALTER,
	"ADD":       ADD,
	"AND":       AND,
	"OR":        OR,
	"NOT":       NOT,
	"NULL":      NULL,
	"LIMIT":     LIMIT,
	"OFFSET":    OFFSET,
	"ORDER":     ORDER,
	"BY":        BY,
	"ASC":       ASC,
	"DESC":      DESC,
	"GROUP":     GROUP,
	"HAVING":    HAVING,
	"JOIN":      JOIN,
	"INNER":     INNER,
	"LEFT":      LEFT,
	"RIGHT":     RIGHT,
	"ON":        ON,
	"AS":        AS,
	"IN":        IN,
	"IS":        IS,
	"TRUE":      TRUE,
	"FALSE":     FALSE,
	"UNION":     UNION,
	"ALL":       ALL,
	"EXISTS":    EXISTS,
	"CASE":      CASE,
	"WHEN":      WHEN,
	"THEN":      THEN,
	"ELSE":      ELSE,
	"END":       END,
	"RETURNING": RETURNING,
	"DEFAULT":   DEFAULT,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keyWords[ident]; ok {
		return tok
	}

	return IDENT
}

func IsConditional(conditional string) bool {
	ops := map[string]struct{}{
		"=":  {},
		"!=": {},
		"<>": {},
		">":  {},
		">=": {},
		"<":  {},
		"<=": {},
	}

	strFormat := strings.TrimSpace(conditional)
	_, ok := ops[strFormat]
	return ok
}

func IsLogicalOperator(op string) bool {
	switch strings.ToUpper(strings.TrimSpace(op)) {
	case "AND", "OR", "NOT", "IN", "LIKE", "BETWEEN", "IS":
		return true

	default:
		return false
	}
}

var dbTypes = map[string]TokenType{
	"UUID":      UUID,
	"TEXT":      TEXT,
	"VARCHAR":   VARCHAR,
	"INT":       INT,
	"BIGINT":    BIGINT,
	"SMALLINT":  SMALLINT,
	"DECIMAL":   DECIMAL,
	"BOOLEAN":   BOOLEAN,
	"TIMESTAMP": TIMESTAMP,
	"DATE":      DATE,
}

func IsDatabaseType(t string) bool {
	if _, ok := dbTypes[t]; ok {
		return true
	}
	return false
}

func IsNowCompatible(tok Token) bool {
	switch tok.Literal {
	case "TIMESTAMP", "TIMESTAMPZ", "DATE", "TIME":
		return true
	default:
		return false
	}
}

func SqlNow(tok Token) string {
	switch tok.Type {
	case TIMESTAMP:
		return time.Now().Format("2006-01-02 15:04:05")
	case DATE:
		return time.Now().Format("2006-01-02")
	}

	return time.Now().String()
}

func SqlToGoType(tok Token) (string, error) {
	switch tok.Literal {
	case "TEXT", "VARCHAR", "UUID":
		return "string", nil
	case "INT", "BIGINT", "SMALLINT":
		return "int", nil
	case "DECIMAL":
		return "float64", nil
	case "BOOLEAN":
		return "bool", nil
	case "TIMESTAMP", "DATE":
		return "time.Time", nil
	}

	if tok.Type == ENUM {
		return tok.Literal, nil
	}

	return "", fmt.Errorf("[PARSER_TOKEN] failed generating go type current token: %s", tok.Literal)
}
