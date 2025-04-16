package sqlparser

import "strings"

type TokenType string

const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	STRING TokenType = "STRING"

	// Identifiers + Literals
	IDENT TokenType = "IDENT" // foobar
	INT   TokenType = "INT"   // 12345

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
	DATABASE TokenType = "DATABASE"
	INDEX    TokenType = "INDEX"
	VIEW     TokenType = "VIEW"
	DROP     TokenType = "DROP"
	ALTER    TokenType = "ALTER"
	TRUNCATE TokenType = "TRUNCATE"

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
	"TABLE":     TABLE,
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
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keyWords[ident]; ok {
		return tok
	}
	return IDENT
}

func IsConditional(conditional string) bool {
	ops := map[string]struct{}{
		"=":       {},
		"!=":      {},
		"<>":      {},
		">":       {},
		">=":      {},
		"<":       {},
		"<=":      {},
		"AND":     {},
		"OR":      {},
		"NOT":     {},
		"IN":      {},
		"LIKE":    {},
		"BETWEEN": {},
		"IS":      {},
	}

	strFormat := strings.ToUpper(strings.TrimSpace(conditional))
	_, ok := ops[strFormat]
	return ok
}
