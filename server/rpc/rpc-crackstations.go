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
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	crackRpcLog = log.NamedLogger("rpc", "crackstations")
)

func (rpc *Server) Crackstations(ctx context.Context, req *commonpb.Empty) (*clientpb.Crackstations, error) {
	return &clientpb.Crackstations{Crackstations: core.AllCrackstations()}, nil
}

func (rpc *Server) CrackstationTrigger(ctx context.Context, req *clientpb.Event) (*commonpb.Empty, error) {

	switch req.EventType {

	case "crack":
		core.EventBroker.Publish(core.Event{
			EventType: req.EventType,
			Data:      req.Data,
		})

	}

	return &commonpb.Empty{}, nil
}

func (rpc *Server) CrackstationRegister(req *clientpb.Crackstation, stream rpcpb.SliverRPC_CrackstationRegisterServer) error {
	hostUUID := uuid.FromStringOrNil(req.HostUUID)
	if hostUUID == uuid.Nil {
		return status.Error(codes.InvalidArgument, "invalid host uuid")
	}
	crackStation := core.NewCrackstation(req)
	err := core.AddCrackstation(crackStation)
	if err == core.ErrDuplicateExternalCrackerName {
		status.Error(codes.AlreadyExists, "crackstation name already exists")
	}
	if err != nil {
		return err
	}

	_, err = db.CrackstationByName(req.Name)
	if err == db.ErrRecordNotFound {
		dbSession := db.Session()
		err = dbSession.Create(&models.Crackstation{ID: req.Name}).Error
		if err != nil {
			crackRpcLog.Errorf("Failed to create crackstation record: %s", err)
			return status.Error(codes.Internal, "failed to register crackstation")
		}
	}

	crackRpcLog.Infof("Crackstation %s (%s) connected", req.Name, req.OperatorName)
	events := core.EventBroker.Subscribe()
	defer func() {
		crackRpcLog.Infof("Crackstation %s disconnected", req.Name)
		core.EventBroker.Unsubscribe(events)
		core.RemoveCrackstation(req.Name)
	}()

	// Only forward these event types
	crackingEvents := []string{}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case msg := <-crackStation.Events: // This event stream is specific to this crackstation
			err := stream.Send(msg)
			if err != nil {
				crackRpcLog.Warnf(err.Error())
				return err
			}
		case event := <-events: // All server-side events
			if !util.Contains(crackingEvents, event.EventType) {
				continue
			}

			pbEvent := &clientpb.Event{
				EventType: event.EventType,
				Data:      event.Data,
			}
			if event.Job != nil {
				pbEvent.Job = event.Job.ToProtobuf()
			}
			if event.Client != nil {
				pbEvent.Client = event.Client.ToProtobuf()
			}
			if event.Session != nil {
				pbEvent.Session = event.Session.ToProtobuf()
			}
			if event.Err != nil {
				pbEvent.Err = event.Err.Error()
			}

			err := stream.Send(pbEvent)
			if err != nil {
				crackRpcLog.Warnf(err.Error())
				return err
			}
		}
	}
}
