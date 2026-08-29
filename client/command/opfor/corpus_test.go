//go:build client

package opfor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type corpusRPC struct {
	rpcpb.SliverRPCClient

	mu       sync.Mutex
	sessions []*clientpb.Session
	calls    []*sliverpb.CallExtensionReq
}

func (rpc *corpusRPC) GetSessions(context.Context, *commonpb.Empty, ...grpc.CallOption) (*clientpb.Sessions, error) {
	return &clientpb.Sessions{Sessions: rpc.sessions}, nil
}

func (*corpusRPC) GetBeacons(context.Context, *commonpb.Empty, ...grpc.CallOption) (*clientpb.Beacons, error) {
	return &clientpb.Beacons{}, nil
}

func (*corpusRPC) GetBeaconTaskContent(context.Context, *clientpb.BeaconTask, ...grpc.CallOption) (*clientpb.BeaconTask, error) {
	return nil, fmt.Errorf("unexpected beacon task poll in session corpus test")
}

func (rpc *corpusRPC) CallExtension(_ context.Context, request *sliverpb.CallExtensionReq, _ ...grpc.CallOption) (*sliverpb.CallExtension, error) {
	rpc.mu.Lock()
	rpc.calls = append(rpc.calls, proto.Clone(request).(*sliverpb.CallExtensionReq))
	rpc.mu.Unlock()
	return &sliverpb.CallExtension{Response: &commonpb.Response{}}, nil
}

func (rpc *corpusRPC) snapshotCalls() []*sliverpb.CallExtensionReq {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	return append([]*sliverpb.CallExtensionReq(nil), rpc.calls...)
}

type corpusPromptUI struct{}

func (corpusPromptUI) Confirm(_ string, defaultValue bool) (bool, error) {
	return defaultValue, nil
}

func (corpusPromptUI) Input(_ string, defaultValue string) (string, error) {
	return defaultValue, nil
}

type corpusOutput struct{}

func (corpusOutput) Printf(string, ...any)        {}
func (corpusOutput) PrintInfof(string, ...any)    {}
func (corpusOutput) PrintSuccessf(string, ...any) {}
func (corpusOutput) PrintErrorf(string, ...any)   {}

func newCorpusManager(t *testing.T, rpc rpcpb.SliverRPCClient) *Manager {
	t.Helper()

	client := console.NewConsole(false)
	client.Rpc = rpc
	manager, err := newManagerWithOutput(client, corpusPromptUI{}, corpusOutput{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = manager.runtime.Close(context.Background())
	})
	return manager
}

func materializeCorpusScript(t *testing.T, fixture, scriptName, objectPath string, object []byte) string {
	t.Helper()

	source, err := os.ReadFile(filepath.Join("testdata", "corpus", filepath.FromSlash(fixture)))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	scriptPath := filepath.Join(root, scriptName)
	if err := os.WriteFile(scriptPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	resourcePath := filepath.Join(root, filepath.FromSlash(objectPath))
	if err := os.MkdirAll(filepath.Dir(resourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resourcePath, object, 0o600); err != nil {
		t.Fatal(err)
	}
	return scriptPath
}

func expectedStringArguments(value string) []byte {
	payload := make([]byte, 4+len(value)+1)
	payload[0] = byte(len(value) + 1)
	payload[1] = byte((len(value) + 1) >> 8)
	payload[2] = byte((len(value) + 1) >> 16)
	payload[3] = byte((len(value) + 1) >> 24)
	copy(payload[4:], value)
	arguments := make([]byte, 4+len(payload))
	arguments[0] = byte(len(payload))
	arguments[1] = byte(len(payload) >> 8)
	arguments[2] = byte(len(payload) >> 16)
	arguments[3] = byte(len(payload) >> 24)
	copy(arguments[4:], payload)
	return arguments
}

func TestSliverArmoryCNACorpusDispatchesBOFs(t *testing.T) {
	tests := []struct {
		name          string
		fixture       string
		scriptName    string
		objectPath    string
		alias         string
		arguments     []string
		packedString  *string
		objectContent []byte
	}{
		{
			name:          "FirefoxDump all",
			fixture:       "firefoxdump/firefoxdump.cna",
			scriptName:    "firefoxdump.cna",
			objectPath:    "bin/firefoxdump.x64.o",
			alias:         "firefoxdump",
			arguments:     []string{"/all"},
			packedString:  stringPointer("/all"),
			objectContent: []byte("firefoxdump-x64-object"),
		},
		{
			name:          "bof_collection cat path with spaces",
			fixture:       "bof_collection/cat.cna",
			scriptName:    "cat.cna",
			objectPath:    "dist/cat.x64.o",
			alias:         "cat",
			arguments:     []string{`C:\Program`, `Files\notes.txt`},
			packedString:  stringPointer(`C:\Program Files\notes.txt`),
			objectContent: []byte("cat-x64-object"),
		},
		{
			name:          "OperatorsKit FindDotnet no arguments",
			fixture:       "operatorskit/finddotnet.cna",
			scriptName:    "finddotnet.cna",
			objectPath:    "finddotnet.o",
			alias:         "finddotnet",
			objectContent: []byte("finddotnet-x64-object"),
		},
		{
			name:          "OperatorsKit FindSysmon registry",
			fixture:       "operatorskit/findsysmon.cna",
			scriptName:    "findsysmon.cna",
			objectPath:    "findsysmon.o",
			alias:         "findsysmon",
			arguments:     []string{"reg"},
			packedString:  stringPointer("reg"),
			objectContent: []byte("findsysmon-x64-object"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := &clientpb.Session{
				ID:           "session-corpus",
				Name:         "corpus",
				OS:           "windows",
				Arch:         "amd64",
				Capabilities: sliverpb.CapabilityBOFV1,
			}
			rpc := &corpusRPC{sessions: []*clientpb.Session{session}}
			manager := newCorpusManager(t, rpc)
			scriptPath := materializeCorpusScript(
				t, test.fixture, test.scriptName, test.objectPath, test.objectContent,
			)

			loadedPath, err := manager.Load(context.Background(), scriptPath)
			if err != nil {
				t.Fatalf("load pinned %s fixture: %v", test.fixture, err)
			}
			if loadedPath != scriptPath {
				t.Fatalf("loaded path = %q, want %q", loadedPath, scriptPath)
			}
			if names := manager.aliasNames(); len(names) != 1 || names[0] != test.alias {
				t.Fatalf("registered aliases = %v, want [%s]", names, test.alias)
			}

			rawInput := strings.Join(append([]string{test.alias}, test.arguments...), " ")
			if err := manager.invokeAlias(context.Background(), test.alias, rawInput, test.arguments, session.ID); err != nil {
				t.Fatalf("invoke %s: %v", test.alias, err)
			}

			calls := rpc.snapshotCalls()
			if len(calls) != 1 {
				t.Fatalf("CallExtension calls = %d, want 1", len(calls))
			}
			call := calls[0]
			if !call.IsBOF || call.Export != "go" {
				t.Fatalf("CallExtension BOF/export = %v/%q, want true/go", call.IsBOF, call.Export)
			}
			if !bytes.Equal(call.BOFData, test.objectContent) {
				t.Fatalf("CallExtension BOF bytes = %q, want %q", call.BOFData, test.objectContent)
			}
			wantArguments := []byte{0, 0, 0, 0}
			if test.packedString != nil {
				wantArguments = expectedStringArguments(*test.packedString)
			}
			if !bytes.Equal(call.Args, wantArguments) {
				t.Fatalf("CallExtension args = %x, want Reflektor framing %x", call.Args, wantArguments)
			}
			if call.GetRequest().GetSessionID() != session.ID || call.GetRequest().GetAsync() {
				t.Fatalf("CallExtension request = %#v, want synchronous session %s", call.GetRequest(), session.ID)
			}
			if len(call.Name) != 64 {
				t.Fatalf("CallExtension name digest length = %d, want 64", len(call.Name))
			}

			if _, err := manager.Unload(context.Background(), scriptPath); err != nil {
				t.Fatalf("unload %s: %v", test.alias, err)
			}
			if names := manager.aliasNames(); len(names) != 0 {
				t.Fatalf("aliases after unload = %v, want none", names)
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}
