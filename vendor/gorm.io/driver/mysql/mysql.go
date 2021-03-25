package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Config struct {
	DriverName                string
	DSN                       string
	Conn                      gorm.ConnPool
	SkipInitializeWithVersion bool
	DefaultStringSize         uint
	DisableDatetimePrecision  bool
	DontSupportRenameIndex    bool
	DontSupportRenameColumn   bool
}

type Dialector struct {
	*Config
}

func Open(dsn string) gorm.Dialector {
	return &Dialector{Config: &Config{DSN: dsn}}
}

func New(config Config) gorm.Dialector {
	return &Dialector{Config: &config}
}

func (dialector Dialector) Name() string {
	return "mysql"
}

func (dialector Dialector) Initialize(db *gorm.DB) (err error) {
	ctx := context.Background()

	// register callbacks
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})

	db.Callback().Update().Replace("gorm:update", Update)

	if dialector.DriverName == "" {
		dialector.DriverName = "mysql"
	}

	if dialector.Conn != nil {
		db.ConnPool = dialector.Conn
	} else {
		db.ConnPool, err = sql.Open(dialector.DriverName, dialector.DSN)
		if err != nil {
			return err
		}
	}

	if !dialector.Config.SkipInitializeWithVersion {
		var version string
		err = db.ConnPool.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
		if err != nil {
			return err
		}

		if strings.Contains(version, "MariaDB") {
			dialector.Config.DontSupportRenameIndex = true
			dialector.Config.DontSupportRenameColumn = true
		} else if strings.HasPrefix(version, "5.6.") {
			dialector.Config.DontSupportRenameIndex = true
			dialector.Config.DontSupportRenameColumn = true
		} else if strings.HasPrefix(version, "5.7.") {
			dialector.Config.DontSupportRenameColumn = true
		} else if strings.HasPrefix(version, "5.") {
			dialector.Config.DisableDatetimePrecision = true
			dialector.Config.DontSupportRenameIndex = true
			dialector.Config.DontSupportRenameColumn = true
		}
	}

	for k, v := range dialector.ClauseBuilders() {
		db.ClauseBuilders[k] = v
	}
	return
}

func (dialector Dialector) ClauseBuilders() map[string]clause.ClauseBuilder {
	return map[string]clause.ClauseBuilder{
		"ON CONFLICT": func(c clause.Clause, builder clause.Builder) {
			if onConflict, ok := c.Expression.(clause.OnConflict); ok {
				builder.WriteString("ON DUPLICATE KEY UPDATE ")
				if len(onConflict.DoUpdates) == 0 {
					if s := builder.(*gorm.Statement).Schema; s != nil {
						var column clause.Column
						onConflict.DoNothing = false

						if s.PrioritizedPrimaryField != nil {
							column = clause.Column{Name: s.PrioritizedPrimaryField.DBName}
						} else if len(s.DBNames) > 0 {
							column = clause.Column{Name: s.DBNames[0]}
						}

						if column.Name != "" {
							onConflict.DoUpdates = []clause.Assignment{{Column: column, Value: column}}
						}
					}
				}

				for idx, assignment := range onConflict.DoUpdates {
					if idx > 0 {
						builder.WriteByte(',')
					}

					builder.WriteQuoted(assignment.Column)
					builder.WriteByte('=')
					if column, ok := assignment.Value.(clause.Column); ok && column.Table == "excluded" {
						column.Table = ""
						builder.WriteString("VALUES(")
						builder.WriteQuoted(column)
						builder.WriteByte(')')
					} else {
						builder.AddVar(builder, assignment.Value)
					}
				}
			} else {
				c.Build(builder)
			}
		},
		"VALUES": func(c clause.Clause, builder clause.Builder) {
			if values, ok := c.Expression.(clause.Values); ok && len(values.Columns) == 0 {
				builder.WriteString("VALUES()")
				return
			}
			c.Build(builder)
		},
	}
}

func (dialector Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (dialector Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{
		Migrator: migrator.Migrator{
			Config: migrator.Config{
				DB:        db,
				Dialector: dialector,
			},
		},
		Dialector: dialector,
	}
}

func (dialector Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

func (dialector Dialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('`')
	if strings.Contains(str, ".") {
		for idx, str := range strings.Split(str, ".") {
			if idx > 0 {
				writer.WriteString(".`")
			}
			writer.WriteString(str)
			writer.WriteByte('`')
		}
	} else {
		writer.WriteString(str)
		writer.WriteByte('`')
	}
}

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `'`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		sqlType := "bigint"
		switch {
		case field.Size <= 8:
			sqlType = "tinyint"
		case field.Size <= 16:
			sqlType = "smallint"
		case field.Size <= 24:
			sqlType = "mediumint"
		case field.Size <= 32:
			sqlType = "int"
		}

		if field.DataType == schema.Uint {
			sqlType += " unsigned"
		}

		if field.AutoIncrement {
			sqlType += " AUTO_INCREMENT"
		}
		return sqlType
	case schema.Float:
		if field.Precision > 0 {
			return fmt.Sprintf("decimal(%d, %d)", field.Precision, field.Scale)
		}

		if field.Size <= 32 {
			return "float"
		}
		return "double"
	case schema.String:
		size := field.Size
		defaultSize := dialector.DefaultStringSize

		if size == 0 {
			if defaultSize > 0 {
				size = int(defaultSize)
			} else {
				hasIndex := field.TagSettings["INDEX"] != "" || field.TagSettings["UNIQUE"] != ""
				// TEXT, GEOMETRY or JSON column can't have a default value
				if field.PrimaryKey || field.HasDefaultValue || hasIndex {
					size = 191 // utf8mb4
				}
			}
		}

		if size >= 65536 && size <= int(math.Pow(2, 24)) {
			return "mediumtext"
		} else if size > int(math.Pow(2, 24)) || size <= 0 {
			return "longtext"
		}
		return fmt.Sprintf("varchar(%d)", size)
	case schema.Time:
		precision := ""

		if !dialector.DisableDatetimePrecision {
			if field.Precision == 0 {
				field.Precision = 3
			}

			if field.Precision > 0 {
				precision = fmt.Sprintf("(%d)", field.Precision)
			}
		}

		if field.NotNull || field.PrimaryKey {
			return "datetime" + precision
		}
		return "datetime" + precision + " NULL"
	case schema.Bytes:
		if field.Size > 0 && field.Size < 65536 {
			return fmt.Sprintf("varbinary(%d)", field.Size)
		}

		if field.Size >= 65536 && field.Size <= int(math.Pow(2, 24)) {
			return "mediumblob"
		}

		return "longblob"
	}

	return string(field.DataType)
}

func (dialectopr Dialector) SavePoint(tx *gorm.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return nil
}

func (dialectopr Dialector) RollbackTo(tx *gorm.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return nil
}
