package rpc

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestServerProfileConsumerFinalArtifactPolicy(t *testing.T) {
	for _, test := range []struct {
		name       string
		artifact   *clientpb.BuildArtifactType
		kind       string
		container  string
		goos       string
		compatible bool
	}{
		{"backdoor shellcode", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64"}, "shellcode", "raw", "windows", true},
		{"backdoor executable", &clientpb.BuildArtifactType{Kind: "executable", Container: "pe", GOOS: "windows", GOARCH: "amd64"}, "shellcode", "raw", "windows", false},
		{"hijack DLL", &clientpb.BuildArtifactType{Kind: "shared-library", Container: "pe", GOOS: "windows", GOARCH: "amd64"}, "shared-library", "pe", "windows", true},
		{"hijack ELF", &clientpb.BuildArtifactType{Kind: "shared-library", Container: "elf", GOOS: "linux", GOARCH: "amd64"}, "shared-library", "pe", "windows", false},
		{"missing architecture", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows"}, "shellcode", "raw", "windows", false},
		{"nil", nil, "shellcode", "raw", "windows", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := profileArtifactMatches(test.artifact, test.kind, test.container, test.goos); got != test.compatible {
				t.Fatalf("artifact %+v: compatible=%t, want %t", test.artifact, got, test.compatible)
			}
		})
	}
}

func TestServerProfileConsumerRejectsChangedFinalArtifact(t *testing.T) {
	definition := &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64", EntrySymbols: []string{"Run"}}
	for _, test := range []struct {
		name      string
		completed *clientpb.BuildArtifactType
		unchanged bool
	}{
		{"same type and target", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64", EntrySymbols: []string{"Run"}}, true},
		{"changed kind", &clientpb.BuildArtifactType{Kind: "stage", Container: "raw", GOOS: "windows", GOARCH: "amd64"}, false},
		{"changed container", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "pe", GOOS: "windows", GOARCH: "amd64"}, false},
		{"changed operating system", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "linux", GOARCH: "amd64"}, false},
		{"changed architecture", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "arm64"}, false},
		{"changed Go build mode", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64", GoBuildMode: "pie", EntrySymbols: []string{"Run"}}, false},
		{"changed entry symbol", &clientpb.BuildArtifactType{Kind: "shellcode", Container: "raw", GOOS: "windows", GOARCH: "amd64", EntrySymbols: []string{"RunOther"}}, false},
		{"missing final type", nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := sameConsumerArtifact(definition, test.completed); got != test.unchanged {
				t.Fatalf("definition %+v, completed %+v: unchanged=%t, want %t", definition, test.completed, got, test.unchanged)
			}
		})
	}
}
