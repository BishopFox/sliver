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

	"github.com/bishopfox/sliver/client/core"
	"github.com/maxlandon/readline"
)

// CompleteSliverEnv - Returns all environment variables found on the host
func CompleteSliverEnv() (completions []*readline.CompletionGroup) {

	grp := &readline.CompletionGroup{
		Name:         "session OS environment",
		MaxLength:    5,
		DisplayType:  readline.TabDisplayGrid,
		TrimSlash:    true, // Some variables can be paths
		Descriptions: map[string]string{},
	}

	// Per-OS path separator
	if core.ActiveSession.OS == "windows" {
		grp.PathSeparator = '\\'
	} else {
		grp.PathSeparator = '/'
	}

	var clientEnv = map[string]string{}
	sessCache := Cache.GetSessionCache(core.ActiveSession.ID)
	if sessCache == nil {
		return nil
	}

	envInfo := sessCache.GetEnvironmentVariables()
	if envInfo == nil {
		return
	}

	for _, pair := range envInfo.Variables {
		clientEnv[pair.Key] = pair.Value
	}

	keys := make([]string, 0, len(clientEnv))
	for k := range clientEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		grp.Suggestions = append(grp.Suggestions, key)
		value := clientEnv[key]
		grp.Descriptions[key] = value
	}

	// Add some special ones
	grp.Aliases = map[string]string{
		"~": "HOME",
	}

	completions = append(completions, grp)

	return
}
