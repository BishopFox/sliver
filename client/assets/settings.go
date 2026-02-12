package assets

/*
	Sliver Implant Framework
	Sliver implant 框架
	Copyright (C) 2021  Bishop Fox
	版权所有 (C) 2021 Bishop Fox

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
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	settingsFileName       = "tui-settings.yaml"
	settingsLegacyFileName = "tui-settings.json"
)

const (
	// Accepted values for tui-settings.yaml "prompt:":
	// tui-settings.yaml "prompt:" 的可接受值：
	// - host: show "[host]" prefix on sliver-client, "[server]" on server console
	// - host：在 sliver-client 上显示 "[host]" 前缀，在 server console 上显示 "[server]"
	// - operator-host: show "[operator@host]" prefix on sliver-client, "[server]" on server console
	// - operator-host：在 sliver-client 上显示 "[operator@host]" 前缀，在 server console 上显示 "[server]"
	// - basic: show "sliver >" (no prefix)
	// - basic：显示 "sliver >"（无前缀）
	// - custom: render full prompt from prompt_template
	// - custom：根据 prompt_template 渲染完整 prompt
	PromptStyleHost         = "host"
	PromptStyleOperatorHost = "operator-host"
	PromptStyleBasic        = "basic"
	PromptStyleCustom       = "custom"
)

// ClientSettings - Client JSON config
// ClientSettings - Client JSON 配置
type ClientSettings struct {
	TableStyle        string `json:"tables" yaml:"tables"`
	AutoAdult         bool   `json:"autoadult" yaml:"autoadult"`
	BeaconAutoResults bool   `json:"beacon_autoresults" yaml:"beacon_autoresults"`
	SmallTermWidth    int    `json:"small_term_width" yaml:"small_term_width"`
	AlwaysOverflow    bool   `json:"always_overflow" yaml:"always_overflow"`
	VimMode           bool   `json:"vim_mode" yaml:"vim_mode"`
	UserConnect       bool   `json:"user_connect" yaml:"user_connect"`
	ConsoleLogs       bool   `json:"console_logs" yaml:"console_logs"`
	PromptStyle       string `json:"prompt" yaml:"prompt"`
	PromptTemplate    string `json:"prompt_template" yaml:"prompt_template"`
}

// LoadSettings - Load the client settings from disk
// LoadSettings - 从磁盘加载 client settings
func LoadSettings() (*ClientSettings, error) {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	settingsPath := filepath.Join(rootDir, settingsFileName)
	legacyPath := filepath.Join(rootDir, settingsLegacyFileName)
	settings := defaultSettings()
	migratedLegacy := false

	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if err = yaml.Unmarshal(data, settings); err != nil {
			return defaultSettings(), err
		}
	} else if !os.IsNotExist(err) {
		return defaultSettings(), err
	} else if data, err = os.ReadFile(legacyPath); err == nil {
		if err = json.Unmarshal(data, settings); err != nil {
			return defaultSettings(), err
		}
		migratedLegacy = true
	} else if !os.IsNotExist(err) {
		return defaultSettings(), err
	}

	// Ensure any missing/unknown values are coerced to a supported prompt style.
	// 确保任何缺失/未知值都被规范为受支持的 prompt 风格。
	settings.PromptStyle = NormalizePromptStyle(settings.PromptStyle)
	if err := SaveSettings(settings); err != nil {
		return settings, err
	}
	if migratedLegacy {
		if err := renameLegacyConfig(legacyPath); err != nil {
			return settings, err
		}
	}
	return settings, nil
}

// NormalizePromptStyle canonicalizes prompt style strings and returns a safe
// NormalizePromptStyle 规范化 prompt 风格字符串，并返回安全的
// default if the value is empty/unknown.
// 默认值（当该值为空/未知时）。
func NormalizePromptStyle(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case PromptStyleOperatorHost:
		return PromptStyleOperatorHost
	case PromptStyleHost:
		return PromptStyleHost
	case PromptStyleBasic:
		return PromptStyleBasic
	case PromptStyleCustom:
		return PromptStyleCustom

	// Backward compatible aliases.
	// 向后兼容的别名。
	case "show host":
		return PromptStyleHost
	case "show user and host":
		return PromptStyleOperatorHost
	case "show operator and host":
		return PromptStyleOperatorHost
	case "operator@host":
		return PromptStyleOperatorHost
	case "user@host":
		return PromptStyleOperatorHost

	case "":
		return PromptStyleHost
	default:
		return PromptStyleHost
	}
}

func defaultSettings() *ClientSettings {
	return &ClientSettings{
		TableStyle:        "SliverDefault",
		AutoAdult:         false,
		BeaconAutoResults: true,
		SmallTermWidth:    170,
		AlwaysOverflow:    false,
		VimMode:           false,
		ConsoleLogs:       true,
		PromptStyle:       PromptStyleHost,
		PromptTemplate:    DefaultPromptTemplate,
	}
}

const DefaultPromptTemplate = `{{- if .IsServer -}}{{ .Styles.Bold.Render "[server]" }} {{ .Styles.Underline.Render "sliver" }}{{ .Target.Suffix }} > {{- else -}}{{- if .Host -}}{{ .Styles.BoldPrimary.Render (printf "[%s]" .Host) }} {{- end -}}{{ .Styles.Underline.Render "sliver" }}{{ .Target.Suffix }} > {{- end -}}`

// SaveSettings - Save the current settings to disk
// SaveSettings - 将当前 settings 保存到磁盘
func SaveSettings(settings *ClientSettings) error {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	if settings == nil {
		settings = defaultSettings()
	}
	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(rootDir, settingsFileName), data, 0o600)
	return err
}
