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
	"strconv"

	"gopkg.in/AlecAivazis/survey.v1"
)

var (
	// Variables used as build-time server configuration (address, certificates, user, etc)
	// When a console is compiled from the server, it may decide which values to inject.

	// HasBuiltinServer - If this is different from the template
	// string below, it means we use the builtin values.
	HasBuiltinServer = `{{.HasBuiltinServer}}`

	// ServerLHost - Host of server
	ServerLHost = `{{.LHost}}`

	// ServerLPort - Port on which to contact server
	ServerLPort = `{{.LPort}}`

	// ServerUser - Username
	ServerUser = `{{.User}}`

	// ServerCACertificate - CA Certificate
	ServerCACertificate = `{{.CACertificate}}`

	// ServerCertificate - CA Certificate
	ServerCertificate = `{{.Certificate}}`

	// ServerPrivateKey - Private key
	ServerPrivateKey = `{{.PrivateKey}}`

	// Token - A unique number for this client binary
	Token = `{{.Token}}`
)

var (
	// config - add config files not yet in configs directory
	config = flag.String("import", "", "import config file to ~/.sliver-client/configs")
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
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

// LoadServerConfig - Determines if this console has either builtin server configuration or if it needs to use a textfile configuration.
// Depending on this, it loads all configuration values and makes them accessible to all packages/components of the client console.
func LoadServerConfig() (err error) {

	// We first check that we have builtin values. If yes, print message and load them.
	if HasBuiltinServer == "true" {
		fmt.Println("[-] Found compile-time server configuration")
	}

	// Check if we have imported a config with os.Flags passed to sliver-client executable.
	// This flag has been parsed when executing main(), before anything else.
	if *config != "" {
		conf, err := readConfig(*config)
		if err != nil {
			fmt.Printf("[!] %s\n", err)
			os.Exit(3)
		}
		saveConfig(conf)
	}

	// Then check if we have textfile configs. If yes, go on.
	configs := getConfigs()
	if len(configs) > 0 {
		fmt.Println("[-] Found text-file server configuration(s)")
	}

	// If we have only builtin values at our disposal, return: they are available already.
	if HasBuiltinServer == "true" && len(configs) == 0 {
		fmt.Println("[-] Using compile-time values (no textfile configs found)")
		return
	}

	// If we have both types of configs, prompt user for choice: use builtin or text files ?
	if HasBuiltinServer == "true" && len(configs) > 0 {
		useBuiltin := promptBuiltinOrTextFile()
		if useBuiltin {
			return
		}

	}

	// Here, textfile is chosen: prompt user for which config.
	// We should not have an error here, so we must exit.
	err = selectConfig(configs)
	if err != nil {
		fmt.Printf("[!] Error with config loading (selection): %s\n", err)
		os.Exit(3)
	}

	// Check a value to see if config is correctly loaded.
	if ServerCertificate == "{{.Certificate}}" {
		fmt.Printf("[!] Error at config values checkup: still having {{.Certificate}}")
		os.Exit(3)
	}

	return nil
}

// getConfigDir - Returns the path to the config dir
func getConfigDir() string {
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

// getConfigs - Returns all available server configurations.
func getConfigs() (configs map[string]*ClientConfig) {
	configDir := getConfigDir()
	configFiles, err := ioutil.ReadDir(configDir)
	if err != nil {
		log.Printf("No configs found %v", err)
		return map[string]*ClientConfig{}
	}

	configs = map[string]*ClientConfig{}
	for _, confFile := range configFiles {
		confFilePath := path.Join(configDir, confFile.Name())
		log.Printf("Parsing config %s", confFilePath)

		conf, err := readConfig(confFilePath)
		if err != nil {
			continue
		}
		digest := sha256.Sum256([]byte(conf.Certificate))
		configs[fmt.Sprintf("%s@%s (%x)", conf.Operator, conf.LHost, digest[:8])] = conf
	}
	return
}

// readConfig - Loads the contents of a config file into the above gloval variables.
// This possibly overwrite default builtin values, but we have previously prompted the user
// to choose between builtin and textfile config values.
func readConfig(confFilePath string) (*ClientConfig, error) {
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

// saveConfig - Save a config to disk
func saveConfig(config *ClientConfig) error {
	if config.LHost == "" || config.Operator == "" {
		return errors.New("Empty config")
	}
	configDir := getConfigDir()
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
func selectConfig(configs map[string]*ClientConfig) (err error) {

	var conf *ClientConfig

	if len(configs) == 0 {
		return nil
	}

	// If only config, load values, else prompt user.
	if len(configs) == 1 {
		for i := range configs {
			conf = configs[i]
		}
	} else {
		answer := struct{ Config string }{}
		qs := getPromptForConfigs(configs)
		err = survey.Ask(qs, &answer)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		conf = configs[answer.Config]
	}

	// Load values for config choice.
	ServerLHost = conf.LHost
	ServerLPort = strconv.Itoa(int(conf.LPort))
	ServerUser = conf.Operator
	ServerCACertificate = conf.CACertificate
	ServerCertificate = conf.Certificate
	ServerPrivateKey = conf.PrivateKey

	log.Printf("Loaded configuration values for server at %s:%s (operator %s)", ServerLHost, ServerLPort, ServerUser)
	return
}

// getPromptForConfigs - Prompt user to choose config
func getPromptForConfigs(configs map[string]*ClientConfig) []*survey.Question {

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

// promptBuiltinOrTextFile - Asks user if he wants to use builtin values or textfile configs.
func promptBuiltinOrTextFile() (builtin bool) {

	choices := []string{"builtin", "text-file"}

	message := "Both compile-time (builtin) and textfile configuration are available. Please choose:"
	question := []*survey.Question{
		{
			Name: "config type",
			Prompt: &survey.Select{
				Message: message,
				Options: choices,
				Default: choices[0],
			},
		},
	}

	answer := struct{ Choice string }{}
	err := survey.Ask(question, &answer)
	if err != nil {
		fmt.Println("[!] Falling back to builtin due to errors: " + err.Error())
		return true // If we have an error, don't bother and use builtin.
	}

	if answer.Choice == "textfile" {
		return false
	}

	return true
}
