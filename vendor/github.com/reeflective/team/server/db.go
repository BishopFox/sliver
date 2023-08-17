package server

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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/internal/db"
	"gorm.io/gorm"
)

const (
	maxIdleConns = 10
	maxOpenConns = 100
)

// Database returns a new teamserver database session, which may not be nil:
// if no custom database backend was passed to the server at creation time,
// this database will be an in-memory one. The default is a file-based Sqlite
// database in the teamserver directory, but it might be a specific database
// passed through options.
func (ts *Server) Database() *gorm.DB {
	return ts.db.Session(&gorm.Session{
		FullSaveAssociations: true,
	})
}

// DatabaseConfig returns the server database backend configuration struct.
// If no configuration could be found on disk, the default Sqlite file-based
// database is returned, with app-corresponding file paths.
func (ts *Server) DatabaseConfig() *db.Config {
	cfg, err := ts.getDatabaseConfig()
	if err != nil {
		return cfg
	}

	return cfg
}

// GetDatabaseConfigPath - File path to config.json.
func (ts *Server) dbConfigPath() string {
	appDir := ts.ConfigsDir()
	log := ts.NamedLogger("config", "database")
	dbFileName := fmt.Sprintf("%s.%s", ts.Name()+"_database", command.ServerConfigExt)
	databaseConfigPath := filepath.Join(appDir, dbFileName)
	log.Debugf("Loading config from %s", databaseConfigPath)

	return databaseConfigPath
}

// Save - Save config file to disk. If the server is configured
// to run in-memory only, the config is not saved.
func (ts *Server) saveDatabaseConfig(cfg *db.Config) error {
	if ts.opts.inMemory {
		return nil
	}

	dblog := ts.NamedLogger("config", "database")

	configPath := ts.dbConfigPath()
	configDir := path.Dir(configPath)

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		dblog.Debugf("Creating config dir %s", configDir)

		err := os.MkdirAll(configDir, assets.DirPerm)
		if err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}

	dblog.Debugf("Saving config to %s", configPath)

	return os.WriteFile(configPath, data, assets.FileReadPerm)
}

// getDatabaseConfig returns a working database configuration,
// either fetched from the file system, adjusted with in-code
// options, or a default one.
// If an error happens, it is returned with a nil configuration.
func (ts *Server) getDatabaseConfig() (*db.Config, error) {
	log := ts.NamedLogger("config", "database")

	// Don't fetch anything if running in-memory only.
	config := ts.opts.dbConfig
	if config.Database == db.SQLiteInMemoryHost {
		return config, nil
	}

	configPath := ts.dbConfigPath()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read config file %w", err)
		}

		err = json.Unmarshal(data, config)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse config file %w", err)
		}
	} else {
		log.Warnf("Database: no config file found, using and saving defaults")
	}

	if config.MaxIdleConns < 1 {
		config.MaxIdleConns = 1
	}

	if config.MaxOpenConns < 1 {
		config.MaxOpenConns = 1
	}

	// This updates the config with any missing fields,
	// failing to save is not critical for operation.
	err := ts.saveDatabaseConfig(config)
	if err != nil {
		log.Errorf("Failed to save default config %s", err)
	}

	return config, nil
}

func (ts *Server) getDefaultDatabaseConfig() *db.Config {
	cfg := &db.Config{
		Dialect:      db.Sqlite,
		MaxIdleConns: maxIdleConns,
		MaxOpenConns: maxOpenConns,

		LogLevel: "warn",
	}

	if ts.opts.inMemory {
		cfg.Database = db.SQLiteInMemoryHost
	} else {
		cfg.Database = filepath.Join(ts.TeamDir(), fmt.Sprintf("%s.teamserver.db", ts.name))
	}

	return cfg
}

// initDatabase should be called once when a teamserver is created.
func (ts *Server) initDatabase() (err error) {
	ts.dbInit.Do(func() {
		dbLogger := ts.NamedLogger("database", "database")

		if ts.db != nil {
			err = ts.db.AutoMigrate(db.Schema()...)
			return
		}

		ts.opts.dbConfig, err = ts.getDatabaseConfig()
		if err != nil {
			return
		}

		ts.db, err = db.NewClient(ts.opts.dbConfig, dbLogger)
		if err != nil {
			return
		}
	})

	return err
}

func (ts *Server) dbSession() *gorm.DB {
	return ts.db.Session(&gorm.Session{
		FullSaveAssociations: true,
	})
}
