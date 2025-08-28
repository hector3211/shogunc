# shogunc

**shogunc** is a SQL code generation tool inspired by [sqlc](https://github.com/sqlc-dev/sqlc), designed for Go developers who want to generate type-safe, idiomatic Go code directly from their SQL schema and query files. Unlike other tools, shogunc features a hand-rolled interpreter for parsing both schema and query files, giving you full control and transparency over the codegen process.

## Features

- **Type-safe code generation**: Generate Go structs and functions from SQL schemas and queries
- **Custom parser**: Hand-rolled interpreter for complete control over parsing logic
- **SQLite3 support**: Built-in support for SQLite3 databases
- **YAML configuration**: Simple configuration through `shogunc.yml`
- **Multiple query types**: Support for `:one`, `:many`, and other query result types
- **Enum support**: Automatic generation of Go constants from SQL ENUM types

## Installation

### Prerequisites

- Go 1.25.0 or later

### Build from source

```bash
git clone https://github.com/hector3211/shogunc.git
cd shogunc
go build -o bin/shogunc main.go
```

## Usage

### Configuration

Create a `shogunc.yml` configuration file in your project root:

```yaml
sql:
  schema: schema.sql
  queries: queries
  driver: sqlite3
  output: /internal/db/generated
```

### Schema File

Define your database schema in `schema.sql`:

```sql
CREATE TYPE "Role" AS ENUM (
    'tenant',
    'landlord',
    'admin',
    'staff'
);

CREATE TABLE IF NOT EXISTS "users" (
    "id"          UUID PRIMARY KEY,
    "clerk_id"    TEXT UNIQUE NOT NULL,
    "first_name"  VARCHAR NOT NULL,
    "last_name"   VARCHAR NOT NULL,
    "email"       VARCHAR NOT NULL,
    "role"        "Role" NOT NULL DEFAULT 'tenant',
    "created_at"  TIMESTAMP DEFAULT now()
);
```

### Query Files

Create query files in the `queries` directory:

```sql
-- name: GetUser :one
SELECT id, first_name, last_name, email, role, created_at
FROM users
WHERE id = $1;

-- name: GetAllUsers :many
SELECT id, first_name, last_name, email, role
FROM users
WHERE status = $1;
```

### Generate Code

```bash
echo "generate" | go run main.go
```

Or using the task runner:

```bash
task build
```

## Project Structure

```
shogunc/
├── cmd/
│   └── generate/          # Code generation commands
├── internal/
│   ├── codegen/           # Code generation logic
│   ├── parser/            # SQL parsing functionality
│   └── types/             # Type definitions
├── queries/               # SQL query files
├── utils/                 # Shared utility functions
├── schema.sql             # Database schema
├── shogunc.yml            # Configuration file
├── main.go                # Application entry point
└── README.md
```

## Development

### Development Mode

For testing purposes, you can enable development mode to automatically generate test query files and configuration:

```bash
DEVELOPMENT=true go run main.go
```

This will automatically create:

- `queries/` directory with sample SQL query files
- `shogunc.yml` configuration file

These files are automatically added to `.gitignore` and are intended only for development and testing.

### Available Commands

- `task build` - Build the application
- `task test` - Run tests
- `task testv` - Run tests with verbose output
- `task test-clean` - Clean test cache
- `task default` - Build and test

### Testing

```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
