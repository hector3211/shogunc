package gogen

import (
	"fmt"
	"shogunc/internal/sqlparser"
	"shogunc/utils"
	"strings"
)

func GenerateEnumType(enumType sqlparser.EnumType) (string, error) {
	var strBuilder strings.Builder

	strBuilder.WriteString(fmt.Sprintf("type %s string\n\n", enumType.Name))
	strBuilder.WriteString("const (\n")

	for _, v := range enumType.Values {
		value := utils.ToPascalCase(v)
		strBuilder.WriteString(fmt.Sprintf("\t%s %s = \"%s\"\n", value, enumType.Name, v))
	}

	strBuilder.WriteString(")\n")
	return strBuilder.String(), nil
}

func GenerateTableType(tableType sqlparser.TableType) (string, error) {
	var strBuilder strings.Builder

	strBuilder.WriteString(fmt.Sprintf("type %s struct {\n", tableType.Name))
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
		jsonTag := fmt.Sprintf("`db:\"%s\"`", f.Name)

		strBuilder.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag))
	}

	strBuilder.WriteString("}\n")
	return strBuilder.String(), nil
}

func GenreateInsertTableType(tableType sqlparser.TableType) (string, error) {
	var strBuilder strings.Builder
	strBuilder.WriteString(fmt.Sprintf("type New%s struct {\n", tableType.Name))
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
		jsonTag := fmt.Sprintf("`db:\"%s\"`", f.Name)

		strBuilder.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag))
	}

	strBuilder.WriteString("}\n")

	return strBuilder.String(), nil
}
