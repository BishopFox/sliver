package generate

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/protobuf/proto"
)

func TestSelectPinnedExternalBuilder(t *testing.T) {
	builders := []*clientpb.Builder{{Name: "alpha"}, {Name: "bravo"}}
	selected, err := selectPinnedExternalBuilder(builders, "bravo")
	if err != nil || selected.Name != "bravo" {
		t.Fatalf("pinned builder selection: %+v, %v", selected, err)
	}
	if _, err := selectPinnedExternalBuilder(builders, "offline"); err == nil {
		t.Fatal("unknown pinned builder was accepted")
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
