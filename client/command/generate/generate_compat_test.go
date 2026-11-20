package generate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type generateCompatibilityRPC struct {
	rpcpb.SliverRPCClient
	generateReq    *clientpb.GenerateReq
	generateName   string
	regenerateName string
	spoofReq       *clientpb.GenerateSpoofMetadataReq
	encodeReq      *clientpb.ShellcodeEncodeReq
	regenerate     bool
	record         *clientpb.BuildRecord
	recordErr      error
	spoofErr       error
	encodeErr      error
}

func (rpc *generateCompatibilityRPC) Generate(_ context.Context, req *clientpb.GenerateReq, _ ...grpc.CallOption) (*clientpb.Generate, error) {
	rpc.generateReq = req
	name := rpc.generateName
	if name == "" {
		name = "payload.exe"
	}
	return &clientpb.Generate{File: &commonpb.File{Name: name, Data: []byte("raw")},
		ImplantName: "payload", ImplantBuildID: "build-id"}, nil
}

func (rpc *generateCompatibilityRPC) GetBuildRecord(_ context.Context, _ *clientpb.BuildRecordRef, _ ...grpc.CallOption) (*clientpb.BuildRecord, error) {
	if rpc.recordErr != nil {
		return nil, rpc.recordErr
	}
	return rpc.record, nil
}

func (rpc *generateCompatibilityRPC) GenerateSpoofMetadata(_ context.Context, req *clientpb.GenerateSpoofMetadataReq, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	rpc.spoofReq = req
	if rpc.spoofErr != nil {
		return nil, rpc.spoofErr
	}
	return &commonpb.Empty{}, nil
}

func (rpc *generateCompatibilityRPC) Regenerate(_ context.Context, _ *clientpb.RegenerateReq, _ ...grpc.CallOption) (*clientpb.Generate, error) {
	rpc.regenerate = true
	name := rpc.regenerateName
	if name == "" {
		name = "payload.exe"
	}
	return &clientpb.Generate{File: &commonpb.File{Name: name, Data: []byte("spoofed")}}, nil
}

func (rpc *generateCompatibilityRPC) ShellcodeEncoderMap(_ context.Context, _ *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.ShellcodeEncoderMap, error) {
	return &clientpb.ShellcodeEncoderMap{Encoders: map[string]*clientpb.ShellcodeEncoderArchMap{
		"amd64": {Encoders: map[string]clientpb.ShellcodeEncoder{"xor": clientpb.ShellcodeEncoder_XOR}},
	}}, nil
}

func (rpc *generateCompatibilityRPC) ShellcodeEncoder(_ context.Context, req *clientpb.ShellcodeEncodeReq, _ ...grpc.CallOption) (*clientpb.ShellcodeEncode, error) {
	rpc.encodeReq = req
	if rpc.encodeErr != nil {
		return nil, rpc.encodeErr
	}
	return &clientpb.ShellcodeEncode{Data: []byte("encoded")}, nil
}

func generateCompatibilityConsole(t *testing.T, rpc *generateCompatibilityRPC) *console.SliverClient {
	t.Helper()
	con := console.NewConsole(false)
	if err := console.StartClient(con, rpc, nil, nil, nil, nil, false, ""); err != nil {
		t.Fatal(err)
	}
	return con
}

func TestGenerateOldServerCompatibilityMetadata(t *testing.T) {
	rpc := &generateCompatibilityRPC{recordErr: status.Error(codes.Unimplemented, "older server")}
	con := generateCompatibilityConsole(t, rpc)
	metadata := &clientpb.SpoofMetadataConfig{PE: &clientpb.PESpoofMetadataConfig{
		Source: &clientpb.SpoofMetadataFile{Name: "donor.exe", Data: []byte("donor")},
	}}
	path := filepath.Join(t.TempDir(), "payload.exe")
	file, err := compile("payload", &clientpb.ImplantConfig{GOOS: "windows", GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE}, metadata, path, con)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, []byte("spoofed")) || !bytes.Equal(file.Data, saved) {
		t.Fatalf("old server artifact was not spoofed: file=%+v saved=%q err=%v", file, saved, err)
	}
	if rpc.generateReq.GetSpoofMetadata() == nil || rpc.spoofReq.GetImplantBuildID() != "build-id" || !rpc.regenerate {
		t.Fatalf("old server metadata compatibility calls were skipped: generate=%+v spoof=%+v regenerate=%t",
			rpc.generateReq, rpc.spoofReq, rpc.regenerate)
	}
}

func TestGenerateOldServerCompatibilityEncoder(t *testing.T) {
	rpc := &generateCompatibilityRPC{recordErr: status.Error(codes.Unimplemented, "older server")}
	con := generateCompatibilityConsole(t, rpc)
	path := filepath.Join(t.TempDir(), "payload.bin")
	file, err := compile("payload", &clientpb.ImplantConfig{GOOS: "windows", GOARCH: "amd64",
		Format: clientpb.OutputFormat_SHELLCODE, IsShellcode: true,
		ShellcodeEncoder: clientpb.ShellcodeEncoder_XOR}, nil, path, con)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, []byte("encoded")) || !bytes.Equal(file.Data, saved) {
		t.Fatalf("old server artifact was not encoded: file=%+v saved=%q err=%v", file, saved, err)
	}
	if rpc.encodeReq == nil || rpc.encodeReq.Encoder != clientpb.ShellcodeEncoder_XOR ||
		!bytes.Equal(rpc.encodeReq.Data, []byte("raw")) {
		t.Fatalf("old server shellcode encoding was skipped: %+v", rpc.encodeReq)
	}
}

func TestGenerateRejectsUnsafeServerArtifactName(t *testing.T) {
	for _, test := range []struct {
		name           string
		generateName   string
		regenerateName string
		metadata       *clientpb.SpoofMetadataConfig
	}{
		{"ordinary Generate", "../escape.exe", "", nil},
		{"legacy Regenerate", "payload.exe", "../escape.exe", &clientpb.SpoofMetadataConfig{
			PE: &clientpb.PESpoofMetadataConfig{Source: &clientpb.SpoofMetadataFile{Name: "donor.exe", Data: []byte("donor")}},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rpc := &generateCompatibilityRPC{recordErr: status.Error(codes.Unimplemented, "older server"),
				generateName: test.generateName, regenerateName: test.regenerateName}
			con := generateCompatibilityConsole(t, rpc)
			root := t.TempDir()
			saveDir := filepath.Join(root, "payloads")
			if err := os.Mkdir(saveDir, 0o700); err != nil {
				t.Fatal(err)
			}
			_, err := compile("payload", &clientpb.ImplantConfig{GOOS: "windows", GOARCH: "amd64",
				Format: clientpb.OutputFormat_EXECUTABLE}, test.metadata, saveDir, con)
			if err == nil {
				t.Fatal("unsafe server filename was accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "escape.exe")); !os.IsNotExist(err) {
				t.Fatalf("server filename escaped save directory: %v", err)
			}
		})
	}
}

func TestGenerateOldServerCompatibilityFailureDoesNotSave(t *testing.T) {
	for _, tc := range []struct {
		name      string
		metadata  bool
		spoofErr  error
		encodeErr error
		want      string
	}{
		{name: "spoof failed", metadata: true, spoofErr: errors.New("spoof rejected"), want: "legacy metadata spoofing failed"},
		{name: "encoder failed", encodeErr: errors.New("encoder rejected"), want: "legacy shellcode encoding failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rpc := &generateCompatibilityRPC{recordErr: status.Error(codes.Unimplemented, "older server"),
				spoofErr: tc.spoofErr, encodeErr: tc.encodeErr}
			con := generateCompatibilityConsole(t, rpc)
			config := &clientpb.ImplantConfig{GOOS: "windows", GOARCH: "amd64",
				Format: clientpb.OutputFormat_SHELLCODE, IsShellcode: true,
				ShellcodeEncoder: clientpb.ShellcodeEncoder_XOR}
			var metadata *clientpb.SpoofMetadataConfig
			if tc.metadata {
				config.Format = clientpb.OutputFormat_EXECUTABLE
				config.IsShellcode = false
				metadata = &clientpb.SpoofMetadataConfig{PE: &clientpb.PESpoofMetadataConfig{
					Source: &clientpb.SpoofMetadataFile{Name: "donor.exe", Data: []byte("donor")},
				}}
			}
			path := filepath.Join(t.TempDir(), "payload.exe")
			_, err := compile("payload", config, metadata, path, con)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got error %v, want %q", err, tc.want)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("failed transform left an output file: %v", err)
			}
		})
	}
}

func TestGenerateOldServerCompatibilityRejectsUnverifiedModernOutput(t *testing.T) {
	data := []byte("raw")
	hash := sha256.Sum256(data)
	rpc := &generateCompatibilityRPC{record: &clientpb.BuildRecord{
		ID: "build-id", State: clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED,
		Plan: &clientpb.BuildPlan{Steps: []*clientpb.BuildPlanStep{{Spec: &clientpb.BuildSpecRef{Name: "builtin/sliver"}}}},
		FinalArtifact: &clientpb.BuildArtifactMetadata{Type: &clientpb.BuildArtifactType{Kind: "executable"},
			FileName: "payload.exe", SizeBytes: uint64(len(data)), SHA256: hex.EncodeToString(hash[:])},
	}}
	con := generateCompatibilityConsole(t, rpc)
	metadata := &clientpb.SpoofMetadataConfig{PE: &clientpb.PESpoofMetadataConfig{
		Source: &clientpb.SpoofMetadataFile{Name: "donor.exe", Data: []byte("donor")},
	}}
	path := filepath.Join(t.TempDir(), "payload.exe")
	_, err := compile("payload", &clientpb.ImplantConfig{GOOS: "windows", GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE}, metadata, path, con)
	if err == nil || !strings.Contains(err.Error(), "omits requested generation transform") {
		t.Fatalf("unspoofed modern artifact accepted: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unverified artifact was saved: %v", err)
	}
	if rpc.spoofReq != nil || rpc.regenerate {
		t.Fatal("modern server received legacy post-processing calls")
	}
}
