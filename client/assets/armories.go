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
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	armoryConfigFileName = "armories.json"
)

var (
	// DefaultArmoryPublicKey - The default public key for the armory
	DefaultArmoryPublicKey string
	// DefaultArmoryRepoURL - The default repo url for the armory
	DefaultArmoryRepoURL string

	defaultArmoryConfig = &ArmoryConfig{
		PublicKey: DefaultArmoryPublicKey,
		RepoURL:   DefaultArmoryRepoURL,
	}
)

// ArmoryConfig - The armory config file
type ArmoryConfig struct {
	PublicKey        string `json:"public_key"`
	RepoURL          string `json:"repo_url"`
	Authorization    string `json:"authorization"`
	AuthorizationCmd string `json:"authorization_cmd"`
}

// GetArmoriesConfig - The parsed armory config file
func GetArmoriesConfig() []*ArmoryConfig {
	armoryConfigPath := filepath.Join(GetRootAppDir(), armoryConfigFileName)
	if _, err := os.Stat(armoryConfigPath); os.IsNotExist(err) {
		return []*ArmoryConfig{defaultArmoryConfig}
	}
	data, err := ioutil.ReadFile(armoryConfigPath)
	if err != nil {
		return []*ArmoryConfig{defaultArmoryConfig}
	}
	var armoryConfigs []*ArmoryConfig
	err = json.Unmarshal(data, &armoryConfigs)
	if err != nil {
		return []*ArmoryConfig{defaultArmoryConfig}
	}
	for _, armoryConfig := range armoryConfigs {
		if armoryConfig.AuthorizationCmd != "" {
			armoryConfig.Authorization = executeAuthorizationCmd(armoryConfig)
		}
	}
	return append(armoryConfigs, defaultArmoryConfig)
}

func executeAuthorizationCmd(armoryConfig *ArmoryConfig) string {
	if armoryConfig.AuthorizationCmd == "" {
		return ""
	}
	out, err := exec.Command(armoryConfig.AuthorizationCmd).CombinedOutput()
	if err != nil {
		log.Printf("Failed to execute authorization_cmd '%s': %v", armoryConfig.AuthorizationCmd, err)
		return ""
	}
	return string(out)
}
