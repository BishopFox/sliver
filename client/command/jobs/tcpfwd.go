package jobs

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// TCPFwdListenerCmd - Start a TCP forwarder on the gVisor virtual network
func TCPFwdListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	wgPort, _ := cmd.Flags().GetUint32("wg-port")
	localAddr, _ := cmd.Flags().GetString("local")

	con.PrintInfof("Starting TCP forwarder (gVisor:%d → %s) ...\n", wgPort, localAddr)
	job, err := con.Rpc.StartTCPFwdListener(context.Background(), &clientpb.TCPFwdListenerReq{
		WGPort:    wgPort,
		LocalAddr: localAddr,
	})
	con.Println()
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", job.JobID)
	}
}
