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
	"gopkg.in/yaml.v3"
)

const (
	serverConfigFileName                     = "server.yaml"
	serverLegacyConfigFileName               = "server.json"
	defaultGRPCKeepaliveMinTimeSeconds int64 = 30
)

var (
	serverConfigLog = log.NamedLogger("config", "server")
)

// GetServerConfigPath - File path to config.yaml
func GetServerConfigPath() string {
	appDir := assets.GetRootAppDir()
	serverConfigPath := filepath.Join(appDir, "configs", serverConfigFileName)
	serverConfigLog.Debugf("Loading config from %s", serverConfigPath)
	return serverConfigPath
}

func getServerLegacyConfigPath() string {
	appDir := assets.GetRootAppDir()
	return filepath.Join(appDir, "configs", serverLegacyConfigFileName)
}

// LogConfig - Server logging config
type LogConfig struct {
	Level              int  `json:"level" yaml:"level"`
	GRPCUnaryPayloads  bool `json:"grpc_unary_payloads" yaml:"grpc_unary_payloads"`
	GRPCStreamPayloads bool `json:"grpc_stream_payloads" yaml:"grpc_stream_payloads"`
	TLSKeyLogger       bool `json:"tls_key_logger" yaml:"tls_key_logger"`
}

// GRPCKeepaliveConfig - gRPC keepalive enforcement settings
type GRPCKeepaliveConfig struct {
	MinTimeSeconds      int64 `json:"min_time_seconds" yaml:"min_time_seconds"`
	PermitWithoutStream *bool `json:"permit_without_stream" yaml:"permit_without_stream"`
}

// GRPCConfig - gRPC server settings
type GRPCConfig struct {
	Keepalive *GRPCKeepaliveConfig `json:"keepalive" yaml:"keepalive"`
}

// DaemonConfig - Configure daemon mode
type DaemonConfig struct {
	Host      string `json:"host" yaml:"host"`
	Port      int    `json:"port" yaml:"port"`
	Tailscale bool   `json:"tailscale" yaml:"tailscale"`
}

// JobConfig - Restart Jobs on Load
type JobConfig struct {
	Multiplayer []*MultiplayerJobConfig `json:"multiplayer" yaml:"multiplayer"`
	MTLS        []*MTLSJobConfig        `json:"mtls,omitempty" yaml:"mtls,omitempty"`
	WG          []*WGJobConfig          `json:"wg,omitempty" yaml:"wg,omitempty"`
	DNS         []*DNSJobConfig         `json:"dns,omitempty" yaml:"dns,omitempty"`
	HTTP        []*HTTPJobConfig        `json:"http,omitempty" yaml:"http,omitempty"`
}

type MultiplayerJobConfig struct {
	Host      string `json:"host" yaml:"host"`
	Port      uint16 `json:"port" yaml:"port"`
	JobID     string `json:"job_id" yaml:"job_id"`
	Tailscale bool   `json:"tailscale" yaml:"tailscale"`
}

// MTLSJobConfig - Per-type job configs
type MTLSJobConfig struct {
	Host  string `json:"host" yaml:"host"`
	Port  uint16 `json:"port" yaml:"port"`
	JobID string `json:"job_id" yaml:"job_id"`
}

// WGJobConfig - Per-type job configs
type WGJobConfig struct {
	Port    uint16 `json:"port" yaml:"port"`
	NPort   uint16 `json:"nport" yaml:"nport"`
	KeyPort uint16 `json:"key_port" yaml:"key_port"`
	JobID   string `json:"job_id" yaml:"job_id"`
}

// DNSJobConfig - Persistent DNS job config
type DNSJobConfig struct {
	Domains    []string `json:"domains" yaml:"domains"`
	Canaries   bool     `json:"canaries" yaml:"canaries"`
	Host       string   `json:"host" yaml:"host"`
	Port       uint16   `json:"port" yaml:"port"`
	JobID      string   `json:"job_id" yaml:"job_id"`
	EnforceOTP bool     `json:"enforce_otp" yaml:"enforce_otp"`
}

// HTTPJobConfig - Persistent HTTP job config
type HTTPJobConfig struct {
	Domain          string `json:"domain" yaml:"domain"`
	Host            string `json:"host" yaml:"host"`
	Port            uint16 `json:"port" yaml:"port"`
	Secure          bool   `json:"secure" yaml:"secure"`
	Website         string `json:"website" yaml:"website"`
	Cert            []byte `json:"cert" yaml:"cert"`
	Key             []byte `json:"key" yaml:"key"`
	ACME            bool   `json:"acme" yaml:"acme"`
	JobID           string `json:"job_id" yaml:"job_id"`
	EnforceOTP      bool   `json:"enforce_otp" yaml:"enforce_otp"`
	LongPollTimeout int64  `json:"long_poll_timeout" yaml:"long_poll_timeout"`
	LongPollJitter  int64  `json:"long_poll_jitter" yaml:"long_poll_jitter"`
	RandomizeJARM   bool   `json:"randomize_jarm" yaml:"randomize_jarm"`
}

// WatchTowerConfig - Watch Tower job config
type WatchTowerConfig struct {
	VTApiKey          string `json:"vt_api_key" yaml:"vt_api_key"`
	XForceApiKey      string `json:"xforce_api_key" yaml:"xforce_api_key"`
	XForceApiPassword string `json:"xforce_api_password" yaml:"xforce_api_password"`
}

// http server defaults for anonymous requests
type HttpDefaultHeader struct {
	Method      string `json:"method" yaml:"method"`
	Name        string `json:"name" yaml:"name"`
	Value       string `json:"value" yaml:"value"`
	Probability int32  `json:"probability" yaml:"probability"`
}

type httpDefaultHeaderLegacy struct {
	Method                string  `json:"method" yaml:"method"`
	Name                  string  `json:"name" yaml:"name"`
	Value                 string  `json:"value" yaml:"value"`
	Probability           int32   `json:"probability" yaml:"probability"`
	ID                    string  `json:"id" yaml:"id"`
	HttpC2ServerConfigID  *string `json:"httpc2serverconfigid" yaml:"httpc2serverconfigid"`
	HttpC2ImplantConfigID *string `json:"httpc2implantconfigid" yaml:"httpc2implantconfigid"`
}

func (h *HttpDefaultHeader) UnmarshalJSON(data []byte) error {
	var raw httpDefaultHeaderLegacy
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	h.Method = raw.Method
	h.Name = raw.Name
	h.Value = raw.Value
	h.Probability = raw.Probability
	return nil
}

func (h *HttpDefaultHeader) UnmarshalYAML(node *yaml.Node) error {
	var raw httpDefaultHeaderLegacy
	if err := node.Decode(&raw); err != nil {
		return err
	}
	h.Method = raw.Method
	h.Name = raw.Name
	h.Value = raw.Value
	h.Probability = raw.Probability
	return nil
}

type HttpDefaultConfig struct {
	Headers []HttpDefaultHeader `json:"headers" yaml:"headers"`
}

// ServerConfig - Server config
type ServerConfig struct {
	DaemonMode    bool                 `json:"daemon_mode" yaml:"daemon_mode"`
	DaemonConfig  *DaemonConfig        `json:"daemon" yaml:"daemon"`
	Logs          *LogConfig           `json:"logs" yaml:"logs"`
	Watchtower    *WatchTowerConfig    `json:"watch_tower" yaml:"watch_tower"`
	GoProxy       string               `json:"go_proxy" yaml:"go_proxy"`
	HTTPDefaults  *HttpDefaultConfig   `json:"http_default" yaml:"http_default"`
	DonutBypass   int                  `json:"donut_bypass" yaml:"donut_bypass"` // 1=skip, 2=abort on fail, 3=continue on fail.
	Notifications *NotificationsConfig `json:"notifications" yaml:"notifications"`

	// 'GOOS/GOARCH' -> CC path
	CC  map[string]string `json:"cc" yaml:"cc"`
	CXX map[string]string `json:"cxx" yaml:"cxx"`
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
	data, err := yaml.Marshal(c)
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
	legacyPath := getServerLegacyConfigPath()
	config := getDefaultServerConfig()
	migratedLegacy := false
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			serverConfigLog.Errorf("Failed to read config file %s", err)
			return config
		}
		err = yaml.Unmarshal(data, config)
		if err != nil {
			serverConfigLog.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(legacyPath)
		if err != nil {
			serverConfigLog.Errorf("Failed to read legacy config file %s", err)
			return config
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			serverConfigLog.Errorf("Failed to parse legacy config file %s", err)
			return config
		}
		migratedLegacy = true
		serverConfigLog.Infof("Migrating legacy config %s to %s", legacyPath, configPath)
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

	if config.DonutBypass < 1 || config.DonutBypass > 3 {
		config.DonutBypass = 1
	}

	if config.GRPC == nil {
		config.GRPC = &GRPCConfig{}
	}
	if config.GRPC.Keepalive == nil {
		config.GRPC.Keepalive = &GRPCKeepaliveConfig{}
	}
	if config.GRPC.Keepalive.MinTimeSeconds <= 0 {
		config.GRPC.Keepalive.MinTimeSeconds = defaultGRPCKeepaliveMinTimeSeconds
	}
	if config.GRPC.Keepalive.PermitWithoutStream == nil {
		defaultPermit := true
		config.GRPC.Keepalive.PermitWithoutStream = &defaultPermit
	}

	err := config.Save() // This updates the config with any missing fields
	if err != nil {
		serverConfigLog.Errorf("Failed to save default config %s", err)
		return config
	}
	if migratedLegacy {
		if err := renameLegacyConfig(legacyPath); err != nil {
			serverConfigLog.Errorf("Failed to rename legacy config %s", err)
		}
	}
	return config
}

func getDefaultServerConfig() *ServerConfig {
	defaultPermitWithoutStream := true
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
		GRPC: &GRPCConfig{
			Keepalive: &GRPCKeepaliveConfig{
				MinTimeSeconds:      defaultGRPCKeepaliveMinTimeSeconds,
				PermitWithoutStream: &defaultPermitWithoutStream,
			},
		},
		HTTPDefaults: &HttpDefaultConfig{
			Headers: []HttpDefaultHeader{
				{
					Method:      "GET",
					Name:        "Cache-Control",
					Value:       "no-store, no-cache, must-revalidate",
					Probability: 100,
				},
			},
		},
		DonutBypass:   3,
		Notifications: defaultNotificationsConfig(),
		CC:            map[string]string{},
		CXX:           map[string]string{},
	}
}
