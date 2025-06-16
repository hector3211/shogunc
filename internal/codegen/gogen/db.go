package gogen

import (
	"bytes"
	"fmt"
)

func GenerateDB(packageName string) string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	buffer.WriteString(`import (
	"context"
	"database/sql"
)
`)
	buffer.WriteString(`type DBX interface {
		Exec(context.Context, string, ...any) error
		Query(context.Context, string, ...any) (*sql.Rows, error)
		QueryRow(context.Context, string, ...any) (*sql.Row, error)
}

`)

	buffer.WriteString(`type Queries struct {
		db DBX
}

`)

	buffer.WriteString(`func New(db DBX) *Queries {
		return &Queries{db: db}
}

`)
	return buffer.String()
}
