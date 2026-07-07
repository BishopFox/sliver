package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	listPivotsToolName = "list_pivots"
)

type listPivotsArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type pivotListener struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	BindAddress string `json:"bind_address"`
	BindPort   int32  `json:"bind_port"`
	SessionID  string `json:"session_id"`
}

type listPivotsResult struct {
	Pivots      []pivotListener `json:"pivots"`
	PivotsCount int             `json:"pivots_count"`
}

func (s *SliverMCPServer) listPivotsHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args listPivotsArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	return s.handleListPivots(ctx, args)
}

func (s *SliverMCPServer) handleListPivots(ctx context.Context, args listPivotsArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(listPivotsToolName, args.SessionID, args.BeaconID)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	// Get all pivot listeners from the graph
	pivotGraph, err := s.Rpc.PivotGraph(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get pivot graph", err), nil
	}

	result := listPivotsResult{
		Pivots: make([]pivotListener, 0),
	}

	for _, session := range pivotGraph.GetSessions() {
		if session == nil {
			continue
		}
		for _, listener := range session.GetListeners() {
			if listener == nil {
				continue
			}
			result.Pivots = append(result.Pivots, pivotListener{
				ID:          listener.ID,
				Type:        listener.Type.String(),
				BindAddress: listener.BindAddress,
				BindPort:    listener.BindPort,
				SessionID:   session.Session.ID,
			})
		}
	}

	result.PivotsCount = len(result.Pivots)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
