package reaction

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// ReactionCmd - Manage reactions to events.
// ReactionCmd - Manage 对 events. 的反应
func ReactionCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	totalReactions := 0
	for _, eventType := range core.ReactableEvents {
		reactions := core.Reactions.On(eventType)
		if 0 < len(reactions) {
			if totalReactions != 0 {
				con.Printf("\n") // Add newline between each table after the first
				con.Printf("\n") // 第一个之后每个表之间的 Add 换行符
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
	tw.SetTitle(console.StyleBold.Render(EventTypeToTitle(eventType)))
	tw.AppendSeparator()
	slackSpace := len(EventTypeToTitle(eventType)) - len("Commands") - len("ID") - 3
	if slackSpace < 0 {
		slackSpace = 1
	}
	tw.AppendHeader(table.Row{
		"ID",
		"Commands" + strings.Repeat(" ", slackSpace), // Leave space for title
		"Commands" + strings.Repeat(" ", slackSpace), // Leave 标题空间
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
// EventTypeToTitle - Convert 和 eventType 到更人性化的 string.
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
