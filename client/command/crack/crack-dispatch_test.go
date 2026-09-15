package crack

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func newCrackDispatchCommand(t *testing.T, flags ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "crack-test"}
	cmd.SetContext(context.Background())
	cmd.Flags().Int64("timeout", 0, "test timeout")
	bindCrackFlags(cmd.Flags())
	if err := cmd.Flags().Parse(flags); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return cmd
}

func TestClassifyCrackInvocation(t *testing.T) {
	tests := []struct {
		name             string
		flags            []string
		args             []string
		wantMode         crackInvocationMode
		wantBackendLevel uint32
		wantHashLevel    uint32
		wantCrackstation string
		wantError        string
	}{
		{name: "dashboard", wantMode: crackInvocationDashboard},
		{name: "timeout only", flags: []string{"--timeout=5"}, wantMode: crackInvocationDashboard},
		{name: "positional hash job", args: []string{"hash"}, wantMode: crackInvocationJob},
		{name: "job flag", flags: []string{"--attack-mode=3"}, wantMode: crackInvocationJob},
		{name: "explicit false is a no-op", flags: []string{"--backend-info=false"}, wantMode: crackInvocationDashboard},
		{name: "explicit false with job flag", flags: []string{"--backend-info=false", "--attack-mode=3"}, wantMode: crackInvocationJob},
		{name: "keyspace false is a no-op", flags: []string{"--keyspace=false"}, wantMode: crackInvocationDashboard},
		{name: "keyspace false with job flag", flags: []string{"--keyspace=false", "--attack-mode=3"}, wantMode: crackInvocationJob},
		{name: "keyspace", flags: []string{"--keyspace"}, wantMode: crackInvocationKeyspace},
		{name: "keyspace candidate flags", flags: []string{"--keyspace", "--attack-mode=3", "--input=?d?d", "--increment"}, wantMode: crackInvocationKeyspace},
		{name: "keyspace selected station", flags: []string{"--keyspace", "--crackstation= station-one "}, wantMode: crackInvocationKeyspace, wantCrackstation: "station-one"},
		{name: "keyspace timeout", flags: []string{"--keyspace", "--timeout=5"}, wantMode: crackInvocationKeyspace},
		{name: "keyspace skip conflict", flags: []string{"--keyspace", "--skip=1"}, wantError: "--skip"},
		{name: "keyspace zero skip no-op", flags: []string{"--keyspace", "--skip=0"}, wantMode: crackInvocationKeyspace},
		{name: "keyspace positional hash conflict", flags: []string{"--keyspace"}, args: []string{"hash"}, wantError: "positional hash arguments"},
		{name: "keyspace hash flag conflict", flags: []string{"--keyspace", "--hash=digest"}, wantError: "--hash"},
		{name: "keyspace credential conflict", flags: []string{"--keyspace", "--credential=deadbeef"}, wantError: "--credential"},
		{name: "keyspace collection conflict", flags: []string{"--keyspace", "--credential-collection=secrets"}, wantError: "--credential-collection"},
		{name: "keyspace include cracked conflict", flags: []string{"--keyspace", "--include-cracked-credentials"}, wantError: "--include-cracked-credentials"},
		{name: "keyspace and backend info conflict", flags: []string{"--keyspace", "--backend-info"}, wantError: "--keyspace"},
		{name: "backend info ignores disabled keyspace", flags: []string{"--backend-info", "--keyspace=false"}, wantMode: crackInvocationBackendInfo, wantBackendLevel: 1},
		{name: "backend info", flags: []string{"--backend-info"}, wantMode: crackInvocationBackendInfo, wantBackendLevel: 1},
		{name: "backend info and timeout", flags: []string{"--backend-info", "--timeout=5"}, wantMode: crackInvocationBackendInfo, wantBackendLevel: 1},
		{name: "level one", flags: []string{"--backend-info-level=1"}, wantMode: crackInvocationBackendInfo, wantBackendLevel: 1},
		{name: "level two", flags: []string{"--backend-info-level=2"}, wantMode: crackInvocationBackendInfo, wantBackendLevel: 2},
		{name: "explicit level wins", flags: []string{"--backend-info", "--backend-info-level=2"}, wantMode: crackInvocationBackendInfo, wantBackendLevel: 2},
		{name: "level independently enables mode", flags: []string{"--backend-info=false", "--backend-info-level=2"}, wantMode: crackInvocationBackendInfo, wantBackendLevel: 2},
		{name: "zero level", flags: []string{"--backend-info-level=0"}, wantError: "must be 1 or 2"},
		{name: "unsupported level", flags: []string{"--backend-info-level=3"}, wantError: "must be 1 or 2"},
		{name: "positional conflict", flags: []string{"--backend-info"}, args: []string{"hash"}, wantError: "positional hash arguments"},
		{name: "job flag conflict", flags: []string{"--backend-info", "--benchmark"}, wantError: "--benchmark"},
		{name: "input conflict", flags: []string{"--backend-info-level=2", "--input=wordlist"}, wantError: "--input"},
		{name: "backend info station conflict", flags: []string{"--backend-info", "--crackstation=station-one"}, wantError: "--crackstation"},
		{name: "total candidates", flags: []string{"--total-candidates", "--attack-mode=3", "--input=?d"}, wantMode: crackInvocationTotalCandidates},
		{name: "total candidates false", flags: []string{"--total-candidates=false"}, wantMode: crackInvocationDashboard},
		{name: "total candidates hash conflict", flags: []string{"--total-candidates", "--hash=digest"}, wantError: "--hash"},
		{name: "total candidates limit conflict", flags: []string{"--total-candidates", "--limit=1"}, wantError: "--limit"},
		{name: "lookup", flags: []string{"--lookup=candidate", "--attack-mode=3", "--input=?d"}, wantMode: crackInvocationLookup},
		{name: "lookup allows window", flags: []string{"--lookup=candidate", "--skip=1", "--limit=2"}, wantMode: crackInvocationLookup},
		{name: "lookup empty", flags: []string{"--lookup="}, wantError: "cannot be empty"},
		{name: "query modes conflict", flags: []string{"--keyspace", "--total-candidates"}, wantError: "cannot be combined"},
		{name: "candidate query terminal conflict", flags: []string{"--keyspace", "--show"}, wantError: "--show"},
		{name: "identify positional", flags: []string{"--identify-mode"}, args: []string{"digest"}, wantMode: crackInvocationIdentify},
		{name: "identify hash flag", flags: []string{"--identify-mode", "--hash=digest"}, wantMode: crackInvocationIdentify},
		{name: "identify false", flags: []string{"--identify-mode=false"}, wantMode: crackInvocationDashboard},
		{name: "identify missing hash", flags: []string{"--identify-mode"}, wantError: "requires a positional hash or --hash"},
		{name: "identify credential conflict", flags: []string{"--identify-mode", "--credential=deadbeef"}, wantError: "--credential"},
		{name: "identify hash mode conflict", flags: []string{"--identify-mode", "--hash=digest", "--hash-mode=0"}, wantError: "--hash-mode"},
		{name: "hash info", flags: []string{"--hash-info"}, wantMode: crackInvocationHashInfo, wantHashLevel: 1},
		{name: "hash info false", flags: []string{"--hash-info=false"}, wantMode: crackInvocationDashboard},
		{name: "hash info level", flags: []string{"--hash-info=false", "--hash-info-level=2", "--hash-mode=0", "--crackstation=station-one"}, wantMode: crackInvocationHashInfo, wantHashLevel: 2, wantCrackstation: "station-one"},
		{name: "hash info zero level", flags: []string{"--hash-info-level=0"}, wantError: "must be 1 or 2"},
		{name: "hash info unsupported level", flags: []string{"--hash-info-level=3"}, wantError: "must be 1 or 2"},
		{name: "hash info hash conflict", flags: []string{"--hash-info", "--hash=digest"}, wantError: "--hash"},
		{name: "crackstation requires query", flags: []string{"--crackstation=station-one"}, wantError: "requires --keyspace"},
		{name: "crackstation cannot be empty", flags: []string{"--keyspace", "--crackstation= "}, wantError: "cannot be empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := classifyCrackInvocation(newCrackDispatchCommand(t, test.flags...), test.args)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("classifyCrackInvocation() error = %v, want substring %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("classifyCrackInvocation() error = %v", err)
			}
			if invocation.mode != test.wantMode || invocation.backendInfoLevel != test.wantBackendLevel ||
				invocation.hashInfoLevel != test.wantHashLevel || invocation.crackstation != test.wantCrackstation {
				t.Fatalf("classifyCrackInvocation() = %#v, want mode %d backend level %d hash level %d station %q",
					invocation, test.wantMode, test.wantBackendLevel, test.wantHashLevel, test.wantCrackstation)
			}
		})
	}
}

type crackDispatchRPCCapture struct {
	rpcpb.SliverRPCClient
	crackstationsCalls int
	crackCalls         int
	crackFilesCalls    int
	hadDeadline        bool
	crackHadDeadline   bool
	err                error
	response           *clientpb.Crackstations
	crackResponse      *clientpb.CrackResponse
	crackFilesResponse *clientpb.CrackFiles
	crackRequest       *clientpb.CrackCommand
}

func (capture *crackDispatchRPCCapture) Crackstations(ctx context.Context, _ *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.Crackstations, error) {
	capture.crackstationsCalls++
	_, capture.hadDeadline = ctx.Deadline()
	if capture.err != nil {
		return nil, capture.err
	}
	if capture.response != nil {
		return capture.response, nil
	}
	return &clientpb.Crackstations{}, nil
}

func (capture *crackDispatchRPCCapture) Crack(ctx context.Context, request *clientpb.CrackCommand, _ ...grpc.CallOption) (*clientpb.CrackResponse, error) {
	capture.crackCalls++
	_, capture.crackHadDeadline = ctx.Deadline()
	capture.crackRequest = request
	if capture.crackResponse != nil {
		return capture.crackResponse, nil
	}
	return &clientpb.CrackResponse{}, nil
}

func (capture *crackDispatchRPCCapture) CrackFilesList(_ context.Context, _ *clientpb.CrackFile, _ ...grpc.CallOption) (*clientpb.CrackFiles, error) {
	capture.crackFilesCalls++
	if capture.crackFilesResponse != nil {
		return capture.crackFilesResponse, nil
	}
	return &clientpb.CrackFiles{}, nil
}

func TestCrackBackendInfoQueriesStationsWithoutCreatingJob(t *testing.T) {
	capture := &crackDispatchRPCCapture{}
	con := console.NewConsole(false)
	emptyCommands := func() *cobra.Command { return &cobra.Command{Use: "test"} }
	if err := console.StartClient(con, capture, nil, nil, emptyCommands, emptyCommands, false, ""); err != nil {
		t.Fatal(err)
	}

	CrackCmd(newCrackDispatchCommand(t, "--backend-info", "--timeout=1"), con, nil)
	if capture.crackstationsCalls != 1 {
		t.Fatalf("Crackstations calls = %d, want 1", capture.crackstationsCalls)
	}
	if capture.crackCalls != 0 || capture.crackFilesCalls != 0 {
		t.Fatalf("backend-info used job path: Crack=%d CrackFilesList=%d", capture.crackCalls, capture.crackFilesCalls)
	}
	if !capture.hadDeadline {
		t.Fatal("Crackstations context did not inherit --timeout")
	}
}

func TestCrackKeyspaceUsesSynchronousCrackResponse(t *testing.T) {
	digest := strings.Repeat("a", 64)
	capture := &crackDispatchRPCCapture{
		crackResponse: &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
			Mode: clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE, Value: "100",
		}},
		crackFilesResponse: &clientpb.CrackFiles{
			Files: []*clientpb.CrackFile{{
				ID: "wordlist-id", Name: "words.txt", Type: clientpb.CrackFileType_WORDLIST, Sha2_256: digest,
			}},
		},
	}
	con := console.NewConsole(false)
	emptyCommands := func() *cobra.Command { return &cobra.Command{Use: "test"} }
	if err := console.StartClient(con, capture, nil, nil, emptyCommands, emptyCommands, false, ""); err != nil {
		t.Fatal(err)
	}

	CrackCmd(newCrackDispatchCommand(t, "--keyspace", "--input=words.txt", "--crackstation=station-one", "--timeout=1"), con, nil)
	if capture.crackCalls != 1 {
		t.Fatalf("Crack calls = %d, want 1", capture.crackCalls)
	}
	if capture.crackstationsCalls != 0 {
		t.Fatalf("keyspace queried crackstations directly %d times", capture.crackstationsCalls)
	}
	if capture.crackFilesCalls != 1 {
		t.Fatalf("CrackFilesList calls = %d, want 1", capture.crackFilesCalls)
	}
	if !capture.crackHadDeadline {
		t.Fatal("Crack context did not inherit --timeout")
	}
	if capture.crackRequest == nil || !capture.crackRequest.Keyspace {
		t.Fatalf("Crack request = %#v, want keyspace mode", capture.crackRequest)
	}
	if capture.crackRequest.Crackstation != "station-one" {
		t.Fatalf("Crack request station = %q, want station-one", capture.crackRequest.Crackstation)
	}
	wantInput := "crackfile://wordlist/" + digest
	if len(capture.crackRequest.PositionalArguments) != 1 || capture.crackRequest.PositionalArguments[0] != wantInput {
		t.Fatalf("resolved keyspace inputs = %#v, want %q", capture.crackRequest.PositionalArguments, wantInput)
	}
	if len(capture.crackRequest.Hashes) != 0 || len(capture.crackRequest.CredentialIDs) != 0 {
		t.Fatalf("keyspace request contains hash selectors: %#v", capture.crackRequest)
	}
}

func TestCrackSynchronousQueriesUseCrackRPCWithoutCreatingJob(t *testing.T) {
	tests := []struct {
		name       string
		flags      []string
		args       []string
		resultMode clientpb.CrackQueryMode
		result     string
		check      func(*testing.T, *clientpb.CrackCommand)
	}{
		{
			name:       "total candidates",
			flags:      []string{"--total-candidates", "--attack-mode=3", "--input=?d", "--crackstation=station-one"},
			resultMode: clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES,
			result:     "100",
			check: func(t *testing.T, command *clientpb.CrackCommand) {
				if !command.TotalCandidates || command.Crackstation != "station-one" {
					t.Fatalf("total-candidates request = %#v", command)
				}
			},
		},
		{
			name:       "lookup",
			flags:      []string{"--lookup=candidate", "--attack-mode=3", "--input=?d"},
			resultMode: clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP,
			result:     "lookup output",
			check: func(t *testing.T, command *clientpb.CrackCommand) {
				if command.Lookup != "candidate" {
					t.Fatalf("lookup request = %#v", command)
				}
			},
		},
		{
			name:       "identify",
			flags:      []string{"--identify-mode"},
			args:       []string{"digest"},
			resultMode: clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY,
			result:     "identify output",
			check: func(t *testing.T, command *clientpb.CrackCommand) {
				if !command.IdentifyMode || len(command.Hashes) != 1 || command.Hashes[0] != "digest" {
					t.Fatalf("identify request = %#v", command)
				}
			},
		},
		{
			name:       "hash info",
			flags:      []string{"--hash-info-level=2", "--hash-mode=1000"},
			resultMode: clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO,
			result:     "hash-info output",
			check: func(t *testing.T, command *clientpb.CrackCommand) {
				if command.HashInfoLevel != 2 || command.HashMode == nil || *command.HashMode != 1000 {
					t.Fatalf("hash-info request = %#v", command)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := &crackDispatchRPCCapture{crackResponse: &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
				Mode: test.resultMode, Value: test.result,
			}}}
			con := console.NewConsole(false)
			emptyCommands := func() *cobra.Command { return &cobra.Command{Use: "test"} }
			if err := console.StartClient(con, capture, nil, nil, emptyCommands, emptyCommands, false, ""); err != nil {
				t.Fatal(err)
			}

			CrackCmd(newCrackDispatchCommand(t, test.flags...), con, test.args)
			if capture.crackCalls != 1 || capture.crackstationsCalls != 0 {
				t.Fatalf("RPC calls: Crack=%d Crackstations=%d", capture.crackCalls, capture.crackstationsCalls)
			}
			if capture.crackRequest == nil {
				t.Fatal("Crack request was not captured")
			}
			test.check(t, capture.crackRequest)
		})
	}
}

func TestCanonicalCrackKeyspace(t *testing.T) {
	tests := []struct {
		name     string
		response *clientpb.CrackResponse
		want     string
		wantErr  bool
	}{
		{name: "canonical", response: &clientpb.CrackResponse{Keyspace: "100"}, want: "100"},
		{name: "normalizes", response: &clientpb.CrackResponse{Keyspace: " 00100 "}, want: "100"},
		{name: "zero", response: &clientpb.CrackResponse{Keyspace: "0"}, want: "0"},
		{name: "missing response", wantErr: true},
		{name: "missing value", response: &clientpb.CrackResponse{}, wantErr: true},
		{name: "negative", response: &clientpb.CrackResponse{Keyspace: "-1"}, wantErr: true},
		{name: "overflow", response: &clientpb.CrackResponse{Keyspace: "18446744073709551616"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := canonicalCrackKeyspace(test.response)
			if (err != nil) != test.wantErr {
				t.Fatalf("canonicalCrackKeyspace() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("canonicalCrackKeyspace() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCrackQueryResponseValue(t *testing.T) {
	tests := []struct {
		name       string
		invocation crackInvocation
		response   *clientpb.CrackResponse
		want       string
		wantError  string
	}{
		{
			name:       "typed keyspace",
			invocation: crackInvocation{mode: crackInvocationKeyspace},
			response: &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
				Mode: clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE, Value: " 00100 ",
			}},
			want: "100",
		},
		{
			name:       "typed total candidates",
			invocation: crackInvocation{mode: crackInvocationTotalCandidates},
			response: &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
				Mode: clientpb.CrackQueryMode_CRACK_QUERY_TOTAL_CANDIDATES, Value: "42\n",
			}},
			want: "42",
		},
		{
			name:       "text remains verbatim",
			invocation: crackInvocation{mode: crackInvocationLookup},
			response: &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
				Mode: clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP, Value: "line one\nline two\n",
			}},
			want: "line one\nline two\n",
		},
		{
			name:       "wrong typed mode",
			invocation: crackInvocation{mode: crackInvocationLookup},
			response: &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
				Mode: clientpb.CrackQueryMode_CRACK_QUERY_IDENTIFY, Value: "output",
			}},
			wantError: "for lookup query",
		},
		{
			name:       "new mode cannot use legacy keyspace",
			invocation: crackInvocation{mode: crackInvocationTotalCandidates},
			response:   &clientpb.CrackResponse{Keyspace: "42"},
			wantError:  "empty total-candidates query result",
		},
		{
			name:       "selected keyspace cannot use legacy response",
			invocation: crackInvocation{mode: crackInvocationKeyspace, crackstation: "station-one"},
			response:   &clientpb.CrackResponse{Keyspace: "42"},
			wantError:  "empty keyspace query result",
		},
		{
			name:       "empty text",
			invocation: crackInvocation{mode: crackInvocationHashInfo},
			response: &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
				Mode: clientpb.CrackQueryMode_CRACK_QUERY_HASH_INFO, Value: " \n",
			}},
			wantError: "empty hash-info query result",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := crackQueryResponseValue(test.response, test.invocation)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("crackQueryResponseValue() error = %v, want substring %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("crackQueryResponseValue() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCrackQueryResponsePreservesStderr(t *testing.T) {
	response := &clientpb.CrackResponse{Query: &clientpb.CrackQueryResult{
		Mode:   clientpb.CrackQueryMode_CRACK_QUERY_LOOKUP,
		Value:  "lookup output",
		Stderr: "hashcat warning\n",
	}}
	value, stderr, err := crackQueryResponse(response, crackInvocation{mode: crackInvocationLookup})
	if err != nil {
		t.Fatal(err)
	}
	if value != "lookup output" || stderr != "hashcat warning\n" {
		t.Fatalf("crackQueryResponse() = (%q, %q), want preserved value and stderr", value, stderr)
	}
}

func TestCrackResponseKeyspaceFieldNumberAndRoundTrip(t *testing.T) {
	field := (&clientpb.CrackResponse{}).ProtoReflect().Descriptor().Fields().ByName("Keyspace")
	if field == nil {
		t.Fatal("CrackResponse is missing field Keyspace")
	}
	if got, want := field.Number(), protoreflect.FieldNumber(2); got != want {
		t.Fatalf("CrackResponse.Keyspace tag = %d, want %d", got, want)
	}
	queryField := (&clientpb.CrackResponse{}).ProtoReflect().Descriptor().Fields().ByName("Query")
	if queryField == nil {
		t.Fatal("CrackResponse is missing field Query")
	}
	if got, want := queryField.Number(), protoreflect.FieldNumber(3); got != want {
		t.Fatalf("CrackResponse.Query tag = %d, want %d", got, want)
	}
	stderrField := (&clientpb.CrackQueryResult{}).ProtoReflect().Descriptor().Fields().ByName("Stderr")
	if stderrField == nil {
		t.Fatal("CrackQueryResult is missing field Stderr")
	}
	if got, want := stderrField.Number(), protoreflect.FieldNumber(6); got != want {
		t.Fatalf("CrackQueryResult.Stderr tag = %d, want %d", got, want)
	}

	want := &clientpb.CrackResponse{
		Job:      &clientpb.CrackJob{ID: "job-id"},
		Keyspace: "18446744073709551615",
		Query: &clientpb.CrackQueryResult{
			Mode: clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE, CrackstationHostUUID: "host-id", Value: "42", Stderr: "warning\n",
		},
	}
	raw, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal crack response: %v", err)
	}
	got := &clientpb.CrackResponse{}
	if err := proto.Unmarshal(raw, got); err != nil {
		t.Fatalf("unmarshal crack response: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("crack response round trip = %v, want %v", got, want)
	}
}

func TestConnectedCrackstationsReturnsRPCError(t *testing.T) {
	want := errors.New("query failed")
	capture := &crackDispatchRPCCapture{err: want}
	if _, err := connectedCrackstations(context.Background(), capture); !errors.Is(err, want) {
		t.Fatalf("connectedCrackstations() error = %v, want %v", err, want)
	}
	if capture.crackstationsCalls != 1 {
		t.Fatalf("Crackstations calls = %d, want 1", capture.crackstationsCalls)
	}
}

func TestConnectedCrackstationsReturnsEveryNonNilStation(t *testing.T) {
	capture := &crackDispatchRPCCapture{response: &clientpb.Crackstations{Crackstations: []*clientpb.Crackstation{
		nil,
		{Name: "one"},
		{Name: "two"},
	}}}
	stations, err := connectedCrackstations(context.Background(), capture)
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) != 2 || stations[0].Name != "one" || stations[1].Name != "two" {
		t.Fatalf("connectedCrackstations() = %#v, want one and two", stations)
	}
}

func TestCrackHashcatFlagsAreRootLocal(t *testing.T) {
	root := Commands(nil)[0]
	if root.Flags().Lookup("backend-info") == nil {
		t.Fatal("root crack command is missing --backend-info")
	}
	if root.Flags().Lookup("crackstation") == nil {
		t.Fatal("root crack command is missing --crackstation")
	}
	if root.PersistentFlags().Lookup("backend-info") != nil {
		t.Fatal("--backend-info must not be inherited by crack subcommands")
	}
	if root.Flags().Lookup("stdout") != nil {
		t.Fatal("root crack command unexpectedly exposes removed --stdout")
	}

	stations, _, err := root.Find([]string{"stations"})
	if err != nil {
		t.Fatalf("find crack stations: %v", err)
	}
	if stations.InheritedFlags().Lookup("backend-info") != nil {
		t.Fatal("crack stations unexpectedly inherits --backend-info")
	}
	if stations.InheritedFlags().Lookup("timeout") == nil {
		t.Fatal("crack stations must continue to inherit --timeout")
	}
}
