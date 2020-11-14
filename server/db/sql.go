package db

import (
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/log"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	// Always include SQLite
	_ "github.com/mattn/go-sqlite3"
)

var (
	sqlLog = log.NamedLogger("db", "sql")
)

// newDBClient - Initialize the db client
func newDBClient() *gorm.DB {
	dbConfig := configs.GetDatabaseConfig()
	switch dbConfig.Dialect {
	case configs.Sqlite:
		return sqliteClient(dbConfig)
	case configs.Postgres:
		return postgresClient(dbConfig)
	}
	panic("Invalid database configuration")
}

func sqliteClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(err)
	}
	sqlLog.Infof("sqlite -> %s", dsn)
	dbClient, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panic(err)
	}
	return dbClient
}

func postgresClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(err)
	}
	sqlLog.Infof("postgres -> %s", dsn)
	dbClient, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return dbClient
}
