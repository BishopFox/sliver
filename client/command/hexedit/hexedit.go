package hexedit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

const (
	beaconPollInterval = 1 * time.Second
)

// HexEditCmd downloads a remote file, opens it in a hex editor, and optionally uploads changes.
// HexEditCmd 下载远程文件，在十六进制编辑器中打开它，并可选择上传 changes.
func HexEditCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	remotePath := strings.TrimSpace(args[0])
	if remotePath == "" {
		con.PrintErrorf("Missing parameter: remote path\n")
		return
	}

	maxSizeStr, _ := cmd.Flags().GetString("max-size")
	maxSize, err := parseByteSize(maxSizeStr)
	if err != nil {
		con.PrintErrorf("Invalid --max-size %q: %s\n", maxSizeStr, err)
		return
	}
	if maxSize <= 0 {
		con.PrintErrorf("--max-size must be greater than zero\n")
		return
	}

	offset, _ := cmd.Flags().GetInt64("offset")
	if offset < 0 {
		con.PrintErrorf("--offset must be zero or positive\n")
		return
	}

	fetchSize := maxSize + 1
	if fetchSize < maxSize {
		fetchSize = maxSize
	}

	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Downloading %s ...", remotePath), ctrl)
	download, err := downloadRemote(ctx, con, cmd, remotePath, fetchSize)
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if download.Response != nil && download.Response.Err != "" {
		con.PrintErrorf("%s\n", download.Response.Err)
		return
	}
	if !download.Exists {
		con.PrintErrorf("Remote file does not exist: %s\n", remotePath)
		return
	}
	if download.IsDir {
		con.PrintErrorf("Remote path is a directory: %s\n", download.Path)
		return
	}

	data := download.Data
	if download.Encoder == "gzip" {
		data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("Decoding failed %s\n", err)
			return
		}
	}

	if int64(len(data)) > maxSize {
		con.PrintErrorf("Remote file size %s exceeds --max-size %s\n", util.ByteCountBinary(int64(len(data))), util.ByteCountBinary(maxSize))
		return
	}

	if offset > 0 {
		if len(data) == 0 {
			con.PrintErrorf("Offset %d out of range for empty file\n", offset)
			return
		}
		if offset >= int64(len(data)) {
			con.PrintErrorf("Offset %d out of range (file size %d bytes)\n", offset, len(data))
			return
		}
	}

	path := download.Path
	if path == "" {
		path = remotePath
	}

	for {
		result, err := runHexEditor(data, path, int(offset))
		if err != nil {
			con.PrintErrorf("Editor error: %s\n", err)
			return
		}
		data = result.Data

		if result.Action == actionNone {
			return
		}

		if !result.Modified {
			if result.Action == actionSaveQuit {
				con.PrintInfof("No changes to upload for %s\n", path)
			}
			return
		}

		switch result.Action {
		case actionSaveQuit:
			uploadNow, err := confirm("Upload changes to remote file (overwrite)?")
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			if !uploadNow {
				keepEditing, err := confirm("Continue editing?")
				if err != nil {
					con.PrintErrorf("%s\n", err)
					return
				}
				if keepEditing {
					continue
				}
				return
			}

			if err := uploadEditedContent(cmd, con, path, data); err != nil {
				con.PrintErrorf("%s\n", err)
			}
			return
		case actionQuit:
			if result.Force {
				return
			}
			discard, err := confirm("Discard changes?")
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			if discard {
				return
			}
		}
	}
}

func runHexEditor(data []byte, filename string, offset int) (editorResult, error) {
	model := newEditorModel(data, filename, offset)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return editorResult{}, err
	}
	editor, ok := finalModel.(*editorModel)
	if !ok {
		return editorResult{}, fmt.Errorf("unexpected editor state")
	}
	return editor.result(), nil
}

func downloadRemote(ctx context.Context, con *console.SliverClient, cmd *cobra.Command, path string, maxBytes int64) (*sliverpb.Download, error) {
	download, err := con.Rpc.Download(ctx, &sliverpb.DownloadReq{
		Request:          con.ActiveTarget.Request(cmd),
		Path:             path,
		RestrictedToFile: true,
		MaxBytes:         maxBytes,
	})
	if err != nil {
		return nil, err
	}
	if download.Response != nil && download.Response.Async {
		taskID := download.Response.TaskID
		download = &sliverpb.Download{}
		if err := waitForBeaconTaskResponse(ctx, con, taskID, download); err != nil {
			return nil, err
		}
	}
	return download, nil
}

func uploadEditedContent(cmd *cobra.Command, con *console.SliverClient, path string, data []byte) error {
	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()

	uploadData, err := new(encoders.Gzip).Encode(data)
	if err != nil {
		return err
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Uploading %s ...", path), ctrl)
	upload, err := con.Rpc.Upload(ctx, &sliverpb.UploadReq{
		Request:     con.ActiveTarget.Request(cmd),
		Path:        path,
		Data:        uploadData,
		Encoder:     "gzip",
		IsDirectory: false,
		Overwrite:   true,
	})
	if err == nil && upload.Response != nil && upload.Response.Async {
		taskID := upload.Response.TaskID
		upload = &sliverpb.Upload{}
		err = waitForBeaconTaskResponse(ctx, con, taskID, upload)
	}
	ctrl <- true
	<-ctrl

	if err != nil {
		return err
	}
	filesystem.PrintUpload(upload, con)
	return nil
}

func waitForBeaconTaskResponse(ctx context.Context, con *console.SliverClient, taskID string, out proto.Message) error {
	task, err := waitForBeaconTask(ctx, con, taskID)
	if err != nil {
		return err
	}
	state := strings.ToLower(task.State)
	if state != "completed" {
		if state == "" {
			state = "unknown"
		}
		return fmt.Errorf("beacon task %s %s", task.ID, state)
	}
	if len(task.Response) == 0 {
		return fmt.Errorf("beacon task %s returned empty response", task.ID)
	}
	return proto.Unmarshal(task.Response, out)
}

func waitForBeaconTask(ctx context.Context, con *console.SliverClient, taskID string) (*clientpb.BeaconTask, error) {
	ticker := time.NewTicker(beaconPollInterval)
	defer ticker.Stop()

	for {
		task, err := con.Rpc.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
		if err != nil {
			return nil, err
		}
		state := strings.ToLower(task.State)
		if state == "completed" || state == "failed" || state == "canceled" {
			return task, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func confirm(title string) (bool, error) {
	answer := false
	if err := forms.Confirm(title, &answer); err != nil {
		if errors.Is(err, forms.ErrUserAborted) {
			return false, nil
		}
		return false, err
	}
	return answer, nil
}

func parseByteSize(input string) (int64, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return 0, fmt.Errorf("size is required")
	}
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ToUpper(value)

	idx := 0
	for idx < len(value) {
		c := value[idx]
		if (c >= '0' && c <= '9') || c == '.' {
			idx++
			continue
		}
		break
	}
	if idx == 0 {
		return 0, fmt.Errorf("invalid size %q", input)
	}

	numStr := value[:idx]
	unitStr := strings.TrimSpace(value[idx:])
	if unitStr == "" {
		unitStr = "B"
	}

	multiplier := int64(1)
	switch unitStr {
	case "B":
		multiplier = 1
	case "K", "KB", "KIB":
		multiplier = 1024
	case "M", "MB", "MIB":
		multiplier = 1024 * 1024
	case "G", "GB", "GIB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB", "TIB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size unit %q", unitStr)
	}

	numVal, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q", input)
	}
	if numVal <= 0 {
		return 0, fmt.Errorf("size must be greater than zero")
	}

	size := int64(numVal * float64(multiplier))
	if size <= 0 {
		return 0, fmt.Errorf("size is out of range")
	}
	return size, nil
}
