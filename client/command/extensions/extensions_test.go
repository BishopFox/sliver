package extensions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

const (
	sample1 = `{
	"name": "test1",
	"version": "1.0.0",
	"extension_author": "test",
	"original_author": "test",
	"repo_url": "https://example.com/",
	"commands":[
	{
		"command_name": "test1",
		"help": "some help",
		"files": [
			{
				"os": "windows",
				"arch": "amd64",
				"path": "foo/test1.dll"
			}
		]
	}
	]
}`

	sample2 = `{
	"name": "test2",
	"command_name": "test2",
	"help": "some help",
	"files": [
		{
			"os": "windows",
			"arch": "amd64",
			"path": "../../../../foo/test1.dll"
		}
	]
}`
	sample3 = `{
	"name": "test3",
	"version": "1.0.0",
	"extension_author": "test",
	"original_author": "test",
	"repo_url": "https://example.com/",
	"commands": [
		{
			"command_name": "test3",
			"help": "some help",
			"files": [
				{
					"os": "windows",
					"arch": "amd64",
					"path": "foo/test1.dll"
				}
			]
		}
	]
}`

	multicmd = `{
		"name": "example-multientry",
		"version": "0.0.0",
		"extension_author": "cs",
		"original_author": "cs",
		"repo_url": "no",
		"commands": [
			{
				"command_name": "startw",
				"help": "startw",
				"entrypoint": "StartW",
				"files": [
					{
						"os": "windows",
						"arch": "amd64",
						"path": "ex.dll"
					}
				]
			},
			{
				"command_name": "Test2",
				"help": "startw",
				"entrypoint": "Test2",
				"files": [
					{
						"os": "windows",
						"arch": "amd64",
						"path": "ex.dll"
					}
				]
			}
		]
	}`
)

type registerExtensionTestRPC struct {
	rpcpb.SliverRPCClient
	register func(context.Context, *sliverpb.RegisterExtensionReq, ...grpc.CallOption) (*sliverpb.RegisterExtension, error)
}

func (f *registerExtensionTestRPC) RegisterExtension(ctx context.Context, req *sliverpb.RegisterExtensionReq, opts ...grpc.CallOption) (*sliverpb.RegisterExtension, error) {
	return f.register(ctx, req, opts...)
}

func newExtensionTestConsole(t *testing.T, rpc rpcpb.SliverRPCClient, session *clientpb.Session, beacon *clientpb.Beacon) *console.SliverClient {
	t.Helper()

	con := console.NewConsole(false)
	con.IsCLI = true
	con.Rpc = rpc
	con.ActiveTarget.Set(session, beacon)
	return con
}

func TestParseExtensionManifest(t *testing.T) {
	expectedPath := util.ResolvePath("foo/test1.dll")

	extManifest, err := ParseExtensionManifest([]byte(sample1))
	if err != nil {
		t.Fatalf("Error parsing extension manifest: %s", err)
	}
	if extManifest.Name != "test1" {
		t.Errorf("Expected extension name 'test1', got '%s'", extManifest.Name)
	}

	if extManifest.Version != "1.0.0" {
		t.Errorf("Expected extension version '1.0.0', got '%s'", extManifest.Version)
	}
	if extManifest.ExtensionAuthor != "test" {
		t.Errorf("Expected extension author 'test', got '%s'", extManifest.ExtensionAuthor)
	}
	if extManifest.OriginalAuthor != "test" {
		t.Errorf("Expected original author 'test', got '%s'", extManifest.OriginalAuthor)
	}
	if extManifest.RepoURL != "https://example.com/" {
		t.Errorf("Expected repo URL 'https://example.com/', got '%s'", extManifest.RepoURL)
	}
	for _, extCmd := range extManifest.ExtCommand { //should only be a single manfiest here, so should pass
		if extCmd.CommandName != "test1" {
			t.Errorf("Expected extension command name 'test1', got '%s'", extCmd.CommandName)
		}
		if extCmd.Help != "some help" {
			t.Errorf("Expected help 'some help', got '%s'", extCmd.Help)
		}
		if len(extCmd.Files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(extCmd.Files))
		}
		if extCmd.Files[0].OS != "windows" {
			t.Errorf("Expected OS 'windows', got '%s'", extCmd.Files[0].OS)
		}
		if extCmd.Files[0].Arch != "amd64" {
			t.Errorf("Expected Arch 'amd64', got '%s'", extCmd.Files[0].Arch)
		}
		if extCmd.Files[0].Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, extCmd.Files[0].Path)
		}
	}

	mextManifest2, err := ParseExtensionManifest([]byte(sample2)) //checking old manifests work good too
	if err != nil {
		t.Fatalf("Error parsing extension manifest (2): %s", err)
	}
	if mextManifest2.Name != "test2" {
		t.Errorf("Expected extension name 'test2', got '%s'", mextManifest2.Name)
	}
	for _, extManifest2 := range mextManifest2.ExtCommand {
		if extManifest2.CommandName != "test2" {
			t.Errorf("Expected extension command name 'test2', got '%s'", extManifest2.CommandName)
		}
		if extManifest2.Help != "some help" {
			t.Errorf("Expected help 'some help', got '%s'", extManifest2.Help)
		}
		if len(extManifest2.Files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(extManifest2.Files))
		}
		if extManifest2.Files[0].OS != "windows" {
			t.Errorf("Expected OS 'windows', got '%s'", extManifest2.Files[0].OS)
		}
		if extManifest2.Files[0].Arch != "amd64" {
			t.Errorf("Expected Arch 'amd64', got '%s'", extManifest2.Files[0].Arch)
		}
		if extManifest2.Files[0].Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, extManifest2.Files[0].Path)
		}
	}

}

func TestConvertOldManifestPreservesInit(t *testing.T) {
	const initExport = "InitializeExtension"

	manifest, err := ParseExtensionManifest([]byte(`{
		"name": "legacy-test",
		"command_name": "legacy-test",
		"help": "some help",
		"init": "InitializeExtension",
		"files": [
			{
				"os": "windows",
				"arch": "amd64",
				"path": "foo/test.dll"
			}
		]
	}`))
	if err != nil {
		t.Fatalf("Error parsing legacy extension manifest: %s", err)
	}
	if len(manifest.ExtCommand) != 1 {
		t.Fatalf("Expected 1 extension command, got %d", len(manifest.ExtCommand))
	}
	if manifest.ExtCommand[0].Init != initExport {
		t.Errorf("Expected init export %q, got %q", initExport, manifest.ExtCommand[0].Init)
	}
}

func TestRegisterExtensionPropagatesInit(t *testing.T) {
	const initExport = "InitializeExtension"

	tests := []struct {
		name      string
		session   *clientpb.Session
		beacon    *clientpb.Beacon
		wantAsync bool
		wantID    string
	}{
		{
			name:    "session",
			session: &clientpb.Session{ID: "session-id"},
			wantID:  "session-id",
		},
		{
			name:      "beacon",
			beacon:    &clientpb.Beacon{ID: "beacon-id"},
			wantAsync: true,
			wantID:    "beacon-id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotReq *sliverpb.RegisterExtensionReq
			rpc := &registerExtensionTestRPC{
				register: func(_ context.Context, req *sliverpb.RegisterExtensionReq, _ ...grpc.CallOption) (*sliverpb.RegisterExtension, error) {
					gotReq = req
					return &sliverpb.RegisterExtension{Response: &commonpb.Response{}}, nil
				},
			}
			con := newExtensionTestConsole(t, rpc, test.session, test.beacon)

			err := registerExtension("linux", &ExtCommand{CommandName: "test", Init: initExport}, []byte("extension"), nil, con)
			if err != nil {
				t.Fatalf("RegisterExtension returned an error: %s", err)
			}
			if gotReq == nil {
				t.Fatal("RegisterExtension RPC was not called")
			}
			if gotReq.Init != initExport {
				t.Errorf("Expected init export %q, got %q", initExport, gotReq.Init)
			}
			if gotReq.Request.GetAsync() != test.wantAsync {
				t.Errorf("Expected async=%t, got %t", test.wantAsync, gotReq.Request.GetAsync())
			}
			if test.wantAsync && gotReq.Request.GetBeaconID() != test.wantID {
				t.Errorf("Expected beacon ID %q, got %q", test.wantID, gotReq.Request.GetBeaconID())
			}
			if !test.wantAsync && gotReq.Request.GetSessionID() != test.wantID {
				t.Errorf("Expected session ID %q, got %q", test.wantID, gotReq.Request.GetSessionID())
			}
		})
	}
}

func TestRegisterExtensionWaitsForBeaconRegistration(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	rpc := &registerExtensionTestRPC{
		register: func(_ context.Context, _ *sliverpb.RegisterExtensionReq, _ ...grpc.CallOption) (*sliverpb.RegisterExtension, error) {
			close(started)
			<-release
			return &sliverpb.RegisterExtension{Response: &commonpb.Response{Async: true}}, nil
		},
	}
	con := newExtensionTestConsole(t, rpc, nil, &clientpb.Beacon{ID: "beacon-id"})
	done := make(chan error, 1)
	go func() {
		done <- registerExtension("linux", &ExtCommand{CommandName: "test"}, []byte("extension"), nil, con)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for RegisterExtension RPC")
	}
	select {
	case err := <-done:
		t.Fatalf("RegisterExtension returned before the beacon registration was enqueued: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RegisterExtension returned an error: %s", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for RegisterExtension to return")
	}
}

func TestRegisterExtensionReturnsBeaconRPCError(t *testing.T) {
	wantErr := errors.New("register failed")
	rpc := &registerExtensionTestRPC{
		register: func(_ context.Context, _ *sliverpb.RegisterExtensionReq, _ ...grpc.CallOption) (*sliverpb.RegisterExtension, error) {
			return nil, wantErr
		},
	}
	con := newExtensionTestConsole(t, rpc, nil, &clientpb.Beacon{ID: "beacon-id"})

	err := registerExtension("linux", &ExtCommand{CommandName: "test"}, []byte("extension"), nil, con)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error %q, got %v", wantErr, err)
	}
}

func TestRegisterExtensionRejectsNilResponse(t *testing.T) {
	rpc := &registerExtensionTestRPC{
		register: func(_ context.Context, _ *sliverpb.RegisterExtensionReq, _ ...grpc.CallOption) (*sliverpb.RegisterExtension, error) {
			return nil, nil
		},
	}
	con := newExtensionTestConsole(t, rpc, &clientpb.Session{ID: "session-id"}, nil)

	if err := registerExtension("linux", &ExtCommand{CommandName: "test"}, []byte("extension"), nil, con); err == nil {
		t.Fatal("RegisterExtension accepted a nil response")
	}
}

func TestExtensionCommandVisibilityUsesExactTargetPairs(t *testing.T) {
	extension := &ExtCommand{Files: []*extensionFile{
		{OS: "windows", Arch: "amd64"},
		{OS: "linux", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
	}}

	tests := []struct {
		name        string
		session     *clientpb.Session
		beacon      *clientpb.Beacon
		wantVisible bool
	}{
		{
			name:        "cross-platform Linux session",
			session:     &clientpb.Session{OS: "linux", Arch: "amd64"},
			wantVisible: true,
		},
		{
			name:        "cross-platform macOS beacon",
			beacon:      &clientpb.Beacon{OS: "darwin", Arch: "arm64"},
			wantVisible: true,
		},
		{
			name:        "cross-platform Windows session",
			session:     &clientpb.Session{OS: "windows", Arch: "amd64"},
			wantVisible: true,
		},
		{
			name:        "Linux architecture mismatch",
			session:     &clientpb.Session{OS: "linux", Arch: "arm64"},
			wantVisible: false,
		},
		{
			name:        "macOS architecture mismatch",
			beacon:      &clientpb.Beacon{OS: "darwin", Arch: "amd64"},
			wantVisible: false,
		},
		{
			name:        "Windows architecture mismatch",
			session:     &clientpb.Session{OS: "windows", Arch: "386"},
			wantVisible: false,
		},
		{
			name:        "unsupported native extension target",
			beacon:      &clientpb.Beacon{OS: "linux", Arch: "riscv64"},
			wantVisible: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			con := newExtensionTestConsole(t, nil, test.session, test.beacon)
			command := &cobra.Command{Annotations: makeCommandPlatformFilters(extension)}
			activeFilters := con.App.Menu(consts.ImplantMenu).ActiveFiltersFor(command)
			visible := len(activeFilters) == 0
			if visible != test.wantVisible {
				t.Fatalf("command visibility = %t, want %t (active filters: %v)", visible, test.wantVisible, activeFilters)
			}
		})
	}
}

func TestParseMultipleCmdManifest(t *testing.T) {
	expectedPath := util.ResolvePath("ex.dll")

	mextManifest, err := ParseExtensionManifest([]byte(multicmd))
	if err != nil {
		t.Errorf("error parsing manifest: %s", err)
	}
	if mextManifest.Name != "example-multientry" {
		t.Errorf("expected name example-multientry, got %s", mextManifest.Name)
	}

	if mextManifest.ExtCommand[0].CommandName != "startw" {
		t.Errorf("expected commandname startw, got %s", mextManifest.ExtCommand[0].CommandName)
	}
	if mextManifest.ExtCommand[1].CommandName != "Test2" {
		t.Errorf("expected commandname Test2, got %s", mextManifest.ExtCommand[1].CommandName)
	}
	if mextManifest.ExtCommand[0].Entrypoint != "StartW" {
		t.Errorf("expected entrypoint StartW, got %s", mextManifest.ExtCommand[0].Entrypoint)
	}
	if mextManifest.ExtCommand[1].Entrypoint != "Test2" {
		t.Errorf("expected entrypoint Test2, got %s", mextManifest.ExtCommand[1].Entrypoint)
	}
	if mextManifest.ExtCommand[0].Files[0].Path != expectedPath { // path normalization is platform-specific
		t.Errorf("expected path %s, got %s", expectedPath, mextManifest.ExtCommand[0].Files[0].Path)
	}
	if mextManifest.ExtCommand[1].Files[0].Path != expectedPath { // path normalization is platform-specific
		t.Errorf("expected path %s, got %s", expectedPath, mextManifest.ExtCommand[1].Files[0].Path)
	}
	//maybe some more? args?
}

func TestParseExtensionManifestErrors(t *testing.T) {
	sample3, err := ParseExtensionManifest([]byte(sample3))
	if err != nil {
		t.Fatalf("Failed to parse initial sample3: %s", err)
	}

	missingName := (*sample3)
	missingName.Name = ""
	data, _ := json.Marshal(missingName)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing name error, got none")
	}

	missingCmdName := (*sample3)
	missingCmdName.ExtCommand[0].CommandName = ""
	data, _ = json.Marshal(missingCmdName)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing command name error, got none")
	}

	missingHelp := (*sample3)
	missingHelp.ExtCommand[0].Help = ""
	data, _ = json.Marshal(missingHelp)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing help error, got none")
	}

	missingFiles := (*sample3)
	missingFiles.ExtCommand[0].Files = []*extensionFile{}
	data, _ = json.Marshal(missingFiles)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing files error, got none")
	}

	missingFileOS := (*sample3)
	missingFileOS.ExtCommand[0].Files = []*extensionFile{
		{
			OS:   "",
			Arch: "amd64",
			Path: "foo/test1.dll",
		},
	}
	data, _ = json.Marshal(missingFileOS)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing files.os error, got none")
	}

	missingFileArch := (*sample3)
	missingFileArch.ExtCommand[0].Files = []*extensionFile{
		{
			OS:   "windows",
			Arch: "",
			Path: "foo/test1.dll",
		},
	}
	data, _ = json.Marshal(missingFileArch)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing files.arch error, got none")
	}

	missingFilePath := (*sample3)
	missingFilePath.ExtCommand[0].Files = []*extensionFile{
		{
			OS:   "windows",
			Arch: "amd64",
			Path: "",
		},
	}
	data, _ = json.Marshal(missingFilePath)
	_, err = ParseExtensionManifest(data)
	if err == nil {
		t.Fatalf("Expected missing files.path error, got none")
	}

	invalidPaths := []string{
		"../../../../../",
		"/../../../../..",
		".",
		"/",
	}
	for _, invalidPath := range invalidPaths {
		missingFilePath2 := (*sample3)
		missingFilePath2.ExtCommand[0].Files = []*extensionFile{
			{
				OS:   "windows",
				Arch: "amd64",
				Path: invalidPath,
			},
		}
		data, _ = json.Marshal(missingFilePath2)
		_, err = ParseExtensionManifest(data)
		if err == nil {
			t.Fatalf("Expected missing files.path error, got none")
		}
	}
}

func TestParseExtensionManifestBOFExecutor(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "multi-command manifest",
			data: `{
				"name":"modern-bof",
				"commands":[{
					"command_name":"modern-bof",
					"help":"test",
					"bof_executor":"reflektor",
					"files":[{"os":"windows","arch":"amd64","path":"modern.o"}]
				}]
			}`,
			want: BOFExecutorReflektor,
		},
		{
			name: "legacy manifest",
			data: `{
				"name":"legacy-bof",
				"command_name":"legacy-bof",
				"help":"test",
				"bof_executor":"coff-loader",
				"depends_on":"custom-loader",
				"files":[{"os":"windows","arch":"amd64","path":"legacy.o"}]
			}`,
			want: BOFExecutorCOFFLoader,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := ParseExtensionManifest([]byte(test.data))
			if err != nil {
				t.Fatalf("ParseExtensionManifest returned an error: %s", err)
			}
			if got := manifest.ExtCommand[0].BOFExecutor; got != test.want {
				t.Fatalf("BOFExecutor = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseExtensionManifestRejectsInvalidBOFExecutor(t *testing.T) {
	tests := []struct {
		name        string
		executor    string
		dependsOn   string
		wantErrPart string
	}{
		{
			name:        "executor must be exact lowercase",
			executor:    "Reflektor",
			wantErrPart: "invalid `bof_executor`",
		},
		{
			name:        "unknown executor",
			executor:    "other-loader",
			wantErrPart: "invalid `bof_executor`",
		},
		{
			name:        "coff loader requires dependency",
			executor:    BOFExecutorCOFFLoader,
			wantErrPart: "no `depends_on` fallback",
		},
		{
			name:        "coff loader rejects whitespace dependency",
			executor:    BOFExecutorCOFFLoader,
			dependsOn:   " \t",
			wantErrPart: "no `depends_on` fallback",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := &ExtensionManifest{
				Name: "test",
				ExtCommand: []*ExtCommand{{
					CommandName: "test",
					Help:        "test",
					BOFExecutor: test.executor,
					DependsOn:   test.dependsOn,
					Files: []*extensionFile{{
						OS: "windows", Arch: "amd64", Path: "test.o",
					}},
				}},
			}
			err := validManifest(manifest)
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("validManifest error = %v, want error containing %q", err, test.wantErrPart)
			}
		})
	}
}

func TestPlanExtensionExecution(t *testing.T) {
	tests := []struct {
		name         string
		ext          *ExtCommand
		path         string
		capabilities uint64
		want         extensionExecutionMode
		wantErr      bool
	}{
		{
			name: "native artifact remains native",
			ext:  &ExtCommand{CommandName: "native"},
			path: "native.dll",
			want: extensionExecutionNative,
		},
		{
			name:         "legacy manifest keeps dependency route",
			ext:          &ExtCommand{CommandName: "legacy", DependsOn: "coff-loader"},
			path:         "legacy.o",
			capabilities: sliverpb.CapabilityBOFV1,
			want:         extensionExecutionCOFFLoader,
		},
		{
			name:         "explicit coff loader is authoritative",
			ext:          &ExtCommand{CommandName: "coff", BOFExecutor: BOFExecutorCOFFLoader, DependsOn: "custom-loader"},
			path:         "coff.bin",
			capabilities: sliverpb.CapabilityBOFV1,
			want:         extensionExecutionCOFFLoader,
		},
		{
			name:         "explicit Reflektor uses capable target",
			ext:          &ExtCommand{CommandName: "builtin", BOFExecutor: BOFExecutorReflektor, DependsOn: "coff-loader"},
			path:         "builtin.o",
			capabilities: sliverpb.CapabilityBOFV1,
			want:         extensionExecutionReflektor,
		},
		{
			name: "explicit Reflektor falls back to dependency",
			ext:  &ExtCommand{CommandName: "fallback", BOFExecutor: BOFExecutorReflektor, DependsOn: "coff-loader"},
			path: "fallback.o",
			want: extensionExecutionCOFFLoader,
		},
		{
			name:         "dependency-free BOF defaults to Reflektor",
			ext:          &ExtCommand{CommandName: "default"},
			path:         "default.o",
			capabilities: sliverpb.CapabilityBOFV1,
			want:         extensionExecutionReflektor,
		},
		{
			name:         "uppercase object extension is a BOF",
			ext:          &ExtCommand{CommandName: "uppercase"},
			path:         "uppercase.O",
			capabilities: sliverpb.CapabilityBOFV1,
			want:         extensionExecutionReflektor,
		},
		{
			name:    "unsupported dependency-free BOF fails closed",
			ext:     &ExtCommand{CommandName: "unsupported"},
			path:    "unsupported.o",
			wantErr: true,
		},
		{
			name:    "coff loader without dependency fails closed",
			ext:     &ExtCommand{CommandName: "missing", BOFExecutor: BOFExecutorCOFFLoader},
			path:    "missing.o",
			wantErr: true,
		},
		{
			name:         "invalid executor fails closed",
			ext:          &ExtCommand{CommandName: "invalid", BOFExecutor: "other"},
			path:         "invalid.o",
			capabilities: sliverpb.CapabilityBOFV1,
			wantErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := planExtensionExecution(test.ext, test.path, test.capabilities)
			if test.wantErr {
				if err == nil {
					t.Fatalf("planExtensionExecution returned mode %d, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("planExtensionExecution returned an error: %s", err)
			}
			if got != test.want {
				t.Fatalf("planExtensionExecution mode = %d, want %d", got, test.want)
			}
		})
	}
}

func TestBuildCallExtensionRequestPreservesLegacyWireFields(t *testing.T) {
	root := t.TempDir()
	objectData := []byte("object-data")
	loaderData := []byte("loader-data")
	objectPath := filepath.Join(root, "object.o")
	loaderPath := filepath.Join(root, "loader.dll")
	if err := os.WriteFile(objectPath, objectData, 0o600); err != nil {
		t.Fatalf("write object fixture: %s", err)
	}
	if err := os.WriteFile(loaderPath, loaderData, 0o600); err != nil {
		t.Fatalf("write loader fixture: %s", err)
	}

	const loaderName = "custom-loader"
	previousLoader, hadPreviousLoader := loadedExtensions[loaderName]
	defer func() {
		if hadPreviousLoader {
			loadedExtensions[loaderName] = previousLoader
		} else {
			delete(loadedExtensions, loaderName)
		}
	}()
	loaderManifest := &ExtensionManifest{Name: loaderName, RootPath: root}
	loadedExtensions[loaderName] = &ExtCommand{
		CommandName: loaderName,
		Entrypoint:  "RunCOFF",
		Files: []*extensionFile{{
			OS: "windows", Arch: "amd64", Path: filepath.Base(loaderPath),
		}},
		Manifest: loaderManifest,
	}

	legacyArgs := []byte{0x01, 0x02, 0x03, 0x04}
	ext := &ExtCommand{CommandName: "test-bof", DependsOn: loaderName}
	got, err := buildCallExtensionRequest("windows", "amd64", extensionExecutionCOFFLoader, ext, objectPath, "RunCOFF", legacyArgs)
	if err != nil {
		t.Fatalf("buildCallExtensionRequest returned an error: %s", err)
	}
	loaderHash := sha256.Sum256(loaderData)
	want := &sliverpb.CallExtensionReq{
		Name:   hex.EncodeToString(loaderHash[:]),
		Export: "RunCOFF",
		Args:   legacyArgs,
	}
	gotWire, err := proto.Marshal(got)
	if err != nil {
		t.Fatalf("marshal actual request: %s", err)
	}
	wantWire, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal expected request: %s", err)
	}
	if !bytes.Equal(gotWire, wantWire) {
		t.Fatalf("legacy request wire bytes changed:\n got: %x\nwant: %x", gotWire, wantWire)
	}
}

func TestBuildCallExtensionRequestForReflektor(t *testing.T) {
	root := t.TempDir()
	objectData := []byte("object-data")
	objectPath := filepath.Join(root, "object.o")
	if err := os.WriteFile(objectPath, objectData, 0o600); err != nil {
		t.Fatalf("write object fixture: %s", err)
	}

	typedArgs := []byte{0x05, 0x06}
	got, err := buildCallExtensionRequest("windows", "amd64", extensionExecutionReflektor, &ExtCommand{CommandName: "test-bof"}, objectPath, "go", typedArgs)
	if err != nil {
		t.Fatalf("buildCallExtensionRequest returned an error: %s", err)
	}
	objectHash := sha256.Sum256(objectData)
	if got.Name != hex.EncodeToString(objectHash[:]) {
		t.Errorf("Name = %q, want object hash %q", got.Name, hex.EncodeToString(objectHash[:]))
	}
	if got.Export != "go" {
		t.Errorf("Export = %q, want %q", got.Export, "go")
	}
	if !bytes.Equal(got.Args, typedArgs) {
		t.Errorf("Args = %x, want %x", got.Args, typedArgs)
	}
	if !bytes.Equal(got.BOFData, objectData) {
		t.Errorf("BOFData = %q, want %q", got.BOFData, objectData)
	}
	if !got.IsBOF {
		t.Error("IsBOF = false, want true")
	}
}
