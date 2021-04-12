package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/generate"
)

func (rpc *Server) GenerateWGClientConfig(ctx context.Context, _ *commonpb.Empty) (*clientpb.WGClientConfig, error) {
	clientIP, err := generate.GenerateUniqueIP()
	if err != nil {
		rpcLog.Errorf("Could not generate WG unique IP: %v", err)
		return nil, err
	}
	privkey, pubkey, err := certs.GenerateWGKeys(true, clientIP.String())
	if err != nil {
		rpcLog.Errorf("Could not generate WG keys: %v", err)
		return nil, err
	}
	_, serverPubKey, err := certs.GetWGServerKeys()
	if err != nil {
		rpcLog.Errorf("Could not get WG server keys: %v", err)
		return nil, err
	}
	resp := &clientpb.WGClientConfig{
		ClientPrivateKey: privkey,
		ClientIP:         clientIP.String(),
		ClientPubKey:     pubkey,
		ServerPubKey:     serverPubKey,
	}

	return resp, nil
}

func (rpc *Server) WGStartPortForward(ctx context.Context, req *sliverpb.WGPortForwardStartReq) (*sliverpb.WGPortForward, error) {
	resp := &sliverpb.WGPortForward{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) WGStopPortForward(ctx context.Context, req *sliverpb.WGPortForwardStopReq) (*sliverpb.WGPortForward, error) {
	resp := &sliverpb.WGPortForward{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) WGStartSocks(ctx context.Context, req *sliverpb.WGSocksStartReq) (*sliverpb.WGSocks, error) {
	resp := &sliverpb.WGSocks{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) WGStopSocks(ctx context.Context, req *sliverpb.WGSocksStopReq) (*sliverpb.WGSocks, error) {
	resp := &sliverpb.WGSocks{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) WGListSocksServers(ctx context.Context, req *sliverpb.WGSocksServersReq) (*sliverpb.WGSocksServers, error) {
	resp := &sliverpb.WGSocksServers{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) WGListForwarders(ctx context.Context, req *sliverpb.WGTCPForwardersReq) (*sliverpb.WGTCPForwarders, error) {
	resp := &sliverpb.WGTCPForwarders{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
