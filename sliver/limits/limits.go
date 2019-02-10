package limits

import (
	"os"
	// {{if .LimitUsername}}
	"os/user"
	// {{end}}

	// {{if .LimitDatetime}}
	"time"
	// {{end}}

	// {{if .LimitHostname}}
	"strings"
	// {{else}}{{end}}
)

// ExecLimits - Checks for execution limitations (domain, hostname, etc)
func ExecLimits() {

	PlatformLimits() // Anti-debug & other platform specific evasion

	// {{if .LimitDomainJoined}}
	ok, err := isDomainJoined()
	if err == nil && !ok {
		os.Exit(1)
	}
	// {{end}}

	// {{if .LimitHostname}}
	hostname, err := os.Hostname()
	if err == nil && strings.ToLower(hostname) != strings.ToLower("{{.LimitHostname}}") {
		os.Exit(1)
	}
	// {{end}}

	// {{if .LimitUsername}}
	currentUser, err := user.Current()
	if err == nil && currentUser.Name != "{{.LimitUsername}}" {
		os.Exit(1)
	}
	// {{end}}

	// {{if .LimitDatetime}} "2014-11-12T11:45:26.371Z"
	expiresAt, err := time.Parse(time.RFC3339, "{{.LimitDatetime}}")
	if err == nil && time.Now().After(expiresAt) {
		os.Exit(1)
	}
	// {{end}}

	os.Executable() // To avoid any "os unused" errors
}
