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
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	ArmoryConfigFileName = "armories.json"
	DefaultArmoryName    = "Default"
)

var (
	// DefaultArmoryPublicKey - The default public key for the armory
	DefaultArmoryPublicKey string
	// DefaultArmoryRepoURL - The default repo url for the armory
	DefaultArmoryRepoURL string

	DefaultArmoryConfig = &ArmoryConfig{
		PublicKey: DefaultArmoryPublicKey,
		RepoURL:   DefaultArmoryRepoURL,
		Name:      DefaultArmoryName,
		Enabled:   true,
	}
)

// ArmoryConfig - The armory config file
type ArmoryConfig struct {
	PublicKey        string `json:"public_key"`
	RepoURL          string `json:"repo_url"`
	Authorization    string `json:"authorization"`
	AuthorizationCmd string `json:"authorization_cmd"`
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
}

func RefreshArmoryAuthorization(armories []*ArmoryConfig) {
	for _, armoryConfig := range armories {
		if armoryConfig.AuthorizationCmd != "" {
			armoryConfig.Authorization = executeAuthorizationCmd(armoryConfig)
		}
	}
}

// GetArmoriesConfig - The parsed armory config file
func GetArmoriesConfig() []*ArmoryConfig {
	armoryConfigPath := filepath.Join(GetRootAppDir(), ArmoryConfigFileName)
	if _, err := os.Stat(armoryConfigPath); os.IsNotExist(err) {
		return []*ArmoryConfig{DefaultArmoryConfig}
	}
	data, err := os.ReadFile(armoryConfigPath)
	if err != nil {
		return []*ArmoryConfig{DefaultArmoryConfig}
	}
	var armoryConfigsFromFile []*ArmoryConfig
	var armoryConfigs []*ArmoryConfig
	err = json.Unmarshal(data, &armoryConfigsFromFile)
	if err != nil {
		return []*ArmoryConfig{DefaultArmoryConfig}
	}

	// Force the default armory to be the last
	defaultArmorySpecified := false

	for _, config := range armoryConfigsFromFile {
		if config.Name == DefaultArmoryName {
			defaultArmorySpecified = true
			continue
		} else {
			armoryConfigs = append(armoryConfigs, config)
		}
	}
	if defaultArmorySpecified {
		armoryConfigs = append(armoryConfigs, DefaultArmoryConfig)
	}
	RefreshArmoryAuthorization(armoryConfigs)

	return armoryConfigs
}

func SaveArmoriesConfig(armories []*ArmoryConfig) error {
	configData, err := json.Marshal(armories)
	if err != nil {
		return err
	}
	armoryConfigPath := filepath.Join(GetRootAppDir(), ArmoryConfigFileName)
	err = os.WriteFile(armoryConfigPath, configData, 0640)
	if err != nil {
		return err
	}
	return nil
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
