package postgres

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// See https://stackoverflow.com/questions/2204058/list-columns-with-indexes-in-postgresql
// Here are some changes:
// - use `LEFT JOIN` instead of `CROSS JOIN`
// - exclude indexes used to support constraints (they are auto-generated)
const indexSql = `
SELECT
	ct.relname AS table_name,
	ci.relname AS index_name,
	i.indisunique AS non_unique,
	i.indisprimary AS primary,
	a.attname AS column_name
FROM
	pg_index i
	LEFT JOIN pg_class ct ON ct.oid = i.indrelid
	LEFT JOIN pg_class ci ON ci.oid = i.indexrelid
	LEFT JOIN pg_attribute a ON a.attrelid = ct.oid
	LEFT JOIN pg_constraint con ON con.conindid = i.indexrelid
WHERE
	a.attnum = ANY(i.indkey)
	AND con.oid IS NULL
	AND ct.relkind = 'r'
	AND ct.relname = ?
`

var typeAliasMap = map[string][]string{
	"int":                      {"integer"},
	"int2":                     {"smallint"},
	"int4":                     {"integer"},
	"int8":                     {"bigint"},
	"smallint":                 {"int2"},
	"integer":                  {"int4"},
	"bigint":                   {"int8"},
	"decimal":                  {"numeric"},
	"numeric":                  {"decimal"},
	"timestamptz":              {"timestamp with time zone"},
	"timestamp with time zone": {"timestamptz"},
	"bool":                     {"boolean"},
	"boolean":                  {"bool"},
	"serial2":                  {"smallserial"},
	"serial4":                  {"serial"},
	"serial8":                  {"bigserial"},
	"varbit":                   {"bit varying"},
	"char":                     {"character"},
	"varchar":                  {"character varying"},
	"float4":                   {"real"},
	"float8":                   {"double precision"},
	"timetz":                   {"time with time zone"},
}

type Migrator struct {
	migrator.Migrator
}

// select querys ignore dryrun
func (m Migrator) queryRaw(sql string, values ...interface{}) (tx *gorm.DB) {
	queryTx := m.DB
	if m.DB.DryRun {
		queryTx = m.DB.Session(&gorm.Session{})
		queryTx.DryRun = false
	}
	return queryTx.Raw(sql, values...)
}

func (m Migrator) CurrentDatabase() (name string) {
	m.queryRaw("SELECT CURRENT_DATABASE()").Scan(&name)
	return
}

func (m Migrator) BuildIndexOptions(opts []schema.IndexOption, stmt *gorm.Statement) (results []interface{}) {
	for _, opt := range opts {
		str := stmt.Quote(opt.DBName)
		if opt.Expression != "" {
			str = opt.Expression
		}

		if opt.Collate != "" {
			str += " COLLATE " + opt.Collate
		}

		if opt.Sort != "" {
			str += " " + opt.Sort
		}
		results = append(results, clause.Expr{SQL: str})
	}
	return
}

func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(name); idx != nil {
				name = idx.Name
			}
		}
		currentSchema, curTable := m.CurrentSchema(stmt, stmt.Table)
		return m.queryRaw(
			"SELECT count(*) FROM pg_indexes WHERE tablename = ? AND indexname = ? AND schemaname = ?", curTable, name, currentSchema,
		).Scan(&count).Error
	})

	return count > 0
}

func (m Migrator) CreateIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(name); idx != nil {
				opts := m.BuildIndexOptions(idx.Fields, stmt)
				values := []interface{}{clause.Column{Name: idx.Name}, m.CurrentTable(stmt), opts}

				createIndexSQL := "CREATE "
				if idx.Class != "" {
					createIndexSQL += idx.Class + " "
				}
				createIndexSQL += "INDEX "

				if strings.TrimSpace(strings.ToUpper(idx.Option)) == "CONCURRENTLY" {
					createIndexSQL += "CONCURRENTLY "
				}

				createIndexSQL += "IF NOT EXISTS ? ON ?"

				if idx.Type != "" {
					createIndexSQL += " USING " + idx.Type + "(?)"
				} else {
					createIndexSQL += " ?"
				}

				if idx.Where != "" {
					createIndexSQL += " WHERE " + idx.Where
				}

				return m.DB.Exec(createIndexSQL, values...).Error
			}
		}

		return fmt.Errorf("failed to create index with name %v", name)
	})
}

func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Exec(
			"ALTER INDEX ? RENAME TO ?",
			clause.Column{Name: oldName}, clause.Column{Name: newName},
		).Error
	})
}

func (m Migrator) DropIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(name); idx != nil {
				name = idx.Name
			}
		}

		return m.DB.Exec("DROP INDEX ?", clause.Column{Name: name}).Error
	})
}

func (m Migrator) GetTables() (tableList []string, err error) {
	currentSchema, _ := m.CurrentSchema(m.DB.Statement, "")
	return tableList, m.queryRaw("SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_type = ?", currentSchema, "BASE TABLE").Scan(&tableList).Error
}

func (m Migrator) CreateTable(values ...interface{}) (err error) {
	if err = m.Migrator.CreateTable(values...); err != nil {
		return
	}
	for _, value := range m.ReorderModels(values, false) {
		if err = m.RunWithValue(value, func(stmt *gorm.Statement) error {
			if stmt.Schema != nil {
				for _, fieldName := range stmt.Schema.DBNames {
					field := stmt.Schema.FieldsByDBName[fieldName]
					if field.Comment != "" {
						if err := m.DB.Exec(
							"COMMENT ON COLUMN ?.? IS ?",
							m.CurrentTable(stmt), clause.Column{Name: field.DBName}, gorm.Expr(m.Migrator.Dialector.Explain("$1", field.Comment)),
						).Error; err != nil {
							return err
						}
					}
				}
			}
			return nil
		}); err != nil {
			return
		}
	}
	return
}

func (m Migrator) HasTable(value interface{}) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentSchema, curTable := m.CurrentSchema(stmt, stmt.Table)
		return m.queryRaw("SELECT count(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ? AND table_type = ?", currentSchema, curTable, "BASE TABLE").Scan(&count).Error
	})
	return count > 0
}

func (m Migrator) DropTable(values ...interface{}) error {
	values = m.ReorderModels(values, false)
	tx := m.DB.Session(&gorm.Session{})
	for i := len(values) - 1; i >= 0; i-- {
		if err := m.RunWithValue(values[i], func(stmt *gorm.Statement) error {
			return tx.Exec("DROP TABLE IF EXISTS ? CASCADE", m.CurrentTable(stmt)).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m Migrator) AddColumn(value interface{}, field string) error {
	if err := m.Migrator.AddColumn(value, field); err != nil {
		return err
	}
	m.resetPreparedStmts()

	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(field); field != nil {
				if field.Comment != "" {
					if err := m.DB.Exec(
						"COMMENT ON COLUMN ?.? IS ?",
						m.CurrentTable(stmt), clause.Column{Name: field.DBName}, gorm.Expr(m.Migrator.Dialector.Explain("$1", field.Comment)),
					).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		name := field
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(field); field != nil {
				name = field.DBName
			}
		}

		currentSchema, curTable := m.CurrentSchema(stmt, stmt.Table)
		return m.queryRaw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?",
			currentSchema, curTable, name,
		).Scan(&count).Error
	})

	return count > 0
}

func (m Migrator) MigrateColumn(value interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	// skip primary field
	if !field.PrimaryKey {
		if err := m.Migrator.MigrateColumn(value, field, columnType); err != nil {
			return err
		}
	}

	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		var description string
		currentSchema, curTable := m.CurrentSchema(stmt, stmt.Table)
		values := []interface{}{currentSchema, curTable, field.DBName, stmt.Table, currentSchema}
		checkSQL := "SELECT description FROM pg_catalog.pg_description "
		checkSQL += "WHERE objsubid = (SELECT ordinal_position FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?) "
		checkSQL += "AND objoid = (SELECT oid FROM pg_catalog.pg_class WHERE relname = ? AND relnamespace = "
		checkSQL += "(SELECT oid FROM pg_catalog.pg_namespace WHERE nspname = ?))"
		m.queryRaw(checkSQL, values...).Scan(&description)

		comment := strings.Trim(field.Comment, "'")
		comment = strings.Trim(comment, `"`)
		if field.Comment != "" && comment != description {
			if err := m.DB.Exec(
				"COMMENT ON COLUMN ?.? IS ?",
				m.CurrentTable(stmt), clause.Column{Name: field.DBName}, gorm.Expr(m.Migrator.Dialector.Explain("$1", field.Comment)),
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// AlterColumn alter value's `field` column' type based on schema definition
func (m Migrator) AlterColumn(value interface{}, field string) error {
	err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(field); field != nil {
				var (
					columnTypes, _  = m.DB.Migrator().ColumnTypes(value)
					fieldColumnType *migrator.ColumnType
				)
				for _, columnType := range columnTypes {
					if columnType.Name() == field.DBName {
						fieldColumnType, _ = columnType.(*migrator.ColumnType)
					}
				}

				fileType := clause.Expr{SQL: m.DataTypeOf(field)}
				// check for typeName and SQL name
				isSameType := true
				if !strings.EqualFold(fieldColumnType.DatabaseTypeName(), fileType.SQL) {
					isSameType = false
					// if different, also check for aliases
					aliases := m.GetTypeAliases(fieldColumnType.DatabaseTypeName())
					for _, alias := range aliases {
						if strings.HasPrefix(fileType.SQL, alias) {
							isSameType = true
							break
						}
					}
				}

				// not same, migrate
				if !isSameType {
					filedColumnAutoIncrement, _ := fieldColumnType.AutoIncrement()
					if field.AutoIncrement && filedColumnAutoIncrement { // update
						serialDatabaseType, _ := getSerialDatabaseType(fileType.SQL)
						if t, _ := fieldColumnType.ColumnType(); t != serialDatabaseType {
							if err := m.UpdateSequence(m.DB, stmt, field, serialDatabaseType); err != nil {
								return err
							}
						}
					} else if field.AutoIncrement && !filedColumnAutoIncrement { // create
						serialDatabaseType, _ := getSerialDatabaseType(fileType.SQL)
						if err := m.CreateSequence(m.DB, stmt, field, serialDatabaseType); err != nil {
							return err
						}
					} else if !field.AutoIncrement && filedColumnAutoIncrement { // delete
						if err := m.DeleteSequence(m.DB, stmt, field, fileType); err != nil {
							return err
						}
					} else {
						if err := m.modifyColumn(stmt, field, fileType, fieldColumnType); err != nil {
							return err
						}
					}
				}

				if null, _ := fieldColumnType.Nullable(); null == field.NotNull {
					if field.NotNull {
						if err := m.DB.Exec("ALTER TABLE ? ALTER COLUMN ? SET NOT NULL", m.CurrentTable(stmt), clause.Column{Name: field.DBName}).Error; err != nil {
							return err
						}
					} else {
						if err := m.DB.Exec("ALTER TABLE ? ALTER COLUMN ? DROP NOT NULL", m.CurrentTable(stmt), clause.Column{Name: field.DBName}).Error; err != nil {
							return err
						}
					}
				}

				if v, ok := fieldColumnType.DefaultValue(); (field.DefaultValueInterface == nil && ok) || v != field.DefaultValue {
					if field.HasDefaultValue && (field.DefaultValueInterface != nil || field.DefaultValue != "") {
						if field.DefaultValueInterface != nil {
							defaultStmt := &gorm.Statement{Vars: []interface{}{field.DefaultValueInterface}}
							m.Dialector.BindVarTo(defaultStmt, defaultStmt, field.DefaultValueInterface)
							if err := m.DB.Exec("ALTER TABLE ? ALTER COLUMN ? SET DEFAULT ?", m.CurrentTable(stmt), clause.Column{Name: field.DBName}, clause.Expr{SQL: m.Dialector.Explain(defaultStmt.SQL.String(), field.DefaultValueInterface)}).Error; err != nil {
								return err
							}
						} else if field.DefaultValue != "(-)" {
							if err := m.DB.Exec("ALTER TABLE ? ALTER COLUMN ? SET DEFAULT ?", m.CurrentTable(stmt), clause.Column{Name: field.DBName}, clause.Expr{SQL: field.DefaultValue}).Error; err != nil {
								return err
							}
						} else {
							if err := m.DB.Exec("ALTER TABLE ? ALTER COLUMN ? DROP DEFAULT", m.CurrentTable(stmt), clause.Column{Name: field.DBName}, clause.Expr{SQL: field.DefaultValue}).Error; err != nil {
								return err
							}
						}
					}
				}
				return nil
			}
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})

	if err != nil {
		return err
	}
	m.resetPreparedStmts()
	return nil
}

func (m Migrator) modifyColumn(stmt *gorm.Statement, field *schema.Field, targetType clause.Expr, existingColumn *migrator.ColumnType) error {
	alterSQL := "ALTER TABLE ? ALTER COLUMN ? TYPE ? USING ?::?"
	isUncastableDefaultValue := false

	if targetType.SQL == "boolean" {
		switch existingColumn.DatabaseTypeName() {
		case "int2", "int8", "numeric":
			alterSQL = "ALTER TABLE ? ALTER COLUMN ? TYPE ? USING ?::int::?"
		}
		isUncastableDefaultValue = true
	}

	if dv, _ := existingColumn.DefaultValue(); dv != "" && isUncastableDefaultValue {
		if err := m.DB.Exec("ALTER TABLE ? ALTER COLUMN ? DROP DEFAULT", m.CurrentTable(stmt), clause.Column{Name: field.DBName}).Error; err != nil {
			return err
		}
	}
	if err := m.DB.Exec(alterSQL, m.CurrentTable(stmt), clause.Column{Name: field.DBName}, targetType, clause.Column{Name: field.DBName}, targetType).Error; err != nil {
		return err
	}
	return nil
}

func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		constraint, table := m.GuessConstraintInterfaceAndTable(stmt, name)
		if constraint != nil {
			name = constraint.GetName()
		}
		currentSchema, curTable := m.CurrentSchema(stmt, table)

		return m.queryRaw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.table_constraints WHERE table_schema = ? AND table_name = ? AND constraint_name = ?",
			currentSchema, curTable, name,
		).Scan(&count).Error
	})

	return count > 0
}

func (m Migrator) ColumnTypes(value interface{}) (columnTypes []gorm.ColumnType, err error) {
	columnTypes = make([]gorm.ColumnType, 0)
	err = m.RunWithValue(value, func(stmt *gorm.Statement) error {
		var (
			currentDatabase      = m.DB.Migrator().CurrentDatabase()
			currentSchema, table = m.CurrentSchema(stmt, stmt.Table)
			columns, err         = m.queryRaw(
				"SELECT c.column_name, c.is_nullable = 'YES', c.udt_name, c.character_maximum_length, c.numeric_precision, c.numeric_precision_radix, c.numeric_scale, c.datetime_precision, 8 * typlen, c.column_default, pd.description, c.identity_increment FROM information_schema.columns AS c JOIN pg_type AS pgt ON c.udt_name = pgt.typname LEFT JOIN pg_catalog.pg_description as pd ON pd.objsubid = c.ordinal_position AND pd.objoid = (SELECT oid FROM pg_catalog.pg_class WHERE relname = c.table_name AND relnamespace = (SELECT oid FROM pg_catalog.pg_namespace WHERE nspname = c.table_schema)) where table_catalog = ? AND table_schema = ? AND table_name = ?",
				currentDatabase, currentSchema, table).Rows()
		)

		if err != nil {
			return err
		}

		for columns.Next() {
			var (
				column = &migrator.ColumnType{
					PrimaryKeyValue: sql.NullBool{Valid: true},
					UniqueValue:     sql.NullBool{Valid: true},
				}
				datetimePrecision sql.NullInt64
				radixValue        sql.NullInt64
				typeLenValue      sql.NullInt64
				identityIncrement sql.NullString
			)

			err = columns.Scan(
				&column.NameValue, &column.NullableValue, &column.DataTypeValue, &column.LengthValue, &column.DecimalSizeValue,
				&radixValue, &column.ScaleValue, &datetimePrecision, &typeLenValue, &column.DefaultValueValue, &column.CommentValue, &identityIncrement,
			)
			if err != nil {
				return err
			}

			if typeLenValue.Valid && typeLenValue.Int64 > 0 {
				column.LengthValue = typeLenValue
			}

			if (strings.HasPrefix(column.DefaultValueValue.String, "nextval('") &&
				strings.HasSuffix(column.DefaultValueValue.String, "seq'::regclass)")) || (identityIncrement.Valid && identityIncrement.String != "") {
				column.AutoIncrementValue = sql.NullBool{Bool: true, Valid: true}
				column.DefaultValueValue = sql.NullString{}
			}

			if column.DefaultValueValue.Valid {
				column.DefaultValueValue.String = parseDefaultValueValue(column.DefaultValueValue.String)
			}

			if datetimePrecision.Valid {
				column.DecimalSizeValue = datetimePrecision
			}

			columnTypes = append(columnTypes, column)
		}
		columns.Close()

		// assign sql column type
		{
			rows, rowsErr := m.GetRows(currentSchema, table)
			if rowsErr != nil {
				return rowsErr
			}
			rawColumnTypes, err := rows.ColumnTypes()
			if err != nil {
				return err
			}
			for _, columnType := range columnTypes {
				for _, c := range rawColumnTypes {
					if c.Name() == columnType.Name() {
						columnType.(*migrator.ColumnType).SQLColumnType = c
						break
					}
				}
			}
			rows.Close()
		}

		// check primary, unique field
		{
			columnTypeRows, err := m.queryRaw("SELECT constraint_name FROM information_schema.table_constraints tc JOIN information_schema.constraint_column_usage AS ccu USING (constraint_schema, constraint_catalog, table_name, constraint_name) JOIN information_schema.columns AS c ON c.table_schema = tc.constraint_schema AND tc.table_name = c.table_name AND ccu.column_name = c.column_name WHERE constraint_type IN ('PRIMARY KEY', 'UNIQUE') AND c.table_catalog = ? AND c.table_schema = ? AND c.table_name = ? AND constraint_type = ?", currentDatabase, currentSchema, table, "UNIQUE").Rows()
			if err != nil {
				return err
			}
			uniqueContraints := map[string]int{}
			for columnTypeRows.Next() {
				var constraintName string
				columnTypeRows.Scan(&constraintName)
				uniqueContraints[constraintName]++
			}
			columnTypeRows.Close()

			columnTypeRows, err = m.queryRaw("SELECT c.column_name, constraint_name, constraint_type FROM information_schema.table_constraints tc JOIN information_schema.constraint_column_usage AS ccu USING (constraint_schema, constraint_catalog, table_name, constraint_name) JOIN information_schema.columns AS c ON c.table_schema = tc.constraint_schema AND tc.table_name = c.table_name AND ccu.column_name = c.column_name WHERE constraint_type IN ('PRIMARY KEY', 'UNIQUE') AND c.table_catalog = ? AND c.table_schema = ? AND c.table_name = ?", currentDatabase, currentSchema, table).Rows()
			if err != nil {
				return err
			}
			for columnTypeRows.Next() {
				var name, constraintName, columnType string
				columnTypeRows.Scan(&name, &constraintName, &columnType)
				for _, c := range columnTypes {
					mc := c.(*migrator.ColumnType)
					if mc.NameValue.String == name {
						switch columnType {
						case "PRIMARY KEY":
							mc.PrimaryKeyValue = sql.NullBool{Bool: true, Valid: true}
						case "UNIQUE":
							if uniqueContraints[constraintName] == 1 {
								mc.UniqueValue = sql.NullBool{Bool: true, Valid: true}
							}
						}
						break
					}
				}
			}
			columnTypeRows.Close()
		}

		// check column type
		{
			dataTypeRows, err := m.queryRaw(`SELECT a.attname as column_name, format_type(a.atttypid, a.atttypmod) AS data_type
		FROM pg_attribute a JOIN pg_class b ON a.attrelid = b.oid AND relnamespace = (SELECT oid FROM pg_catalog.pg_namespace WHERE nspname = ?)
		WHERE a.attnum > 0 -- hide internal columns
		AND NOT a.attisdropped -- hide deleted columns
		AND b.relname = ?`, currentSchema, table).Rows()
			if err != nil {
				return err
			}

			for dataTypeRows.Next() {
				var name, dataType string
				dataTypeRows.Scan(&name, &dataType)
				for _, c := range columnTypes {
					mc := c.(*migrator.ColumnType)
					if mc.NameValue.String == name {
						mc.ColumnTypeValue = sql.NullString{String: dataType, Valid: true}
						// Handle array type: _text -> text[] , _int4 -> integer[]
						// Not support array size limits and array size limits because:
						// https://www.postgresql.org/docs/current/arrays.html#ARRAYS-DECLARATION
						if strings.HasPrefix(mc.DataTypeValue.String, "_") {
							mc.DataTypeValue = sql.NullString{String: dataType, Valid: true}
						}
						break
					}
				}
			}
			dataTypeRows.Close()
		}

		return err
	})
	return
}

func (m Migrator) GetRows(currentSchema interface{}, table interface{}) (*sql.Rows, error) {
	name := table.(string)
	if _, ok := currentSchema.(string); ok {
		name = fmt.Sprintf("%v.%v", currentSchema, table)
	}

	return m.DB.Session(&gorm.Session{}).Table(name).Limit(1).Scopes(func(d *gorm.DB) *gorm.DB {
		dialector, _ := m.Dialector.(Dialector)
		// use simple protocol
		if !m.DB.PrepareStmt && (dialector.Config != nil && (dialector.Config.DriverName == "" || dialector.Config.DriverName == "pgx")) {
			d.Statement.Vars = append([]interface{}{pgx.QueryExecModeSimpleProtocol}, d.Statement.Vars...)
		}
		return d
	}).Rows()
}

func (m Migrator) CurrentSchema(stmt *gorm.Statement, table string) (interface{}, interface{}) {
	if strings.Contains(table, ".") {
		if tables := strings.Split(table, `.`); len(tables) == 2 {
			return tables[0], tables[1]
		}
	}

	if stmt.TableExpr != nil {
		if tables := strings.Split(stmt.TableExpr.SQL, `"."`); len(tables) == 2 {
			return strings.TrimPrefix(tables[0], `"`), table
		}
	}
	return clause.Expr{SQL: "CURRENT_SCHEMA()"}, table
}

func (m Migrator) CreateSequence(tx *gorm.DB, stmt *gorm.Statement, field *schema.Field,
	serialDatabaseType string) (err error) {

	_, table := m.CurrentSchema(stmt, stmt.Table)
	tableName := table.(string)

	sequenceName := strings.Join([]string{tableName, field.DBName, "seq"}, "_")
	if err = tx.Exec(`CREATE SEQUENCE IF NOT EXISTS ? AS ?`, clause.Expr{SQL: sequenceName},
		clause.Expr{SQL: serialDatabaseType}).Error; err != nil {
		return err
	}

	if err := tx.Exec("ALTER TABLE ? ALTER COLUMN ? SET DEFAULT nextval('?')",
		clause.Expr{SQL: tableName}, clause.Expr{SQL: field.DBName}, clause.Expr{SQL: sequenceName}).Error; err != nil {
		return err
	}

	if err := tx.Exec("ALTER SEQUENCE ? OWNED BY ?.?",
		clause.Expr{SQL: sequenceName}, clause.Expr{SQL: tableName}, clause.Expr{SQL: field.DBName}).Error; err != nil {
		return err
	}
	return
}

func (m Migrator) UpdateSequence(tx *gorm.DB, stmt *gorm.Statement, field *schema.Field,
	serialDatabaseType string) (err error) {

	sequenceName, err := m.getColumnSequenceName(tx, stmt, field)
	if err != nil {
		return err
	}

	if err = tx.Exec(`ALTER SEQUENCE IF EXISTS ? AS ?`, clause.Expr{SQL: sequenceName}, clause.Expr{SQL: serialDatabaseType}).Error; err != nil {
		return err
	}

	if err := tx.Exec("ALTER TABLE ? ALTER COLUMN ? TYPE ?",
		m.CurrentTable(stmt), clause.Expr{SQL: field.DBName}, clause.Expr{SQL: serialDatabaseType}).Error; err != nil {
		return err
	}
	return
}

func (m Migrator) DeleteSequence(tx *gorm.DB, stmt *gorm.Statement, field *schema.Field,
	fileType clause.Expr) (err error) {

	sequenceName, err := m.getColumnSequenceName(tx, stmt, field)
	if err != nil {
		return err
	}

	if err := tx.Exec("ALTER TABLE ? ALTER COLUMN ? TYPE ?", m.CurrentTable(stmt), clause.Column{Name: field.DBName}, fileType).Error; err != nil {
		return err
	}

	if err := tx.Exec("ALTER TABLE ? ALTER COLUMN ? DROP DEFAULT",
		m.CurrentTable(stmt), clause.Expr{SQL: field.DBName}).Error; err != nil {
		return err
	}

	if err = tx.Exec(`DROP SEQUENCE IF EXISTS ?`, clause.Expr{SQL: sequenceName}).Error; err != nil {
		return err
	}

	return
}

func (m Migrator) getColumnSequenceName(tx *gorm.DB, stmt *gorm.Statement, field *schema.Field) (
	sequenceName string, err error) {
	_, table := m.CurrentSchema(stmt, stmt.Table)

	// DefaultValueValue is reset by ColumnTypes, search again.
	var columnDefault string
	err = tx.Raw(
		`SELECT column_default FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
		table, field.DBName).Scan(&columnDefault).Error

	if err != nil {
		return
	}

	sequenceName = strings.TrimSuffix(
		strings.TrimPrefix(columnDefault, `nextval('`),
		`'::regclass)`,
	)
	return
}

func (m Migrator) GetIndexes(value interface{}) ([]gorm.Index, error) {
	indexes := make([]gorm.Index, 0)

	err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
		result := make([]*Index, 0)
		scanErr := m.queryRaw(indexSql, stmt.Table).Scan(&result).Error
		if scanErr != nil {
			return scanErr
		}
		indexMap := groupByIndexName(result)
		for _, idx := range indexMap {
			tempIdx := &migrator.Index{
				TableName: idx[0].TableName,
				NameValue: idx[0].IndexName,
				PrimaryKeyValue: sql.NullBool{
					Bool:  idx[0].Primary,
					Valid: true,
				},
				UniqueValue: sql.NullBool{
					Bool:  idx[0].NonUnique,
					Valid: true,
				},
			}
			for _, x := range idx {
				tempIdx.ColumnList = append(tempIdx.ColumnList, x.ColumnName)
			}
			indexes = append(indexes, tempIdx)
		}
		return nil
	})
	return indexes, err
}

// Index table index info
type Index struct {
	TableName  string `gorm:"column:table_name"`
	ColumnName string `gorm:"column:column_name"`
	IndexName  string `gorm:"column:index_name"`
	NonUnique  bool   `gorm:"column:non_unique"`
	Primary    bool   `gorm:"column:primary"`
}

func groupByIndexName(indexList []*Index) map[string][]*Index {
	columnIndexMap := make(map[string][]*Index, len(indexList))
	for _, idx := range indexList {
		columnIndexMap[idx.IndexName] = append(columnIndexMap[idx.IndexName], idx)
	}
	return columnIndexMap
}

func (m Migrator) GetTypeAliases(databaseTypeName string) []string {
	return typeAliasMap[databaseTypeName]
}

// should reset prepared stmts when table changed
func (m Migrator) resetPreparedStmts() {
	if m.DB.PrepareStmt {
		if pdb, ok := m.DB.ConnPool.(*gorm.PreparedStmtDB); ok {
			pdb.Reset()
		}
	}
}

func (m Migrator) DropColumn(dst interface{}, field string) error {
	if err := m.Migrator.DropColumn(dst, field); err != nil {
		return err
	}

	m.resetPreparedStmts()
	return nil
}

func (m Migrator) RenameColumn(dst interface{}, oldName, field string) error {
	if err := m.Migrator.RenameColumn(dst, oldName, field); err != nil {
		return err
	}

	m.resetPreparedStmts()
	return nil
}

func parseDefaultValueValue(defaultValue string) string {
	value := regexp.MustCompile(`^(.*?)(?:::.*)?$`).ReplaceAllString(defaultValue, "$1")
	return strings.Trim(value, "'")
}
