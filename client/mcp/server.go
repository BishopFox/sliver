package mcp

import (
	"context"
	"log"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	listSessionsAndBeaconsToolName = "list_sessions_and_beacons"
)

type sessionSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Transport     string `json:"transport"`
	RemoteAddress string `json:"remote_address"`
	Hostname      string `json:"hostname"`
	Username      string `json:"username"`
	PID           int32  `json:"pid"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Locale        string `json:"locale"`
	LastCheckin   int64  `json:"last_checkin"`
	IsDead        bool   `json:"is_dead"`
	Integrity     string `json:"integrity"`
}

type beaconSummary struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Transport           string `json:"transport"`
	RemoteAddress       string `json:"remote_address"`
	Hostname            string `json:"hostname"`
	Username            string `json:"username"`
	PID                 int32  `json:"pid"`
	OS                  string `json:"os"`
	Arch                string `json:"arch"`
	Locale              string `json:"locale"`
	LastCheckin         int64  `json:"last_checkin"`
	NextCheckin         int64  `json:"next_checkin"`
	Interval            int64  `json:"interval"`
	Jitter              int64  `json:"jitter"`
	TasksCount          int64  `json:"tasks_count"`
	TasksCountCompleted int64  `json:"tasks_count_completed"`
	IsDead              bool   `json:"is_dead"`
	Integrity           string `json:"integrity"`
}

type listSessionsAndBeaconsResult struct {
	Sessions      []sessionSummary `json:"sessions"`
	SessionsCount int              `json:"sessions_count"`
	Beacons       []beaconSummary  `json:"beacons"`
	BeaconsCount  int              `json:"beacons_count"`
}

// SliverMCPServer wraps the MCP server with Sliver RPC access for handlers.
type SliverMCPServer struct {
	Rpc    rpcpb.SliverRPCClient
	server *mcpserver.MCPServer
	logger *log.Logger
}

func newServer(cfg Config, rpc rpcpb.SliverRPCClient, logger *log.Logger) *SliverMCPServer {
	base := mcpserver.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		mcpserver.WithToolCapabilities(false),
	)

	listSessionsAndBeaconsTool := mcpapi.NewTool(
		listSessionsAndBeaconsToolName,
		mcpapi.WithDescription("List active sessions and beacons."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithDestructiveHintAnnotation(false),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	lsTool := mcpapi.NewTool(
		lsToolName,
		mcpapi.WithDescription("List the contents of a remote directory."),
		mcpapi.WithInputSchema[lsArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithDestructiveHintAnnotation(false),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	cdTool := mcpapi.NewTool(
		cdToolName,
		mcpapi.WithDescription("Change the current directory on a remote session or beacon."),
		mcpapi.WithInputSchema[cdArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(false),
	)
	catTool := mcpapi.NewTool(
		catToolName,
		mcpapi.WithDescription("Download and return the contents of a remote file."),
		mcpapi.WithInputSchema[catArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithDestructiveHintAnnotation(false),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	pwdTool := mcpapi.NewTool(
		pwdToolName,
		mcpapi.WithDescription("Return the current working directory of a remote session or beacon."),
		mcpapi.WithInputSchema[pwdArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithDestructiveHintAnnotation(false),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	rmTool := mcpapi.NewTool(
		rmToolName,
		mcpapi.WithDescription("Remove a file or directory on the remote target."),
		mcpapi.WithInputSchema[rmArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	mvTool := mcpapi.NewTool(
		mvToolName,
		mcpapi.WithDescription("Move or rename a file or directory on the remote target."),
		mcpapi.WithInputSchema[mvArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	cpTool := mcpapi.NewTool(
		cpToolName,
		mcpapi.WithDescription("Copy a file on the remote target."),
		mcpapi.WithInputSchema[cpArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	mkdirTool := mcpapi.NewTool(
		mkdirToolName,
		mcpapi.WithDescription("Create a directory on the remote target."),
		mcpapi.WithInputSchema[mkdirArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	chmodTool := mcpapi.NewTool(
		chmodToolName,
		mcpapi.WithDescription("Change file or directory permissions on the remote target."),
		mcpapi.WithInputSchema[chmodArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	chownTool := mcpapi.NewTool(
		chownToolName,
		mcpapi.WithDescription("Change file or directory ownership on the remote target."),
		mcpapi.WithInputSchema[chownArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv := &SliverMCPServer{
		Rpc:    rpc,
		server: base,
		logger: logger,
	}
	srv.server.AddTool(listSessionsAndBeaconsTool, srv.listSessionsAndBeaconsHandler)
	srv.server.AddTool(lsTool, srv.lsHandler)
	srv.server.AddTool(cdTool, srv.cdHandler)
	srv.server.AddTool(catTool, srv.catHandler)
	srv.server.AddTool(pwdTool, srv.pwdHandler)
	srv.server.AddTool(rmTool, srv.rmHandler)
	srv.server.AddTool(mvTool, srv.mvHandler)
	srv.server.AddTool(cpTool, srv.cpHandler)
	srv.server.AddTool(mkdirTool, srv.mkdirHandler)
	srv.server.AddTool(chmodTool, srv.chmodHandler)
	srv.server.AddTool(chownTool, srv.chownHandler)
	return srv
}

func (s *SliverMCPServer) listSessionsAndBeaconsHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(listSessionsAndBeaconsToolName, "", "")

	sessionsResp, err := s.Rpc.GetSessions(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list sessions", err), nil
	}

	beaconsResp, err := s.Rpc.GetBeacons(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list beacons", err), nil
	}

	result := listSessionsAndBeaconsResult{
		Sessions: make([]sessionSummary, 0, len(sessionsResp.GetSessions())),
		Beacons:  make([]beaconSummary, 0, len(beaconsResp.GetBeacons())),
	}

	for _, session := range sessionsResp.GetSessions() {
		if session == nil {
			continue
		}
		result.Sessions = append(result.Sessions, sessionSummary{
			ID:            session.ID,
			Name:          session.Name,
			Transport:     session.Transport,
			RemoteAddress: session.RemoteAddress,
			Hostname:      session.Hostname,
			Username:      session.Username,
			PID:           session.PID,
			OS:            session.OS,
			Arch:          session.Arch,
			Locale:        session.Locale,
			LastCheckin:   session.LastCheckin,
			IsDead:        session.IsDead,
			Integrity:     session.Integrity,
		})
	}

	for _, beacon := range beaconsResp.GetBeacons() {
		if beacon == nil {
			continue
		}
		result.Beacons = append(result.Beacons, beaconSummary{
			ID:                  beacon.ID,
			Name:                beacon.Name,
			Transport:           beacon.Transport,
			RemoteAddress:       beacon.RemoteAddress,
			Hostname:            beacon.Hostname,
			Username:            beacon.Username,
			PID:                 beacon.PID,
			OS:                  beacon.OS,
			Arch:                beacon.Arch,
			Locale:              beacon.Locale,
			LastCheckin:         beacon.LastCheckin,
			NextCheckin:         beacon.NextCheckin,
			Interval:            beacon.Interval,
			Jitter:              beacon.Jitter,
			TasksCount:          beacon.TasksCount,
			TasksCountCompleted: beacon.TasksCountCompleted,
			IsDead:              beacon.IsDead,
			Integrity:           beacon.Integrity,
		})
	}

	result.SessionsCount = len(result.Sessions)
	result.BeaconsCount = len(result.Beacons)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
