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
)

const (
	settingsFileName = "tui-settings.json"
)

// ClientSettings - Client JSON config
type ClientSettings struct {
	TableStyle        string `json:"tables"`
	AutoAdult         bool   `json:"autoadult"`
	BeaconAutoResults bool   `json:"beacon_autoresults"`
	SmallTermWidth    int    `json:"small_term_width"`
	AlwaysOverflow    bool   `json:"always_overflow"`
	VimMode           bool   `json:"vim_mode"`
	UserConnect       bool   `json:"user_connect"`
	ConsoleLogs       bool   `json:"console_logs"`
}

// LoadSettings - Load the client settings from disk
func LoadSettings() (*ClientSettings, error) {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	data, err := os.ReadFile(filepath.Join(rootDir, settingsFileName))
	if err != nil {
		return defaultSettings(), err
	}
	settings := defaultSettings()
	err = json.Unmarshal(data, settings)
	if err != nil {
		return defaultSettings(), err
	}
	return settings, nil
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
	}
}

// SaveSettings - Save the current settings to disk
func SaveSettings(settings *ClientSettings) error {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	if settings == nil {
		settings = defaultSettings()
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(rootDir, settingsFileName), data, 0o600)
	return err
}
