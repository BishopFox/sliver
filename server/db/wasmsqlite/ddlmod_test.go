package wasmsqlite

import (
	"database/sql"
	"testing"

	"gorm.io/gorm/migrator"
	"gorm.io/gorm/utils/tests"
)

func TestParseDDL(t *testing.T) {
	params := []struct {
		name    string
		sql     []string
		nFields int
		columns []migrator.ColumnType
	}{
		{"with_fk", []string{
			"CREATE TABLE `notes` (`id` integer NOT NULL,`text` varchar(500) DEFAULT \"hello\",`age` integer DEFAULT 18,`user_id` integer,PRIMARY KEY (`id`),CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))",
			"CREATE UNIQUE INDEX `idx_profiles_refer` ON `profiles`(`text`)",
		}, 6, []migrator.ColumnType{
			{NameValue: sql.NullString{String: "id", Valid: true}, DataTypeValue: sql.NullString{String: "integer", Valid: true}, ColumnTypeValue: sql.NullString{String: "integer", Valid: true}, PrimaryKeyValue: sql.NullBool{Bool: true, Valid: true}, NullableValue: sql.NullBool{Valid: true}, UniqueValue: sql.NullBool{Valid: true}, DefaultValueValue: sql.NullString{Valid: false}},
			{NameValue: sql.NullString{String: "text", Valid: true}, DataTypeValue: sql.NullString{String: "varchar", Valid: true}, LengthValue: sql.NullInt64{Int64: 500, Valid: true}, ColumnTypeValue: sql.NullString{String: "varchar(500)", Valid: true}, DefaultValueValue: sql.NullString{String: "hello", Valid: true}, NullableValue: sql.NullBool{Valid: true}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
			{NameValue: sql.NullString{String: "age", Valid: true}, DataTypeValue: sql.NullString{String: "integer", Valid: true}, ColumnTypeValue: sql.NullString{String: "integer", Valid: true}, DefaultValueValue: sql.NullString{String: "18", Valid: true}, NullableValue: sql.NullBool{Valid: true}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
			{NameValue: sql.NullString{String: "user_id", Valid: true}, DataTypeValue: sql.NullString{String: "integer", Valid: true}, ColumnTypeValue: sql.NullString{String: "integer", Valid: true}, DefaultValueValue: sql.NullString{Valid: false}, NullableValue: sql.NullBool{Valid: true}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
		},
		},
		{"with_check", []string{"CREATE TABLE Persons (ID int NOT NULL,LastName varchar(255) NOT NULL,FirstName varchar(255),Age int,CHECK (Age>=18),CHECK (FirstName<>'John'))"}, 6, []migrator.ColumnType{
			{NameValue: sql.NullString{String: "ID", Valid: true}, DataTypeValue: sql.NullString{String: "int", Valid: true}, ColumnTypeValue: sql.NullString{String: "int", Valid: true}, NullableValue: sql.NullBool{Valid: true}, DefaultValueValue: sql.NullString{Valid: false}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
			{NameValue: sql.NullString{String: "LastName", Valid: true}, DataTypeValue: sql.NullString{String: "varchar", Valid: true}, LengthValue: sql.NullInt64{Int64: 255, Valid: true}, ColumnTypeValue: sql.NullString{String: "varchar(255)", Valid: true}, NullableValue: sql.NullBool{Bool: false, Valid: true}, DefaultValueValue: sql.NullString{Valid: false}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
			{NameValue: sql.NullString{String: "FirstName", Valid: true}, DataTypeValue: sql.NullString{String: "varchar", Valid: true}, LengthValue: sql.NullInt64{Int64: 255, Valid: true}, ColumnTypeValue: sql.NullString{String: "varchar(255)", Valid: true}, DefaultValueValue: sql.NullString{Valid: false}, NullableValue: sql.NullBool{Valid: true}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
			{NameValue: sql.NullString{String: "Age", Valid: true}, DataTypeValue: sql.NullString{String: "int", Valid: true}, ColumnTypeValue: sql.NullString{String: "int", Valid: true}, DefaultValueValue: sql.NullString{Valid: false}, NullableValue: sql.NullBool{Valid: true}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
		}},
		{"lowercase", []string{"create table test (ID int NOT NULL)"}, 1, []migrator.ColumnType{
			{NameValue: sql.NullString{String: "ID", Valid: true}, DataTypeValue: sql.NullString{String: "int", Valid: true}, ColumnTypeValue: sql.NullString{String: "int", Valid: true}, NullableValue: sql.NullBool{Bool: false, Valid: true}, DefaultValueValue: sql.NullString{Valid: false}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
		},
		},
		{"no brackets", []string{"create table test"}, 0, nil},
		{"with_special_characters", []string{
			"CREATE TABLE `test` (`text` varchar(10) DEFAULT \"测试, \")",
		}, 1, []migrator.ColumnType{
			{NameValue: sql.NullString{String: "text", Valid: true}, DataTypeValue: sql.NullString{String: "varchar", Valid: true}, LengthValue: sql.NullInt64{Int64: 10, Valid: true}, ColumnTypeValue: sql.NullString{String: "varchar(10)", Valid: true}, DefaultValueValue: sql.NullString{String: "测试, ", Valid: true}, NullableValue: sql.NullBool{Valid: true}, UniqueValue: sql.NullBool{Valid: true}, PrimaryKeyValue: sql.NullBool{Valid: true}},
		},
		},
		{
			"table_name_with_dash",
			[]string{
				"CREATE TABLE `test-a` (`id` int NOT NULL)",
				"CREATE UNIQUE INDEX `idx_test-a_id` ON `test-a`(`id`)",
			},
			1,
			[]migrator.ColumnType{
				{
					NameValue:         sql.NullString{String: "id", Valid: true},
					DataTypeValue:     sql.NullString{String: "int", Valid: true},
					ColumnTypeValue:   sql.NullString{String: "int", Valid: true},
					NullableValue:     sql.NullBool{Bool: false, Valid: true},
					DefaultValueValue: sql.NullString{Valid: false},
					UniqueValue:       sql.NullBool{Valid: true},
					PrimaryKeyValue:   sql.NullBool{Valid: true},
				},
			},
		},
	}

	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			ddl, err := parseDDL(p.sql...)

			if err != nil {
				panic(err.Error())
			}

			tests.AssertEqual(t, p.sql[0], ddl.compile())
			if len(ddl.fields) != p.nFields {
				t.Fatalf("fields length doesn't match: expect: %v, got %v", p.nFields, len(ddl.fields))
			}
			tests.AssertEqual(t, ddl.columns, p.columns)
		})
	}
}

func TestParseDDL_error(t *testing.T) {
	params := []struct {
		name string
		sql  string
	}{
		{"invalid_cmd", "CREATE TABLE"},
		{"unbalanced_brackets", "CREATE TABLE test (ID int NOT NULL,Name varchar(255)"},
		{"unbalanced_brackets2", "CREATE TABLE test (ID int NOT NULL,Name varchar(255)))"},
	}

	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			_, err := parseDDL(p.sql)
			if err == nil {
				t.Fail()
			}
		})
	}
}

func TestAddConstraint(t *testing.T) {
	params := []struct {
		name   string
		fields []string
		cName  string
		sql    string
		expect []string
	}{
		{
			name:   "add_new",
			fields: []string{"`id` integer NOT NULL"},
			cName:  "fk_users_notes",
			sql:    "CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))",
			expect: []string{"`id` integer NOT NULL", "CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))"},
		},
		{
			name:   "update",
			fields: []string{"`id` integer NOT NULL", "CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))"},
			cName:  "fk_users_notes",
			sql:    "CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`)) ON UPDATE CASCADE ON DELETE CASCADE",
			expect: []string{"`id` integer NOT NULL", "CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`)) ON UPDATE CASCADE ON DELETE CASCADE"},
		},
		{
			name:   "add_check",
			fields: []string{"`id` integer NOT NULL"},
			cName:  "name_checker",
			sql:    "CONSTRAINT `name_checker` CHECK (`name` <> 'jinzhu')",
			expect: []string{"`id` integer NOT NULL", "CONSTRAINT `name_checker` CHECK (`name` <> 'jinzhu')"},
		},
		{
			name:   "update_check",
			fields: []string{"`id` integer NOT NULL", "CONSTRAINT `name_checker` CHECK (`name` <> 'thetadev')"},
			cName:  "name_checker",
			sql:    "CONSTRAINT `name_checker` CHECK (`name` <> 'jinzhu')",
			expect: []string{"`id` integer NOT NULL", "CONSTRAINT `name_checker` CHECK (`name` <> 'jinzhu')"},
		},
	}

	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			testDDL := ddl{fields: p.fields}

			testDDL.addConstraint(p.cName, p.sql)
			tests.AssertEqual(t, p.expect, testDDL.fields)
		})
	}
}

func TestRemoveConstraint(t *testing.T) {
	params := []struct {
		name    string
		fields  []string
		cName   string
		success bool
		expect  []string
	}{
		{
			name:    "fk",
			fields:  []string{"`id` integer NOT NULL", "CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))"},
			cName:   "fk_users_notes",
			success: true,
			expect:  []string{"`id` integer NOT NULL"},
		},
		{
			name:    "check",
			fields:  []string{"CONSTRAINT `name_checker` CHECK (`name` <> 'thetadev')", "`id` integer NOT NULL"},
			cName:   "name_checker",
			success: true,
			expect:  []string{"`id` integer NOT NULL"},
		},
		{
			name:    "none",
			fields:  []string{"CONSTRAINT `name_checker` CHECK (`name` <> 'thetadev')", "`id` integer NOT NULL"},
			cName:   "nothing",
			success: false,
			expect:  []string{"CONSTRAINT `name_checker` CHECK (`name` <> 'thetadev')", "`id` integer NOT NULL"},
		},
	}

	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			testDDL := ddl{fields: p.fields}

			success := testDDL.removeConstraint(p.cName)

			tests.AssertEqual(t, p.success, success)
			tests.AssertEqual(t, p.expect, testDDL.fields)
		})
	}
}

func TestGetColumns(t *testing.T) {
	params := []struct {
		name    string
		ddl     string
		columns []string
	}{
		{
			name:    "with_fk",
			ddl:     "CREATE TABLE `notes` (`id` integer NOT NULL,`text` varchar(500),`user_id` integer,PRIMARY KEY (`id`),CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))",
			columns: []string{"`id`", "`text`", "`user_id`"},
		},
		{
			name:    "with_check",
			ddl:     "CREATE TABLE Persons (ID int NOT NULL,LastName varchar(255) NOT NULL,FirstName varchar(255),Age int,CHECK (Age>=18),CHECK (FirstName!='John'))",
			columns: []string{"`ID`", "`LastName`", "`FirstName`", "`Age`"},
		},
		{
			name:    "with_escaped_quote",
			ddl:     "CREATE TABLE Persons (ID int NOT NULL,LastName varchar(255) NOT NULL DEFAULT \"\",FirstName varchar(255))",
			columns: []string{"`ID`", "`LastName`", "`FirstName`"},
		},
		{
			name:    "with_generated_column",
			ddl:     "CREATE TABLE Persons (ID int NOT NULL,LastName varchar(255) NOT NULL,FirstName varchar(255),FullName varchar(255) GENERATED ALWAYS AS (FirstName || ' ' || LastName))",
			columns: []string{"`ID`", "`LastName`", "`FirstName`"},
		},
		{
			name: "with_new_line",
			ddl: `CREATE TABLE "tb_sys_role_menu__temp" (
  "id" integer  PRIMARY KEY AUTOINCREMENT,
  "created_at" datetime NOT NULL,
  "updated_at" datetime NOT NULL,
  "created_by" integer NOT NULL DEFAULT 0,
  "updated_by" integer NOT NULL DEFAULT 0,
  "role_id" integer NOT NULL,
  "menu_id" bigint NOT NULL
)`,
			columns: []string{"`id`", "`created_at`", "`updated_at`", "`created_by`", "`updated_by`", "`role_id`", "`menu_id`"},
		},
	}

	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			testDDL, err := parseDDL(p.ddl)
			if err != nil {
				panic(err.Error())
			}

			cols := testDDL.getColumns()

			tests.AssertEqual(t, p.columns, cols)
		})
	}
}
