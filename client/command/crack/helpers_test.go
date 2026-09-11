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
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type crackFilesListCapture struct {
	rpcpb.SliverRPCClient
	requests []clientpb.CrackFileType
}

type crackJobsListCapture struct {
	rpcpb.SliverRPCClient
	jobs  *clientpb.CrackJobs
	err   error
	calls int
}

func (capture *crackJobsListCapture) CrackJobs(_ context.Context, _ *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.CrackJobs, error) {
	capture.calls++
	return capture.jobs, capture.err
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

func TestCrackJobCompleterRequestsDurableJobs(t *testing.T) {
	capture := &crackJobsListCapture{jobs: &clientpb.CrackJobs{Jobs: []*clientpb.CrackJob{{
		ID:          "12345678-1234-1234-1234-123456789abc",
		Status:      clientpb.CrackJobStatus_IN_PROGRESS,
		CreatedAt:   "2026-09-11T12:00:00Z",
		Keyspace:    "1000",
		Tasks:       []*clientpb.CrackTask{{ID: "task-id"}},
		ResultCount: 2,
	}}}}
	client := &console.SliverClient{Rpc: capture}
	_ = CrackJobIDCompleter(client).Invoke(carapace.NewContext(""))
	if capture.calls != 1 {
		t.Fatalf("CrackJobs calls = %d, want 1", capture.calls)
	}
}

func TestCrackJobCobraCompletionUsesFullIDsAndFiltersPrefix(t *testing.T) {
	firstID := "12345678-1234-1234-1234-123456789abc"
	secondID := "abcdefab-cdef-cdef-cdef-abcdefabcdef"
	capture := &crackJobsListCapture{jobs: &clientpb.CrackJobs{Jobs: []*clientpb.CrackJob{
		{
			ID:          firstID,
			Status:      clientpb.CrackJobStatus_IN_PROGRESS,
			CreatedAt:   "2026-09-11T12:00:00Z",
			Keyspace:    "1000",
			Tasks:       []*clientpb.CrackTask{{ID: "task-id"}},
			ResultCount: 2,
		},
		{ID: secondID, Status: clientpb.CrackJobStatus_COMPLETED},
		nil,
		{},
	}}}
	client := &console.SliverClient{Rpc: capture}
	root := Commands(client)[0]
	job, _, err := root.Find([]string{"job"})
	if err != nil {
		t.Fatalf("find crack job: %v", err)
	}

	values, directive := job.ValidArgsFunction(job, nil, "1234")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completion directive = %v, want no-file-completion", directive)
	}
	if len(values) != 1 {
		t.Fatalf("completion values = %#v, want one filtered result", values)
	}
	if !strings.HasPrefix(values[0], firstID+"\t") {
		t.Fatalf("completion = %q, want full ID %q", values[0], firstID)
	}
	for _, want := range []string{"IN_PROGRESS", "2026-09-11T12:00:00Z", "keyspace 1000", "1 task(s)", "2 result(s)"} {
		if !strings.Contains(values[0], want) {
			t.Errorf("completion %q does not contain %q", values[0], want)
		}
	}

	values, directive = job.ValidArgsFunction(job, []string{firstID}, "")
	if len(values) != 0 || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("second positional completion = %#v, %v", values, directive)
	}
}

func TestCrackJobCompletionHandlesRPCFailuresAndNilData(t *testing.T) {
	for _, test := range []struct {
		name    string
		capture *crackJobsListCapture
	}{
		{name: "rpc error", capture: &crackJobsListCapture{err: errors.New("rpc failed")}},
		{name: "nil response", capture: &crackJobsListCapture{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &console.SliverClient{Rpc: test.capture}
			root := Commands(client)[0]
			job, _, err := root.Find([]string{"job"})
			if err != nil {
				t.Fatalf("find crack job: %v", err)
			}
			values, directive := job.ValidArgsFunction(job, nil, "")
			if len(values) != 0 || directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("completion = %#v, %v", values, directive)
			}
		})
	}
}
