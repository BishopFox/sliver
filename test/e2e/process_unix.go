//go:build darwin || linux

package e2e

import (
	"errors"
	"os/exec"
	"syscall"
)

type processTree struct {
	pgid int
}

func prepareCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func attachProcessTree(cmd *exec.Cmd) (processTree, error) {
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return processTree{}, err
	}
	if pgid != cmd.Process.Pid {
		return processTree{}, errors.New("child did not start as its own process group")
	}
	return processTree{pgid: pgid}, nil
}

func terminateProcessTree(tree processTree, _ *exec.Cmd) error {
	return signalProcessGroup(tree.pgid, syscall.SIGTERM)
}

func killProcessTree(tree processTree, _ *exec.Cmd) error {
	return signalProcessGroup(tree.pgid, syscall.SIGKILL)
}

func closeProcessTree(processTree) error {
	return nil
}

func signalProcessGroup(pgid int, signal syscall.Signal) error {
	err := syscall.Kill(-pgid, signal)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
