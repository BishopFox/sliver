package webhooks

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

import (
	"fmt"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"github.com/slack-go/slack"
)

var (
	slackLog = log.NamedLogger("webhooks", "slack")
)

type SlackWebhookConfig struct {
	Enabled bool `json:"enabled"`

	AuthToken string   `json:"auth_token"`
	Channels  []string `json:"channels"`

	SessionOpened    bool `json:"session_opened"`
	BeaconRegistered bool `json:"beacon_registered"`
}

// StartSlackWebhook - Start a slack webhook
func StartSlackWebhook(config *SlackWebhookConfig) {
	StopSlackWebhook() // Stop any existing web hook
	events := core.EventBroker.Subscribe()
	client := slack.New(config.AuthToken)
	if client == nil {
		fmt.Printf("Invalid slack token")
		return
	}
	go func() {
		// This goroutine will exit when events is closed
		for event := range events {
			switch event.EventType {

			// Sessions
			case consts.SessionOpenedEvent:
				if config.SessionOpened {
					sendSlackMsg(client, config.Channels, "New session")
				}

			// Beacons
			case consts.BeaconRegisteredEvent:
				if config.BeaconRegistered {
					sendSlackMsg(client, config.Channels, "New beacon")
				}

			}
		}
	}()
	webhooks.Store(Slack, events)
}

// StopSlackWebhook - Stop a slack webhook
func StopSlackWebhook() {
	if events, ok := webhooks.LoadAndDelete(Slack); ok {
		slackLog.Debugf("Stopping slack webhook %v", events)
		close(events.(chan core.Event))
	}
}

func sendSlackMsg(client *slack.Client, channels []string, msg string) {
	attachment := slack.Attachment{
		Pretext: "some pretext",
		Text:    msg,
	}
	for _, channelID := range channels {
		_, _, err := client.PostMessage(
			channelID,
			slack.MsgOptionText("Some text", false),
			slack.MsgOptionAttachments(attachment),
			slack.MsgOptionAsUser(true),
		)
		if err != nil {
			slackLog.Errorf("%s\n", err)
			return
		}
	}
}
