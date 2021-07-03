package command

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
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/encoders"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/desertbit/grumble"
)

func Ls(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	remotePath := ctx.Args.String("path")

	ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    remotePath,
	})
	if err != nil {
		con.PrintWarnf("%s\n", err)
	} else {
		PrintDirList(con.App.Stdout(), ls)
	}
}

func PrintDirList(stdout io.Writer, dirList *sliverpb.Ls) {
	fmt.Fprintf(stdout, "%s\n", dirList.Path)
	fmt.Fprintf(stdout, "%s\n", strings.Repeat("=", len(dirList.Path)))

	table := tabwriter.NewWriter(stdout, 0, 2, 2, ' ', 0)
	for _, fileInfo := range dirList.Files {
		if fileInfo.IsDir {
			fmt.Fprintf(table, "%s\t<dir>\t\n", fileInfo.Name)
		} else {
			fmt.Fprintf(table, "%s\t%s\t\n", fileInfo.Name, util.ByteCountBinary(fileInfo.Size))
		}
	}
	table.Flush()
}

func Rm(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	filePath := ctx.Args.String("path")

	if filePath == "" {
		con.PrintErrorf("Missing parameter: file or directory name\n")
		return
	}

	rm, err := con.Rpc.Rm(context.Background(), &sliverpb.RmReq{
		Request:   con.ActiveSession.Request(ctx),
		Path:      filePath,
		Recursive: ctx.Flags.Bool("recursive"),
		Force:     ctx.Flags.Bool("force"),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("%s\n", rm.Path)
	}
}

func Mkdir(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	filePath := ctx.Args.String("path")

	if filePath == "" {
		con.PrintErrorf("Missing parameter: directory name\n")
		return
	}

	mkdir, err := con.Rpc.Mkdir(context.Background(), &sliverpb.MkdirReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    filePath,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("%s\n", mkdir.Path)
	}
}

func Cd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	filePath := ctx.Args.String("path")

	pwd, err := con.Rpc.Cd(context.Background(), &sliverpb.CdReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    filePath,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("%s\n", pwd.Path)
	}
}

func Pwd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	pwd, err := con.Rpc.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: con.ActiveSession.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("%s\n", pwd.Path)
	}
}

func Cat(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	filePath := ctx.Args.String("path")
	if filePath == "" {
		con.PrintErrorf("Missing parameter: file name\n")
		return
	}

	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    filePath,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	if ctx.Flags.Bool("colorize-output") {
		if err = colorize(download); err != nil {
			fmt.Println(string(download.Data))
		}
	} else {
		fmt.Println(string(download.Data))
	}
	// if ctx.Flags.Bool("loot") && 0 < len(download.Data) {
	// 	err = AddLootFile(rpc, fmt.Sprintf("[cat] %s", filepath.Base(filePath)), filePath, download.Data, false)
	// 	if err != nil {
	// 		con.PrintErrorf("Failed to save output as loot: %s", err)
	// 	} else {
	// 		fmt.Printf(clearln + Info + "Output saved as loot\n")
	// 	}
	// }
}

func colorize(f *sliverpb.Download) error {
	lexer := lexers.Match(f.GetPath())
	if lexer == nil {
		lexer = lexers.Analyse(string(f.GetData()))
		if lexer == nil {
			lexer = lexers.Fallback
		}
	}
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	if lexer != nil {
		iterator, err := lexer.Tokenise(nil, string(f.GetData()))
		if err != nil {
			return err
		}
		err = formatter.Format(os.Stdout, style, iterator)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no lexer found")
	}
	return nil
}

func Download(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	remotePath := ctx.Args.String("remote-path")
	localPath := ctx.Args.String("local-path")

	src := remotePath
	fileName := filepath.Base(src)
	dst, _ := filepath.Abs(localPath)
	fi, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		con.PrintErrorf("%s\n", err)
		return
	}
	if err == nil && fi.IsDir() {
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
	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    remotePath,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("Decoding failed %s", err)
			return
		}
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		con.PrintErrorf("Failed to open local file %s: %s\n", dst, err)
		return
	}
	defer dstFile.Close()
	n, err := dstFile.Write(download.Data)
	if err != nil {
		con.PrintErrorf("Failed to write data %v\n", err)
	} else {
		con.PrintInfof("Wrote %d bytes to %s\n", n, dstFile.Name())
	}

	// if ctx.Flags.Bool("loot") && 0 < len(download.Data) {
	// 	err = AddLootFile(rpc, fmt.Sprintf("[download] %s", filepath.Base(remotePath)), remotePath, download.Data, false)
	// 	if err != nil {
	// 		con.PrintErrorf("Failed to save output as loot: %s", err)
	// 	} else {
	// 		fmt.Printf(Info + "Output saved as loot\n")
	// 	}
	// }
}

func Upload(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	localPath := ctx.Args.String("local-path")
	remotePath := ctx.Args.String("remote-path")

	if localPath == "" {
		con.PrintErrorf("Missing parameter, see `help upload`\n")
		return
	}

	src, _ := filepath.Abs(localPath)
	_, err := os.Stat(src)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if remotePath == "" {
		fileName := filepath.Base(src)
		remotePath = fileName
	}
	dst := remotePath

	fileBuf, err := ioutil.ReadFile(src)
	uploadGzip := new(encoders.Gzip).Encode(fileBuf)

	ctrl := make(chan bool)
	go spin.Until(fmt.Sprintf("%s -> %s", src, dst), ctrl)
	upload, err := con.Rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    dst,
		Data:    uploadGzip,
		Encoder: "gzip",
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Wrote file to %s\n", upload.Path)
	}
}
