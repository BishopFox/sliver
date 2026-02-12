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
	"errors"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/spf13/cobra"
)

// ErrNonReactableEvent - Event does not exist or is not supported by reactions.
// ErrNonReactableEvent - Event 不存在或不受 reactions. 支持
var ErrNonReactableEvent = errors.New("non-reactable event type")

// ReactionSetCmd - Set a reaction upon an event.
// ReactionSetCmd - Set 对 event. 的反应
func ReactionSetCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	eventType, err := getEventType(cmd, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.Println()
	con.PrintInfof("Setting reaction to: %s\n", EventTypeToTitle(eventType))
	con.Println()
	rawCommands, err := userCommands()
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	commands := []string{}
	for _, rawCommand := range strings.Split(rawCommands, "\n") {
		if rawCommand != "" {
			commands = append(commands, rawCommand)
		}
	}

	reaction := core.Reactions.Add(core.Reaction{
		EventType: eventType,
		Commands:  commands,
	})

	con.Println()
	con.PrintInfof("Set reaction to %s (id: %d)\n", eventType, reaction.ID)
}

func getEventType(cmd *cobra.Command, con *console.SliverClient) (string, error) {
	rawEventType, _ := cmd.Flags().GetString("event")
	if rawEventType == "" {
		return selectEventType(con)
	} else {
		for _, eventType := range core.ReactableEvents {
			if eventType == rawEventType {
				return eventType, nil
			}
		}
		return "", ErrNonReactableEvent
	}
}

func selectEventType(con *console.SliverClient) (string, error) {
	selection := ""
	err := forms.Select("Select an event:", core.ReactableEvents, &selection)
	if err != nil {
		return "", err
	}
	for _, eventType := range core.ReactableEvents {
		if eventType == selection {
			return eventType, nil
		}
	}
	return "", ErrNonReactableEvent
}

func userCommands() (string, error) {
	text := ""
	err := forms.Text("Enter commands:", &text)
	return text, err
}
