package completion

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"sort"
	"strings"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/maxlandon/gonsole"
	"github.com/maxlandon/readline"
)

var (
	// Console - Some completion functions need to make choices depending on the
	// current menu context, so they need access to the Console state.
	Console *gonsole.Console
)

// PromptServerItems - Queries the console context prompt for all its callbacks and passes them as completions.
func PromptServerItems(lastWord string) (prefix string, comps []*readline.CompletionGroup) {
	serverContext := Console.GetMenu(constants.ServerMenu)
	serverPromptItems := serverContext.Prompt.Callbacks
	promptEffects := serverContext.Prompt.Colors

	// Items
	sComp := &readline.CompletionGroup{
		Name:         "server items",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	var keys []string
	for item := range serverPromptItems {
		keys = append(keys, item)
	}
	sort.Strings(keys)
	for _, item := range keys {
		if strings.HasPrefix(item, lastWord) {
			desc, ok := serverPromptItemsDesc[item]
			if ok {
				sComp.Suggestions = append(sComp.Suggestions, item)
				sComp.Descriptions[item] = readline.Dim(desc)
			} else {
				sComp.Suggestions = append(sComp.Suggestions, item)
			}
		}
	}
	comps = append(comps, sComp)

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
		if strings.HasPrefix(item, lastWord) {
			desc, ok := promptEffectsDesc[item]
			if ok {
				cComp.Suggestions = append(cComp.Suggestions, item)
				cComp.Descriptions[item] = readline.Dim(desc)
			} else {
				cComp.Suggestions = append(cComp.Suggestions, item)
			}
		}
	}
	comps = append(comps, cComp)

	return
}

// PromptSliverItems - Queries the console context prompt for all its callbacks and passes them as completions.
func PromptSliverItems(lastWord string) (prefix string, comps []*readline.CompletionGroup) {
	sliverContext := Console.GetMenu(constants.SliverMenu)
	sliverPromptItems := sliverContext.Prompt.Callbacks

	_, serverComps := PromptServerItems(lastWord)
	comps = append(comps, serverComps...)

	// Sliver Items
	sComp := &readline.CompletionGroup{
		Name:         "sliver session items",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	var keys []string
	for item := range sliverPromptItems {
		keys = append(keys, item)
	}
	sort.Strings(keys)
	for _, item := range keys {
		if strings.HasPrefix(item, lastWord) {
			desc, ok := sliverPromptItemsDesc[item]
			if ok {
				sComp.Suggestions = append(sComp.Suggestions, item)
				sComp.Descriptions[item] = readline.Dim(desc)
			} else {
				sComp.Suggestions = append(sComp.Suggestions, item)
			}
		}
	}
	comps = append(comps, sComp)

	return
}

var (
	// serverPromptItems - All items available to the prompt in server context
	// These values can also be used in the Sliver context
	serverPromptItemsDesc = map[string]string{
		"{cwd}":       "console working directory",
		"{server_ip}": "address of the server as specified in the config file",
		"{local_ip}":  "first non-loopback interface found in the client host",
		"{jobs}":      "number of current jobs",
		"{sessions}":  "number of registered sessions",
		"{timestamp}": "timestamp refreshed at each prompt print",
	}

	// sliverPromptItems - All items available to the prompt in Sliver context
	// Can only be used in the Sliver context !
	sliverPromptItemsDesc = map[string]string{
		"{session_name}": "name of implant session",
		"{user}":         "session username",
		"{host}":         "session hostname",
		"{platform}":     "OS/arch",
		"{os}":           "OS",
		"{arch}":         "arch",
		"{wd}":           "currrent implant working directory",
		"{address}":      "remote address (can be long because of routes)",
		"{status}":       "implant status",
		"{version}":      "host version info",
		"{uid}":          "user ID",
		"{gid}":          "user group ID",
		"{pid}":          "process ID",
	}

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
