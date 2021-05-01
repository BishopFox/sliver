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
	"context"
	"fmt"
	"strconv"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// SessionIDs - Completes session IDs along with a description.
func SessionIDs() (comps []*readline.CompletionGroup) {

	comp := &readline.CompletionGroup{
		Name:         "sessions",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return
	}
	for _, s := range sessions.Sessions {
		sessionID := strconv.Itoa(int(s.ID))
		comp.Suggestions = append(comp.Suggestions, sessionID)
		desc := fmt.Sprintf("[%s] - %s@%s - %s", s.Name, s.Username, s.Hostname, s.RemoteAddress)
		comp.Descriptions[sessionID] = readline.DIM + desc + readline.RESET
	}

	return []*readline.CompletionGroup{comp}
}
