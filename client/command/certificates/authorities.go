package certificates

import (
	"context"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func CertificateAuthoritiesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	authorityInfo, err := con.Rpc.GetCertificateAuthorityInfo(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("could not get certificate authority information from database: %s", err.Error())
		return
	}

	printCertificateAuthorityInfo(con, authorityInfo.Info)
}

func printCertificateAuthorityInfo(con *console.SliverClient, authData []*clientpb.CertificateAuthorityData) {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 999
	}

	if len(authData) == 0 {
		con.PrintWarnf("There are no certificate authorities in the database.\n")
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	wideTermWidth := con.Settings.SmallTermWidth < width

	if wideTermWidth {
		tw.AppendHeader(table.Row{
			"ID",
			"Common Name",
			"Creation Time",
			"Authority Type",
			"Key Algorithm",
			"Validity Start",
			"Expires",
		})
	} else {
		tw.AppendHeader(table.Row{
			"ID",
			"Common Name",
			"Expires",
		})
	}

	for _, authority := range authData {
		rowColor := console.Normal

		expiry, err := time.Parse(timeFormat, authority.ValidityExpiry)
		if err == nil {
			rowColor = checkCertExpiry(expiry)
		}

		if wideTermWidth {
			tw.AppendRow(table.Row{
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.ID),
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.CN),
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.CreationTime),
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.Type),
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.KeyAlgorithm),
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.ValidityStart),
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.ValidityExpiry),
			})
		} else {
			tw.AppendRow(table.Row{
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.ID),
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.CN),
				fmt.Sprintf(rowColor+"%s"+console.Normal, authority.ValidityExpiry),
			})
		}
	}

	tw.SortBy([]table.SortBy{{Name: "Expires", Mode: table.Dsc}})

	con.Printf("%s\n", tw.Render())
}
