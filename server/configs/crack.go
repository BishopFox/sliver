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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
)

const (
	crackConfigFileName = "crack.json"
)

var (
	crackConfigLog = log.NamedLogger("config", "crack")
)

// getCrackConfigPath - File path to config.json
func getCrackConfigPath() string {
	appDir := assets.GetRootAppDir()
	confPath := filepath.Join(appDir, "configs", crackConfigFileName)
	crackConfigLog.Debugf("Loading crack config from %s", confPath)
	return confPath
}

// Save - Save config file to disk
func SaveCrackConfig(config *clientpb.CrackConfig) error {
	configPath := getCrackConfigPath()
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		crackConfigLog.Debugf("Creating config dir %s", configDir)
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	crackConfigLog.Infof("Saving crack config to %s", configPath)
	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		crackConfigLog.Errorf("Failed to write config %s", err)
	}
	return nil
}

// LoadCrackConfig - Get config value
func LoadCrackConfig() (*clientpb.CrackConfig, error) {
	configPath := getCrackConfigPath()
	const defaultMaxFileSize = 10 * 1024 * 1024 * 1024 // 10GB  - max size of any one file
	const defaultMaxDiskUsage = 5 * defaultMaxFileSize // 50GB  - total disk usage for all files
	const maxChunkSize = 1024 * 1024 * 1024            // 1GB   - max size of any one chunk
	const defaultChunkSize = 1024 * 1024 * 64          // 64MB - default size of chunk
	config := &clientpb.CrackConfig{
		AutoFire:     true,
		MaxFileSize:  defaultMaxFileSize,
		ChunkSize:    defaultChunkSize,
		MaxDiskUsage: defaultMaxDiskUsage,
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			crackConfigLog.Errorf("Failed to read crack config file %s", err)
			return config, err
		}
		err = json.Unmarshal(data, config)
		if err != nil {
			crackConfigLog.Errorf("Failed to parse crack config file %s", err)
			return &clientpb.CrackConfig{
				AutoFire:     true,
				MaxFileSize:  defaultMaxFileSize,
				ChunkSize:    defaultChunkSize,
				MaxDiskUsage: defaultMaxDiskUsage,
			}, err
		}
		if config.ChunkSize < 1024 {
			crackConfigLog.Warnf("Invalid chunk size %d, using default %d", config.ChunkSize, defaultChunkSize)
			config.ChunkSize = defaultChunkSize
		}
		if config.MaxFileSize < 1024 {
			crackConfigLog.Warnf("Invalid max file size %d, using default %d", config.MaxFileSize, defaultMaxFileSize)
			config.MaxFileSize = defaultMaxFileSize
		}
		if maxChunkSize < config.ChunkSize {
			crackConfigLog.Warnf("Chunk size is too large, setting to %d", maxChunkSize)
			config.ChunkSize = maxChunkSize
		}
		if config.MaxFileSize < config.ChunkSize {
			crackConfigLog.Warnf("Chunk size is larger than max file size, setting to %d", config.MaxFileSize)
			config.ChunkSize = config.MaxFileSize
		}
		if config.MaxDiskUsage < config.MaxFileSize {
			crackConfigLog.Warnf("Max disk usage is smaller than max file size, setting to %d", config.MaxFileSize)
			config.MaxDiskUsage = config.MaxFileSize
		}
		return config, nil
	} else {
		crackConfigLog.Debugf("Crack config file does not exist, using defaults")
	}
	err := SaveCrackConfig(config) // This updates the config with any missing fields
	if err != nil {
		crackConfigLog.Errorf("Failed to save default config %s", err)
	}
	return config, nil
}
