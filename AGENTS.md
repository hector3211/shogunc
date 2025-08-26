# Agent Guidelines for Shogunc

## Build, Lint, and Test Commands

### Build Commands

- **Build application**: `go build -o bin/app main.go`
- **Default build + test**: `task default` (or `go build -o bin/app main.go && go test ./...`)

### Test Commands

- **Run all tests**: `go test ./...`
- **Run tests with verbose output**: `go test -v ./...`
- **Run tests with race detection and coverage**: `go test ./... -v -race -cover`
- **Clean test cache**: `go clean -testcache`
- **Run specific test**: `go test -run TestName ./path/to/package`
- **Run tests in verbose mode**: `task testv`

### Task Commands (via Taskfile.yml)

- **Build**: `task build`
- **Test**: `task test`
- **Test (verbose)**: `task testv`
- **Clean test cache**: `task test-clean`
- **Default (build + test)**: `task default`

## Code Style Guidelines

### General Conventions

- Follow standard Go formatting (`go fmt`)
- Use `gofmt` or `goimports` for consistent formatting
- Maximum line length: ~100 characters
- Use meaningful variable and function names

### Naming Conventions

- **Exported functions/types**: PascalCase (e.g., `NewAst`, `ToPascalCase`, `GenerateTableType`)
- **Unexported functions/types**: camelCase (e.g., `generateSelectFunc`, `shoguncConditionalOp`)
- **Variables**: camelCase (e.g., `queryBlock`, `paramStruct`)
- **Constants**: PascalCase for exported, camelCase for unexported
- **Struct fields**: PascalCase for exported, camelCase for unexported
- **Database columns**: snake_case in SQL, converted to PascalCase in Go structs

### Error Handling

- Functions return `(result, error)` pattern consistently
- Use `fmt.Errorf` for error wrapping with context
- Handle errors immediately, don't ignore them
- Use `log.Fatalf` for fatal errors in main functions

### Imports

- Group imports: standard library, third-party, local packages
- Use aliases only when necessary to avoid naming conflicts

### Struct Tags

- Use `db:"column_name"` tags for database field mapping
- Use lowercase column names in database tags
- PascalCase field names in Go structs

### Database Operations

- Use `context.Context` for all database operations
- Implement proper connection handling
- Use prepared statements with bind parameters
- Return proper error types from database operations

### Testing

- Use table-driven tests for multiple test cases
- Use descriptive test names with `TestXxx` format
- Use `t.Run()` for subtests with descriptive names
- Test both success and error cases
- Use test helpers and setup functions when appropriate

### Code Generation

- Generate Go types from SQL schema definitions
- Use consistent type mapping from SQL to Go types
- Generate both selectable and insertable types
- Handle nullable fields with pointer types

### Utilities

- Use `strings.Builder` for efficient string concatenation
- Implement proper type conversion functions
- Use utility functions for common string operations (PascalCase, Capitalize, etc.)

## Project Structure

- `cmd/`: Executable commands
- `internal/`: Private application code
  - `codegen/`: Code generation logic
  - `parser/`: SQL parsing functionality
  - `types/`: Type definitions
- `utils/`: Shared utility functions
- `queries/`: SQL query files
- Root level: Main application files and configuration

## Dependencies

- Go 1.24.2 minimum
- External dependencies defined in `go.mod`
- Use `go mod tidy` to manage dependencies

