package reaction

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
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// ReactionCmd - Manage reactions to events.
func ReactionCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	totalReactions := 0
	for _, eventType := range core.ReactableEvents {
		reactions := core.Reactions.On(eventType)
		if 0 < len(reactions) {
			if totalReactions != 0 {
				con.Printf("\n") // Add newline between each table after the first
			}
			displayReactionsTable(eventType, reactions, con)
		}
		totalReactions += len(reactions)
	}
	if totalReactions == 0 {
		con.PrintInfof("No reactions set\n")
	}
}

func displayReactionsTable(eventType string, reactions []core.Reaction, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(fmt.Sprintf(console.Bold+"%s"+console.Normal, EventTypeToTitle(eventType)))
	tw.AppendSeparator()
	slackSpace := len(EventTypeToTitle(eventType)) - len("Commands") - len("ID") - 3
	if slackSpace < 0 {
		slackSpace = 1
	}
	tw.AppendHeader(table.Row{
		"ID",
		"Commands" + strings.Repeat(" ", slackSpace), // Leave space for title
	})
	for _, react := range reactions {
		tw.AppendRow(table.Row{
			react.ID,
			strings.Join(react.Commands, ","),
		})
	}
	con.Printf("%s\n", tw.Render())
}

// EventTypeToTitle - Convert an eventType to a more human friendly string.
func EventTypeToTitle(eventType string) string {
	switch eventType {

	case consts.SessionOpenedEvent:
		return "Session Opened"
	case consts.SessionClosedEvent:
		return "Session Closed"
	case consts.SessionUpdateEvent:
		return "Session Updated"

	case consts.BeaconRegisteredEvent:
		return "Beacon Registered"

	case consts.CanaryEvent:
		return "Canary Trigger"

	case consts.WatchtowerEvent:
		return "Watchtower Trigger"

	default:
		return eventType
	}
}
