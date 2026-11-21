//go:build windows

package magebuild

import (
	"os/exec"
	"strconv"
)

func configureProcess(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// taskkill /T terminates compiler descendants as well as the Mage runner.
		// Fall back to killing the runner if taskkill is unavailable.
		err := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
		if err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	setWaitDelay(cmd)
}
