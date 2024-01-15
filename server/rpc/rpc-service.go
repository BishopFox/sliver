package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Services - List and control services
func (rpc *Server) Services(ctx context.Context, req *sliverpb.ServicesReq) (*sliverpb.Services, error) {
	resp := &sliverpb.Services{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) ServiceDetail(ctx context.Context, req *sliverpb.ServiceDetailReq) (*sliverpb.ServiceDetail, error) {
	resp := &sliverpb.ServiceDetail{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// StartService creates and starts a Windows service on a remote host
func (rpc *Server) StartService(ctx context.Context, req *sliverpb.StartServiceReq) (*sliverpb.ServiceInfo, error) {
	resp := &sliverpb.ServiceInfo{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) StartServiceByName(ctx context.Context, req *sliverpb.StartServiceByNameReq) (*sliverpb.ServiceInfo, error) {
	resp := &sliverpb.ServiceInfo{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// StopService stops a remote service
func (rpc *Server) StopService(ctx context.Context, req *sliverpb.StopServiceReq) (*sliverpb.ServiceInfo, error) {
	resp := &sliverpb.ServiceInfo{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RemoveService deletes a service from the remote system
func (rpc *Server) RemoveService(ctx context.Context, req *sliverpb.RemoveServiceReq) (*sliverpb.ServiceInfo, error) {
	resp := &sliverpb.ServiceInfo{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
