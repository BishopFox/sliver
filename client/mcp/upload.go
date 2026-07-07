package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	uploadFileToolName = "upload_file"
)

type uploadFileArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	LocalPath      string `json:"local_path,omitempty"`
	RemotePath     string `json:"remote_path"`
	DataBase64     string `json:"data_base64,omitempty"`
	Overwrite      bool   `json:"overwrite,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type uploadFileResult struct {
	LocalPath    string `json:"local_path,omitempty"`
	RemotePath   string `json:"remote_path"`
	BytesWritten int64  `json:"bytes_written"`
	Async        bool   `json:"async"`
	Operation    string `json:"operation,omitempty"`
	TaskID       string `json:"task_id,omitempty"`
	BeaconID     string `json:"beacon_id,omitempty"`
	State        string `json:"state,omitempty"`
}

func (s *SliverMCPServer) uploadFileHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args uploadFileArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.RemotePath == "" {
		return mcpapi.NewToolResultError("remote_path is required"), nil
	}

	if args.LocalPath == "" && args.DataBase64 == "" {
		return mcpapi.NewToolResultError("either local_path or data_base64 is required"), nil
	}

	return s.handleUploadFile(ctx, args)
}

func (s *SliverMCPServer) handleUploadFile(ctx context.Context, args uploadFileArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	extras := []string{fmt.Sprintf("remote_path=%q", args.RemotePath)}
	if args.LocalPath != "" {
		extras = append(extras, fmt.Sprintf("local_path=%q", args.LocalPath))
	}
	if args.Overwrite {
		extras = append(extras, "overwrite=true")
	}
	s.logToolCall(uploadFileToolName, args.SessionID, args.BeaconID, extras...)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	var data []byte
	if args.LocalPath != "" {
		// Read from local file
		fileData, err := os.ReadFile(args.LocalPath)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("failed to read local file: %v", err)), nil
		}
		data = fileData
	} else if args.DataBase64 != "" {
		// Decode base64 data
		decoded, err := base64.StdEncoding.DecodeString(args.DataBase64)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("failed to decode base64 data: %v", err)), nil
		}
		data = decoded
	}

	// Encode data for transmission
	encoder := new(encoders.Gzip)
	encodedData, err := encoder.Encode(data)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("failed to encode data: %v", err)), nil
	}

	uploadReq := &sliverpb.UploadReq{
		Request:  req,
		Path:     args.RemotePath,
		Data:     encodedData,
		Encoder:  "gzip",
		FileName: filepath.Base(args.RemotePath),
		Overwrite: args.Overwrite,
	}

	uploadResp, err := s.Rpc.Upload(ctx, uploadReq)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to upload file", err), nil
	}

	if uploadResp.Response != nil && uploadResp.Response.Err != "" {
		return mcpapi.NewToolResultError(uploadResp.Response.Err), nil
	}

	if isBeacon && uploadResp.Response != nil && uploadResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("upload_file", uploadResp.Response.TaskID, uploadResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Upload{}
		if err := s.waitForBeaconTaskResponse(ctx, uploadResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await upload task", err), nil
		}
		uploadResp = resolved
		if uploadResp.Response != nil && uploadResp.Response.Err != "" {
			return mcpapi.NewToolResultError(uploadResp.Response.Err), nil
		}
	}

	result := uploadFileResult{
		LocalPath:    args.LocalPath,
		RemotePath:   uploadResp.Path,
		BytesWritten: int64(len(data)),
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
