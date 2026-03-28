package aitools

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

const aiToolDefaultWaitSeconds = int64(60)

// Backend captures the RPC methods the AI tool executor needs.
type Backend interface {
	GetSessions(context.Context, *commonpb.Empty) (*clientpb.Sessions, error)
	GetBeacons(context.Context, *commonpb.Empty) (*clientpb.Beacons, error)
	Ls(context.Context, *sliverpb.LsReq) (*sliverpb.Ls, error)
	Mv(context.Context, *sliverpb.MvReq) (*sliverpb.Mv, error)
	Cp(context.Context, *sliverpb.CpReq) (*sliverpb.Cp, error)
	Rm(context.Context, *sliverpb.RmReq) (*sliverpb.Rm, error)
	Mkdir(context.Context, *sliverpb.MkdirReq) (*sliverpb.Mkdir, error)
	Cd(context.Context, *sliverpb.CdReq) (*sliverpb.Pwd, error)
	Download(context.Context, *sliverpb.DownloadReq) (*sliverpb.Download, error)
	Pwd(context.Context, *sliverpb.PwdReq) (*sliverpb.Pwd, error)
	Chmod(context.Context, *sliverpb.ChmodReq) (*sliverpb.Chmod, error)
	Chown(context.Context, *sliverpb.ChownReq) (*sliverpb.Chown, error)
	Chtimes(context.Context, *sliverpb.ChtimesReq) (*sliverpb.Chtimes, error)
	Mount(context.Context, *sliverpb.MountReq) (*sliverpb.Mount, error)
	Ifconfig(context.Context, *sliverpb.IfconfigReq) (*sliverpb.Ifconfig, error)
	Netstat(context.Context, *sliverpb.NetstatReq) (*sliverpb.Netstat, error)
	Ps(context.Context, *sliverpb.PsReq) (*sliverpb.Ps, error)
	GetEnv(context.Context, *sliverpb.EnvReq) (*sliverpb.EnvInfo, error)
	Ping(context.Context, *sliverpb.Ping) (*sliverpb.Ping, error)
	Screenshot(context.Context, *sliverpb.ScreenshotReq) (*sliverpb.Screenshot, error)
	Execute(context.Context, *sliverpb.ExecuteReq) (*sliverpb.Execute, error)
	ExecuteWindows(context.Context, *sliverpb.ExecuteWindowsReq) (*sliverpb.Execute, error)
	ExecuteAssembly(context.Context, *sliverpb.ExecuteAssemblyReq) (*sliverpb.ExecuteAssembly, error)
	Sideload(context.Context, *sliverpb.SideloadReq) (*sliverpb.Sideload, error)
	SpawnDll(context.Context, *sliverpb.InvokeSpawnDllReq) (*sliverpb.SpawnDll, error)
	RegisterExtension(context.Context, *sliverpb.RegisterExtensionReq) (*sliverpb.RegisterExtension, error)
	ListExtensions(context.Context, *sliverpb.ListExtensionsReq) (*sliverpb.ListExtensions, error)
	CallExtension(context.Context, *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error)
}

type executor struct {
	backend      Backend
	conversation *clientpb.AIConversation
}

type targetArgs struct {
	SessionID string `json:"session_id,omitempty"`
	BeaconID  string `json:"beacon_id,omitempty"`
}

type lsToolArgs struct {
	targetArgs
	Path string `json:"path,omitempty"`
}

type catToolArgs struct {
	targetArgs
	Path     string `json:"path,omitempty"`
	MaxBytes int64  `json:"max_bytes,omitempty"`
	MaxLines int64  `json:"max_lines,omitempty"`
}

type lsFileEntry struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
	Mode    string `json:"mode"`
	Link    string `json:"link,omitempty"`
	UID     string `json:"uid,omitempty"`
	GID     string `json:"gid,omitempty"`
}

type lsResult struct {
	Path           string        `json:"path"`
	Exists         bool          `json:"exists"`
	Timezone       string        `json:"timezone"`
	TimezoneOffset int32         `json:"timezone_offset"`
	Files          []lsFileEntry `json:"files"`
}

type catResult struct {
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

type pathResult struct {
	Path string `json:"path"`
}

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

type sessionsAndBeaconsResult struct {
	Sessions      []sessionSummary `json:"sessions"`
	SessionsCount int              `json:"sessions_count"`
	Beacons       []beaconSummary  `json:"beacons"`
	BeaconsCount  int              `json:"beacons_count"`
}

type toolTarget struct {
	SessionID string
	BeaconID  string
}

// NewExecutor returns the Sliver-backed tool executor used by the AI agent loop.
func NewExecutor(backend Backend, conversation *clientpb.AIConversation) serverai.AgenticToolExecutor {
	return &executor{
		backend:      backend,
		conversation: conversation,
	}
}

func (e *executor) ToolDefinitions() []serverai.AgenticToolDefinition {
	definitions := []serverai.AgenticToolDefinition{
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
	definitions = append(definitions, filesystemToolDefinitions()...)
	definitions = append(definitions, systemToolDefinitions()...)
	definitions = append(definitions, packageToolDefinitions()...)
	return definitions
}

func (e *executor) CallTool(ctx context.Context, name string, arguments string) (string, error) {
	switch strings.TrimSpace(name) {
	case "list_sessions_and_beacons":
		return e.callListSessionsAndBeacons(ctx)
	case "fs_ls":
		var args lsToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callLS(ctx, args)
	case "fs_cat":
		var args catToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callCat(ctx, args)
	case "fs_pwd":
		var args targetArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callPWD(ctx, args)
	case "fs_mv":
		var args moveToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callMV(ctx, args)
	case "fs_cp":
		var args copyToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callCP(ctx, args)
	case "fs_rm":
		var args removeToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callRM(ctx, args)
	case "fs_mkdir":
		var args mkdirToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callMkdir(ctx, args)
	case "fs_cd":
		var args cdToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callCD(ctx, args)
	case "fs_head":
		var args headTailToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callHead(ctx, args)
	case "fs_tail":
		var args headTailToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callTail(ctx, args)
	case "fs_chmod":
		var args chmodToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callChmod(ctx, args)
	case "fs_chown":
		var args chownToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callChown(ctx, args)
	case "fs_chtimes":
		var args chtimesToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callChtimes(ctx, args)
	case "fs_mount":
		var args targetArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callMount(ctx, args)
	case "exec_command":
		var args execToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callExec(ctx, args)
	case "ifconfig":
		var args targetArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callIfconfig(ctx, args)
	case "netstat":
		var args netstatToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callNetstat(ctx, args)
	case "ps":
		var args psToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callPS(ctx, args)
	case "env":
		var args envToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callEnv(ctx, args)
	case "getpid":
		var args targetArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callGetPID(ctx, args)
	case "getuid":
		var args targetArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callGetUID(ctx, args)
	case "info":
		var args targetArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callInfo(ctx, args)
	case "ping":
		var args pingToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callPing(ctx, args)
	case "screenshot":
		var args screenshotToolArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", err
		}
		return e.callScreenshot(ctx, args)
	default:
		if result, handled, err := e.callPackageTool(ctx, name, arguments); handled {
			return result, err
		}
		return "", fmt.Errorf("unsupported AI tool %q", name)
	}
}

func (e *executor) callListSessionsAndBeacons(ctx context.Context) (string, error) {
	if e == nil || e.backend == nil {
		return "", fmt.Errorf("AI tool executor is unavailable")
	}

	sessionsResp, err := e.backend.GetSessions(ctx, &commonpb.Empty{})
	if err != nil {
		return "", err
	}

	beaconsResp, err := e.backend.GetBeacons(ctx, &commonpb.Empty{})
	if err != nil {
		return "", err
	}

	result := sessionsAndBeaconsResult{
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
	return marshalToolResult(result)
}

func (e *executor) callLS(ctx context.Context, args lsToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Path) == "" {
		args.Path = "."
	}

	req, isBeacon, callCtx, cancel, err := buildRequestContext(ctx, target)
	if err != nil {
		return "", err
	}
	if cancel != nil {
		defer cancel()
	}

	resp, err := e.backend.Ls(callCtx, &sliverpb.LsReq{
		Request: req,
		Path:    args.Path,
	})
	if err != nil {
		return "", err
	}
	if err := genericResponseError(resp.Response); err != nil {
		return "", err
	}
	if isBeacon && resp.Response != nil && resp.Response.Async {
		resolved := &sliverpb.Ls{}
		if err := waitForBeaconTaskResponse(callCtx, resp.Response.TaskID, resolved); err != nil {
			return "", err
		}
		resp = resolved
		if err := genericResponseError(resp.Response); err != nil {
			return "", err
		}
	}

	result := lsResult{
		Path:           resp.Path,
		Exists:         resp.Exists,
		Timezone:       resp.Timezone,
		TimezoneOffset: resp.TimezoneOffset,
		Files:          make([]lsFileEntry, 0, len(resp.Files)),
	}
	for _, fileInfo := range resp.Files {
		if fileInfo == nil {
			continue
		}
		result.Files = append(result.Files, lsFileEntry{
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
	return marshalToolResult(result)
}

func (e *executor) callCat(ctx context.Context, args catToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Path) == "" {
		return "", fmt.Errorf("path is required")
	}

	req, isBeacon, callCtx, cancel, err := buildRequestContext(ctx, target)
	if err != nil {
		return "", err
	}
	if cancel != nil {
		defer cancel()
	}

	resp, err := e.backend.Download(callCtx, &sliverpb.DownloadReq{
		Request:          req,
		RestrictedToFile: true,
		Path:             args.Path,
		MaxBytes:         args.MaxBytes,
		MaxLines:         args.MaxLines,
	})
	if err != nil {
		return "", err
	}
	if err := genericResponseError(resp.Response); err != nil {
		return "", err
	}
	if isBeacon && resp.Response != nil && resp.Response.Async {
		resolved := &sliverpb.Download{}
		if err := waitForBeaconTaskResponse(callCtx, resp.Response.TaskID, resolved); err != nil {
			return "", err
		}
		resp = resolved
		if err := genericResponseError(resp.Response); err != nil {
			return "", err
		}
	}

	result, err := buildCatResult(resp, args.Path)
	if err != nil {
		return "", err
	}
	return marshalToolResult(result)
}

func (e *executor) callPWD(ctx context.Context, args targetArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	req, isBeacon, callCtx, cancel, err := buildRequestContext(ctx, target)
	if err != nil {
		return "", err
	}
	if cancel != nil {
		defer cancel()
	}

	resp, err := e.backend.Pwd(callCtx, &sliverpb.PwdReq{Request: req})
	if err != nil {
		return "", err
	}
	if err := genericResponseError(resp.Response); err != nil {
		return "", err
	}
	if isBeacon && resp.Response != nil && resp.Response.Async {
		resolved := &sliverpb.Pwd{}
		if err := waitForBeaconTaskResponse(callCtx, resp.Response.TaskID, resolved); err != nil {
			return "", err
		}
		resp = resolved
		if err := genericResponseError(resp.Response); err != nil {
			return "", err
		}
	}

	return marshalToolResult(pathResult{Path: resp.Path})
}

func (e *executor) resolveTarget(sessionID, beaconID string) (toolTarget, error) {
	sessionID = strings.TrimSpace(sessionID)
	beaconID = strings.TrimSpace(beaconID)
	if sessionID != "" && beaconID != "" {
		return toolTarget{}, fmt.Errorf("provide only one of session_id or beacon_id")
	}
	if sessionID == "" && beaconID == "" && e != nil && e.conversation != nil {
		sessionID = strings.TrimSpace(e.conversation.GetTargetSessionID())
		beaconID = strings.TrimSpace(e.conversation.GetTargetBeaconID())
	}
	if sessionID == "" && beaconID == "" {
		return toolTarget{}, fmt.Errorf("session_id or beacon_id is required; call list_sessions_and_beacons or start a conversation from an active target")
	}
	return toolTarget{
		SessionID: sessionID,
		BeaconID:  beaconID,
	}, nil
}

func buildRequestContext(ctx context.Context, target toolTarget) (*commonpb.Request, bool, context.Context, context.CancelFunc, error) {
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

func decodeToolArgs(raw string, out any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "{}"
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	return nil
}

func marshalToolResult(result any) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func genericResponseError(resp *commonpb.Response) error {
	if resp == nil || strings.TrimSpace(resp.Err) == "" {
		return nil
	}
	return fmt.Errorf("%s", strings.TrimSpace(resp.Err))
}

func waitForBeaconTaskResponse(ctx context.Context, taskID string, out proto.Message) error {
	task, err := waitForBeaconTask(ctx, taskID)
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

func waitForBeaconTask(ctx context.Context, taskID string) (*clientpb.BeaconTask, error) {
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

func buildCatResult(download *sliverpb.Download, requestedPath string) (*catResult, error) {
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

	result := &catResult{
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
