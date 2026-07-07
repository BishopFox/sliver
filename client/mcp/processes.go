package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	listProcessesToolName = "list_processes"
	killProcessToolName   = "kill_process"
)

// list_processes 工具参数
type listProcessesArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	FullInfo       bool   `json:"full_info,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

// kill_process 工具参数
type killProcessArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	PID            int32  `json:"pid"`
	Force          bool   `json:"force,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

// list_processes 结果
type processInfo struct {
	PID          int32    `json:"pid"`
	PPID         int32    `json:"ppid"`
	Executable   string   `json:"executable"`
	Owner        string   `json:"owner"`
	Architecture string   `json:"architecture,omitempty"`
	SessionID    int32    `json:"session_id,omitempty"`
	CmdLine      []string `json:"cmd_line,omitempty"`
}

type listProcessesResult struct {
	Processes []processInfo `json:"processes"`
	Count     int           `json:"count"`
}

// kill_process 结果
type killProcessResult struct {
	PID     int32 `json:"pid"`
	Success bool  `json:"success"`
}

// list_processes 处理器
func (s *SliverMCPServer) listProcessesHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args listProcessesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	return s.handleListProcesses(ctx, args)
}

func (s *SliverMCPServer) handleListProcesses(ctx context.Context, args listProcessesArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(listProcessesToolName, args.SessionID, args.BeaconID)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	psResp, err := s.Rpc.Ps(ctx, &sliverpb.PsReq{
		Request:  req,
		FullInfo: args.FullInfo,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list processes", err), nil
	}

	if psResp.Response != nil && psResp.Response.Err != "" {
		return mcpapi.NewToolResultError(psResp.Response.Err), nil
	}

	if isBeacon && psResp.Response != nil && psResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("list_processes", psResp.Response.TaskID, psResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Ps{}
		if err := s.waitForBeaconTaskResponse(ctx, psResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await list processes task", err), nil
		}
		psResp = resolved
		if psResp.Response != nil && psResp.Response.Err != "" {
			return mcpapi.NewToolResultError(psResp.Response.Err), nil
		}
	}

	result := listProcessesResult{
		Processes: make([]processInfo, 0, len(psResp.Processes)),
	}

	for _, proc := range psResp.Processes {
		if proc == nil {
			continue
		}
		result.Processes = append(result.Processes, processInfo{
			PID:          proc.Pid,
			PPID:         proc.Ppid,
			Executable:   proc.Executable,
			Owner:        proc.Owner,
			Architecture: proc.Architecture,
			SessionID:    proc.SessionID,
			CmdLine:      proc.CmdLine,
		})
	}

	result.Count = len(result.Processes)
	return mcpapi.NewToolResultStructuredOnly(result), nil
}

// kill_process 处理器
func (s *SliverMCPServer) killProcessHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args killProcessArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.PID <= 0 {
		return mcpapi.NewToolResultError("pid is required and must be positive"), nil
	}

	return s.handleKillProcess(ctx, args)
}

func (s *SliverMCPServer) handleKillProcess(ctx context.Context, args killProcessArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(
		killProcessToolName,
		args.SessionID,
		args.BeaconID,
		fmt.Sprintf("pid=%d", args.PID),
		fmt.Sprintf("force=%t", args.Force),
	)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	terminateResp, err := s.Rpc.Terminate(ctx, &sliverpb.TerminateReq{
		Request: req,
		Pid:     args.PID,
		Force:   args.Force,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to terminate process", err), nil
	}

	if terminateResp.Response != nil && terminateResp.Response.Err != "" {
		return mcpapi.NewToolResultError(terminateResp.Response.Err), nil
	}

	if isBeacon && terminateResp.Response != nil && terminateResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("kill_process", terminateResp.Response.TaskID, terminateResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Terminate{}
		if err := s.waitForBeaconTaskResponse(ctx, terminateResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await kill process task", err), nil
		}
		terminateResp = resolved
		if terminateResp.Response != nil && terminateResp.Response.Err != "" {
			return mcpapi.NewToolResultError(terminateResp.Response.Err), nil
		}
	}

	return mcpapi.NewToolResultStructuredOnly(killProcessResult{
		PID:     terminateResp.Pid,
		Success: true,
	}), nil
}
