# shogunc

**shogunc** is a SQL code generation tool inspired by [sqlc](https://github.com/sqlc-dev/sqlc), designed for Go developers who want to generate type-safe, idiomatic Go code directly from their SQL schema and query files. Unlike other tools, shogunc features a hand-rolled interpreter for parsing both schema and query files, giving you full control and transparency over the codegen process.

## Features

- **Type-safe code generation**: Generate Go structs and functions from SQL schemas and queries
- **Custom parser**: Hand-rolled interpreter for complete control over parsing logic
- **SQLite3 support**: Built-in support for SQLite3 databases
- **YAML configuration**: Simple configuration through `shogunc.yml`
- **Multiple query types**: Support for `:one`, `:many`, and other query result types
- **Enum support**: Automatic generation of Go constants from SQL ENUM types

## Current Status

### ✅ Implemented

- **Core Architecture**: Basic project structure with modular design
- **SQL Parser**: Hand-rolled lexer and parser for SQL schema and query files
- **Code Generation**: Basic Go code generation from parsed SQL structures
- **Configuration System**: YAML-based configuration support (`shogunc.yml`)
- **Development Mode**: Automatic generation of test files when `DEVELOPMENT=true`
- **CLI Interface**: Command-line interface with input handling
- **Type System**: Basic type mapping from SQL to Go types
- **Utility Functions**: String manipulation and type conversion utilities

### 📋 Planned

- **Multiple Database Support**: PostgreSQL and other database drivers
- **Advanced Query Types**: Support for `:exec`
- **Documentation**: Comprehensive API documentation and usage examples

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
├── utils/                 # Shared utility functions
├── main.go                # Application entry point
└── README.md
```

## Development

### Development Mode

For testing purposes, you can enable development mode to automatically generate test files:

```bash
DEVELOPMENT=true go run main.go
```

This will automatically create:

- `schema.sql` database schema file
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
