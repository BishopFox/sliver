package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
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
