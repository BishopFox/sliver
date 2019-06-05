package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/spin"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/util"

	"github.com/AlecAivazis/survey"
	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func ls(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	data, _ := proto.Marshal(&sliverpb.LsReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgLsReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	dirList := &sliverpb.Ls{}
	err := proto.Unmarshal(resp.Data, dirList)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	printDirList(dirList)
}

func printDirList(dirList *sliverpb.Ls) {
	fmt.Printf("%s\n", dirList.Path)
	fmt.Printf("%s\n", strings.Repeat("=", len(dirList.Path)))

	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, fileInfo := range dirList.Files {
		if fileInfo.IsDir {
			fmt.Fprintf(table, "%s\t<dir>\t\n", fileInfo.Name)
		} else {
			fmt.Fprintf(table, "%s\t%s\t\n", fileInfo.Name, util.ByteCountBinary(fileInfo.Size))
		}
	}
	table.Flush()
}

func rm(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: file or directory name\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.RmReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgRmReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	rm := &sliverpb.Rm{}
	err := proto.Unmarshal(resp.Data, rm)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if rm.Success {
		fmt.Printf(Info+"%s\n", rm.Path)
	} else {
		fmt.Printf(Warn+"%s\n", rm.Err)
	}

}

func mkdir(ctx *grumble.Context, rpc RPCServer) {

	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: directory name\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.MkdirReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgMkdirReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	mkdir := &sliverpb.Mkdir{}
	err := proto.Unmarshal(resp.Data, mkdir)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if mkdir.Success {
		fmt.Printf(Info+"%s\n", mkdir.Path)
	} else {
		fmt.Printf(Warn+"%s\n", mkdir.Err)
	}
}

func cd(ctx *grumble.Context, rpc RPCServer) {

	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) < 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	data, _ := proto.Marshal(&sliverpb.CdReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgCdReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	pwd := &sliverpb.Pwd{}
	err := proto.Unmarshal(resp.Data, pwd)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	fmt.Printf(Info+"%s\n", pwd.Path)
}

func pwd(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.PwdReq{
		SliverID: ActiveSliver.Sliver.ID,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgPwdReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	pwd := &sliverpb.Pwd{}
	err := proto.Unmarshal(resp.Data, pwd)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	fmt.Printf(Info+"%s\n", pwd.Path)
}

func cat(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing parameter: file name\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.DownloadReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgDownloadReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	download := &sliverpb.Download{}
	proto.Unmarshal(resp.Data, download)
	if download.Encoder == "gzip" {
		download.Data, _ = new(util.Gzip).Decode(download.Data)
	}
	fmt.Printf(string(download.Data))
}

func download(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second

	if len(ctx.Args) < 1 {
		fmt.Println(Warn + "Missing parameter(s), see `help download`\n")
		return
	}
	if len(ctx.Args) == 1 {
		ctx.Args = append(ctx.Args, ".")
	}

	src := ctx.Args[0]
	fileName := filepath.Base(src)
	dst, _ := filepath.Abs(ctx.Args[1])
	fi, err := os.Stat(dst)
	if err != nil {
		fmt.Printf(Warn+"%v\n", err)
		return
	}
	if fi.IsDir() {
		dst = path.Join(dst, fileName)
	}

	if _, err := os.Stat(dst); err == nil {
		overwrite := false
		prompt := &survey.Confirm{Message: "Overwrite local file?"}
		survey.AskOne(prompt, &overwrite, nil)
		if !overwrite {
			return
		}
	}

	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("%s -> %s", fileName, dst), ctrl)
	data, _ := proto.Marshal(&sliverpb.DownloadReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     ctx.Args[0],
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgDownloadReq,
		Data: data,
	}, cmdTimeout)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	download := &sliverpb.Download{}
	proto.Unmarshal(resp.Data, download)
	if download.Encoder == "gzip" {
		download.Data, _ = new(util.Gzip).Decode(download.Data)
	}
	f, err := os.Create(dst)
	if err != nil {
		fmt.Printf(Warn+"Failed to open local file %s: %v\n", dst, err)
	}
	defer f.Close()
	n, err := f.Write(download.Data)
	if err != nil {
		fmt.Printf(Warn+"Failed to write data %v\n", err)
	} else {
		fmt.Printf(Info+"Wrote %d bytes to %s\n", n, dst)
	}
}

func upload(ctx *grumble.Context, rpc RPCServer) {

	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	if len(ctx.Args) < 1 {
		fmt.Println(Warn + "Missing parameter, see `help upload`\n")
		return
	}

	cmdTimeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second

	src, _ := filepath.Abs(ctx.Args[0])
	_, err := os.Stat(src)
	if err != nil {
		fmt.Printf(Warn+"%v\n", err)
		return
	}

	if len(ctx.Args) == 1 {
		fileName := filepath.Base(src)
		ctx.Args = append(ctx.Args, fileName)
	}
	dst := ctx.Args[1]

	fileBuf, err := ioutil.ReadFile(src)
	uploadGzip := bytes.NewBuffer([]byte{})
	new(util.Gzip).Encode(uploadGzip, fileBuf)

	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("%s -> %s", src, dst), ctrl)
	data, _ := proto.Marshal(&sliverpb.UploadReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     dst,
		Data:     uploadGzip.Bytes(),
		Encoder:  "gzip",
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgUploadReq,
		Data: data,
	}, cmdTimeout)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	upload := &sliverpb.Upload{}
	err = proto.Unmarshal(resp.Data, upload)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if upload.Success {
		fmt.Printf(clearln+Info+"Written to %s\n", upload.Path)
	} else {
		fmt.Printf(Warn+"Error %s\n", upload.Err)
	}

}
