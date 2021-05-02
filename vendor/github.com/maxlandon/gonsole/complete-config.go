package gonsole

import (
	"fmt"
	"sort"
	"strings"

	"github.com/maxlandon/readline"
)

var (
	promptEffectsDesc = map[string]string{
		"{blink}": "blinking", // blinking
		"{bold}":  "bold text",
		"{dim}":   "obscured text",
		"{fr}":    "fore red",
		"{g}":     "fore green",
		"{b}":     "fore blue",
		"{y}":     "fore yellow",
		"{fw}":    "fore white",
		"{bdg}":   "back dark gray",
		"{br}":    "back red",
		"{bg}":    "back green",
		"{by}":    "back yellow",
		"{blb}":   "back light blue",
		"{reset}": "reset effects",
		// Custom colors
		"{ly}":   "light yellow",
		"{lb}":   "light blue (VSCode keyword)", // like VSCode var keyword
		"{db}":   "dark blue",
		"{bddg}": "back dark dark gray",
	}
)

// promptItems - Queries the console menu prompt for all its callbacks and passes them as completions.
func (c *CommandCompleter) promptItems(lastWord string) (prefix string, comps []*readline.CompletionGroup) {

	cc := c.console.current
	serverPromptItems := cc.Prompt.Callbacks

	// Items
	sComp := &readline.CompletionGroup{
		Name:         fmt.Sprintf("%s prompt items", cc.Name),
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
	}

	var keys []string
	for item := range serverPromptItems {
		keys = append(keys, item)
	}
	sort.Strings(keys)
	for _, item := range keys {
		sComp.Suggestions = append(sComp.Suggestions, item)
	}
	comps = append(comps, sComp)

	// Colors & effects
	comps = append(comps, c.promptColors()...)

	return
}

func (c *CommandCompleter) promptColors() (comps []*readline.CompletionGroup) {

	cc := c.console.current
	promptEffects := cc.Prompt.Colors

	// Colors & effects
	cComp := &readline.CompletionGroup{
		Name:         "colors/effects",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	var colorKeys []string
	for item := range promptEffects {
		colorKeys = append(colorKeys, item)
	}
	sort.Strings(colorKeys)
	for _, item := range colorKeys {
		desc, ok := promptEffectsDesc[item]
		if ok {
			cComp.Suggestions = append(cComp.Suggestions, item)
			cComp.Descriptions[item] = readline.Dim(desc)
		} else {
			cComp.Suggestions = append(cComp.Suggestions, item)
		}
	}
	comps = append(comps, cComp)

	return
}

func (c *CommandCompleter) hints() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "hint verbosity",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
		Suggestions:  []string{"show", "hide"},
	}
	return []*readline.CompletionGroup{comp}
}

func (c *CommandCompleter) inputModes() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "input/editing modes",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
		Suggestions:  []string{"vim", "emacs"},
	}
	return []*readline.CompletionGroup{comp}
}

func (c *CommandCompleter) menus() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "console menus (menus)",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
	}
	for _, cc := range c.console.menus {
		comp.Suggestions = append(comp.Suggestions, cc.Name)
	}
	return []*readline.CompletionGroup{comp}
}

func (c *CommandCompleter) highlightTokens() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "line tokens",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	var highlightingTokens = map[string]string{
		"{command}":          "highlight the command words",
		"{command-argument}": "highlight the command arguments",
		"{option}":           "highlight the option name",
		"{option-argument}":  "highlight the option arguments",
		"{hint-text}":        "color of the hint text displayed below prompt",
		// We will dynamically add all <$-env> items as well.
	}

	// Add user-added expansion variables
	for exp, completer := range c.console.current.expansionComps {
		groups := completer()
		var titles []string
		for _, grp := range groups {
			titles = append(titles, grp.Name)
		}
		highlightingTokens[string(exp)] = "(user-added) " + strings.Join(titles, ",")
	}

	// Sort and add to comp group.
	var keys []string
	for item := range highlightingTokens {
		keys = append(keys, item)
	}
	sort.Strings(keys)

	for _, token := range keys {
		comp.Suggestions = append(comp.Suggestions, token)
		comp.Descriptions[token] = readline.DIM + highlightingTokens[token] + readline.RESET
	}

	return []*readline.CompletionGroup{comp}
}
