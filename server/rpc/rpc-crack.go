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
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
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
		"Crack request: attack=%s hashType=%s hashes=%d status=%t benchmark=%t",
		req.AttackMode.String(),
		req.HashType.String(),
		len(req.Hashes),
		req.Status,
		req.Benchmark || req.BenchmarkAll,
	)

	jobID, err := uuid.NewV4()
	if err != nil {
		return &clientpb.CrackResponse{}, nil
	}

	job := &clientpb.CrackJob{
		ID:        jobID.String(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    clientpb.CrackJobStatus_IN_PROGRESS,
		Command:   req,
	}

	return &clientpb.CrackResponse{Job: job}, nil
}
