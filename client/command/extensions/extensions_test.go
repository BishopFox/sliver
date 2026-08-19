package extensions

import (
	"context"
	"encoding/json"
	"errors"
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
