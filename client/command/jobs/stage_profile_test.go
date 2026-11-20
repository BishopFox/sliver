package jobs

import (
	"bytes"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

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
