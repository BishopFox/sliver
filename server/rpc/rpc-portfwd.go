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
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func validatePortfwdTarget(req *sliverpb.PortfwdReq) error {
	if req == nil {
		return errors.New("port forward request is required")
	}
	if req.Host == "" || strings.IndexFunc(req.Host, func(value rune) bool {
		return unicode.IsSpace(value) || unicode.IsControl(value)
	}) >= 0 {
		return fmt.Errorf("invalid port forward host %q", req.Host)
	}
	if req.Port == 0 || req.Port > 65535 {
		return fmt.Errorf("invalid port forward port %d: must be from 1 to 65535", req.Port)
	}
	if req.Protocol != sliverpb.PortFwdProtoTCP {
		return fmt.Errorf("unsupported port forward protocol %d", req.Protocol)
	}
	return nil
}

// Portfwd - Open an in-band port forward
func (s *Server) Portfwd(ctx context.Context, req *sliverpb.PortfwdReq) (*sliverpb.Portfwd, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}
	if err := validatePortfwdTarget(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	session := core.Sessions.Get(req.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	tunnel := core.Tunnels.Get(req.TunnelID)
	if tunnel == nil {
		return nil, rpcError(core.ErrInvalidTunnelID)
	}
	if tunnel.SessionID != session.ID {
		return nil, rpcError(core.ErrInvalidTunnelID)
	}
	// Session IDs may be reused after a reconnect. The tunnel remains owned by
	// the exact implant connection that created it, so never send setup through
	// a replacement session while data and teardown still target the creator.
	if session.Connection != tunnel.ImplantConnection() {
		return nil, rpcError(core.ErrInvalidTunnelID)
	}
	// The client must bind its TunnelData stream before the implant opens the
	// destination socket. Otherwise an abandoned client setup can leave a live
	// port-forward connection without a consumer.
	select {
	case <-tunnel.ClientBound():
	case <-tunnel.Done():
		return nil, rpcError(core.ErrInvalidTunnelID)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case <-tunnel.Done():
		return nil, rpcError(core.ErrInvalidTunnelID)
	default:
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return nil, rpcError(err)
	}
	requestCtx, cancelRequest := context.WithTimeout(ctx, s.getTimeout(req))
	go func() {
		select {
		case <-tunnel.Done():
			cancelRequest()
		case <-requestCtx.Done():
		}
	}()
	data, err := session.RequestContext(requestCtx, sliverpb.MsgNumber(req), reqData)
	requestContextErr := requestCtx.Err()
	cancelRequest()
	if err != nil {
		// Preserve the existing implant-timeout error while allowing an earlier
		// caller or tunnel cancellation to propagate immediately.
		if errors.Is(requestContextErr, context.DeadlineExceeded) && ctx.Err() == nil {
			err = core.ErrImplantTimeout
		}
		return nil, rpcError(err)
	}
	portfwd := &sliverpb.Portfwd{}
	err = proto.Unmarshal(data, portfwd)
	if err != nil {
		return nil, rpcError(err)
	}
	return portfwd, nil
}
