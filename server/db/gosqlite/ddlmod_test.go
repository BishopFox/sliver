package gosqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDDL(t *testing.T) {
	params := []struct {
		name    string
		sql     string
		nFields int
	}{
		{"with_fk", "CREATE TABLE `notes` (`id` integer NOT NULL,`text` varchar(500),`user_id` integer,PRIMARY KEY (`id`),CONSTRAINT `fk_users_notes` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))", 5},
		{"with_check", "CREATE TABLE Persons (ID int NOT NULL,LastName varchar(255) NOT NULL,FirstName varchar(255),Age int,CHECK (Age>=18),CHECK (FirstName<>'John'))", 6},
		{"lowercase", "create table test (ID int NOT NULL)", 1},
		{"no brackets", "create table test", 0},
	}

	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			ddl, err := parseDDL(p.sql)

			if err != nil {
				panic(err.Error())
			}

			assert.Equal(t, p.sql, ddl.compile())
			assert.Len(t, ddl.fields, p.nFields)
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
			assert.Equal(t, p.expect, testDDL.fields)
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

			assert.Equal(t, p.success, success)
			assert.Equal(t, p.expect, testDDL.fields)
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
	}

	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			testDDL, err := parseDDL(p.ddl)
			if err != nil {
				panic(err.Error())
			}

			cols := testDDL.getColumns()

			assert.Equal(t, p.columns, cols)
		})
	}
}
