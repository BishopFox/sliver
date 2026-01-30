package edit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

const beaconPollInterval = 1 * time.Second

// EditCmd downloads a remote text file, opens it in a TUI editor, and optionally uploads changes.
func EditCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	remotePath := strings.TrimSpace(args[0])
	if remotePath == "" {
		con.PrintErrorf("Missing parameter: remote path\n")
		return
	}

	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Downloading %s ...", remotePath), ctrl)
	download, err := downloadRemote(ctx, con, cmd, remotePath)
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
	if !isText(data) {
		con.PrintErrorf("Refusing to edit non-text file: %s\n", download.Path)
		return
	}

	path := download.Path
	if path == "" {
		path = remotePath
	}

	lineEnding := detectLineEnding(data)
	content := normalizeLineEndings(data)

	for {
		result, err := runEditor(content, path)
		if err != nil {
			con.PrintErrorf("Editor error: %s\n", err)
			return
		}
		content = result.Content

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

			uploadData := applyLineEnding(content, lineEnding)
			if err := uploadEditedContent(cmd, con, path, uploadData); err != nil {
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

func runEditor(content, filename string) (editorResult, error) {
	model := newEditorModel(content, filename)
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

func downloadRemote(ctx context.Context, con *console.SliverClient, cmd *cobra.Command, path string) (*sliverpb.Download, error) {
	download, err := con.Rpc.Download(ctx, &sliverpb.DownloadReq{
		Request:          con.ActiveTarget.Request(cmd),
		Path:             path,
		RestrictedToFile: true,
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

func detectLineEnding(data []byte) string {
	if strings.Contains(string(data), "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func normalizeLineEndings(data []byte) string {
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func applyLineEnding(content, lineEnding string) []byte {
	if lineEnding == "\r\n" {
		content = strings.ReplaceAll(content, "\n", "\r\n")
	}
	return []byte(content)
}

func isText(sample []byte) bool {
	const max = 1024
	if len(sample) > max {
		sample = sample[:max]
	}
	for i, c := range string(sample) {
		if i+utf8.UTFMax > len(sample) {
			break
		}
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' && c != '\r' {
			return false
		}
	}
	return true
}
