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
	"fmt"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetRportFwdListeners - Get a list of all reverse port forwards listeners from an implant
func (rpc *Server) GetRportFwdListeners(ctx context.Context, req *sliverpb.RportFwdListenersReq) (*sliverpb.RportFwdListeners, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}

	resp := &sliverpb.RportFwdListeners{Response: &commonpb.Response{}}
	// Listener inventory is server-owned authority and must remain available
	// even when an implant is malicious, disconnected, or no longer polling.
	// Never gate this RPC on an implant response.
	listeners := make([]*sliverpb.RportFwdListener, 0)
	for _, authorization := range rpc.rportFwdRegistry().List(req.Request.SessionID) {
		if authorization.State != rtunnels.AuthorizationActive || !authorization.HasListenerID {
			continue
		}
		listeners = append(listeners, listenerFromAuthorization(authorization, resp.Response))
	}
	resp.Listeners = listeners
	return resp, nil
}

// StartRportfwdListener - Instruct the implant to start a reverse port forward
func (rpc *Server) StartRportFwdListener(ctx context.Context, req *sliverpb.RportFwdStartListenerReq) (*sliverpb.RportFwdListener, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}

	sessionID := req.Request.SessionID
	registry := rpc.rportFwdRegistry()
	authorizationID, err := registry.BeginSpec(
		sessionID,
		req.BindAddress,
		req.ForwardAddress,
		req.KeepAlive,
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	rollback := true
	defer func() {
		if rollback {
			registry.Revoke(sessionID, authorizationID)
		}
	}()

	// Send the server-generated capability and the canonical operator target to
	// the implant. Neither field from the implant response is trusted later.
	authorization, ok := registry.Lookup(sessionID, authorizationID)
	if !ok {
		return nil, status.Error(codes.Internal, "reverse port forward authorization disappeared")
	}
	req.AuthorizationID = authorizationID.String()
	req.ForwardAddress = authorization.Address

	resp := &sliverpb.RportFwdListener{Response: &commonpb.Response{}}
	err = rpc.invokeGenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Response.GetErr() != "" {
		return listenerFromAuthorization(authorization, resp.Response), nil
	}
	if resp.ID == 0 {
		return nil, status.Error(codes.FailedPrecondition, "implant returned an invalid reverse port forward listener ID")
	}
	requiresAuthorizationID := resp.AuthorizationID == authorizationID.String()
	if err := registry.ActivateProtocol(sessionID, authorizationID, resp.ID, requiresAuthorizationID); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("failed to activate reverse port forward authorization: %v", err))
	}

	rollback = false
	authorization, _ = registry.Lookup(sessionID, authorizationID)
	return listenerFromAuthorization(authorization, resp.Response), nil
}

// StopRportfwdListener - Instruct the implant to stop a reverse port forward
func (rpc *Server) StopRportFwdListener(ctx context.Context, req *sliverpb.RportFwdStopListenerReq) (*sliverpb.RportFwdListener, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}

	// Revoke before asking the implant to stop. A malicious, disconnected, or
	// buggy implant cannot retain teamserver dial authority by failing its reply.
	sessionID := req.Request.SessionID
	registry := rpc.rportFwdRegistry()
	authorization, found := registry.LookupListener(sessionID, req.ID)
	registry.RevokeListener(sessionID, req.ID)
	if found {
		rtunnels.CloseAuthorization(sessionID, authorization.AuthorizationID)
	}

	resp := &sliverpb.RportFwdListener{Response: &commonpb.Response{}}
	err := rpc.invokeGenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	if found {
		authorization.State = rtunnels.AuthorizationRevoked
		return listenerFromAuthorization(authorization, resp.Response), nil
	}
	// There was no server authorization, but still allow the operator to clean
	// up an implant-side listener left by older state. Do not trust its metadata.
	resp.ID = req.ID
	resp.BindAddress = ""
	resp.ForwardAddress = ""
	resp.AuthorizationID = ""
	return resp, nil
}

func listenerFromAuthorization(authorization rtunnels.Authorization, response *commonpb.Response) *sliverpb.RportFwdListener {
	authorizationID := ""
	if authorization.RequiresAuthorizationID {
		authorizationID = authorization.AuthorizationID.String()
	}
	return &sliverpb.RportFwdListener{
		ID:              authorization.ImplantListenerID,
		BindAddress:     authorization.BindAddress,
		ForwardAddress:  authorization.Address,
		AuthorizationID: authorizationID,
		Response:        response,
	}
}
