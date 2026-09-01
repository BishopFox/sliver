package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

const (
	bestEffortRportFwdInventoryTimeout = 5 * time.Second
	// This application-payload cap is applied before the inventory parser. The
	// transport has already materialized the enclosing envelope; transport-level
	// envelope limits are a separate concern.
	maxRportFwdInventoryResponseBytes = 4 * 1024 * 1024
	// This compatibility-only inventory is never an authorization source. The
	// cap is deliberately generous for honest long-lived implants while bounding
	// attacker-controlled response and CLI amplification.
	maxUntrustedRportFwdInventoryIDs       = 4096
	maxRportFwdInventoryTopLevelFields     = maxUntrustedRportFwdInventoryIDs + 16
	maxRportFwdInventoryNestedFields       = 16
	maxRportFwdInventoryNestedMessageBytes = 4 * 1024
	maxConcurrentRportFwdInventoryProbes   = 8
)

var rportFwdInventoryProbes = struct {
	sync.Mutex
	activeSessions map[string]struct{}
	global         chan struct{}
}{
	activeSessions: map[string]struct{}{},
	global:         make(chan struct{}, maxConcurrentRportFwdInventoryProbes),
}

// GetRportFwdListeners - Get a list of all reverse port forwards listeners from an implant
func (rpc *Server) GetRportFwdListeners(ctx context.Context, req *sliverpb.RportFwdListenersReq) (*sliverpb.RportFwdListeners, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}

	registry := rpc.rportFwdRegistry()
	// Remember IDs that are authoritative before the probe. If one is revoked
	// while the implant reply is in flight, that stale reply must not resurrect it
	// as a compatibility-only orphan.
	authoritativeBeforeProbe := map[uint32]struct{}{}
	for _, authorization := range registry.List(req.Request.SessionID) {
		if authorization.State == rtunnels.AuthorizationActive && authorization.HasListenerID {
			authoritativeBeforeProbe[authorization.ImplantListenerID] = struct{}{}
		}
	}
	// Collect untrusted IDs, then take a fresh authoritative snapshot. The final
	// snapshot supplies all metadata; the initial snapshot only suppresses IDs
	// revoked during the compatibility probe.
	implantListenerIDs := rpc.bestEffortImplantRportFwdListenerIDs(ctx, req.Request.SessionID)
	resp := &sliverpb.RportFwdListeners{Response: &commonpb.Response{}}
	listeners := make([]*sliverpb.RportFwdListener, 0)
	knownListenerIDs := map[uint32]struct{}{}
	hasStartingAuthorization := false
	// Listener metadata and dial authority are exclusively server-owned. Build
	// that authoritative view first so an implant can never override it.
	for _, authorization := range registry.List(req.Request.SessionID) {
		if authorization.State == rtunnels.AuthorizationStarting {
			hasStartingAuthorization = true
			continue
		}
		if authorization.State != rtunnels.AuthorizationActive || !authorization.HasListenerID {
			continue
		}
		knownListenerIDs[authorization.ImplantListenerID] = struct{}{}
		listeners = append(listeners, listenerFromAuthorization(authorization, resp.Response))
	}
	// Older implants retain their listener across a C2 reconnect even though it
	// remains bound to the dead Connection generation. Query their process-local
	// inventory only as a bounded, best-effort compatibility aid. Unknown IDs
	// are useful for the safe Stop fallback, but every implant-supplied metadata
	// field remains untrusted and is intentionally discarded.
	for _, listenerID := range implantListenerIDs {
		// During Start, the implant can bind and report an ID before the server
		// commits ActivateProtocol. Exposing any untrusted ID in that window would
		// let a concurrent Stop remove implant state before Start activates it.
		if hasStartingAuthorization {
			break
		}
		if _, known := knownListenerIDs[listenerID]; known {
			continue
		}
		if _, revokedDuringProbe := authoritativeBeforeProbe[listenerID]; revokedDuringProbe {
			continue
		}
		knownListenerIDs[listenerID] = struct{}{}
		listeners = append(listeners, &sliverpb.RportFwdListener{
			ID:       listenerID,
			Response: resp.Response,
		})
	}
	resp.Listeners = listeners
	return resp, nil
}

func (rpc *Server) bestEffortImplantRportFwdListenerIDs(ctx context.Context, sessionID string) []uint32 {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return nil
	}
	releaseProbe := tryAcquireRportFwdInventoryProbe(sessionID)
	if releaseProbe == nil {
		return nil
	}
	defer releaseProbe()

	queryCtx, cancel := context.WithTimeout(ctx, bestEffortRportFwdInventoryTimeout)
	defer cancel()
	request := &sliverpb.RportFwdListenersReq{Request: &commonpb.Request{
		SessionID: sessionID,
		Timeout:   int64(bestEffortRportFwdInventoryTimeout),
	}}
	rawResponse, err := rpc.queryImplantRportFwdInventory(queryCtx, request)
	if err != nil || len(rawResponse) > maxRportFwdInventoryResponseBytes {
		return nil
	}
	listenerIDs, responseError, err := parseRportFwdInventoryIDs(rawResponse)
	if err != nil || responseError {
		return nil
	}
	return listenerIDs
}

func tryAcquireRportFwdInventoryProbe(sessionID string) func() {
	select {
	case rportFwdInventoryProbes.global <- struct{}{}:
	default:
		return nil
	}

	rportFwdInventoryProbes.Lock()
	if _, active := rportFwdInventoryProbes.activeSessions[sessionID]; active {
		rportFwdInventoryProbes.Unlock()
		<-rportFwdInventoryProbes.global
		return nil
	}
	rportFwdInventoryProbes.activeSessions[sessionID] = struct{}{}
	rportFwdInventoryProbes.Unlock()

	return func() {
		rportFwdInventoryProbes.Lock()
		delete(rportFwdInventoryProbes.activeSessions, sessionID)
		rportFwdInventoryProbes.Unlock()
		<-rportFwdInventoryProbes.global
	}
}

func (rpc *Server) queryImplantRportFwdInventory(ctx context.Context, request *sliverpb.RportFwdListenersReq) ([]byte, error) {
	if rpc.rportFwdInventoryQuery != nil {
		return rpc.rportFwdInventoryQuery(ctx, request)
	}
	session := core.Sessions.Get(request.Request.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	requestData, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	return session.RequestContext(ctx, sliverpb.MsgNumber(request), requestData)
}

// parseRportFwdInventoryIDs consumes only the listener ID and Response.Err wire
// fields. In particular, it never allocates a RportFwdListener for repeated
// empty or metadata-heavy entries supplied by an untrusted implant.
func parseRportFwdInventoryIDs(raw []byte) ([]uint32, bool, error) {
	seen := make(map[uint32]struct{})
	listenerIDs := make([]uint32, 0)
	responseError := false
	topLevelFields := 0
	listenerEntries := 0
	for len(raw) > 0 {
		topLevelFields++
		if topLevelFields > maxRportFwdInventoryTopLevelFields {
			return nil, false, fmt.Errorf("rportfwd inventory has too many top-level fields")
		}
		fieldNumber, wireType, tagLength := protowire.ConsumeTag(raw)
		if tagLength < 0 {
			return nil, false, protowire.ParseError(tagLength)
		}
		raw = raw[tagLength:]
		switch fieldNumber {
		case 1:
			listenerEntries++
			if listenerEntries > maxUntrustedRportFwdInventoryIDs {
				return nil, false, fmt.Errorf("rportfwd inventory has too many listener entries")
			}
			if wireType != protowire.BytesType {
				return nil, false, fmt.Errorf("rportfwd inventory listeners field has wire type %d", wireType)
			}
			listenerData, valueLength := protowire.ConsumeBytes(raw)
			if valueLength < 0 {
				return nil, false, protowire.ParseError(valueLength)
			}
			raw = raw[valueLength:]
			if len(listenerData) > maxRportFwdInventoryNestedMessageBytes {
				return nil, false, fmt.Errorf("rportfwd inventory listener entry is too large")
			}
			listenerID, err := parseRportFwdInventoryListenerID(listenerData)
			if err != nil {
				return nil, false, err
			}
			if listenerID == 0 {
				continue
			}
			if _, duplicate := seen[listenerID]; duplicate {
				continue
			}
			seen[listenerID] = struct{}{}
			listenerIDs = append(listenerIDs, listenerID)
		case 9:
			if wireType != protowire.BytesType {
				return nil, false, fmt.Errorf("rportfwd inventory response field has wire type %d", wireType)
			}
			responseData, valueLength := protowire.ConsumeBytes(raw)
			if valueLength < 0 {
				return nil, false, protowire.ParseError(valueLength)
			}
			raw = raw[valueLength:]
			if len(responseData) > maxRportFwdInventoryNestedMessageBytes {
				return nil, false, fmt.Errorf("rportfwd inventory response is too large")
			}
			hasError, err := parseRportFwdInventoryResponseError(responseData)
			if err != nil {
				return nil, false, err
			}
			responseError = responseError || hasError
		default:
			valueLength, err := consumeRportFwdInventoryFieldValue(fieldNumber, wireType, raw)
			if err != nil {
				return nil, false, err
			}
			raw = raw[valueLength:]
		}
	}
	sort.Slice(listenerIDs, func(i, j int) bool { return listenerIDs[i] < listenerIDs[j] })
	return listenerIDs, responseError, nil
}

func parseRportFwdInventoryListenerID(raw []byte) (uint32, error) {
	var listenerID uint32
	fields := 0
	for len(raw) > 0 {
		fields++
		if fields > maxRportFwdInventoryNestedFields {
			return 0, fmt.Errorf("rportfwd listener has too many fields")
		}
		fieldNumber, wireType, tagLength := protowire.ConsumeTag(raw)
		if tagLength < 0 {
			return 0, protowire.ParseError(tagLength)
		}
		raw = raw[tagLength:]
		if fieldNumber == 1 {
			if wireType != protowire.VarintType {
				return 0, fmt.Errorf("rportfwd listener ID field has wire type %d", wireType)
			}
			value, valueLength := protowire.ConsumeVarint(raw)
			if valueLength < 0 {
				return 0, protowire.ParseError(valueLength)
			}
			if value > math.MaxUint32 {
				return 0, fmt.Errorf("rportfwd listener ID exceeds uint32")
			}
			listenerID = uint32(value)
			raw = raw[valueLength:]
			continue
		}
		valueLength, err := consumeRportFwdInventoryFieldValue(fieldNumber, wireType, raw)
		if err != nil {
			return 0, err
		}
		raw = raw[valueLength:]
	}
	return listenerID, nil
}

func parseRportFwdInventoryResponseError(raw []byte) (bool, error) {
	hasError := false
	fields := 0
	for len(raw) > 0 {
		fields++
		if fields > maxRportFwdInventoryNestedFields {
			return false, fmt.Errorf("rportfwd inventory response has too many fields")
		}
		fieldNumber, wireType, tagLength := protowire.ConsumeTag(raw)
		if tagLength < 0 {
			return false, protowire.ParseError(tagLength)
		}
		raw = raw[tagLength:]
		if fieldNumber == 1 {
			if wireType != protowire.BytesType {
				return false, fmt.Errorf("rportfwd inventory error field has wire type %d", wireType)
			}
			value, valueLength := protowire.ConsumeBytes(raw)
			if valueLength < 0 {
				return false, protowire.ParseError(valueLength)
			}
			hasError = hasError || len(value) != 0
			raw = raw[valueLength:]
			continue
		}
		valueLength, err := consumeRportFwdInventoryFieldValue(fieldNumber, wireType, raw)
		if err != nil {
			return false, err
		}
		raw = raw[valueLength:]
	}
	return hasError, nil
}

func consumeRportFwdInventoryFieldValue(fieldNumber protowire.Number, wireType protowire.Type, raw []byte) (int, error) {
	switch wireType {
	case protowire.VarintType, protowire.Fixed32Type, protowire.Fixed64Type, protowire.BytesType:
		// These scalar and length-delimited encodings are nonrecursive.
	default:
		return 0, fmt.Errorf("rportfwd inventory field has unsupported wire type %d", wireType)
	}
	valueLength := protowire.ConsumeFieldValue(fieldNumber, wireType, raw)
	if valueLength < 0 {
		return 0, protowire.ParseError(valueLength)
	}
	return valueLength, nil
}

// StartRportfwdListener - Instruct the implant to start a reverse port forward
func (rpc *Server) StartRportFwdListener(ctx context.Context, req *sliverpb.RportFwdStartListenerReq) (*sliverpb.RportFwdListener, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}

	sessionID := req.Request.SessionID
	registry := rpc.rportFwdRegistry()
	authorizationID, err := registry.BeginSpec(
		sessionID,
		req.BindAddress,
		req.ForwardAddress,
		req.KeepAlive,
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	rollback := true
	defer func() {
		if rollback {
			registry.Revoke(sessionID, authorizationID)
		}
	}()

	// Send the server-generated capability and the canonical operator target to
	// the implant. Neither field from the implant response is trusted later.
	authorization, ok := registry.Lookup(sessionID, authorizationID)
	if !ok {
		return nil, status.Error(codes.Internal, "reverse port forward authorization disappeared")
	}
	req.AuthorizationID = authorizationID.String()
	req.ForwardAddress = authorization.Address

	resp := &sliverpb.RportFwdListener{Response: &commonpb.Response{}}
	err = rpc.invokeGenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Response.GetErr() != "" {
		return listenerFromAuthorization(authorization, resp.Response), nil
	}
	if resp.ID == 0 {
		return nil, status.Error(codes.FailedPrecondition, "implant returned an invalid reverse port forward listener ID")
	}
	requiresAuthorizationID := resp.AuthorizationID == authorizationID.String()
	if err := registry.ActivateProtocol(sessionID, authorizationID, resp.ID, requiresAuthorizationID); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("failed to activate reverse port forward authorization: %v", err))
	}

	rollback = false
	authorization, _ = registry.Lookup(sessionID, authorizationID)
	return listenerFromAuthorization(authorization, resp.Response), nil
}

// StopRportfwdListener - Instruct the implant to stop a reverse port forward
func (rpc *Server) StopRportFwdListener(ctx context.Context, req *sliverpb.RportFwdStopListenerReq) (*sliverpb.RportFwdListener, error) {
	if req == nil || req.Request == nil {
		return nil, ErrMissingRequestField
	}

	// Revoke before asking the implant to stop. A malicious, disconnected, or
	// buggy implant cannot retain teamserver dial authority by failing its reply.
	sessionID := req.Request.SessionID
	registry := rpc.rportFwdRegistry()
	authorization, found := registry.LookupListener(sessionID, req.ID)
	registry.RevokeListener(sessionID, req.ID)
	if found {
		rtunnels.CloseAuthorization(sessionID, authorization.AuthorizationID)
	}

	resp := &sliverpb.RportFwdListener{Response: &commonpb.Response{}}
	err := rpc.invokeGenericHandler(req, resp)
	if err != nil {
		return nil, err
	}
	if found {
		authorization.State = rtunnels.AuthorizationRevoked
		return listenerFromAuthorization(authorization, resp.Response), nil
	}
	// There was no server authorization, but still allow the operator to clean
	// up an implant-side listener left by older state. Do not trust its metadata.
	resp.ID = req.ID
	resp.BindAddress = ""
	resp.BindPort = 0
	resp.ForwardAddress = ""
	resp.ForwardPort = 0
	resp.AuthorizationID = ""
	return resp, nil
}

func listenerFromAuthorization(authorization rtunnels.Authorization, response *commonpb.Response) *sliverpb.RportFwdListener {
	authorizationID := ""
	if authorization.RequiresAuthorizationID {
		authorizationID = authorization.AuthorizationID.String()
	}
	return &sliverpb.RportFwdListener{
		ID:              authorization.ImplantListenerID,
		BindAddress:     authorization.BindAddress,
		ForwardAddress:  authorization.Address,
		AuthorizationID: authorizationID,
		Response:        response,
	}
}
