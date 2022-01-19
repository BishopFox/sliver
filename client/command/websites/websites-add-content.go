package websites

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// WebsitesAddContentCmd - Add static content to a website
func WebsitesAddContentCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	websiteName := ctx.Flags.String("website")
	if websiteName == "" {
		con.PrintErrorf("Must specify a website name via --website, see --help\n")
		return
	}
	webPath := ctx.Flags.String("web-path")
	if webPath == "" {
		con.PrintErrorf("Must specify a web path via --web-path, see --help\n")
		return
	}
	contentPath := ctx.Flags.String("content")
	if contentPath == "" {
		con.PrintErrorf("Must specify some --content\n")
		return
	}
	contentPath, _ = filepath.Abs(contentPath)
	contentType := ctx.Flags.String("content-type")
	recursive := ctx.Flags.Bool("recursive")

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
