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
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/spf13/cobra"
)

// ReactionUnsetCmd - Unset a reaction upon an event.
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

	// Prompt user with options
	prompt := &survey.Select{
		Message: "Select a reaction:",
		Options: options,
	}
	selection := ""
	err := survey.AskOne(prompt, &selection)
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
