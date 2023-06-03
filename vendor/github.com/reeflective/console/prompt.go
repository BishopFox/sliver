package console

import (
	"fmt"
	"strings"

	"github.com/reeflective/readline"
)

// Prompt - A prompt is a set of functions that return the strings to print
// for each prompt type. The console will call these functions to retrieve
// the prompt strings to print. Each menu has its own prompt.
type Prompt struct {
	Primary   func() string            // Primary is the main prompt.
	Secondary func() string            // Secondary is the prompt used when the user is typing a multi-line command.
	Transient func() string            // Transient is used if the console shell is configured to be transient.
	Right     func() string            // Right is the prompt printed on the right side of the screen.
	Tooltip   func(word string) string // Tooltip is used to hint on the root command, replacing right prompts if not empty.

	console *Console
}

func newPrompt(app *Console) *Prompt {
	prompt := &Prompt{console: app}

	prompt.Primary = func() string {
		promptStr := app.name

		menu := app.activeMenu()

		if menu.name == "" {
			return promptStr + " > "
		}

		promptStr += fmt.Sprintf(" [%s]", menu.name)

		// If the buffered command output is not empty,
		// add a special status indicator to the prompt.
		if strings.TrimSpace(menu.out.String()) != "" {
			promptStr += " $(...)"
		}

		return promptStr + " > "
	}

	return prompt
}

// bind reassigns the prompt printing functions to the shell helpers.
func (p *Prompt) bind(shell *readline.Shell) {
	prompt := shell.Prompt

	// If the user has bound its own primary prompt and the shell
	// must leave a newline after command/log output, wrap its function
	// to add a newline before the prompt.
	primary := func() string {
		if p.Primary == nil {
			return ""
		}

		prompt := p.Primary()

		return prompt
	}

	prompt.Primary(primary)
	prompt.Right(p.Right)
	prompt.Secondary(p.Secondary)
	prompt.Transient(p.Transient)
	prompt.Tooltip(p.Tooltip)
}
