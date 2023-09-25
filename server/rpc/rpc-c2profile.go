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

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"golang.org/x/exp/slices"
)

// GetC2Profiles - Retrieve C2 Profile names and id's
func (rpc *Server) GetHTTPC2Profiles(ctx context.Context, req *commonpb.Empty) (*clientpb.HTTPC2Configs, error) {
	c2Configs := clientpb.HTTPC2Configs{}
	httpC2Config, err := db.LoadHTTPC2s()
	if err != nil {
		return nil, err
	}

	for _, c2Config := range *httpC2Config {
		c2Configs.Configs = append(c2Configs.Configs, c2Config.ToProtobuf())
	}

	return &c2Configs, nil
}

// GetC2ProfileByName - Retrieve C2 Profile by name
func (rpc *Server) GetHTTPC2ProfileByName(ctx context.Context, req *clientpb.C2ProfileReq) (*clientpb.HTTPC2Config, error) {
	httpC2Config, err := db.LoadHTTPC2ConfigByName(req.Name)
	if err != nil {
		return nil, err
	}

	return httpC2Config.ToProtobuf(), nil
}

// Save HTTP C2 Profile
func (rpc *Server) SaveHTTPC2Profile(ctx context.Context, req *clientpb.HTTPC2Config) (*commonpb.Empty, error) {
	protocols := []string{constants.HttpStr, constants.HttpsStr}
	err := configs.CheckHTTPC2ConfigErrors(req)
	if err != nil {
		return nil, err
	}

	err = db.SearchStageExtensions(req.ImplantConfig.StagerFileExtension)
	if err != nil {
		return nil, err
	}

	httpC2Config, err := db.LoadHTTPC2ConfigByName(req.Name)
	if err != nil {
		return nil, err
	}
	if httpC2Config.Name != "" {
		return nil, configs.ErrDuplicateC2ProfileName
	}

	httpC2ConfigModel := models.HTTPC2ConfigFromProtobuf(req)
	err = db.HTTPC2ConfigSave(httpC2ConfigModel)
	if err != nil {
		log.Printf("Error:\n%s", err)
		os.Exit(-1)
	}
	// reload jobs to include new profile
	for _, job := range core.Jobs.All() {
		if job != nil && slices.Contains(protocols, job.Name) {
			job.JobCtrl <- true
		}
	}
	listenerJobs, err := db.ListenerJobs()
	if err != nil {
		return nil, err
	}

	for _, j := range *listenerJobs {
		listenerJob, err := db.ListenerByJobID(j.JobID)
		if err != nil {
			return nil, err
		}
		if slices.Contains(protocols, j.Type) {
			job, err := c2.StartHTTPListenerJob(listenerJob.ToProtobuf().HTTPConf)
			if err != nil {
				return nil, err
			}
			j.JobID = uint32(job.ID)
			db.HTTPC2ListenerUpdate(&j)
		}
	}

	return &commonpb.Empty{}, nil
}
