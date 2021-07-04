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
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"github.com/desertbit/grumble"
)

const (
	fileSampleSize  = 512
	defaultMimeType = "application/octet-stream"
)

func WebsitesCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	websiteName := ctx.Args.String("name")
	if websiteName == "" {
		ListWebsites(ctx, con)
	} else {
		ListWebsiteContent(websiteName, con)
	}
}

func ListWebsites(ctx *grumble.Context, con *console.SliverConsoleClient) {
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

func ListWebsiteContent(websiteName string, con *console.SliverConsoleClient) {
	website, err := con.Rpc.Website(context.Background(), &clientpb.Website{
		Name: websiteName,
	})
	if err != nil {
		con.PrintErrorf("Failed to list website content %s", err)
		return
	}
	if 0 < len(website.Contents) {
		displayWebsite(website, con)
	} else {
		con.PrintInfof("No content for '%s'", websiteName)
	}
}

func displayWebsite(web *clientpb.Website, con *console.SliverConsoleClient) {
	con.Println(console.Clearln + console.Info + web.Name)
	con.Println()
	table := tabwriter.NewWriter(con.App.Stdout(), 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "Path\tContent-type\tSize\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Path")),
		strings.Repeat("=", len("Content-type")),
		strings.Repeat("=", len("Size")))
	sortedContents := []*clientpb.WebContent{}
	for _, content := range web.Contents {
		sortedContents = append(sortedContents, content)
	}
	sort.SliceStable(sortedContents, func(i, j int) bool {
		return sortedContents[i].Path < sortedContents[j].Path
	})
	for _, content := range sortedContents {
		fmt.Fprintf(table, "%s\t%s\t%d\t\n", content.Path, content.ContentType, content.Size)
	}
	table.Flush()
}
