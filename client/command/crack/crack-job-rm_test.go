package crack

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

func TestCrackJobRmCommandIsRegisteredAtExactHierarchy(t *testing.T) {
	root := Commands(nil)[0]
	rm, args, err := root.Find([]string{"job", "rm", "job-id"})
	if err != nil {
		t.Fatalf("find crack job rm: %v", err)
	}
	if got := rm.CommandPath(); got != "crack job rm" {
		t.Fatalf("command path = %q, want %q", got, "crack job rm")
	}
	if !reflect.DeepEqual(args, []string{"job-id"}) {
		t.Fatalf("remaining args = %#v, want [job-id]", args)
	}
	if rm.Use != "rm [id]" {
		t.Fatalf("crack job rm use = %q, want %q", rm.Use, "rm [id]")
	}
	if err := rm.Args(rm, nil); err != nil {
		t.Fatalf("crack job rm rejected interactive selection: %v", err)
	}
	if err := rm.Args(rm, []string{"job-id"}); err != nil {
		t.Fatalf("crack job rm rejected one job ID: %v", err)
	}
	if err := rm.Args(rm, []string{"first", "second"}); err == nil {
		t.Fatal("crack job rm accepted more than one job ID")
	}
	if rm.ValidArgsFunction == nil {
		t.Fatal("crack job rm is missing ID completion")
	}
	if rm.InheritedFlags().Lookup("timeout") == nil {
		t.Fatal("crack job rm must inherit --timeout")
	}
	if rm.InheritedFlags().Lookup("watch") != nil || rm.InheritedFlags().Lookup("poll-interval") != nil {
		t.Fatal("crack job view flags leaked into crack job rm")
	}
}

func TestCrackJobRmCompletionOnlyOffersTerminalJobs(t *testing.T) {
	activeID := "aaaaaaaa-1111-4111-8111-111111111111"
	completedID := "bbbbbbbb-2222-4222-8222-222222222222"
	capture := &crackJobsListCapture{jobs: &clientpb.CrackJobs{Jobs: []*clientpb.CrackJob{
		{ID: activeID, Status: clientpb.CrackJobStatus_IN_PROGRESS},
		{ID: completedID, Status: clientpb.CrackJobStatus_COMPLETED},
	}}}
	root := Commands(&console.SliverClient{Rpc: capture})[0]
	rm, _, err := root.Find([]string{"job", "rm"})
	if err != nil {
		t.Fatalf("find crack job rm: %v", err)
	}

	values, directive := rm.ValidArgsFunction(rm, nil, "")
	if len(values) != 1 || !strings.HasPrefix(values[0], completedID+"\t") {
		t.Fatalf("completion values = %#v, want only terminal job %q", values, completedID)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completion directive = %v, want no-file-completion", directive)
	}
	if capture.calls != 1 {
		t.Fatalf("CrackJobs calls = %d, want 1", capture.calls)
	}
}

func TestRemoveCrackJobConfirmsCascadeAndDeletesByID(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "expected")
	job := &clientpb.CrackJob{
		ID:          "  aaaaaaaa-1111-4111-8111-111111111111  ",
		Status:      clientpb.CrackJobStatus_COMPLETED,
		Tasks:       []*clientpb.CrackTask{{ID: "one"}, {ID: "two"}},
		ResultCount: 1,
	}
	confirmCalls := 0
	deleteCalls := 0
	removed, err := removeCrackJob(ctx, job, func(prompt string) (bool, error) {
		confirmCalls++
		want := "Permanently delete crack job \"aaaaaaaa-1111-4111-8111-111111111111\" (COMPLETED), including 2 tasks and 1 result?"
		if prompt != want {
			t.Fatalf("confirmation prompt = %q, want %q", prompt, want)
		}
		return true, nil
	}, func(gotCtx context.Context, request *clientpb.CrackJob) (*commonpb.Empty, error) {
		deleteCalls++
		if gotCtx.Value(contextKey{}) != "expected" {
			t.Fatal("delete did not receive the caller context")
		}
		if request.GetID() != "aaaaaaaa-1111-4111-8111-111111111111" || request.GetCommand() != nil || len(request.GetTasks()) != 0 || len(request.GetResults()) != 0 {
			t.Fatalf("delete request = %#v, want ID-only request", request)
		}
		return &commonpb.Empty{}, nil
	})
	if err != nil {
		t.Fatalf("removeCrackJob: %v", err)
	}
	if !removed || confirmCalls != 1 || deleteCalls != 1 {
		t.Fatalf("removed=%v confirm calls=%d delete calls=%d, want true/1/1", removed, confirmCalls, deleteCalls)
	}
}

func TestRemoveFetchedCrackJobRejectsMismatchedIDBeforeConfirmation(t *testing.T) {
	requestedID := "aaaaaaaa-1111-4111-8111-111111111111"
	returnedID := "bbbbbbbb-2222-4222-8222-222222222222"
	confirmCalls := 0
	deleteCalls := 0

	removed, err := removeFetchedCrackJob(
		t.Context(),
		requestedID,
		&clientpb.CrackJob{ID: returnedID, Status: clientpb.CrackJobStatus_COMPLETED},
		func(string) (bool, error) {
			confirmCalls++
			return true, nil
		},
		func(context.Context, *clientpb.CrackJob) (*commonpb.Empty, error) {
			deleteCalls++
			return &commonpb.Empty{}, nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), requestedID) || !strings.Contains(err.Error(), returnedID) {
		t.Fatalf("mismatched detail error = %v, want both job IDs", err)
	}
	if removed || confirmCalls != 0 || deleteCalls != 0 {
		t.Fatalf("removed=%v confirm calls=%d delete calls=%d, want false/0/0", removed, confirmCalls, deleteCalls)
	}
}

func TestRemoveCrackJobDeclineAndErrorsDoNotReportRemoval(t *testing.T) {
	sentinel := errors.New("sentinel")
	tests := []struct {
		name        string
		job         *clientpb.CrackJob
		confirm     crackJobConfirmFunc
		deleteJob   crackJobDeleteFunc
		wantErr     string
		wantConfirm int
		wantDelete  int
	}{
		{
			name:    "active job rejected before confirmation",
			job:     &clientpb.CrackJob{ID: "active", Status: clientpb.CrackJobStatus_IN_PROGRESS},
			wantErr: "only completed, failed, or cancelled jobs can be removed",
		},
		{
			name:        "declined",
			job:         &clientpb.CrackJob{ID: "completed", Status: clientpb.CrackJobStatus_COMPLETED},
			confirm:     func(string) (bool, error) { return false, nil },
			wantConfirm: 1,
		},
		{
			name:        "confirmation error",
			job:         &clientpb.CrackJob{ID: "failed", Status: clientpb.CrackJobStatus_FAILED},
			confirm:     func(string) (bool, error) { return false, sentinel },
			wantErr:     "sentinel",
			wantConfirm: 1,
		},
		{
			name:        "delete error",
			job:         &clientpb.CrackJob{ID: "cancelled", Status: clientpb.CrackJobStatus_CANCELLED},
			confirm:     func(string) (bool, error) { return true, nil },
			deleteJob:   func(context.Context, *clientpb.CrackJob) (*commonpb.Empty, error) { return nil, sentinel },
			wantErr:     "sentinel",
			wantConfirm: 1,
			wantDelete:  1,
		},
		{
			name:    "empty response",
			wantErr: "server returned an empty crack job",
		},
		{
			name:    "empty ID",
			job:     &clientpb.CrackJob{Status: clientpb.CrackJobStatus_COMPLETED},
			wantErr: "crack job ID cannot be empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			confirmCalls := 0
			deleteCalls := 0
			confirm := test.confirm
			if confirm == nil {
				confirm = func(string) (bool, error) {
					confirmCalls++
					return true, nil
				}
			} else {
				wrapped := confirm
				confirm = func(prompt string) (bool, error) {
					confirmCalls++
					return wrapped(prompt)
				}
			}
			deleteJob := test.deleteJob
			if deleteJob == nil {
				deleteJob = func(context.Context, *clientpb.CrackJob) (*commonpb.Empty, error) {
					deleteCalls++
					return &commonpb.Empty{}, nil
				}
			} else {
				wrapped := deleteJob
				deleteJob = func(ctx context.Context, job *clientpb.CrackJob) (*commonpb.Empty, error) {
					deleteCalls++
					return wrapped(ctx, job)
				}
			}

			removed, err := removeCrackJob(t.Context(), test.job, confirm, deleteJob)
			if removed {
				t.Fatal("removeCrackJob reported removal")
			}
			if test.wantErr == "" && err != nil {
				t.Fatalf("removeCrackJob error = %v, want nil", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("removeCrackJob error = %v, want substring %q", err, test.wantErr)
			}
			if confirmCalls != test.wantConfirm || deleteCalls != test.wantDelete {
				t.Fatalf("confirm calls=%d delete calls=%d, want %d/%d", confirmCalls, deleteCalls, test.wantConfirm, test.wantDelete)
			}
		})
	}
}

func TestTerminalCrackJobsFiltersWithoutReordering(t *testing.T) {
	jobs := []*clientpb.CrackJob{
		{ID: "active", Status: clientpb.CrackJobStatus_IN_PROGRESS},
		nil,
		{ID: "failed", Status: clientpb.CrackJobStatus_FAILED},
		{ID: "completed", Status: clientpb.CrackJobStatus_COMPLETED},
		{ID: "cancelled", Status: clientpb.CrackJobStatus_CANCELLED},
	}
	got := terminalCrackJobs(jobs)
	want := []*clientpb.CrackJob{jobs[2], jobs[3], jobs[4]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("terminal jobs = %#v, want %#v", got, want)
	}
}
