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
	"errors"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"github.com/golang/protobuf/proto"
)

var (
	rpcLog = log.NamedLogger("rpc", "server")

	// ErrInvalidSessionID - Invalid Session ID in request
	ErrInvalidSessionID = errors.New("Invalid session ID")
)

// Server - gRPC server
type Server struct{}

// GenericRequest - Generic request interface to use with generic handlers
type GenericRequest interface {
	Reset()
	String() string
	ProtoMessage()

	GetRequest() *commonpb.Request
}

// GenericResponse - Generic response interface to use with generic handlers
type GenericResponse interface {
	GetResponse() *commonpb.Response
}

// NewServer - Create new server instance
func NewServer() *Server {
	return &Server{}
}

// GenericHandler - Pass the request to the Sliver/Session
func (rpc *Server) GenericHandler(req GenericRequest, resp proto.Message) error {
	session := core.Sessions.Get(req.GetRequest().SessionID)
	if session == nil {
		return ErrInvalidSessionID
	}

	reqData, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	timeout := time.Duration(req.GetRequest().Timeout)
	data, err := session.Request(sliverpb.MsgNumber(req), timeout, reqData)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return err
	}

	if resp.(GenericResponse).GetResponse().Err != "" {
		return errors.New(resp.(GenericResponse).GetResponse().Err)
	}

	return nil
}
