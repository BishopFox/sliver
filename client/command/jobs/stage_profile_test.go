package jobs

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

type stageLegacyProfileRPC struct {
	rpcpb.SliverRPCClient
	executeErr    error
	profileCalls  int
	generateCalls int
	saveCalls     int
}

func (rpc *stageLegacyProfileRPC) ExecuteProfileBuild(_ context.Context, _ *clientpb.BuildProfileExecuteRequest, _ ...grpc.CallOption) (*clientpb.BuildRecord, error) {
	return nil, rpc.executeErr
}

func (rpc *stageLegacyProfileRPC) ImplantProfiles(_ context.Context, _ *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.ImplantProfiles, error) {
	rpc.profileCalls++
	return &clientpb.ImplantProfiles{Profiles: []*clientpb.ImplantProfile{{
		Name: "legacy", Config: &clientpb.ImplantConfig{Format: clientpb.OutputFormat_SHELLCODE},
	}}}, nil
}

func (rpc *stageLegacyProfileRPC) Generate(_ context.Context, _ *clientpb.GenerateReq, _ ...grpc.CallOption) (*clientpb.Generate, error) {
	rpc.generateCalls++
	return &clientpb.Generate{File: &commonpb.File{Data: []byte("legacy shellcode")}}, nil
}

func (rpc *stageLegacyProfileRPC) SaveImplantProfile(_ context.Context, profile *clientpb.ImplantProfile, _ ...grpc.CallOption) (*clientpb.ImplantProfile, error) {
	rpc.saveCalls++
	return profile, nil
}

func TestStageListenerProfileFinalArtifactPolicy(t *testing.T) {
	for _, test := range []struct {
		name     string
		artifact *clientpb.BuildArtifactType
		prepared bool
		allowed  bool
	}{
		{"shellcode", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64"}, false, true},
		{"prepared stage", &clientpb.BuildArtifactType{Kind: "stage", Container: "raw", GOOS: "windows", GOARCH: "amd64"}, true, true},
		{"service executable", &clientpb.BuildArtifactType{Kind: "service-executable", Container: "pe", GOOS: "windows", GOARCH: "amd64"}, false, false},
		{"archive bundle", &clientpb.BuildArtifactType{Kind: "c-archive-bundle", Container: "zip", GOOS: "windows", GOARCH: "amd64"}, false, false},
		{"wrong container", &clientpb.BuildArtifactType{Kind: "stage", Container: "zip", GOOS: "windows", GOARCH: "amd64"}, false, false},
		{"no target", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw"}, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepared, allowed := stageListenerProfileArtifact(test.artifact)
			if prepared != test.prepared || allowed != test.allowed {
				t.Fatalf("artifact %+v: prepared=%t allowed=%t, want %t %t", test.artifact, prepared, allowed, test.prepared, test.allowed)
			}
		})
	}
}

func TestTCPStageListenerFramesFinalPayloadBytes(t *testing.T) {
	payload := []byte{0x4d, 0x5a, 0x90}
	got := prependPayloadSize(payload)
	want := []byte{3, 0, 0, 0, 0x4d, 0x5a, 0x90}
	if !bytes.Equal(got, want) {
		t.Fatalf("framed payload = %x, want %x", got, want)
	}
}

func TestStageListenerLegacyProfileFallback(t *testing.T) {
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
			rpc := &stageLegacyProfileRPC{executeErr: test.executeError}
			got, err := stageListenerProfilePayload("legacy", test.definition, &console.SliverClient{Rpc: rpc})
			if test.wantLegacy {
				if err != nil || !bytes.Equal(got, []byte("legacy shellcode")) {
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
