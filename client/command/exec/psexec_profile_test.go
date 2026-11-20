package exec

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

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
