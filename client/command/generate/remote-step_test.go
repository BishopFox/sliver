package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type remoteWithoutLocalCompilerRPC struct {
	rpcpb.SliverRPCClient
	compilerCalls int
	request       *clientpb.BuildRequest
}

func (rpc *remoteWithoutLocalCompilerRPC) GetCompiler(context.Context, *commonpb.Empty, ...grpc.CallOption) (*clientpb.Compiler, error) {
	rpc.compilerCalls++
	return &clientpb.Compiler{}, nil
}

func (rpc *remoteWithoutLocalCompilerRPC) ExecuteBuild(_ context.Context, request *clientpb.BuildRequest, _ ...grpc.CallOption) (*clientpb.BuildRecord, error) {
	rpc.request = request
	return nil, errors.New("stopped after recording remote build request")
}

func TestRemoteBuildSkipsLocalCompilerCheck(t *testing.T) {
	rpc := &remoteWithoutLocalCompilerRPC{}
	con := console.NewConsole(false)
	if err := console.StartClient(con, rpc, nil, nil, nil, nil, false, ""); err != nil {
		t.Fatal(err)
	}
	rpc.compilerCalls = 0
	cmd := cleanupCommand(t, false)
	if err := cmd.Flags().Parse([]string{"--mtls", "example.test", "--os", "darwin", "--arch", "arm64",
		"--format", "shared", "--exports", "Run", "-E"}); err != nil {
		t.Fatal(err)
	}
	GenerateCmd(cmd, con, nil)
	if rpc.compilerCalls != 0 {
		t.Fatalf("remote build queried the local compiler %d times", rpc.compilerCalls)
	}
	if rpc.request == nil || rpc.request.GetProvider().GetMode() != "remote" ||
		rpc.request.GetTarget().GetGOOS() != "darwin" || rpc.request.GetTarget().GetGOARCH() != "arm64" {
		t.Fatalf("remote build did not reach assigned-step planning: %+v", rpc.request)
	}
}

func TestExternalBuilderFlagUsesAssignedStepPlan(t *testing.T) {
	cmd := previewTestCommand(t, false)
	if err := cmd.Flags().Parse([]string{"--mtls", "example.test", "-E"}); err != nil {
		t.Fatal(err)
	}
	if !remoteStepRequested(cmd) {
		t.Fatal("-E did not select the assigned-step remote route")
	}
	request, err := planRequestFromFlags(cmd, false)
	if err != nil {
		t.Fatal(err)
	}
	if request.GetProvider().GetMode() != "remote" || request.GetCore().GetName() != "builtin/sliver" {
		t.Fatalf("-E did not produce a remote core build request: %+v", request)
	}
}

func TestRemoteBuildCarriesSelectedShellcodeEncoder(t *testing.T) {
	rpc := &remoteWithoutLocalCompilerRPC{}
	con := console.NewConsole(false)
	if err := console.StartClient(con, rpc, nil, nil, nil, nil, false, ""); err != nil {
		t.Fatal(err)
	}
	cmd := previewTestCommand(t, false)
	if err := cmd.Flags().Parse([]string{"--mtls", "example.test", "--os", "linux", "--arch", "amd64",
		"--format", "shellcode", "-E"}); err != nil {
		t.Fatal(err)
	}
	preview, err := planRequestFromFlags(cmd, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range preview.GetTransforms() {
		if step.GetSpec().GetName() == "builtin/xor-dynamic" {
			t.Fatal("flag-only preview included an interactive encoder choice")
		}
	}
	_, err = remoteStepBuild(cmd, "payload", &clientpb.ImplantConfig{
		GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_SHELLCODE,
		ShellcodeEncoder: clientpb.ShellcodeEncoder_XOR_DYNAMIC,
	}, nil, t.TempDir(), con)
	if err == nil || rpc.request == nil {
		t.Fatalf("remote request was not captured: request=%+v err=%v", rpc.request, err)
	}
	if cmd.Flags().Changed("shellcode-encoder") {
		t.Fatal("actual build mutated the preview flags")
	}
	transforms := rpc.request.GetTransforms()
	if len(transforms) < 2 || transforms[len(transforms)-1].GetSpec().GetName() != "builtin/xor-dynamic" {
		t.Fatalf("interactive encoder choice missing from remote build: %+v", transforms)
	}
}

func TestVerifiedRemoteStepArtifactRejectsMismatchedHandoff(t *testing.T) {
	data := []byte("final artifact bytes")
	hash := sha256.Sum256(data)
	checksum := hex.EncodeToString(hash[:])
	artifactType := &clientpb.BuildArtifactType{Kind: "executable", Container: "elf", GOOS: "linux", GOARCH: "amd64"}
	record := &clientpb.BuildRecord{FinalArtifact: &clientpb.BuildArtifactMetadata{
		Type: artifactType, FileName: "payload", SizeBytes: uint64(len(data)), SHA256: checksum,
	}}
	response := &clientpb.BuildArtifactResponse{
		File: &commonpb.File{Name: "payload", Data: data},
		Metadata: &clientpb.BuildArtifactMetadata{
			Type:     proto.Clone(artifactType).(*clientpb.BuildArtifactType),
			FileName: "payload", SizeBytes: uint64(len(data)), SHA256: checksum,
		},
	}
	if got, err := verifiedRemoteStepArtifact(record, response); err != nil || got != response.File {
		t.Fatalf("valid remote artifact rejected: %v", err)
	}
	for _, test := range []struct {
		name   string
		change func(*clientpb.BuildRecord, *clientpb.BuildArtifactResponse)
	}{
		{"record filename", func(record *clientpb.BuildRecord, _ *clientpb.BuildArtifactResponse) {
			record.FinalArtifact.FileName = "other"
		}},
		{"metadata filename", func(_ *clientpb.BuildRecord, response *clientpb.BuildArtifactResponse) {
			response.Metadata.FileName = "other"
		}},
		{"unsafe filename", func(_ *clientpb.BuildRecord, response *clientpb.BuildArtifactResponse) {
			response.File.Name = "../payload"
		}},
		{"control filename", func(_ *clientpb.BuildRecord, response *clientpb.BuildArtifactResponse) {
			response.File.Name = "payload\nnext"
		}},
		{"artifact type", func(_ *clientpb.BuildRecord, response *clientpb.BuildArtifactResponse) {
			response.Metadata.Type.GOARCH = "arm64"
		}},
		{"artifact bytes", func(_ *clientpb.BuildRecord, response *clientpb.BuildArtifactResponse) {
			response.File.Data = []byte("changed")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutatedRecord := proto.Clone(record).(*clientpb.BuildRecord)
			mutatedResponse := proto.Clone(response).(*clientpb.BuildArtifactResponse)
			test.change(mutatedRecord, mutatedResponse)
			if _, err := verifiedRemoteStepArtifact(mutatedRecord, mutatedResponse); err == nil {
				t.Fatal("mismatched remote artifact was accepted")
			}
		})
	}
}
