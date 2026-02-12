package websites

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"sort"

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
// ListWebsites - Display websites. 的列表
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
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{"Name", "Objects"})
	for _, site := range websites.Websites {
		tw.AppendRow(table.Row{site.Name, len(site.Contents)})
	}

	con.Println(tw.Render())
}

// ListWebsiteContent - List the static contents of a website.
// ListWebsiteContent - List website. 的静态内容
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
// PrintWebsite - Print 网站及其内容、路径、etc.
func PrintWebsite(web *clientpb.Website, con *console.SliverClient) {
	con.Println(console.Clearln + console.Info + web.Name)
	con.Println()
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Path",
		"Content-type",
		"Size",
		"Original File",
		"SHA256",
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
			content.OriginalFile,
			content.Sha256,
		})
	}
	con.Println(tw.Render())
}

// WebsiteNameCompleter completes the names of available websites.
// WebsiteNameCompleter 补全可用 websites. 的名称
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
