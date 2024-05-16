package db

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

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
	"fmt"
	"time"

	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	clientLog = log.NamedLogger("db", "client")
)

// newDBClient - Initialize the db client
func newDBClient() *gorm.DB {
	dbConfig := configs.GetDatabaseConfig()

	var dbClient *gorm.DB
	switch dbConfig.Dialect {
	case configs.Sqlite:
		dbClient = sqliteClient(dbConfig)
	case configs.Postgres:
		dbClient = postgresClient(dbConfig)
	case configs.MySQL:
		dbClient = mySQLClient(dbConfig)
	default:
		panic(fmt.Sprintf("Unknown DB Dialect: '%s'", dbConfig.Dialect))
	}

	// We cannot pass all of these into AutoMigrate at once because if one fails, subsequent models will not be created
	var allDBModels []interface{} = append(make([]interface{}, 0),
		&models.HttpC2Header{},
		&models.HttpC2ServerConfig{},
		&models.HttpC2ImplantConfig{},
		&models.HttpC2Config{},
		&models.HttpC2URLParameter{},
		&models.HttpC2PathSegment{},
		&models.Beacon{},
		&models.BeaconTask{},
		&models.DNSCanary{},
		&models.Crackstation{},
		&models.Benchmark{},
		&models.CrackTask{},
		&models.CrackCommand{},
		&models.CrackFile{},
		&models.CrackFileChunk{},
		&models.Certificate{},
		&models.Host{},
		&models.KeyValue{},
		&models.WGKeys{},
		&models.WGPeer{},
		&models.ResourceID{},
		&models.HttpC2Cookie{},
		&models.IOC{},
		&models.ExtensionData{},
		&models.ImplantProfile{},
		&models.ImplantConfig{},
		&models.ImplantBuild{},
		&models.ImplantC2{},
		&models.EncoderAsset{},
		&models.KeyExHistory{},
		&models.CanaryDomain{},
		&models.Loot{},
		&models.Credential{},
		&models.Operator{},
		&models.Website{},
		&models.WebContent{},
		&models.ListenerJob{},
		&models.HTTPListener{},
		&models.DNSListener{},
		&models.WGListener{},
		&models.MultiplayerListener{},
		&models.MtlsListener{},
		&models.DnsDomain{},
		&models.MonitoringProvider{},
	)

	var err error

	for _, model := range allDBModels {
		err = dbClient.AutoMigrate(model)
		if err != nil {
			clientLog.Error(err)
		}
	}

	// Get generic database object sql.DB to use its functions
	sqlDB, err := dbClient.DB()
	if err != nil {
		clientLog.Error(err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	return dbClient
}

func postgresClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(err)
	}
	dbClient, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      getGormLogger(dbConfig),
	})
	if err != nil {
		panic(err)
	}
	return dbClient
}

func mySQLClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(err)
	}
	dbClient, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      getGormLogger(dbConfig),
	})
	if err != nil {
		panic(err)
	}
	return dbClient
}
