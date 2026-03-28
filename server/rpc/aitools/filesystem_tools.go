package aitools

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	serverai "github.com/bishopfox/sliver/server/ai"
)

type moveToolArgs struct {
	targetArgs
	Src string `json:"src,omitempty"`
	Dst string `json:"dst,omitempty"`
}

type copyToolArgs struct {
	targetArgs
	Src string `json:"src,omitempty"`
	Dst string `json:"dst,omitempty"`
}

type removeToolArgs struct {
	targetArgs
	Path      string `json:"path,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

type mkdirToolArgs struct {
	targetArgs
	Path string `json:"path,omitempty"`
}

type cdToolArgs struct {
	targetArgs
	Path string `json:"path,omitempty"`
}

type headTailToolArgs struct {
	targetArgs
	Path     string `json:"path,omitempty"`
	MaxBytes int64  `json:"max_bytes,omitempty"`
	MaxLines int64  `json:"max_lines,omitempty"`
}

type chmodToolArgs struct {
	targetArgs
	Path      string `json:"path,omitempty"`
	FileMode  string `json:"file_mode,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
}

type chownToolArgs struct {
	targetArgs
	Path      string `json:"path,omitempty"`
	UID       string `json:"uid,omitempty"`
	GID       string `json:"gid,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
}

type chtimesToolArgs struct {
	targetArgs
	Path      string `json:"path,omitempty"`
	ATimeUnix int64  `json:"atime_unix,omitempty"`
	MTimeUnix int64  `json:"mtime_unix,omitempty"`
}

type moveResult struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type copyResult struct {
	Src          string `json:"src"`
	Dst          string `json:"dst"`
	BytesWritten int64  `json:"bytes_written"`
}

type contentResult struct {
	Operation         string `json:"operation"`
	RequestedMaxBytes int64  `json:"requested_max_bytes,omitempty"`
	RequestedMaxLines int64  `json:"requested_max_lines,omitempty"`
	catResult
}

type chtimesResult struct {
	Path      string `json:"path"`
	ATimeUnix int64  `json:"atime_unix"`
	MTimeUnix int64  `json:"mtime_unix"`
}

type mountInfoResult struct {
	VolumeName   string `json:"volume_name"`
	VolumeType   string `json:"volume_type"`
	MountPoint   string `json:"mount_point"`
	Label        string `json:"label"`
	FileSystem   string `json:"file_system"`
	UsedSpace    uint64 `json:"used_space"`
	FreeSpace    uint64 `json:"free_space"`
	TotalSpace   uint64 `json:"total_space"`
	MountOptions string `json:"mount_options"`
}

type mountResult struct {
	Mounts []mountInfoResult `json:"mounts"`
	Count  int               `json:"count"`
}

func filesystemToolDefinitions() []serverai.AgenticToolDefinition {
	return []serverai.AgenticToolDefinition{
		{
			Name:        "fs_mv",
			Description: "Move or rename a remote file or directory. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"src": map[string]any{"type": "string", "description": "Source path to move."},
					"dst": map[string]any{"type": "string", "description": "Destination path."},
				}),
				"required":             []string{"src", "dst"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_cp",
			Description: "Copy a remote file. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"src": map[string]any{"type": "string", "description": "Source path to copy."},
					"dst": map[string]any{"type": "string", "description": "Destination path."},
				}),
				"required":             []string{"src", "dst"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_rm",
			Description: "Remove a remote file or directory. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path":      map[string]any{"type": "string", "description": "Path to remove."},
					"recursive": map[string]any{"type": "boolean", "description": "Recursively remove directories."},
					"force":     map[string]any{"type": "boolean", "description": "Force removal."},
				}),
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_mkdir",
			Description: "Create a remote directory. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path": map[string]any{"type": "string", "description": "Directory path to create."},
				}),
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_cd",
			Description: "Change the remote working directory. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path": map[string]any{"type": "string", "description": "Directory to change into. Defaults to . when omitted."},
				}),
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_head",
			Description: "Return the first lines or bytes from a remote file. Defaults to the first 10 lines when no limit is provided.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path":      map[string]any{"type": "string", "description": "Remote file path to read."},
					"max_bytes": map[string]any{"type": "integer", "description": "Optional number of bytes to read from the start of the file."},
					"max_lines": map[string]any{"type": "integer", "description": "Optional number of lines to read from the start of the file."},
				}),
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_tail",
			Description: "Return the last lines or bytes from a remote file. Defaults to the last 10 lines when no limit is provided.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path":      map[string]any{"type": "string", "description": "Remote file path to read."},
					"max_bytes": map[string]any{"type": "integer", "description": "Optional number of bytes to read from the end of the file."},
					"max_lines": map[string]any{"type": "integer", "description": "Optional number of lines to read from the end of the file."},
				}),
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_chmod",
			Description: "Change permissions on a remote file or directory. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path":      map[string]any{"type": "string", "description": "Path to change permissions on."},
					"file_mode": map[string]any{"type": "string", "description": "Mode string such as 0644 or 755."},
					"recursive": map[string]any{"type": "boolean", "description": "Apply recursively."},
				}),
				"required":             []string{"path", "file_mode"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_chown",
			Description: "Change ownership on a remote file or directory. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path":      map[string]any{"type": "string", "description": "Path to change ownership on."},
					"uid":       map[string]any{"type": "string", "description": "User ID to assign."},
					"gid":       map[string]any{"type": "string", "description": "Group ID to assign."},
					"recursive": map[string]any{"type": "boolean", "description": "Apply recursively."},
				}),
				"required":             []string{"path", "uid", "gid"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_chtimes",
			Description: "Change access and modification times on a remote file or directory using Unix timestamps in seconds.",
			Parameters: map[string]any{
				"type": "object",
				"properties": mergeSchemaProperties(map[string]any{
					"path":       map[string]any{"type": "string", "description": "Path to modify."},
					"atime_unix": map[string]any{"type": "integer", "description": "Access time as a Unix timestamp in seconds."},
					"mtime_unix": map[string]any{"type": "integer", "description": "Modification time as a Unix timestamp in seconds."},
				}),
				"required":             []string{"path", "atime_unix", "mtime_unix"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "fs_mount",
			Description: "List mounted filesystems or drives on the remote host. If session_id and beacon_id are both omitted, the conversation target is used when available.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           targetSchemaProperties(),
				"additionalProperties": false,
			},
		},
	}
}

func (e *executor) callMV(ctx context.Context, args moveToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Src == "" || args.Dst == "" {
		return "", fmt.Errorf("src and dst are required")
	}

	_, err = callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Mv, error) {
			return e.backend.Mv(callCtx, &sliverpb.MvReq{Request: req, Src: args.Src, Dst: args.Dst})
		},
		func() *sliverpb.Mv { return &sliverpb.Mv{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(moveResult{Src: args.Src, Dst: args.Dst})
}

func (e *executor) callCP(ctx context.Context, args copyToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Src == "" || args.Dst == "" {
		return "", fmt.Errorf("src and dst are required")
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Cp, error) {
			return e.backend.Cp(callCtx, &sliverpb.CpReq{Request: req, Src: args.Src, Dst: args.Dst})
		},
		func() *sliverpb.Cp { return &sliverpb.Cp{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(copyResult{Src: args.Src, Dst: args.Dst, BytesWritten: resp.BytesWritten})
}

func (e *executor) callRM(ctx context.Context, args removeToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Rm, error) {
			return e.backend.Rm(callCtx, &sliverpb.RmReq{
				Request:   req,
				Path:      args.Path,
				Recursive: args.Recursive,
				Force:     args.Force,
			})
		},
		func() *sliverpb.Rm { return &sliverpb.Rm{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(pathResult{Path: resp.Path})
}

func (e *executor) callMkdir(ctx context.Context, args mkdirToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Mkdir, error) {
			return e.backend.Mkdir(callCtx, &sliverpb.MkdirReq{Request: req, Path: args.Path})
		},
		func() *sliverpb.Mkdir { return &sliverpb.Mkdir{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(pathResult{Path: resp.Path})
}

func (e *executor) callCD(ctx context.Context, args cdToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Path == "" {
		args.Path = "."
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Pwd, error) {
			return e.backend.Cd(callCtx, &sliverpb.CdReq{Request: req, Path: args.Path})
		},
		func() *sliverpb.Pwd { return &sliverpb.Pwd{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(pathResult{Path: resp.Path})
}

func (e *executor) callHead(ctx context.Context, args headTailToolArgs) (string, error) {
	return e.callHeadTail(ctx, args, true)
}

func (e *executor) callTail(ctx context.Context, args headTailToolArgs) (string, error) {
	return e.callHeadTail(ctx, args, false)
}

func (e *executor) callHeadTail(ctx context.Context, args headTailToolArgs, head bool) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if args.MaxBytes < 0 || args.MaxLines < 0 {
		return "", fmt.Errorf("max_bytes and max_lines must be non-negative")
	}
	if args.MaxBytes != 0 && args.MaxLines != 0 {
		return "", fmt.Errorf("provide only one of max_bytes or max_lines")
	}
	if args.MaxBytes == 0 && args.MaxLines == 0 {
		args.MaxLines = 10
	}

	requestedBytes := args.MaxBytes
	requestedLines := args.MaxLines
	if !head {
		requestedBytes *= -1
		requestedLines *= -1
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Download, error) {
			return e.backend.Download(callCtx, &sliverpb.DownloadReq{
				Request:          req,
				RestrictedToFile: true,
				Path:             args.Path,
				MaxBytes:         requestedBytes,
				MaxLines:         requestedLines,
			})
		},
		func() *sliverpb.Download { return &sliverpb.Download{} },
	)
	if err != nil {
		return "", err
	}

	built, err := buildCatResult(resp, args.Path)
	if err != nil {
		return "", err
	}
	operation := "head"
	if !head {
		operation = "tail"
	}
	return marshalToolResult(contentResult{
		Operation:         operation,
		RequestedMaxBytes: args.MaxBytes,
		RequestedMaxLines: args.MaxLines,
		catResult:         *built,
	})
}

func (e *executor) callChmod(ctx context.Context, args chmodToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Path == "" || args.FileMode == "" {
		return "", fmt.Errorf("path and file_mode are required")
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Chmod, error) {
			return e.backend.Chmod(callCtx, &sliverpb.ChmodReq{
				Request:   req,
				Path:      args.Path,
				FileMode:  args.FileMode,
				Recursive: args.Recursive,
			})
		},
		func() *sliverpb.Chmod { return &sliverpb.Chmod{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(pathResult{Path: resp.Path})
}

func (e *executor) callChown(ctx context.Context, args chownToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Path == "" || args.UID == "" || args.GID == "" {
		return "", fmt.Errorf("path, uid, and gid are required")
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Chown, error) {
			return e.backend.Chown(callCtx, &sliverpb.ChownReq{
				Request:   req,
				Path:      args.Path,
				Uid:       args.UID,
				Gid:       args.GID,
				Recursive: args.Recursive,
			})
		},
		func() *sliverpb.Chown { return &sliverpb.Chown{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(pathResult{Path: resp.Path})
}

func (e *executor) callChtimes(ctx context.Context, args chtimesToolArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}
	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Chtimes, error) {
			return e.backend.Chtimes(callCtx, &sliverpb.ChtimesReq{
				Request: req,
				Path:    args.Path,
				ATime:   args.ATimeUnix,
				MTime:   args.MTimeUnix,
			})
		},
		func() *sliverpb.Chtimes { return &sliverpb.Chtimes{} },
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(chtimesResult{
		Path:      resp.Path,
		ATimeUnix: args.ATimeUnix,
		MTimeUnix: args.MTimeUnix,
	})
}

func (e *executor) callMount(ctx context.Context, args targetArgs) (string, error) {
	target, err := e.resolveTarget(args.SessionID, args.BeaconID)
	if err != nil {
		return "", err
	}

	resp, err := callTargetRPC(
		ctx,
		target,
		func(callCtx context.Context, req *commonpb.Request) (*sliverpb.Mount, error) {
			return e.backend.Mount(callCtx, &sliverpb.MountReq{Request: req})
		},
		func() *sliverpb.Mount { return &sliverpb.Mount{} },
	)
	if err != nil {
		return "", err
	}

	result := mountResult{
		Mounts: make([]mountInfoResult, 0, len(resp.Info)),
	}
	for _, mountInfo := range resp.Info {
		if mountInfo == nil {
			continue
		}
		result.Mounts = append(result.Mounts, mountInfoResult{
			VolumeName:   mountInfo.VolumeName,
			VolumeType:   mountInfo.VolumeType,
			MountPoint:   mountInfo.MountPoint,
			Label:        mountInfo.Label,
			FileSystem:   mountInfo.FileSystem,
			UsedSpace:    mountInfo.UsedSpace,
			FreeSpace:    mountInfo.FreeSpace,
			TotalSpace:   mountInfo.TotalSpace,
			MountOptions: mountInfo.MountOptions,
		})
	}
	result.Count = len(result.Mounts)
	return marshalToolResult(result)
}
