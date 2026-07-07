package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	executeCommandToolName = "execute_command"
)

type executeCommandArgs struct {
	SessionID      string   `json:"session_id,omitempty"`
	BeaconID       string   `json:"beacon_id,omitempty"`
	Command        string   `json:"command"`
	Args           []string `json:"args,omitempty"`
	Output         bool     `json:"output,omitempty"`
	Env            []string `json:"env,omitempty"`
	Background     bool     `json:"background,omitempty"`
	Wait           bool     `json:"wait,omitempty"`
	TimeoutSeconds int64    `json:"timeout_seconds,omitempty"`
}

type executeCommandResult struct {
	StdOut       string `json:"stdout,omitempty"`
	StdErr       string `json:"stderr,omitempty"`
	ExitStatus   int32  `json:"exit_status"`
	Background   bool   `json:"background"`
	Async        bool   `json:"async"`
	Operation    string `json:"operation,omitempty"`
	TaskID       string `json:"task_id,omitempty"`
	BeaconID     string `json:"beacon_id,omitempty"`
	State        string `json:"state,omitempty"`
}

func (s *SliverMCPServer) executeCommandHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args executeCommandArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Command == "" {
		return mcpapi.NewToolResultError("command is required"), nil
	}

	return s.handleExecuteCommand(ctx, args)
}

func (s *SliverMCPServer) handleExecuteCommand(ctx context.Context, args executeCommandArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	extras := []string{fmt.Sprintf("command=%q", args.Command)}
	if len(args.Args) > 0 {
		extras = append(extras, fmt.Sprintf("args=%v", args.Args))
	}
	if args.Background {
		extras = append(extras, "background=true")
	}
	s.logToolCall(executeCommandToolName, args.SessionID, args.BeaconID, extras...)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	// Parse environment variables
	envMap := make(map[string]string)
	for _, env := range args.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	execReq := &sliverpb.ExecuteReq{
		Request:         req,
		Path:            args.Command,
		Args:            args.Args,
		Output:          args.Output || !args.Background,
		Env:             envMap,
		Background:      args.Background,
		EnvInheritance:  len(envMap) == 0,
	}

	execResp, err := s.Rpc.Execute(ctx, execReq)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to execute command", err), nil
	}

	if execResp.Response != nil && execResp.Response.Err != "" {
		return mcpapi.NewToolResultError(execResp.Response.Err), nil
	}

	if isBeacon && execResp.Response != nil && execResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("execute_command", execResp.Response.TaskID, execResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Execute{}
		if err := s.waitForBeaconTaskResponse(ctx, execResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await execute task", err), nil
		}
		execResp = resolved
		if execResp.Response != nil && execResp.Response.Err != "" {
			return mcpapi.NewToolResultError(execResp.Response.Err), nil
		}
	}

	result := executeCommandResult{
		ExitStatus: execResp.Status,
		Background: execResp.Background,
	}

	if execResp.Result != "" {
		result.StdOut = execResp.Result
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
