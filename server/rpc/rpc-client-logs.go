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
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	rpcClientConsoleLog = log.NamedLogger("rpc", "client-console-log")

	ErrInvalidStreamName = status.Error(codes.InvalidArgument, "Invalid stream name")

	streamName = regexp.MustCompile("^[a-z0-9_-]+$")
)

type LogStream struct {
	logFile *os.File
	logGzip *gzip.Writer
}

// ClientLogData - Send client console log data
func (rpc *Server) ClientLog(stream rpcpb.SliverRPC_ClientLogServer) error {
	commonName := rpc.getClientCommonName(stream.Context())
	logsDir, err := getClientLogsDir(commonName)
	if err != nil {
		return err
	}
	streams := make(map[string]*LogStream)
	defer func() {
		for _, stream := range streams {
			stream.logGzip.Close()
			stream.logFile.Close()
		}
	}()
	for {
		fromClient, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		streamName := fromClient.GetStream()
		if _, ok := streams[streamName]; !ok {
			streams[streamName], err = openNewLogStream(logsDir, streamName)
			if err != nil {
				return err
			}
		}
		streams[streamName].logGzip.Write(fromClient.GetData())
	}
	return nil
}

func openNewLogStream(logsDir string, stream string) (*LogStream, error) {
	if !streamName.MatchString(stream) {
		return nil, ErrInvalidStreamName
	}
	stream = filepath.Base(stream)
	dateTime := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, filepath.Base(fmt.Sprintf("%s_%s.gzip", stream, dateTime)))
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		rpcClientConsoleLog.Warnf("Failed to open client console log file: %s", err)
		return nil, err
	}
	logGzip, _ := gzip.NewWriterLevel(logFile, gzip.BestCompression)
	return &LogStream{logFile: logFile, logGzip: logGzip}, nil
}

func getClientLogsDir(client string) (string, error) {
	parentLogDir := filepath.Join(log.GetLogDir(), "clients")
	if err := os.MkdirAll(parentLogDir, 0o700); err != nil {
		rpcClientConsoleLog.Warnf("Failed to create client console log directory: %s", err)
		return "", err
	}
	logDir := filepath.Join(parentLogDir, filepath.Base(client))
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		rpcClientConsoleLog.Warnf("Failed to create client console log directory: %s", err)
		return "", err
	}
	return logDir, nil
}
