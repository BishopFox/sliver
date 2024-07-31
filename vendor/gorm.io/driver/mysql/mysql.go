package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

const (
	DefaultDriverName = "mysql"

	AutoRandomTag = "auto_random()" // Treated as an auto_random field for tidb
)

type Config struct {
	DriverName                    string
	ServerVersion                 string
	DSN                           string
	DSNConfig                     *mysql.Config
	Conn                          gorm.ConnPool
	SkipInitializeWithVersion     bool
	DefaultStringSize             uint
	DefaultDatetimePrecision      *int
	DisableWithReturning          bool
	DisableDatetimePrecision      bool
	DontSupportRenameIndex        bool
	DontSupportRenameColumn       bool
	DontSupportForShareClause     bool
	DontSupportNullAsDefaultValue bool
	DontSupportRenameColumnUnique bool
	// As of MySQL 8.0.19, ALTER TABLE permits more general (and SQL standard) syntax
	// for dropping and altering existing constraints of any type.
	// see https://dev.mysql.com/doc/refman/8.0/en/alter-table.html
	DontSupportDropConstraint bool
}

type Dialector struct {
	*Config
}

var (
	// CreateClauses create clauses
	CreateClauses = []string{"INSERT", "VALUES", "ON CONFLICT"}
	// QueryClauses query clauses
	QueryClauses = []string{}
	// UpdateClauses update clauses
	UpdateClauses = []string{"UPDATE", "SET", "WHERE", "ORDER BY", "LIMIT"}
	// DeleteClauses delete clauses
	DeleteClauses = []string{"DELETE", "FROM", "WHERE", "ORDER BY", "LIMIT"}

	defaultDatetimePrecision = 3
)

func Open(dsn string) gorm.Dialector {
	dsnConf, _ := mysql.ParseDSN(dsn)
	return &Dialector{Config: &Config{DSN: dsn, DSNConfig: dsnConf}}
}

func New(config Config) gorm.Dialector {
	switch {
	case config.DSN == "" && config.DSNConfig != nil:
		config.DSN = config.DSNConfig.FormatDSN()
	case config.DSN != "" && config.DSNConfig == nil:
		config.DSNConfig, _ = mysql.ParseDSN(config.DSN)
	}
	return &Dialector{Config: &config}
}

func (dialector Dialector) Name() string {
	return DefaultDriverName
}

// NowFunc return now func
func (dialector Dialector) NowFunc(n int) func() time.Time {
	return func() time.Time {
		round := time.Second / time.Duration(math.Pow10(n))
		return time.Now().Round(round)
	}
}

func (dialector Dialector) Apply(config *gorm.Config) error {
	if config.NowFunc != nil {
		return nil
	}

	if dialector.DefaultDatetimePrecision == nil {
		dialector.DefaultDatetimePrecision = &defaultDatetimePrecision
	}
	// while maintaining the readability of the code, separate the business logic from
	// the general part and leave it to the function to do it here.
	config.NowFunc = dialector.NowFunc(*dialector.DefaultDatetimePrecision)
	return nil
}

func (dialector Dialector) Initialize(db *gorm.DB) (err error) {
	if dialector.DriverName == "" {
		dialector.DriverName = DefaultDriverName
	}

	if dialector.DefaultDatetimePrecision == nil {
		dialector.DefaultDatetimePrecision = &defaultDatetimePrecision
	}

	if dialector.Conn != nil {
		db.ConnPool = dialector.Conn
	} else {
		db.ConnPool, err = sql.Open(dialector.DriverName, dialector.DSN)
		if err != nil {
			return err
		}
	}

	withReturning := false
	if !dialector.Config.SkipInitializeWithVersion {
		err = db.ConnPool.QueryRowContext(context.Background(), "SELECT VERSION()").Scan(&dialector.ServerVersion)
		if err != nil {
			return err
		}

		if strings.Contains(dialector.ServerVersion, "MariaDB") {
			dialector.Config.DontSupportRenameIndex = true
			dialector.Config.DontSupportRenameColumn = true
			dialector.Config.DontSupportForShareClause = true
			dialector.Config.DontSupportNullAsDefaultValue = true
			withReturning = checkVersion(dialector.ServerVersion, "10.5")
		} else if strings.HasPrefix(dialector.ServerVersion, "5.6.") {
			dialector.Config.DontSupportRenameIndex = true
			dialector.Config.DontSupportRenameColumn = true
			dialector.Config.DontSupportForShareClause = true
			dialector.Config.DontSupportDropConstraint = true
		} else if strings.HasPrefix(dialector.ServerVersion, "5.7.") {
			dialector.Config.DontSupportRenameColumn = true
			dialector.Config.DontSupportForShareClause = true
			dialector.Config.DontSupportDropConstraint = true
		} else if strings.HasPrefix(dialector.ServerVersion, "5.") {
			dialector.Config.DisableDatetimePrecision = true
			dialector.Config.DontSupportRenameIndex = true
			dialector.Config.DontSupportRenameColumn = true
			dialector.Config.DontSupportForShareClause = true
			dialector.Config.DontSupportDropConstraint = true
		}

		if strings.Contains(dialector.ServerVersion, "TiDB") {
			dialector.Config.DontSupportRenameColumnUnique = true
		}
	}

	// register callbacks
	callbackConfig := &callbacks.Config{
		CreateClauses: CreateClauses,
		QueryClauses:  QueryClauses,
		UpdateClauses: UpdateClauses,
		DeleteClauses: DeleteClauses,
	}

	if !dialector.Config.DisableWithReturning && withReturning {
		if !utils.Contains(callbackConfig.CreateClauses, "RETURNING") {
			callbackConfig.CreateClauses = append(callbackConfig.CreateClauses, "RETURNING")
		}

		if !utils.Contains(callbackConfig.UpdateClauses, "RETURNING") {
			callbackConfig.UpdateClauses = append(callbackConfig.UpdateClauses, "RETURNING")
		}

		if !utils.Contains(callbackConfig.DeleteClauses, "RETURNING") {
			callbackConfig.DeleteClauses = append(callbackConfig.DeleteClauses, "RETURNING")
		}
	}

	callbacks.RegisterDefaultCallbacks(db, callbackConfig)

	for k, v := range dialector.ClauseBuilders() {
		db.ClauseBuilders[k] = v
	}
	return
}

const (
	// ClauseOnConflict for clause.ClauseBuilder ON CONFLICT key
	ClauseOnConflict = "ON CONFLICT"
	// ClauseValues for clause.ClauseBuilder VALUES key
	ClauseValues = "VALUES"
	// ClauseFor for clause.ClauseBuilder FOR key
	ClauseFor = "FOR"
)

func (dialector Dialector) ClauseBuilders() map[string]clause.ClauseBuilder {
	clauseBuilders := map[string]clause.ClauseBuilder{
		ClauseOnConflict: func(c clause.Clause, builder clause.Builder) {
			onConflict, ok := c.Expression.(clause.OnConflict)
			if !ok {
				c.Build(builder)
				return
			}

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

					builder.(*gorm.Statement).AddClause(onConflict)
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
		},
		ClauseValues: func(c clause.Clause, builder clause.Builder) {
			if values, ok := c.Expression.(clause.Values); ok && len(values.Columns) == 0 {
				builder.WriteString("VALUES()")
				return
			}
			c.Build(builder)
		},
	}

	if dialector.Config.DontSupportForShareClause {
		clauseBuilders[ClauseFor] = func(c clause.Clause, builder clause.Builder) {
			if values, ok := c.Expression.(clause.Locking); ok && strings.EqualFold(values.Strength, "SHARE") {
				builder.WriteString("LOCK IN SHARE MODE")
				return
			}
			c.Build(builder)
		}
	}

	return clauseBuilders
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
	var (
		underQuoted, selfQuoted bool
		continuousBacktick      int8
		shiftDelimiter          int8
	)

	for _, v := range []byte(str) {
		switch v {
		case '`':
			continuousBacktick++
			if continuousBacktick == 2 {
				writer.WriteString("``")
				continuousBacktick = 0
			}
		case '.':
			if continuousBacktick > 0 || !selfQuoted {
				shiftDelimiter = 0
				underQuoted = false
				continuousBacktick = 0
				writer.WriteByte('`')
			}
			writer.WriteByte(v)
			continue
		default:
			if shiftDelimiter-continuousBacktick <= 0 && !underQuoted {
				writer.WriteByte('`')
				underQuoted = true
				if selfQuoted = continuousBacktick > 0; selfQuoted {
					continuousBacktick -= 1
				}
			}

			for ; continuousBacktick > 0; continuousBacktick -= 1 {
				writer.WriteString("``")
			}

			writer.WriteByte(v)
		}
		shiftDelimiter++
	}

	if continuousBacktick > 0 && !selfQuoted {
		writer.WriteString("``")
	}
	writer.WriteByte('`')
}

type localTimeInterface interface {
	In(loc *time.Location) time.Time
}

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	if dialector.DSNConfig != nil && dialector.DSNConfig.Loc != nil {
		for i, v := range vars {
			if p, ok := v.(localTimeInterface); ok {
				func(i int, t localTimeInterface) {
					defer func() {
						recover()
					}()
					vars[i] = t.In(dialector.DSNConfig.Loc)
				}(i, p)
			}
		}
	}
	return logger.ExplainSQL(sql, nil, `'`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		return dialector.getSchemaIntAndUnitType(field)
	case schema.Float:
		return dialector.getSchemaFloatType(field)
	case schema.String:
		return dialector.getSchemaStringType(field)
	case schema.Time:
		return dialector.getSchemaTimeType(field)
	case schema.Bytes:
		return dialector.getSchemaBytesType(field)
	default:
		return dialector.getSchemaCustomType(field)
	}
}

func (dialector Dialector) getSchemaFloatType(field *schema.Field) string {
	if field.Precision > 0 {
		return fmt.Sprintf("decimal(%d, %d)", field.Precision, field.Scale)
	}

	if field.Size <= 32 {
		return "float"
	}

	return "double"
}

func (dialector Dialector) getSchemaStringType(field *schema.Field) string {
	size := field.Size
	if size == 0 {
		if dialector.DefaultStringSize > 0 {
			size = int(dialector.DefaultStringSize)
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
	}

	if size > int(math.Pow(2, 24)) || size <= 0 {
		return "longtext"
	}

	return fmt.Sprintf("varchar(%d)", size)
}

func (dialector Dialector) getSchemaTimeType(field *schema.Field) string {
	if !dialector.DisableDatetimePrecision && field.Precision == 0 && field.TagSettings["PRECISION"] == "" {
		field.Precision = *dialector.DefaultDatetimePrecision
	}

	var precision string
	if field.Precision > 0 {
		precision = fmt.Sprintf("(%d)", field.Precision)
	}

	if field.NotNull || field.PrimaryKey {
		return "datetime" + precision
	}
	return "datetime" + precision + " NULL"
}

func (dialector Dialector) getSchemaBytesType(field *schema.Field) string {
	if field.Size > 0 && field.Size < 65536 {
		return fmt.Sprintf("varbinary(%d)", field.Size)
	}

	if field.Size >= 65536 && field.Size <= int(math.Pow(2, 24)) {
		return "mediumblob"
	}

	return "longblob"
}

// autoRandomType
// field.DataType MUST be `schema.Int` or `schema.Uint`
// Judgement logic:
// 1. Is PrimaryKey;
// 2. Has default value;
// 3. Default value is "auto_random()";
// 4. IGNORE the field.Size, it MUST be bigint;
// 5. CLEAR the default tag, and return true;
// 6. Otherwise, return false.
func autoRandomType(field *schema.Field) (bool, string) {
	if field.PrimaryKey && field.HasDefaultValue &&
		strings.ToLower(strings.TrimSpace(field.DefaultValue)) == AutoRandomTag {
		field.DefaultValue = ""

		sqlType := "bigint"
		if field.DataType == schema.Uint {
			sqlType += " unsigned"
		}
		sqlType += " auto_random"
		return true, sqlType
	}

	return false, ""
}

func (dialector Dialector) getSchemaIntAndUnitType(field *schema.Field) string {
	if autoRandom, typeString := autoRandomType(field); autoRandom {
		return typeString
	}

	constraint := func(sqlType string) string {
		if field.DataType == schema.Uint {
			sqlType += " unsigned"
		}
		if field.AutoIncrement {
			sqlType += " AUTO_INCREMENT"
		}
		return sqlType
	}

	switch {
	case field.Size <= 8:
		return constraint("tinyint")
	case field.Size <= 16:
		return constraint("smallint")
	case field.Size <= 24:
		return constraint("mediumint")
	case field.Size <= 32:
		return constraint("int")
	default:
		return constraint("bigint")
	}
}

func (dialector Dialector) getSchemaCustomType(field *schema.Field) string {
	sqlType := string(field.DataType)

	if field.AutoIncrement && !strings.Contains(strings.ToLower(sqlType), " auto_increment") {
		sqlType += " AUTO_INCREMENT"
	}

	return sqlType
}

func (dialector Dialector) SavePoint(tx *gorm.DB, name string) error {
	return tx.Exec("SAVEPOINT " + name).Error
}

func (dialector Dialector) RollbackTo(tx *gorm.DB, name string) error {
	return tx.Exec("ROLLBACK TO SAVEPOINT " + name).Error
}

// checkVersion newer or equal returns true, old returns false
func checkVersion(newVersion, oldVersion string) bool {
	if newVersion == oldVersion {
		return true
	}

	var (
		versionTrimmerRegexp = regexp.MustCompile(`^(\d+).*$`)

		newVersions = strings.Split(newVersion, ".")
		oldVersions = strings.Split(oldVersion, ".")
	)
	for idx, nv := range newVersions {
		if len(oldVersions) <= idx {
			return true
		}

		nvi, _ := strconv.Atoi(versionTrimmerRegexp.ReplaceAllString(nv, "$1"))
		ovi, _ := strconv.Atoi(versionTrimmerRegexp.ReplaceAllString(oldVersions[idx], "$1"))
		if nvi == ovi {
			continue
		}
		return nvi > ovi
	}

	return false
}
