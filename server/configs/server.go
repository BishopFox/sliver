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
	"os"
	"path/filepath"

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
	serverConfigPath := filepath.Join(appDir, "configs", serverConfigFileName)
	serverConfigLog.Debugf("Loading config from %s", serverConfigPath)
	return serverConfigPath
}

// LogConfig - Server logging config
type LogConfig struct {
	Level              int  `json:"level"`
	GRPCUnaryPayloads  bool `json:"grpc_unary_payloads"`
	GRPCStreamPayloads bool `json:"grpc_stream_payloads"`
	TLSKeyLogger       bool `json:"tls_key_logger"`
}

// DaemonConfig - Configure daemon mode
type DaemonConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Tailscale bool   `json:"tailscale"`
}

// JobConfig - Restart Jobs on Load
type JobConfig struct {
	Multiplayer []*MultiplayerJobConfig `json:"multiplayer"`
	MTLS        []*MTLSJobConfig        `json:"mtls,omitempty"`
	WG          []*WGJobConfig          `json:"wg,omitempty"`
	DNS         []*DNSJobConfig         `json:"dns,omitempty"`
	HTTP        []*HTTPJobConfig        `json:"http,omitempty"`
}

type MultiplayerJobConfig struct {
	Host      string `json:"host"`
	Port      uint16 `json:"port"`
	JobID     string `json:"job_id"`
	Tailscale bool   `json:"tailscale"`
}

// MTLSJobConfig - Per-type job configs
type MTLSJobConfig struct {
	Host  string `json:"host"`
	Port  uint16 `json:"port"`
	JobID string `json:"job_id"`
}

// WGJobConfig - Per-type job configs
type WGJobConfig struct {
	Port    uint16 `json:"port"`
	NPort   uint16 `json:"nport"`
	KeyPort uint16 `json:"key_port"`
	JobID   string `json:"job_id"`
}

// DNSJobConfig - Persistent DNS job config
type DNSJobConfig struct {
	Domains    []string `json:"domains"`
	Canaries   bool     `json:"canaries"`
	Host       string   `json:"host"`
	Port       uint16   `json:"port"`
	JobID      string   `json:"job_id"`
	EnforceOTP bool     `json:"enforce_otp"`
}

// HTTPJobConfig - Persistent HTTP job config
type HTTPJobConfig struct {
	Domain          string `json:"domain"`
	Host            string `json:"host"`
	Port            uint16 `json:"port"`
	Secure          bool   `json:"secure"`
	Website         string `json:"website"`
	Cert            []byte `json:"cert"`
	Key             []byte `json:"key"`
	ACME            bool   `json:"acme"`
	JobID           string `json:"job_id"`
	EnforceOTP      bool   `json:"enforce_otp"`
	LongPollTimeout int64  `json:"long_poll_timeout"`
	LongPollJitter  int64  `json:"long_poll_jitter"`
	RandomizeJARM   bool   `json:"randomize_jarm"`
}

// WatchTowerConfig - Watch Tower job config
type WatchTowerConfig struct {
	VTApiKey          string `json:"vt_api_key"`
	XForceApiKey      string `json:"xforce_api_key"`
	XForceApiPassword string `json:"xforce_api_password"`
}

// ServerConfig - Server config
type ServerConfig struct {
	DaemonMode   bool              `json:"daemon_mode"`
	DaemonConfig *DaemonConfig     `json:"daemon"`
	Logs         *LogConfig        `json:"logs"`
	Watchtower   *WatchTowerConfig `json:"watch_tower"`
	GoProxy      string            `json:"go_proxy"`

	// 'GOOS/GOARCH' -> CC path
	CC  map[string]string `json:"cc"`
	CXX map[string]string `json:"cxx"`
}

// Save - Save config file to disk
func (c *ServerConfig) Save() error {
	configPath := GetServerConfigPath()
	configDir := filepath.Dir(configPath)
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
	err = os.WriteFile(configPath, data, 0600)
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
		data, err := os.ReadFile(configPath)
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

	if config.Logs.Level < 0 {
		config.Logs.Level = 0
	}
	if 6 < config.Logs.Level {
		config.Logs.Level = 6
	}
	log.RootLogger.SetLevel(log.LevelFrom(config.Logs.Level))

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
			Level:              int(logrus.InfoLevel),
			GRPCUnaryPayloads:  false,
			GRPCStreamPayloads: false,
		},
		CC:  map[string]string{},
		CXX: map[string]string{},
	}
}
