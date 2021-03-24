package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func (rpc *Server) RegistryRead(ctx context.Context, req *sliverpb.RegistryReadReq) (*sliverpb.RegistryRead, error) {
	resp := &sliverpb.RegistryRead{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) RegistryWrite(ctx context.Context, req *sliverpb.RegistryWriteReq) (*sliverpb.RegistryWrite, error) {
	resp := &sliverpb.RegistryWrite{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) RegistryCreateKey(ctx context.Context, req *sliverpb.RegistryCreateKeyReq) (*sliverpb.RegistryCreateKey, error) {
	resp := &sliverpb.RegistryCreateKey{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
