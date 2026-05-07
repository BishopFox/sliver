package aitools

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	serverai "github.com/bishopfox/sliver/server/ai"
)

type netstatToolArgs struct {
	targetArgs
	TCP       bool `json:"tcp,omitempty"`
	UDP       bool `json:"udp,omitempty"`
	IP4       bool `json:"ip4,omitempty"`
	IP6       bool `json:"ip6,omitempty"`
	Listening bool `json:"listening,omitempty"`
}

type psToolArgs struct {
	targetArgs
	FullInfo bool   `json:"full_info,omitempty"`
	PID      int32  `json:"pid,omitempty"`
	Exe      string `json:"exe,omitempty"`
	Owner    string `json:"owner,omitempty"`
}

type envToolArgs struct {
	targetArgs
	Name string `json:"name,omitempty"`
}

type execToolArgs struct {
	targetArgs
	Path           string            `json:"path,omitempty"`
	Args           []string          `json:"args,omitempty"`
	CaptureOutput  *bool             `json:"capture_output,omitempty"`
	Background     bool              `json:"background,omitempty"`
	Stdout         string            `json:"stdout,omitempty"`
	Stderr         string            `json:"stderr,omitempty"`
	EnvInheritance bool              `json:"env_inheritance,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	UseToken       bool              `json:"use_token,omitempty"`
	HideWindow     bool              `json:"hide_window,omitempty"`
	PPID           uint32            `json:"ppid,omitempty"`
}

type pingToolArgs struct {
	targetArgs
	Nonce *int32 `json:"nonce,omitempty"`
}

type screenshotToolArgs struct {
	targetArgs
	IncludeData bool `json:"include_data,omitempty"`
}

type ifconfigInterfaceResult struct {
	Index       int32    `json:"index"`
	Name        string   `json:"name"`
	MAC         string   `json:"mac,omitempty"`
	IPAddresses []string `json:"ip_addresses"`
}

type ifconfigResult struct {
	Interfaces []ifconfigInterfaceResult `json:"interfaces"`
	Count      int                       `json:"count"`
}

type processResult struct {
	PID          int32    `json:"pid"`
	PPID         int32    `json:"ppid"`
	Executable   string   `json:"executable"`
	Owner        string   `json:"owner,omitempty"`
	Architecture string   `json:"architecture,omitempty"`
	SessionID    int32    `json:"session_id,omitempty"`
	CmdLine      []string `json:"cmd_line,omitempty"`
}

type psResult struct {
	Processes []processResult `json:"processes"`
	Count     int             `json:"count"`
}

type sockAddrResult struct {
	IP   string `json:"ip"`
	Port uint32 `json:"port"`
}

type netstatEntryResult struct {
	LocalAddr  sockAddrResult `json:"local_addr"`
	RemoteAddr sockAddrResult `json:"remote_addr"`
	State      string         `json:"state"`
	UID        uint32         `json:"uid"`
	Protocol   string         `json:"protocol"`
	Process    *processResult `json:"process,omitempty"`
}

type netstatResult struct {
	Entries []netstatEntryResult `json:"entries"`
	Count   int                  `json:"count"`
}

type envVarResult struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type envResult struct {
	Variables []envVarResult `json:"variables"`
	Count     int            `json:"count"`
}

type targetInfoResult struct {
	TargetType          string `json:"target_type"`
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Hostname            string `json:"hostname"`
	UUID                string `json:"uuid"`
	Username            string `json:"username"`
	UID                 string `json:"uid"`
	GID                 string `json:"gid"`
	PID                 int32  `json:"pid"`
	OS                  string `json:"os"`
	Arch                string `json:"arch"`
	Transport           string `json:"transport"`
	ActiveC2            string `json:"active_c2"`
	RemoteAddress       string `json:"remote_address"`
	ProxyURL            string `json:"proxy_url"`
	Version             string `json:"version"`
	Locale              string `json:"locale"`
	Integrity           string `json:"integrity"`
	IsDead              bool   `json:"is_dead"`
	FirstContact        int64  `json:"first_contact"`
	LastCheckin         int64  `json:"last_checkin"`
	ReconnectInterval   int64  `json:"reconnect_interval"`
	Interval            int64  `json:"interval,omitempty"`
	Jitter              int64  `json:"jitter,omitempty"`
	NextCheckin         int64  `json:"next_checkin,omitempty"`
	TasksCount          int64  `json:"tasks_count,omitempty"`
	TasksCountCompleted int64  `json:"tasks_count_completed,omitempty"`
}

type pidResult struct {
	PID int32 `json:"pid"`
}

type uidResult struct {
	UID string `json:"uid"`
}

type pingResult struct {
	Nonce int32 `json:"nonce"`
}

type screenshotResult struct {
	ByteLen    int    `json:"byte_len"`
	SHA256     string `json:"sha256"`
	DataBase64 string `json:"data_base64,omitempty"`
}

type execResult struct {
	PID          uint32 `json:"pid"`
	Status       uint32 `json:"status"`
	StdoutLen    int    `json:"stdout_len"`
	StderrLen    int    `json:"stderr_len"`
	StdoutText   string `json:"stdout_text,omitempty"`
	StdoutBase64 string `json:"stdout_base64,omitempty"`
	StderrText   string `json:"stderr_text,omitempty"`
	StderrBase64 string `json:"stderr_base64,omitempty"`
}

func systemToolDefinitions() []serverai.AgenticToolDefinition {
	return []serverai.AgenticToolDefinition{
		{
			Name:        "exec_command",
			Description: "Execute a remote command and optionally capture stdout/stderr. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path":            map[string]any{"type": "string", "description": "Executable or command path."},
					"args":            map[string]any{"type": "array", "description": "Optional argument list.", "items": map[string]any{"type": "string"}},
					"capture_output":  map[string]any{"type": "boolean", "description": "Capture stdout/stderr inline. Defaults to true unless background is true."},
					"background":      map[string]any{"type": "boolean", "description": "Run the command in the background."},
					"stdout":          map[string]any{"type": "string", "description": "Optional remote path where stdout should be written."},
					"stderr":          map[string]any{"type": "string", "description": "Optional remote path where stderr should be written."},
					"env_inheritance": map[string]any{"type": "boolean", "description": "Inherit the target environment."},
					"env": map[string]any{
						"type":                 "object",
						"description":          "Optional environment variables to set for the process.",
						"additionalProperties": map[string]any{"type": "string"},
					},
					"use_token":   map[string]any{"type": "boolean", "description": "Windows only: use the current token."},
					"hide_window": map[string]any{"type": "boolean", "description": "Windows only: hide the window."},
					"ppid":        map[string]any{"type": "integer", "description": "Windows only: spoof the parent PID."},
				}),
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "ifconfig",
			Description: "List network interfaces and assigned IP addresses for the target.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           targetSchemaProperties(),
				"additionalProperties": false,
			},
		},
		{
			Name:        "netstat",
			Description: "List active network connections on the target.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"tcp":       map[string]any{"type": "boolean", "description": "Include TCP sockets."},
					"udp":       map[string]any{"type": "boolean", "description": "Include UDP sockets."},
					"ip4":       map[string]any{"type": "boolean", "description": "Include IPv4 sockets."},
					"ip6":       map[string]any{"type": "boolean", "description": "Include IPv6 sockets."},
					"listening": map[string]any{"type": "boolean", "description": "Only show listening sockets."},
				}),
				"additionalProperties": false,
			},
		},
		{
			Name:        "ps",
			Description: "List target processes. Set full_info when owner or command line details are needed.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"full_info": map[string]any{"type": "boolean", "description": "Request owner, architecture, session, and command line information."},
					"pid":       map[string]any{"type": "integer", "description": "Optional PID filter applied after retrieval."},
					"exe":       map[string]any{"type": "string", "description": "Optional substring filter on the executable name."},
					"owner":     map[string]any{"type": "string", "description": "Optional substring filter on the process owner."},
				}),
				"additionalProperties": false,
			},
		},
		{
			Name:        "env",
			Description: "Return environment variables for the target, or one variable when name is provided.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"name": map[string]any{"type": "string", "description": "Optional environment variable name to retrieve."},
				}),
				"additionalProperties": false,
			},
		},
		{
			Name:        "getpid",
			Description: "Return the current target process PID from session or beacon metadata.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           targetSchemaProperties(),
				"additionalProperties": false,
			},
		},
		{
			Name:        "getuid",
			Description: "Return the current target process UID from session or beacon metadata.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           targetSchemaProperties(),
				"additionalProperties": false,
			},
		},
		{
			Name:        "info",
			Description: "Return detailed session or beacon metadata for the selected target.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           targetSchemaProperties(),
				"additionalProperties": false,
			},
		},
		{
			Name:        "ping",
			Description: "Send a round-trip C2 ping to the target and return the echoed nonce.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"nonce": map[string]any{"type": "integer", "description": "Optional nonce to send. A random one is generated when omitted."},
				}),
				"additionalProperties": false,
			},
		},
		{
			Name:        "screenshot",
			Description: "Capture a screenshot from the target. Set include_data to true to include the PNG bytes as base64.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"include_data": map[string]any{"type": "boolean", "description": "Include the raw screenshot bytes as base64 in the result."},
				}),
				"additionalProperties": false,
			},
		},
	}
}

func (e *executor) callIfconfig(ctx context.Context, args targetArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Ifconfig, error) {
			return e.backend.Ifconfig(callCtx, &sliverpb.IfconfigReq{Request: req})
		},
		func() *sliverpb.Ifconfig { return &sliverpb.Ifconfig{} },
	)
	if err != nil {
		return "", err
	}

	result := ifconfigResult{
		Interfaces: make([]ifconfigInterfaceResult, 0, len(resp.NetInterfaces)),
	}
	for _, iface := range resp.NetInterfaces {
		if iface == nil {
			continue
		}
		result.Interfaces = append(result.Interfaces, ifconfigInterfaceResult{
			Index:       iface.Index,
			Name:        iface.Name,
			MAC:         iface.MAC,
			IPAddresses: append([]string(nil), iface.IPAddresses...),
		})
	}
	sort.Slice(result.Interfaces, func(i, j int) bool {
		return result.Interfaces[i].Index < result.Interfaces[j].Index
	})
	result.Count = len(result.Interfaces)
	return marshalToolResult(result)
}

func (e *executor) callNetstat(ctx context.Context, args netstatToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Netstat, error) {
			return e.backend.Netstat(callCtx, &sliverpb.NetstatReq{
				Request:   req,
				TCP:       args.TCP,
				UDP:       args.UDP,
				IP4:       args.IP4,
				IP6:       args.IP6,
				Listening: args.Listening,
			})
		},
		func() *sliverpb.Netstat { return &sliverpb.Netstat{} },
	)
	if err != nil {
		return "", err
	}

	result := netstatResult{
		Entries: make([]netstatEntryResult, 0, len(resp.Entries)),
	}
	for _, entry := range resp.Entries {
		if entry == nil {
			continue
		}
		resultEntry := netstatEntryResult{
			State:    entry.SkState,
			UID:      entry.UID,
			Protocol: entry.Protocol,
		}
		if entry.LocalAddr != nil {
			resultEntry.LocalAddr = sockAddrResult{IP: entry.LocalAddr.Ip, Port: entry.LocalAddr.Port}
		}
		if entry.RemoteAddr != nil {
			resultEntry.RemoteAddr = sockAddrResult{IP: entry.RemoteAddr.Ip, Port: entry.RemoteAddr.Port}
		}
		if entry.Process != nil {
			resultEntry.Process = processToResult(entry.Process)
		}
		result.Entries = append(result.Entries, resultEntry)
	}
	result.Count = len(result.Entries)
	return marshalToolResult(result)
}

func (e *executor) callPS(ctx context.Context, args psToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Ps, error) {
			return e.backend.Ps(callCtx, &sliverpb.PsReq{
				Request:  req,
				FullInfo: args.FullInfo,
			})
		},
		func() *sliverpb.Ps { return &sliverpb.Ps{} },
	)
	if err != nil {
		return "", err
	}

	processes := make([]processResult, 0, len(resp.Processes))
	for _, process := range resp.Processes {
		if process == nil {
			continue
		}
		if args.PID != 0 && process.Pid != args.PID {
			continue
		}
		if args.Exe != "" && !strings.Contains(strings.ToLower(process.Executable), strings.ToLower(args.Exe)) {
			continue
		}
		if args.Owner != "" && !strings.Contains(strings.ToLower(process.Owner), strings.ToLower(args.Owner)) {
			continue
		}
		processes = append(processes, *processToResult(process))
	}
	sort.Slice(processes, func(i, j int) bool {
		if processes[i].PID == processes[j].PID {
			return processes[i].PPID < processes[j].PPID
		}
		return processes[i].PID < processes[j].PID
	})

	return marshalToolResult(psResult{
		Processes: processes,
		Count:     len(processes),
	})
}

func (e *executor) callEnv(ctx context.Context, args envToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.EnvInfo, error) {
			return e.backend.GetEnv(callCtx, &sliverpb.EnvReq{
				Request: req,
				Name:    args.Name,
			})
		},
		func() *sliverpb.EnvInfo { return &sliverpb.EnvInfo{} },
	)
	if err != nil {
		return "", err
	}

	result := envResult{
		Variables: make([]envVarResult, 0, len(resp.Variables)),
	}
	for _, variable := range resp.Variables {
		if variable == nil {
			continue
		}
		result.Variables = append(result.Variables, envVarResult{Key: variable.Key, Value: variable.Value})
	}
	sort.Slice(result.Variables, func(i, j int) bool {
		return result.Variables[i].Key < result.Variables[j].Key
	})
	result.Count = len(result.Variables)
	return marshalToolResult(result)
}

func (e *executor) callGetPID(ctx context.Context, args targetArgs) (string, error) {
	session, beacon, err := e.lookupTargetMetadata(ctx, args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if session != nil {
		return marshalToolResult(pidResult{PID: session.PID})
	}
	return marshalToolResult(pidResult{PID: beacon.PID})
}

func (e *executor) callGetUID(ctx context.Context, args targetArgs) (string, error) {
	session, beacon, err := e.lookupTargetMetadata(ctx, args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if session != nil {
		return marshalToolResult(uidResult{UID: session.UID})
	}
	return marshalToolResult(uidResult{UID: beacon.UID})
}

func (e *executor) callInfo(ctx context.Context, args targetArgs) (string, error) {
	session, beacon, err := e.lookupTargetMetadata(ctx, args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if session != nil {
		return marshalToolResult(targetInfoResult{
			TargetType:        "session",
			ID:                session.ID,
			Name:              session.Name,
			Hostname:          session.Hostname,
			UUID:              session.UUID,
			Username:          session.Username,
			UID:               session.UID,
			GID:               session.GID,
			PID:               session.PID,
			OS:                session.OS,
			Arch:              session.Arch,
			Transport:         session.Transport,
			ActiveC2:          session.ActiveC2,
			RemoteAddress:     session.RemoteAddress,
			ProxyURL:          session.ProxyURL,
			Version:           session.Version,
			Locale:            session.Locale,
			Integrity:         session.Integrity,
			IsDead:            session.IsDead,
			FirstContact:      session.FirstContact,
			LastCheckin:       session.LastCheckin,
			ReconnectInterval: session.ReconnectInterval,
		})
	}
	return marshalToolResult(targetInfoResult{
		TargetType:          "beacon",
		ID:                  beacon.ID,
		Name:                beacon.Name,
		Hostname:            beacon.Hostname,
		UUID:                beacon.UUID,
		Username:            beacon.Username,
		UID:                 beacon.UID,
		GID:                 beacon.GID,
		PID:                 beacon.PID,
		OS:                  beacon.OS,
		Arch:                beacon.Arch,
		Transport:           beacon.Transport,
		ActiveC2:            beacon.ActiveC2,
		RemoteAddress:       beacon.RemoteAddress,
		ProxyURL:            beacon.ProxyURL,
		Version:             beacon.Version,
		Locale:              beacon.Locale,
		Integrity:           beacon.Integrity,
		IsDead:              beacon.IsDead,
		FirstContact:        beacon.FirstContact,
		LastCheckin:         beacon.LastCheckin,
		ReconnectInterval:   beacon.ReconnectInterval,
		Interval:            beacon.Interval,
		Jitter:              beacon.Jitter,
		NextCheckin:         beacon.NextCheckin,
		TasksCount:          beacon.TasksCount,
		TasksCountCompleted: beacon.TasksCountCompleted,
	})
}

func (e *executor) callPing(ctx context.Context, args pingToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	nonce := int32(time.Now().UnixNano() % 1_000_000)
	if args.Nonce != nil {
		nonce = *args.Nonce
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Ping, error) {
			return e.backend.Ping(callCtx, &sliverpb.Ping{
				Request: req,
				Nonce:   nonce,
			})
		},
		func() *sliverpb.Ping { return &sliverpb.Ping{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(pingResult{Nonce: resp.Nonce})
}

func (e *executor) callScreenshot(ctx context.Context, args screenshotToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Screenshot, error) {
			return e.backend.Screenshot(callCtx, &sliverpb.ScreenshotReq{Request: req})
		},
		func() *sliverpb.Screenshot { return &sliverpb.Screenshot{} },
	)
	if err != nil {
		return "", err
	}

	result := screenshotResult{
		ByteLen: len(resp.Data),
		SHA256:  sha256Hex(resp.Data),
	}
	if args.IncludeData {
		result.DataBase64 = base64.StdEncoding.EncodeToString(resp.Data)
	}
	return marshalToolResult(result)
}

func (e *executor) callExec(ctx context.Context, args execToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	captureOutput := !args.Background
	if args.CaptureOutput != nil {
		captureOutput = *args.CaptureOutput && !args.Background
	}

	var resp *sliverpb.Execute
	if args.UseToken || args.HideWindow || args.PPID != 0 {
		session, beacon, err := e.lookupTargetMetadata(ctx, args.SessionID, args.BeaconID)
		if err != nil {
			return "", err
		}
		targetOS := ""
		if session != nil {
			targetOS = session.OS
		} else if beacon != nil {
			targetOS = beacon.OS
		}
		if targetOS != "windows" {
			return "", fmt.Errorf("use_token, hide_window, and ppid are only supported on windows targets")
		}
		if args.EnvInheritance || len(args.Env) > 0 {
			return "", fmt.Errorf("env and env_inheritance are not supported with use_token, hide_window, or ppid")
		}

		resp, err = callTargetRPC(
			ctx,
			target,
			func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Execute, error) {
				return e.backend.ExecuteWindows(callCtx, &sliverpb.ExecuteWindowsReq{
					Request:    req,
					Path:       args.Path,
					Args:       args.Args,
					Output:     captureOutput,
					Background: args.Background,
					Stdout:     args.Stdout,
					Stderr:     args.Stderr,
					UseToken:   args.UseToken,
					HideWindow: args.HideWindow,
					PPid:       args.PPID,
				})
			},
			func() *sliverpb.Execute { return &sliverpb.Execute{} },
		)
	} else {
		resp, err = callTargetRPC(
			ctx,
			target,
			func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Execute, error) {
				return e.backend.Execute(callCtx, &sliverpb.ExecuteReq{
					Request:        req,
					Path:           args.Path,
					Args:           args.Args,
					Output:         captureOutput,
					Background:     args.Background,
					Stdout:         args.Stdout,
					Stderr:         args.Stderr,
					EnvInheritance: args.EnvInheritance,
					Env:            args.Env,
				})
			},
			func() *sliverpb.Execute { return &sliverpb.Execute{} },
		)
	}
	if err != nil {
		return "", err
	}

	stdoutText, stdoutBase64 := bytesToTextAndBase64(resp.Stdout)
	stderrText, stderrBase64 := bytesToTextAndBase64(resp.Stderr)

	return marshalToolResult(execResult{
		PID:          resp.Pid,
		Status:       resp.Status,
		StdoutLen:    len(resp.Stdout),
		StderrLen:    len(resp.Stderr),
		StdoutText:   stdoutText,
		StdoutBase64: stdoutBase64,
		StderrText:   stderrText,
		StderrBase64: stderrBase64,
	})
}

func processToResult(process *commonpb.Process) *processResult {
	if process == nil {
		return nil
	}
	return &processResult{
		PID:          process.Pid,
		PPID:         process.Ppid,
		Executable:   process.Executable,
		Owner:        process.Owner,
		Architecture: process.Architecture,
		SessionID:    process.SessionID,
		CmdLine:      append([]string(nil), process.CmdLine...),
	}
}
