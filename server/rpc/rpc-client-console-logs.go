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
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/log"
)

var (
	rpcClientConsoleLog = log.NamedLogger("rpc", "client-console-log")
)

// ClientLogData - Send client console log data
func (rpc *Server) ClientConsoleLog(stream rpcpb.SliverRPC_ClientConsoleLogServer) error {
	commonName := rpc.getClientCommonName(stream.Context())
	logsDir, err := getClientConsoleLogDir(commonName)
	if err != nil {
		return err
	}

	dateTime := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("%s.gzip", dateTime))
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		rpcClientConsoleLog.Warnf("Failed to open client console log file: %s", err)
		return err
	}
	defer logFile.Close()
	logGzip, _ := gzip.NewWriterLevel(logFile, gzip.BestCompression)
	defer logGzip.Close()
	for {
		fromClient, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		_, err = logGzip.Write(fromClient.Data)
		if err != nil {
			return err
		}
	}
	return nil
}

func getClientConsoleLogDir(client string) (string, error) {
	parentLogDir := filepath.Join(log.GetLogDir(), "client-console")
	if err := os.MkdirAll(parentLogDir, 0700); err != nil {
		rpcClientConsoleLog.Warnf("Failed to create client console log directory: %s", err)
		return "", err
	}
	logDir := filepath.Join(parentLogDir, filepath.Base(client))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		rpcClientConsoleLog.Warnf("Failed to create client console log directory: %s", err)
		return "", err
	}
	return logDir, nil
}
