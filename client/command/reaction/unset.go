package reaction

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/desertbit/grumble"
)

// ReactionUnsetCmd - Unset a reaction upon an event
func ReactionUnsetCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	reactionID := ctx.Flags.Int("id")
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
		con.PrintWarnf("No reaction found with id %d", reactionID)
	}
	con.Println()
}

func selectReaction(con *console.SliverConsoleClient) (*core.Reaction, error) {
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
