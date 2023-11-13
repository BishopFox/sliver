package forwarder

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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

// {{if .Config.IncludeWG}}
var (
	tcpForwarders map[int]*WGTCPForwarder
	socksServers  map[int]*WGSocksServer
)

// GetTCPForwarders - Returns a map of WireGuard TCP forwarders
func GetTCPForwarders() map[int]*WGTCPForwarder {
	return tcpForwarders
}

// GetSocksServers - Returns a map of WireGuard SOCKS proxies
func GetSocksServers() map[int]*WGSocksServer {
	return socksServers
}

// GetTCPForwarder - Returns a WireGuard TCP forwarder by id
func GetTCPForwarder(id int) *WGTCPForwarder {
	if f, ok := tcpForwarders[id]; ok {
		return f
	}
	return nil
}

// RemoteTCPForwarder - Remove a TCP forwarder by id
func RemoveTCPForwarder(id int) {
	delete(tcpForwarders, id)
}

// GetSocksServer - Returns a WireGuard SOCKS proxy by id
func GetSocksServer(id int) *WGSocksServer {
	if s, ok := socksServers[id]; ok {
		return s
	}
	return nil
}

// RemoveSocksServer - Remove a SOCKS proxy by id
func RemoveSocksServer(id int) {
	delete(socksServers, id)
}

func init() {
	tcpForwarders = make(map[int]*WGTCPForwarder)
	socksServers = make(map[int]*WGSocksServer)
}

// {{end}}
