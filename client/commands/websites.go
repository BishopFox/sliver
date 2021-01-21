package commands

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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/evilsocket/islazy/tui"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

const (
	fileSampleSize  = 512
	defaultMimeType = "application/octet-stream"
)

// WebsiteOptions - General website options
type WebsiteOptions struct {
	ContentOptions struct {
		Website     string `long:"website" description:"Website name" required:"true"`
		WebPath     string `long:"web-path" description:"HTTP path to host file at" default:"/" required:"true"`
		Content     string `long:"content" description:"local file path/dir (must use --recursive if dir)"`
		ContentType string `long:"content-type" description:"MIME content-type (if blank, use file ext.)"`
		Recursive   bool   `long:"recursive" description:"Apply command (delete/add) recursively"`
	} `group:"content options"`
}

// Websites - All websites management commands
type Websites struct {
	Positional struct {
		Name string `description:"website content name to display"`
	} `positional-args:"yes"`
}

// Execute - Command
func (w *Websites) Execute(args []string) (err error) {
	if w.Positional.Name != "" {
		listWebsiteContent(w, transport.RPC)
	} else {
		listWebsites(w, transport.RPC)
	}
	return
}

func listWebsites(w *Websites, rpc rpcpb.SliverRPCClient) {
	websites, err := rpc.Websites(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"Failed to list websites %s\n", err)
		return
	}
	if len(websites.Websites) < 1 {
		fmt.Printf(util.Info + "No websites\n")
		return
	}
	fmt.Println(tui.Bold(tui.Yellow("Websites")))
	fmt.Println(strings.Repeat("=", len("Websites")))
	for _, site := range websites.Websites {
		fmt.Printf("%s%s%s - %d page(s)\n", tui.BOLD, site.Name, tui.RESET, len(site.Contents))
	}
}

func listWebsiteContent(w *Websites, rpc rpcpb.SliverRPCClient) {
	if w.Positional.Name == "" {
		return
	}
	website, err := rpc.Website(context.Background(), &clientpb.Website{
		Name: w.Positional.Name,
	})
	if err != nil {
		fmt.Printf(util.Error+"Failed to list website content %s\n", err)
		return
	}
	if 0 < len(website.Contents) {
		displayWebsite(website)
	} else {
		fmt.Printf(util.Info+"No content for '%s'\n", w.Positional.Name)
	}
}

func displayWebsite(web *clientpb.Website) {

	table := util.NewTable(tui.Bold(tui.Yellow(web.Name)))
	headers := []string{"Path", "Content-Type", "Size"}
	headLen := []int{0, 10, 0}
	table.SetColumns(headers, headLen)

	sortedContents := []*clientpb.WebContent{}
	for _, content := range web.Contents {
		sortedContents = append(sortedContents, content)
	}
	sort.SliceStable(sortedContents, func(i, j int) bool {
		return sortedContents[i].Path < sortedContents[j].Path
	})

	for _, content := range sortedContents {
		size := tui.Dim(fmt.Sprintf("%d", content.Size))
		path := tui.Bold(content.Path)
		table.AppendRow([]string{path, content.ContentType, size})
	}
	table.Output()
}

// WebsitesDelete - Remove an entire website
type WebsitesDelete struct {
	Positional struct {
		WebsiteName []string `description:"website content name to display" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Command
func (w *WebsitesDelete) Execute(args []string) (err error) {

	for _, name := range w.Positional.WebsiteName {
		_, err := transport.RPC.WebsiteRemove(context.Background(), &clientpb.Website{
			Name: name,
		})
		if err != nil {
			fmt.Printf(util.Error+"Failed to remove website %s\n", err)
		} else {
			fmt.Printf(util.Info+"Removed website %s%s%s\n", tui.YELLOW, name, tui.RESET)
		}
	}
	return
}

// WebsitesDeleteContent - Remove content from a website
type WebsitesDeleteContent struct {
	WebsiteOptions
}

// Execute - Command
func (w *WebsitesDeleteContent) Execute(args []string) (err error) {
	name := w.ContentOptions.Website
	webPath := w.ContentOptions.WebPath
	recursive := w.ContentOptions.Recursive

	if name == "" {
		fmt.Printf(util.Error + "Must specify a website name via --website, see --help\n")
		return
	}
	if webPath == "" {
		fmt.Printf(util.Error + "Must specify a web path via --web-path, see --help\n")
		return
	}

	website, err := transport.RPC.Website(context.Background(), &clientpb.Website{
		Name: name,
	})
	if err != nil {
		fmt.Printf(util.Error+"%s", err)
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
	web, err := transport.RPC.WebsiteRemoveContent(context.Background(), rmWebContent)
	if err != nil {
		fmt.Printf(util.Error+"Failed to remove content %s\n", err)
		return
	}
	displayWebsite(web)
	return
}

// WebsitesAddContent - Add content to a website
type WebsitesAddContent struct {
	WebsiteOptions
}

// Execute - Command
func (w *WebsitesAddContent) Execute(args []string) (err error) {
	websiteName := w.ContentOptions.Website
	if websiteName == "" {
		fmt.Printf(util.Error + "Must specify a website name via --website, see --help\n")
		return
	}
	webPath := w.ContentOptions.WebPath
	if webPath == "" {
		fmt.Printf(util.Error + "Must specify a web path via --web-path, see --help\n")
		return
	}
	contentPath := w.ContentOptions.Content
	if contentPath == "" {
		fmt.Println(util.Error + "Must specify some --content\n")
		return
	}
	contentPath, _ = filepath.Abs(contentPath)
	contentType := w.ContentOptions.ContentType
	recursive := w.ContentOptions.Recursive

	fileInfo, err := os.Stat(contentPath)
	if err != nil {
		fmt.Printf(util.Error+"Error adding content %s\n", err)
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

	web, err := transport.RPC.WebsiteAddContent(context.Background(), addWeb)
	if err != nil {
		fmt.Printf(util.Error+"%s", err)
		return
	}
	displayWebsite(web)
	return
}

func confirmAddDirectory() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "Recursively add entire directory?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
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
	data, err := ioutil.ReadAll(file)
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

// WebsiteType - Update a path's content-type
type WebsiteType struct {
	WebsiteOptions
}

// Execute - Command
func (w *WebsiteType) Execute(args []string) (err error) {
	websiteName := w.ContentOptions.Website
	if websiteName == "" {
		fmt.Printf(util.Error + "Must specify a website name via --website, see --help\n")
		return
	}
	webPath := w.ContentOptions.WebPath
	if webPath == "" {
		fmt.Printf(util.Error + "Must specify a web path via --web-path, see --help\n")
		return
	}
	contentType := w.ContentOptions.ContentType
	if contentType == "" {
		fmt.Println(util.Error + "Must specify a new --content-type, see --help\n")
		return
	}

	updateWeb := &clientpb.WebsiteAddContent{
		Name:     websiteName,
		Contents: map[string]*clientpb.WebContent{},
	}
	updateWeb.Contents[webPath] = &clientpb.WebContent{
		ContentType: contentType,
	}

	web, err := transport.RPC.WebsiteUpdateContent(context.Background(), updateWeb)
	if err != nil {
		fmt.Printf(util.Error+"%s", err)
		return
	}
	displayWebsite(web)
	return
}
