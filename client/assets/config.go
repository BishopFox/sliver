package assets

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

const (
	// ConfigDirName - Directory name containing config files
	ConfigDirName = "configs"
)

// ClientConfig - Client JSON config
type ClientConfig struct {
	Operator      string `json:"operator"`
	LHost         string `json:"lhost"`
	LPort         int    `json:"lport"`
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

// GetConfigDir - Returns the path to the config dir
func GetConfigDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := path.Join(rootDir, ConfigDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetConfigs - Returns a list of available configs
func GetConfigs() map[string]*ClientConfig {
	configDir := GetConfigDir()
	configFiles, err := ioutil.ReadDir(configDir)
	if err != nil {
		log.Printf("No configs found %v", err)
		return map[string]*ClientConfig{}
	}

	confs := map[string]*ClientConfig{}
	for _, confFile := range configFiles {
		confFilePath := path.Join(configDir, confFile.Name())
		log.Printf("Parsing config %s", confFilePath)

		conf, err := ReadConfig(confFilePath)
		if err != nil {
			continue
		}
		confs[conf.LHost] = conf
	}
	return confs
}

// ReadConfig - Load config into struct
func ReadConfig(confFilePath string) (*ClientConfig, error) {
	confFile, err := os.Open(confFilePath)
	defer confFile.Close()
	if err != nil {
		log.Printf("Open failed %v", err)
		return nil, err
	}
	data, err := ioutil.ReadAll(confFile)
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
		return errors.New("Empty config")
	}
	configDir := GetConfigDir()
	filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(config.Operator), filepath.Base(config.LHost))
	saveTo, _ := filepath.Abs(path.Join(configDir, filename))
	configJSON, _ := json.Marshal(config)
	err := ioutil.WriteFile(saveTo, configJSON, 0644)
	if err != nil {
		log.Printf("Failed to write config to: %s (%v)", saveTo, err)
		return err
	}
	log.Printf("Saved new client config to: %s", saveTo)
	return nil
}
