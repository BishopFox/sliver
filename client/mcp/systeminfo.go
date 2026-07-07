package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	getSystemInfoToolName = "get_system_info"
)

type getSystemInfoArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type systemInfoResult struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Version      string `json:"version"`
	Arch         string `json:"arch"`
	Username     string `json:"username"`
	UID          string `json:"uid,omitempty"`
	GID          string `json:"gid,omitempty"`
	PID          int32  `json:"pid"`
	Locale       string `json:"locale,omitempty"`
	ActiveC2     string `json:"active_c2,omitempty"`
	ProxyURL     string `json:"proxy_url,omitempty"`
}

func (s *SliverMCPServer) getSystemInfoHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args getSystemInfoArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	return s.handleGetSystemInfo(ctx, args)
}

func (s *SliverMCPServer) handleGetSystemInfo(ctx context.Context, args getSystemInfoArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(getSystemInfoToolName, args.SessionID, args.BeaconID)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	sysInfoReq := &sliverpb.SystemInfoReq{
		Request: req,
	}

	sysInfoResp, err := s.Rpc.SystemInfo(ctx, sysInfoReq)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get system info", err), nil
	}

	if sysInfoResp.Response != nil && sysInfoResp.Response.Err != "" {
		return mcpapi.NewToolResultError(sysInfoResp.Response.Err), nil
	}

	if isBeacon && sysInfoResp.Response != nil && sysInfoResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("get_system_info", sysInfoResp.Response.TaskID, sysInfoResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.SystemInfo{}
		if err := s.waitForBeaconTaskResponse(ctx, sysInfoResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await system info task", err), nil
		}
		sysInfoResp = resolved
		if sysInfoResp.Response != nil && sysInfoResp.Response.Err != "" {
			return mcpapi.NewToolResultError(sysInfoResp.Response.Err), nil
		}
	}

	result := systemInfoResult{
		Hostname: sysInfoResp.Hostname,
		OS:       sysInfoResp.OS,
		Version:  sysInfoResp.Version,
		Arch:     sysInfoResp.Arch,
		Username: sysInfoResp.Username,
		UID:      sysInfoResp.Uid,
		GID:      sysInfoResp.Gid,
		PID:      sysInfoResp.Pid,
		Locale:   sysInfoResp.Locale,
		ActiveC2: sysInfoResp.ActiveC2,
		ProxyURL: sysInfoResp.ProxyURL,
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
