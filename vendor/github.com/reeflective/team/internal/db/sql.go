package db

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"errors"
	"fmt"
	"time"

	"github.com/reeflective/team/internal/log"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	// SQLiteInMemoryHost is the default string used by SQLite
	// as a host when ran in memory (string value is ":memory:").
	SQLiteInMemoryHost = ":memory:"
)

var (
	// ErrRecordNotFound - Record not found error.
	ErrRecordNotFound = gorm.ErrRecordNotFound

	// ErrUnsupportedDialect - An invalid dialect was specified.
	ErrUnsupportedDialect = errors.New("Unknown/unsupported DB Dialect")
)

// NewClient initializes a database client connection to a backend specified in config.
func NewClient(dbConfig *Config, dbLogger *logrus.Entry) (*gorm.DB, error) {
	var dbClient *gorm.DB

	dsn, err := dbConfig.DSN()
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal database DSN: %w", err)
	}

	// Logging middleware (queries)
	dbLog := log.NewDatabase(dbLogger, dbConfig.LogLevel)
	logDbDsn := fmt.Sprintf("%s (%s:%d)", dbConfig.Database, dbConfig.Host, dbConfig.Port)

	switch dbConfig.Dialect {
	case Sqlite:
		dbLogger.Infof("Connecting to SQLite database %s", logDbDsn)

		dbClient, err = sqliteClient(dsn, dbLog)
		if err != nil {
			return nil, fmt.Errorf("Database connection failed: %w", err)
		}

	case Postgres:
		dbLogger.Infof("Connecting to PostgreSQL database %s", logDbDsn)

		dbClient, err = postgresClient(dsn, dbLog)
		if err != nil {
			return nil, fmt.Errorf("Database connection failed: %w", err)
		}

	case MySQL:
		dbLogger.Infof("Connecting to MySQL database %s", logDbDsn)

		dbClient, err = mySQLClient(dsn, dbLog)
		if err != nil {
			return nil, fmt.Errorf("Database connection failed: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w: '%s'", ErrUnsupportedDialect, dbConfig.Dialect)
	}

	err = dbClient.AutoMigrate(Schema()...)
	if err != nil {
		dbLogger.Error(err)
	}

	// Get generic database object sql.DB to use its functions
	sqlDB, err := dbClient.DB()
	if err != nil {
		dbLogger.Error(err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	return dbClient, nil
}

// Schema returns all objects which should be registered
// to the teamserver database backend.
func Schema() []any {
	return []any{
		&Certificate{},
		&User{},
	}
}

func postgresClient(dsn string, log logger.Interface) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      log,
	})
}

func mySQLClient(dsn string, log logger.Interface) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      log,
	})
}
