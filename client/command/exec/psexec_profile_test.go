package exec

import (
	"bytes"
	"context"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type psexecLegacyProfileRPC struct {
	rpcpb.SliverRPCClient
	executeErr    error
	profileCalls  int
	generateCalls int
	saveCalls     int
}

func (rpc *psexecLegacyProfileRPC) ExecuteProfileBuild(_ context.Context, _ *clientpb.BuildProfileExecuteRequest, _ ...grpc.CallOption) (*clientpb.BuildRecord, error) {
	return nil, rpc.executeErr
}

func (rpc *psexecLegacyProfileRPC) ImplantProfiles(_ context.Context, _ *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.ImplantProfiles, error) {
	rpc.profileCalls++
	return &clientpb.ImplantProfiles{Profiles: []*clientpb.ImplantProfile{{
		Name: "legacy", Config: &clientpb.ImplantConfig{Format: clientpb.OutputFormat_EXECUTABLE},
	}}}, nil
}

func (rpc *psexecLegacyProfileRPC) Generate(_ context.Context, _ *clientpb.GenerateReq, _ ...grpc.CallOption) (*clientpb.Generate, error) {
	rpc.generateCalls++
	return &clientpb.Generate{File: &commonpb.File{Data: []byte("legacy service")}}, nil
}

func (rpc *psexecLegacyProfileRPC) SaveImplantProfile(_ context.Context, profile *clientpb.ImplantProfile, _ ...grpc.CallOption) (*clientpb.ImplantProfile, error) {
	rpc.saveCalls++
	return profile, nil
}

func TestPsExecProfileFinalArtifactPolicy(t *testing.T) {
	for _, test := range []struct {
		name     string
		artifact *clientpb.BuildArtifactType
		allowed  bool
	}{
		{"service", &clientpb.BuildArtifactType{Kind: "service-executable", Container: "pe", GOOS: "windows", GOARCH: "amd64"}, true},
		{"ordinary executable", &clientpb.BuildArtifactType{Kind: "executable", Container: "pe", GOOS: "windows", GOARCH: "amd64"}, false},
		{"shellcode", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64"}, false},
		{"other OS", &clientpb.BuildArtifactType{Kind: "service-executable", Container: "elf", GOOS: "linux", GOARCH: "amd64"}, false},
		{"no architecture", &clientpb.BuildArtifactType{Kind: "service-executable", Container: "pe", GOOS: "windows"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := psexecProfileArtifact(test.artifact); got != test.allowed {
				t.Fatalf("artifact %+v: allowed=%t, want %t", test.artifact, got, test.allowed)
			}
		})
	}
}

func TestPsExecLegacyProfileFallback(t *testing.T) {
	unsupported := status.Error(codes.Unimplemented, "profile builds unavailable")
	for _, test := range []struct {
		name         string
		definition   *clientpb.BuildProfileDocument
		executeError error
		wantLegacy   bool
	}{
		{"older server", nil, nil, true},
		{"migrated profile", &clientpb.BuildProfileDocument{Migrated: true}, unsupported, true},
		{"authored YAML", &clientpb.BuildProfileDocument{Migrated: false}, unsupported, false},
		{"migration error", &clientpb.BuildProfileDocument{Migrated: true, MigrationError: "unsupported"}, unsupported, false},
		{"other build failure", &clientpb.BuildProfileDocument{Migrated: true}, status.Error(codes.PermissionDenied, "denied"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			rpc := &psexecLegacyProfileRPC{executeErr: test.executeError}
			got, err := psexecProfileBinary("legacy", test.definition, &console.SliverClient{Rpc: rpc})
			if test.wantLegacy {
				if err != nil || !bytes.Equal(got, []byte("legacy service")) {
					t.Fatalf("legacy profile build = %q, %v", got, err)
				}
				if rpc.profileCalls != 1 || rpc.generateCalls != 1 || rpc.saveCalls != 1 {
					t.Fatalf("legacy RPC calls: profiles=%d generate=%d save=%d", rpc.profileCalls, rpc.generateCalls, rpc.saveCalls)
				}
			} else {
				if err == nil || rpc.profileCalls != 0 || rpc.generateCalls != 0 || rpc.saveCalls != 0 {
					t.Fatalf("unexpected legacy fallback: result=%q err=%v calls=%+v", got, err, rpc)
				}
			}
		})
	}
}
