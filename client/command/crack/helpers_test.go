package crack

import (
	"context"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/rsteube/carapace"
	"google.golang.org/grpc"
)

type crackFilesListCapture struct {
	rpcpb.SliverRPCClient
	requests []clientpb.CrackFileType
}

func (capture *crackFilesListCapture) CrackFilesList(_ context.Context, request *clientpb.CrackFile, _ ...grpc.CallOption) (*clientpb.CrackFiles, error) {
	capture.requests = append(capture.requests, request.GetType())
	return &clientpb.CrackFiles{}, nil
}

func TestCrackFileCompletersRequestTheirOwnFileType(t *testing.T) {
	tests := []struct {
		name   string
		want   clientpb.CrackFileType
		action func(*console.SliverClient) carapace.Action
	}{
		{name: "wordlists", want: clientpb.CrackFileType_WORDLIST, action: CrackWordlistCompleter},
		{name: "rules", want: clientpb.CrackFileType_RULES, action: CrackRulesCompleter},
		{name: "hcstat2", want: clientpb.CrackFileType_MARKOV_HCSTAT2, action: CrackHcstat2Completer},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := &crackFilesListCapture{}
			client := &console.SliverClient{Rpc: capture}
			_ = test.action(client).Invoke(carapace.NewContext(""))
			if len(capture.requests) != 1 || capture.requests[0] != test.want {
				t.Fatalf("completion requests = %v, want [%s]", capture.requests, test.want)
			}
		})
	}
}
