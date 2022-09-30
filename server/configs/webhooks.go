package configs

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
)

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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

type SlackWebhookConfig struct {
	Enabled bool `json:"enabled"`

	AuthToken string       `json:"auth_token"`
	Channels  []string     `json:"channels"`
	Events    EventsConfig `json:"events"`
}

func SlackWebhookConfigFromProtobuf(config *clientpb.SlackWebhook) *SlackWebhookConfig {
	return &SlackWebhookConfig{
		Enabled:   config.Enabled,
		AuthToken: config.AuthToken,
		Channels:  config.Channels,
		Events: EventsConfig{
			SessionOpened:    config.Events.SessionOpened,
			BeaconRegistered: config.Events.BeaconRegistered,
		},
	}
}

type EventsConfig struct {
	SessionOpened    bool `json:"session_opened"`
	BeaconRegistered bool `json:"beacon_registered"`
}

func SaveSlackConfig(config *SlackWebhookConfig) error {
	slackConfigPath := filepath.Join(assets.GetRootAppDir(), "configs", "slack.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(slackConfigPath, data, 0600)
}

func LoadSlackConfig() (*SlackWebhookConfig, error) {
	slackConfigPath := filepath.Join(assets.GetRootAppDir(), "configs", "slack.json")
	if _, err := os.Stat(slackConfigPath); os.IsNotExist(err) {
		return nil, nil
	}
	data, err := os.ReadFile(slackConfigPath)
	if err != nil {
		return nil, err
	}
	config := &SlackWebhookConfig{}
	err = json.Unmarshal(data, config)
	return config, err
}
