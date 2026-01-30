package mautrix

import (
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
)

const Version = "v0.26.0"

var GoModVersion = ""
var Commit = ""
var VersionWithCommit = Version

var DefaultUserAgent = "mautrix-go/" + Version + " go/" + strings.TrimPrefix(runtime.Version(), "go")

func init() {
	if GoModVersion == "" {
		info, _ := debug.ReadBuildInfo()
		if info != nil {
			for _, mod := range info.Deps {
				if mod.Path == "maunium.net/go/mautrix" {
					GoModVersion = mod.Version
					break
				}
			}
		}
	}
	if GoModVersion != "" {
		match := regexp.MustCompile(`v.+\d{14}-([0-9a-f]{12})`).FindStringSubmatch(GoModVersion)
		if match != nil {
			Commit = match[1]
		}
	}
	if Commit != "" {
		VersionWithCommit = fmt.Sprintf("%s+dev.%s", Version, Commit[:8])
		DefaultUserAgent = strings.Replace(DefaultUserAgent, "mautrix-go/"+Version, "mautrix-go/"+VersionWithCommit, 1)
	}
}
