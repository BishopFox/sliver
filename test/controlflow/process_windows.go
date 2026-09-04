//go:build sliver_controlflow_e2e && windows

package main

import (
	"os/exec"
	"strconv"
)

func prepareCommand(_ *exec.Cmd) {}

func terminateProcessTree(cmd *exec.Cmd) {
	_ = exec.Command("taskkill.exe", "/PID", strconv.Itoa(cmd.Process.Pid), "/T").Run()
}

func killProcessTree(cmd *exec.Cmd) {
	_ = exec.Command("taskkill.exe", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
}
