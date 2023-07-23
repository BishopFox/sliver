package websites

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
	"context"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

const (
	fileSampleSize  = 512
	defaultMimeType = "application/octet-stream"
)

// WebsitesCmd - Manage websites.
func WebsitesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(args) > 0 {
		websiteName := args[0]
		ListWebsiteContent(websiteName, con)
	} else {
		ListWebsites(cmd, con, args)
	}
}

// ListWebsites - Display a list of websites.
func ListWebsites(cmd *cobra.Command, con *console.SliverClient, args []string) {
	websites, err := con.Rpc.Websites(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to list websites %s", err)
		return
	}
	if len(websites.Websites) < 1 {
		con.PrintInfof("No websites\n")
		return
	}
	con.Println("Websites")
	con.Println(strings.Repeat("=", len("Websites")))
	for _, site := range websites.Websites {
		con.Printf("%s%s%s - %d page(s)\n", console.Bold, site.Name, console.Normal, len(site.Contents))
	}
}

// ListWebsiteContent - List the static contents of a website.
func ListWebsiteContent(websiteName string, con *console.SliverClient) {
	website, err := con.Rpc.Website(context.Background(), &clientpb.Website{
		Name: websiteName,
	})
	if err != nil {
		con.PrintErrorf("Failed to list website content %s", err)
		return
	}
	if 0 < len(website.Contents) {
		PrintWebsite(website, con)
	} else {
		con.PrintInfof("No content for '%s'", websiteName)
	}
}

// PrintWebsite - Print a website and its contents, paths, etc.
func PrintWebsite(web *clientpb.Website, con *console.SliverClient) {
	con.Println(console.Clearln + console.Info + web.Name)
	con.Println()
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Path",
		"Content-type",
		"Size",
	})
	sortedContents := []*clientpb.WebContent{}
	for _, content := range web.Contents {
		sortedContents = append(sortedContents, content)
	}
	sort.SliceStable(sortedContents, func(i, j int) bool {
		return sortedContents[i].Path < sortedContents[j].Path
	})
	for _, content := range sortedContents {
		tw.AppendRow(table.Row{
			content.Path,
			content.ContentType,
			content.Size,
		})
	}
	con.Println(tw.Render())
}

// WebsiteNameCompleter completes the names of available websites.
func WebsiteNameCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		websites, err := con.Rpc.Websites(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("Failed to list websites %s", err)
		}

		for _, ws := range websites.Websites {
			results = append(results, ws.Name)
		}

		if len(results) == 0 {
			return carapace.ActionMessage("no available websites")
		}

		return carapace.ActionValues(results...).Tag("websites").Usage("website name")
	})
}
