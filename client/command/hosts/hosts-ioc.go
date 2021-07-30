package hosts

import (
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// HostsIOCsCmd - Remove a host from the database
func HostsIOCsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	host, err := SelectHost(con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if 0 < len(host.IOCs) {
		con.Printf("%s\n", hostIOCsTable(host, con))
	} else {
		con.Println()
		con.PrintInfof("No IOCs tracked on host\n")
	}
}

func hostIOCsTable(host *clientpb.Host, con *console.SliverConsoleClient) string {
	tw := table.NewWriter()
	tw.SetStyle(table.StyleBold)
	tw.AppendHeader(table.Row{"File Path", "SHA-256"})
	for _, ioc := range host.IOCs {
		tw.AppendRow(table.Row{
			ioc.Path,
			ioc.FileHash,
		})
	}
	return tw.Render()
}
