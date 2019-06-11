package limits

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	// {{if .Debug}}
	"log"
	// {{end}}
	"os"

	// {{if .LimitUsername}}
	"runtime"
	// {{end}}

	// {{if .LimitUsername}}
	"os/user"

	// {{end}}

	// {{if .LimitDatetime}}
	"time"
	// {{end}}

	// {{if or .LimitHostname .LimitUsername}}
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
		// {{if .Debug}}
		log.Printf("%#v != %#v", strings.ToLower(hostname), strings.ToLower("{{.LimitHostname}}"))
		// {{end}}
		os.Exit(1)
	}
	// {{end}}

	// {{if .LimitUsername}}
	currentUser, _ := user.Current()
	if runtime.GOOS == "windows" {
		username := strings.Split(currentUser.Username, "\\")
		if len(username) == 2 && username[1] != "{{.LimitUsername}}" {
			// {{if .Debug}}
			log.Printf("%#v != %#v", currentUser.Name, "{{.LimitUsername}}")
			// {{end}}
			os.Exit(1)
		}
	} else if currentUser.Name != "{{.LimitUsername}}" {
		// {{if .Debug}}
		log.Printf("%#v != %#v", currentUser.Name, "{{.LimitUsername}}")
		// {{end}}
		os.Exit(1)
	}
	// {{end}}

	// {{if .LimitDatetime}} "2014-11-12T11:45:26.371Z"
	expiresAt, err := time.Parse(time.RFC3339, "{{.LimitDatetime}}")
	if err == nil && time.Now().After(expiresAt) {
		// {{if .Debug}}
		log.Printf("Timelimit %#v expired", "{{.LimitDatetime}}")
		// {{end}}
		os.Exit(1)
	}
	// {{end}}

	// {{if .Debug}}
	log.Printf("Limit checks completed")
	// {{end}}

	os.Executable() // To avoid any "os unused" errors
}
