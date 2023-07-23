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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// WebsitesUpdateContentCmd - Update metadata about static website content.
func WebsitesUpdateContentCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	websiteName, _ := cmd.Flags().GetString("website")
	if websiteName == "" {
		con.PrintErrorf("Must specify a website name via --website, see --help\n")
		return
	}
	webPath, _ := cmd.Flags().GetString("web-path")
	if webPath == "" {
		con.PrintErrorf("Must specify a web path via --web-path, see --help\n")
		return
	}
	contentType, _ := cmd.Flags().GetString("content-type")
	if contentType == "" {
		con.PrintErrorf("Must specify a new --content-type, see --help\n")
		return
	}

	updateWeb := &clientpb.WebsiteAddContent{
		Name:     websiteName,
		Contents: map[string]*clientpb.WebContent{},
	}
	updateWeb.Contents[webPath] = &clientpb.WebContent{
		ContentType: contentType,
	}

	web, err := con.Rpc.WebsiteUpdateContent(context.Background(), updateWeb)
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	PrintWebsite(web, con)
}
