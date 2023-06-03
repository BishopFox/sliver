package assets

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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

const (
	// ConfigDirName - Directory name containing config files
	ConfigDirName = "configs"
)

// ClientConfig - Client JSON config
type ClientConfig struct {
	Operator      string `json:"operator"` // This value is actually ignored for the most part (cert CN is used instead)
	LHost         string `json:"lhost"`
	LPort         int    `json:"lport"`
	Token         string `json:"token"`
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

// GetConfigDir - Returns the path to the config dir
func GetConfigDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ConfigDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetConfigs - Returns a list of available configs
func GetConfigs() map[string]*ClientConfig {
	configDir := GetConfigDir()
	configFiles, err := os.ReadDir(configDir)
	if err != nil {
		log.Printf("No configs found %v", err)
		return map[string]*ClientConfig{}
	}

	confs := map[string]*ClientConfig{}
	for _, confFile := range configFiles {
		confFilePath := filepath.Join(configDir, confFile.Name())
		// log.Printf("Parsing config %s", confFilePath)

		conf, err := ReadConfig(confFilePath)
		if err != nil {
			continue
		}
		digest := sha256.Sum256([]byte(conf.Certificate))
		confs[fmt.Sprintf("%s@%s (%x)", conf.Operator, conf.LHost, digest[:8])] = conf
	}
	return confs
}

// ReadConfig - Load config into struct
func ReadConfig(confFilePath string) (*ClientConfig, error) {
	confFile, err := os.Open(confFilePath)
	if err != nil {
		log.Printf("Open failed %v", err)
		return nil, err
	}
	defer confFile.Close()
	data, err := io.ReadAll(confFile)
	if err != nil {
		log.Printf("Read failed %v", err)
		return nil, err
	}
	conf := &ClientConfig{}
	err = json.Unmarshal(data, conf)
	if err != nil {
		log.Printf("Parse failed %v", err)
		return nil, err
	}
	return conf, nil
}

// SaveConfig - Save a config to disk
func SaveConfig(config *ClientConfig) error {
	if config.LHost == "" || config.Operator == "" {
		return errors.New("empty config")
	}
	configDir := GetConfigDir()
	filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(config.Operator), filepath.Base(config.LHost))
	saveTo, _ := filepath.Abs(filepath.Join(configDir, filename))
	configJSON, _ := json.Marshal(config)
	err := os.WriteFile(saveTo, configJSON, 0o600)
	if err != nil {
		log.Printf("Failed to write config to: %s (%v)", saveTo, err)
		return err
	}
	log.Printf("Saved new client config to: %s", saveTo)
	return nil
}
