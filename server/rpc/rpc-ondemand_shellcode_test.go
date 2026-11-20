package rpc

import (
	"errors"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestOnDemandShellcodeConfigAndName(t *testing.T) {
	original := &clientpb.ImplantConfig{
		ID: "saved-config", GOOS: "windows", GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE, IsService: true, IsSharedLib: true,
		ShellcodeEncoder: clientpb.ShellcodeEncoder_XOR, SGNEnabled: true,
		ImplantBuilds: []*clientpb.ImplantBuild{{Name: "old-build"}},
		C2:            []*clientpb.ImplantC2{{URL: "mtls://127.0.0.1:8888"}},
	}
	before := proto.Clone(original)
	for _, obfuscate := range []bool{false, true} {
		config, err := onDemandShellcodeConfig(original, obfuscate)
		if err != nil {
			t.Fatal(err)
		}
		if config.ID != "" || len(config.ImplantBuilds) != 0 || config.Format != clientpb.OutputFormat_SHELLCODE ||
			!config.IsShellcode || config.IsSharedLib || config.IsService ||
			config.ObfuscateSymbols != obfuscate || config.ShellcodeEncoder != clientpb.ShellcodeEncoder_NONE ||
			config.SGNEnabled || len(config.Exports) != 1 || config.Exports[0] != "StartW" ||
			len(config.C2) != 1 || config.C2[0].URL != original.C2[0].URL {
			t.Fatalf("on-demand config lost its shellcode or C2 requirements: %+v", config)
		}
	}
	if !proto.Equal(original, before) {
		t.Fatal("on-demand generation mutated the active session config")
	}
	if _, err := onDemandShellcodeConfig(nil, true); err == nil {
		t.Fatal("missing active config was accepted")
	}

	name, err := uniqueTaskBuildName("existing-session", func(string) (bool, error) { return false, nil })
	if err != nil || name == "existing-session" || !strings.HasPrefix(name, "existing-session_") || util.AllowedName(name) != nil {
		t.Fatalf("on-demand build did not get a unique safe name: %q, %v", name, err)
	}
	second, err := uniqueTaskBuildName("existing-session", func(string) (bool, error) { return false, nil })
	if err != nil || second == name || !strings.HasPrefix(second, "existing-session_") || util.AllowedName(second) != nil {
		t.Fatalf("repeated on-demand allocation reused a name: first=%q second=%q err=%v", name, second, err)
	}
	lookups := 0
	name, err = uniqueTaskBuildName("existing-session", func(string) (bool, error) {
		lookups++
		return lookups == 1, nil
	})
	if err != nil || lookups != 2 || name == "existing-session" ||
		!strings.HasPrefix(name, "existing-session_") || util.AllowedName(name) != nil {
		t.Fatalf("colliding name was not safely replaced: %q, lookups=%d, err=%v", name, lookups, err)
	}
	if _, err := uniqueTaskBuildName("../unsafe", func(string) (bool, error) { return false, nil }); err == nil {
		t.Fatal("unsafe name was accepted")
	}
	lookupFailure := errors.New("name lookup failed")
	if _, err := uniqueTaskBuildName("safe", func(string) (bool, error) { return false, lookupFailure }); !errors.Is(err, lookupFailure) {
		t.Fatalf("name lookup error = %v", err)
	}
}

func TestGetSystemRejectsInvalidNamedBuild(t *testing.T) {
	session := core.NewSession(core.NewImplantConnection("test", "n/a"))
	core.Sessions.Add(session)
	t.Cleanup(func() { core.Sessions.Remove(session.ID) })
	_, err := (&Server{}).GetSystem(t.Context(), &clientpb.GetSystemReq{
		Name: "../unsafe", Config: &clientpb.ImplantConfig{},
		Request: &commonpb.Request{SessionID: session.ID},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unsafe named GetSystem build was accepted: %v", err)
	}
}
