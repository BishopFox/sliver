package commands

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

// Portfwd - Manage port forwarders on the client
type Portfwd struct{}

// Execute - Print the port forwarders by default
func (p *Portfwd) Execute(args []string) (err error) {

	return
}

// PortfwdOpen - Start a client-implant port forwarder
type PortfwdOpen struct {
	Options struct {
		Protocol  string `long:"protocol" description:"Transport protocol" default:"tcp"`
		Reverse   bool   `long:"reverse" description:"Reverse forwards from Rhost (implant) to LHost (client)"`
		LHost     string `long:"lhost" description:"Console address to dial/listen on" default:"127.0.0.1"`
		LPort     int    `long:"lport" description:"Console listen port" default:"2020"`
		RHost     string `long:"rhost" description:"Remote host address to dial/listen on" default:"0.0.0.0"`
		RPort     int    `long:"rport" description:"Remote port number" required:"true"`
		SessionID uint32 `long:"session-id" description:"Start the forwarder on a specific session"`
	} `group:"forwarder options"`
}

// Execute -  Start a client-implant port forwarder
func (p *PortfwdOpen) Execute(args []string) (err error) {
	return
}

// PortfwdClose - Stop a port forwarder
type PortfwdClose struct {
	Options struct {
		ID         []string `long:"id" description:"Forwarder IDs, comma-separated" env-delim:","`
		Protocol   string   `long:"protocol" description:"Close only for given transport"`
		Reverse    bool     `long:"reverse" description:"Close only if reverse"`
		SessionID  uint32   `long:"session-id" description:"Close if forwarder belongs to session"`
		CloseConns bool     `long:"close-conns" description:"Close active connections initiated by forwarder (TCP-only)"`
	} `group:"forwarder options"`
}

// Execute -  Start a client-implant port forwarder
func (p *PortfwdClose) Execute(args []string) (err error) {
	return
}
