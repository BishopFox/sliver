package assets

/*
	Sliver Implant Framework
	Sliver implant 框架
	Copyright (C) 2019  Bishop Fox
	版权所有 (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	本程序是自由软件：你可以再发布和/或修改它
	it under the terms of the GNU General Public License as published by
	在自由软件基金会发布的 GNU General Public License 条款下，
	the Free Software Foundation, either version 3 of the License, or
	可以使用许可证第 3 版，或
	(at your option) any later version.
	（由你选择）任何更高版本。

	This program is distributed in the hope that it will be useful,
	发布本程序是希望它能发挥作用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但不提供任何担保；甚至不包括
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	对适销性或特定用途适用性的默示担保。请参阅
	GNU General Public License for more details.
	GNU General Public License 以获取更多细节。

	You should have received a copy of the GNU General Public License
	你应当已随本程序收到一份 GNU General Public License 副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	如果没有，请参见 <https://www.gnu.org/licenses/>。
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
	// DefaultArmoryPublicKey - armory 的默认公钥
	DefaultArmoryPublicKey string
	// DefaultArmoryRepoURL - The default repo url for the armory
	// DefaultArmoryRepoURL - armory 的默认仓库 URL
	DefaultArmoryRepoURL string

	DefaultArmoryConfig = &ArmoryConfig{
		PublicKey: DefaultArmoryPublicKey,
		RepoURL:   DefaultArmoryRepoURL,
		Name:      DefaultArmoryName,
		Enabled:   true,
	}
)

// ArmoryConfig - The armory config file
// ArmoryConfig - armory 配置文件
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
// GetArmoriesConfig - 解析后的 armory 配置文件
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
	// 强制将默认 armory 放在最后
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
