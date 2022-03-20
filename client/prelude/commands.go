package prelude

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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

//RunCommand executes a given command
func RunCommand(message string, executor string, payload []byte, agentSession *OperatorImplantBridge, onFinish func(string, int, int)) (string, int, int) {
	switch executor {
	case "keyword":
		task := splitMessage(message, '.')
		switch task[0] {
		case "bof":
			if len(task) != 2 {
				break
			}
			var bargs []bofArgs
			argStr := strings.ReplaceAll(task[1], `\`, `\\`)
			err := json.Unmarshal([]byte(argStr), &bargs)
			if err != nil {
				return err.Error(), ErrorExitStatus, ErrorExitStatus
			}
			if payload == nil {
				return "missing BOF file", ErrorExitStatus, ErrorExitStatus
			}
			out, err := runBOF(agentSession.Implant, agentSession.RPC, payload, bargs, onFinish)
			if err != nil {
				return err.Error(), ErrorExitStatus, ErrorExitStatus
			}
			return out, 0, 0
		case "execute-assembly":
			if len(task) < 2 {
				break
			}
			var eaargs execasmArgs
			argStr := strings.ReplaceAll(task[1], `\`, `\\`)
			err := json.Unmarshal([]byte(argStr), &eaargs)
			if err != nil {
				return err.Error(), ErrorExitStatus, ErrorExitStatus
			}
			if payload == nil {
				return "missing .NET assembly", ErrorExitStatus, ErrorExitStatus
			}
			out, err := execAsm(agentSession.Implant, agentSession.RPC, payload, eaargs, onFinish)
			if err != nil {
				return err.Error(), ErrorExitStatus, ErrorExitStatus
			}
			return out, 0, 0
		case "exit":
			return shutdown(agentSession)
		default:
			return "Keyword selected not available for agent", ErrorExitStatus, ErrorExitStatus
		}
	default:
		bites, status, pid := execute(message, executor, agentSession, onFinish)
		return string(bites), status, pid
	}
	return "", ErrorExitStatus, ErrorExitStatus
}

func execute(cmd string, executor string, implantBridge *OperatorImplantBridge, onFinishCallback func(string, int, int)) (string, int, int) {
	args := append(getCmdArg(executor), cmd)
	if executor == "psh" {
		executor = "powershell.exe"
	} else if executor == "exec" {
		commandSections := strings.Fields(cmd)
		executor = commandSections[0]
		args = commandSections[1:]
	}
	execResp, err := implantBridge.RPC.Execute(context.Background(), &sliverpb.ExecuteReq{
		Path:    executor,
		Args:    args,
		Output:  true,
		Request: MakeRequest(implantBridge.Implant),
	})

	if err != nil {
		return fmt.Sprintf("Error: %s\n", err.Error()), -1, -1
	}

	// Beacon
	if execResp.Response != nil && execResp.Response.Async {
		implantBridge.BeaconCallback(execResp.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, execResp)
			if err != nil {
				return
			}
			onFinishCallback(string(execResp.Stdout), int(execResp.Status), int(execResp.Pid))
		})
		return "", 0, 0
	}

	// Session
	if execResp.Response != nil && execResp.Response.Err != "" {
		return execResp.Response.Err, SuccessExitStatus, SuccessExitStatus
	}
	return string(execResp.Stdout), int(execResp.Status), int(execResp.Pid)
}

func getCmdArg(executor string) []string {
	var args []string
	switch executor {
	case "cmd":
		args = []string{"/S", "/C"}
	case "powershell", "psh":
		args = []string{"-execu", "-C"}
	case "exec":
		args = []string{}
	case "sh", "bash", "zsh":
		args = []string{"-c"}
	}
	return args
}

func splitMessage(message string, splitRune rune) []string {
	quoted := false
	values := strings.FieldsFunc(message, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == splitRune
	})
	return values
}

func shutdown(agentSession *OperatorImplantBridge) (string, int, int) {
	_, err := agentSession.RPC.Kill(context.Background(), &sliverpb.KillReq{
		Force:   false,
		Request: MakeRequest(agentSession.Implant),
	})
	if err != nil {
		return err.Error(), ErrorExitStatus, ErrorExitStatus
	}
	return fmt.Sprintf("Terminated %s", agentSession.Implant.GetID()), SuccessExitStatus, int(agentSession.Implant.GetPID())
}
