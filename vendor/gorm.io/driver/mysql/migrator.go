package mysql

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

const indexSql = `
SELECT
	TABLE_NAME,
	COLUMN_NAME,
	INDEX_NAME,
	NON_UNIQUE 
FROM
	information_schema.STATISTICS 
WHERE
	TABLE_SCHEMA = ? 
	AND TABLE_NAME = ? 
ORDER BY
	INDEX_NAME,
	SEQ_IN_INDEX`

var typeAliasMap = map[string][]string{
	"bool":    {"tinyint"},
	"tinyint": {"bool"},
}

type Migrator struct {
	migrator.Migrator
	Dialector
}

func (m Migrator) FullDataTypeOf(field *schema.Field) clause.Expr {
	expr := m.Migrator.FullDataTypeOf(field)

	if value, ok := field.TagSettings["COMMENT"]; ok {
		expr.SQL += " COMMENT " + m.Dialector.Explain("?", value)
	}

	return expr
}

// MigrateColumnUnique migrate column's UNIQUE constraint.
// In MySQL, ColumnType's Unique is affected by UniqueIndex, so we have to take care of the UniqueIndex.
func (m Migrator) MigrateColumnUnique(value interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	unique, ok := columnType.Unique()
	if !ok || field.PrimaryKey {
		return nil // skip primary key
	}

	queryTx, execTx := m.GetQueryAndExecTx()
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		// We're currently only receiving boolean values on `Unique` tag,
		// so the UniqueConstraint name is fixed
		constraint := m.DB.NamingStrategy.UniqueName(stmt.Table, field.DBName)
		if unique {
			// Clean up redundant unique indexes
			indexes, _ := queryTx.Migrator().GetIndexes(value)
			for _, index := range indexes {
				if uni, ok := index.Unique(); !ok || !uni {
					continue
				}
				if columns := index.Columns(); len(columns) != 1 || columns[0] != field.DBName {
					continue
				}
				if name := index.Name(); name == constraint || name == field.UniqueIndex {
					continue
				}
				if err := execTx.Migrator().DropIndex(value, index.Name()); err != nil {
					return err
				}
			}

			hasConstraint := queryTx.Migrator().HasConstraint(value, constraint)
			switch {
			case field.Unique && !hasConstraint:
				if field.Unique {
					if err := execTx.Migrator().CreateConstraint(value, constraint); err != nil {
						return err
					}
				}
			// field isn't Unique but ColumnType's Unique is reported by UniqueConstraint.
			case !field.Unique && hasConstraint:
				if err := execTx.Migrator().DropConstraint(value, constraint); err != nil {
					return err
				}
				if field.UniqueIndex != "" {
					if err := execTx.Migrator().CreateIndex(value, field.UniqueIndex); err != nil {
						return err
					}
				}
			}

			if field.UniqueIndex != "" && !queryTx.Migrator().HasIndex(value, field.UniqueIndex) {
				if err := execTx.Migrator().CreateIndex(value, field.UniqueIndex); err != nil {
					return err
				}
			}
		} else {
			if field.Unique {
				if err := execTx.Migrator().CreateConstraint(value, constraint); err != nil {
					return err
				}
			}
			if field.UniqueIndex != "" && !queryTx.Migrator().HasIndex(value, field.UniqueIndex) {
				if err := execTx.Migrator().CreateIndex(value, field.UniqueIndex); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (m Migrator) AddColumn(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		// avoid using the same name field
		f := stmt.Schema.LookUpField(name)
		if f == nil {
			return fmt.Errorf("failed to look up field with name: %s", name)
		}

		if !f.IgnoreMigration {
			fieldType := m.FullDataTypeOf(f)
			columnName := clause.Column{Name: f.DBName}
			values := []interface{}{m.CurrentTable(stmt), columnName, fieldType}
			var alterSql strings.Builder
			alterSql.WriteString("ALTER TABLE ? ADD ? ?")
			if f.PrimaryKey || strings.Contains(strings.ToLower(fieldType.SQL), "auto_increment") {
				alterSql.WriteString(", ADD PRIMARY KEY (?)")
				values = append(values, columnName)
			}
			return m.DB.Exec(alterSql.String(), values...).Error
		}

		return nil
	})
}

func (m Migrator) AlterColumn(value interface{}, field string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(field); field != nil {
				fullDataType := m.FullDataTypeOf(field)
				if m.Dialector.DontSupportRenameColumnUnique {
					fullDataType.SQL = strings.Replace(fullDataType.SQL, " UNIQUE ", " ", 1)
				}

				return m.DB.Exec(
					"ALTER TABLE ? MODIFY COLUMN ? ?",
					m.CurrentTable(stmt), clause.Column{Name: field.DBName}, fullDataType,
				).Error
			}
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (m Migrator) TiDBVersion() (isTiDB bool, major, minor, patch int, err error) {
	// TiDB version string looks like:
	// "5.7.25-TiDB-v6.5.0" or "5.7.25-TiDB-v6.4.0-serverless"
	tidbVersionArray := strings.Split(m.Dialector.ServerVersion, "-")
	if len(tidbVersionArray) < 3 || tidbVersionArray[1] != "TiDB" {
		// It isn't TiDB
		return
	}

	rawVersion := strings.TrimPrefix(tidbVersionArray[2], "v")
	realVersionArray := strings.Split(rawVersion, ".")
	if major, err = strconv.Atoi(realVersionArray[0]); err != nil {
		err = fmt.Errorf("failed to parse the version of TiDB, the major version is: %s", realVersionArray[0])
		return
	}

	if minor, err = strconv.Atoi(realVersionArray[1]); err != nil {
		err = fmt.Errorf("failed to parse the version of TiDB, the minor version is: %s", realVersionArray[1])
		return
	}

	if patch, err = strconv.Atoi(realVersionArray[2]); err != nil {
		err = fmt.Errorf("failed to parse the version of TiDB, the patch version is: %s", realVersionArray[2])
		return
	}

	isTiDB = true
	return
}

func (m Migrator) RenameColumn(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if !m.Dialector.DontSupportRenameColumn {
			return m.Migrator.RenameColumn(value, oldName, newName)
		}

		var field *schema.Field
		if stmt.Schema != nil {
			if f := stmt.Schema.LookUpField(oldName); f != nil {
				oldName = f.DBName
				field = f
			}

			if f := stmt.Schema.LookUpField(newName); f != nil {
				newName = f.DBName
				field = f
			}
		}

		if field != nil {
			return m.DB.Exec(
				"ALTER TABLE ? CHANGE ? ? ?",
				m.CurrentTable(stmt), clause.Column{Name: oldName},
				clause.Column{Name: newName}, m.FullDataTypeOf(field),
			).Error
		}

		return fmt.Errorf("failed to look up field with name: %s", newName)
	})
}

func (m Migrator) DropConstraint(value interface{}, name string) error {
	if !m.Dialector.Config.DontSupportDropConstraint {
		return m.Migrator.DropConstraint(value, name)
	}

	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		constraint, table := m.GuessConstraintInterfaceAndTable(stmt, name)
		if constraint != nil {
			name = constraint.GetName()
			switch constraint.(type) {
			case *schema.Constraint:
				return m.DB.Exec("ALTER TABLE ? DROP FOREIGN KEY ?", clause.Table{Name: table}, clause.Column{Name: name}).Error
			case *schema.CheckConstraint:
				return m.DB.Exec("ALTER TABLE ? DROP CHECK ?", clause.Table{Name: table}, clause.Column{Name: name}).Error
			}
		}
		if m.HasIndex(value, name) {
			return m.DB.Exec("ALTER TABLE ? DROP INDEX ?", clause.Table{Name: table}, clause.Column{Name: name}).Error
		}
		return nil
	})
}

func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	if !m.Dialector.DontSupportRenameIndex {
		return m.RunWithValue(value, func(stmt *gorm.Statement) error {
			return m.DB.Exec(
				"ALTER TABLE ? RENAME INDEX ? TO ?",
				m.CurrentTable(stmt), clause.Column{Name: oldName}, clause.Column{Name: newName},
			).Error
		})
	}

	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		err := m.DropIndex(value, oldName)
		if err != nil {
			return err
		}

		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(newName); idx == nil {
				if idx = stmt.Schema.LookIndex(oldName); idx != nil {
					opts := m.BuildIndexOptions(idx.Fields, stmt)
					values := []interface{}{clause.Column{Name: newName}, m.CurrentTable(stmt), opts}

					createIndexSQL := "CREATE "
					if idx.Class != "" {
						createIndexSQL += idx.Class + " "
					}
					createIndexSQL += "INDEX ? ON ??"

					if idx.Type != "" {
						createIndexSQL += " USING " + idx.Type
					}

					return m.DB.Exec(createIndexSQL, values...).Error
				}
			}
		}

		return m.CreateIndex(value, newName)
	})

}

func (m Migrator) DropTable(values ...interface{}) error {
	values = m.ReorderModels(values, false)
	return m.DB.Connection(func(tx *gorm.DB) error {
		tx.Exec("SET FOREIGN_KEY_CHECKS = 0;")
		for i := len(values) - 1; i >= 0; i-- {
			if err := m.RunWithValue(values[i], func(stmt *gorm.Statement) error {
				return tx.Exec("DROP TABLE IF EXISTS ? CASCADE", m.CurrentTable(stmt)).Error
			}); err != nil {
				return err
			}
		}
		return tx.Exec("SET FOREIGN_KEY_CHECKS = 1;").Error
	})
}

// ColumnTypes column types return columnTypes,error
func (m Migrator) ColumnTypes(value interface{}) ([]gorm.ColumnType, error) {
	columnTypes := make([]gorm.ColumnType, 0)
	err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
		var (
			currentDatabase, table = m.CurrentSchema(stmt, stmt.Table)
			columnTypeSQL          = "SELECT column_name, column_default, is_nullable = 'YES', data_type, character_maximum_length, column_type, column_key, extra, column_comment, numeric_precision, numeric_scale "
			rows, err              = m.DB.Session(&gorm.Session{}).Table(table).Limit(1).Rows()
		)

		if err != nil {
			return err
		}

		rawColumnTypes, err := rows.ColumnTypes()

		if err != nil {
			return err
		}

		if err := rows.Close(); err != nil {
			return err
		}

		if !m.DisableDatetimePrecision {
			columnTypeSQL += ", datetime_precision "
		}
		columnTypeSQL += "FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ORDINAL_POSITION"

		columns, rowErr := m.DB.Table(table).Raw(columnTypeSQL, currentDatabase, table).Rows()
		if rowErr != nil {
			return rowErr
		}

		defer columns.Close()

		for columns.Next() {
			var (
				column            migrator.ColumnType
				datetimePrecision sql.NullInt64
				extraValue        sql.NullString
				columnKey         sql.NullString
				values            = []interface{}{
					&column.NameValue, &column.DefaultValueValue, &column.NullableValue, &column.DataTypeValue, &column.LengthValue, &column.ColumnTypeValue, &columnKey, &extraValue, &column.CommentValue, &column.DecimalSizeValue, &column.ScaleValue,
				}
			)

			if !m.DisableDatetimePrecision {
				values = append(values, &datetimePrecision)
			}

			if scanErr := columns.Scan(values...); scanErr != nil {
				return scanErr
			}

			column.PrimaryKeyValue = sql.NullBool{Bool: false, Valid: true}
			column.UniqueValue = sql.NullBool{Bool: false, Valid: true}
			switch columnKey.String {
			case "PRI":
				column.PrimaryKeyValue = sql.NullBool{Bool: true, Valid: true}
			case "UNI":
				column.UniqueValue = sql.NullBool{Bool: true, Valid: true}
			}

			if strings.Contains(extraValue.String, "auto_increment") {
				column.AutoIncrementValue = sql.NullBool{Bool: true, Valid: true}
			}

			// only trim paired single-quotes
			s := column.DefaultValueValue.String
			for (len(s) >= 3 && s[0] == '\'' && s[len(s)-1] == '\'' && s[len(s)-2] != '\\') ||
				(len(s) == 2 && s == "''") {
				s = s[1 : len(s)-1]
			}
			column.DefaultValueValue.String = s
			if m.Dialector.DontSupportNullAsDefaultValue {
				// rewrite mariadb default value like other version
				if column.DefaultValueValue.Valid && column.DefaultValueValue.String == "NULL" {
					column.DefaultValueValue.Valid = false
					column.DefaultValueValue.String = ""
				}
			}

			if datetimePrecision.Valid {
				column.DecimalSizeValue = datetimePrecision
			}

			for _, c := range rawColumnTypes {
				if c.Name() == column.NameValue.String {
					column.SQLColumnType = c
					break
				}
			}

			columnTypes = append(columnTypes, column)
		}

		return nil
	})

	return columnTypes, err
}

func (m Migrator) CurrentDatabase() (name string) {
	baseName := m.Migrator.CurrentDatabase()
	m.DB.Raw(
		"SELECT SCHEMA_NAME from Information_schema.SCHEMATA where SCHEMA_NAME LIKE ? ORDER BY SCHEMA_NAME=? DESC,SCHEMA_NAME limit 1",
		baseName+"%", baseName).Scan(&name)
	return
}

func (m Migrator) GetTables() (tableList []string, err error) {
	err = m.DB.Raw("SELECT TABLE_NAME FROM information_schema.tables where TABLE_SCHEMA=?", m.CurrentDatabase()).
		Scan(&tableList).Error
	return
}

func (m Migrator) GetIndexes(value interface{}) ([]gorm.Index, error) {
	indexes := make([]gorm.Index, 0)
	err := m.RunWithValue(value, func(stmt *gorm.Statement) error {

		result := make([]*Index, 0)
		schema, table := m.CurrentSchema(stmt, stmt.Table)
		scanErr := m.DB.Table(table).Raw(indexSql, schema, table).Scan(&result).Error
		if scanErr != nil {
			return scanErr
		}
		indexMap, indexNames := groupByIndexName(result)

		for _, name := range indexNames {
			idx := indexMap[name]
			if len(idx) == 0 {
				continue
			}
			tempIdx := &migrator.Index{
				TableName: idx[0].TableName,
				NameValue: idx[0].IndexName,
				PrimaryKeyValue: sql.NullBool{
					Bool:  idx[0].IndexName == "PRIMARY",
					Valid: true,
				},
				UniqueValue: sql.NullBool{
					Bool:  idx[0].NonUnique == 0,
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
	TableName  string `gorm:"column:TABLE_NAME"`
	ColumnName string `gorm:"column:COLUMN_NAME"`
	IndexName  string `gorm:"column:INDEX_NAME"`
	NonUnique  int32  `gorm:"column:NON_UNIQUE"`
}

func groupByIndexName(indexList []*Index) (map[string][]*Index, []string) {
	columnIndexMap := make(map[string][]*Index, len(indexList))
	indexNames := make([]string, 0, len(indexList))
	for _, idx := range indexList {
		if _, ok := columnIndexMap[idx.IndexName]; !ok {
			indexNames = append(indexNames, idx.IndexName)
		}
		columnIndexMap[idx.IndexName] = append(columnIndexMap[idx.IndexName], idx)
	}
	return columnIndexMap, indexNames
}

func (m Migrator) CurrentSchema(stmt *gorm.Statement, table string) (string, string) {
	if tables := strings.Split(table, `.`); len(tables) == 2 {
		return tables[0], tables[1]
	}
	m.DB = m.DB.Table(table)
	return m.CurrentDatabase(), table
}

func (m Migrator) GetTypeAliases(databaseTypeName string) []string {
	return typeAliasMap[databaseTypeName]
}

// TableType table type return tableType,error
func (m Migrator) TableType(value interface{}) (tableType gorm.TableType, err error) {
	var table migrator.TableType

	err = m.RunWithValue(value, func(stmt *gorm.Statement) error {
		var (
			values = []interface{}{
				&table.SchemaValue, &table.NameValue, &table.TypeValue, &table.CommentValue,
			}
			currentDatabase, tableName = m.CurrentSchema(stmt, stmt.Table)
			tableTypeSQL               = "SELECT table_schema, table_name, table_type, table_comment FROM information_schema.tables WHERE table_schema = ? AND table_name = ?"
		)

		row := m.DB.Table(tableName).Raw(tableTypeSQL, currentDatabase, tableName).Row()

		if scanErr := row.Scan(values...); scanErr != nil {
			return scanErr
		}

		return nil
	})

	return table, err
}
