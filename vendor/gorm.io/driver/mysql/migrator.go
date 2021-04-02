package mysql

import (
	"database/sql"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Migrator struct {
	migrator.Migrator
	Dialector
}

type Column struct {
	name              string
	nullable          sql.NullString
	datatype          string
	maxlen            sql.NullInt64
	precision         sql.NullInt64
	scale             sql.NullInt64
	datetimeprecision sql.NullInt64
}

func (c Column) Name() string {
	return c.name
}

func (c Column) DatabaseTypeName() string {
	return c.datatype
}

func (c Column) Length() (length int64, ok bool) {
	ok = c.maxlen.Valid
	if ok {
		length = c.maxlen.Int64
	} else {
		length = 0
	}
	return
}

func (c Column) Nullable() (nullable bool, ok bool) {
	if c.nullable.Valid {
		nullable, ok = c.nullable.String == "YES", true
	} else {
		nullable, ok = false, false
	}
	return
}

func (c Column) DecimalSize() (precision int64, scale int64, ok bool) {
	if c.precision.Valid {
		if c.scale.Valid {
			precision, scale, ok = c.precision.Int64, c.scale.Int64, true
		} else {
			precision, scale, ok = c.precision.Int64, 0, true
		}
	} else if c.datetimeprecision.Valid {
		precision, scale, ok = c.datetimeprecision.Int64, 0, true
	} else {
		precision, scale, ok = 0, 0, false
	}
	return
}

func (m Migrator) FullDataTypeOf(field *schema.Field) clause.Expr {
	expr := m.Migrator.FullDataTypeOf(field)

	if value, ok := field.TagSettings["COMMENT"]; ok {
		expr.SQL += " COMMENT " + m.Dialector.Explain("?", value)
	}

	return expr
}

func (m Migrator) AlterColumn(value interface{}, field string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			return m.DB.Exec(
				"ALTER TABLE ? MODIFY COLUMN ? ?",
				clause.Table{Name: stmt.Table}, clause.Column{Name: field.DBName}, m.FullDataTypeOf(field),
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (m Migrator) RenameColumn(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if m.Dialector.DontSupportRenameColumn {
			var field *schema.Field
			if f := stmt.Schema.LookUpField(oldName); f != nil {
				oldName = f.DBName
				field = f
			}

			if f := stmt.Schema.LookUpField(newName); f != nil {
				newName = f.DBName
				field = f
			}

			if field != nil {
				return m.DB.Exec(
					"ALTER TABLE ? CHANGE ? ? ?",
					clause.Table{Name: stmt.Table}, clause.Column{Name: oldName}, clause.Column{Name: newName}, m.FullDataTypeOf(field),
				).Error
			}
		} else {
			return m.Migrator.RenameColumn(value, oldName, newName)
		}

		return fmt.Errorf("failed to look up field with name: %s", newName)
	})
}

func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	if m.Dialector.DontSupportRenameIndex {
		return m.RunWithValue(value, func(stmt *gorm.Statement) error {
			err := m.DropIndex(value, oldName)
			if err == nil {
				if idx := stmt.Schema.LookIndex(newName); idx == nil {
					if idx = stmt.Schema.LookIndex(oldName); idx != nil {
						opts := m.BuildIndexOptions(idx.Fields, stmt)
						values := []interface{}{clause.Column{Name: newName}, clause.Table{Name: stmt.Table}, opts}

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

				err = m.CreateIndex(value, newName)
			}

			return err
		})
	} else {
		return m.RunWithValue(value, func(stmt *gorm.Statement) error {
			return m.DB.Exec(
				"ALTER TABLE ? RENAME INDEX ? TO ?",
				clause.Table{Name: stmt.Table}, clause.Column{Name: oldName}, clause.Column{Name: newName},
			).Error
		})
	}
}

func (m Migrator) DropTable(values ...interface{}) error {
	values = m.ReorderModels(values, false)
	tx := m.DB.Session(&gorm.Session{})
	tx.Exec("SET FOREIGN_KEY_CHECKS = 0;")
	for i := len(values) - 1; i >= 0; i-- {
		if err := m.RunWithValue(values[i], func(stmt *gorm.Statement) error {
			return tx.Exec("DROP TABLE IF EXISTS ? CASCADE", clause.Table{Name: stmt.Table}).Error
		}); err != nil {
			return err
		}
	}
	tx.Exec("SET FOREIGN_KEY_CHECKS = 1;")
	return nil
}

func (m Migrator) DropConstraint(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		for _, chk := range stmt.Schema.ParseCheckConstraints() {
			if chk.Name == name {
				return m.DB.Exec(
					"ALTER TABLE ? DROP CHECK ?",
					clause.Table{Name: stmt.Table}, clause.Column{Name: name},
				).Error
			}
		}

		return m.DB.Exec(
			"ALTER TABLE ? DROP FOREIGN KEY ?",
			clause.Table{Name: stmt.Table}, clause.Column{Name: name},
		).Error
	})
}

func (m Migrator) ColumnTypes(value interface{}) (columnTypes []gorm.ColumnType, err error) {
	columnTypes = make([]gorm.ColumnType, 0)
	err = m.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := m.DB.Migrator().CurrentDatabase()
		columns, err := m.DB.Raw(
			"SELECT column_name, is_nullable, data_type, character_maximum_length, "+
				"numeric_precision, numeric_scale, datetime_precision "+
				"FROM information_schema.columns WHERE table_schema = ? AND table_name = ?",
			currentDatabase, stmt.Table).Rows()
		if err != nil {
			return err
		}
		defer columns.Close()

		for columns.Next() {
			var column Column
			err = columns.Scan(
				&column.name,
				&column.nullable,
				&column.datatype,
				&column.maxlen,
				&column.precision,
				&column.scale,
				&column.datetimeprecision,
			)
			if err != nil {
				return err
			}
			columnTypes = append(columnTypes, column)
		}

		return err
	})
	return
}
