package util

func DetermineExecutors(platform string, arch string) []string {
	platformExecutors := map[string]map[string][]string{
		"windows": {
			"file":     {"pwsh.exe", "powershell.exe", "cmd.exe"},
			"executor": {"pwsh", "psh", "cmd"},
		},
		"linux": {
			"file":     {"python3", "pwsh", "sh", "bash"},
			"executor": {"python", "pwsh", "sh", "bash"},
		},
		"darwin": {
			"file":     {"python3", "pwsh", "zsh", "sh", "osascript", "bash"},
			"executor": {"python", "pwsh", "zsh", "sh", "osa", "bash"},
		},
	}
	var executors []string
	for platformKey, platformValue := range platformExecutors {
		if platform == platformKey {
			for i := range platformValue["file"] {
				// if checkIfExecutorAvailable(platformValue["file"][i]) {
				executors = append(executors, platformExecutors[platformKey]["executor"][i])
				// }
			}
		}
	}
	executors = append([]string{"keyword"}, executors...)
	return executors
}
