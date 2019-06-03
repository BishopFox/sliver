package version

import "fmt"

const (
	// ClientVersion - Client version number
	ClientVersion = "0.0.6"
	normal        = "\033[0m"
	bold          = "\033[1m"
)

// FullVersion - Full version string
func FullVersion() string {
	ver := fmt.Sprintf("%s - %s", ClientVersion, GitVersion)
	if GitDirty != "" {
		ver += " - " + bold + GitDirty + normal
	}
	return ver
}
