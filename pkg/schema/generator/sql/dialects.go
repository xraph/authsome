package sql

import (
	"fmt"
	"strings"

	"github.com/xraph/authsome/pkg/schema/definition"
)

// PostgreSQLDialect implements PostgreSQL dialect.
type PostgreSQLDialect struct{}

func (d *PostgreSQLDialect) Name() string {
	return "PostgreSQL"
}

func (d *PostgreSQLDialect) QuoteIdentifier(name string) string {
	return `"` + name + `"`
}

func (d *PostgreSQLDialect) AutoIncrement() string {
	return "SERIAL"
}

func (d *PostgreSQLDialect) BooleanType() string {
	return "BOOLEAN"
}

func (d *PostgreSQLDialect) MapType(fieldType definition.FieldType, length int, precision int, scale int) string {
	switch fieldType {
	case definition.FieldTypeString:
		if length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", length)
		}

		return "VARCHAR(255)"
	case definition.FieldTypeText:
		return "TEXT"
	case definition.FieldTypeInteger:
		return "INTEGER"
	case definition.FieldTypeBigInt:
		return "BIGINT"
	case definition.FieldTypeFloat:
		return "DOUBLE PRECISION"
	case definition.FieldTypeDecimal:
		if precision > 0 && scale > 0 {
			return fmt.Sprintf("DECIMAL(%d,%d)", precision, scale)
		}

		return "DECIMAL"
	case definition.FieldTypeBoolean:
		return "BOOLEAN"
	case definition.FieldTypeTimestamp:
		return "TIMESTAMP"
	case definition.FieldTypeDate:
		return "DATE"
	case definition.FieldTypeTime:
		return "TIME"
	case definition.FieldTypeUUID:
		return "UUID"
	case definition.FieldTypeJSON:
		return "JSON"
	case definition.FieldTypeJSONB:
		return "JSONB"
	case definition.FieldTypeBinary:
		return "BYTEA"
	default:
		return "TEXT"
	}
}

func (d *PostgreSQLDialect) DefaultValue(value any, fieldType definition.FieldType) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		// Special PostgreSQL functions
		if v == "current_timestamp" || v == "now()" {
			return "CURRENT_TIMESTAMP"
		}
		// Boolean values
		if fieldType == definition.FieldTypeBoolean {
			if v == "true" || v == "1" {
				return "true"
			}

			if v == "false" || v == "0" {
				return "false"
			}
		}

		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "true"
		}

		return "false"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// MySQLDialect implements MySQL dialect.
type MySQLDialect struct{}

func (d *MySQLDialect) Name() string {
	return "MySQL"
}

func (d *MySQLDialect) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}

func (d *MySQLDialect) AutoIncrement() string {
	return "AUTO_INCREMENT"
}

func (d *MySQLDialect) BooleanType() string {
	return "TINYINT(1)"
}

func (d *MySQLDialect) MapType(fieldType definition.FieldType, length int, precision int, scale int) string {
	switch fieldType {
	case definition.FieldTypeString:
		if length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", length)
		}

		return "VARCHAR(255)"
	case definition.FieldTypeText:
		return "TEXT"
	case definition.FieldTypeInteger:
		return "INT"
	case definition.FieldTypeBigInt:
		return "BIGINT"
	case definition.FieldTypeFloat:
		return "DOUBLE"
	case definition.FieldTypeDecimal:
		if precision > 0 && scale > 0 {
			return fmt.Sprintf("DECIMAL(%d,%d)", precision, scale)
		}

		return "DECIMAL(10,2)"
	case definition.FieldTypeBoolean:
		return "TINYINT(1)"
	case definition.FieldTypeTimestamp:
		return "DATETIME"
	case definition.FieldTypeDate:
		return "DATE"
	case definition.FieldTypeTime:
		return "TIME"
	case definition.FieldTypeUUID:
		return "CHAR(36)"
	case definition.FieldTypeJSON, definition.FieldTypeJSONB:
		return "JSON"
	case definition.FieldTypeBinary:
		return "BLOB"
	default:
		return "TEXT"
	}
}

func (d *MySQLDialect) DefaultValue(value any, fieldType definition.FieldType) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		// Special MySQL functions
		if v == "current_timestamp" || v == "now()" {
			return "CURRENT_TIMESTAMP"
		}
		// Boolean values
		if fieldType == definition.FieldTypeBoolean {
			if v == "true" || v == "1" {
				return "1"
			}

			if v == "false" || v == "0" {
				return "0"
			}
		}

		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}

		return "0"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// SQLiteDialect implements SQLite dialect.
type SQLiteDialect struct{}

func (d *SQLiteDialect) Name() string {
	return "SQLite"
}

func (d *SQLiteDialect) QuoteIdentifier(name string) string {
	return `"` + name + `"`
}

func (d *SQLiteDialect) AutoIncrement() string {
	return "AUTOINCREMENT"
}

func (d *SQLiteDialect) BooleanType() string {
	return "INTEGER"
}

func (d *SQLiteDialect) MapType(fieldType definition.FieldType, length int, precision int, scale int) string {
	switch fieldType {
	case definition.FieldTypeString, definition.FieldTypeText, definition.FieldTypeUUID:
		return "TEXT"
	case definition.FieldTypeInteger, definition.FieldTypeBigInt:
		return "INTEGER"
	case definition.FieldTypeFloat, definition.FieldTypeDecimal:
		return "REAL"
	case definition.FieldTypeBoolean:
		return "INTEGER"
	case definition.FieldTypeTimestamp, definition.FieldTypeDate, definition.FieldTypeTime:
		return "DATETIME"
	case definition.FieldTypeJSON, definition.FieldTypeJSONB:
		return "TEXT"
	case definition.FieldTypeBinary:
		return "BLOB"
	default:
		return "TEXT"
	}
}

func (d *SQLiteDialect) DefaultValue(value any, fieldType definition.FieldType) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		// Special SQLite functions
		if v == "current_timestamp" || v == "now()" {
			return "CURRENT_TIMESTAMP"
		}
		// Boolean values
		if fieldType == definition.FieldTypeBoolean {
			if v == "true" || v == "1" {
				return "1"
			}

			if v == "false" || v == "0" {
				return "0"
			}
		}

		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}

		return "0"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}
