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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// WebsiteRmCmd - Remove a website and all its static content.
func WebsiteRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		con.PrintErrorf("No website name specified\n")
		return
	}

	site, err := con.Rpc.Website(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("Failed to fetch website %s", err)
		return
	}

	confirm := false
	_ = forms.Confirm(websiteRemoveConfirmationPrompt(name, len(site.Contents)), &confirm)
	if !confirm {
		return
	}

	_, err = con.Rpc.WebsiteRemove(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("Failed to remove website %s", err)
		return
	}
	con.PrintInfof("%s\n", websiteRemoveSuccessMessage(name, len(site.Contents)))
}

func websiteRemoveConfirmationPrompt(name string, contentCount int) string {
	contentLabel := "content item"
	if contentCount != 1 {
		contentLabel = "content items"
	}
	return fmt.Sprintf("Delete website '%s' and %d %s?", name, contentCount, contentLabel)
}

func websiteRemoveSuccessMessage(name string, contentCount int) string {
	return fmt.Sprintf("Removed %s and %d content items", name, contentCount)
}
