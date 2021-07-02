package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// GetEnv - Retrieve the environment variables list from the current session
func (rpc *Server) GetEnv(ctx context.Context, req *sliverpb.EnvReq) (*sliverpb.EnvInfo, error) {
	resp := &sliverpb.EnvInfo{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SetEnv - Set an environment variable
func (rpc *Server) SetEnv(ctx context.Context, req *sliverpb.SetEnvReq) (*sliverpb.SetEnv, error) {
	resp := &sliverpb.SetEnv{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UnsetEnv - Set an environment variable
func (rpc *Server) UnsetEnv(ctx context.Context, req *sliverpb.UnsetEnvReq) (*sliverpb.UnsetEnv, error) {
	resp := &sliverpb.UnsetEnv{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
