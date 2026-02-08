package assets

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	// - host: show "[host]" prefix on sliver-client, "[server]" on server console
	// - operator-host: show "[operator@host]" prefix on sliver-client, "[server]" on server console
	// - basic: show "sliver >" (no prefix)
	// - custom: render full prompt from prompt_template
	PromptStyleHost         = "host"
	PromptStyleOperatorHost = "operator-host"
	PromptStyleBasic        = "basic"
	PromptStyleCustom       = "custom"
)

// ClientSettings - Client JSON config
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
// default if the value is empty/unknown.
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
