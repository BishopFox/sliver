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

	"github.com/bishopfox/sliver/client/prelude/bridge"
	"github.com/bishopfox/sliver/client/prelude/config"
	"github.com/bishopfox/sliver/client/prelude/executor"
	"github.com/bishopfox/sliver/client/prelude/implant"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

// Exxpected message looks like:
// Ls:
// {
//	"Command": "ls",
//	"Arguments": {
//		"Path": "/"
//	  }
//	"Output": "json"
// }
// Extension (BOF):
// {
//	"Command": "run-extension",
//	"Arguments": {
//		"Name": "sa-cacls",
//   	"Arguments": [{"type":"wstring","value":"/"}]
//	  }
// }
// Extension (shared lib):
// {
//	"Command": "run-extension",
//	"Arguments": {
//		"Name": "proc-dump",
//   	"Arguments": ["4343"]
//	  }
// }
type sliverExecutorMessage struct {
	Command   string
	Output    string // "text" or "json", default is "text"
	Arguments interface{}
}

//RunCommand executes a given command
func RunCommand(message string, executor string, payload []byte, agentSession *bridge.OperatorImplantBridge, onFinish func(string, int, int)) (string, int, int) {
	switch executor {
	case "sliver":
		sliverMsg := &sliverExecutorMessage{}
		err := json.Unmarshal([]byte(message), sliverMsg)
		if err != nil {
			return fmt.Sprintf("Error: %s\n", err.Error()), config.ErrorExitStatus, config.ErrorExitStatus
		}
		return runSliverExecutor(sliverMsg, payload, agentSession, onFinish)
	default:
		bites, status, pid := execute(message, executor, agentSession, onFinish)
		return string(bites), status, pid
	}
}

func runSliverExecutor(msg *sliverExecutorMessage, payload []byte, impBridge *bridge.OperatorImplantBridge, onFinish func(string, int, int)) (string, int, int) {
	handler := executor.GetHandler(msg.Command)
	if handler != nil {
		return handler(msg.Arguments, payload, impBridge, onFinish, msg.Output)
	}
	return "Unknown command", config.ErrorExitStatus, config.ErrorExitStatus
}

func execute(cmd string, executor string, implantBridge *bridge.OperatorImplantBridge, onFinishCallback func(string, int, int)) (string, int, int) {
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
		Request: implant.MakeRequest(implantBridge.Implant),
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
		return execResp.Response.Err, config.SuccessExitStatus, config.SuccessExitStatus
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
