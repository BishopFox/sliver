package completers

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
	"strings"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/util"
)

// completeEnvironmentVariables - Returns all environment variables as suggestions
func completeEnvironmentVariables(lastWord string) (last string, completions []*readline.CompletionGroup) {

	// Check if last input is made of several different variables
	allVars := strings.Split(lastWord, "/")
	lastVar := allVars[len(allVars)-1]

	var evaluated = map[string]string{}

	grp := &readline.CompletionGroup{
		Name:        "console OS environment",
		MaxLength:   5, // Should be plenty enough
		DisplayType: readline.TabDisplayGrid,
		TrimSlash:   true, // Some variables can be paths
	}

	for k, v := range util.ClientEnv {
		if strings.HasPrefix("$"+k, lastVar) {
			grp.Suggestions = append(grp.Suggestions, "$"+k+"/")
			evaluated[k] = v
		}
	}

	completions = append(completions, grp)

	return lastVar, completions
}
