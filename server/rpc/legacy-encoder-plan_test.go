package rpc

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestLegacyBuildRequestAppendsShellcodeEncoder(t *testing.T) {
	for _, test := range []struct {
		name    string
		encoder clientpb.ShellcodeEncoder
		want    string
	}{
		{"sgn", clientpb.ShellcodeEncoder_SHIKATA_GA_NAI, "builtin/shikata-ga-nai"},
		{"xor", clientpb.ShellcodeEncoder_XOR, "builtin/xor"},
		{"dynamic-xor", clientpb.ShellcodeEncoder_XOR_DYNAMIC, "builtin/xor-dynamic"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := &clientpb.ImplantConfig{
				GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_SHELLCODE,
				ShellcodeEncoder: test.encoder,
			}
			request, err := legacyBuildRequest("encoder-test", config)
			if err != nil {
				t.Fatal(err)
			}
			if len(request.Transforms) != 2 || request.Transforms[0].Spec.Name != "builtin/donut" ||
				request.Transforms[1].Spec.Name != test.want {
				t.Fatalf("converter and encoder must be ordered in one plan: %+v", request.Transforms)
			}
		})
	}
}

func TestLegacyBuildRequestSGNFallback(t *testing.T) {
	config := &clientpb.ImplantConfig{
		GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_SHELLCODE,
		SGNEnabled: true,
	}
	request, err := legacyBuildRequest("legacy-sgn", config)
	if err != nil || len(request.Transforms) != 2 || request.Transforms[1].Spec.Name != "builtin/shikata-ga-nai" {
		t.Fatalf("legacy SGN flag should select typed transform: steps=%+v err=%v", request.Transforms, err)
	}
	config.GOARCH = "arm64"
	request, err = legacyBuildRequest("legacy-sgn", config)
	if err != nil || len(request.Transforms) != 1 {
		t.Fatalf("unsupported legacy SGN architecture should keep original skip behavior: steps=%+v err=%v", request.Transforms, err)
	}
}

func TestLegacyBuildRequestRejectsUnknownShellcodeEncoder(t *testing.T) {
	config := &clientpb.ImplantConfig{
		GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_SHELLCODE,
		ShellcodeEncoder: clientpb.ShellcodeEncoder(99),
	}
	if _, err := legacyBuildRequest("unknown-encoder", config); err == nil {
		t.Fatal("unknown shellcode encoder enum should fail before compilation")
	}
}
