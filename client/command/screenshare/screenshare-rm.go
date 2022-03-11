package screenshare

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

func killAllScreenShare(con *console.SliverConsoleClient) {
	for _, v := range ScreenShares {
		v.Cleanup()
		delete(ScreenShares, v.ID)
	}

}
func screenShareKill(id uint32, con *console.SliverConsoleClient) error {
	ScreenShares[id].Cleanup()
	delete(ScreenShares, id)
	return nil
}
func ScreenshareRm(c *grumble.Context, con *console.SliverConsoleClient) {
	if c.Flags.Int("id") == -1 {
		con.PrintErrorf("Must specify a valid Screenshare id\n")
		return
	}
	err := screenShareKill(uint32(c.Flags.Int("id")), con)
	if err != nil {
		con.PrintErrorf("No Screenshare with id  %s\n", err.Error())
	} else {
		con.PrintInfof("Removed Screenshare\n")
	}
}
