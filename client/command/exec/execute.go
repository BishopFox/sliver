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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// ExecuteCmd - Run a command on the remote system.
func ExecuteCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	cmdPath := args[0]
	args = args[1:]

	token, _ := cmd.Flags().GetBool("token")
	hidden, _ := cmd.Flags().GetBool("hidden")
	output, _ := cmd.Flags().GetBool("output")
	stdout, _ := cmd.Flags().GetString("stdout")
	stderr, _ := cmd.Flags().GetString("stderr")
	saveLoot, _ := cmd.Flags().GetBool("loot")
	saveOutput, _ := cmd.Flags().GetBool("save")
	ppid, _ := cmd.Flags().GetUint32("ppid")
	hostName := getHostname(session, beacon)

	// If the user wants to loot or save the output, we have to capture it regardless of if they specified -o
	captureOutput := output || saveLoot || saveOutput

	if output && beacon != nil {
		con.PrintWarnf("Using --output in beacon mode, if the command blocks the task will never complete\n\n")
	}

	var exec *sliverpb.Execute
	var err error

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Executing %s %s ...", cmdPath, strings.Join(args, " ")), ctrl)
	if token || hidden || ppid != 0 {
		if (session != nil && session.OS != "windows") || (beacon != nil && beacon.OS != "windows") {
			con.PrintErrorf("The token, hide window, and ppid options are not valid on %s\n", session.OS)
			return
		}
		exec, err = con.Rpc.ExecuteWindows(context.Background(), &sliverpb.ExecuteWindowsReq{
			Request:    con.ActiveTarget.Request(cmd),
			Path:       cmdPath,
			Args:       args,
			Output:     captureOutput,
			Stderr:     stderr,
			Stdout:     stdout,
			UseToken:   token,
			HideWindow: hidden,
			PPid:       ppid,
		})
	} else {
		exec, err = con.Rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
			Request: con.ActiveTarget.Request(cmd),
			Path:    cmdPath,
			Args:    args,
			Output:  captureOutput,
			Stderr:  stderr,
			Stdout:  stdout,
		})
	}
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	if exec.Response != nil && exec.Response.Async {
		con.AddBeaconCallback(exec.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, exec)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			HandleExecuteResponse(exec, cmdPath, hostName, cmd, con)
		})
		con.PrintAsyncResponse(exec.Response)
	} else {
		HandleExecuteResponse(exec, cmdPath, hostName, cmd, con)
	}
}

func HandleExecuteResponse(exec *sliverpb.Execute, cmdPath string, hostName string, cmd *cobra.Command, con *console.SliverClient) {
	var lootedOutput []byte
	stdout, _ := cmd.Flags().GetString("stdout")
	saveLoot, _ := cmd.Flags().GetBool("loot")
	saveOutput, _ := cmd.Flags().GetBool("save")
	lootName, _ := cmd.Flags().GetString("name")
	ignoreStderr, _ := cmd.Flags().GetBool("ignore-stderr")

	if saveLoot || saveOutput {
		lootedOutput = combineCommandOutput(exec, stdout == "", !ignoreStderr && 0 < len(exec.Stderr))
	}

	if saveLoot {
		LootExecute(lootedOutput, lootName, cmd.Name(), cmdPath, hostName, con)
	}

	if saveOutput {
		SaveExecutionOutput(string(lootedOutput), cmd.Name(), hostName, con)
	}

	PrintExecute(exec, cmd, con)
}

// PrintExecute - Print the output of an executed command.
func PrintExecute(exec *sliverpb.Execute, cmd *cobra.Command, con *console.SliverClient) {
	ignoreStderr, _ := cmd.Flags().GetBool("ignore-stderr")
	stdout, _ := cmd.Flags().GetString("stdout")
	stderr, _ := cmd.Flags().GetString("stderr")

	output, _ := cmd.Flags().GetBool("output")
	if !output {
		if exec.Status == 0 {
			con.PrintInfof("Command executed successfully\n")
		} else {
			con.PrintErrorf("Exit code %d\n", exec.Status)
		}
		return
	}

	if stdout == "" {
		con.PrintInfof("Output:\n%s", string(exec.Stdout))
	} else {
		con.PrintInfof("Stdout saved at %s\n", stdout)
	}

	if stderr == "" {
		if !ignoreStderr && 0 < len(exec.Stderr) {
			con.PrintInfof("Stderr:\n%s", string(exec.Stderr))
		}
	} else {
		con.PrintInfof("Stderr saved at %s\n", stderr)
	}

	if exec.Status != 0 {
		con.PrintErrorf("Exited with status %d!\n", exec.Status)
	}
}

func getHostname(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.Hostname
	}
	if beacon != nil {
		return beacon.Hostname
	}
	return ""
}

func determineCommandName(command string) string {
	commandName := strings.ReplaceAll(command, "\\", "/")

	commandName = commandName[strings.LastIndex(commandName, "/")+1:]

	if strings.Contains(commandName, ".") {
		commandName = commandName[:strings.LastIndex(commandName, ".")]
	}

	return commandName
}

func combineCommandOutput(exec *sliverpb.Execute, combineStdOut bool, combineStdErr bool) []byte {
	var outputString string = ""

	if combineStdOut {
		outputString += "Output (stdout):\n" + string(exec.Stdout)
	}

	if combineStdErr {
		if combineStdOut {
			outputString += "\n"
		}
		outputString += "Stderr:\n" + string(exec.Stderr)
	}

	return []byte(outputString)
}

func LootExecute(commandOutput []byte, lootName string, sliverCmdName string, cmdName string, hostName string, con *console.SliverClient) {
	if len(commandOutput) == 0 {
		con.PrintInfof("There was no output from execution, so there is nothing to loot.\n")
		return
	}

	timeNow := time.Now().UTC().Format("20060102150405")

	shortCommandName := determineCommandName(cmdName)

	fileName := fmt.Sprintf("%s_%s_%s_%s.log", sliverCmdName, hostName, shortCommandName, timeNow)
	if lootName == "" {
		lootName = fmt.Sprintf("[%s] %s on %s (%s)", sliverCmdName, shortCommandName, hostName, timeNow)
	}

	lootMessage := loot.CreateLootMessage(con.ActiveTarget.GetHostUUID(), fileName, lootName, clientpb.FileType_TEXT, commandOutput)
	loot.SendLootMessage(lootMessage, con)
}

func PrintExecutionOutput(executionOutput string, saveOutput bool, commandName string, hostName string, con *console.SliverClient) {
	con.PrintInfof("Output:\n%s", executionOutput)

	if saveOutput {
		SaveExecutionOutput(executionOutput, commandName, hostName, con)
	}
}

func SaveExecutionOutput(executionOutput string, commandName string, hostName string, con *console.SliverClient) {
	var outFilePath *os.File
	var err error

	if len(executionOutput) == 0 {
		con.PrintInfof("There was no output from execution, so there is nothing to save.")
		return
	}

	timeNow := time.Now().UTC().Format("20060102150405")

	outFileName := filepath.Base(fmt.Sprintf("%s_%s_%s*.log", commandName, hostName, timeNow))

	outFilePath, err = os.CreateTemp("", outFileName)

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if outFilePath != nil {
		outFilePath.WriteString(executionOutput)
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}
}
