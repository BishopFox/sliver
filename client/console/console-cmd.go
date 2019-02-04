package console

import (
	"fmt"

	"github.com/desertbit/grumble"
)

var (
	// Stylizes known processes in the `ps` command
	knownProcs = map[string]string{
		"ccSvcHst.exe": red, // SEP
		"cb.exe":       red, // Carbon Black
	}
)

// ---------------------- Command Implementations ----------------------

func sessionsCmd(ctx *grumble.Context) {

}

func backgroundCmd(ctx *grumble.Context) {

}

func killCmd(ctx *grumble.Context) {

}

func infoCmd(ctx *grumble.Context) {

}

func useCmd(ctx *grumble.Context) {

}

func generateCmd(ctx *grumble.Context) {

}

func msfCmd(ctx *grumble.Context) {

}

func injectCmd(ctx *grumble.Context) {

}

func psCmd(ctx *grumble.Context) {

}

func pingCmd(ctx *grumble.Context) {

}

func rmCmd(ctx *grumble.Context) {

}

func mkdirCmd(ctx *grumble.Context) {

}

func cdCmd(ctx *grumble.Context) {

}

func pwdCmd(ctx *grumble.Context) {

}

func lsCmd(ctx *grumble.Context) {

}

func catCmd(ctx *grumble.Context) {

}

func downloadCmd(ctx *grumble.Context) {

}

func uploadCmd(ctx *grumble.Context) {

}

func procdumpCmd(ctx *grumble.Context) {

}

func byteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
