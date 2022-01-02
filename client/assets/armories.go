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
	"os"
	"path/filepath"
)

const (
	armoryConfigFileName = "armories.json"
)

var (
	DefaultArmoryPublicKey string
	DefaultArmoryRepoURL   string

	defaultArmoryConfig = &ArmoryConfig{
		PublicKey: DefaultArmoryPublicKey,
		RepoURL:   DefaultArmoryRepoURL,
	}
)

type ArmoryConfig struct {
	PublicKey string `json:"public_key"`
	RepoURL   string `json:"repo_url"`
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
	return append(armoryConfigs, defaultArmoryConfig)
}
