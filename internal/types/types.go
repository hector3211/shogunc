package types

type LogicalOp string

const (
	And     LogicalOp = "And"
	Or      LogicalOp = "Or"
	Illegal LogicalOp = ""
)

func ToLogicOp(op string) LogicalOp {
	switch op {
	case "AND":
		return And
	case "OR":
		return Or
	default:
		return Illegal
	}
}

type ConditionOp string

const (
	EQUAL       ConditionOp = "="
	NOTEQUAL    ConditionOp = "!="
	LESSTHAN    ConditionOp = "<"
	GREATERTHAN ConditionOp = ">"
	BETWEEN     ConditionOp = "BETWEEN"
	ISNULL      ConditionOp = "IS NULL"
	NOTNULL     ConditionOp = "IS NOT NULL"
)

type Bind struct {
	Column   string  // Column
	Position int     // $1 $2
	Value    *string // true | false
}

type Condition struct {
	Column   string      // Column
	Operator ConditionOp // = | != | >
	Value    Bind
	ChainOp  LogicalOp // AND | OR | NOT
}

type SelectStatement struct {
	Columns    []string
	Conditions []Condition
	TableName  string
	Distinct   bool
	Limit      int
	Offset     int
}

type InsertStatement struct {
	TableName       string
	Columns         []string
	Values          []Bind
	ReturningFields []string
	InsertMode      []byte
}

type Type string // exec | one | many

const (
	EXEC Type = "exec"
	ONE  Type = "one"
	MANY Type = "many"
)

type TagType struct {
	Name []byte
	Type Type
}

type QueryBlock struct {
	Name     string // -name: GetUser
	Type     Type
	SQL      string // sql query statement
	Filename string // for debug or error reporting
	DataType any    // Infered type
}
