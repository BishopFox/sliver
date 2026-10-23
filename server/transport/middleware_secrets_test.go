package transport

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/protobuf/proto"
)

func TestSanitizeAuditRequestRedactsCredentialLists(t *testing.T) {
	request := &clientpb.Credentials{Credentials: []*clientpb.Credential{
		{
			ID: "credential-one", Username: "alice", Plaintext: "secret-plaintext-one",
			Hash: "secret-hash-one", HashType: clientpb.HashType_SHA1,
			IsCracked: true, OriginHostUUID: "host-one", Collection: "collection-one",
		},
		{
			ID: "credential-two", Username: "bob", Plaintext: "secret-plaintext-two",
			Hash: "secret-hash-two", OriginHostUUID: "host-two",
		},
	}}
	original := proto.Clone(request)
	want := proto.Clone(request).(*clientpb.Credentials)
	for _, credential := range want.Credentials {
		credential.Plaintext = ""
		credential.Hash = ""
	}

	for _, method := range []string{
		rpcpb.SliverRPC_CredsAdd_FullMethodName,
		rpcpb.SliverRPC_CredsRm_FullMethodName,
		rpcpb.SliverRPC_CredsUpdate_FullMethodName,
	} {
		t.Run(method, func(t *testing.T) {
			got, ok := sanitizeAuditRequest(method, request).(*clientpb.Credentials)
			if !ok {
				t.Fatal("sanitized credential list has unexpected type")
			}
			if got == request || !proto.Equal(request, original) {
				t.Fatal("sanitization mutated or reused the handler request")
			}
			if !proto.Equal(got, want) {
				t.Fatalf("sanitized credentials = %v, want %v", got, want)
			}
			assertAuditJSONExcludesSecrets(t, got, "secret-plaintext", "secret-hash")
		})
	}
}

func TestSanitizeAuditRequestRedactsSingleCredentials(t *testing.T) {
	request := &clientpb.Credential{
		ID: "credential-id", Username: "alice", Plaintext: "secret-plaintext",
		Hash: "secret-hash", HashType: clientpb.HashType_SHA1,
		IsCracked: true, OriginHostUUID: "host-id", Collection: "collection-id",
	}
	original := proto.Clone(request)
	want := proto.Clone(request).(*clientpb.Credential)
	want.Plaintext = ""
	want.Hash = ""

	for _, method := range []string{
		rpcpb.SliverRPC_GetCredByID_FullMethodName,
		rpcpb.SliverRPC_GetCredsByHashType_FullMethodName,
		rpcpb.SliverRPC_GetPlaintextCredsByHashType_FullMethodName,
		rpcpb.SliverRPC_CredsSniffHashType_FullMethodName,
	} {
		t.Run(method, func(t *testing.T) {
			got, ok := sanitizeAuditRequest(method, request).(*clientpb.Credential)
			if !ok {
				t.Fatal("sanitized credential has unexpected type")
			}
			if got == request || !proto.Equal(request, original) {
				t.Fatal("sanitization mutated or reused the handler request")
			}
			if !proto.Equal(got, want) {
				t.Fatalf("sanitized credential = %v, want %v", got, want)
			}
			assertAuditJSONExcludesSecrets(t, got, "secret-plaintext", "secret-hash")
		})
	}
}

func TestSanitizeAuditRequestRedactsMonitoringProvider(t *testing.T) {
	request := &clientpb.MonitoringProvider{
		ID: "provider-id", Type: "provider-type",
		APIKey: "secret-api-key", APIPassword: "secret-api-password",
	}
	original := proto.Clone(request)
	want := &clientpb.MonitoringProvider{ID: request.ID, Type: request.Type}

	for _, method := range []string{
		rpcpb.SliverRPC_MonitorAddConfig_FullMethodName,
		rpcpb.SliverRPC_MonitorDelConfig_FullMethodName,
	} {
		t.Run(method, func(t *testing.T) {
			got, ok := sanitizeAuditRequest(method, request).(*clientpb.MonitoringProvider)
			if !ok {
				t.Fatal("sanitized provider has unexpected type")
			}
			if got == request || !proto.Equal(request, original) {
				t.Fatal("sanitization mutated or reused the handler request")
			}
			if !proto.Equal(got, want) {
				t.Fatalf("sanitized provider = %v, want %v", got, want)
			}
			assertAuditJSONExcludesSecrets(t, got, "secret-api-key", "secret-api-password")
		})
	}
}

func assertAuditJSONExcludesSecrets(t *testing.T, value proto.Message, secrets ...string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal sanitized audit request: %v", err)
	}
	for _, secret := range secrets {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("sanitized audit request contains %q: %s", secret, encoded)
		}
	}
}

func TestCredentialAndProviderPayloadLoggingDeciderAlwaysSuppressesSecrets(t *testing.T) {
	original := serverConfig.Logs.GRPCUnaryPayloads
	serverConfig.Logs.GRPCUnaryPayloads = true
	t.Cleanup(func() { serverConfig.Logs.GRPCUnaryPayloads = original })

	for _, method := range []string{
		rpcpb.SliverRPC_Creds_FullMethodName,
		rpcpb.SliverRPC_CredsAdd_FullMethodName,
		rpcpb.SliverRPC_CredsRm_FullMethodName,
		rpcpb.SliverRPC_CredsUpdate_FullMethodName,
		rpcpb.SliverRPC_GetCredByID_FullMethodName,
		rpcpb.SliverRPC_GetCredsByHashType_FullMethodName,
		rpcpb.SliverRPC_GetPlaintextCredsByHashType_FullMethodName,
		rpcpb.SliverRPC_CredsSniffHashType_FullMethodName,
		rpcpb.SliverRPC_MonitorListConfig_FullMethodName,
		rpcpb.SliverRPC_MonitorAddConfig_FullMethodName,
		rpcpb.SliverRPC_MonitorDelConfig_FullMethodName,
	} {
		if deciderUnary(context.Background(), method, nil) {
			t.Errorf("payload logging enabled for %s", method)
		}
	}
	if !deciderUnary(context.Background(), rpcpb.SliverRPC_Hosts_FullMethodName, nil) {
		t.Fatal("payload logging disabled for unrelated Hosts method")
	}
}
