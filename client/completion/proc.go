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
	"fmt"
	"strconv"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/core"
)

// SessionProcesses - Gives a list of remote Session processes, along with their description.
func SessionProcesses() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "host processes",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
		MaxLength:    20,
	}

	session := core.ActiveSession
	if session == nil {
		return
	}

	// Get the session completions cache
	sessCache := Cache.GetSessionCache(core.ActiveSession.ID)
	if sessCache == nil {
		return
	}

	// And get the processes
	ps := sessCache.GetProcesses()
	if ps == nil {
		return
	}

	for _, proc := range ps.Processes {
		pid := strconv.Itoa(int(proc.Pid))
		comp.Suggestions = append(comp.Suggestions, pid)
		var color string
		if session != nil && proc.Pid == session.PID {
			color = readline.GREEN
		}
		desc := fmt.Sprintf("%s(%d - %s)  %s", color, proc.Ppid, proc.Owner, proc.Executable)
		comp.Descriptions[pid] = readline.DIM + desc + readline.RESET
	}

	return []*readline.CompletionGroup{comp}
}

//SessionProcessNames - Returns a map of remote Session processes, to be grabbed by their names
func SessionProcessNames() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "host process names",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayMap,
		MaxLength:    20,
	}

	session := core.ActiveSession
	if session == nil {
		return
	}

	// Get the session completions cache
	sessCache := Cache.GetSessionCache(core.ActiveSession.ID)
	if sessCache == nil {
		return
	}

	// And get the processes
	ps := sessCache.GetProcesses()
	if ps == nil {
		return
	}

	for _, proc := range ps.Processes {
		comp.Suggestions = append(comp.Suggestions, proc.Executable)
		var color string
		if session != nil && proc.Pid == session.PID {
			color = readline.GREEN
		}
		desc := fmt.Sprintf("%s%d  (%d - %s)  ", color, proc.Pid, proc.Ppid, proc.Owner)
		comp.Descriptions[proc.Executable] = readline.DIM + desc + readline.RESET
	}

	return []*readline.CompletionGroup{comp}
}
