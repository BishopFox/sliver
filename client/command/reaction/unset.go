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
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/spf13/cobra"
)

// ReactionUnsetCmd - Unset a reaction upon an event.
// ReactionUnsetCmd - Unset 对 event. 的反应
func ReactionUnsetCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	reactionID, _ := cmd.Flags().GetInt("id")
	if reactionID == 0 {
		reaction, err := selectReaction(con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		reactionID = reaction.ID
	}
	success := core.Reactions.Remove(reactionID)
	if success {
		con.Println()
		con.PrintInfof("Successfully removed reaction with id %d", reactionID)
	} else {
		con.PrintErrorf("No reaction found with id %d", reactionID)
	}
	con.Println()
}

func selectReaction(con *console.SliverClient) (*core.Reaction, error) {
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)
	allReactions := core.Reactions.All()
	for _, react := range allReactions {
		fmt.Fprintf(table, "%d\t%s\t%s\t\n",
			react.ID, react.EventType, strings.Join(react.Commands, ", "),
		)
	}
	table.Flush()
	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove blank line at the end
	options = options[:len(options)-1] // Remove 末尾空行

	selection := ""
	err := forms.Select("Select a reaction:", options, &selection)
	if err != nil {
		return nil, err
	}
	for index, option := range options {
		if option == selection {
			return &allReactions[index], nil
		}
	}
	return nil, errors.New("reaction not found")
}
