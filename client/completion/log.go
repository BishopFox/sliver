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

	"github.com/bishopfox/sliver/client/log"
	"github.com/maxlandon/readline"
)

// Command/option argument choices
var (
	// Logs & components
	logLevels = []string{"trace", "debug", "info", "warning", "error"}
)

func LogLevels() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "levels",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
	}
	for _, lvl := range logLevels {
		comp.Suggestions = append(comp.Suggestions, lvl)
	}

	return []*readline.CompletionGroup{comp}
}

// Loggers - Returns the names of all loggers instantiated in the client console
func Loggers() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "loggers",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayGrid,
	}
	var loggerNames []string
	for name := range log.Loggers {
		loggerNames = append(loggerNames, name)
	}
	sort.Strings(loggerNames)

	for _, name := range loggerNames {
		comp.Suggestions = append(comp.Suggestions, name)
	}

	return []*readline.CompletionGroup{comp}
}
