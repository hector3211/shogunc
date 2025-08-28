# Shogunc Project Todo List

## Project Status Summary

**Current Phase**: INSERT Operations Fully Implemented & Production Ready

- ✅ Core code generation working with proper AST-based approach
- ✅ All tests passing, comprehensive coverage achieved (41/41 tests)
- ✅ Generated code has correct syntax and follows Go conventions
- ✅ Parameter binding improved to use named parameters (params.FieldName)
- ✅ Database integration implemented with query execution, error handling, and context support
- ✅ Fixed all SQL query generation issues (quoting, variable declarations, parameter conversion)
- ✅ **INSERT operations fully implemented with proper SQL bind parameters**
- ✅ **CleanBindParam utility function created for SQL parameter cleaning**
- ✅ Removed unnecessary dependencies from main project (sqlite driver moved to generated code)
- ✅ Added build constraints to prevent generated files from interfering with main project builds
- ✅ Generated files properly isolated in /tmp/db/generated/ directory
- ✅ Ready for expanded query support (UPDATE, DELETE, EXEC, etc.)

**Next Phase**: Extended Query Support & Advanced Features

## High Priority Issues

- **Fix Schema Parsing Issues**
  - [x] Add missing enum types referenced in schema.sql (Role, Account_Status)
  - [x] Fix enum type generation to handle all defined enums properly
  - [x] Validate schema parsing handles all SQL data types correctly

- **Fix Code Generation Issues**
  - [x] Fix missing imports in generated code (time package, etc.)
  - [x] Fix SQL query generation (column name casing, literal value handling)
  - [x] Fix function body generation (proper return statements, error handling)
  - [x] Add package declaration to generated code
  - [x] Standardize on AST-based generation (eliminated string-based generation)
  - [x] Fix return type syntax (changed from ((User, error), error) to (User, error))
  - [x] Fix SQL query quoting (add proper quotes around SQL strings)
  - [x] Fix variable declarations (proper := vs = usage in AST generation)
  - [x] Fix parameter name conversion (snake_case to PascalCase)
  - [x] Fix QueryRow/Scan pattern implementation with proper error handling

- **Fix Parameter Struct Generation**
  - [x] Match data types correctly between SQL schema and Go types
  - [x] Extract column names and SQL types accurately
  - [x] Generate proper Go parameter structs with correct field types
  - [x] Handle nullable fields correctly in parameter structs
  - [x] Use PascalCase field names consistently in all generated structs

## Medium Priority Features

- **Implement Database Integration**
   - [x] Implement actual database query execution (replace TODO comments)
   - [x] Add database connection handling
   - [x] Implement proper error handling for database operations
   - [x] Add context support for database queries

- **Add Support for INSERT Operations**
  - [x] Implement INSERT statement parsing
  - [x] Generate INSERT functions with proper parameter handling
  - [x] Handle RETURNING clauses
  - [x] Support INSERT with bind parameters
  - [x] Fix SQL bind parameter quoting issues
  - [x] Create CleanBindParam utility function
  - [x] Support up to 15 bind parameters ($1-$15)

- **Add Support for EXEC Operations**
  - Implement EXEC statement parsing
  - Generate EXEC functions for non-SELECT queries
  - Handle UPDATE, DELETE, and other DDL operations

- **Improve Parameter Binding**
   - [x] Change from positional parameters ($1) to named parameters (params.FieldName)
   - [x] Update shoguncConditionalOp function to generate params.FieldName references
   - [x] Update tests to expect named parameter syntax
   - [x] Improve type safety and readability of generated SQL conditions

## Low Priority Improvements

- **Enhance Error Handling**
  - Add better error messages with context
  - Implement proper error wrapping
  - Add validation for generated code

- **Add Testing**
  - Add integration tests for end-to-end code generation
  - Add tests for database operations
  - Add performance tests

- **Documentation and Examples**
  - Add comprehensive README with usage examples
  - Add documentation for configuration options
  - Create example projects showing usage

- **CLI Improvements**
  - Add command-line flags for configuration
  - Add verbose output options
  - Add validation commands

## Completed Tasks

- [x] Basic project structure and parser implementation
- [x] Schema parsing for tables and enums
- [x] Basic SELECT query generation
- [x] Parameter struct generation framework
- [x] AGENTS.md file creation with build/lint/test commands
- [x] Refactored code generation to use AST nodes instead of strings
- [x] Fixed return type syntax issues in generated functions
- [x] Standardized PascalCase field naming across all generated structs
- [x] Improved type safety and consistency in code generation
- [x] All tests passing with comprehensive coverage (41/41 tests)
- [x] End-to-end code generation working correctly
- [x] Fixed SQL query quoting issues (proper quotes around SQL strings)
- [x] Fixed variable declaration issues (proper := vs = usage)
- [x] Fixed parameter name conversion (snake_case to PascalCase)
- [x] Fixed QueryRow/Scan pattern with proper error handling
- [x] **INSERT operations fully implemented with proper SQL bind parameters**
- [x] **SELECT operations enhanced with CleanBindParam utility**
- [x] **Created CleanBindParam utility function for SQL parameter cleaning**
- [x] **Extended bind parameter support up to $15**
- [x] **Fixed critical SQL syntax bug (quoted vs unquoted bind parameters)**
- [x] Removed unnecessary dependencies from main project (sqlite driver)
- [x] Added build constraints to prevent generated files from interfering with main project
- [x] Isolated generated files in /tmp/db/generated/ directory
