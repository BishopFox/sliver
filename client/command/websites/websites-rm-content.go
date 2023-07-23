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
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// WebsitesRmContent - Remove static content from a website.
func WebsitesRmContent(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, _ := cmd.Flags().GetString("website")
	webPath, _ := cmd.Flags().GetString("web-path")
	recursive, _ := cmd.Flags().GetBool("recursive")

	if name == "" {
		con.PrintErrorf("Must specify a website name via --website, see --help\n")
		return
	}
	if webPath == "" {
		con.PrintErrorf("Must specify a web path via --web-path, see --help\n")
		return
	}

	website, err := con.Rpc.Website(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	rmWebContent := &clientpb.WebsiteRemoveContent{
		Name:  name,
		Paths: []string{},
	}
	if recursive {
		for contentPath := range website.Contents {
			if strings.HasPrefix(contentPath, webPath) {
				rmWebContent.Paths = append(rmWebContent.Paths, contentPath)
			}
		}
	} else {
		rmWebContent.Paths = append(rmWebContent.Paths, webPath)
	}
	web, err := con.Rpc.WebsiteRemoveContent(context.Background(), rmWebContent)
	if err != nil {
		con.PrintErrorf("Failed to remove content %s", err)
		return
	}
	PrintWebsite(web, con)
}
