package reaction

import (
	"errors"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/desertbit/grumble"
)

var (
	// ErrNonReactableEvent - Event does not exist or is not supported by reactions
	ErrNonReactableEvent = errors.New("non-reactable event type")
)

// ReactionSetCmd - Set a reaction upon an event
func ReactionSetCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	eventType, err := getEventType(ctx, con)
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

func getEventType(ctx *grumble.Context, con *console.SliverConsoleClient) (string, error) {
	rawEventType := ctx.Flags.String("event")
	if rawEventType == "" {
		return selectEventType(con)
	} else {
		for _, eventType := range ReactableEvents {
			if eventType == rawEventType {
				return eventType, nil
			}
		}
		return "", ErrNonReactableEvent
	}
}

func selectEventType(con *console.SliverConsoleClient) (string, error) {
	prompt := &survey.Select{
		Message: "Select an event:",
		Options: ReactableEvents,
	}
	selection := ""
	err := survey.AskOne(prompt, &selection)
	if err != nil {
		return "", err
	}
	for _, eventType := range ReactableEvents {
		if eventType == selection {
			return eventType, nil
		}
	}
	return "", ErrNonReactableEvent
}

func userCommands() (string, error) {
	text := ""
	prompt := &survey.Multiline{
		Message: "Enter commands: ",
	}
	err := survey.AskOne(prompt, &text)
	return text, err
}
