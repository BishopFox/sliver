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
	"flag"
	"fmt"
	"os"

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

// LoadServerConfig - Determines if this console has either builtin server configuration or if it needs to use a textfile configuration.
// Depending on this, it loads all configuration values and makes them accessible to all packages/components of the client console.
func LoadServerConfig() error {

	// We first check that we have builtin values. If yes, load them.

	// Check if we have imported a config with os.Flags passed to sliver-client executable.
	// This flag has been parsed when executing main(), before anything else.
	if *config != "" {
		conf, err := readConfig(*config)
		if err != nil {
			fmt.Printf("[!] %s\n", err)
			os.Exit(3)
		}
		assets.SaveConfig(conf)
	}

	// Then check if we have textfile configs. If yes, go on.

	// If we have both types of configs, prompt user for choice:
	// use builtin or text files.

	// If textfile is chosen, prompt user for which config, if more than 1 available.

	// Depending on all steps above, load values and return

	return nil
}

// selectConfig - Prompt user to choose which server configuration to load/use.
func selectConfig() *ClientConfig {
	return nil
}

// getConfigs - Returns all available server configurations.
func getConfigs() (configs map[string]*ClientConfig) {
	return
}

// getPromptForConfigs - Prompt user to choose config
func getPromptForConfigs(configs map[string]*assets.ClientConfig) []*survey.Question {
	return nil
}

// readConfig - Loads the contents of a config file into the above gloval variables.
// This possibly overwrite default builtin values, but we have previously prompted the user
// to choose between builtin and textfile config values.
func readConfig(confFilePath string) (*ClientConfig, error) {
	return nil, nil
}

// saveConfig - Save a config to disk
func saveConfig(config *ClientConfig) error {
	return nil
}
