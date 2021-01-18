package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	Thiv program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/assets"
)

// GetHistory - A console requests a command history line.
func (c *Server) GetHistory(in context.Context, req *pb.HistoryRequest) (res *pb.History, err error) {

	// Get an ID/operator name for this client, so that the Comms system knows
	// where to route back connections that are meant for this client proxy/portfwd utilities.
	name := c.getClientCommonName(in)

	// Find file data, cut it and process it. If the name is empty,
	// we are the server and we write to a dedicated file.
	var filename string
	if name == "" {
		filename = filepath.Join(assets.GetRootAppDir(), ".history")
	} else {
		path := assets.GetUserHistoryDir(name)
		filename = filepath.Join(path, ".history")
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return &pb.History{Response: &commonpb.Response{Err: err.Error()}}, nil
	}
	lines := strings.Split(string(data), "\n") // For all returns we have a new line

	return &pb.History{Line: lines[req.Index], HistLength: int32(len(lines))}, nil
}

// AddToHistory - A client has sent a new command input line to be saved.
func (c *Server) AddToHistory(in context.Context, req *pb.AddCmdHistoryRequest) (res *pb.AddCmdHistory, err error) {

	// Get an ID/operator name for this client, so that the Comms system knows
	// where to route back connections that are meant for this client proxy/portfwd utilities.
	name := c.getClientCommonName(in)

	// Filter various useless commands
	if stringInSlice(strings.TrimSpace(req.Line), uselessCmds) {
		return &pb.AddCmdHistory{Doublon: true, Response: &commonpb.Response{}}, nil
	}

	// Find file data, cut it and process it. If the name is empty,
	// we are the server and we write to a dedicated file.
	var filename string
	if name == "" {
		filename = filepath.Join(assets.GetRootAppDir(), ".history")
	} else {
		path := assets.GetUserHistoryDir(name)
		filename = filepath.Join(path, ".history")
	}

	// Write to client console file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.New("server could not find your client when requesting history: " + err.Error())
	}
	if _, err = f.WriteString(req.Line + "\n"); err != nil {
		return nil, errors.New("server could not find your client when requesting history: " + err.Error())
	}
	f.Close()

	return &pb.AddCmdHistory{Response: &commonpb.Response{}}, nil
}

// A list of commands that are useless to save if they are strictly as short as in the list
var uselessCmds = []string{
	"exit",
	"cd",
	"ls",
	"cat",
	"jobs",
	"pwd",
	"use",
	"clear",
	"back",
	"pop",
	"push",
	"stack",
	"config",
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
