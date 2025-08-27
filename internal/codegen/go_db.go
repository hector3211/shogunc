package codegen

import (
	"fmt"
	"os"
	"strings"
)

func GenerateGoDb(packageName *string) (string, error) {
	var pkgName string
	var dbFilePath string
	if packageName != nil {
		pkgName = *packageName
	} else {
		pkgName = "db"
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}

		err = os.MkdirAll(fmt.Sprintf("%s/internal/db", cwd), 0755)
		if err != nil {
			return "", err
		}
		dbFilePath = fmt.Sprintf("%s/internal/db/db.go", cwd)
	}
	var buffer strings.Builder
	buffer.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

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
	generatedCode := buffer.String()
	if dbFilePath != "" {
		err := os.WriteFile(dbFilePath, []byte(generatedCode), 0644)
		if err != nil {
			return "", err
		}
	}
	return generatedCode, nil
}
