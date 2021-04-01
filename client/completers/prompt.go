package completers

import (
	"sort"
	"strings"

	"github.com/maxlandon/readline"
)

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

var (
	hints = []string{"show", "hide"}
)

var (
	// serverPromptItems - All items available to the prompt in server context
	// These values can also be used in the Sliver context
	serverPromptItems = map[string]string{
		"{cwd}":       "console working directory",
		"{server_ip}": "address of the server as specified in the config file",
		"{local_ip}":  "first non-loopback interface found in the client host",
		"{jobs}":      "number of current jobs",
		"{sessions}":  "number of registered sessions",
		"{timestamp}": "timestamp refreshed at each prompt print",
	}

	// sliverPromptItems - All items available to the prompt in Sliver context
	// Can only be used in the Sliver context !
	sliverPromptItems = map[string]string{
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

	promptEffects = map[string]string{
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

func promptServerItems(lastWord string) (comps []*readline.CompletionGroup) {

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
			desc := serverPromptItems[item]
			sComp.Suggestions = append(sComp.Suggestions, item)
			sComp.Descriptions[item] = readline.Dim(desc)
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
			desc := promptEffects[item]
			cComp.Suggestions = append(cComp.Suggestions, item)
			cComp.Descriptions[item] = readline.Dim(desc)
		}
	}
	comps = append(comps, cComp)

	return
}

func promptSliverItems(lastWord string) (comps []*readline.CompletionGroup) {

	// Add server values and colors, as all are accessible any time.
	comps = append(comps, promptServerItems(lastWord)...)

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
			desc := sliverPromptItems[item]
			sComp.Suggestions = append(sComp.Suggestions, item)
			sComp.Descriptions[item] = readline.Dim(desc)
		}
	}
	comps = append(comps, sComp)

	return
}
