package completion

import (
	"strings"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/term"
)

func (e *Engine) hintCompletions(comps Values) {
	hint := ""

	// First add the command/flag usage string if any,
	// and only if we don't have completions.
	if len(comps.values) == 0 || e.config.GetBool("usage-hint-always") {
		if comps.Usage != "" {
			hint += color.Dim + comps.Usage + color.Reset + term.NewlineReturn
		}
	}

	// Add application-specific messages.
	// There is full support for color in them, but in case those messages
	// don't include any, we tame the color a little bit first, like hints.
	messages := strings.Join(comps.Messages.Get(), term.NewlineReturn)
	messages = strings.TrimSuffix(messages, term.NewlineReturn)

	if messages != "" {
		hint = hint + color.Dim + messages
	}

	// If we don't have any completions, and no messages, let's say it.
	if e.Matches() == 0 && hint == color.Dim+term.NewlineReturn && !e.auto {
		hint = e.hintNoMatches()
	}

	hint = strings.TrimSuffix(hint, term.NewlineReturn)
	if hint == "" {
		return
	}

	// Add the hint to the shell.
	e.hint.Set(hint + color.Reset)
}

func (e *Engine) hintNoMatches() string {
	noMatches := color.Dim + "no matching"

	var groups []string

	for _, group := range e.groups {
		if group.tag == "" {
			continue
		}

		groups = append(groups, group.tag)
	}

	if len(groups) > 0 {
		groupsStr := strings.Join(groups, ", ")
		noMatches += "'" + groupsStr + "'"
	}

	return noMatches + " completions"
}
