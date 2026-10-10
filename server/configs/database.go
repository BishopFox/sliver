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
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
	"gopkg.in/yaml.v3"
)

const (
	// Sqlite - SQLite protocol
	Sqlite = "sqlite3"
	// Postgres - Postgresql protocol
	Postgres = "postgresql"
	// MySQL - MySQL protocol
	MySQL = "mysql"

	databaseConfigFileName       = "database.yaml"
	databaseLegacyConfigFileName = "database.json"
)

var (
	// ErrInvalidDialect - An invalid dialect was specified
	ErrInvalidDialect = errors.New("invalid SQL Dialect")

	databaseConfigLog = log.NamedLogger("config", "database")

	defaultSQLitePragmas = map[string]string{
		"journal_mode": "WAL",    // reduce writer blocking, better concurrency
		"busy_timeout": "5000",   // wait for locks instead of failing fast (ms)
		"synchronous":  "NORMAL", // faster WAL syncs while retaining durability
		"temp_store":   "MEMORY", // keep temp structures off disk for quicker startup
	}
)

// GetDatabaseConfigPath - File path to config.yaml
func GetDatabaseConfigPath() string {
	appDir := assets.GetRootAppDir()
	databaseConfigPath := filepath.Join(appDir, "configs", databaseConfigFileName)
	databaseConfigLog.Debugf("Loading config from %s", databaseConfigPath)
	return databaseConfigPath
}

func getDatabaseLegacyConfigPath() string {
	appDir := assets.GetRootAppDir()
	return filepath.Join(appDir, "configs", databaseLegacyConfigFileName)
}

// DatabaseConfig - Server config
type DatabaseConfig struct {
	Dialect  string `json:"dialect" yaml:"dialect"`
	Database string `json:"database" yaml:"database"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Host     string `json:"host" yaml:"host"`
	Port     uint16 `json:"port" yaml:"port"`

	Params  map[string]string `json:"params" yaml:"params"`
	Pragmas map[string]string `json:"pragmas" yaml:"pragmas"`

	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns"`

	LogLevel string `json:"log_level" yaml:"log_level"`
}

// DSN - Get the db connections string
// https://github.com/go-sql-driver/mysql#examples
func (c *DatabaseConfig) DSN() (string, error) {
	switch c.Dialect {
	case Sqlite:
		filePath := filepath.Join(assets.GetRootAppDir(), "sliver.db")
		params := encodeSQLiteParams(c.Params, c.Pragmas)
		return fmt.Sprintf("file:%s?%s", filePath, params), nil
	case MySQL:
		if strings.Contains(c.Username, ":") {
			return "", errors.New("MySQL DSN cannot represent a colon in the username")
		}
		host := net.JoinHostPort(c.Host, strconv.Itoa(int(c.Port)))
		params := encodeParams(c.Params)
		databaseConfigLog.Infof("Connecting to MySQL database %q at %q", c.Database, host)
		return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", c.Username, c.Password, host, url.PathEscape(c.Database), params), nil
	case Postgres:
		parts := []string{
			"host=" + quotePostgresDSNValue(c.Host),
			"port=" + strconv.Itoa(int(c.Port)),
			"user=" + quotePostgresDSNValue(c.Username),
			"password=" + quotePostgresDSNValue(c.Password),
			"dbname=" + quotePostgresDSNValue(c.Database),
		}
		keys := make([]string, 0, len(c.Params))
		for key := range c.Params {
			if key == "" || strings.ContainsAny(key, "= \t\n\r\v\f'\\\x00") {
				return "", fmt.Errorf("invalid PostgreSQL parameter key %q", key)
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			parts = append(parts, key+"="+quotePostgresDSNValue(c.Params[key]))
		}
		databaseConfigLog.Infof("Connecting to Postgres database %q at %q:%d", c.Database, c.Host, c.Port)
		return strings.Join(parts, " "), nil
	default:
		return "", ErrInvalidDialect
	}
}

func quotePostgresDSNValue(value string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `'`, `\'`) + "'"
}

func encodeParams(rawParams map[string]string) string {
	params := url.Values{}
	for key, value := range rawParams {
		params.Add(key, value)
	}
	return params.Encode()
}

func encodeSQLiteParams(rawParams map[string]string, pragmas map[string]string) string {
	if rawParams == nil {
		rawParams = map[string]string{}
	}
	params := url.Values{}

	// Preserve any user-provided parameters first.
	for key, value := range rawParams {
		params.Add(key, value)
	}

	// Apply safer defaults only when the user has not set any custom pragma.
	if _, ok := rawParams["_pragma"]; !ok {
		selectedPragmas := pragmas
		if len(selectedPragmas) == 0 {
			selectedPragmas = defaultSQLitePragmas
		}

		keys := make([]string, 0, len(selectedPragmas))
		for key := range selectedPragmas {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := selectedPragmas[key]
			params.Add("_pragma", fmt.Sprintf("%s(%s)", key, value))
		}
	}

	// Encourage shared cache to play nicer with WAL and multiple connections.
	if _, ok := rawParams["cache"]; !ok {
		params.Add("cache", "shared")
	}

	return params.Encode()
}

// Save - Save config file to disk
func (c *DatabaseConfig) Save() error {
	configPath := GetDatabaseConfigPath()
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	databaseConfigLog.Infof("Saving config to %s", configPath)
	err = atomicWriteConfig(configPath, data)
	if err != nil {
		databaseConfigLog.Errorf("Failed to write config %s: %s", configPath, err)
	}
	return err
}

// GetDatabaseConfig - Get config value
func GetDatabaseConfig() *DatabaseConfig {
	configPath := GetDatabaseConfigPath()
	legacyPath := getDatabaseLegacyConfigPath()
	config := getDefaultDatabaseConfig()
	migratedLegacy := false
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			databaseConfigLog.Errorf("Failed to read config file %s", err)
			return config
		}
		err = yaml.Unmarshal(data, config)
		if err != nil {
			databaseConfigLog.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(legacyPath)
		if err != nil {
			databaseConfigLog.Errorf("Failed to read legacy config file %s", err)
			return config
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			databaseConfigLog.Errorf("Failed to parse legacy config file %s", err)
			return config
		}
		migratedLegacy = true
		databaseConfigLog.Infof("Migrating legacy config %s to %s", legacyPath, configPath)
	} else {
		databaseConfigLog.Warnf("Config file does not exist, using defaults")
	}

	if config.MaxIdleConns < 1 {
		config.MaxIdleConns = 1
	}
	if config.MaxOpenConns < 1 {
		config.MaxOpenConns = 1
	}

	ensureSQLiteDefaults(config)

	err := config.Save() // This updates the config with any missing fields
	if err != nil {
		databaseConfigLog.Errorf("Failed to save default config %s", err)
		return config
	}
	if migratedLegacy {
		if err := renameLegacyConfig(legacyPath); err != nil {
			databaseConfigLog.Errorf("Failed to rename legacy config %s", err)
		}
	}
	return config
}

func getDefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Dialect:      Sqlite,
		Pragmas:      defaultSQLitePragmas,
		MaxIdleConns: 10,
		MaxOpenConns: 100,

		LogLevel: "warn",
	}
}

func ensureSQLiteDefaults(c *DatabaseConfig) {
	if c.Dialect != Sqlite {
		return
	}
	if c.Params == nil {
		c.Params = map[string]string{}
	}
	if _, ok := c.Params["_pragma"]; ok {
		// User provided explicit pragmas; leave Pragmas untouched to avoid confusion.
		return
	}
	if len(c.Pragmas) == 0 {
		c.Pragmas = defaultSQLitePragmas
	}
	if _, ok := c.Params["cache"]; !ok {
		c.Params["cache"] = "shared"
	}
}
