package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// RegistryRead - gRPC interface to read a registry key from a session
func (rpc *Server) RegistryRead(ctx context.Context, req *sliverpb.RegistryReadReq) (*sliverpb.RegistryRead, error) {
	resp := &sliverpb.RegistryRead{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RegistryWrite - gRPC interface to write to a registry key on a session
func (rpc *Server) RegistryWrite(ctx context.Context, req *sliverpb.RegistryWriteReq) (*sliverpb.RegistryWrite, error) {
	resp := &sliverpb.RegistryWrite{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RegistryCreateKey - gRPC interface to create a registry key on a session
func (rpc *Server) RegistryCreateKey(ctx context.Context, req *sliverpb.RegistryCreateKeyReq) (*sliverpb.RegistryCreateKey, error) {
	resp := &sliverpb.RegistryCreateKey{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RegistryListSubKeys - gRPC interface to list the sub keys of a registry key
func (rpc *Server) RegistryListSubKeys(ctx context.Context, req *sliverpb.RegistrySubKeyListReq) (*sliverpb.RegistrySubKeyList, error) {
	resp := &sliverpb.RegistrySubKeyList{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RegistryListSubKeys - gRPC interface to list the sub keys of a registry key
func (rpc *Server) RegistryListValues(ctx context.Context, req *sliverpb.RegistryListValuesReq) (*sliverpb.RegistryValuesList, error) {
	resp := &sliverpb.RegistryValuesList{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
