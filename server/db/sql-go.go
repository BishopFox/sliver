//go:build go_sqlite

package db

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"github.com/bishopfox/sliver/server/configs"
	gosqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func sqliteClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(err)
	}
	clientLog.Infof("sqlite -> %s", dsn)

	dbClient, err := gorm.Open(gosqlite.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      getGormLogger(dbConfig),
	})
	if err != nil {
		panic(err)
	}
	return dbClient
}
