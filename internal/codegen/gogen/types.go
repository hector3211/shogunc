package gogen

import (
	"bytes"
	"fmt"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
	"strings"
)

func GenerateEnumType(enumType *sqlparser.EnumType) (string, error) {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("type %s string\n\n", enumType.Name))
	buffer.WriteString("const (\n")

	for _, v := range enumType.Values {
		value := utils.ToPascalCase(v)
		buffer.WriteString(fmt.Sprintf("\t%s %s = %q\n", value, enumType.Name, v))
	}

	buffer.WriteString(")\n")
	return buffer.String(), nil
}

func GenerateTableType(tableType *sqlparser.TableType) (string, error) {
	var buffer bytes.Buffer

	selectType, err := generateSelectableTableType(tableType)
	if err != nil {
		return "", err
	}
	buffer.WriteString(selectType)

	insertType, err := genreateInsertTableType(tableType)
	if err != nil {
		return "", err
	}
	buffer.WriteString(insertType)

	return buffer.String(), nil
}

func generateSelectableTableType(tableType *sqlparser.TableType) (string, error) {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("type %s struct {\n", strings.ToUpper(tableType.Name[:1])+tableType.Name[1:]))
	for _, f := range tableType.Fields {
		goDataType := sqlparser.SqlToGoType(f.DataType)
		if goDataType == "" {
			return "", fmt.Errorf("[BUILDER] failed parsing %s %v to GO type", f.DataType.Literal, f.DataType.Type)
		}

		fieldType := goDataType
		// if !f.NotNull {
		// 	fieldType = "*" + goDataType
		// }

		fieldName := utils.ToPascalCase(f.Name)
		jsonTag := fmt.Sprintf("`db:%q`", f.Name)

		buffer.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag))
	}

	buffer.WriteString("}\n")
	return buffer.String(), nil
}

func genreateInsertTableType(tableType *sqlparser.TableType) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("type New%s struct {\n", tableType.Name[:len(tableType.Name)-1]))
	for _, f := range tableType.Fields {
		goDataType := sqlparser.SqlToGoType(f.DataType)
		if goDataType == "" {
			return "", fmt.Errorf("[BUILDER] failed parsing %s to GO type", f.DataType.Literal)
		}

		fieldType := goDataType
		if !f.NotNull {
			fieldType = "*" + goDataType
		}

		fieldName := utils.ToPascalCase(f.Name)
		jsonTag := fmt.Sprintf("`db:%q`", f.Name)

		buffer.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag))
	}

	buffer.WriteString("}\n")

	return buffer.String(), nil
}
