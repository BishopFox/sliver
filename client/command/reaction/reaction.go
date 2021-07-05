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
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/desertbit/grumble"
)

// ReactionCmd - Manage reactions to events
func ReactionCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	totalReactions := 0
	for _, eventType := range core.ReactableEvents {
		reactions := core.Reactions.On(eventType)
		if 0 < len(reactions) {
			displayReactionsTable(eventType, reactions, con)
		}
		totalReactions += len(reactions)
	}
	if totalReactions == 0 {
		con.PrintInfof("No reactions set\n")
	}
}

func displayReactionsTable(eventType string, reactions []core.Reaction, con *console.SliverConsoleClient) {

	// Title
	con.Printf("%s%s%s\n", console.Bold, EventTypeToTitle(eventType), console.Normal)

	// Table
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "ID\tCommands\t\n")
	fmt.Fprintf(table, "%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Commands")),
	)
	for _, react := range reactions {
		fmt.Fprintf(table, "%d\t%s\t\n",
			react.ID, strings.Join(react.Commands, ","),
		)
	}
	table.Flush()
	con.Printf("%s", outputBuf.String())
}

// EventTypeToTitle - Convert an eventType to a more human friendly string
func EventTypeToTitle(eventType string) string {
	switch eventType {

	case consts.SessionOpenedEvent:
		return "Session Opened"
	case consts.SessionClosedEvent:
		return "Session Closed"
	case consts.SessionUpdateEvent:
		return "Session Updated"

	case consts.CanaryEvent:
		return "Canary Trigger"

	case consts.WatchtowerEvent:
		return "Watchtower Trigger"

	default:
		return eventType
	}
}
