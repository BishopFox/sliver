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

	// WebsiteUploaded returns both kinds of content, the uploads decide whether the
	// website record survives so the operator has to see them before confirming
	site, err := con.Rpc.WebsiteUploaded(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("Failed to fetch website %s", err)
		return
	}

	confirm := false
	_ = forms.Confirm(websiteRemoveConfirmationPrompt(name, len(site.Contents), len(site.Uploaded)), &confirm)
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
	con.PrintInfof("%s\n", websiteRemoveSuccessMessage(name, len(site.Contents), len(site.Uploaded)))
}

func websiteRemoveConfirmationPrompt(name string, contentCount int, uploadCount int) string {
	contentLabel := "content item"
	if contentCount != 1 {
		contentLabel = "content items"
	}
	if uploadCount > 0 {
		return fmt.Sprintf("Delete %d %s from '%s'? The website is kept, it still has %s.",
			contentCount, contentLabel, name, websiteUploadCountLabel(uploadCount))
	}
	return fmt.Sprintf("Delete website '%s' and %d %s?", name, contentCount, contentLabel)
}

func websiteRemoveSuccessMessage(name string, contentCount int, uploadCount int) string {
	if uploadCount > 0 {
		return fmt.Sprintf("Removed %d content items, kept %s because it still has %s",
			contentCount, name, websiteUploadCountLabel(uploadCount))
	}
	return fmt.Sprintf("Removed %s and %d content items", name, contentCount)
}

func websiteUploadCountLabel(uploadCount int) string {
	if uploadCount == 1 {
		return "1 upload"
	}
	return fmt.Sprintf("%d uploads", uploadCount)
}
