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
)

/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox

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

const (
	timeFormat = "2006-01-02 15:04:05 UTC-0700"
)

func CertificateInfoCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	// Ask the server for information about certificates
	certificateInfo, err := con.Rpc.GetCertificateInfo(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("could not get certificate information from database: %s", err.Error())
		return
	}

	printCertificateInfo(con, certificateInfo.Info)
}

func checkCertExpiry(expiryTime time.Time) string {
	if expiryTime.Before(time.Now()) || expiryTime.Equal(time.Now()) {
		return console.Bold + console.Red
	}

	// One week is 168 hours - this is bad
	if expiryTime.Before(time.Now().Add(168 * time.Hour)) {
		return console.Bold + console.Red
	}

	// One month is approximately 730 hours - this is a warning
	if expiryTime.Before(time.Now().Add(730 * time.Hour)) {
		return console.Bold + console.Orange
	}

	return console.Normal
}

func printCertificateInfo(con *console.SliverClient, certData []*clientpb.CertificateData) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Common Name",
		"Creation Time",
		"Certificate Type",
		"Key Algorithm",
		"Validity Start",
		"Expires",
	})

	for _, cert := range certData {
		rowColor := console.Normal

		expiry, err := time.Parse(timeFormat, cert.ValidityExpiry)
		// This should not error out, but if it does, the row will not be colored
		if err == nil {
			rowColor = checkCertExpiry(expiry)
		}
		tw.AppendRow(table.Row{
			fmt.Sprintf(rowColor+"%s"+console.Normal, cert.CN),
			fmt.Sprintf(rowColor+"%s"+console.Normal, cert.CreationTime),
			fmt.Sprintf(rowColor+"%s"+console.Normal, cert.Type),
			fmt.Sprintf(rowColor+"%s"+console.Normal, cert.KeyAlgorithm),
			fmt.Sprintf(rowColor+"%s"+console.Normal, cert.ValidityStart),
			fmt.Sprintf(rowColor+"%s"+console.Normal, cert.ValidityExpiry),
		})
	}

	tw.SortBy([]table.SortBy{{Name: "Expires", Mode: table.Dsc}})

	con.Printf("%s\n", tw.Render())
}
