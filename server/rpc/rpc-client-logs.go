package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"compress/gzip"
	"fmt"
	"io"
	insecureRand "math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrInvalidStreamName - Invalid stream name
	ErrInvalidStreamName = status.Error(codes.InvalidArgument, "Invalid stream name")

	rpcClientLogs     = log.NamedLogger("rpc", "client-logs")
	streamNamePattern = regexp.MustCompile("^[a-z0-9_-]+$")
)

type LogStream struct {
	stream  string
	logFile *os.File
	lock    *sync.Mutex
	parts   int
}

func (l *LogStream) Write(data []byte) (int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	n, err := l.logFile.Write(data)
	if err != nil {
		return n, err
	}
	l.logFile.Sync()
	fi, err := os.Stat(l.logFile.Name())
	if err != nil {
		return n, err
	}
	// Rotate log file if it's over 50MB
	if fi.Size() > (1024 * 1024 * 50) {
		rpcClientLogs.Infof("Rotating client console log file: %s", l.logFile.Name())
		l.logFile.Close()
		l.parts++
		fileName := l.logFile.Name()
		partFileName := fileName + fmt.Sprintf(".part%d", l.parts)
		err = os.Rename(fileName, partFileName)
		if err != nil {
			return n, err
		}
		go gzipFile(partFileName)
		l.logFile, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			return n, err
		}
	}
	return n, err
}

// ClientLogData - Send client console log data
func (rpc *Server) ClientLog(stream rpcpb.SliverRPC_ClientLogServer) error {
	commonName := rpc.getClientCommonName(stream.Context())
	logsDir, err := getClientLogsDir(commonName)
	if err != nil {
		rpcClientLogs.Errorf("Failed to get client console log directory: %s", err)
		return err
	}
	streams := make(map[string]*LogStream)
	defer func() {
		for _, stream := range streams {
			rpcClientLogs.Infof("Closing client console log file: %s", stream.logFile.Name())
			stream.logFile.Close()
		}
	}()
	for {
		fromClient, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			rpcClientLogs.Errorf("Failed to receive client console log data: %s", err)
			return err
		}
		streamName := fromClient.GetStream()
		if _, ok := streams[streamName]; !ok {
			streams[streamName], err = openNewLogStream(logsDir, streamName)
			if err != nil {
				rpcClientLogs.Errorf("Failed to open client console log file: %s", err)
				return err
			}
		}
		rpcClientLogs.Debugf("Received %d bytes of client console log data for stream %s", len(fromClient.GetData()), streamName)
		streams[streamName].Write(fromClient.GetData())
	}
	return nil
}

func openNewLogStream(logsDir string, stream string) (*LogStream, error) {
	if !streamNamePattern.MatchString(stream) {
		return nil, ErrInvalidStreamName
	}
	stream = filepath.Base(stream)
	dateTime := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, filepath.Base(fmt.Sprintf("%s_%s.log", stream, dateTime)))
	if _, err := os.Stat(logPath); err == nil {
		rpcClientLogs.Warnf("Client console log file already exists: %s", logPath)
		logPath = filepath.Join(logsDir, filepath.Base(fmt.Sprintf("%s_%s_%s.log", stream, dateTime, randomSuffix(6))))
	}
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &LogStream{stream: stream, parts: 0, logFile: logFile, lock: &sync.Mutex{}}, nil
}

func randomSuffix(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	buf := make([]rune, n)
	for i := range buf {
		buf[i] = letterRunes[insecureRand.Intn(len(letterRunes))]
	}
	return string(buf)
}

func getClientLogsDir(client string) (string, error) {
	parentLogDir := filepath.Join(log.GetLogDir(), "clients")
	if err := os.MkdirAll(parentLogDir, 0o700); err != nil {
		rpcClientLogs.Warnf("Failed to create client console log directory: %s", err)
		return "", err
	}
	logDir := filepath.Join(parentLogDir, filepath.Base(client))
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		rpcClientLogs.Warnf("Failed to create client console log directory: %s", err)
		return "", err
	}
	return logDir, nil
}

func gzipFile(filePath string) {
	inputFile, err := os.Open(filePath)
	if err != nil {
		rpcClientLogs.Errorf("Failed to open client console log file: %s", err)
		return
	}
	defer inputFile.Close()
	outFile, err := os.OpenFile(filePath+".gz", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		rpcClientLogs.Errorf("Failed to open gz client console log file: %s", err)
		return
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()
	_, err = io.Copy(gzWriter, inputFile)
	if err != nil {
		rpcClientLogs.Errorf("Failed to gzip client console log file: %s", err)
		return
	}
	inputFile.Close()
	err = os.Remove(filePath)
	if err != nil {
		rpcClientLogs.Errorf("Failed to remove client console log file: %s", err)
		return
	}
}
