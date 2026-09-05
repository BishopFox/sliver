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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// uploadedMetadata - What gets written next to the raw content as JSON
type uploadedMetadata struct {
	ID             string `json:"id"`
	WebsiteID      string `json:"website_id"`
	Method         string `json:"method"`
	Path           string `json:"path"`
	URLParameters  string `json:"url_parameters,omitempty"`
	RemoteAddress  string `json:"remote_address"`
	UserAgent      string `json:"user_agent"`
	Headers        any    `json:"headers"`
	ContentType    string `json:"content_type"`
	Size           uint64 `json:"size"`
	Sha256         string `json:"sha256"`
	ReceivedAt     string `json:"received_at"`
	ReceivedAtUnix int64  `json:"received_at_unix"`
	ContentFile    string `json:"content_file"`
}

// WebsitesUploadEnableCmd - Let a website accept content via PUT/POST.
func WebsitesUploadEnableCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	setUploadAllowed(con, args[0], true)
}

// WebsitesUploadDisableCmd - Stop a website from accepting content via PUT/POST.
func WebsitesUploadDisableCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	setUploadAllowed(con, args[0], false)
}

func setUploadAllowed(con *console.SliverClient, websiteName string, allowed bool) {
	website, err := con.Rpc.WebsiteUpdate(context.Background(), &clientpb.Website{
		Name:         websiteName,
		AllowsUpload: allowed,
	})
	if err != nil {
		con.PrintErrorf("Failed to update website %s\n", err)
		return
	}
	if website.AllowsUpload {
		con.PrintInfof("Uploads enabled for '%s', PUT/POST requests are now stored\n", website.Name)
	} else {
		con.PrintInfof("Uploads disabled for '%s'\n", website.Name)
	}
}

// WebsitesUploadListCmd - List the content uploaded to a website via PUT/POST.
func WebsitesUploadListCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	websiteName := args[0]
	website, err := con.Rpc.WebsiteUploaded(context.Background(), &clientpb.Website{
		Name: websiteName,
	})
	if err != nil {
		con.PrintErrorf("Failed to list uploaded content %s\n", err)
		return
	}
	if len(website.Uploaded) < 1 {
		if !website.AllowsUpload {
			con.PrintWarnf("Uploads are disabled for '%s', run 'websites upload enable %s'\n", websiteName, websiteName)
			return
		}
		con.PrintInfof("No uploaded content for '%s'\n", websiteName)
		return
	}
	PrintUploadedContent(website, con)
}

// PrintUploadedContent - Print the content uploaded to a website.
func PrintUploadedContent(web *clientpb.Website, con *console.SliverClient) {
	con.Println(console.Clearln + console.Info + web.Name)
	con.Println()
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Received",
		"Method",
		"Path",
		"Remote Address",
		"Size",
		"Content-type",
		"SHA256",
	})
	for _, content := range web.Uploaded {
		tw.AppendRow(table.Row{
			content.ID,
			time.Unix(content.ReceivedAt, 0).Format(time.RFC1123),
			content.Method,
			content.Path,
			content.RemoteAddress,
			content.Size,
			content.ContentType,
			content.Sha256,
		})
	}
	con.Println(tw.Render())
}

// WebsitesUploadRmCmd - Remove every piece of content uploaded to a website.
func WebsitesUploadRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	websiteName := args[0]
	website, err := con.Rpc.WebsiteUploaded(context.Background(), &clientpb.Website{
		Name: websiteName,
	})
	if err != nil {
		con.PrintErrorf("Failed to fetch website %s\n", err)
		return
	}
	if len(website.Uploaded) < 1 {
		con.PrintInfof("No uploaded content for '%s'\n", websiteName)
		return
	}

	confirm := false
	_ = forms.Confirm(uploadedRemoveConfirmationPrompt(websiteName, len(website.Uploaded)), &confirm)
	if !confirm {
		return
	}

	// Send the IDs we just confirmed, so an upload that lands in the meantime is not swept up
	ids := make([]string, 0, len(website.Uploaded))
	for _, content := range website.Uploaded {
		ids = append(ids, content.ID)
	}
	_, err = con.Rpc.WebsiteRemoveUploaded(context.Background(), &clientpb.WebsiteRemoveUploaded{
		Name: websiteName,
		IDs:  ids,
	})
	if err != nil {
		con.PrintErrorf("Failed to remove uploaded content %s\n", err)
		return
	}
	con.PrintInfof("%s\n", uploadedRemoveSuccessMessage(websiteName, len(ids)))
}

func uploadedRemoveConfirmationPrompt(name string, contentCount int) string {
	return fmt.Sprintf("Delete %d %s received by '%s'?", contentCount, uploadedContentLabel(contentCount), name)
}

func uploadedRemoveSuccessMessage(name string, contentCount int) string {
	return fmt.Sprintf("Removed %d %s from %s", contentCount, uploadedContentLabel(contentCount), name)
}

func uploadedContentLabel(contentCount int) string {
	if contentCount == 1 {
		return "upload"
	}
	return "uploads"
}

// WebsitesUploadRmContentCmd - Remove specific content uploaded to a website.
func WebsitesUploadRmContentCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, _ := cmd.Flags().GetString("website")
	webPath, _ := cmd.Flags().GetString("web-path")
	contentID, _ := cmd.Flags().GetString("id")
	recursive, _ := cmd.Flags().GetBool("recursive")

	if name == "" {
		con.PrintErrorf("Must specify a website name via --website, see --help\n")
		return
	}
	if webPath == "" && contentID == "" {
		con.PrintErrorf("Must specify a web path via --web-path or an upload via --id, see --help\n")
		return
	}
	if webPath != "" && contentID != "" {
		con.PrintErrorf("--web-path and --id are mutually exclusive, see --help\n")
		return
	}

	website, err := con.Rpc.WebsiteUploaded(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	// Uploads are addressed by ID, several of them can share the same path
	rmUploaded := &clientpb.WebsiteRemoveUploaded{
		Name: name,
		IDs:  []string{},
	}
	for _, content := range website.Uploaded {
		switch {
		case contentID != "":
			if content.ID == contentID {
				rmUploaded.IDs = append(rmUploaded.IDs, content.ID)
			}
		case recursive:
			if strings.HasPrefix(content.Path, webPath) {
				rmUploaded.IDs = append(rmUploaded.IDs, content.ID)
			}
		default:
			if content.Path == webPath {
				rmUploaded.IDs = append(rmUploaded.IDs, content.ID)
			}
		}
	}
	if len(rmUploaded.IDs) < 1 {
		if contentID != "" {
			con.PrintErrorf("No upload '%s' on website '%s'\n", contentID, name)
		} else {
			con.PrintErrorf("No uploaded content matching '%s' on website '%s'\n", webPath, name)
		}
		return
	}

	web, err := con.Rpc.WebsiteRemoveUploaded(context.Background(), rmUploaded)
	if err != nil {
		con.PrintErrorf("Failed to remove uploaded content %s\n", err)
		return
	}
	con.PrintInfof("%s\n", uploadedRemoveSuccessMessage(name, len(rmUploaded.IDs)))
	if len(web.Uploaded) > 0 {
		PrintUploadedContent(web, con)
	}
}

// WebsitesUploadDownloadCmd - Download one uploaded file plus its metadata.
func WebsitesUploadDownloadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	contentID := args[0]
	content, err := con.Rpc.WebsiteUploadedContent(context.Background(), &clientpb.WebUploadedContent{
		ID: contentID,
	})
	if err != nil {
		con.PrintErrorf("Failed to fetch uploaded content %s\n", err)
		return
	}

	saveTo, _ := cmd.Flags().GetString("save")
	if saveTo == "" {
		saveTo = "."
	}
	saveTo, err = filepath.Abs(saveTo)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if fi, err := os.Stat(saveTo); err != nil || !fi.IsDir() {
		con.PrintErrorf("'%s' is not a directory\n", saveTo)
		return
	}

	contentPath := filepath.Join(saveTo, uploadedFileName(content))
	metadataPath := contentPath + ".json"
	for _, path := range []string{contentPath, metadataPath} {
		if _, err := os.Stat(path); err == nil {
			overwrite := false
			_ = forms.Confirm(fmt.Sprintf("Overwrite '%s'?", path), &overwrite)
			if !overwrite {
				return
			}
		}
	}

	// The raw body exactly as it was received
	err = os.WriteFile(contentPath, content.Content, 0600)
	if err != nil {
		con.PrintErrorf("Failed to write %s: %s\n", contentPath, err)
		return
	}

	metadata, err := json.MarshalIndent(uploadedMetadataOf(content, filepath.Base(contentPath)), "", "  ")
	if err != nil {
		con.PrintErrorf("Failed to serialize metadata: %s\n", err)
		return
	}
	err = os.WriteFile(metadataPath, append(metadata, '\n'), 0600)
	if err != nil {
		con.PrintErrorf("Failed to write %s: %s\n", metadataPath, err)
		return
	}

	con.PrintInfof("Wrote %d byte(s) to %s\n", len(content.Content), contentPath)
	con.PrintInfof("Wrote metadata to %s\n", metadataPath)

	// The server hashes the body on the way in, so a mismatch means the stored file drifted
	if content.Sha256 != "" {
		digest := sha256.Sum256(content.Content)
		if actual := hex.EncodeToString(digest[:]); actual != content.Sha256 {
			con.PrintWarnf("SHA256 mismatch: expected %s, got %s\n", content.Sha256, actual)
		}
	}
}

func uploadedMetadataOf(content *clientpb.WebUploadedContent, contentFile string) uploadedMetadata {
	metadata := uploadedMetadata{
		ID:             content.ID,
		WebsiteID:      content.WebsiteID,
		Method:         content.Method,
		Path:           content.Path,
		URLParameters:  content.URLParameters,
		RemoteAddress:  content.RemoteAddress,
		UserAgent:      content.UserAgent,
		Headers:        content.Headers,
		ContentType:    content.ContentType,
		Size:           content.Size,
		Sha256:         content.Sha256,
		ReceivedAt:     time.Unix(content.ReceivedAt, 0).Format(time.RFC3339),
		ReceivedAtUnix: content.ReceivedAt,
		ContentFile:    contentFile,
	}
	// Headers are stored as a JSON blob, nest them instead of escaping them into a string
	headers := map[string][]string{}
	if err := json.Unmarshal([]byte(content.Headers), &headers); err == nil {
		metadata.Headers = headers
	}
	return metadata
}

var unsafeFileNameRegex = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// uploadedFileName - A local file name that keeps the upload identifiable and is safe on every OS
func uploadedFileName(content *clientpb.WebUploadedContent) string {
	name := filepath.Base(strings.ReplaceAll(content.Path, "\\", "/"))
	name = strings.Trim(unsafeFileNameRegex.ReplaceAllString(name, "_"), "._")
	if name == "" {
		name = "upload"
	}
	return fmt.Sprintf("%s_%s", content.ID, name)
}

// UploadedContentCompleter completes the IDs of content uploaded to any website.
func UploadedContentCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		websites, err := con.Rpc.Websites(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("Failed to list websites %s", err)
		}

		// Websites already carries the uploads, no need for a call per website
		results := make([]string, 0)
		for _, ws := range websites.Websites {
			for _, content := range ws.Uploaded {
				results = append(results, content.ID, fmt.Sprintf("%s %s (%s)", content.Method, content.Path, ws.Name))
			}
		}

		if len(results) == 0 {
			return carapace.ActionMessage("no uploaded content")
		}

		return carapace.ActionValuesDescribed(results...).Tag("uploaded content").Usage("uploaded content id")
	})
}
