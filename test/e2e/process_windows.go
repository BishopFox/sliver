//go:build windows

package e2e

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processTree struct {
	job windows.Handle
}

func prepareCommand(_ *exec.Cmd) {}

func attachProcessTree(cmd *exec.Cmd) (processTree, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return processTree{}, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return processTree{}, fmt.Errorf("set kill-on-close job limit: %w", err)
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return processTree{}, err
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		_ = windows.CloseHandle(job)
		return processTree{}, err
	}
	return processTree{job: job}, nil
}

func killPreparedProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

func terminateProcessTree(tree processTree, cmd *exec.Cmd) error {
	err := exec.Command("taskkill.exe", "/PID", strconv.Itoa(cmd.Process.Pid), "/T").Run()
	if err == nil {
		return nil
	}
	if tree.job == 0 {
		return err
	}
	if forceErr := windows.TerminateJobObject(tree.job, 1); forceErr != nil {
		return errors.Join(fmt.Errorf("taskkill: %w", err), fmt.Errorf("terminate job: %w", forceErr))
	}
	return nil
}

func killProcessTree(tree processTree, _ *exec.Cmd) error {
	if tree.job == 0 {
		return nil
	}
	return windows.TerminateJobObject(tree.job, 1)
}

func closeProcessTree(tree processTree) error {
	if tree.job == 0 {
		return nil
	}
	return windows.CloseHandle(tree.job)
}
