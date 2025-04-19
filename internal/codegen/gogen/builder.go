package gogen

import (
	"errors"
	"shogunc/internal/sqlparser"
)

type Builder struct {
	Instructions []byte
	Opcode       byte
}

func (b *Builder) Compile(node sqlparser.Node) error {
	switch node.(type) {
	case *sqlparser.SelectStatement:
	case *sqlparser.InsertStatement:
	default:
		return errors.New("[BUILDER] unknown statement")
	}

	return nil
}
