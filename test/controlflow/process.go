//go:build sliver_controlflow_e2e

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type managedProcess struct {
	cmd     *exec.Cmd
	done    chan struct{}
	err     error
	logPath string
}

func startProcess(path string, args []string, dir string, env []string, logPath string) (*managedProcess, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	prepareCommand(cmd)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	process := &managedProcess{
		cmd:     cmd,
		done:    make(chan struct{}),
		logPath: logPath,
	}
	go func() {
		process.err = cmd.Wait()
		_ = logFile.Close()
		close(process.done)
	}()
	return process, nil
}

func (process *managedProcess) stop() {
	select {
	case <-process.done:
		return
	default:
	}
	terminateProcessTree(process.cmd)
	select {
	case <-process.done:
		return
	case <-time.After(cleanupGraceTimeout):
	}
	killProcessTree(process.cmd)
	select {
	case <-process.done:
	case <-time.After(cleanupProcessTimeout):
		fmt.Fprintf(os.Stderr, "timed out cleaning up owned process group for %s (pid %d)\n", process.cmd.Path, process.cmd.Process.Pid)
	}
}

func (process *managedProcess) failure(message string) error {
	processErr := process.err
	if processErr == nil {
		processErr = errors.New("process exited")
	}
	return fmt.Errorf("%s: %w\n%s", message, processErr, readLogTail(process.logPath))
}

func readLogTail(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("(could not read %s: %v)", path, err)
	}
	if len(data) > processLogTailBytes {
		data = data[len(data)-processLogTailBytes:]
	}
	return strings.TrimSpace(string(data))
}
