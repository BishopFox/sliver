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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"

	"gopkg.in/AlecAivazis/survey.v1"
)

var (
	// config - add config files not yet in configs directory
	config = flag.String("import", "", "import config file to ~/.sliver-client/configs")

	// Config - The configuration for the server to which the client is connected.
	Config *ServerConfig
)

const (
	// ConfigDirName - Directory name containing config files
	ConfigDirName = "configs"
)

// ServerConfig - Client JSON config
type ServerConfig struct {
	Operator          string `json:"operator"` // This value is actually ignored for the most part (cert CN is used instead)
	LHost             string `json:"lhost"`
	LPort             int    `json:"lport"`
	CACertificate     string `json:"ca_certificate"`
	PrivateKey        string `json:"private_key"`
	Certificate       string `json:"certificate"`
	ServerFingerprint string `json:"server_fingerprint"`
}

// LoadServerConfig - Determines if this console has either builtin server configuration or if it needs to use a textfile configuration.
// Depending on this, it loads all configuration values and makes them accessible to all packages/components of the client console.
func LoadServerConfig() (err error) {

	// Check if we have imported a config with os.Flags passed to sliver-client executable.
	// This flag has been parsed when executing main(), before anything else.
	if *config != "" {
		conf, err := ReadConfig(*config)
		if err != nil {
			fmt.Printf("[!] %s\n", err)
			os.Exit(3)
		}
		SaveConfig(conf)
	}

	// Then check if we have textfile configs. If yes, go on.
	configs := GetConfigs()

	// prompt user for which config.
	// We should not have an error here, so we must exit.
	err = selectConfig(configs)
	if err != nil {
		fmt.Printf("[!] Error with config loading (selection): %s\n", err)
		os.Exit(3)
	}

	// We should have a config
	if Config == nil {
		fmt.Printf("[!] Error with config loading: config is nil\n")
		os.Exit(3)
		return
	}

	return nil
}

// GetConfigDir - Returns the path to the config dir
func GetConfigDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := path.Join(rootDir, ConfigDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetConfigs - Returns all available server configurations.
func GetConfigs() (configs map[string]*ServerConfig) {
	configDir := GetConfigDir()
	configFiles, err := ioutil.ReadDir(configDir)
	if err != nil {
		return map[string]*ServerConfig{}
	}

	configs = map[string]*ServerConfig{}
	for _, confFile := range configFiles {
		confFilePath := path.Join(configDir, confFile.Name())

		conf, err := ReadConfig(confFilePath)
		if err != nil {
			continue
		}
		digest := sha256.Sum256([]byte(conf.Certificate))
		configs[fmt.Sprintf("%s@%s (%x)", conf.Operator, conf.LHost, digest[:8])] = conf
	}
	return
}

// ReadConfig - Loads the contents of a config file into the above gloval variables.
// This possibly overwrite default builtin values, but we have previously prompted the user
// to choose between builtin and textfile config values.
func ReadConfig(confFilePath string) (*ServerConfig, error) {
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
	conf := &ServerConfig{}
	err = json.Unmarshal(data, conf)
	if err != nil {
		log.Printf("Parse failed %v", err)
		return nil, err
	}
	return conf, nil
}

// SaveConfig - Save a config to disk
func SaveConfig(config *ServerConfig) error {
	if config.LHost == "" || config.Operator == "" {
		return errors.New("Empty config")
	}
	configDir := GetConfigDir()
	filename := fmt.Sprintf("%s_%s.cfg", filepath.Base(config.Operator), filepath.Base(config.LHost))
	saveTo, _ := filepath.Abs(path.Join(configDir, filename))
	configJSON, _ := json.Marshal(config)
	err := ioutil.WriteFile(saveTo, configJSON, 0600)
	if err != nil {
		log.Printf("Failed to write config to: %s (%v)", saveTo, err)
		return err
	}
	log.Printf("Saved new client config to: %s", saveTo)

	return nil
}

// selectConfig - Prompt user to choose which server configuration to load/use.
func selectConfig(configs map[string]*ServerConfig) (err error) {

	if len(configs) == 0 {
		return nil
	}

	// If only config, load values, else prompt user.
	if len(configs) == 1 {
		for i := range configs {
			Config = configs[i]
		}
	} else {
		answer := struct{ Config string }{}
		qs := getPromptForConfigs(configs)
		err = survey.Ask(qs, &answer)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		Config = configs[answer.Config]
	}

	return
}

// getPromptForConfigs - Prompt user to choose config
func getPromptForConfigs(configs map[string]*ServerConfig) []*survey.Question {

	keys := []string{}
	for k := range configs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return []*survey.Question{
		{
			Name: "config",
			Prompt: &survey.Select{
				Message: "Select a server:",
				Options: keys,
				Default: keys[0],
			},
		},
	}
}
