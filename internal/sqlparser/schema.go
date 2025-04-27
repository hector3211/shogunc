package sqlparser

type Field struct {
	Name      string  // "description"
	DataType  string  // "TEXT"
	NotNull   bool    // true if NOT NULL, false if nullable
	Default   *string // optional default value
	IsPrimary bool    // true if PRIMARY KEY
	IsUnique  bool    // true if UNIQUE
	Comment   *string // optional comment
}

type TableType struct {
	Name  []byte
	Field []Field
}

type EnumType struct {
	name   []byte
	values []string
}

func NewTableType(name []byte, fields []Field) *TableType {
	return &TableType{
		Name:  name,
		Field: fields,
	}
}

func NewEnumType(name []byte, values []string) *EnumType {
	return &EnumType{
		name:   name,
		values: values,
	}
}
