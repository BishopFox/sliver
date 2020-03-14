package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Execute - Execute a remote process
func (rpc *Server) Execute(ctx context.Context, req *sliverpb.ExecuteReq) (*sliverpb.Execute, error) {
	resp := &sliverpb.Execute{}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
