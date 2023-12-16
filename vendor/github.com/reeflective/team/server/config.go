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
	"encoding/hex"
	"encoding/json"
	"fmt"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/command"
	"github.com/sirupsen/logrus"
)

const (
	blankHost   = "-"
	blankPort   = uint16(0)
	tokenLength = 32
	defaultPort = 31416 // Should be 31415, but... go to hell with limits.
)

// Config represents the configuration of a given application teamserver.
// It contains anonymous embedded structs as subsections, for logging,
// daemon mode bind addresses, and persistent teamserver listeners
//
// Its default path is ~/.app/teamserver/configs/app.teamserver.cfg.
// It uses the following default values:
//   - Daemon host: ""
//   - Daemon port: 31416
//   - logging file level: Info.
type Config struct {
	// When the teamserver command `app teamserver daemon` is executed
	// without --host/--port flags, the teamserver will use the config.
	DaemonMode struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"daemon_mode"`

	// Logging controls the file-based logging level, whether or not
	// to log TLS keys to file, and whether to log specific gRPC payloads.
	Log struct {
		Level              int  `json:"level"`
		GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
		GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
		TLSKeyLogger       bool `json:"tls_key_logger"`
	} `json:"log"`

	// Listeners is a list of persistent teamserver listeners.
	// They are started when the teamserver daemon command/mode is.
	Listeners []struct {
		Name string `json:"name"`
		Host string `json:"host"`
		Port uint16 `json:"port"`
		ID   string `json:"id"`
	} `json:"listeners"`
}

// ConfigPath returns the path to the server config.json file, on disk or in-memory.
func (ts *Server) ConfigPath() string {
	appDir := ts.ConfigsDir()

	err := ts.fs.MkdirAll(appDir, assets.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s config dir: %s", appDir, err)
	}

	serverConfigPath := filepath.Join(appDir, fmt.Sprintf("%s.%s", ts.Name(), command.ServerConfigExt))

	return serverConfigPath
}

// GetConfig returns the team server configuration as a struct.
// If no server configuration file is found on disk, the default one is used.
func (ts *Server) GetConfig() *Config {
	cfgLog := ts.NamedLogger("config", "server")

	if ts.opts.inMemory {
		return ts.opts.config
	}

	configPath := ts.ConfigPath()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		cfgLog.Debugf("Loading config from %s", configPath)

		data, err := os.ReadFile(configPath)
		if err != nil {
			cfgLog.Errorf("Failed to read config file %s", err)
			return ts.opts.config
		}

		err = json.Unmarshal(data, ts.opts.config)
		if err != nil {
			cfgLog.Errorf("Failed to parse config file %s", err)
			return ts.opts.config
		}
	} else {
		cfgLog.Warnf("Teamserver: no config file found, using and saving defaults")
	}

	if ts.opts.config.Log.Level < 0 {
		ts.opts.config.Log.Level = 0
	}

	if int(logrus.TraceLevel) < ts.opts.config.Log.Level {
		ts.opts.config.Log.Level = int(logrus.TraceLevel)
	}

	// This updates the config with any missing fields
	err := ts.SaveConfig(ts.opts.config)
	if err != nil {
		cfgLog.Errorf("Failed to save default config %s", err)
	}

	return ts.opts.config
}

// SaveConfig saves config file to disk.
// This uses the on-disk filesystem even if the teamclient is in memory mode.
func (ts *Server) SaveConfig(cfg *Config) error {
	cfgLog := ts.NamedLogger("config", "server")

	if ts.opts.inMemory {
		return nil
	}

	configPath := ts.ConfigPath()
	configDir := filepath.Dir(configPath)

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		cfgLog.Debugf("Creating config dir %s", configDir)

		err := os.MkdirAll(configDir, assets.DirPerm)
		if err != nil {
			return ts.errorf("%w: %w", ErrConfig, err)
		}
	}

	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}

	cfgLog.Debugf("Saving config to %s", configPath)

	err = os.WriteFile(configPath, data, assets.FileReadPerm)
	if err != nil {
		return ts.errorf("%w: failed to write config: %s", ErrConfig, err)
	}

	return nil
}

func getDefaultServerConfig() *Config {
	return &Config{
		DaemonMode: struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}{
			Port: defaultPort, // 31416
		},
		Log: struct {
			Level              int  `json:"level"`
			GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
			GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
			TLSKeyLogger       bool `json:"tls_key_logger"`
		}{
			Level: int(logrus.InfoLevel),
		},
		Listeners: []struct {
			Name string `json:"name"`
			Host string `json:"host"`
			Port uint16 `json:"port"`
			ID   string `json:"id"`
		}{},
	}
}

func getRandomID() string {
	seededRand := insecureRand.New(insecureRand.NewSource(time.Now().UnixNano()))
	buf := make([]byte, tokenLength)
	seededRand.Read(buf)

	return hex.EncodeToString(buf)
}
