package handlers

import (
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

const (
	maxTrackedExecuteChildren = 256
)

type executeChildEntry struct {
	PID       int32
	Path      string
	Args      []string
	Stdout    string
	Stderr    string
	StartTime int64

	Exited   bool
	ExitCode int32
	ExitTime int64
	Error    string

	cmd *exec.Cmd
}

var (
	executeChildrenMu      sync.RWMutex
	trackedExecuteChildren = map[int32]*executeChildEntry{}
)

func trimTrackedExecuteChildrenLocked() {
	if len(trackedExecuteChildren) <= maxTrackedExecuteChildren {
		return
	}

	// Prefer trimming completed entries first, oldest exit time first.
	exited := make([]*executeChildEntry, 0, len(trackedExecuteChildren))
	for _, entry := range trackedExecuteChildren {
		if entry.Exited {
			exited = append(exited, entry)
		}
	}
	sort.Slice(exited, func(i, j int) bool {
		return exited[i].ExitTime < exited[j].ExitTime
	})

	for len(trackedExecuteChildren) > maxTrackedExecuteChildren && 0 < len(exited) {
		delete(trackedExecuteChildren, exited[0].PID)
		exited = exited[1:]
	}
}

func startExecuteChild(cmd *exec.Cmd, track bool, path string, args []string, stdoutPath string, stderrPath string) (int32, error) {
	var (
		stdoutFile *os.File
		stderrFile *os.File
		err        error
	)

	startTime := time.Now().Unix()

	if stdoutPath != "" {
		stdoutFile, err = os.Create(stdoutPath)
		if err != nil {
			return 0, err
		}
		cmd.Stdout = stdoutFile
	}
	if stderrPath != "" {
		stderrFile, err = os.Create(stderrPath)
		if err != nil {
			if stdoutFile != nil {
				stdoutFile.Close()
			}
			return 0, err
		}
		cmd.Stderr = stderrFile
	}

	err = cmd.Start()
	if err != nil {
		if stdoutFile != nil {
			stdoutFile.Close()
		}
		if stderrFile != nil {
			stderrFile.Close()
		}
		return 0, err
	}

	var pid int32
	if cmd.Process != nil {
		pid = int32(cmd.Process.Pid)
	}

	if track && pid != 0 {
		entry := &executeChildEntry{
			PID:       pid,
			Path:      path,
			Args:      append([]string(nil), args...),
			Stdout:    stdoutPath,
			Stderr:    stderrPath,
			StartTime: startTime,
			cmd:       cmd,
		}
		executeChildrenMu.Lock()
		trackedExecuteChildren[pid] = entry
		trimTrackedExecuteChildrenLocked()
		executeChildrenMu.Unlock()
	}

	go func(pid int32, cmd *exec.Cmd, stdoutFile *os.File, stderrFile *os.File, track bool) {
		waitErr := cmd.Wait()

		exitCode := int32(0)
		waitErrStr := ""
		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				exitCode = int32(exitErr.ExitCode())
			} else {
				exitCode = -1
				waitErrStr = waitErr.Error()
			}
		}

		if track && pid != 0 {
			executeChildrenMu.Lock()
			entry := trackedExecuteChildren[pid]
			if entry != nil {
				entry.Exited = true
				entry.ExitCode = exitCode
				entry.ExitTime = time.Now().Unix()
				entry.Error = waitErrStr
				entry.cmd = nil
			}
			executeChildrenMu.Unlock()
		}

		if stdoutFile != nil {
			stdoutFile.Close()
		}
		if stderrFile != nil {
			stderrFile.Close()
		}
	}(pid, cmd, stdoutFile, stderrFile, track)

	return pid, nil
}

func snapshotExecuteChildren() []*sliverpb.ExecuteChild {
	executeChildrenMu.RLock()
	children := make([]*sliverpb.ExecuteChild, 0, len(trackedExecuteChildren))
	for _, entry := range trackedExecuteChildren {
		children = append(children, &sliverpb.ExecuteChild{
			Pid:       entry.PID,
			Path:      entry.Path,
			Args:      append([]string(nil), entry.Args...),
			StartTime: entry.StartTime,
			Exited:    entry.Exited,
			ExitCode:  entry.ExitCode,
			ExitTime:  entry.ExitTime,
			Stdout:    entry.Stdout,
			Stderr:    entry.Stderr,
			Error:     entry.Error,
		})
	}
	executeChildrenMu.RUnlock()

	sort.Slice(children, func(i, j int) bool {
		return children[i].StartTime < children[j].StartTime
	})
	return children
}

func executeChildrenHandler(data []byte, resp RPCResponse) {
	req := &sliverpb.ExecuteChildrenReq{}
	err := proto.Unmarshal(data, req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	execChildren := &sliverpb.ExecuteChildren{
		Children: snapshotExecuteChildren(),
		Response: &commonpb.Response{},
	}
	data, err = proto.Marshal(execChildren)
	resp(data, err)
}
