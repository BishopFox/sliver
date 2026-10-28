package mcp

import (
	"context"
	"fmt"

	mcpapi "github.com/mark3labs/mcp-go/mcp"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	executeToolName = "execute"
)

// executeArgs defines the parameters for command execution
type executeArgs struct {
	SessionID      string   `json:"session_id,omitempty"`
	BeaconID       string   `json:"beacon_id,omitempty"`
	Path           string   `json:"path"`
	Args           []string `json:"args,omitempty"`
	Output         bool     `json:"output,omitempty"`
	Wait           bool     `json:"wait,omitempty"`
	TimeoutSeconds int64    `json:"timeout_seconds,omitempty"`
}

// executeResult defines the execution result
type executeResult struct {
	Stdout string `json:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty"`
	Status uint32 `json:"status"`
	Pid    uint32 `json:"pid,omitempty"`
}

func (s *SliverMCPServer) executeHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args executeArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Path == "" {
		return mcpapi.NewToolResultError("path is required"), nil
	}

	// Default to capturing output
	if !args.Output {
		args.Output = true
	}

	return s.handleExecute(ctx, args)
}

func (s *SliverMCPServer) handleExecute(ctx context.Context, args executeArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(executeToolName, args.SessionID, args.BeaconID, fmt.Sprintf("path=%q args=%v", args.Path, args.Args))

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	exec, err := s.Rpc.Execute(ctx, &sliverpb.ExecuteReq{
		Request: req,
		Path:    args.Path,
		Args:    args.Args,
		Output:  args.Output,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to execute command", err), nil
	}

	if exec.Response != nil && exec.Response.Err != "" {
		return mcpapi.NewToolResultError(exec.Response.Err), nil
	}

	if isBeacon && exec.Response != nil && exec.Response.Async {
		if !args.Wait {
			return newAsyncResult("execute", exec.Response.TaskID, exec.Response.BeaconID), nil
		}
		resolved := &sliverpb.Execute{}
		if err := s.waitForBeaconTaskResponse(ctx, exec.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await execute task", err), nil
		}
		exec = resolved
		if exec.Response != nil && exec.Response.Err != "" {
			return mcpapi.NewToolResultError(exec.Response.Err), nil
		}
	}

	result := executeResult{
		Stdout: string(exec.Stdout),
		Stderr: string(exec.Stderr),
		Status: exec.Status,
		Pid:    exec.Pid,
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
