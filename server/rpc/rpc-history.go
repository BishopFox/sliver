package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/grpc/status"
)

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

// GetImplantHistory returns a list of commands ran on an implant, by a given user or all of them.
func (rpc *Server) GetImplantHistory(ctx context.Context, req *clientpb.HistoryRequest) (*clientpb.History, error) {
	commonName := rpc.getClientCommonName(ctx)
	logsDir, err := getImplantHistoryDir()
	if err != nil {
		rpcClientLogs.Errorf("Failed to get implant log directory: %s", err)
		return nil, err
	}

	// Don't create if the history file does not exist.
	streamName := fmt.Sprintf("%s_%s", req.ImplantName, req.ImplantID)
	rpcClientLogs.Infof("Opening file %s", streamName)
	logPath := filepath.Join(logsDir, streamName+".history")
	if _, err := os.Stat(logPath); err != nil {
		if os.IsNotExist(err) {
			return &clientpb.History{}, nil
		}

		return nil, err
	}

	// Read the file and unmarshal in here.
	var commands []*clientpb.ImplantCommand

	data, err := os.ReadFile(logPath)
	if err != nil {
		rpcClientLogs.Errorf("Failed to read implant history file: %s", err)
		return nil, err
	}

	err = json.Unmarshal(formatJSONList(data), &commands)
	if err != nil {
		rpcClientLogs.Error(err)
	}

	history := &clientpb.History{}

	// Get only user commands if required to.
	if req.UserOnly {
		history.UserOnly = true

		for _, cmd := range commands {
			if cmd.GetUser() == commonName {
				history.Commands = append(history.Commands, cmd)
			}
		}
	} else {
		history.Commands = commands
	}

	// And if requested for only a certain number of commands, cut the list.
	if req.MaxLines > 0 && int(req.MaxLines) < len(history.Commands) {
		oldest := len(history.Commands) - int(req.MaxLines)
		history.Commands = history.Commands[oldest:]
	}

	history.HistoryLen = int32(len(commands))

	return history, nil
}

// ImplantHistory is used by clients to log the command lines they execute on implants.
func (rpc *Server) ImplantHistory(stream rpcpb.SliverRPC_ImplantHistoryServer) error {
	commonName := rpc.getClientCommonName(stream.Context())
	logsDir, err := getImplantHistoryDir()
	if err != nil {
		rpcClientLogs.Errorf("Failed to get implant log directory: %s", err)
		return err
	}

	streams := make(map[string]*LogStream)
	defer func() {
		for _, stream := range streams {
			rpcClientLogs.Infof("Closing implant log file: %s", stream.logFile.Name())
			stream.logFile.Close()
		}
	}()

	for {
		fromClient, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// gRPC errors are a pain to work with...
			canceled := errors.New(context.Canceled.Error())

			if !errors.As(errors.New(status.Convert(err).Message()), &canceled) {
				rpcClientLogs.Errorf("Failed to receive implant history data: %s", err)
			}
			return err
		}

		streamName := fmt.Sprintf("%s_%s", fromClient.ImplantName, fromClient.ImplantID)

		// Remove useless fields and write to file.
		fromClient.User = commonName
		fromClient.Request = nil

		data, err := json.Marshal(fromClient)
		if err != nil {
			return err
		}

		data = append([]byte(",\n"), data...)

		if _, ok := streams[streamName]; !ok {
			streams[streamName], err = openNewHistoryStream(logsDir, streamName)
			if err != nil {
				rpcClientLogs.Errorf("Failed to open implant history log file: %s", err)
				return err
			}
		}
		rpcClientLogs.Debugf("Received %d bytes of implant history data for %s", len(data), streamName)
		streams[streamName].Write(data)
	}
	return nil
}

func getImplantHistoryDir() (string, error) {
	parentLogDir := filepath.Join(log.GetLogDir(), "implants")
	if err := os.MkdirAll(parentLogDir, 0o700); err != nil {
		rpcClientLogs.Warnf("Failed to create client console log directory: %s", err)
		return "", err
	}
	return parentLogDir, nil
}

func openNewHistoryStream(logsDir string, stream string) (*LogStream, error) {
	if !streamNamePattern.MatchString(stream) {
		return nil, ErrInvalidStreamName
	}
	stream = filepath.Base(stream)
	logPath := filepath.Join(logsDir, filepath.Base(stream+".history"))
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &LogStream{stream: stream, parts: 0, logFile: logFile, lock: &sync.Mutex{}}, nil
}

func formatJSONList(data []byte) []byte {
	data = []byte(strings.TrimPrefix(string(data), ",\n"))
	data = append(data, ']')
	data = append([]byte("["), data...)

	return data
}
