package aka

import (
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func AkaListCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(akaAliases) == 0 {
		con.PrintInfof("No command aliases created. Use `aka create` to create one.\n")
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Alias",
		"Command",
	})

	for alias, cmd := range akaAliases {
		tw.AppendRow(table.Row{
			alias,
			cmd.Description,
		})
	}

	con.Printf("%s\n", tw.Render())
}
