package prelude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

//RunCommand executes a given command
func RunCommand(message string, executor string, payload []byte, agentSession *AgentSession) (string, int, int) {
	switch executor {
	case "keyword":
		task := splitMessage(message, '.')
		switch task[0] {
		case "sliver":
			if len(task) != 2 {
				break
			}
			// sliverCommand := task[1]
		case "bof":
			if len(task) != 2 {
				break
			}
			var bargs []bofArgs
			argStr := strings.ReplaceAll(task[1], `\`, `\\`)
			err := json.Unmarshal([]byte(argStr), &bargs)
			if err != nil {
				return fmt.Sprintf("JSON parsing error: %s", err.Error()), ErrorExitStatus, ErrorExitStatus
			}
			if payload == nil {
				return "missing BOF file", ErrorExitStatus, ErrorExitStatus
			}
			out, err := runBOF(agentSession.Session, agentSession.RPC, payload, bargs)
			if err != nil {
				return err.Error(), ErrorExitStatus, ErrorExitStatus
			}
			return out, 0, int(agentSession.Session.PID)
		case "exit":
			return shutdown(agentSession)
		default:
			return "Keyword selected not available for agent", ErrorExitStatus, ErrorExitStatus
		}
	default:
		bites, status, pid := execute(message, executor, agentSession)
		return string(bites), status, pid
	}
	return "", ErrorExitStatus, ErrorExitStatus
}

func execute(cmd string, executor string, agentSession *AgentSession) (string, int, int) {
	args := append(getCmdArg(executor), cmd)
	if executor == "psh" {
		executor = "powershell.exe"
	}
	execResp, err := agentSession.RPC.Execute(context.Background(), &sliverpb.ExecuteReq{
		Path:    executor,
		Args:    args,
		Output:  true,
		Request: MakeRequest(agentSession.Session),
	})

	if err != nil {
		return fmt.Sprintf("Error: %s\n", err.Error()), 0, 0
	}

	if execResp.Response != nil && execResp.Response.Err != "" {
		return execResp.Response.Err, 0, 0
	}
	return execResp.Stdout, int(execResp.Status), int(execResp.Pid)
}

func getCmdArg(executor string) []string {
	var args []string
	switch executor {
	case "cmd":
		args = []string{"/C", "/S"}
	case "powershell", "psh":
		args = []string{"-execu", "ByPasS", "-C"}
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

func shutdown(agentSession *AgentSession) (string, int, int) {
	_, err := agentSession.RPC.KillSession(context.Background(), &sliverpb.KillSessionReq{
		Force:   false,
		Request: MakeRequest(agentSession.Session),
	})
	if err != nil {
		return err.Error(), ErrorExitStatus, ErrorExitStatus
	}
	return fmt.Sprintf("Terminated %s", agentSession.Session.Name), SuccessExitStatus, SuccessExitStatus
}
