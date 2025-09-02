package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GenerateGoDb(outputPath string, driver string) (string, error) {
	switch strings.ToLower(driver) {
	case "postgres":
		return generateGoDbPostgres(outputPath)
	case "sqlite", "sqlite3":
		return generateGoDbSQLite(outputPath)
	default:
		return generateGoDbGeneric(outputPath)
	}
}

func generateGoDbGeneric(outputPath string) (string, error) {
	pkgName := "db"
	var dbFilePath string
	if outputPath != "" {
		dbFilePath = outputPath
	} else {
		dbFilePath = "/tmp/internal/generated/db.go"
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
		err := os.MkdirAll(filepath.Dir(dbFilePath), 0755)
		if err != nil {
			return "", err
		}
		err = os.WriteFile(dbFilePath, []byte(generatedCode), 0644)
		if err != nil {
			return "", err
		}
	}
	return generatedCode, nil
}

func generateGoDbSQLite(outputPath string) (string, error) {
	pkgName := "db"
	var dbFilePath string
	if outputPath != "" {
		dbFilePath = outputPath
	} else {
		pkgName = "db"
		dbFilePath = "/tmp/internal/generated/db.go"
	}
	var buffer strings.Builder
	buffer.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	buffer.WriteString(`import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
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
		err := os.MkdirAll(filepath.Dir(dbFilePath), 0755)
		if err != nil {
			return "", err
		}
		err = os.WriteFile(dbFilePath, []byte(generatedCode), 0644)
		if err != nil {
			return "", err
		}
	}
	return generatedCode, nil
}

func generateGoDbPostgres(outputPath string) (string, error) {
	pkgName := "db"
	var dbFilePath string
	if outputPath != "" {
		dbFilePath = outputPath
	} else {
		dbFilePath = "/tmp/internal/generated/db.go"
	}
	var buffer strings.Builder
	buffer.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	buffer.WriteString(`import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5"
)
`)
	buffer.WriteString(`type DBX interface {
	Exec(context.Context, string, ...any) error
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
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
	// NewFromPool creates a new Queries instance from a pgxpool.Pool
	buffer.WriteString(`func NewFromPool(pool *pgxpool.Pool) *Queries {
	return &Queries{db: pool}
}
`)
	generatedCode := buffer.String()
	if dbFilePath != "" {
		err := os.MkdirAll(filepath.Dir(dbFilePath), 0755)
		if err != nil {
			return "", err
		}
		err = os.WriteFile(dbFilePath, []byte(generatedCode), 0644)
		if err != nil {
			return "", err
		}
	}
	return generatedCode, nil
}
