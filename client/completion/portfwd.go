package completion

import (
	"fmt"
	"strconv"

	"github.com/bishopfox/sliver/client/core"
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

// CompleteInBandForwarders - Returns the list of active in-band port forwarders ID (all sessions)
func CompleteInBandForwarders() (comps []*readline.CompletionGroup) {

	comp := &readline.CompletionGroup{
		Name:         "in-band port forwarders",
		MaxLength:    20,
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	fwdList := core.Portfwds.List()

	if len(fwdList) == 0 {
		return
	}

	for _, fwd := range fwdList {
		id := strconv.Itoa(int(fwd.ID))
		comp.Suggestions = append(comp.Suggestions, id)
		desc := fmt.Sprintf(" %s  -->  %s (%d)", fwd.BindAddr, fwd.RemoteAddr, fwd.SessionID)
		comp.Descriptions[id] = readline.DIM + desc + readline.RESET
	}

	return []*readline.CompletionGroup{comp}
}
