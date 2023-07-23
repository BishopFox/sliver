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
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// WebsitesAddContentCmd - Add static content to a website.
func WebsitesAddContentCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
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
	contentPath, _ := cmd.Flags().GetString("content")
	if contentPath == "" {
		con.PrintErrorf("Must specify some --content\n")
		return
	}
	contentPath, _ = filepath.Abs(contentPath)
	contentType, _ := cmd.Flags().GetString("content-type")
	recursive, _ := cmd.Flags().GetBool("recursive")

	fileInfo, err := os.Stat(contentPath)
	if err != nil {
		con.PrintErrorf("Error adding content %s\n", err)
		return
	}

	addWeb := &clientpb.WebsiteAddContent{
		Name:     websiteName,
		Contents: map[string]*clientpb.WebContent{},
	}

	if fileInfo.IsDir() {
		if !recursive && !confirmAddDirectory() {
			return
		}
		webAddDirectory(addWeb, webPath, contentPath)
	} else {
		webAddFile(addWeb, webPath, contentType, contentPath)
	}

	web, err := con.Rpc.WebsiteAddContent(context.Background(), addWeb)
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	PrintWebsite(web, con)
}

func webAddDirectory(web *clientpb.WebsiteAddContent, webpath string, contentPath string) {
	fullLocalPath, _ := filepath.Abs(contentPath)
	filepath.Walk(contentPath, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// localPath is the full absolute path to the file, so we cut it down
			fullWebpath := path.Join(webpath, localPath[len(fullLocalPath):])
			webAddFile(web, fullWebpath, "", localPath)
		}
		return nil
	})
}

func webAddFile(web *clientpb.WebsiteAddContent, webpath string, contentType string, contentPath string) error {
	fileInfo, err := os.Stat(contentPath)
	if os.IsNotExist(err) {
		return err // contentPath does not exist
	}
	if fileInfo.IsDir() {
		return errors.New("file content path is directory")
	}

	file, err := os.Open(contentPath)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if contentType == "" {
		contentType = sniffContentType(file)
	}

	web.Contents[webpath] = &clientpb.WebContent{
		Path:        webpath,
		ContentType: contentType,
		Content:     data,
	}
	return nil
}

func confirmAddDirectory() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "Recursively add entire directory?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}

func sniffContentType(out *os.File) string {
	out.Seek(0, io.SeekStart)
	buffer := make([]byte, fileSampleSize)
	_, err := out.Read(buffer)
	if err != nil {
		return defaultMimeType
	}
	contentType := http.DetectContentType(buffer)
	return contentType
}
