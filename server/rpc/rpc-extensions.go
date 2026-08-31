package rpc

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type capabilityLookup func(string) (uint64, bool)

// RegisterExtension registers a new extension in the implant
func (rpc *Server) RegisterExtension(ctx context.Context, req *sliverpb.RegisterExtensionReq) (*sliverpb.RegisterExtension, error) {
	resp := &sliverpb.RegisterExtension{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListExtensions lists the registered extensions
func (rpc *Server) ListExtensions(ctx context.Context, req *sliverpb.ListExtensionsReq) (*sliverpb.ListExtensions, error) {
	resp := &sliverpb.ListExtensions{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CallExtension calls a specific export of the loaded extension
func (rpc *Server) CallExtension(ctx context.Context, req *sliverpb.CallExtensionReq) (*sliverpb.CallExtension, error) {
	if err := requireCallExtensionBOFCapability(req); err != nil {
		return nil, err
	}
	resp := &sliverpb.CallExtension{Response: &commonpb.Response{}}
	err := rpc.GenericHandler(req, resp)
	if err != nil {
		// A gRPC error discards the response message, including any BOF output
		// records emitted before execution failed. Typed BOF callers explicitly
		// negotiate response-level errors so they can consume partial output.
		if req.GetIsBOF() && req.GetWantBOFOutputs() && resp.GetResponse().GetErr() != "" {
			return resp, nil
		}
		return nil, err
	}
	return resp, nil
}

func requireCallExtensionBOFCapability(req *sliverpb.CallExtensionReq) error {
	return requireCallExtensionBOFCapabilityWithLookups(req, sessionCapabilities, beaconCapabilities)
}

func requireCallExtensionBOFCapabilityWithLookups(req *sliverpb.CallExtensionReq, sessionLookup, beaconLookup capabilityLookup) error {
	if !req.GetIsBOF() {
		return nil
	}

	request := req.GetRequest()
	if request == nil {
		return ErrMissingRequestField
	}

	targetKind := "session"
	targetID := request.SessionID
	lookup := sessionLookup
	invalidTargetErr := ErrInvalidSessionID
	if request.Async {
		targetKind = "beacon"
		targetID = request.BeaconID
		lookup = beaconLookup
		invalidTargetErr = ErrInvalidBeaconID
	}

	capabilities, ok := lookup(targetID)
	if !ok {
		return invalidTargetErr
	}
	if capabilities&sliverpb.CapabilityBOFV1 == 0 {
		return status.Errorf(codes.FailedPrecondition, "target %s does not support built-in BOF execution", targetKind)
	}
	return nil
}

func sessionCapabilities(id string) (uint64, bool) {
	session := core.Sessions.Get(id)
	if session == nil {
		return 0, false
	}
	return session.Capabilities, true
}

func beaconCapabilities(id string) (uint64, bool) {
	beacon, err := db.BeaconByID(id)
	if err != nil || beacon == nil {
		return 0, false
	}
	return beacon.Capabilities, true
}
