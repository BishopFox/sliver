package comm

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
	"net/url"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

// newConn - Populate a connection message with the handler from which it originates
// for being passed to the server Comm and then routed to its portfwd destination.
func newConn(handler *commpb.Handler, uri *url.URL) *commpb.Conn {

	// We pass the handler's remote address, as it all connections arriving
	// are supposed to be port forwarded to the handler's remote address
	info := &commpb.Conn{
		ID:        handler.ID, // ID of the forwarder.
		Transport: handler.Transport,
		RHost:     handler.RHost,
		RPort:     handler.RPort,
	}

	// Transport / Application protocols
	switch uri.Scheme {
	case "mtls", "http", "https", "socks", "socks5", "ftp", "smtp":
		info.Transport = commpb.Transport_TCP
		switch uri.Scheme {
		case "mtls":
			info.Application = commpb.Application_MTLS
		case "http":
			info.Application = commpb.Application_HTTP
		case "https":
			info.Application = commpb.Application_HTTPS
		case "socks", "socks5":
			info.Application = commpb.Application_Socks5
		case "ftp":
			info.Application = commpb.Application_FTP
		case "smtp":
			info.Application = commpb.Application_SMTP
		case "named_pipe", "named_pipes", "namedpipe", "pipe":
			info.Application = commpb.Application_NamedPipe
		}
	}

	return info
}
