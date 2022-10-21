package configs

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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
)

const (
	// Sqlite - SQLite protocol
	Sqlite = "sqlite3"
	// Postgres - Postgresql protocol
	Postgres = "postgresql"
	// MySQL - MySQL protocol
	MySQL = "mysql"

	databaseConfigFileName = "database.json"
)

var (
	// ErrInvalidDialect - An invalid dialect was specified
	ErrInvalidDialect = errors.New("invalid SQL Dialect")

	databaseConfigLog = log.NamedLogger("config", "database")
)

// GetDatabaseConfigPath - File path to config.json
func GetDatabaseConfigPath() string {
	appDir := assets.GetRootAppDir()
	databaseConfigPath := filepath.Join(appDir, "configs", databaseConfigFileName)
	databaseConfigLog.Debugf("Loading config from %s", databaseConfigPath)
	return databaseConfigPath
}

// DatabaseConfig - Server config
type DatabaseConfig struct {
	Dialect  string `json:"dialect"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`

	Params map[string]string `json:"params"`

	MaxIdleConns int `json:"max_idle_conns"`
	MaxOpenConns int `json:"max_open_conns"`

	LogLevel string `json:"log_level"`
}

// DSN - Get the db connections string
// https://github.com/go-sql-driver/mysql#examples
func (c *DatabaseConfig) DSN() (string, error) {
	switch c.Dialect {
	case Sqlite:
		filePath := filepath.Join(assets.GetRootAppDir(), "sliver.db")
		params := encodeParams(c.Params)
		return fmt.Sprintf("file:%s?%s", filePath, params), nil
	case MySQL:
		user := url.QueryEscape(c.Username)
		password := url.QueryEscape(c.Password)
		db := url.QueryEscape(c.Database)
		host := fmt.Sprintf("%s:%d", url.QueryEscape(c.Host), c.Port)
		params := encodeParams(c.Params)
		databaseConfigLog.Infof("Connecting to MySQL database %s@%s/%s", user, host, db)
		return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", user, password, host, db, params), nil
	case Postgres:
		user := url.QueryEscape(c.Username)
		password := url.QueryEscape(c.Password)
		db := url.QueryEscape(c.Database)
		host := url.QueryEscape(c.Host)
		port := c.Port
		params := encodeParams(c.Params)
		databaseConfigLog.Infof("Connecting to Postgres database %s@%s:%d/%s", user, host, port, db)
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s %s", host, port, user, password, db, params), nil
	default:
		return "", ErrInvalidDialect
	}
}

func encodeParams(rawParams map[string]string) string {
	params := url.Values{}
	for key, value := range rawParams {
		params.Add(key, value)
	}
	return params.Encode()
}

// Save - Save config file to disk
func (c *DatabaseConfig) Save() error {
	configPath := GetDatabaseConfigPath()
	configDir := path.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		databaseConfigLog.Debugf("Creating config dir %s", configDir)
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	databaseConfigLog.Infof("Saving config to %s", configPath)
	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		databaseConfigLog.Errorf("Failed to write config %s", err)
	}
	return nil
}

// GetDatabaseConfig - Get config value
func GetDatabaseConfig() *DatabaseConfig {
	configPath := GetDatabaseConfigPath()
	config := getDefaultDatabaseConfig()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			databaseConfigLog.Errorf("Failed to read config file %s", err)
			return config
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			databaseConfigLog.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else {
		databaseConfigLog.Warnf("Config file does not exist, using defaults")
	}

	if config.MaxIdleConns < 1 {
		config.MaxIdleConns = 1
	}
	if config.MaxOpenConns < 1 {
		config.MaxOpenConns = 1
	}

	err := config.Save() // This updates the config with any missing fields
	if err != nil {
		databaseConfigLog.Errorf("Failed to save default config %s", err)
	}
	return config
}

func getDefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Dialect:      Sqlite,
		MaxIdleConns: 10,
		MaxOpenConns: 100,

		LogLevel: "warn",
	}
}
