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
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/implant/sliver/handlers/matcher"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func determineDirPathFilter(targetPath string) (string, string) {
	// The filter
	filter := ""

	// The path the filter applies to
	path := ""

	/*
		Check to see if the remote path is a filter or contains a filter.
		If the path passes the test to be a filter, then it is a filter
		because paths are not valid filters.
	*/
	if targetPath != "." {

		// Check if the path contains a filter
		// Test on a standardized version of the path (change any \ to /)
		testPath := strings.Replace(targetPath, "\\", "/", -1)
		/*
			Cannot use the path or filepath libraries because the OS
			of the client does not necessarily match the OS of the
			implant
		*/
		lastSeparatorOccurrence := strings.LastIndex(testPath, "/")

		if lastSeparatorOccurrence == -1 {
			// Then this is only a filter
			filter = targetPath
			path = "."
		} else {
			// Then we need to test for a filter on the end of the string

			// The indices should be the same because we did not change the length of the string
			path = targetPath[:lastSeparatorOccurrence+1]
			filter = targetPath[lastSeparatorOccurrence+1:]
		}
	} else {
		path = targetPath
		// filter remains blank
	}

	return path, filter
}

func standarizeArchiveFileName(path string, uploadPathSpecified string) string {
	// Change all backslashes to forward slashes
	var standardFilePath string = strings.ReplaceAll(path, "\\", "/")
	var standardUploadPath string = strings.ReplaceAll(uploadPathSpecified, "\\", "/")

	/*
		Remove the volume / root from the directory
		path. This makes it so that files on the
		system where the archive is extracted to
		will not be clobbered.

		Tried with filepath.VolumeName, but that function
		does not work reliably with Windows paths
	*/
	if strings.HasPrefix(standardFilePath, "//") {
		// If this a UNC path, filepath.Rel is not going to work
		return standardFilePath[2:]
	} else {
		beginningOfUploadPath, _, _ := strings.Cut(standardUploadPath, "/")
		if index := strings.Index(standardFilePath, beginningOfUploadPath); index != -1 {
			// Add a / in front to make the specified upload path the root
			standardFilePath = "/" + standardFilePath[index:]
		}
		// Calculate a path relative to the root
		pathParts := strings.SplitN(standardFilePath, "/", 2)
		if len(pathParts) < 2 {
			// Then something is wrong with this path
			return standardFilePath
		}

		basePath := pathParts[0]
		fileRelPath := pathParts[1]

		if basePath == "" {
			// If base path is blank, that means it started with / and / is the root
			return fileRelPath
		} else {
			/*
				Then this is almost certainly Windows, and we will set the archive up
				so that it preserves the path but without the colon.
				Something like:
				c/windows/system32/file.whatever
				Colons are not legal in Windows filenames, so let's get rid of them.
			*/
			basePath = strings.ReplaceAll(basePath, ":", "")
			basePath = strings.ToLower(basePath)
			return basePath + "/" + fileRelPath
		}
	}
}

func tarDirectory(sourcePath string, pathAsSpecified string, sourceFilter string, recurse bool, preserveDirStructure bool) ([]byte, int, int, error) {
	readFiles := 0
	unreadableFiles := 0
	var matchingFiles []string
	var buffer bytes.Buffer
	tarWriter := tar.NewWriter(&buffer)

	/*
		Build the list of files to include in the archive.

		Walking the directory can take a long time and do a lot of unnecessary work
		if we do not need to recurse through subdirectories.

		If we are not recursing, then read the directory without worrying about
		subdirectories.
	*/
	if !recurse {
		testPath := strings.ReplaceAll(sourcePath, "\\", "/")
		directory, err := os.Open(sourcePath)
		if err != nil {
			return nil, readFiles, unreadableFiles, err
		}
		directoryFiles, err := directory.Readdirnames(0)
		directory.Close()
		if err != nil {
			return nil, readFiles, unreadableFiles, err
		}

		for _, fileName := range directoryFiles {
			standardFileName := strings.ReplaceAll(testPath+fileName, "\\", "/")
			if sourceFilter != "" {
				match, err := matcher.Match(sourcePath+sourceFilter, standardFileName)
				if err == nil && match {
					matchingFiles = append(matchingFiles, standardFileName)
				}
			} else {
				matchingFiles = append(matchingFiles, standardFileName)
			}
		}
	} else {
		filepath.WalkDir(sourcePath, func(file string, d os.DirEntry, err error) error {
			filePath := strings.ReplaceAll(file, "\\", "/")
			if sourceFilter != "" {
				// Normalize paths
				testPath := strings.ReplaceAll(filepath.Dir(file), "\\", "/") + "/"
				match, matchErr := matcher.Match(testPath+sourceFilter, filePath)
				if !match || matchErr != nil {
					// If there is an error, it is because the filter is bad, so it is not a match
					return nil
				}
				matchingFiles = append(matchingFiles, file)
			} else {
				matchingFiles = append(matchingFiles, file)
			}
			return nil
		})
	}

	var fileName string

	for _, file := range matchingFiles {
		fi, err := os.Stat(file)
		if err != nil {
			// Cannot get info on the file, so skip it
			unreadableFiles += 1
			continue
		}

		if preserveDirStructure {
			fileName = standarizeArchiveFileName(file, pathAsSpecified)
		} else {
			fileName = fi.Name()
		}

		// If the file is a SymLink replace fileInfo and path with the symlink destination.
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			file, err = filepath.EvalSymlinks(file)
			if err != nil {
				unreadableFiles += 1
				continue
			}

			fi, err = os.Lstat(file)
			if err != nil {
				unreadableFiles += 1
				continue
			}
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			unreadableFiles += 1
			continue
		}
		// Keep the symlink file path for the header name.
		header.Name = filepath.ToSlash(fileName)
		// Check that we can open the file before we try to write it to the archive
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				// Skip this file and do not write it to the archive.
				unreadableFiles += 1
				continue
			}
			if err := tarWriter.WriteHeader(header); err != nil {
				unreadableFiles += 1
				data.Close()
				continue
			}
			if _, err := io.Copy(tarWriter, data); err != nil {
				unreadableFiles += 1
				data.Close()
				continue
			}
			data.Close()
			readFiles += 1
		} else {
			if err := tarWriter.WriteHeader(header); err != nil {
				unreadableFiles += 1
				continue
			}
		}
	}

	if err := tarWriter.Close(); err != nil {
		return buffer.Bytes(), readFiles, unreadableFiles, err
	}
	return buffer.Bytes(), readFiles, unreadableFiles, nil
}

// UploadCmd - Upload a file to the remote system
func UploadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	readFiles := 0
	unreadableFiles := 0
	var isDirectory bool
	var fileName string
	var sourceInfomation fs.FileInfo
	var uploadData []byte

	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	remotePath := ""

	localPath := args[0]
	if len(args) > 1 {
		remotePath = args[1]
	}

	recurse, _ := cmd.Flags().GetBool("recurse")
	preserve, _ := cmd.Flags().GetBool("preserve")
	isIOC, _ := cmd.Flags().GetBool("ioc")
	overwrite, _ := cmd.Flags().GetBool("overwrite")

	if localPath == "" {
		con.PrintErrorf("Missing parameter, see `help upload`\n")
		return
	}

	src, _ := filepath.Abs(localPath)

	if remotePath == "" {
		remotePath = fileName
	}

	dst := remotePath

	// Get information on the source - is it a directory?
	srcPath, srcFilter := determineDirPathFilter(src)
	sourceInfomation, err := os.Stat(src)

	if err == nil && sourceInfomation.IsDir() {
		if !strings.HasSuffix(src, string(os.PathSeparator)) {
			src += string(os.PathSeparator)
		}
		srcPath = src
		srcFilter = ""
	}

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Check to see if src ends with a filter
			sourceInfomation, err = os.Stat(srcPath)
			if errors.Is(err, fs.ErrNotExist) {
				con.PrintErrorf("The source %s does not exist.\n", src)
				return
			} else if errors.Is(err, fs.ErrPermission) {
				con.PrintErrorf("Permissions error when trying to read %s.\n", src)
				return
			}
		}
		if errors.Is(err, fs.ErrPermission) {
			con.PrintErrorf("Permissions error when trying to read %s.\n", src)
			return
		}
	}

	// If we still do not have information about the source at this point, bail.
	if sourceInfomation == nil {
		con.PrintErrorf("Could not get information about the upload source %s\n", src)
		return
	}

	if sourceInfomation.IsDir() {
		// tar the directory and send it over
		uploadData, readFiles, unreadableFiles, err = tarDirectory(srcPath, localPath, srcFilter, recurse, preserve)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		if readFiles == 0 {
			con.PrintErrorf("Could not find any files matching %s\n", src)
			return
		}
		isDirectory = true
		fileName = ""
	} else {
		uploadData, err = os.ReadFile(src)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		readFiles = 1
		unreadableFiles = 0
		isDirectory = false
		fileName = filepath.Base(src)
	}

	uploadGzip, _ := new(encoders.Gzip).Encode(uploadData)

	var readFilesQualifier string
	var unsuccessfulFiles string

	if readFiles == 1 {
		readFilesQualifier = ""
	} else {
		readFilesQualifier = "s"
	}

	if unreadableFiles > 0 {
		if unreadableFiles == 1 {
			unsuccessfulFiles = fmt.Sprintf(" (could not read %d file)", unreadableFiles)
		} else {
			unsuccessfulFiles = fmt.Sprintf(" (could not read %d files)", unreadableFiles)
		}
	} else {
		unsuccessfulFiles = ""
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Uploading %d file%s %sfrom %s to %s", readFiles,
		readFilesQualifier,
		unsuccessfulFiles,
		src,
		dst), ctrl)
	upload, err := con.Rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Request:     con.ActiveTarget.Request(cmd),
		Path:        dst,
		Data:        uploadGzip,
		Encoder:     "gzip",
		IsIOC:       isIOC,
		FileName:    fileName,
		IsDirectory: isDirectory,
		Overwrite:   overwrite,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if upload.Response != nil && upload.Response.Async {
		con.AddBeaconCallback(upload.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, upload)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintUpload(upload, con)
		})
		con.PrintAsyncResponse(upload.Response)
	} else {
		PrintUpload(upload, con)
	}
}

// PrintUpload - Print the result of the upload command.
func PrintUpload(upload *sliverpb.Upload, con *console.SliverClient) {
	if upload.Response != nil && upload.Response.Err != "" {
		con.PrintErrorf("%s\n", upload.Response.Err)
		return
	}
	writtenFileQualifier := "s"
	if upload.WrittenFiles == 1 {
		writtenFileQualifier = ""
	}
	unwrittenFileQualifier := "s"
	if upload.UnwriteableFiles == 1 {
		unwrittenFileQualifier = ""
	}
	if upload.WrittenFiles > 0 {
		if upload.UnwriteableFiles > 0 {
			con.PrintInfof("Wrote %d file%s successfully (%d file%s not written) to %s\n", upload.WrittenFiles, writtenFileQualifier, upload.UnwriteableFiles, unwrittenFileQualifier, upload.Path)
		} else {
			con.PrintInfof("Wrote %d file%s successfully to %s\n", upload.WrittenFiles, writtenFileQualifier, upload.Path)
		}
	} else {
		con.PrintInfof("Could not write %d file%s to %s: the files already exist on the target\n", upload.UnwriteableFiles, unwrittenFileQualifier, upload.Path)
	}
}
