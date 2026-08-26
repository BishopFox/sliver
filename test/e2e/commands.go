package e2e

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	e2ecoverage "github.com/bishopfox/sliver/test/e2e/coverage"
	"google.golang.org/protobuf/proto"
)

type implantCommandError struct {
	operation string
	message   string
}

func (err *implantCommandError) Error() string {
	return fmt.Sprintf("%s: %s", err.operation, err.message)
}

func invokeRPC[T proto.Message](
	s *suite,
	target implantTarget,
	operation string,
	call func(context.Context, *commonpb.Request) (T, error),
	response func(T) *commonpb.Response,
) (T, error) {
	var zero T
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	defer cancel()

	cursor := s.hub.cursor()
	result, err := call(ctx, target.request(s.opts.commandTimeout))
	if err != nil {
		return zero, err
	}
	value := reflect.ValueOf(result)
	if !value.IsValid() || (value.Kind() == reflect.Ptr && value.IsNil()) {
		return zero, fmt.Errorf("%s returned an empty response", operation)
	}

	metadata := response(result)
	if metadata != nil && metadata.Err != "" {
		return result, &implantCommandError{operation: operation, message: metadata.Err}
	}
	if target.beacon == nil {
		if metadata != nil && metadata.Async {
			return result, fmt.Errorf("%s unexpectedly returned an asynchronous session response", operation)
		}
		return result, nil
	}

	if metadata == nil || !metadata.Async || metadata.TaskID == "" {
		return result, fmt.Errorf("%s did not return beacon task metadata", operation)
	}
	if metadata.BeaconID != target.beacon.ID {
		return result, fmt.Errorf("%s beacon metadata got beacon %q, want %q", operation, metadata.BeaconID, target.beacon.ID)
	}
	taskID := metadata.TaskID
	_, _, err = s.hub.wait(ctx, cursor, func(event *clientpb.Event) bool {
		if event.EventType != constants.BeaconTaskResultEvent {
			return false
		}
		task := &clientpb.BeaconTask{}
		return proto.Unmarshal(event.Data, task) == nil && task.ID == taskID && task.BeaconID == target.beacon.ID
	})
	if err != nil {
		diagnosticCtx, diagnosticCancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer diagnosticCancel()
		task, fetchErr := s.rpc.GetBeaconTaskContent(diagnosticCtx, &clientpb.BeaconTask{ID: taskID})
		if fetchErr != nil {
			return result, fmt.Errorf("wait for %s beacon task %s: %w (fetch task state: %v)", operation, taskID, err, fetchErr)
		}
		return result, fmt.Errorf("wait for %s beacon task %s: %w (state=%q sent_at=%d completed_at=%d response_bytes=%d)", operation, taskID, err, task.State, task.SentAt, task.CompletedAt, len(task.Response))
	}
	task, err := s.rpc.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
	if err != nil {
		return result, fmt.Errorf("fetch %s beacon task %s: %w", operation, taskID, err)
	}
	if task == nil || task.ID != taskID || task.BeaconID != target.beacon.ID {
		return result, fmt.Errorf("%s beacon task correlation mismatch", operation)
	}
	if task.State != "completed" || task.CompletedAt <= 0 {
		return result, fmt.Errorf("%s beacon task incomplete: state=%q completed_at=%d response_bytes=%d", operation, task.State, task.CompletedAt, len(task.Response))
	}
	// A successful zero-valued protobuf message (for example Mv) is encoded as
	// an empty byte slice, so completion metadata is the authoritative result.
	proto.Reset(result)
	if err := proto.Unmarshal(task.Response, result); err != nil {
		return result, fmt.Errorf("decode %s beacon response: %w", operation, err)
	}
	metadata = response(result)
	if metadata != nil && metadata.Err != "" {
		return result, &implantCommandError{operation: operation, message: metadata.Err}
	}
	return result, nil
}

func (s *suite) step(target implantTarget, transport string, rpcName string, scenario string, fn func() error) error {
	started := time.Now()
	err := fn()
	duration := time.Since(started)
	status := e2ecoverage.StatusPass
	detail := ""
	if err != nil {
		status = e2ecoverage.StatusFail
		detail = err.Error()
	}
	recordErr := s.coverage.Add(e2ecoverage.Observation{
		Transport: transport, ImplantMode: target.mode(), GRPCMethod: rpcName, Scenario: scenario,
		Status: status, Duration: duration, Detail: detail,
	})
	if recordErr != nil {
		if err != nil {
			return errors.Join(err, fmt.Errorf("record E2E coverage: %w", recordErr))
		}
		return fmt.Errorf("record E2E coverage for %s/%s: %w", rpcName, scenario, recordErr)
	}
	if err != nil {
		return fmt.Errorf("%s (%s, %s/%s %s/%s, %s): %w", rpcName, scenario, s.opts.targetOS, s.opts.targetArch, transport, target.mode(), duration.Round(time.Millisecond), err)
	}
	s.t.Logf("PASS %s/%s %s %s %s (%s)", s.opts.targetOS, s.opts.targetArch, transport, target.mode(), rpcName+"/"+scenario, duration.Round(time.Millisecond))
	return nil
}

func (s *suite) localStep(name string, scenario string, fn func() error) error {
	started := time.Now()
	if err := fn(); err != nil {
		return fmt.Errorf("%s (%s, %s): %w", name, scenario, time.Since(started).Round(time.Millisecond), err)
	}
	s.t.Logf("PASS local %s/%s (%s)", name, scenario, time.Since(started).Round(time.Millisecond))
	return nil
}

func (s *suite) exerciseCommands(target implantTarget, remoteRoot string, transport string) error {
	var commandErrors []error
	commandErrors = appendIfError(commandErrors, s.exercisePing(target, transport))
	commandErrors = appendIfError(commandErrors, s.exerciseFilesystem(target, remoteRoot, transport))
	commandErrors = appendIfError(commandErrors, s.exerciseNetwork(target, transport))
	commandErrors = appendIfError(commandErrors, s.exerciseEnvironment(target, transport))
	commandErrors = appendIfError(commandErrors, s.exerciseProcesses(target, transport))
	commandErrors = appendIfError(commandErrors, s.exerciseMount(target, transport))
	commandErrors = appendIfError(commandErrors, s.exerciseWasmExtensions(target, transport))
	if s.opts.targetOS == "darwin" || s.opts.targetOS == "linux" {
		commandErrors = appendIfError(commandErrors, s.exerciseUnixCommands(target, remoteRoot, transport))
	}
	if s.opts.targetOS == "linux" {
		commandErrors = appendIfError(commandErrors, s.exerciseLinuxCommands(target, transport))
	}
	if s.opts.targetOS == "windows" {
		commandErrors = appendIfError(commandErrors, s.exerciseWindowsCommands(target, transport))
	}
	commandErrors = appendIfError(commandErrors, s.exerciseArmory(target, remoteRoot, transport))
	return errors.Join(commandErrors...)
}

func appendIfError(errs []error, err error) []error {
	if err != nil {
		return append(errs, err)
	}
	return errs
}

func (s *suite) exercisePing(target implantTarget, transport string) error {
	return s.step(target, transport, "Ping", "exact nonce round trip", func() error {
		const nonce = int32(0x5a17c0de)
		response, err := invokeRPC(s, target, "Ping", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ping, error) {
			return s.rpc.Ping(ctx, &sliverpb.Ping{Nonce: nonce, Request: request})
		}, func(response *sliverpb.Ping) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.Nonce != nonce {
			return fmt.Errorf("nonce mismatch: got %d, want %d", response.Nonce, nonce)
		}
		return nil
	})
}

func (s *suite) exerciseFilesystem(target implantTarget, root string, transport string) error {
	knownDir := filepath.Join(root, "known")
	nestedDir := filepath.Join(knownDir, "nested")
	seedPath := filepath.Join(knownDir, "seed.txt")
	childPath := filepath.Join(nestedDir, "child.txt")
	workDir := filepath.Join(root, "work", "one", "two")
	uploadDir := filepath.Join(root, "uploaded-tree")
	sentinelPath := filepath.Join(root, "outside-test-sentinel.txt")

	if err := s.step(target, transport, "Pwd", "initial working directory", func() error {
		response, err := invokeRPC(s, target, "Pwd", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Pwd, error) {
			return s.rpc.Pwd(ctx, &sliverpb.PwdReq{Request: request})
		}, func(response *sliverpb.Pwd) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if !samePath(response.Path, root) {
			return fmt.Errorf("got %q, want %q", response.Path, root)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Mkdir", "recursive nested directory", func() error {
		response, err := invokeRPC(s, target, "Mkdir", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Mkdir, error) {
			return s.rpc.Mkdir(ctx, &sliverpb.MkdirReq{Path: workDir, Request: request})
		}, func(response *sliverpb.Mkdir) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if !samePath(response.Path, workDir) {
			return fmt.Errorf("created path mismatch: got %q", response.Path)
		}
		stat, err := os.Stat(workDir)
		if err != nil || !stat.IsDir() {
			return fmt.Errorf("nested directory was not created: %v", err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Ls", "directory and wildcard metadata", func() error {
		directory, err := invokeRPC(s, target, "Ls", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ls, error) {
			return s.rpc.Ls(ctx, &sliverpb.LsReq{Path: knownDir, Request: request})
		}, func(response *sliverpb.Ls) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if !directory.Exists || !fileInfoNamed(directory.Files, "seed.txt") || !fileInfoNamed(directory.Files, "nested") {
			return fmt.Errorf("known directory listing missing expected entries: %+v", fileInfoNames(directory.Files))
		}
		wildcard, err := invokeRPC(s, target, "Ls", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ls, error) {
			return s.rpc.Ls(ctx, &sliverpb.LsReq{Path: filepath.Join(knownDir, "*.txt"), Request: request})
		}, func(response *sliverpb.Ls) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if len(wildcard.Files) != 1 || wildcard.Files[0].Name != "seed.txt" || wildcard.Files[0].Size != int64(len("alpha\nbeta\ngamma\n")) {
			return fmt.Errorf("unexpected wildcard listing: %+v", wildcard.Files)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Cd", "relative, parent, and rejected missing path", func() error {
		response, err := invokeRPC(s, target, "Cd", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Pwd, error) {
			return s.rpc.Cd(ctx, &sliverpb.CdReq{Path: filepath.Join("known", "nested"), Request: request})
		}, func(response *sliverpb.Pwd) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if !samePath(response.Path, nestedDir) {
			return fmt.Errorf("relative cd got %q, want %q", response.Path, nestedDir)
		}
		response, err = invokeRPC(s, target, "Cd", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Pwd, error) {
			return s.rpc.Cd(ctx, &sliverpb.CdReq{Path: "..", Request: request})
		}, func(response *sliverpb.Pwd) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if !samePath(response.Path, knownDir) {
			return fmt.Errorf("parent cd got %q, want %q", response.Path, knownDir)
		}
		_, missingErr := invokeRPC(s, target, "Cd", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Pwd, error) {
			return s.rpc.Cd(ctx, &sliverpb.CdReq{Path: filepath.Join(root, "does-not-exist"), Request: request})
		}, func(response *sliverpb.Pwd) *commonpb.Response { return response.GetResponse() })
		if missingErr == nil {
			return errors.New("cd to a missing path unexpectedly succeeded")
		}
		response, err = invokeRPC(s, target, "Cd", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Pwd, error) {
			return s.rpc.Cd(ctx, &sliverpb.CdReq{Path: root, Request: request})
		}, func(response *sliverpb.Pwd) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if !samePath(response.Path, root) {
			return fmt.Errorf("restore cd got %q, want %q", response.Path, root)
		}
		return nil
	}); err != nil {
		return err
	}

	uploadedPath := filepath.Join(root, "work", "uploaded.txt")
	if err := s.step(target, transport, "Upload", "gzip file and overwrite", func() error {
		first := []byte("first upload payload with more bytes\n")
		response, err := invokeRPC(s, target, "Upload", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Upload, error) {
			return s.rpc.Upload(ctx, &sliverpb.UploadReq{Path: filepath.Dir(uploadedPath), FileName: filepath.Base(uploadedPath), Data: gzipBytes(first), Encoder: "gzip", Overwrite: false, Request: request})
		}, func(response *sliverpb.Upload) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.WrittenFiles != 1 || response.UnwriteableFiles != 0 {
			return fmt.Errorf("unexpected first upload counts: %+v", response)
		}
		second := []byte("short\n")
		response, err = invokeRPC(s, target, "Upload", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Upload, error) {
			return s.rpc.Upload(ctx, &sliverpb.UploadReq{Path: uploadedPath, FileName: filepath.Base(uploadedPath), Data: gzipBytes(second), Encoder: "gzip", Overwrite: true, Request: request})
		}, func(response *sliverpb.Upload) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		data, err := os.ReadFile(uploadedPath)
		if err != nil {
			return err
		}
		if !bytes.Equal(data, second) {
			return fmt.Errorf("overwritten file mismatch: got %q", data)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Upload", "tar directory recursive overwrite truncation", func() error {
		firstFiles := map[string][]byte{
			"bundle/item.txt":         []byte("this is deliberately much longer than the replacement\n"),
			"bundle/nested/child.txt": []byte("directory-upload-child\n"),
		}
		response, err := invokeRPC(s, target, "Upload", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Upload, error) {
			return s.rpc.Upload(ctx, &sliverpb.UploadReq{Path: uploadDir, Data: gzipBytes(tarBytes(firstFiles)), Encoder: "gzip", IsDirectory: true, Overwrite: false, Request: request})
		}, func(response *sliverpb.Upload) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.WrittenFiles != int32(len(firstFiles)) {
			return fmt.Errorf("directory upload wrote %d files, want %d", response.WrittenFiles, len(firstFiles))
		}
		replacement := []byte("short\n")
		response, err = invokeRPC(s, target, "Upload", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Upload, error) {
			return s.rpc.Upload(ctx, &sliverpb.UploadReq{Path: uploadDir, Data: gzipBytes(tarBytes(map[string][]byte{"bundle/item.txt": replacement})), Encoder: "gzip", IsDirectory: true, Overwrite: true, Request: request})
		}, func(response *sliverpb.Upload) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Join(uploadDir, "bundle", "item.txt"))
		if err != nil {
			return err
		}
		if !bytes.Equal(data, replacement) {
			return fmt.Errorf("directory overwrite retained stale bytes: got %q, want %q", data, replacement)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Download", "file, byte/line limits, and recursive directory", func() error {
		full, err := s.download(target, &sliverpb.DownloadReq{Path: seedPath, RestrictedToFile: true})
		if err != nil {
			return err
		}
		if string(full) != "alpha\nbeta\ngamma\n" {
			return fmt.Errorf("full download mismatch: %q", full)
		}
		head, err := s.download(target, &sliverpb.DownloadReq{Path: seedPath, RestrictedToFile: true, MaxBytes: 5})
		if err != nil || string(head) != "alpha" {
			return fmt.Errorf("MaxBytes head mismatch: %q (%v)", head, err)
		}
		tail, err := s.download(target, &sliverpb.DownloadReq{Path: seedPath, RestrictedToFile: true, MaxBytes: -6})
		if err != nil || string(tail) != "gamma\n" {
			return fmt.Errorf("negative MaxBytes tail mismatch: %q (%v)", tail, err)
		}
		lines, err := s.download(target, &sliverpb.DownloadReq{Path: seedPath, RestrictedToFile: true, MaxLines: 2})
		if err != nil || string(lines) != "alpha\nbeta\n" {
			return fmt.Errorf("MaxLines mismatch: %q (%v)", lines, err)
		}
		tailLines, err := s.download(target, &sliverpb.DownloadReq{Path: seedPath, RestrictedToFile: true, MaxLines: -2})
		if err != nil || string(tailLines) != "gamma\n" {
			return fmt.Errorf("negative MaxLines tail mismatch: %q (%v)", tailLines, err)
		}
		directory, err := invokeRPC(s, target, "Download", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Download, error) {
			return s.rpc.Download(ctx, &sliverpb.DownloadReq{Path: knownDir, Recurse: true, Request: request})
		}, func(response *sliverpb.Download) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		archive, err := decodeDownload(directory)
		if err != nil {
			return err
		}
		entries, err := readTarGzip(archive)
		if err != nil {
			return err
		}
		expectedFiles := map[string]string{
			path.Join("known", "seed.txt"):              "alpha\nbeta\ngamma\n",
			path.Join("known", "nested", "child.txt"):   "child-marker\n",
			path.Join("known", "nested", "another.log"): "log-marker\n",
		}
		if len(entries) != len(expectedFiles) {
			return fmt.Errorf("recursive download returned %d regular files, want %d: %v", len(entries), len(expectedFiles), mapKeys(entries))
		}
		for suffix, expected := range expectedFiles {
			content, err := archiveContentBySuffix(entries, suffix)
			if err != nil {
				return err
			}
			if string(content) != expected {
				return fmt.Errorf("recursive download %s mismatch: got %q, want %q", suffix, content, expected)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Grep", "context and recursive regex", func() error {
		direct, err := invokeRPC(s, target, "Grep", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Grep, error) {
			return s.rpc.Grep(ctx, &sliverpb.GrepReq{SearchPattern: "beta", Path: seedPath, LinesBefore: 1, LinesAfter: 1, Request: request})
		}, func(response *sliverpb.Grep) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if !grepHasExactMatchWithContext(direct, seedPath, 2, "beta", []string{"alpha"}, []string{"gamma"}) {
			return fmt.Errorf("direct grep missing exact %s:2 beta match with requested context", seedPath)
		}
		recursive, err := invokeRPC(s, target, "Grep", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Grep, error) {
			return s.rpc.Grep(ctx, &sliverpb.GrepReq{SearchPattern: "child-(marker|missing)", Path: knownDir, Recursive: true, Request: request})
		}, func(response *sliverpb.Grep) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if !grepHasExactMatch(recursive, childPath, 1, "child-marker") {
			return fmt.Errorf("recursive grep missing exact %s:1 child-marker match", childPath)
		}
		return nil
	}); err != nil {
		return err
	}

	copiedPath := filepath.Join(root, "work", "copied.txt")
	movedPath := filepath.Join(root, "work", "moved.txt")
	if err := s.step(target, transport, "Cp", "copy exact bytes", func() error {
		response, err := invokeRPC(s, target, "Cp", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Cp, error) {
			return s.rpc.Cp(ctx, &sliverpb.CpReq{Src: childPath, Dst: copiedPath, Request: request})
		}, func(response *sliverpb.Cp) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.BytesWritten != int64(len("child-marker\n")) {
			return fmt.Errorf("copied %d bytes", response.BytesWritten)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "Mv", "rename within test root", func() error {
		_, err := invokeRPC(s, target, "Mv", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Mv, error) {
			return s.rpc.Mv(ctx, &sliverpb.MvReq{Src: copiedPath, Dst: movedPath, Request: request})
		}, func(response *sliverpb.Mv) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if _, err := os.Stat(copiedPath); !os.IsNotExist(err) {
			return fmt.Errorf("source still exists after move: %v", err)
		}
		data, err := os.ReadFile(movedPath)
		if err != nil || string(data) != "child-marker\n" {
			return fmt.Errorf("moved file mismatch: %q (%v)", data, err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Chtimes", "exact access and modification time", func() error {
		const unixTime = int64(1_700_000_123)
		_, err := invokeRPC(s, target, "Chtimes", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Chtimes, error) {
			return s.rpc.Chtimes(ctx, &sliverpb.ChtimesReq{Path: movedPath, ATime: unixTime, MTime: unixTime, Request: request})
		}, func(response *sliverpb.Chtimes) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		stat, err := os.Stat(movedPath)
		if err != nil {
			return err
		}
		if stat.ModTime().Unix() != unixTime {
			return fmt.Errorf("mtime mismatch: got %d", stat.ModTime().Unix())
		}
		accessTime, err := accessTimeUnix(stat)
		if err != nil {
			return err
		}
		if accessTime != unixTime {
			return fmt.Errorf("atime mismatch: got %d", accessTime)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Rm", "file then recursive directory with force", func() error {
		for _, candidate := range []string{movedPath, uploadDir, filepath.Join(root, "work")} {
			if err := ensureWithinRoot(root, candidate); err != nil {
				return err
			}
		}
		_, err := invokeRPC(s, target, "Rm", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Rm, error) {
			return s.rpc.Rm(ctx, &sliverpb.RmReq{Path: movedPath, Recursive: false, Force: false, Request: request})
		}, func(response *sliverpb.Rm) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if _, statErr := os.Stat(movedPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("file rm did not remove test file: %v", statErr)
		}
		_, err = invokeRPC(s, target, "Rm", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Rm, error) {
			return s.rpc.Rm(ctx, &sliverpb.RmReq{Path: uploadDir, Recursive: true, Force: true, Request: request})
		}, func(response *sliverpb.Rm) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if _, err := os.Stat(uploadDir); !os.IsNotExist(err) {
			return fmt.Errorf("recursive rm did not remove test directory: %v", err)
		}
		data, err := os.ReadFile(sentinelPath)
		if err != nil || string(data) != "must-survive\n" {
			return fmt.Errorf("sentinel outside deletion targets changed: %q (%v)", data, err)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *suite) exerciseNetwork(target implantTarget, transport string) error {
	fixture, err := newNetworkFixture()
	if err != nil {
		return err
	}
	defer fixture.close()

	if err := s.step(target, transport, "Ifconfig", "loopback interface and parseable addresses", func() error {
		response, err := invokeRPC(s, target, "Ifconfig", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ifconfig, error) {
			return s.rpc.Ifconfig(ctx, &sliverpb.IfconfigReq{Request: request})
		}, func(response *sliverpb.Ifconfig) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		loopback := false
		for _, iface := range response.NetInterfaces {
			for _, address := range iface.IPAddresses {
				if strings.TrimSpace(address) == "" {
					continue
				}
				ip := parseAddressIP(address)
				if ip == nil {
					return fmt.Errorf("interface %q returned unparseable address %q", iface.Name, address)
				}
				if ip.IsLoopback() {
					loopback = true
				}
			}
		}
		if !loopback {
			return fmt.Errorf("no loopback address in %+v", response.NetInterfaces)
		}
		return nil
	}); err != nil {
		return err
	}

	variants := []struct {
		name     string
		req      *sliverpb.NetstatReq
		port     uint32
		protocol string
		state    string
	}{
		{name: "TCP IPv4 listening", req: &sliverpb.NetstatReq{TCP: true, IP4: true, Listening: true}, port: uint32(fixture.listenPort), protocol: "tcp", state: "LISTEN"},
		{name: "TCP IPv4 established", req: &sliverpb.NetstatReq{TCP: true, IP4: true, Listening: false}, port: uint32(fixture.clientPort), protocol: "tcp", state: "ESTABLISHED"},
		{name: "UDP-only IPv4", req: &sliverpb.NetstatReq{UDP: true, IP4: true}, port: uint32(fixture.udpPort), protocol: "udp"},
	}
	for _, variant := range variants {
		variant := variant
		if err := s.step(target, transport, "Netstat", variant.name, func() error {
			response, err := invokeRPC(s, target, "Netstat", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Netstat, error) {
				variant.req.Request = request
				return s.rpc.Netstat(ctx, variant.req)
			}, func(response *sliverpb.Netstat) *commonpb.Response { return response.GetResponse() })
			if err != nil {
				return err
			}
			if !netstatContainsSocket(response.Entries, variant.port, variant.protocol, variant.state) {
				return fmt.Errorf("known %s/%s socket on port %d missing from %d entries", variant.protocol, variant.state, variant.port, len(response.Entries))
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *suite) exerciseEnvironment(target implantTarget, transport string) error {
	key := "SLIVER_E2E_MUTABLE_" + strings.ToUpper(target.mode())
	value := fmt.Sprintf("%s-%s-%s", transport, s.opts.targetArch, target.mode())
	if err := s.step(target, transport, "GetEnv", "full inherited environment", func() error {
		response, err := invokeRPC(s, target, "GetEnv", func(ctx context.Context, request *commonpb.Request) (*sliverpb.EnvInfo, error) {
			return s.rpc.GetEnv(ctx, &sliverpb.EnvReq{Request: request})
		}, func(response *sliverpb.EnvInfo) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if envValue(response.Variables, "SLIVER_E2E_MARKER") == "" || len(response.Variables) < 2 {
			return fmt.Errorf("inherited environment missing marker or unexpectedly small: %d variables", len(response.Variables))
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "SetEnv", "set unique process variable", func() error {
		_, err := invokeRPC(s, target, "SetEnv", func(ctx context.Context, request *commonpb.Request) (*sliverpb.SetEnv, error) {
			return s.rpc.SetEnv(ctx, &sliverpb.SetEnvReq{Variable: &commonpb.EnvVar{Key: key, Value: value}, Request: request})
		}, func(response *sliverpb.SetEnv) *commonpb.Response { return response.GetResponse() })
		return err
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "GetEnv", "named variable after set", func() error {
		response, err := invokeRPC(s, target, "GetEnv", func(ctx context.Context, request *commonpb.Request) (*sliverpb.EnvInfo, error) {
			return s.rpc.GetEnv(ctx, &sliverpb.EnvReq{Name: key, Request: request})
		}, func(response *sliverpb.EnvInfo) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if envValue(response.Variables, key) != value {
			return fmt.Errorf("got %q, want %q", envValue(response.Variables, key), value)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "UnsetEnv", "unset unique process variable", func() error {
		_, err := invokeRPC(s, target, "UnsetEnv", func(ctx context.Context, request *commonpb.Request) (*sliverpb.UnsetEnv, error) {
			return s.rpc.UnsetEnv(ctx, &sliverpb.UnsetEnvReq{Name: key, Request: request})
		}, func(response *sliverpb.UnsetEnv) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		response, err := invokeRPC(s, target, "GetEnv", func(ctx context.Context, request *commonpb.Request) (*sliverpb.EnvInfo, error) {
			return s.rpc.GetEnv(ctx, &sliverpb.EnvReq{Request: request})
		}, func(response *sliverpb.EnvInfo) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if envHasKey(response.Variables, key) {
			return fmt.Errorf("variable %q remained present after unset", key)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *suite) exerciseProcesses(target implantTarget, transport string) error {
	targetPID := int32(0)
	if target.session != nil {
		targetPID = target.session.PID
	} else if target.beacon != nil {
		targetPID = target.beacon.PID
	}
	for _, fullInfo := range []bool{false, true} {
		fullInfo := fullInfo
		if err := s.step(target, transport, "Ps", fmt.Sprintf("FullInfo=%t and implant PID", fullInfo), func() error {
			response, err := invokeRPC(s, target, "Ps", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ps, error) {
				return s.rpc.Ps(ctx, &sliverpb.PsReq{FullInfo: fullInfo, Request: request})
			}, func(response *sliverpb.Ps) *commonpb.Response { return response.GetResponse() })
			if err != nil {
				return err
			}
			for _, process := range response.Processes {
				if process.Pid == targetPID {
					if fullInfo && strings.TrimSpace(process.Executable) == "" {
						return errors.New("implant process full-info executable was empty")
					}
					if fullInfo && process.Architecture != "" && normalizeProcessArch(process.Architecture) != s.opts.targetArch {
						return fmt.Errorf("implant process architecture got %q, want %q", process.Architecture, s.opts.targetArch)
					}
					if fullInfo && s.opts.targetOS != "darwin" && process.Architecture == "" {
						return errors.New("implant process full-info architecture was empty")
					}
					return nil
				}
			}
			return fmt.Errorf("implant PID %d missing from %d processes", targetPID, len(response.Processes))
		}); err != nil {
			return err
		}
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate E2E helper executable: %w", err)
	}
	marker := fmt.Sprintf("%s-%s-%s", transport, target.mode(), target.id())
	if err := s.step(target, transport, "Execute", "captured stdout stderr status and explicit environment", func() error {
		response, err := invokeRPC(s, target, "Execute", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Execute, error) {
			return s.rpc.Execute(ctx, &sliverpb.ExecuteReq{
				Path: executable, Output: true, EnvInheritance: true,
				Env:     map[string]string{"SLIVER_E2E_HELPER": "sync", "SLIVER_E2E_EXEC_MARKER": marker},
				Request: request,
			})
		}, func(response *sliverpb.Execute) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.Status != 7 || !strings.Contains(string(response.Stdout), "stdout:"+marker) || !strings.Contains(string(response.Stderr), "stderr:"+marker) {
			return fmt.Errorf("unexpected execute result: status=%d stdout=%q stderr=%q", response.Status, response.Stdout, response.Stderr)
		}
		return nil
	}); err != nil {
		return err
	}

	var childPID int32
	if err := s.step(target, transport, "Execute", "tracked background child", func() error {
		response, err := invokeRPC(s, target, "Execute", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Execute, error) {
			return s.rpc.Execute(ctx, &sliverpb.ExecuteReq{
				Path: executable, Background: true, EnvInheritance: true,
				Env:     map[string]string{"SLIVER_E2E_HELPER": "child", "SLIVER_E2E_EXEC_MARKER": marker},
				Request: request,
			})
		}, func(response *sliverpb.Execute) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		childPID = int32(response.Pid)
		if childPID <= 1 {
			return fmt.Errorf("invalid background PID %d", childPID)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "ExecuteChildren", "find tracked live child", func() error {
		response, err := invokeRPC(s, target, "ExecuteChildren", func(ctx context.Context, request *commonpb.Request) (*sliverpb.ExecuteChildren, error) {
			return s.rpc.ExecuteChildren(ctx, &sliverpb.ExecuteChildrenReq{Request: request})
		}, func(response *sliverpb.ExecuteChildren) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		return verifyTrackedChild(response.Children, childPID, executable)
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Terminate", "kill only tracked test child", func() error {
		children, err := invokeRPC(s, target, "ExecuteChildren", func(ctx context.Context, request *commonpb.Request) (*sliverpb.ExecuteChildren, error) {
			return s.rpc.ExecuteChildren(ctx, &sliverpb.ExecuteChildrenReq{Request: request})
		}, func(response *sliverpb.ExecuteChildren) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return fmt.Errorf("re-verify child immediately before terminate: %w", err)
		}
		if err := verifyTrackedChild(children.Children, childPID, executable); err != nil {
			return fmt.Errorf("refuse terminate without exact live child match: %w", err)
		}
		response, err := invokeRPC(s, target, "Terminate", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Terminate, error) {
			return s.rpc.Terminate(ctx, &sliverpb.TerminateReq{Pid: childPID, Force: false, Request: request})
		}, func(response *sliverpb.Terminate) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.Pid != childPID {
			return fmt.Errorf("terminated PID %d, want %d", response.Pid, childPID)
		}
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			children, err := invokeRPC(s, target, "ExecuteChildren", func(ctx context.Context, request *commonpb.Request) (*sliverpb.ExecuteChildren, error) {
				return s.rpc.ExecuteChildren(ctx, &sliverpb.ExecuteChildrenReq{Request: request})
			}, func(response *sliverpb.ExecuteChildren) *commonpb.Response { return response.GetResponse() })
			if err != nil {
				return err
			}
			for _, child := range children.Children {
				if child.Pid == childPID && child.Exited {
					return nil
				}
			}
			time.Sleep(250 * time.Millisecond)
		}
		return fmt.Errorf("child PID %d was not recorded as exited", childPID)
	}); err != nil {
		return err
	}
	childPID = 0
	return nil
}

func normalizeProcessArch(architecture string) string {
	switch strings.ToLower(strings.TrimSpace(architecture)) {
	case "386", "i386", "i686", "x86":
		return "386"
	case "amd64", "x86_64", "x64":
		return "amd64"
	case "arm64", "aarch64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(architecture))
	}
}

func verifyTrackedChild(children []*sliverpb.ExecuteChild, pid int32, executable string) error {
	for _, child := range children {
		if child.Pid != pid {
			continue
		}
		if child.Exited {
			return fmt.Errorf("tracked child PID %d already exited", pid)
		}
		if !samePath(child.Path, executable) {
			return fmt.Errorf("tracked child PID %d path got %q, want %q", pid, child.Path, executable)
		}
		return nil
	}
	return fmt.Errorf("live child PID %d missing from %+v", pid, children)
}

func (s *suite) exerciseMount(target implantTarget, transport string) error {
	return s.step(target, transport, "Mount", "nonempty read-only mount inventory", func() error {
		response, err := invokeRPC(s, target, "Mount", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Mount, error) {
			return s.rpc.Mount(ctx, &sliverpb.MountReq{Request: request})
		}, func(response *sliverpb.Mount) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if len(response.Info) == 0 {
			return errors.New("mount inventory was empty")
		}
		return nil
	})
}

func (s *suite) exerciseWasmExtensions(target implantTarget, transport string) error {
	return s.step(target, transport, "ListWasmExtensions", "empty initial extension inventory", func() error {
		response, err := invokeRPC(s, target, "ListWasmExtensions", func(ctx context.Context, request *commonpb.Request) (*sliverpb.ListWasmExtensions, error) {
			return s.rpc.ListWasmExtensions(ctx, &sliverpb.ListWasmExtensionsReq{Request: request})
		}, func(response *sliverpb.ListWasmExtensions) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if len(response.Names) != 0 {
			return fmt.Errorf("fresh implant unexpectedly had registered WASM extensions: %v", response.Names)
		}
		return nil
	})
}

func (s *suite) exerciseUnixCommands(target implantTarget, root string, transport string) error {
	chmodRoot := filepath.Join(root, "known", "nested")
	if err := s.step(target, transport, "Chmod", "recursive mode change inside test root", func() error {
		_, err := invokeRPC(s, target, "Chmod", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Chmod, error) {
			return s.rpc.Chmod(ctx, &sliverpb.ChmodReq{Path: chmodRoot, FileMode: "0700", Recursive: true, Request: request})
		}, func(response *sliverpb.Chmod) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		stat, err := os.Stat(filepath.Join(chmodRoot, "child.txt"))
		if err != nil || stat.Mode().Perm() != 0o700 {
			return fmt.Errorf("recursive chmod mode=%v err=%v", statMode(stat), err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.step(target, transport, "Chown", "recursive no-op to current owner", func() error {
		current, err := user.Current()
		if err != nil {
			return err
		}
		group, err := user.LookupGroupId(current.Gid)
		if err != nil {
			return err
		}
		_, err = invokeRPC(s, target, "Chown", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Chown, error) {
			return s.rpc.Chown(ctx, &sliverpb.ChownReq{Path: chmodRoot, Uid: current.Username, Gid: group.Name, Recursive: true, Request: request})
		}, func(response *sliverpb.Chown) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		return filepath.WalkDir(chmodRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			uid, gid, err := fileOwnerIDs(info)
			if err != nil {
				return fmt.Errorf("stat ownership for %s: %w", path, err)
			}
			if uid != current.Uid || gid != group.Gid {
				return fmt.Errorf("recursive chown ownership for %s=%s:%s, want %s:%s", path, uid, gid, current.Uid, group.Gid)
			}
			return nil
		})
	}); err != nil {
		return err
	}
	return nil
}

func (s *suite) exerciseLinuxCommands(target implantTarget, transport string) error {
	var fd int64
	if err := s.step(target, transport, "MemfilesAdd", "create anonymous memfd", func() error {
		response, err := invokeRPC(s, target, "MemfilesAdd", func(ctx context.Context, request *commonpb.Request) (*sliverpb.MemfilesAdd, error) {
			return s.rpc.MemfilesAdd(ctx, &sliverpb.MemfilesAddReq{Request: request})
		}, func(response *sliverpb.MemfilesAdd) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		fd = response.Fd
		if fd < 3 {
			return fmt.Errorf("invalid memfd %d", fd)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "MemfilesList", "list exact anonymous memfd", func() error {
		response, err := invokeRPC(s, target, "MemfilesList", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ls, error) {
			return s.rpc.MemfilesList(ctx, &sliverpb.MemfilesListReq{Request: request})
		}, func(response *sliverpb.Ls) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		for _, file := range response.Files {
			if file.Name == strconv.FormatInt(fd, 10) && strings.Contains(file.Link, "memfd:") {
				return nil
			}
		}
		return fmt.Errorf("memfd %d missing from %+v", fd, response.Files)
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "MemfilesRm", "close only created anonymous memfd", func() error {
		response, err := invokeRPC(s, target, "MemfilesRm", func(ctx context.Context, request *commonpb.Request) (*sliverpb.MemfilesRm, error) {
			return s.rpc.MemfilesRm(ctx, &sliverpb.MemfilesRmReq{Fd: fd, Request: request})
		}, func(response *sliverpb.MemfilesRm) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.Fd != fd {
			return fmt.Errorf("removed fd %d, want %d", response.Fd, fd)
		}
		listed, err := invokeRPC(s, target, "MemfilesList", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ls, error) {
			return s.rpc.MemfilesList(ctx, &sliverpb.MemfilesListReq{Request: request})
		}, func(response *sliverpb.Ls) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		for _, file := range listed.Files {
			if file.Name == strconv.FormatInt(fd, 10) {
				return fmt.Errorf("removed memfd %d still present in descriptor inventory", fd)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *suite) exerciseWindowsCommands(target implantTarget, transport string) error {
	if err := s.step(target, transport, "CurrentTokenOwner", "nonempty current token identity", func() error {
		response, err := invokeRPC(s, target, "CurrentTokenOwner", func(ctx context.Context, request *commonpb.Request) (*sliverpb.CurrentTokenOwner, error) {
			return s.rpc.CurrentTokenOwner(ctx, &sliverpb.CurrentTokenOwnerReq{Request: request})
		}, func(response *sliverpb.CurrentTokenOwner) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if strings.TrimSpace(response.Output) == "" {
			return errors.New("current token owner was empty")
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "GetPrivs", "read-only process privilege inventory", func() error {
		response, err := invokeRPC(s, target, "GetPrivs", func(ctx context.Context, request *commonpb.Request) (*sliverpb.GetPrivs, error) {
			return s.rpc.GetPrivs(ctx, &sliverpb.GetPrivsReq{Request: request})
		}, func(response *sliverpb.GetPrivs) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.ProcessName == "" || len(response.PrivInfo) == 0 {
			return fmt.Errorf("incomplete privilege inventory: process=%q entries=%d", response.ProcessName, len(response.PrivInfo))
		}
		return nil
	}); err != nil {
		return err
	}
	var selectedService *sliverpb.ServiceDetails
	if err := s.step(target, transport, "Services", "read-only local service inventory", func() error {
		response, err := invokeRPC(s, target, "Services", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Services, error) {
			return s.rpc.Services(ctx, &sliverpb.ServicesReq{Request: request})
		}, func(response *sliverpb.Services) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		selectedService, err = selectServiceByName(response, "EventLog")
		if err != nil {
			return err
		}
		if response.Error != "" {
			s.t.Logf("Services returned %d usable entries with a partial-inventory warning: %s", len(response.Details), response.Error)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.step(target, transport, "ServiceDetail", "exact detail for inventoried service", func() error {
		response, err := invokeRPC(s, target, "ServiceDetail", func(ctx context.Context, request *commonpb.Request) (*sliverpb.ServiceDetail, error) {
			return s.rpc.ServiceDetail(ctx, &sliverpb.ServiceDetailReq{
				ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: selectedService.Name},
				Request:     request,
			})
		}, func(response *sliverpb.ServiceDetail) *commonpb.Response { return response.GetResponse() })
		if err != nil {
			return err
		}
		if response.Message != "" {
			return fmt.Errorf("service detail returned a partial-result warning: %s", response.Message)
		}
		if response.Detail == nil {
			return errors.New("service detail was empty")
		}
		if !proto.Equal(response.Detail, selectedService) {
			return fmt.Errorf("service detail mismatch: got %+v, want %+v", response.Detail, selectedService)
		}
		return nil
	}); err != nil {
		return err
	}
	return s.exerciseWindowsRegistry(target, transport)
}

func selectServiceByName(response *sliverpb.Services, serviceName string) (*sliverpb.ServiceDetails, error) {
	if response == nil {
		return nil, errors.New("service inventory response was empty")
	}
	if len(response.Details) == 0 {
		return nil, fmt.Errorf("service inventory contained no usable entries (warning=%q)", response.Error)
	}
	for _, detail := range response.Details {
		if detail != nil && strings.EqualFold(detail.Name, serviceName) {
			return proto.Clone(detail).(*sliverpb.ServiceDetails), nil
		}
	}
	return nil, fmt.Errorf("service inventory did not contain the stable %s service (warning=%q)", serviceName, response.Error)
}

func (s *suite) download(target implantTarget, requestData *sliverpb.DownloadReq) ([]byte, error) {
	response, err := invokeRPC(s, target, "Download", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Download, error) {
		requestData.Request = request
		return s.rpc.Download(ctx, requestData)
	}, func(response *sliverpb.Download) *commonpb.Response { return response.GetResponse() })
	if err != nil {
		return nil, err
	}
	return decodeDownload(response)
}

func decodeDownload(response *sliverpb.Download) ([]byte, error) {
	if response == nil || !response.Exists {
		return nil, errors.New("download response did not identify an existing path")
	}
	if response.Encoder == "" {
		return response.Data, nil
	}
	if response.Encoder != "gzip" {
		return nil, fmt.Errorf("unsupported download encoder %q", response.Encoder)
	}
	reader, err := gzip.NewReader(bytes.NewReader(response.Data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func gzipBytes(data []byte) []byte {
	buffer := bytes.NewBuffer(nil)
	writer := gzip.NewWriter(buffer)
	_, _ = writer.Write(data)
	_ = writer.Close()
	return buffer.Bytes()
}

func tarBytes(files map[string][]byte) []byte {
	buffer := bytes.NewBuffer(nil)
	writer := tar.NewWriter(buffer)
	names := mapKeys(files)
	for _, name := range names {
		data := files[name]
		header := &tar.Header{Name: filepath.ToSlash(name), Mode: 0o600, Size: int64(len(data)), Typeflag: tar.TypeReg}
		_ = writer.WriteHeader(header)
		_, _ = writer.Write(data)
	}
	_ = writer.Close()
	return buffer.Bytes()
}

func readTar(data []byte) (map[string][]byte, error) {
	result := map[string][]byte{}
	reader := tar.NewReader(bytes.NewReader(data))
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		name := strings.ReplaceAll(header.Name, "\\", "/")
		cleanName := path.Clean(name)
		if cleanName == "." || path.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, "../") {
			return nil, fmt.Errorf("archive contains unsafe path %q", header.Name)
		}
		if _, exists := result[cleanName]; exists {
			return nil, fmt.Errorf("archive contains duplicate path %q", cleanName)
		}
		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		result[cleanName] = content
	}
}

func fileInfoNamed(files []*sliverpb.FileInfo, name string) bool {
	for _, file := range files {
		if file.Name == name {
			return true
		}
	}
	return false
}

func fileInfoNames(files []*sliverpb.FileInfo) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name)
	}
	return names
}

func grepHasExactMatch(response *sliverpb.Grep, expectedPath string, expectedLine int64, marker string) bool {
	return grepHasExactMatchWithContext(response, expectedPath, expectedLine, marker, nil, nil)
}

func grepHasExactMatchWithContext(response *sliverpb.Grep, expectedPath string, expectedLine int64, marker string, before, after []string) bool {
	for resultPath, result := range response.Results {
		if !samePath(resultPath, expectedPath) {
			continue
		}
		for _, match := range result.FileResults {
			if match.LineNumber == expectedLine && len(match.Positions) > 0 && strings.Contains(match.Line, marker) && stringSlicesEqual(match.LinesBefore, before) && stringSlicesEqual(match.LinesAfter, after) {
				return true
			}
		}
	}
	return false
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func samePath(left string, right string) bool {
	normalize := func(path string) string {
		absolute, err := filepath.Abs(path)
		if err == nil {
			path = absolute
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			path = resolved
		}
		return filepath.Clean(path)
	}
	left = normalize(left)
	right = normalize(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func ensureWithinRoot(root string, candidate string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return err
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("refusing destructive request outside test root: %q", candidate)
	}
	return nil
}

func envValue(variables []*commonpb.EnvVar, key string) string {
	for _, variable := range variables {
		if strings.EqualFold(variable.Key, key) {
			return variable.Value
		}
	}
	return ""
}

func envHasKey(variables []*commonpb.EnvVar, key string) bool {
	for _, variable := range variables {
		if strings.EqualFold(variable.Key, key) {
			return true
		}
	}
	return false
}

func mapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slicesSort(keys)
	return keys
}

func slicesSort(values []string) {
	for index := 1; index < len(values); index++ {
		for cursor := index; cursor > 0 && values[cursor] < values[cursor-1]; cursor-- {
			values[cursor], values[cursor-1] = values[cursor-1], values[cursor]
		}
	}
}

func archiveContentBySuffix(entries map[string][]byte, suffix string) ([]byte, error) {
	suffix = path.Clean(strings.ReplaceAll(suffix, "\\", "/"))
	var content []byte
	found := false
	for name, data := range entries {
		if name == suffix || strings.HasSuffix(name, "/"+suffix) {
			if found {
				return nil, fmt.Errorf("archive contains duplicate relative suffix %q", suffix)
			}
			content = data
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("archive missing relative suffix %q; entries: %v", suffix, mapKeys(entries))
	}
	return content, nil
}

func parseAddressIP(address string) net.IP {
	if ip, _, err := net.ParseCIDR(address); err == nil {
		return ip
	}
	address = strings.Trim(address, "[]")
	if before, _, ok := strings.CutLast(address, "%"); ok {
		address = before
	}
	return net.ParseIP(address)
}

func netstatContainsSocket(entries []*sliverpb.SockTabEntry, port uint32, protocol, state string) bool {
	for _, entry := range entries {
		if !strings.EqualFold(entry.Protocol, protocol) || (state != "" && !strings.EqualFold(entry.SkState, state)) {
			continue
		}
		if (entry.LocalAddr != nil && entry.LocalAddr.Port == port) || (entry.RemoteAddr != nil && entry.RemoteAddr.Port == port) {
			return true
		}
	}
	return false
}

type networkFixture struct {
	listener   net.Listener
	client     net.Conn
	server     net.Conn
	udp        *net.UDPConn
	listenPort int
	clientPort int
	udpPort    int
}

func newNetworkFixture() (*networkFixture, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- conn
	}()
	client, err := net.Dial("tcp4", listener.Addr().String())
	if err != nil {
		listener.Close()
		return nil, err
	}
	var server net.Conn
	select {
	case server = <-accepted:
	case err := <-acceptErr:
		client.Close()
		listener.Close()
		return nil, err
	case <-time.After(5 * time.Second):
		client.Close()
		listener.Close()
		return nil, errors.New("timed out accepting network fixture connection")
	}
	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		server.Close()
		client.Close()
		listener.Close()
		return nil, err
	}
	return &networkFixture{
		listener:   listener,
		client:     client,
		server:     server,
		udp:        udp,
		listenPort: listener.Addr().(*net.TCPAddr).Port,
		clientPort: client.LocalAddr().(*net.TCPAddr).Port,
		udpPort:    udp.LocalAddr().(*net.UDPAddr).Port,
	}, nil
}

func (fixture *networkFixture) close() {
	if fixture.udp != nil {
		_ = fixture.udp.Close()
	}
	if fixture.server != nil {
		_ = fixture.server.Close()
	}
	if fixture.client != nil {
		_ = fixture.client.Close()
	}
	if fixture.listener != nil {
		_ = fixture.listener.Close()
	}
}

func statMode(stat os.FileInfo) os.FileMode {
	if stat == nil {
		return 0
	}
	return stat.Mode().Perm()
}
