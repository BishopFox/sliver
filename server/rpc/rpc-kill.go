package rpc

import (
	"context"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/protobuf/proto"
)

func (rpc *Server) Kill(ctx context.Context, kill *sliverpb.KillReq) (*commonpb.Empty, error) {
	var (
		beacon *models.Beacon
		err    error
	)
	session := core.Sessions.Get(kill.Request.SessionID)
	if session == nil {
		beacon, err = db.BeaconByID(kill.Request.BeaconID)
		if err != nil {
			return &commonpb.Empty{}, ErrInvalidBeaconID
		} else {
			return rpc.killBeacon(kill, beacon)
		}
	}
	return rpc.killSession(kill, session)
}

func (rpc *Server) killSession(kill *sliverpb.KillReq, session *core.Session) (*commonpb.Empty, error) {
	core.Sessions.Remove(session.ID)
	data, err := proto.Marshal(kill)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(kill.Request.GetTimeout())
	session.Request(sliverpb.MsgNumber(kill), timeout, data)
	return &commonpb.Empty{}, nil
}

func (rpc *Server) killBeacon(kill *sliverpb.KillReq, beacon *models.Beacon) (*commonpb.Empty, error) {
	resp := &commonpb.Empty{}
	request := kill.GetRequest()
	request.SessionID = 0
	request.Async = true
	request.BeaconID = beacon.ID.String()
	reqData, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	task, err := beacon.Task(&sliverpb.Envelope{
		Type: sliverpb.MsgKillSessionReq,
		Data: reqData,
	})
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(kill.ProtoReflect().Descriptor().FullName().Name()), ".")
	name := parts[len(parts)-1]
	task.Description = name
	// Update db
	err = db.Session().Save(task).Error
	return resp, err
}
