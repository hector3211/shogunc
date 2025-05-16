package gogen

import (
	"fmt"
	"strings"
)

func GenerateDB(packageName string) string {
	var strBuilder strings.Builder

	strBuilder.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	strBuilder.WriteString(`import (
	"context"
	"database/sql"
)
`)
	strBuilder.WriteString(`type DBX interface {
		Exec(context.Context, string, ...any) error
		Query(context.Context, string, ...any) (*sql.Rows, error)
		QueryRow(context.Context, string, ...any) (*sql.Row, error)
}

`)

	strBuilder.WriteString(`type Queries struct {
		db DBX
}

`)

	strBuilder.WriteString(`func New(db DBX) *Queries {
		return &Queries{db: db}
}

`)
	return strBuilder.String()
}
