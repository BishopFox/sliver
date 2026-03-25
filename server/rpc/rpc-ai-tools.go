package rpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	serverai "github.com/bishopfox/sliver/server/ai"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"
)

const (
	aiToolDefaultWaitSeconds = int64(60)
)

type aiToolExecutor struct {
	rpc          *Server
	conversation *clientpb.AIConversation
}

type aiTargetArgs struct {
	SessionID string `json:"session_id,omitempty"`
	BeaconID  string `json:"beacon_id,omitempty"`
}

type aiLSToolArgs struct {
	aiTargetArgs
	Path string `json:"path,omitempty"`
}

type aiCatToolArgs struct {
	aiTargetArgs
	Path     string `json:"path,omitempty"`
	MaxBytes int64  `json:"max_bytes,omitempty"`
	MaxLines int64  `json:"max_lines,omitempty"`
}

type aiLSFileEntry struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
	Mode    string `json:"mode"`
	Link    string `json:"link,omitempty"`
	UID     string `json:"uid,omitempty"`
	GID     string `json:"gid,omitempty"`
}

type aiLSResult struct {
	Path           string          `json:"path"`
	Exists         bool            `json:"exists"`
	Timezone       string          `json:"timezone"`
	TimezoneOffset int32           `json:"timezone_offset"`
	Files          []aiLSFileEntry `json:"files"`
}

type aiCatResult struct {
	Path          string `json:"path"`
	RequestedPath string `json:"requested_path"`
	Exists        bool   `json:"exists"`
	IsDir         bool   `json:"is_dir"`
	Encoder       string `json:"encoder,omitempty"`
	Decoded       bool   `json:"decoded"`
	ByteLen       int    `json:"byte_len"`
	DataBase64    string `json:"data_base64"`
	Text          string `json:"text,omitempty"`
}

type aiPathResult struct {
	Path string `json:"path"`
}

type aiSessionSummary struct {
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

type aiBeaconSummary struct {
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

type aiSessionsAndBeaconsResult struct {
	Sessions      []aiSessionSummary `json:"sessions"`
	SessionsCount int                `json:"sessions_count"`
	Beacons       []aiBeaconSummary  `json:"beacons"`
	BeaconsCount  int                `json:"beacons_count"`
}

func newAIToolExecutor(rpc *Server, conversation *clientpb.AIConversation) *aiToolExecutor {
	return &aiToolExecutor{
		rpc:          rpc,
		conversation: conversation,
	}
}

func (e *aiToolExecutor) ToolDefinitions() []serverai.AgenticToolDefinition {
	return []serverai.AgenticToolDefinition{
		{
			Name:        "list_sessions_and_beacons",
			Description: "List active sessions and beacons so the model can discover valid target IDs before invoking filesystem tools.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_ls",
			Description: "List the contents of a remote directory. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Interactive session ID to run the command against."},
					"beacon_id":  map[string]any{"type": "string", "description": "Beacon ID to run the command against."},
					"path":       map[string]any{"type": "string", "description": "Directory path to list. Defaults to . when omitted."},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_cat",
			Description: "Download and return the contents of a remote file. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Interactive session ID to run the command against."},
					"beacon_id":  map[string]any{"type": "string", "description": "Beacon ID to run the command against."},
					"path":       map[string]any{"type": "string", "description": "Remote file path to download."},
					"max_bytes":  map[string]any{"type": "integer", "description": "Optional byte limit for the download."},
					"max_lines":  map[string]any{"type": "integer", "description": "Optional line limit for the download."},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_pwd",
			Description: "Return the current working directory of a remote session or beacon. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Interactive session ID to run the command against."},
					"beacon_id":  map[string]any{"type": "string", "description": "Beacon ID to run the command against."},
				},
				"additionalProperties": false,
			},
		},
	}
}

func (e *aiToolExecutor) CallTool(ctx context.Context, name string, arguments string) (string, error) {
	switch strings.TrimSpace(name) {
	case "list_sessions_and_beacons":
		return e.callListSessionsAndBeacons(ctx)
	case "fs_ls":
		var args aiLSToolArgs
		if err := decodeAIToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callLS(ctx, args)
	case "fs_cat":
		var args aiCatToolArgs
		if err := decodeAIToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callCat(ctx, args)
	case "fs_pwd":
		var args aiTargetArgs
		if err := decodeAIToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callPWD(ctx, args)
	default:
		return "", fmt.Errorf("unsupported AI tool %q", name)
	}
}

func (e *aiToolExecutor) callListSessionsAndBeacons(ctx context.Context) (string, error) {
	if e == nil || e.rpc == nil {
		return "", fmt.Errorf("AI tool executor is unavailable")
	}

	sessionsResp, err := e.rpc.GetSessions(ctx, &commonpb.Empty{})
	if err != nil {
		return "", err
	}

	beaconsResp, err := e.rpc.GetBeacons(ctx, &commonpb.Empty{})
	if err != nil {
		return "", err
	}

	result := aiSessionsAndBeaconsResult{
		Sessions: make([]aiSessionSummary, 0, len(sessionsResp.GetSessions())),
		Beacons:  make([]aiBeaconSummary, 0, len(beaconsResp.GetBeacons())),
	}

	for _, session := range sessionsResp.GetSessions() {
		if session == nil {
			continue
		}
		result.Sessions = append(result.Sessions, aiSessionSummary{
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
		result.Beacons = append(result.Beacons, aiBeaconSummary{
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
	return marshalAIToolResult(result)
}

func (e *aiToolExecutor) callLS(ctx context.Context, args aiLSToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Path) == "" {
		args.Path = "."
	}

	req, isBeacon, callCtx, cancel, err := e.buildRequestContext(ctx, target)
	if err != nil {
		return "", err
	}
	if cancel != nil {
		defer cancel()
	}

	resp, err := e.rpc.Ls(callCtx, &sliverpb.LsReq{
		Request: req,
		Path:    args.Path,
	})
	if err != nil {
		return "", err
	}
	if err := aiGenericResponseError(resp.Response); err != nil {
		return "", err
	}
	if isBeacon && resp.Response != nil && resp.Response.Async {
		resolved := &sliverpb.Ls{}
		if err := waitForAIBeaconTaskResponse(callCtx, resp.Response.TaskID, resolved); err != nil {
			return "", err
		}
		resp = resolved
		if err := aiGenericResponseError(resp.Response); err != nil {
			return "", err
		}
	}

	result := aiLSResult{
		Path:           resp.Path,
		Exists:         resp.Exists,
		Timezone:       resp.Timezone,
		TimezoneOffset: resp.TimezoneOffset,
		Files:          make([]aiLSFileEntry, 0, len(resp.Files)),
	}
	for _, fileInfo := range resp.Files {
		if fileInfo == nil {
			continue
		}
		result.Files = append(result.Files, aiLSFileEntry{
			Name:    fileInfo.Name,
			IsDir:   fileInfo.IsDir,
			Size:    fileInfo.Size,
			ModTime: fileInfo.ModTime,
			Mode:    fileInfo.Mode,
			Link:    fileInfo.Link,
			UID:     fileInfo.Uid,
			GID:     fileInfo.Gid,
		})
	}
	return marshalAIToolResult(result)
}

func (e *aiToolExecutor) callCat(ctx context.Context, args aiCatToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Path) == "" {
		return "", fmt.Errorf("path is required")
	}

	req, isBeacon, callCtx, cancel, err := e.buildRequestContext(ctx, target)
	if err != nil {
		return "", err
	}
	if cancel != nil {
		defer cancel()
	}

	resp, err := e.rpc.Download(callCtx, &sliverpb.DownloadReq{
		Request:          req,
		RestrictedToFile: true,
		Path:             args.Path,
		MaxBytes:         args.MaxBytes,
		MaxLines:         args.MaxLines,
	})
	if err != nil {
		return "", err
	}
	if err := aiGenericResponseError(resp.Response); err != nil {
		return "", err
	}
	if isBeacon && resp.Response != nil && resp.Response.Async {
		resolved := &sliverpb.Download{}
		if err := waitForAIBeaconTaskResponse(callCtx, resp.Response.TaskID, resolved); err != nil {
			return "", err
		}
		resp = resolved
		if err := aiGenericResponseError(resp.Response); err != nil {
			return "", err
		}
	}

	result, err := buildAICatResult(resp, args.Path)
	if err != nil {
		return "", err
	}
	return marshalAIToolResult(result)
}

func (e *aiToolExecutor) callPWD(ctx context.Context, args aiTargetArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	req, isBeacon, callCtx, cancel, err := e.buildRequestContext(ctx, target)
	if err != nil {
		return "", err
	}
	if cancel != nil {
		defer cancel()
	}

	resp, err := e.rpc.Pwd(callCtx, &sliverpb.PwdReq{Request: req})
	if err != nil {
		return "", err
	}
	if err := aiGenericResponseError(resp.Response); err != nil {
		return "", err
	}
	if isBeacon && resp.Response != nil && resp.Response.Async {
		resolved := &sliverpb.Pwd{}
		if err := waitForAIBeaconTaskResponse(callCtx, resp.Response.TaskID, resolved); err != nil {
			return "", err
		}
		resp = resolved
		if err := aiGenericResponseError(resp.Response); err != nil {
			return "", err
		}
	}

	return marshalAIToolResult(aiPathResult{Path: resp.Path})
}

type aiToolTarget struct {
	SessionID string
	BeaconID  string
}

func (e *aiToolExecutor) resolveTarget(sessionID, beaconID string) (aiToolTarget, error) {
	sessionID = strings.TrimSpace(sessionID)
	beaconID = strings.TrimSpace(beaconID)
	if sessionID != "" && beaconID != "" {
		return aiToolTarget{}, fmt.Errorf("provide only one of session_id or beacon_id")
	}
	if sessionID == "" && beaconID == "" && e != nil && e.conversation != nil {
		sessionID = strings.TrimSpace(e.conversation.GetTargetSessionID())
		beaconID = strings.TrimSpace(e.conversation.GetTargetBeaconID())
	}
	if sessionID == "" && beaconID == "" {
		return aiToolTarget{}, fmt.Errorf("session_id or beacon_id is required; call list_sessions_and_beacons or start a conversation from an active target")
	}
	return aiToolTarget{
		SessionID: sessionID,
		BeaconID:  beaconID,
	}, nil
}

func (e *aiToolExecutor) buildRequestContext(ctx context.Context, target aiToolTarget) (*commonpb.Request, bool, context.Context, context.CancelFunc, error) {
	if strings.TrimSpace(target.SessionID) != "" && strings.TrimSpace(target.BeaconID) != "" {
		return nil, false, nil, nil, fmt.Errorf("provide only one of session_id or beacon_id")
	}

	timeoutSeconds := aiToolDefaultWaitSeconds
	req := &commonpb.Request{
		Timeout: (int64(time.Second) * timeoutSeconds) - 1,
	}
	isBeacon := false

	if strings.TrimSpace(target.SessionID) != "" {
		req.SessionID = strings.TrimSpace(target.SessionID)
		req.Async = false
	} else {
		req.BeaconID = strings.TrimSpace(target.BeaconID)
		req.Async = true
		isBeacon = true
	}

	callCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	return req, isBeacon, callCtx, cancel, nil
}

func decodeAIToolArgs(raw string, out any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "{}"
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	return nil
}

func marshalAIToolResult(result any) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func aiGenericResponseError(resp *commonpb.Response) error {
	if resp == nil || strings.TrimSpace(resp.Err) == "" {
		return nil
	}
	return fmt.Errorf("%s", strings.TrimSpace(resp.Err))
}

func waitForAIBeaconTaskResponse(ctx context.Context, taskID string, out proto.Message) error {
	task, err := waitForAIBeaconTask(ctx, taskID)
	if err != nil {
		return err
	}
	state := strings.ToLower(strings.TrimSpace(task.GetState()))
	if state != "completed" {
		if state == "" {
			state = "unknown"
		}
		return fmt.Errorf("beacon task %s %s", task.GetID(), state)
	}
	if len(task.GetResponse()) == 0 {
		return fmt.Errorf("beacon task %s returned empty response", task.GetID())
	}
	return proto.Unmarshal(task.GetResponse(), out)
}

func waitForAIBeaconTask(ctx context.Context, taskID string) (*clientpb.BeaconTask, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		task, err := db.BeaconTaskByID(taskID)
		if err != nil {
			return nil, err
		}
		switch strings.ToLower(strings.TrimSpace(task.GetState())) {
		case "completed", "failed", "canceled":
			return task, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func buildAICatResult(download *sliverpb.Download, requestedPath string) (*aiCatResult, error) {
	data := download.Data
	decoded := false
	if download.Encoder == "gzip" {
		decoded = true
		var err error
		data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			return nil, err
		}
	}

	result := &aiCatResult{
		Path:          download.Path,
		RequestedPath: requestedPath,
		Exists:        download.Exists,
		IsDir:         download.IsDir,
		Encoder:       download.Encoder,
		Decoded:       decoded,
		ByteLen:       len(data),
		DataBase64:    base64.StdEncoding.EncodeToString(data),
	}
	if len(data) == 0 {
		return result, nil
	}
	if utf8.Valid(data) {
		result.Text = string(data)
	}
	return result, nil
}
