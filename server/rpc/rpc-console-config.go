package rpc

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
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/assets"
)

// LoadConsoleConfig - The client requires its console configuration (per-user)
func (rpc *Server) LoadConsoleConfig(ctx context.Context, req *clientpb.GetConsoleConfigReq) (*clientpb.GetConsoleConfig, error) {

	// Get an ID/operator name for this client
	name := rpc.getClientCommonName(ctx)

	// Find file data, cut it and process it. If the name is empty,
	// we are the server and we write to a dedicated file.
	var filename string
	if name == "" {
		filename = filepath.Join(assets.GetRootAppDir(), "console.config")
	} else {
		path := assets.GetUserDirectory(name)
		filename = filepath.Join(path, "console.config")
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return &clientpb.GetConsoleConfig{Response: &commonpb.Response{Err: "could not find user console configuration"}}, nil
	}

	conf := &clientpb.ConsoleConfig{}
	err = json.Unmarshal(data, conf)
	if err != nil {
		return &clientpb.GetConsoleConfig{Response: &commonpb.Response{Err: "failed to unmarshal user console configuration"}}, nil
	}

	return &clientpb.GetConsoleConfig{Config: conf, Response: &commonpb.Response{}}, nil
}

// SaveUserConsoleConfig - The client user wants to save its current console configuration.
func (rpc *Server) SaveUserConsoleConfig(ctx context.Context, req *clientpb.SaveConsoleConfigReq) (*clientpb.SaveConsoleConfig, error) {

	// Get an ID/operator name for this client
	name := rpc.getClientCommonName(ctx)

	// Find file data, cut it and process it. If the name is empty,
	// we are the server and we write to a dedicated file.
	var filename string
	if name == "" {
		filename = filepath.Join(assets.GetRootAppDir(), "console.config")
	} else {
		path := assets.GetUserDirectory(name)
		filename = filepath.Join(path, "console.config")
	}

	// Marshal config
	data, err := json.Marshal(req.Config)
	if err != nil {
		return &clientpb.SaveConsoleConfig{Response: &commonpb.Response{Err: "failed to marshal user console configuration"}}, nil
	}

	// Write to client history file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return &clientpb.SaveConsoleConfig{Response: &commonpb.Response{Err: "Could not find and/or overwrite confosle config file"}}, nil
	}
	if _, err = f.Write(data); err != nil {
		return &clientpb.SaveConsoleConfig{Response: &commonpb.Response{Err: "Could not write/overwrite confosle config file"}}, nil
	}
	f.Close()

	return &clientpb.SaveConsoleConfig{Response: &commonpb.Response{}}, nil
}
