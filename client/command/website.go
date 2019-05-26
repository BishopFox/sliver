package command

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/golang/protobuf/proto"

	"github.com/desertbit/grumble"
)

const (
	fileSampleSize  = 512
	defaultMimeType = "application/octet-stream"
)

func websites(ctx *grumble.Context, rpc RPCServer) {
	if len(ctx.Args) < 1 {
		listWebsites(ctx, rpc)
		return
	}
	if ctx.Flags.String("website") == "" {
		fmt.Println(Warn + "Subcommand must specify a --website")
		return
	}
	switch strings.ToLower(ctx.Args[0]) {
	case "ls":
		listWebsiteContent(ctx, rpc)
	case "add":
		addWebsiteContent(ctx, rpc)
	case "rm":
		removeWebsiteContent(ctx, rpc)
	default:
		fmt.Println(Warn + "Invalid subcommand, see 'help websites'")
	}
}

func listWebsites(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteList,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

	websites := &clientpb.Websites{}
	proto.Unmarshal(resp.Data, websites)

	if len(websites.Sites) < 1 {
		fmt.Printf(Info + "No websites\n")
		return
	}

	fmt.Println("Websites")
	fmt.Println(strings.Repeat("=", len("Websites")))
	for _, site := range websites.Sites {
		fmt.Printf("%s%s%s - %d page(s)\n", bold, site.Name, normal, len(site.Content))
	}
}

func listWebsiteContent(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteList,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

	websites := &clientpb.Websites{}
	proto.Unmarshal(resp.Data, websites)
	websiteName := ctx.Flags.String("website")
	for _, web := range websites.Sites {
		if web.Name == websiteName {
			webDisplay(web)
		}
	}
}

func addWebsiteContent(ctx *grumble.Context, rpc RPCServer) {
	websiteName := ctx.Flags.String("website")
	webPath := ctx.Flags.String("web-path")
	contentPath := ctx.Flags.String("content")
	if contentPath == "" {
		fmt.Println(Warn + "Must specify some --content")
		return
	}
	contentPath, _ = filepath.Abs(contentPath)
	contentType := ctx.Flags.String("content-type")
	recursive := ctx.Flags.Bool("recursive")

	addWebsite := &clientpb.Website{
		Name:    websiteName,
		Content: map[string]*clientpb.WebContent{},
	}

	fileInfo, err := os.Stat(contentPath)
	if err != nil {
		fmt.Printf(Warn+"Error adding content %s\n", err)
		return
	}
	if fileInfo.IsDir() {
		if !recursive && !confirmAddDirectory() {
			return
		}
		webAddDirectory(addWebsite, webPath, contentPath)
	} else {
		webAddFile(addWebsite, webPath, contentType, contentPath)
	}

	data, err := proto.Marshal(addWebsite)
	if err != nil {
		fmt.Printf(Warn+"Failed to marshal data %s\n", err)
		return
	}
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteAddContent,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}
	for _, content := range addWebsite.Content {
		fmt.Printf(Info+"Content added (%s): %s \n", content.ContentType, content.Path)
	}
}

func removeWebsiteContent(ctx *grumble.Context, rpc RPCServer) {
	rmWeb := &clientpb.Website{
		Name:    ctx.Flags.String("website"),
		Content: map[string]*clientpb.WebContent{},
	}
	webpath := ctx.Flags.String("web-path")
	rmWeb.Content[webpath] = &clientpb.WebContent{}

	data, _ := proto.Marshal(rmWeb)
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteRemoveContent,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}
	for webpath := range rmWeb.Content {
		fmt.Printf(Info+"Removed %s\n", webpath)
	}
}

func webDisplay(web *clientpb.Website) {
	fmt.Println(Info + web.Name)
	fmt.Println()
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "Path\tContent-type\tSize\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Path")),
		strings.Repeat("=", len("Content-type")),
		strings.Repeat("=", len("Size")))
	for path, content := range web.Content {
		fmt.Fprintf(table, "%s\t%s\t%d\t\n", path, content.ContentType, content.Size)
	}
	table.Flush()
}

func webAddDirectory(web *clientpb.Website, webpath string, contentPath string) {
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

func webAddFile(web *clientpb.Website, webpath string, contentType string, contentPath string) error {

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

	web.Content[webpath] = &clientpb.WebContent{
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
