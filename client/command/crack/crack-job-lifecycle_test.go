package crack

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

func TestCrackJobLifecycleCommandsAreRegisteredAtExactHierarchy(t *testing.T) {
	root := Commands(nil)[0]
	for _, name := range []string{"cancel", "pause", "resume"} {
		t.Run(name, func(t *testing.T) {
			command, args, err := root.Find([]string{"job", name, "job-id"})
			if err != nil {
				t.Fatalf("find crack job %s: %v", name, err)
			}
			if got, want := command.CommandPath(), "crack job "+name; got != want {
				t.Fatalf("command path = %q, want %q", got, want)
			}
			if !reflect.DeepEqual(args, []string{"job-id"}) {
				t.Fatalf("remaining args = %#v, want [job-id]", args)
			}
			if got, want := command.Use, name+" [id]"; got != want {
				t.Fatalf("use = %q, want %q", got, want)
			}
			if err := command.Args(command, nil); err != nil {
				t.Fatalf("interactive selection rejected: %v", err)
			}
			if err := command.Args(command, []string{"job-id"}); err != nil {
				t.Fatalf("one job ID rejected: %v", err)
			}
			if err := command.Args(command, []string{"first", "second"}); err == nil {
				t.Fatal("more than one job ID was accepted")
			}
			if command.ValidArgsFunction == nil {
				t.Fatal("job ID completion is missing")
			}
			if command.InheritedFlags().Lookup("timeout") == nil {
				t.Fatal("--timeout is not inherited")
			}
			if command.InheritedFlags().Lookup("watch") != nil || command.InheritedFlags().Lookup("poll-interval") != nil {
				t.Fatal("crack job view flags leaked into lifecycle command")
			}
		})
	}
}

func TestCrackJobLifecycleCompletionOnlyOffersEligibleFullIDs(t *testing.T) {
	activeID := "aaaaaaaa-1111-4111-8111-111111111111"
	pausedID := "bbbbbbbb-2222-4222-8222-222222222222"
	completedID := "cccccccc-3333-4333-8333-333333333333"
	cancelledID := "dddddddd-4444-4444-8444-444444444444"

	tests := []struct {
		name string
		want []string
	}{
		{name: "cancel", want: []string{activeID, pausedID}},
		{name: "pause", want: []string{activeID}},
		{name: "resume", want: []string{pausedID}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capture := &crackJobsListCapture{jobs: &clientpb.CrackJobs{Jobs: []*clientpb.CrackJob{
				{ID: activeID, Status: clientpb.CrackJobStatus_IN_PROGRESS},
				{ID: pausedID, Status: clientpb.CrackJobStatus_PAUSED},
				{ID: completedID, Status: clientpb.CrackJobStatus_COMPLETED},
				{ID: cancelledID, Status: clientpb.CrackJobStatus_CANCELLED},
				nil,
				{},
			}}}
			root := Commands(&console.SliverClient{Rpc: capture})[0]
			command, _, err := root.Find([]string{"job", test.name})
			if err != nil {
				t.Fatalf("find crack job %s: %v", test.name, err)
			}

			values, directive := command.ValidArgsFunction(command, nil, "")
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("completion directive = %v, want no-file-completion", directive)
			}
			got := make([]string, 0, len(values))
			for _, value := range values {
				got = append(got, strings.SplitN(value, "\t", 2)[0])
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("completion IDs = %#v, want %#v", got, test.want)
			}
			if capture.calls != 1 {
				t.Fatalf("CrackJobs calls = %d, want 1", capture.calls)
			}
		})
	}
}

func TestCrackJobLifecycleEligibility(t *testing.T) {
	tests := []struct {
		name        string
		job         *clientpb.CrackJob
		pausable    bool
		resumable   bool
		cancellable bool
	}{
		{name: "nil"},
		{name: "in progress", job: &clientpb.CrackJob{Status: clientpb.CrackJobStatus_IN_PROGRESS}, pausable: true, cancellable: true},
		{name: "paused", job: &clientpb.CrackJob{Status: clientpb.CrackJobStatus_PAUSED}, resumable: true, cancellable: true},
		{name: "completed", job: &clientpb.CrackJob{Status: clientpb.CrackJobStatus_COMPLETED}},
		{name: "failed", job: &clientpb.CrackJob{Status: clientpb.CrackJobStatus_FAILED}},
		{name: "cancelled", job: &clientpb.CrackJob{Status: clientpb.CrackJobStatus_CANCELLED}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := crackJobPausable(test.job); got != test.pausable {
				t.Errorf("pausable = %v, want %v", got, test.pausable)
			}
			if got := crackJobResumable(test.job); got != test.resumable {
				t.Errorf("resumable = %v, want %v", got, test.resumable)
			}
			if got := crackJobCancellable(test.job); got != test.cancellable {
				t.Errorf("cancellable = %v, want %v", got, test.cancellable)
			}
		})
	}
}

func TestCrackJobLifecycleEmptySelectionMessagesAreActionSpecific(t *testing.T) {
	for _, test := range []struct {
		name   string
		action crackJobLifecycleAction
		want   string
	}{
		{name: "cancel", action: newCrackJobCancelAction(nil), want: "No cancellable crack jobs"},
		{name: "pause", action: newCrackJobPauseAction(nil), want: "No pausable crack jobs"},
		{name: "resume", action: newCrackJobResumeAction(nil), want: "No resumable crack jobs"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.action.noEligibleJobsMessage != test.want {
				t.Fatalf("empty-selection message = %q, want %q", test.action.noEligibleJobsMessage, test.want)
			}
		})
	}
}

func TestApplyCrackJobLifecycleActionsConfirmAndMutateByID(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "expected")
	jobID := "aaaaaaaa-1111-4111-8111-111111111111"

	tests := []struct {
		name             string
		status           clientpb.CrackJobStatus
		updatedStatus    clientpb.CrackJobStatus
		newAction        func(crackJobMutationFunc) crackJobLifecycleAction
		wantConfirmation bool
		promptParts      []string
	}{
		{
			name:             "cancel active",
			status:           clientpb.CrackJobStatus_IN_PROGRESS,
			updatedStatus:    clientpb.CrackJobStatus_CANCELLED,
			newAction:        newCrackJobCancelAction,
			wantConfirmation: true,
			promptParts:      []string{"Cancel crack job", jobID, "Unfinished work will be discarded"},
		},
		{
			name:             "cancel paused",
			status:           clientpb.CrackJobStatus_PAUSED,
			updatedStatus:    clientpb.CrackJobStatus_CANCELLED,
			newAction:        newCrackJobCancelAction,
			wantConfirmation: true,
			promptParts:      []string{"Cancel crack job", jobID, "PAUSED"},
		},
		{
			name:             "pause",
			status:           clientpb.CrackJobStatus_IN_PROGRESS,
			updatedStatus:    clientpb.CrackJobStatus_PAUSED,
			newAction:        newCrackJobPauseAction,
			wantConfirmation: true,
			promptParts:      []string{"Pause crack job", jobID, "stopped", "beginning of their shards when resumed"},
		},
		{
			name:          "resume",
			status:        clientpb.CrackJobStatus_PAUSED,
			updatedStatus: clientpb.CrackJobStatus_IN_PROGRESS,
			newAction:     newCrackJobResumeAction,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutateCalls := 0
			confirmCalls := 0
			action := test.newAction(func(gotCtx context.Context, request *clientpb.CrackJob) (*clientpb.CrackJob, error) {
				mutateCalls++
				if gotCtx.Value(contextKey{}) != "expected" {
					t.Fatal("mutation did not receive the caller context")
				}
				if request.GetID() != jobID || request.GetCommand() != nil || len(request.GetTasks()) != 0 || len(request.GetResults()) != 0 {
					t.Fatalf("mutation request = %#v, want ID-only request", request)
				}
				return &clientpb.CrackJob{ID: jobID, Status: test.updatedStatus, UpdatedAt: 1}, nil
			})
			updated, changed, err := applyCrackJobLifecycleAction(ctx, &clientpb.CrackJob{ID: "  " + jobID + "  ", Status: test.status}, action,
				func(prompt string) (bool, error) {
					confirmCalls++
					for _, part := range test.promptParts {
						if !strings.Contains(prompt, part) {
							t.Errorf("confirmation prompt %q does not contain %q", prompt, part)
						}
					}
					return true, nil
				})
			if err != nil {
				t.Fatalf("apply lifecycle action: %v", err)
			}
			if !changed || updated.GetID() != jobID || updated.GetStatus() != test.updatedStatus {
				t.Fatalf("updated=%#v changed=%v", updated, changed)
			}
			wantConfirmCalls := 0
			if test.wantConfirmation {
				wantConfirmCalls = 1
			}
			if confirmCalls != wantConfirmCalls || mutateCalls != 1 {
				t.Fatalf("confirm calls=%d mutate calls=%d, want %d/1", confirmCalls, mutateCalls, wantConfirmCalls)
			}
		})
	}
}

func TestApplyFetchedCrackJobLifecycleActionRejectsMismatchedIDBeforeConfirmation(t *testing.T) {
	requestedID := "aaaaaaaa-1111-4111-8111-111111111111"
	returnedID := "bbbbbbbb-2222-4222-8222-222222222222"
	confirmCalls := 0
	mutationCalls := 0
	action := newCrackJobPauseAction(func(context.Context, *clientpb.CrackJob) (*clientpb.CrackJob, error) {
		mutationCalls++
		return nil, nil
	})

	updated, changed, err := applyFetchedCrackJobLifecycleAction(
		t.Context(),
		requestedID,
		&clientpb.CrackJob{ID: returnedID, Status: clientpb.CrackJobStatus_IN_PROGRESS},
		action,
		func(string) (bool, error) {
			confirmCalls++
			return true, nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), requestedID) || !strings.Contains(err.Error(), returnedID) {
		t.Fatalf("mismatched detail error = %v, want both job IDs", err)
	}
	if updated != nil || changed || confirmCalls != 0 || mutationCalls != 0 {
		t.Fatalf("updated=%#v changed=%v confirm calls=%d mutation calls=%d, want nil/false/0/0", updated, changed, confirmCalls, mutationCalls)
	}
}

func TestApplyCrackJobLifecycleActionDeclineAndValidationErrorsDoNotMutate(t *testing.T) {
	sentinel := errors.New("sentinel")
	mutationCalls := 0
	mutation := func(context.Context, *clientpb.CrackJob) (*clientpb.CrackJob, error) {
		mutationCalls++
		return nil, sentinel
	}

	for _, test := range []struct {
		name    string
		job     *clientpb.CrackJob
		action  crackJobLifecycleAction
		confirm crackJobConfirmFunc
		wantErr string
	}{
		{
			name:    "cancel declined",
			job:     &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_IN_PROGRESS},
			action:  newCrackJobCancelAction(mutation),
			confirm: func(string) (bool, error) { return false, nil },
		},
		{
			name:    "pause confirmation unavailable",
			job:     &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_IN_PROGRESS},
			action:  newCrackJobPauseAction(mutation),
			wantErr: "confirmation is unavailable",
		},
		{
			name:    "cannot pause completed",
			job:     &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_COMPLETED},
			action:  newCrackJobPauseAction(mutation),
			confirm: func(string) (bool, error) { return true, nil },
			wantErr: "only in-progress jobs can be paused",
		},
		{
			name:    "cannot resume active",
			job:     &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_IN_PROGRESS},
			action:  newCrackJobResumeAction(mutation),
			wantErr: "only paused jobs can be resumed",
		},
		{
			name:    "cannot cancel terminal",
			job:     &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_CANCELLED},
			action:  newCrackJobCancelAction(mutation),
			confirm: func(string) (bool, error) { return true, nil },
			wantErr: "only in-progress or paused jobs can be cancelled",
		},
		{name: "empty job", action: newCrackJobResumeAction(mutation), wantErr: "empty crack job"},
		{name: "empty ID", job: &clientpb.CrackJob{Status: clientpb.CrackJobStatus_PAUSED}, action: newCrackJobResumeAction(mutation), wantErr: "ID cannot be empty"},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := mutationCalls
			updated, changed, err := applyCrackJobLifecycleAction(t.Context(), test.job, test.action, test.confirm)
			if updated != nil || changed {
				t.Fatalf("updated=%#v changed=%v, want nil/false", updated, changed)
			}
			if test.wantErr == "" && err != nil {
				t.Fatalf("error = %v, want nil", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
			if mutationCalls != before {
				t.Fatal("mutation ran despite decline or validation error")
			}
		})
	}
}

func TestApplyCrackJobLifecycleActionValidatesAuthoritativeResponse(t *testing.T) {
	job := &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_PAUSED}
	for _, test := range []struct {
		name      string
		result    *clientpb.CrackJob
		resultErr error
		wantErr   string
	}{
		{name: "RPC error", resultErr: errors.New("sentinel"), wantErr: "sentinel"},
		{name: "nil", wantErr: "empty crack job after resume"},
		{name: "empty ID", result: &clientpb.CrackJob{Status: clientpb.CrackJobStatus_IN_PROGRESS}, wantErr: "without an ID after resume"},
		{name: "wrong ID", result: &clientpb.CrackJob{ID: "another-job", Status: clientpb.CrackJobStatus_IN_PROGRESS, UpdatedAt: 1}, wantErr: "after resume request for job-id"},
		{name: "missing timestamp", result: &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_IN_PROGRESS}, wantErr: "without an update timestamp after resume"},
		{name: "wrong status", result: &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_PAUSED, UpdatedAt: 1}, wantErr: "unexpected crack job status PAUSED after resume"},
	} {
		t.Run(test.name, func(t *testing.T) {
			action := newCrackJobResumeAction(func(context.Context, *clientpb.CrackJob) (*clientpb.CrackJob, error) {
				return test.result, test.resultErr
			})
			updated, changed, err := applyCrackJobLifecycleAction(t.Context(), job, action, nil)
			if updated != nil || changed || err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("updated=%#v changed=%v error=%v, want error containing %q", updated, changed, err, test.wantErr)
			}
		})
	}
}

func TestCrackJobLifecycleMutationUsesCommandTimeout(t *testing.T) {
	root := Commands(nil)[0]
	pause, _, err := root.Find([]string{"job", "pause"})
	if err != nil {
		t.Fatalf("find crack job pause: %v", err)
	}
	if err := pause.InheritedFlags().Set("timeout", "2"); err != nil {
		t.Fatalf("set timeout: %v", err)
	}
	started := time.Now()
	var deadline time.Time
	action := newCrackJobPauseAction(func(parent context.Context, request *clientpb.CrackJob) (*clientpb.CrackJob, error) {
		ctx, cancel := crackCommandContext(parent, pause)
		defer cancel()
		var ok bool
		deadline, ok = ctx.Deadline()
		if !ok {
			t.Fatal("mutation context has no deadline")
		}
		return &clientpb.CrackJob{ID: request.GetID(), Status: clientpb.CrackJobStatus_PAUSED, UpdatedAt: 1}, nil
	})
	if _, changed, err := applyCrackJobLifecycleAction(t.Context(), &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_IN_PROGRESS}, action,
		func(string) (bool, error) { return true, nil }); err != nil || !changed {
		t.Fatalf("apply lifecycle action changed=%v error=%v", changed, err)
	}
	if deadline.Before(started.Add(1500*time.Millisecond)) || deadline.After(started.Add(2500*time.Millisecond)) {
		t.Fatalf("mutation deadline = %s, want about two seconds after %s", deadline, started)
	}
}
