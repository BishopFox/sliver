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
	"errors"
	"runtime"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	rpcLog = log.NamedLogger("rpc", "server")
)

const (
	minTimeout = time.Duration(30 * time.Second)
)

// Server - gRPC server
type Server struct {
	// Magical methods to break backwards compatibility
	// Here be dragons: https://github.com/grpc/grpc-go/issues/3794
	rpcpb.UnimplementedSliverRPCServer
}

// GenericRequest - Generic request interface to use with generic handlers
type GenericRequest interface {
	Reset()
	String() string
	ProtoMessage()
	ProtoReflect() protoreflect.Message

	GetRequest() *commonpb.Request
}

// GenericResponse - Generic response interface to use with generic handlers
type GenericResponse interface {
	Reset()
	String() string
	ProtoMessage()
	ProtoReflect() protoreflect.Message

	GetResponse() *commonpb.Response
}

// NewServer - Create new server instance
func NewServer() *Server {
	core.StartEventAutomation()
	return &Server{}
}

// GetVersion - Get the server version
func (rpc *Server) GetVersion(ctx context.Context, _ *commonpb.Empty) (*clientpb.Version, error) {
	dirty := version.GitDirty != ""
	semVer := version.SemanticVersion()
	compiled, _ := version.Compiled()
	return &clientpb.Version{
		Major:      int32(semVer[0]),
		Minor:      int32(semVer[1]),
		Patch:      int32(semVer[2]),
		Commit:     version.GitCommit,
		Dirty:      dirty,
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

// GenericHandler - Pass the request to the Sliver/Session
func (rpc *Server) GenericHandler(req GenericRequest, resp GenericResponse) error {
	var err error
	request := req.GetRequest()
	if request == nil {
		return ErrMissingRequestField
	}
	if request.Async {
		err = rpc.asyncGenericHandler(req, resp)
		return err
	}

	// Sync request
	session := core.Sessions.Get(request.SessionID)
	if session == nil {
		return ErrInvalidSessionID
	}

	reqData, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	data, err := session.Request(sliverpb.MsgNumber(req), rpc.getTimeout(req), reqData)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return err
	}
	return rpc.getError(resp)
}

// asyncGenericHandler - Generic handler for async request/response's for beacon tasks
func (rpc *Server) asyncGenericHandler(req GenericRequest, resp GenericResponse) error {
	// VERY VERBOSE
	// rpcLog.Debugf("Async Generic Handler: %#v", req)
	request := req.GetRequest()
	if request == nil {
		return ErrMissingRequestField
	}

	beacon, err := db.BeaconByID(request.BeaconID)
	if beacon == nil || err != nil {
		rpcLog.Errorf("Invalid beacon ID in request: %s", err)
		return ErrInvalidBeaconID
	}

	// Overwrite unused implant fields before re-serializing
	request.SessionID = ""
	request.BeaconID = ""
	reqData, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	taskResponse := resp.GetResponse()
	taskResponse.Async = true
	taskResponse.BeaconID = beacon.ID.String()
	task, err := beacon.Task(&sliverpb.Envelope{
		Type: sliverpb.MsgNumber(req),
		Data: reqData,
	})
	if err != nil {
		rpcLog.Errorf("Database error: %s", err)
		return ErrDatabaseFailure
	}
	parts := strings.Split(string(req.ProtoReflect().Descriptor().FullName().Name()), ".")
	name := parts[len(parts)-1]
	task.Description = name
	err = db.Session().Save(task).Error
	if err != nil {
		rpcLog.Errorf("Database error: %s", err)
		return ErrDatabaseFailure
	}
	taskResponse.TaskID = task.ID.String()
	rpcLog.Debugf("Successfully tasked beacon: %#v", taskResponse)
	return nil
}

func (rpc *Server) getClientCommonName(ctx context.Context) string {
	client, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	tlsAuth, ok := client.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ""
	}
	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		return ""
	}
	if tlsAuth.State.VerifiedChains[0][0].Subject.CommonName != "" {
		return tlsAuth.State.VerifiedChains[0][0].Subject.CommonName
	}
	return ""
}

// getTimeout - Get the specified timeout from the request or the default
func (rpc *Server) getTimeout(req GenericRequest) time.Duration {
	timeout := req.GetRequest().Timeout
	if time.Duration(timeout) < time.Second {
		return minTimeout
	}
	return time.Duration(timeout)
}

// getError - Check an implant's response for Err and convert it to an `error` type
func (rpc *Server) getError(resp GenericResponse) error {
	respHeader := resp.GetResponse()
	if respHeader != nil && respHeader.Err != "" {
		return errors.New(respHeader.Err)
	}
	return nil
}
