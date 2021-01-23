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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/evilsocket/islazy/tui"
	"gopkg.in/AlecAivazis/survey.v1"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
)

// ChangeDirectory - Change the working directory of the client console
type ChangeDirectory struct {
	Positional struct {
		Path string `description:"remote path" required:"1-1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Handler for ChangeDirectory
func (cd *ChangeDirectory) Execute(args []string) (err error) {

	path := cd.Positional.Path
	if (path == "~" || path == "~/") && cctx.Context.Sliver.OS == "linux" {
		path = filepath.Join("/home", cctx.Context.Sliver.Username)
	}

	pwd, err := transport.RPC.Cd(context.Background(), &sliverpb.CdReq{
		Path:    path,
		Request: ContextRequest(cctx.Context.Sliver.Session),
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	} else {
		fmt.Printf(util.Info+"%s\n", pwd.Path)
		cctx.Context.Sliver.WorkingDir = pwd.Path
	}

	return
}

// ListSessionDirectories - List directory contents
type ListSessionDirectories struct {
	Positional struct {
		Path []string `description:"session directory/file"`
	} `positional-args:"yes"`
}

// Execute - Command
func (ls *ListSessionDirectories) Execute(args []string) error {

	if len(ls.Positional.Path) == 0 {
		ls.Positional.Path = []string{"."}
	}

	// Other paths/files
	for _, path := range ls.Positional.Path {
		if (path == "~" || path == "~/") && cctx.Context.Sliver.OS == "linux" {
			path = filepath.Join("/home", cctx.Context.Sliver.Username)
		}
		resp, err := transport.RPC.Ls(context.Background(), &sliverpb.LsReq{
			Path:    path,
			Request: ContextRequest(cctx.Context.Sliver.Session),
		})
		if err != nil {
			fmt.Printf(util.Error+"%s\n", err)
		} else {
			printDirList(resp)
		}
	}

	return nil
}

func printDirList(dirList *sliverpb.Ls) {
	title := fmt.Sprintf("%s%s%s%s", tui.BOLD, tui.BLUE, dirList.Path, tui.RESET)

	table := util.NewTable(title)
	headers := []string{"Name", "Size"}
	headLen := []int{0, 0}
	table.SetColumns(headers, headLen)

	for _, fileInfo := range dirList.Files {
		var row []string
		if fileInfo.IsDir {
			row = []string{tui.Blue(fileInfo.Name), ""}
		} else {
			row = []string{fileInfo.Name, util.ByteCountBinary(fileInfo.Size)}
		}
		table.AppendRow(row)
	}
	table.Output()
	fmt.Println()
}

// Rm - Remove a one or more files/directories from the implant target host.
type Rm struct {
	Positional struct {
		Path []string `description:"session directory/file" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Recursive bool `long:"recursive " short:"r" description:"recursively remove directory contents"`
		Force     bool `long:"force" short:"f" description:"ignore nonexistent files, never prompt"`
	} `group:"rm options"`
}

// Execute - Command
func (rm *Rm) Execute(args []string) (err error) {

	for _, other := range rm.Positional.Path {
		res, err := transport.RPC.Rm(context.Background(), &sliverpb.RmReq{
			Path:      other,
			Recursive: rm.Options.Recursive,
			Force:     rm.Options.Force,
			Request:   ContextRequest(cctx.Context.Sliver.Session),
		})
		if err != nil {
			fmt.Printf(util.Error+"%s\n", err)
		} else {
			fmt.Printf(util.Info+"Removed %s\n", res.Path)
		}
	}
	return
}

// Mkdir - Create one or more directories on the implant's host.
type Mkdir struct {
	Positional struct {
		Path []string `description:"directory name" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Command
func (md *Mkdir) Execute(args []string) (err error) {

	for _, other := range md.Positional.Path {
		mkdir, err := transport.RPC.Mkdir(context.Background(), &sliverpb.MkdirReq{
			Path:    other,
			Request: ContextRequest(cctx.Context.Sliver.Session),
		})
		if err != nil {
			fmt.Printf(util.Error+"%s\n", err)
		} else {
			fmt.Printf(util.Info+"%s\n", mkdir.Path)
		}
	}

	return
}

// Pwd - Print the session current working directory.
type Pwd struct{}

// Execute - Command
func (p *Pwd) Execute(args []string) (err error) {

	pwd, err := transport.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: ContextRequest(cctx.Context.Sliver.Session),
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
	} else {
		fmt.Printf(util.Info+"%s\n", pwd.Path)
	}

	return
}

// Cat - Print one or more files to screen
type Cat struct {
	Positional struct {
		Path []string `description:"remote file name" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Colorize bool `short:"c" long:"colorize" description:"colorize output according to file extension"`
	} `group:"rm options"`
}

// Execute - Command
func (c *Cat) Execute(args []string) (err error) {

	// Other files
	for _, other := range c.Positional.Path {
		download, err := transport.RPC.Download(context.Background(), &sliverpb.DownloadReq{
			Path:    other,
			Request: ContextRequest(cctx.Context.Sliver.Session),
		})
		if err != nil {
			fmt.Printf(util.Error+"%s\n", err)
			continue
		}
		if download.Encoder == "gzip" {
			download.Data, err = new(encoders.Gzip).Decode(download.Data)
			if err != nil {
				fmt.Printf(util.Error+"Encoder error: %s\n", err)
				return nil
			}
		}
		if c.Options.Colorize {
			if err = colorize(download); err != nil {
				fmt.Println(string(download.Data))
			}
		} else {
			fmt.Println(string(download.Data))
		}
	}

	return
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

// Download - Download one or more files from the target to the client.
type Download struct {
	Positional struct {
		LocalPath  string   `description:"console directory/file to save in/as" required:"1-1"`
		RemotePath []string `description:"remote directory name" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Command.
// Behavior is similar to Linu rm: any number of files arguments will be moved into
// the last, mandary path. The latter can be a file path if file arguments == 1,
// or a directory where everything will be moved when file arguments > 1
func (c *Download) Execute(args []string) (err error) {

	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	// Local destination
	dlDst, _ := filepath.Abs(c.Positional.LocalPath)
	fi, err := os.Stat(dlDst)
	if err != nil && !os.IsNotExist(err) {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}

	// If we have more than one file to download, the destination must be a directory.
	if len(c.Positional.RemotePath) > 1 && !fi.IsDir() {
		fmt.Printf(util.Error+"%s is not a directory (must be if you download multiple files)\n", dlDst)
		return nil
	}

	// This fucntion verifies that files are not directly overwritten
	var checkDestination = func(src, dst string) (dstFile string, err error) {
		// If our destination is a directory, adjust path
		if fi.IsDir() {
			fileName := filepath.Base(src)
			dst = path.Join(dst, fileName)
			if _, err := os.Stat(dst); err == nil {
				overwrite := false
				prompt := &survey.Confirm{Message: "Overwrite local file?"}
				survey.AskOne(prompt, &overwrite, nil)
				if !overwrite {
					return "", err
				}
			}
			return dst, nil
		}
		// Else directly check and prompt
		if _, err := os.Stat(dst); err == nil {
			overwrite := false
			prompt := &survey.Confirm{Message: "Overwrite local file?"}
			survey.AskOne(prompt, &overwrite, nil)
			if !overwrite {
				return "", err
			}
		}
		return dst, nil
	}

	// Prepare a download function & spinner to be used multiple times.
	var downloadFile = func(src string, dst string) {
		fileName := filepath.Base(src)

		ctrl := make(chan bool)
		go spin.Until(fmt.Sprintf("%s -> %s", fileName, dst), ctrl)
		download, err := transport.RPC.Download(context.Background(), &sliverpb.DownloadReq{
			Path:    src,
			Request: ContextRequest(session),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			fmt.Printf(util.Error+"%s\n", err)
			return
		}

		if download.Encoder == "gzip" {
			download.Data, err = new(encoders.Gzip).Decode(download.Data)
			if err != nil {
				fmt.Printf(util.Warn+"Decoding failed %s", err)
				return
			}
		}
		dstFile, err := os.Create(dst)
		if err != nil {
			fmt.Printf(util.Warn+"Failed to open local file %s: %s\n", dst, err)
			return
		}
		defer dstFile.Close()
		n, err := dstFile.Write(download.Data)
		if err != nil {
			fmt.Printf(util.Error+"Failed to write data %v\n", err)
		} else {
			fmt.Printf(util.Info+"Wrote %d bytes to %s\n", n, dstFile.Name())
		}
	}

	// For each file in the positional arguments, download
	for _, src := range c.Positional.RemotePath {
		dst, err := checkDestination(src, dlDst)
		if err != nil {
			continue
		}
		downloadFile(src, dst)
	}

	return
}

// Upload - Upload one or more files from the client to the target filesystem.
type Upload struct {
	Positional struct {
		RemotePath string   `description:"remote directory/file to save in/as" required:"1-1"`
		LocalPath  []string `description:"directory name" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Command.
// Behavior is similar to Linu rm: any number of files arguments will be moved into
// the last, mandary path. The latter can be a file path if file arguments == 1,
// or a directory where everything will be moved when file arguments > 1
func (c *Upload) Execute(args []string) (err error) {

	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	// If multile files to be uploaded, check destination is a directory.
	var dst string // Absolute path of destination directory, resolved below.
	if len(c.Positional.LocalPath) > 1 {
		resp, err := transport.RPC.Ls(context.Background(), &sliverpb.LsReq{
			Path:    c.Positional.RemotePath,
			Request: ContextRequest(session),
		})
		if err != nil {
			fmt.Printf(util.Error+" %s\n", err)
			return nil
		}
		if !resp.Exists {
			fmt.Printf(util.Error+" %s does not exists or is not a directory\n", c.Positional.RemotePath)
			return nil
		}
		dst = resp.Path
	}

	// For each file to upload, send data
	for _, file := range c.Positional.LocalPath {
		src, _ := filepath.Abs(file)
		_, err := os.Stat(src)
		if err != nil {
			fmt.Printf(util.Error+"%s\n", err)
			continue
		}
		fileBuf, err := ioutil.ReadFile(src)
		uploadGzip := new(encoders.Gzip).Encode(fileBuf)

		// Adjust dest with filename
		fileDst := filepath.Join(dst, filepath.Base(src))

		ctrl := make(chan bool)
		go spin.Until(fmt.Sprintf("%s -> %s", src, dst), ctrl)
		upload, err := transport.RPC.Upload(context.Background(), &sliverpb.UploadReq{
			Path:    fileDst,
			Data:    uploadGzip,
			Encoder: "gzip",
			Request: ContextRequest(session),
		})
		ctrl <- true
		<-ctrl
		if err != nil {
			fmt.Printf(util.Error+"Upload error: %s\n", err)
		} else {
			fmt.Printf(util.Info+"Wrote file to %s\n", upload.Path)
		}
	}

	return
}
