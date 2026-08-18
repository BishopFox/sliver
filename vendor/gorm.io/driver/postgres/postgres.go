package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Dialector struct {
	*Config
}

type Config struct {
	DriverName           string
	DSN                  string
	WithoutQuotingCheck  bool
	PreferSimpleProtocol bool
	WithoutReturning     bool
	Conn                 gorm.ConnPool
	OptionOpenDB         []stdlib.OptionOpenDB
}

var (
	timeZoneMatcher         = regexp.MustCompile("(time_zone|TimeZone|timezone)=(.*?)($|&| )")
	defaultIdentifierLength = 63 //maximum identifier length for postgres
)

func Open(dsn string) gorm.Dialector {
	return &Dialector{&Config{DSN: dsn}}
}

func New(config Config) gorm.Dialector {
	return &Dialector{Config: &config}
}

func (dialector Dialector) Name() string {
	return "postgres"
}

func (dialector Dialector) Apply(config *gorm.Config) error {
	if config.NamingStrategy == nil {
		config.NamingStrategy = schema.NamingStrategy{
			IdentifierMaxLength: defaultIdentifierLength,
		}
		return nil
	}

	switch v := config.NamingStrategy.(type) {
	case *schema.NamingStrategy:
		if v.IdentifierMaxLength <= 0 {
			v.IdentifierMaxLength = defaultIdentifierLength
		}
	case schema.NamingStrategy:
		if v.IdentifierMaxLength <= 0 {
			v.IdentifierMaxLength = defaultIdentifierLength
			config.NamingStrategy = v
		}
	}

	return nil
}

func (dialector Dialector) Initialize(db *gorm.DB) (err error) {
	callbackConfig := &callbacks.Config{
		CreateClauses: []string{"INSERT", "VALUES", "ON CONFLICT"},
		UpdateClauses: []string{"UPDATE", "SET", "FROM", "WHERE"},
		DeleteClauses: []string{"DELETE", "FROM", "WHERE"},
	}
	// register callbacks
	if !dialector.WithoutReturning {
		callbackConfig.CreateClauses = append(callbackConfig.CreateClauses, "RETURNING")
		callbackConfig.UpdateClauses = append(callbackConfig.UpdateClauses, "RETURNING")
		callbackConfig.DeleteClauses = append(callbackConfig.DeleteClauses, "RETURNING")
	}
	callbacks.RegisterDefaultCallbacks(db, callbackConfig)

	if dialector.Conn != nil {
		db.ConnPool = dialector.Conn
	} else if dialector.DriverName != "" {
		db.ConnPool, err = sql.Open(dialector.DriverName, dialector.Config.DSN)
	} else {
		var config *pgx.ConnConfig

		config, err = pgx.ParseConfig(dialector.Config.DSN)
		if err != nil {
			return
		}
		if dialector.Config.PreferSimpleProtocol {
			config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
		}
		result := timeZoneMatcher.FindStringSubmatch(dialector.Config.DSN)
		if len(result) > 2 {
			config.RuntimeParams["timezone"] = result[2]
			dialector.OptionOpenDB = append(dialector.OptionOpenDB, stdlib.OptionAfterConnect(func(ctx context.Context, conn *pgx.Conn) error {
				loc, tzErr := time.LoadLocation(result[2])
				if tzErr != nil {
					return tzErr
				}
				conn.TypeMap().RegisterType(&pgtype.Type{
					Name:  "timestamp",
					OID:   pgtype.TimestampOID,
					Codec: &pgtype.TimestampCodec{ScanLocation: loc},
				})
				return nil
			}))
		}
		db.ConnPool = stdlib.OpenDB(*config, dialector.OptionOpenDB...)
	}
	return
}

func (dialector Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{migrator.Migrator{Config: migrator.Config{
		DB:                          db,
		Dialector:                   dialector,
		CreateIndexAfterCreateTable: true,
	}}}
}

func (dialector Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (dialector Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('$')
	index := 0
	varLen := len(stmt.Vars)
	if varLen > 0 {
		switch stmt.Vars[0].(type) {
		case pgx.QueryExecMode:
			index++
		}
	}
	writer.WriteString(strconv.Itoa(varLen - index))
}

func (dialector Dialector) QuoteTo(writer clause.Writer, str string) {
	if dialector.WithoutQuotingCheck {
		writer.WriteString(str)
		return
	}

	var (
		underQuoted, selfQuoted bool
		continuousBacktick      int8
		shiftDelimiter          int8
	)

	for _, v := range []byte(str) {
		switch v {
		case '"':
			continuousBacktick++
			if continuousBacktick == 2 {
				writer.WriteString(`""`)
				continuousBacktick = 0
			}
		case '.':
			if continuousBacktick > 0 || !selfQuoted {
				shiftDelimiter = 0
				underQuoted = false
				continuousBacktick = 0
				writer.WriteByte('"')
			}
			writer.WriteByte(v)
			continue
		default:
			if shiftDelimiter-continuousBacktick <= 0 && !underQuoted {
				writer.WriteByte('"')
				underQuoted = true
				if selfQuoted = continuousBacktick > 0; selfQuoted {
					continuousBacktick -= 1
				}
			}

			for ; continuousBacktick > 0; continuousBacktick -= 1 {
				writer.WriteString(`""`)
			}

			writer.WriteByte(v)
		}
		shiftDelimiter++
	}

	if continuousBacktick > 0 && !selfQuoted {
		writer.WriteString(`""`)
	}
	writer.WriteByte('"')
}

var numericPlaceholder = regexp.MustCompile(`\$(\d+)`)

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, numericPlaceholder, `'`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	// PostgreSQL 10+ generated columns. The value-generation strategy is carried
	// by the `generated` tag, intentionally kept separate from the column `type`:
	//
	//	`gorm:"generated:identity"`          -> <int>  GENERATED BY DEFAULT AS IDENTITY
	//	`gorm:"generated:identity always"`   -> <int>  GENERATED ALWAYS AS IDENTITY
	//	`gorm:"generated:price * quantity"`  -> <type> GENERATED ALWAYS AS (price * quantity) STORED
	//
	// https://github.com/go-gorm/gorm/issues/7191
	if gen, ok := generatedColumnOf(field); ok {
		if gen.identity {
			return dialector.getSchemaIntType(field) + " GENERATED " + gen.mode + " AS IDENTITY"
		}
		return dialector.getSchemaBaseType(field) + " GENERATED ALWAYS AS (" + gen.expr + ") STORED"
	}

	return dialector.getSchemaBaseType(field)
}

func (dialector Dialector) getSchemaBaseType(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		intType := dialector.getSchemaIntType(field)
		if field.AutoIncrement {
			switch intType {
			case "smallint":
				return "smallserial"
			case "integer":
				return "serial"
			default:
				return "bigserial"
			}
		}
		return intType
	case schema.Float:
		if field.Precision > 0 {
			if field.Scale > 0 {
				return fmt.Sprintf("numeric(%d, %d)", field.Precision, field.Scale)
			}
			return fmt.Sprintf("numeric(%d)", field.Precision)
		}
		return "decimal"
	case schema.String:
		if field.Size > 0 && field.Size <= 10485760 {
			return fmt.Sprintf("varchar(%d)", field.Size)
		}
		return "text"
	case schema.Time:
		if field.Precision > 0 {
			return fmt.Sprintf("timestamptz(%d)", field.Precision)
		}
		return "timestamptz"
	case schema.Bytes:
		return "bytea"
	default:
		return dialector.getSchemaCustomType(field)
	}
}

func (dialector Dialector) getSchemaIntType(field *schema.Field) string {
	size := field.Size
	if field.DataType == schema.Uint {
		size++
	}

	switch {
	case size <= 16:
		return "smallint"
	case size <= 32:
		return "integer"
	default:
		return "bigint"
	}
}

func (dialector Dialector) getSchemaCustomType(field *schema.Field) string {
	sqlType := string(field.DataType)

	if field.AutoIncrement && !strings.Contains(strings.ToLower(sqlType), "serial") {
		size := field.Size
		if field.GORMDataType == schema.Uint {
			size++
		}
		switch {
		case size <= 16:
			sqlType = "smallserial"
		case size <= 32:
			sqlType = "serial"
		default:
			sqlType = "bigserial"
		}
	}

	return sqlType
}

func (dialector Dialector) SavePoint(tx *gorm.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return nil
}

func (dialector Dialector) RollbackTo(tx *gorm.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return nil
}

func getSerialDatabaseType(s string) (dbType string, ok bool) {
	switch s {
	case "smallserial":
		return "smallint", true
	case "serial":
		return "integer", true
	case "bigserial":
		return "bigint", true
	default:
		return "", false
	}
}

// generatedColumn describes a PostgreSQL generated column parsed from a
// `generated` tag: either an identity column or a STORED computed column.
type generatedColumn struct {
	identity bool   // identity column: GENERATED { mode } AS IDENTITY
	mode     string // identity generation mode: "BY DEFAULT" or "ALWAYS"
	expr     string // computed column expression: GENERATED ALWAYS AS (expr) STORED
}

// generatedColumnOf parses the `generated` tag. The value is either the keyword
// `identity` (optionally combined with the mode `always` / `by default`) for an
// identity column, or any other value, which is taken verbatim as the expression
// of a STORED computed column.
func generatedColumnOf(field *schema.Field) (generatedColumn, bool) {
	value, ok := field.TagSettings["GENERATED"]
	if !ok {
		return generatedColumn{}, false
	}

	// Ignore an empty value or a bare `generated` tag, which the tag parser
	// stores as the upper-cased key, rather than treating it as an expression.
	if value = strings.TrimSpace(value); value == "" || value == "GENERATED" {
		return generatedColumn{}, false
	}

	if mode, isIdentity := identityMode(value); isIdentity {
		return generatedColumn{identity: true, mode: mode}, true
	}

	return generatedColumn{expr: value}, true
}

// identityMode reports whether value describes an identity column and, if so,
// its generation mode. The recognized keywords are `identity`, `always` and
// `by default`, in any order; any other token means value is a computed
// expression rather than an identity specification.
func identityMode(value string) (mode string, ok bool) {
	mode = "BY DEFAULT"
	for _, token := range strings.Fields(strings.ToLower(value)) {
		switch token {
		case "identity":
			ok = true
		case "always":
			mode = "ALWAYS"
		case "by", "default":
			// part of the "by default" mode, which is the default; ignore
		default:
			return "", false
		}
	}
	return mode, ok
}
