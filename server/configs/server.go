package configs

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"io/ioutil"
	"os"
	"path"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
	"github.com/sirupsen/logrus"
)

const (
	serverConfigFileName = "server.json"
)

var (
	serverConfigLog = log.NamedLogger("config", "server")
)

// GetServerConfigPath - File path to config.json
func GetServerConfigPath() string {
	appDir := assets.GetRootAppDir()
	serverConfigPath := path.Join(appDir, "configs", serverConfigFileName)
	serverConfigLog.Infof("Loading config from %s", serverConfigPath)
	return serverConfigPath
}

// LogConfig - Server logging config
type LogConfig struct {
	Level              int  `json:"level"`
	GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
	GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
}

// DaemonConfig - Configure daemon mode
type DaemonConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ServerConfig - Server config
type ServerConfig struct {
	DaemonMode   bool          `json:"daemon_mode"`
	DaemonConfig *DaemonConfig `json:"daemon"`
	Logs         *LogConfig    `json:"logs"`
}

// Save - Save config file to disk
func (c *ServerConfig) Save() error {
	configPath := GetServerConfigPath()
	configDir := path.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		serverConfigLog.Debugf("Creating config dir %s", configDir)
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	serverConfigLog.Infof("Saving config to %s", configPath)
	err = ioutil.WriteFile(configPath, data, 0600)
	if err != nil {
		serverConfigLog.Errorf("Failed to write config %s", err)
	}
	return nil
}

// GetServerConfig - Get config value
func GetServerConfig() *ServerConfig {
	configPath := GetServerConfigPath()
	config := getDefaultServerConfig()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			serverConfigLog.Errorf("Failed to read config file %s", err)
			return config
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			serverConfigLog.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else {
		serverConfigLog.Warnf("Config file does not exist, using defaults")
	}
	err := config.Save() // This updates the config with any missing fields
	if err != nil {
		serverConfigLog.Errorf("Failed to save default config %s", err)
	}
	return config
}

func getDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		DaemonMode: false,
		DaemonConfig: &DaemonConfig{
			Host: "",
			Port: 31337,
		},
		Logs: &LogConfig{
			Level:              int(logrus.DebugLevel),
			GRPCUnaryPayloads:  true,
			GRPCStreamPayloads: true,
		},
	}
}
