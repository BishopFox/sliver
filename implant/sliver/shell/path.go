package shell

import "strings"

// normalizeWindowsShellPath converts the slash form accepted by Windows file
// APIs into the native form expected by command interpreters parsing argv[0].
func normalizeWindowsShellPath(path string) string {
	return strings.ReplaceAll(path, "/", `\`)
}
