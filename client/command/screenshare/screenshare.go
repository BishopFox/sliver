package screenshare

import (
	"fmt"
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

type ScreenTask struct {
	ID        uint32
	SessionID string
	Display   uint32
	Server    *http.Server
	Recording bool
	Cleanup   func()
	Auth      string
}

var ScreenShares = map[uint32]*ScreenTask{}

func printScreenShare() {
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "ID\tSessionID\tDisplay\tServer\tRecording\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("SessionID")),
		strings.Repeat("=", len("Display")),
		strings.Repeat("=", len("Server")),
		strings.Repeat("=", len("Recording")),
	)

	var keys []int
	for _, job := range ScreenShares {
		keys = append(keys, int(job.ID))
	}
	sort.Ints(keys) // Fucking Go can't sort int32's, so we convert to/from int's

	for _, k := range keys {
		job := ScreenShares[uint32(k)]
		fmt.Fprintf(table, "%d\t%d\t%d\t%s\t%t\t\n", job.ID, job.SessionID, job.Display, job.Server.Addr, job.Recording)
	}
	table.Flush()
}

// ScreenshareCmd - Take a screenshot of the remote system
func ScreenshareCmd(c *grumble.Context, con *console.SliverConsoleClient) {
	if c.Flags.Int("kill") != -1 {
		screenShareKill(uint32(c.Flags.Int("kill")), con)
	} else if c.Flags.Bool("kill-all") {
		killAllScreenShare(con)
	} else {
		printScreenShare()
	}
}
