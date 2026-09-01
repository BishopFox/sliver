package rpc

import (
	"bytes"
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestStartRportFwdListenerIgnoresImplantDestination(t *testing.T) {
	const (
		sessionID       = "operator-session"
		operatorInput   = "Example.COM.:04444"
		operatorTarget  = "example.com:4444"
		implantTarget   = "169.254.169.254:80"
		implantListener = uint32(73)
	)
	registry := newTestRportFwdRegistry(t)

	var authorizationID rtunnels.AuthorizationID
	server := &Server{reversePortForwardRegistry: registry, genericHandler: func(request GenericRequest, response GenericResponse) error {
		startRequest, ok := request.(*sliverpb.RportFwdStartListenerReq)
		if !ok {
			t.Fatalf("generic request type = %T", request)
		}
		if startRequest.AuthorizationID == "" {
			t.Fatal("teamserver did not attach an authorization ID before contacting implant")
		}
		if startRequest.ForwardAddress != operatorTarget {
			t.Fatalf("forwarded destination = %q, want canonical operator destination %q", startRequest.ForwardAddress, operatorTarget)
		}
		authorizationID = rtunnels.AuthorizationID(startRequest.AuthorizationID)

		startResponse, ok := response.(*sliverpb.RportFwdListener)
		if !ok {
			t.Fatalf("generic response type = %T", response)
		}
		startResponse.ID = implantListener
		startResponse.BindAddress = "implant-controlled-bind:1"
		startResponse.ForwardAddress = implantTarget
		startResponse.AuthorizationID = startRequest.AuthorizationID
		startResponse.Response = &commonpb.Response{}
		return nil
	}}

	request := &sliverpb.RportFwdStartListenerReq{
		BindAddress:    "0.0.0.0:8080",
		ForwardAddress: operatorInput,
		KeepAlive:      17,
		Request:        &commonpb.Request{SessionID: sessionID},
	}
	response, err := server.StartRportFwdListener(context.Background(), request)
	if err != nil {
		t.Fatalf("StartRportFwdListener() error = %v", err)
	}
	if authorizationID == "" {
		t.Fatal("generic handler was not invoked")
	}

	authorization, ok := registry.LookupListener(sessionID, implantListener)
	if !ok {
		t.Fatalf("listener %d was not activated", implantListener)
	}
	if authorization.AuthorizationID != authorizationID {
		t.Fatalf("listener authorization = %q, want %q", authorization.AuthorizationID, authorizationID)
	}
	if authorization.Address != operatorTarget {
		t.Fatalf("stored destination = %q, want operator destination %q", authorization.Address, operatorTarget)
	}
	if authorization.Address == implantTarget {
		t.Fatalf("implant response poisoned stored destination: %q", authorization.Address)
	}
	if !authorization.RequiresAuthorizationID {
		t.Fatal("implant authorization echo did not enable strict tunnel opens")
	}
	if response.ForwardAddress != operatorTarget {
		t.Fatalf("returned destination = %q, want server-authoritative %q", response.ForwardAddress, operatorTarget)
	}
	if response.AuthorizationID != authorizationID.String() {
		t.Fatalf("returned authorization = %q, want %q", response.AuthorizationID, authorizationID)
	}
}

func TestStartRportFwdListenerNegotiatesLegacyCompatibility(t *testing.T) {
	const sessionID = "legacy-session"
	registry := newTestRportFwdRegistry(t)
	server := &Server{reversePortForwardRegistry: registry, genericHandler: func(_ GenericRequest, response GenericResponse) error {
		startResponse := response.(*sliverpb.RportFwdListener)
		startResponse.ID = 17
		startResponse.AuthorizationID = ""
		startResponse.Response = &commonpb.Response{}
		return nil
	}}

	response, err := server.StartRportFwdListener(context.Background(), &sliverpb.RportFwdStartListenerReq{
		BindAddress:    "127.0.0.1:8080",
		ForwardAddress: "127.0.0.1:4444",
		Request:        &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("StartRportFwdListener() error = %v", err)
	}
	if response.GetAuthorizationID() != "" {
		t.Fatalf("legacy listener advertised strict authorization %q", response.GetAuthorizationID())
	}
	authorization, ok := registry.LookupListener(sessionID, 17)
	if !ok {
		t.Fatal("legacy listener was not registered")
	}
	if authorization.RequiresAuthorizationID {
		t.Fatal("legacy implant without an authorization echo became strict")
	}
}

//nolint:gocyclo // The test verifies registry authority, orphan sanitization, and bounded inventory behavior together.
func TestGetRportFwdListenersKeepsRegistryMetadataAndSanitizesUnknownIDs(t *testing.T) {
	const (
		sessionID  = "operator-session"
		listenerID = uint32(91)
		orphanID   = uint32(77)
	)
	registry := newTestRportFwdRegistry(t)
	authorizationID, err := registry.BeginSpec(sessionID, "0.0.0.0:8080", "Example.COM.:0443", 11)
	if err != nil {
		t.Fatalf("BeginSpec() error = %v", err)
	}
	if err := registry.ActivateProtocol(sessionID, authorizationID, listenerID, true); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	otherID, err := registry.Begin("other-session", "127.0.0.1:6666", 0)
	if err != nil {
		t.Fatalf("Begin() other session error = %v", err)
	}
	if err := registry.Activate("other-session", otherID, 92); err != nil {
		t.Fatalf("Activate() other session error = %v", err)
	}

	queryCalled := false
	server := &Server{reversePortForwardRegistry: registry, rportFwdInventoryQuery: func(_ context.Context, inventoryRequest *sliverpb.RportFwdListenersReq) ([]byte, error) {
		queryCalled = true
		if got := time.Duration(inventoryRequest.Request.Timeout); got != bestEffortRportFwdInventoryTimeout {
			t.Fatalf("inventory timeout = %v, want %v", got, bestEffortRportFwdInventoryTimeout)
		}
		inventoryResponse := &sliverpb.RportFwdListeners{Response: &commonpb.Response{}, Listeners: []*sliverpb.RportFwdListener{
			{
				ID:              listenerID,
				BindAddress:     "implant-poisoned-bind:1",
				ForwardAddress:  "169.254.169.254:80",
				AuthorizationID: "implant-poisoned-authorization",
			},
			{
				ID:              orphanID,
				BindAddress:     "implant-orphan-bind:2",
				ForwardAddress:  "127.0.0.1:1",
				AuthorizationID: "implant-orphan-authorization",
			},
			{ID: orphanID, BindAddress: "duplicate-poison"},
			nil,
			{ID: 0, BindAddress: "zero-poison"},
		}}
		return mustMarshalRportFwdInventory(t, inventoryResponse), nil
	}}

	response, err := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", err)
	}
	if !queryCalled {
		t.Fatal("best-effort implant inventory was not requested")
	}
	if len(response.Listeners) != 2 {
		t.Fatalf("listener count = %d, want 2: %#v", len(response.Listeners), response.Listeners)
	}
	listener := response.Listeners[0]
	if listener.ID != listenerID {
		t.Fatalf("listener ID = %d, want %d", listener.ID, listenerID)
	}
	if listener.BindAddress != "0.0.0.0:8080" {
		t.Fatalf("bind address = %q, want registry value", listener.BindAddress)
	}
	if listener.ForwardAddress != "example.com:443" {
		t.Fatalf("forward address = %q, want canonical registry value", listener.ForwardAddress)
	}
	if listener.AuthorizationID != authorizationID.String() {
		t.Fatalf("authorization ID = %q, want %q", listener.AuthorizationID, authorizationID)
	}
	orphan := response.Listeners[1]
	if orphan.ID != orphanID {
		t.Fatalf("orphan listener ID = %d, want %d", orphan.ID, orphanID)
	}
	if orphan.BindAddress != "" || orphan.ForwardAddress != "" || orphan.AuthorizationID != "" || orphan.BindPort != 0 || orphan.ForwardPort != 0 {
		t.Fatalf("orphan metadata was not scrubbed: %#v", orphan)
	}
}

func TestGetRportFwdListenersRegistryActivationDuringInventoryWins(t *testing.T) {
	const (
		sessionID  = "operator-session"
		listenerID = uint32(94)
	)
	registry := newTestRportFwdRegistry(t)
	var authorizationID rtunnels.AuthorizationID
	server := &Server{reversePortForwardRegistry: registry, rportFwdInventoryQuery: func(_ context.Context, _ *sliverpb.RportFwdListenersReq) ([]byte, error) {
		var err error
		authorizationID, err = registry.BeginSpec(sessionID, "127.0.0.1:8080", "127.0.0.1:4444", 0)
		if err != nil {
			t.Fatalf("BeginSpec() error = %v", err)
		}
		if err := registry.ActivateProtocol(sessionID, authorizationID, listenerID, true); err != nil {
			t.Fatalf("ActivateProtocol() error = %v", err)
		}
		inventoryResponse := &sliverpb.RportFwdListeners{Response: &commonpb.Response{}, Listeners: []*sliverpb.RportFwdListener{{
			ID:              listenerID,
			BindAddress:     "implant-poisoned-bind:1",
			ForwardAddress:  "169.254.169.254:80",
			AuthorizationID: "implant-poisoned-authorization",
		}}}
		return mustMarshalRportFwdInventory(t, inventoryResponse), nil
	}}

	response, err := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", err)
	}
	if len(response.Listeners) != 1 {
		t.Fatalf("listener count = %d, want 1", len(response.Listeners))
	}
	listener := response.Listeners[0]
	if listener.ID != listenerID || listener.BindAddress != "127.0.0.1:8080" || listener.ForwardAddress != "127.0.0.1:4444" || listener.AuthorizationID != authorizationID.String() {
		t.Fatalf("racing activation did not retain registry metadata: %#v", listener)
	}
}

func TestGetRportFwdListenersSuppressesKnownListenerRevokedDuringProbe(t *testing.T) {
	const (
		sessionID  = "operator-session"
		listenerID = uint32(97)
	)
	registry := newTestRportFwdRegistry(t)
	authorizationID, err := registry.BeginSpec(sessionID, "127.0.0.1:8080", "127.0.0.1:4444", 0)
	if err != nil {
		t.Fatalf("BeginSpec() error = %v", err)
	}
	if err := registry.ActivateProtocol(sessionID, authorizationID, listenerID, true); err != nil {
		t.Fatalf("ActivateProtocol() error = %v", err)
	}
	probeCaptured := make(chan struct{})
	releaseProbe := make(chan struct{})
	server := &Server{
		reversePortForwardRegistry: registry,
		rportFwdInventoryQuery: func(context.Context, *sliverpb.RportFwdListenersReq) ([]byte, error) {
			raw := mustMarshalRportFwdInventory(t, &sliverpb.RportFwdListeners{
				Response:  &commonpb.Response{},
				Listeners: []*sliverpb.RportFwdListener{{ID: listenerID}},
			})
			close(probeCaptured)
			<-releaseProbe
			return raw, nil
		},
		genericHandler: func(_ GenericRequest, response GenericResponse) error {
			stopResponse := response.(*sliverpb.RportFwdListener)
			stopResponse.ID = listenerID
			stopResponse.Response = &commonpb.Response{}
			return nil
		},
	}
	getResult := make(chan *sliverpb.RportFwdListeners, 1)
	getErr := make(chan error, 1)
	go func() {
		response, rpcErr := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
			Request: &commonpb.Request{SessionID: sessionID},
		})
		getResult <- response
		getErr <- rpcErr
	}()
	select {
	case <-probeCaptured:
	case <-time.After(time.Second):
		close(releaseProbe)
		t.Fatal("inventory probe did not capture listener")
	}
	if _, err := server.StopRportFwdListener(context.Background(), &sliverpb.RportFwdStopListenerReq{
		ID:      listenerID,
		Request: &commonpb.Request{SessionID: sessionID},
	}); err != nil {
		close(releaseProbe)
		t.Fatalf("StopRportFwdListener() error = %v", err)
	}
	close(releaseProbe)
	var response *sliverpb.RportFwdListeners
	select {
	case response = <-getResult:
	case <-time.After(time.Second):
		t.Fatal("inventory did not return after probe release")
	}
	var rpcErr error
	select {
	case rpcErr = <-getErr:
	case <-time.After(time.Second):
		t.Fatal("inventory error result did not return after probe release")
	}
	if rpcErr != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", rpcErr)
	}
	if len(response.Listeners) != 0 {
		t.Fatalf("revoked listener resurfaced as compatibility orphan: %#v", response.Listeners)
	}
}

func TestGetRportFwdListenersSuppressesUntrustedIDsWhileStartIsPending(t *testing.T) {
	const (
		sessionID  = "operator-session"
		listenerID = uint32(98)
	)
	registry := newTestRportFwdRegistry(t)
	if _, err := registry.BeginSpec(sessionID, "127.0.0.1:8080", "127.0.0.1:4444", 0); err != nil {
		t.Fatalf("BeginSpec() error = %v", err)
	}
	server := &Server{reversePortForwardRegistry: registry, rportFwdInventoryQuery: func(context.Context, *sliverpb.RportFwdListenersReq) ([]byte, error) {
		return mustMarshalRportFwdInventory(t, &sliverpb.RportFwdListeners{
			Response:  &commonpb.Response{},
			Listeners: []*sliverpb.RportFwdListener{{ID: listenerID}},
		}), nil
	}}

	response, err := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", err)
	}
	if len(response.Listeners) != 0 {
		t.Fatalf("pending start exposed untrusted listener ID: %#v", response.Listeners)
	}
}

func TestGetRportFwdListenersInventoryFailurePreservesRegistryEntries(t *testing.T) {
	const (
		sessionID  = "operator-session"
		listenerID = uint32(93)
	)
	registry := newTestRportFwdRegistry(t)
	authorizationID, err := registry.BeginSpec(sessionID, "127.0.0.1:8080", "127.0.0.1:4444", 0)
	if err != nil {
		t.Fatalf("BeginSpec() error = %v", err)
	}
	if err := registry.ActivateProtocol(sessionID, authorizationID, listenerID, true); err != nil {
		t.Fatalf("ActivateProtocol() error = %v", err)
	}
	tests := []struct {
		name    string
		handler func(context.Context, *sliverpb.RportFwdListenersReq) ([]byte, error)
	}{
		{
			name: "transport error",
			handler: func(context.Context, *sliverpb.RportFwdListenersReq) ([]byte, error) {
				return nil, errors.New("implant inventory unavailable")
			},
		},
		{
			name: "implant response error",
			handler: func(_ context.Context, _ *sliverpb.RportFwdListenersReq) ([]byte, error) {
				inventoryResponse := &sliverpb.RportFwdListeners{Response: &commonpb.Response{Err: "implant inventory refused"}, Listeners: []*sliverpb.RportFwdListener{{
					ID:              999,
					BindAddress:     "implant-controlled-bind",
					ForwardAddress:  "169.254.169.254:80",
					AuthorizationID: "implant-controlled-authorization",
				}}}
				return mustMarshalRportFwdInventory(t, inventoryResponse), nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := &Server{reversePortForwardRegistry: registry, rportFwdInventoryQuery: test.handler}
			response, err := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
				Request: &commonpb.Request{SessionID: sessionID},
			})
			if err != nil {
				t.Fatalf("GetRportFwdListeners() error = %v", err)
			}
			if response.Response == nil || response.Response.Err != "" {
				t.Fatalf("inventory failure leaked response error: %#v", response.Response)
			}
			if len(response.Listeners) != 1 {
				t.Fatalf("listener count = %d, want 1", len(response.Listeners))
			}
			listener := response.Listeners[0]
			if listener.ID != listenerID || listener.BindAddress != "127.0.0.1:8080" || listener.ForwardAddress != "127.0.0.1:4444" || listener.AuthorizationID != authorizationID.String() {
				t.Fatalf("inventory failure changed registry listener: %#v", listener)
			}
		})
	}
}

func TestGetRportFwdListenersBoundsUntrustedInventory(t *testing.T) {
	const sessionID = "operator-session"
	server := &Server{reversePortForwardRegistry: newTestRportFwdRegistry(t), rportFwdInventoryQuery: func(_ context.Context, _ *sliverpb.RportFwdListenersReq) ([]byte, error) {
		inventoryResponse := &sliverpb.RportFwdListeners{Response: &commonpb.Response{}}
		inventoryResponse.Listeners = make([]*sliverpb.RportFwdListener, maxUntrustedRportFwdInventoryIDs)
		for index := range inventoryResponse.Listeners {
			inventoryResponse.Listeners[index] = &sliverpb.RportFwdListener{
				ID:              uint32(index + 1),
				BindAddress:     "implant-controlled-bind",
				ForwardAddress:  "169.254.169.254:80",
				AuthorizationID: "implant-controlled-authorization",
			}
		}
		return mustMarshalRportFwdInventory(t, inventoryResponse), nil
	}}

	response, err := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", err)
	}
	if len(response.Listeners) != maxUntrustedRportFwdInventoryIDs {
		t.Fatalf("listener count = %d, want bounded %d", len(response.Listeners), maxUntrustedRportFwdInventoryIDs)
	}
	for index, listener := range response.Listeners {
		if listener.ID != uint32(index+1) {
			t.Fatalf("listener %d ID = %d, want %d", index, listener.ID, index+1)
		}
		if listener.BindAddress != "" || listener.ForwardAddress != "" || listener.AuthorizationID != "" {
			t.Fatalf("listener %d retained untrusted metadata: %#v", index, listener)
		}
	}
}

func TestParseRportFwdInventoryRejectsMalformedAndAmplifyingInputs(t *testing.T) {
	wrongListenerWire := protowire.AppendTag(nil, 1, protowire.VarintType)
	wrongListenerWire = protowire.AppendVarint(wrongListenerWire, 1)

	wrongIDWireEntry := protowire.AppendTag(nil, 1, protowire.BytesType)
	wrongIDWireEntry = protowire.AppendBytes(wrongIDWireEntry, []byte("not-a-varint"))
	wrongIDWire := protowire.AppendTag(nil, 1, protowire.BytesType)
	wrongIDWire = protowire.AppendBytes(wrongIDWire, wrongIDWireEntry)

	overflowIDEntry := protowire.AppendTag(nil, 1, protowire.VarintType)
	overflowIDEntry = protowire.AppendVarint(overflowIDEntry, uint64(math.MaxUint32)+1)
	overflowID := protowire.AppendTag(nil, 1, protowire.BytesType)
	overflowID = protowire.AppendBytes(overflowID, overflowIDEntry)

	wrongResponseWire := protowire.AppendTag(nil, 9, protowire.VarintType)
	wrongResponseWire = protowire.AppendVarint(wrongResponseWire, 1)
	topLevelGroup := protowire.AppendTag(nil, 2, protowire.StartGroupType)
	topLevelGroup = protowire.AppendTag(topLevelGroup, 2, protowire.EndGroupType)
	listenerGroupEntry := protowire.AppendTag(nil, 2, protowire.StartGroupType)
	listenerGroupEntry = protowire.AppendTag(listenerGroupEntry, 2, protowire.EndGroupType)
	listenerGroup := protowire.AppendTag(nil, 1, protowire.BytesType)
	listenerGroup = protowire.AppendBytes(listenerGroup, listenerGroupEntry)
	responseGroupEntry := protowire.AppendTag(nil, 2, protowire.StartGroupType)
	responseGroupEntry = protowire.AppendTag(responseGroupEntry, 2, protowire.EndGroupType)
	responseGroup := protowire.AppendTag(nil, 9, protowire.BytesType)
	responseGroup = protowire.AppendBytes(responseGroup, responseGroupEntry)
	tooManyTopLevelFields := bytes.Repeat([]byte{0x10, 0x00}, maxRportFwdInventoryTopLevelFields+1)
	tooManyNestedFieldsEntry := bytes.Repeat([]byte{0x10, 0x00}, maxRportFwdInventoryNestedFields+1)
	tooManyNestedFields := protowire.AppendTag(nil, 1, protowire.BytesType)
	tooManyNestedFields = protowire.AppendBytes(tooManyNestedFields, tooManyNestedFieldsEntry)
	oversizedListenerEntry := protowire.AppendTag(nil, 1, protowire.BytesType)
	oversizedListenerEntry = protowire.AppendBytes(oversizedListenerEntry, make([]byte, maxRportFwdInventoryNestedMessageBytes+1))
	oversizedResponseEntry := protowire.AppendTag(nil, 9, protowire.BytesType)
	oversizedResponseEntry = protowire.AppendBytes(oversizedResponseEntry, make([]byte, maxRportFwdInventoryNestedMessageBytes+1))

	// Two bytes encode one empty listener. This represents more than two
	// million repeated objects at the raw response cap, but the streaming parser
	// rejects it at the structural listener-entry limit.
	millionsOfEmptyListeners := bytes.Repeat([]byte{0x0a, 0x00}, maxRportFwdInventoryResponseBytes/2)

	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "truncated length", raw: []byte{0x0a, 0x80}},
		{name: "wrong listener wire type", raw: wrongListenerWire},
		{name: "wrong ID wire type", raw: wrongIDWire},
		{name: "overflowing ID", raw: overflowID},
		{name: "wrong response wire type", raw: wrongResponseWire},
		{name: "top-level group", raw: topLevelGroup},
		{name: "listener group", raw: listenerGroup},
		{name: "response group", raw: responseGroup},
		{name: "too many top-level fields", raw: tooManyTopLevelFields},
		{name: "too many nested fields", raw: tooManyNestedFields},
		{name: "oversized listener entry", raw: oversizedListenerEntry},
		{name: "oversized response entry", raw: oversizedResponseEntry},
		{name: "millions of empty listeners", raw: millionsOfEmptyListeners},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			listenerIDs, responseError, err := parseRportFwdInventoryIDs(test.raw)
			if err == nil {
				t.Fatalf("parseRportFwdInventoryIDs() IDs = %v, response error = %v, want structural error", listenerIDs, responseError)
			}
		})
	}
}

func TestGetRportFwdListenersOversizedRawResponsePreservesRegistry(t *testing.T) {
	const (
		sessionID  = "operator-session"
		listenerID = uint32(96)
	)
	registry := newTestRportFwdRegistry(t)
	authorizationID, err := registry.BeginSpec(sessionID, "127.0.0.1:8080", "127.0.0.1:4444", 0)
	if err != nil {
		t.Fatalf("BeginSpec() error = %v", err)
	}
	if err := registry.ActivateProtocol(sessionID, authorizationID, listenerID, true); err != nil {
		t.Fatalf("ActivateProtocol() error = %v", err)
	}
	server := &Server{reversePortForwardRegistry: registry, rportFwdInventoryQuery: func(context.Context, *sliverpb.RportFwdListenersReq) ([]byte, error) {
		// Invalid protobuf at one byte over the application cap proves the size
		// check runs before any inventory parsing or object materialization.
		return make([]byte, maxRportFwdInventoryResponseBytes+1), nil
	}}

	response, err := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", err)
	}
	if len(response.Listeners) != 1 {
		t.Fatalf("listener count = %d, want 1", len(response.Listeners))
	}
	listener := response.Listeners[0]
	if listener.ID != listenerID || listener.AuthorizationID != authorizationID.String() {
		t.Fatalf("oversized inventory changed registry listener: %#v", listener)
	}
}

func TestGetRportFwdListenersPreCanceledContextSkipsProbe(t *testing.T) {
	const sessionID = "operator-session"
	queryCalled := false
	server := &Server{reversePortForwardRegistry: newTestRportFwdRegistry(t), rportFwdInventoryQuery: func(context.Context, *sliverpb.RportFwdListenersReq) ([]byte, error) {
		queryCalled = true
		return nil, nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	response, err := server.GetRportFwdListeners(ctx, &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", err)
	}
	if queryCalled {
		t.Fatal("pre-canceled inventory request invoked probe")
	}
	if len(response.Listeners) != 0 {
		t.Fatalf("pre-canceled inventory returned listeners: %#v", response.Listeners)
	}
}

func TestRportFwdInventoryProbeLimiterIsNonblockingAndRecoverable(t *testing.T) {
	t.Run("global capacity", func(t *testing.T) {
		for index := 0; index < maxConcurrentRportFwdInventoryProbes; index++ {
			release := tryAcquireRportFwdInventoryProbe(string(rune('a' + index)))
			if release == nil {
				t.Fatalf("global probe slot %d was unexpectedly unavailable", index)
			}
			defer release()
		}
		if release := tryAcquireRportFwdInventoryProbe("overflow-session"); release != nil {
			defer release()
			t.Fatal("probe limiter exceeded its global capacity")
		}
	})

	release := tryAcquireRportFwdInventoryProbe("same-session")
	if release == nil {
		t.Fatal("probe limiter did not recover after releases")
	}
	defer release()
	if duplicateRelease := tryAcquireRportFwdInventoryProbe("same-session"); duplicateRelease != nil {
		duplicateRelease()
		t.Fatal("probe limiter allowed concurrent probes for one session")
	}
}

func TestGetRportFwdListenersContextCancellationPreservesRegistryEntries(t *testing.T) {
	const (
		sessionID  = "operator-session"
		listenerID = uint32(95)
	)
	registry := newTestRportFwdRegistry(t)
	authorizationID, err := registry.BeginSpec(sessionID, "127.0.0.1:8080", "127.0.0.1:4444", 0)
	if err != nil {
		t.Fatalf("BeginSpec() error = %v", err)
	}
	if err := registry.ActivateProtocol(sessionID, authorizationID, listenerID, true); err != nil {
		t.Fatalf("ActivateProtocol() error = %v", err)
	}
	queryStarted := make(chan struct{})
	queryFinished := make(chan struct{})
	server := &Server{reversePortForwardRegistry: registry, rportFwdInventoryQuery: func(ctx context.Context, _ *sliverpb.RportFwdListenersReq) ([]byte, error) {
		close(queryStarted)
		<-ctx.Done()
		close(queryFinished)
		return nil, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan *sliverpb.RportFwdListeners, 1)
	errResult := make(chan error, 1)
	go func() {
		response, rpcErr := server.GetRportFwdListeners(ctx, &sliverpb.RportFwdListenersReq{
			Request: &commonpb.Request{SessionID: sessionID},
		})
		result <- response
		errResult <- rpcErr
	}()
	select {
	case <-queryStarted:
	case <-time.After(time.Second):
		t.Fatal("implant inventory query did not start")
	}
	cancel()

	var response *sliverpb.RportFwdListeners
	select {
	case response = <-result:
	case <-time.After(time.Second):
		t.Fatal("context cancellation did not bound implant inventory")
	}
	if rpcErr := <-errResult; rpcErr != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", rpcErr)
	}
	if len(response.Listeners) != 1 || response.Listeners[0].ID != listenerID || response.Listeners[0].AuthorizationID != authorizationID.String() {
		t.Fatalf("context cancellation lost registry listener: %#v", response.Listeners)
	}
	select {
	case <-queryFinished:
	case <-time.After(time.Second):
		t.Fatal("released implant inventory query did not finish")
	}
}

func TestStartRportFwdListenerFailuresRevokeCandidateAuthorization(t *testing.T) {
	transportFailure := errors.New("implant disconnected")
	tests := []struct {
		name            string
		listenerID      uint32
		implantErr      string
		transportErr    error
		duplicate       bool
		wantRPCErr      bool
		wantActiveCount int
	}{
		{name: "transport error", listenerID: 10, transportErr: transportFailure, wantRPCErr: true},
		{name: "implant response error", listenerID: 11, implantErr: "implant failed to bind"},
		{name: "zero listener ID", listenerID: 0, wantRPCErr: true},
		{name: "duplicate listener ID", listenerID: 12, duplicate: true, wantRPCErr: true, wantActiveCount: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const sessionID = "operator-session"
			registry := newTestRportFwdRegistry(t)
			var existingID rtunnels.AuthorizationID
			if test.duplicate {
				var err error
				existingID, err = registry.Begin(sessionID, "127.0.0.1:3333", 0)
				if err != nil {
					t.Fatalf("Begin() existing authorization error = %v", err)
				}
				if err := registry.Activate(sessionID, existingID, test.listenerID); err != nil {
					t.Fatalf("Activate() existing authorization error = %v", err)
				}
			}

			var candidateID rtunnels.AuthorizationID
			server := &Server{reversePortForwardRegistry: registry, genericHandler: func(request GenericRequest, response GenericResponse) error {
				startRequest := request.(*sliverpb.RportFwdStartListenerReq)
				candidateID = rtunnels.AuthorizationID(startRequest.AuthorizationID)
				if test.transportErr != nil {
					return test.transportErr
				}
				startResponse := response.(*sliverpb.RportFwdListener)
				startResponse.ID = test.listenerID
				startResponse.Response = &commonpb.Response{Err: test.implantErr}
				return nil
			}}

			_, rpcErr := server.StartRportFwdListener(context.Background(), &sliverpb.RportFwdStartListenerReq{
				BindAddress:    "0.0.0.0:8080",
				ForwardAddress: "127.0.0.1:4444",
				Request:        &commonpb.Request{SessionID: sessionID},
			})
			if candidateID == "" {
				t.Fatal("start request did not carry candidate authorization")
			}
			if (rpcErr != nil) != test.wantRPCErr {
				t.Fatalf("StartRportFwdListener() error = %v, want error %v", rpcErr, test.wantRPCErr)
			}
			assertAuthorizationRemoved(t, registry, sessionID, candidateID)
			if got := len(registry.List(sessionID)); got != test.wantActiveCount {
				t.Fatalf("active authorization count = %d, want %d", got, test.wantActiveCount)
			}
			if test.duplicate {
				authorization, ok := registry.LookupListener(sessionID, test.listenerID)
				if !ok || authorization.AuthorizationID != existingID {
					t.Fatalf("duplicate listener replaced existing authorization: %#v", authorization)
				}
			}
		})
	}
}

func TestStopRportFwdListenerRevokesBeforeImplantResponse(t *testing.T) {
	const (
		sessionID = "operator-session"
		listener  = uint32(81)
	)

	tests := []struct {
		name         string
		implantErr   string
		transportErr error
		wantRPCErr   bool
	}{
		{name: "implant response error", implantErr: "implant refused stop"},
		{name: "transport error", transportErr: errors.New("implant disconnected"), wantRPCErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := newTestRportFwdRegistry(t)
			authorizationID, err := registry.Begin(sessionID, "127.0.0.1:4444", 0)
			if err != nil {
				t.Fatalf("Begin() error = %v", err)
			}
			if err := registry.Activate(sessionID, authorizationID, listener); err != nil {
				t.Fatalf("Activate() error = %v", err)
			}

			called := false
			server := &Server{reversePortForwardRegistry: registry, genericHandler: func(_ GenericRequest, response GenericResponse) error {
				called = true
				assertAuthorizationRemoved(t, registry, sessionID, authorizationID)
				stopResponse := response.(*sliverpb.RportFwdListener)
				stopResponse.ID = listener
				stopResponse.Response = &commonpb.Response{Err: test.implantErr}
				return test.transportErr
			}}

			response, rpcErr := server.StopRportFwdListener(context.Background(), &sliverpb.RportFwdStopListenerReq{
				ID:      listener,
				Request: &commonpb.Request{SessionID: sessionID},
			})
			if !called {
				t.Fatal("implant stop handler was not invoked")
			}
			if (rpcErr != nil) != test.wantRPCErr {
				t.Fatalf("StopRportFwdListener() error = %v, want error %v", rpcErr, test.wantRPCErr)
			}
			if !test.wantRPCErr {
				if response == nil || response.GetResponse().GetErr() != test.implantErr {
					t.Fatalf("implant error response = %#v, want %q", response, test.implantErr)
				}
			}
			assertAuthorizationRemoved(t, registry, sessionID, authorizationID)
			if _, ok := registry.LookupListener(sessionID, listener); ok {
				t.Fatalf("listener %d remained registered after stop", listener)
			}
		})
	}
}

//nolint:gocyclo // The test keeps inventory, stop, sanitization, and post-stop reconciliation in one lifecycle.
func TestLegacyOrphanInventoryStopLifecycle(t *testing.T) {
	const (
		sessionID = "reconnected-session"
		orphanID  = uint32(117)
	)
	registry := newTestRportFwdRegistry(t)
	implantListeners := map[uint32]*sliverpb.RportFwdListener{
		orphanID: {
			ID:              orphanID,
			BindAddress:     "implant-controlled-bind:1",
			BindPort:        1111,
			ForwardAddress:  "169.254.169.254:80",
			ForwardPort:     2222,
			AuthorizationID: "implant-controlled-authorization",
		},
	}
	server := &Server{
		reversePortForwardRegistry: registry,
		rportFwdInventoryQuery: func(_ context.Context, _ *sliverpb.RportFwdListenersReq) ([]byte, error) {
			inventoryResponse := &sliverpb.RportFwdListeners{Response: &commonpb.Response{}, Listeners: make([]*sliverpb.RportFwdListener, 0, len(implantListeners))}
			for _, listener := range implantListeners {
				inventoryResponse.Listeners = append(inventoryResponse.Listeners, listener)
			}
			return mustMarshalRportFwdInventory(t, inventoryResponse), nil
		},
		genericHandler: func(request GenericRequest, response GenericResponse) error {
			switch typedRequest := request.(type) {
			case *sliverpb.RportFwdStopListenerReq:
				listener, ok := implantListeners[typedRequest.ID]
				if !ok {
					return errors.New("legacy listener not found")
				}
				delete(implantListeners, typedRequest.ID)
				stopResponse := response.(*sliverpb.RportFwdListener)
				proto.Merge(stopResponse, listener)
				stopResponse.Response = &commonpb.Response{}
				return nil
			default:
				return errors.New("unexpected generic request type")
			}
		},
	}

	inventory, err := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", err)
	}
	if len(inventory.Listeners) != 1 {
		t.Fatalf("orphan inventory count = %d, want 1", len(inventory.Listeners))
	}
	orphan := inventory.Listeners[0]
	if orphan.ID != orphanID || orphan.BindAddress != "" || orphan.BindPort != 0 || orphan.ForwardAddress != "" || orphan.ForwardPort != 0 || orphan.AuthorizationID != "" {
		t.Fatalf("orphan inventory was not ID-only: %#v", orphan)
	}
	if got := registry.List(sessionID); len(got) != 0 {
		t.Fatalf("orphan inventory created %d authorizations", len(got))
	}

	stopped, err := server.StopRportFwdListener(context.Background(), &sliverpb.RportFwdStopListenerReq{
		ID:      orphanID,
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("StopRportFwdListener() error = %v", err)
	}
	if stopped.ID != orphanID {
		t.Fatalf("stopped listener ID = %d, want %d", stopped.ID, orphanID)
	}
	if stopped.BindAddress != "" || stopped.BindPort != 0 || stopped.ForwardAddress != "" || stopped.ForwardPort != 0 || stopped.AuthorizationID != "" {
		t.Fatalf("unregistered stop response retained implant metadata: %#v", stopped)
	}
	if len(implantListeners) != 0 {
		t.Fatalf("stop retained %d implant-side listeners", len(implantListeners))
	}
	if got := registry.List(sessionID); len(got) != 0 {
		t.Fatalf("unregistered stop created %d authorizations", len(got))
	}

	inventory, err = server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() after stop error = %v", err)
	}
	if len(inventory.Listeners) != 0 {
		t.Fatalf("stopped orphan remained in inventory: %#v", inventory.Listeners)
	}
	if got := registry.List(sessionID); len(got) != 0 {
		t.Fatalf("post-stop inventory created %d authorizations", len(got))
	}
}

func newTestRportFwdRegistry(t *testing.T) *rtunnels.Registry {
	t.Helper()
	return rtunnels.NewRegistry()
}

func mustMarshalRportFwdInventory(t *testing.T, inventory *sliverpb.RportFwdListeners) []byte {
	t.Helper()
	raw, err := proto.Marshal(inventory)
	if err != nil {
		t.Fatalf("proto.Marshal(rportfwd inventory) error = %v", err)
	}
	return raw
}

func assertAuthorizationRemoved(t *testing.T, registry *rtunnels.Registry, sessionID string, authorizationID rtunnels.AuthorizationID) {
	t.Helper()
	if authorization, ok := registry.Lookup(sessionID, authorizationID); ok {
		t.Fatalf("authorization %q retained after revocation: %#v", authorizationID, authorization)
	}
}
