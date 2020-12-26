package handlers

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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/transports"
)

// RPCResponse - Request/response callback
type RPCResponse func([]byte, error)

// RPCHandler - Request handler
type RPCHandler func([]byte, RPCResponse)

// SpecialHandler - Handlers that need to interact directly with the transport
type SpecialHandler func([]byte, *transports.Connection) error

// TunnelHandler - Tunnel related functionality for duplex connections
type TunnelHandler func(*sliverpb.Envelope, *transports.Connection)

// PivotHandler - Handler related to pivoting
type PivotHandler func(*sliverpb.Envelope, *transports.Connection)

// CommHandler - Handler for managing network routes
type CommHandler func(*sliverpb.Envelope, *transports.Connection)
