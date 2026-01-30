package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

var (
	crackCommandRpcLog = log.NamedLogger("rpc", "crack")
)

func (rpc *Server) Crack(ctx context.Context, req *clientpb.CrackCommand) (*clientpb.CrackResponse, error) {
	if req == nil {
		crackCommandRpcLog.Warn("Received empty crack command")
		return &clientpb.CrackResponse{}, nil
	}

	crackCommandRpcLog.Infof(
		"Crack request: attack=%s hashType=%s hashes=%d status=%t",
		req.AttackMode.String(),
		req.HashType.String(),
		len(req.Hashes),
		req.Status,
	)

	jobID, err := uuid.NewV4()
	if err != nil {
		crackCommandRpcLog.Errorf("Failed to generate crack job ID: %s", err)
		return nil, status.Error(codes.Internal, "failed to generate crack job id")
	}

	job := &models.CrackJob{
		ID: jobID,
	}
	command := models.CrackCommand{}.FromProtobuf(req)
	command.CrackJobID = jobID

	if err := db.Session().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(job).Error; err != nil {
			return err
		}
		return tx.Create(command).Error
	}); err != nil {
		crackCommandRpcLog.Errorf("Failed to save crack job: %s", err)
		return nil, status.Error(codes.Internal, "failed to save crack job")
	}

	job.Command = *command
	return &clientpb.CrackResponse{Job: job.ToProtobuf()}, nil
}
