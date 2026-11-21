package generate

import (
	"context"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type profileCompatRPC struct {
	rpcpb.SliverRPCClient
	definition *clientpb.BuildProfileDocument
	err        error
}

func (rpc *profileCompatRPC) GetBuildProfile(_ context.Context, _ *clientpb.BuildProfileRef, _ ...grpc.CallOption) (*clientpb.BuildProfileDocument, error) {
	return rpc.definition, rpc.err
}

func TestGetProfileDefinitionOrLegacy(t *testing.T) {
	unsupported := status.Error(codes.Unimplemented, "older server")
	definition := &clientpb.BuildProfileDocument{Name: "profile", Migrated: false}
	for _, test := range []struct {
		name       string
		definition *clientpb.BuildProfileDocument
		rpcError   error
		wantLegacy bool
		wantError  bool
	}{
		{"older server", nil, unsupported, true, false},
		{"authored YAML", definition, nil, false, false},
		{"permission denied", nil, status.Error(codes.PermissionDenied, "denied"), false, true},
		{"empty reply", nil, nil, false, true},
		{"ambiguous reply", definition, unsupported, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rpc := &profileCompatRPC{definition: test.definition, err: test.rpcError}
			got, legacy, err := GetProfileDefinitionOrLegacy(context.Background(), "profile", &console.SliverClient{Rpc: rpc})
			if legacy != test.wantLegacy || (err != nil) != test.wantError {
				t.Fatalf("definition=%v legacy=%t err=%v", got, legacy, err)
			}
			if err == nil && got != test.definition {
				t.Fatalf("definition=%v, want %v", got, test.definition)
			}
		})
	}
}
