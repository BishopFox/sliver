package rpc

import (
	"context"
	"errors"
	"testing"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core/rtunnels"
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
	server := &Server{reversePortForwardRegistry: registry, genericHandler: func(request GenericRequest, response GenericResponse) error {
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

func TestGetRportFwdListenersReturnsOnlyServerRegistryMetadata(t *testing.T) {
	const (
		sessionID  = "operator-session"
		listenerID = uint32(91)
	)
	registry := newTestRportFwdRegistry(t)
	authorizationID, err := registry.BeginSpec(sessionID, "0.0.0.0:8080", "Example.COM.:0443", 11)
	if err != nil {
		t.Fatalf("BeginSpec() error = %v", err)
	}
	if err := registry.ActivateProtocol(sessionID, authorizationID, listenerID, true); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if _, err := registry.Begin(sessionID, "127.0.0.1:5555", 0); err != nil {
		t.Fatalf("Begin() starting authorization error = %v", err)
	}
	otherID, err := registry.Begin("other-session", "127.0.0.1:6666", 0)
	if err != nil {
		t.Fatalf("Begin() other session error = %v", err)
	}
	if err := registry.Activate("other-session", otherID, 92); err != nil {
		t.Fatalf("Activate() other session error = %v", err)
	}

	genericCalled := false
	server := &Server{reversePortForwardRegistry: registry, genericHandler: func(request GenericRequest, response GenericResponse) error {
		genericCalled = true
		return errors.New("listener inventory must not contact the implant")
	}}

	response, err := server.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil {
		t.Fatalf("GetRportFwdListeners() error = %v", err)
	}
	if genericCalled {
		t.Fatal("server-authoritative inventory contacted the implant")
	}
	if len(response.Listeners) != 1 {
		t.Fatalf("listener count = %d, want 1: %#v", len(response.Listeners), response.Listeners)
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
			server := &Server{reversePortForwardRegistry: registry, genericHandler: func(request GenericRequest, response GenericResponse) error {
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

func newTestRportFwdRegistry(t *testing.T) *rtunnels.Registry {
	t.Helper()
	return rtunnels.NewRegistry()
}

func assertAuthorizationRemoved(t *testing.T, registry *rtunnels.Registry, sessionID string, authorizationID rtunnels.AuthorizationID) {
	t.Helper()
	if authorization, ok := registry.Lookup(sessionID, authorizationID); ok {
		t.Fatalf("authorization %q retained after revocation: %#v", authorizationID, authorization)
	}
}
