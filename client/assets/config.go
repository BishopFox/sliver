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
	// ConfigDirName - 包含配置文件的目录名
	ConfigDirName = "configs"
)

// ClientConfig - Client JSON config
// ClientConfig - Client JSON 配置
type ClientConfig struct {
	Operator      string `json:"operator"` // This value is actually ignored for the most part (cert CN is used instead)
	Operator      string `json:"operator"` // This 值实际上大部分被忽略（使用证书 CN 代替）
	// 该值在大多数情况下会被忽略（改用 cert CN）
	LHost         string `json:"lhost"`
	LPort         int    `json:"lport"`
	Token         string `json:"token"`
	CACertificate string `json:"ca_certificate"`
	PrivateKey    string `json:"private_key"`
	Certificate   string `json:"certificate"`
}

// GetConfigDir - Returns the path to the config dir
// GetConfigDir - 返回配置目录路径
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
// GetConfigs - 返回可用配置列表
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
		// log.Printf("解析配置 %s", confFilePath)

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
// ReadConfig - 将配置加载到结构体
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
// SaveConfig - 将配置保存到磁盘
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
