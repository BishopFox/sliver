// Package ps provides shell determination by process name
package ps

import (
	"os"
	"strings"

	"github.com/rsteube/carapace/third_party/github.com/mitchellh/go-ps"
)

// DetermineShell determines shell by parent process name.
func DetermineShell() string {
	process, err := ps.FindProcess(os.Getpid())
	if err != nil {
		return ""
	}
	for {
		if process, err = ps.FindProcess(process.PPid()); err != nil || process == nil {
			return ""
		}

		executable := process.Executable()
		switch strings.SplitN(strings.TrimSuffix(executable, ".exe"), "-", 2)[0] {
		case "bash":
			if isBLE() {
				return "bash-ble"
			}
			return "bash"
		case "elvish":
			return "elvish"
		case "fish":
			return "fish"
		case "ion":
			return "ion"
		case "nu":
			return "nushell"
		case "oil":
			return "oil"
		case "osh":
			return "oil"
		case "powershell":
			return "powershell"
		case "pwsh":
			return "powershell"
		case "tcsh":
			return "tcsh"
		case "xonsh":
			return "xonsh"
		case "zsh":
			return "zsh"
		default:
			if strings.Contains(executable, "xonsh-wrapped") { // nix packaged version
				return "xonsh"
			}
		}
	}
}

func isBLE() bool {
	bleEnvs := []string{
		"_ble_util_fd_null",
		"_ble_util_fd_stderr",
		"_ble_util_fd_stdin",
		"_ble_util_fd_stdout",
		"_ble_util_fd_zero",
	}
	for _, e := range bleEnvs {
		if _, ok := os.LookupEnv(e); ok {
			return true
		}
	}
	return false
}
