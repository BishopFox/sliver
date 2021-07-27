package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// RegisterExtension registers a new extension in the implant
func (rpc *Server) RegisterExtension(ctx context.Context, req *sliverpb.RegisterExtensionReq) (*sliverpb.RegisterExtension, error) {
	resp := &sliverpb.RegisterExtension{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListExtensions lists the registered extensions
func (rpc *Server) ListExtensions(ctx context.Context, req *sliverpb.ListExtensionsReq) (*sliverpb.ListExtensions, error) {
	resp := &sliverpb.ListExtensions{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CallExtension calls a specific export of the loaded extension
func (rpc *Server) CallExtension(ctx context.Context, req *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
	resp := &sliverpb.CallExtension{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
