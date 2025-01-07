package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"log"
	"os"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db"
)

// GetC2Profiles - Retrieve C2 Profile names and id's
func (rpc *Server) GetHTTPC2Profiles(ctx context.Context, req *commonpb.Empty) (*clientpb.HTTPC2Configs, error) {
	c2Configs := clientpb.HTTPC2Configs{}
	httpC2Config, err := db.LoadHTTPC2s()
	if err != nil {
		return nil, err
	}

	c2Configs.Configs = httpC2Config

	return &c2Configs, nil
}

// GetC2ProfileByName - Retrieve C2 Profile by name
func (rpc *Server) GetHTTPC2ProfileByName(ctx context.Context, req *clientpb.C2ProfileReq) (*clientpb.HTTPC2Config, error) {
	httpC2Config, err := db.LoadHTTPC2ConfigByName(req.Name)
	if err != nil {
		return nil, err
	}

	return httpC2Config, nil
}

// Save HTTP C2 Profile
func (rpc *Server) SaveHTTPC2Profile(ctx context.Context, req *clientpb.HTTPC2ConfigReq) (*commonpb.Empty, error) {
	err := configs.CheckHTTPC2ConfigErrors(req.C2Config)
	if err != nil {
		return nil, err
	}

	if req.Overwrite && req.C2Config.Name == "" {
		return nil, configs.ErrMissingC2ProfileName
	}
	err = db.SearchStageExtensions(req.C2Config.ImplantConfig.StagerFileExtension, req.C2Config.Name)
	if err != nil {
		return nil, err
	}

	err = db.SearchStartSessionExtensions(req.C2Config.ImplantConfig.StartSessionFileExtension, req.C2Config.Name)
	if err != nil {
		return nil, err
	}

	httpC2Config, err := db.LoadHTTPC2ConfigByName(req.C2Config.Name)
	if err != nil {
		return nil, err
	}
	if httpC2Config.Name != "" && req.Overwrite == false {
		return nil, configs.ErrDuplicateC2ProfileName
	}

	if req.Overwrite {
		if httpC2Config.Name == "" {
			return nil, configs.ErrC2ProfileNotFound
		}
		err = db.HTTPC2ConfigUpdate(req.C2Config, httpC2Config)
		if err != nil {
			log.Printf("Error:\n%s", err)
			os.Exit(-1)
		}
	} else {
		err = db.SaveHTTPC2Config(req.C2Config)
		if err != nil {
			log.Printf("Error:\n%s", err)
			os.Exit(-1)
		}
	}
	return &commonpb.Empty{}, nil
}
