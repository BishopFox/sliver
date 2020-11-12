package console

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
	"github.com/BishopFox/sliver/protobuf/clientpb"
)

// startEventHandler - Handle all events coming from the server.
func (c *console) startEventHandler() (err error) {

	// Listen for events on the RPC stream.

	for {
		// Few things to note:
		// 1 - Change the EventTypes or regroup them in the functions below,
		// because each type of event may trigger different console behavior.
	}
}

// ServerEvent - Events that concern only the server and its client, not implants.
func serverEvent(event clientpb.Event) {}
