package crack

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/x/ansi"
)

const crackTopActionTestJobID = "dddddddd-4444-4444-8444-444444444444"

func newCrackTopJobActionTestModel(
	status clientpb.CrackJobStatus,
	actions map[crackTopJobAction]crackTopJobActionFunc,
) *crackTopModel {
	snapshot := &crackTopSnapshot{
		Jobs: []*clientpb.CrackJob{{
			ID:        crackTopActionTestJobID,
			Status:    status,
			UpdatedAt: 100,
			Tasks: []*clientpb.CrackTask{{
				ID:              "action-task",
				HostUUID:        "action-station",
				Kind:            clientpb.CrackTaskKind_CRACK_TASK_CRACK,
				State:           clientpb.CrackTaskState_CRACK_TASK_RUNNING,
				LastHeartbeatAt: 1_700_000_000,
			}},
		}},
		Stations: []*clientpb.Crackstation{{
			ID:       "action-station",
			HostUUID: "action-station",
			Status: &clientpb.CrackstationStatus{
				HostUUID:          "action-station",
				State:             clientpb.States_CRACKING,
				CurrentCrackJobID: crackTopActionTestJobID,
			},
		}},
		RefreshedAt: time.Unix(1_700_000_000, 0),
		taskTelemetry: map[string]crackTopTaskTelemetry{
			"action-task": {status: &crackStatusView{Devices: []crackDeviceView{{Speed: "100"}}}},
		},
	}
	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.jobActions = actions
	model.snapshot = snapshot
	model.dashboard = buildCrackTopDashboard(snapshot)
	model.refreshing = false
	model.width = 100
	model.height = 28
	model.normalizeJobSelection()
	return model
}

func singleCrackTopAction(action crackTopJobAction, actionJob crackTopJobActionFunc) map[crackTopJobAction]crackTopJobActionFunc {
	return map[crackTopJobAction]crackTopJobActionFunc{action: actionJob}
}

func TestCrackTopDeleteConfirmsAndRemovesTerminalJob(t *testing.T) {
	deletedID := ""
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_COMPLETED,
		singleCrackTopAction(crackTopJobActionDelete, func(_ context.Context, jobID string) (*clientpb.CrackJob, error) {
			deletedID = jobID
			return nil, nil
		}),
	)

	_, openCommand := model.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	if openCommand != nil || model.pendingJobAction != (crackTopPendingJobAction{action: crackTopJobActionDelete, jobID: crackTopActionTestJobID}) {
		t.Fatalf("delete prompt = command:%v prompt:%#v", openCommand != nil, model.pendingJobAction)
	}
	plain := ansi.Strip(model.View().Content)
	for _, expected := range []string{"Delete crack job?", crackTopActionTestJobID, "This cannot be undone", "y delete"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("delete confirmation is missing %q:\n%s", expected, plain)
		}
	}
	if got := lipgloss.Width(model.View().Content); got > model.width {
		t.Fatalf("delete modal width = %d, want <= %d", got, model.width)
	}

	_, deleteCommand := model.Update(tea.KeyPressMsg{Text: "y", Code: 'y'})
	if deleteCommand == nil || model.pendingJobAction.jobID != "" || model.runningJobAction.jobID != crackTopActionTestJobID {
		t.Fatalf("confirmed delete = command:%v pending:%#v running:%#v", deleteCommand != nil, model.pendingJobAction, model.runningJobAction)
	}
	rawMessage := deleteCommand()
	message, ok := rawMessage.(crackTopJobActionCompletedMsg)
	if !ok {
		t.Fatalf("delete command returned %T, want crackTopJobActionCompletedMsg", rawMessage)
	}
	if deletedID != crackTopActionTestJobID || message.action != crackTopJobActionDelete || message.jobID != crackTopActionTestJobID || message.err != nil {
		t.Fatalf("delete call/message = called:%q message:%#v", deletedID, message)
	}

	_, refreshCommand := model.Update(message)
	if refreshCommand == nil || !model.refreshing {
		t.Fatalf("successful delete refresh = command:%v refreshing:%v", refreshCommand != nil, model.refreshing)
	}
	if len(model.snapshot.Jobs) != 0 || len(model.dashboard.Jobs) != 0 || model.selectedJobID != "" {
		t.Fatalf("deleted job remains in dashboard: snapshot=%#v dashboard=%#v selected=%q", model.snapshot.Jobs, model.dashboard.Jobs, model.selectedJobID)
	}
	if _, ok := model.snapshot.taskTelemetry["action-task"]; ok {
		t.Fatal("deleted job telemetry remains cached")
	}
	if model.toastLevel != "success" || !strings.Contains(model.toast, "Deleted crack job dddddddd") {
		t.Fatalf("success toast = level:%q text:%q", model.toastLevel, model.toast)
	}
}

func TestCrackTopDeleteRejectsInProgressJob(t *testing.T) {
	deleteCalls := 0
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_IN_PROGRESS,
		singleCrackTopAction(crackTopJobActionDelete, func(context.Context, string) (*clientpb.CrackJob, error) {
			deleteCalls++
			return nil, nil
		}),
	)

	_, command := model.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	if command == nil {
		t.Fatal("in-progress delete did not schedule a warning")
	}
	if model.pendingJobAction.jobID != "" || model.runningJobAction.jobID != "" || deleteCalls != 0 {
		t.Fatalf("in-progress delete changed state: pending=%#v running=%#v calls=%d", model.pendingJobAction, model.runningJobAction, deleteCalls)
	}
	if model.toastLevel != "warning" || !strings.Contains(model.toast, "Only completed, failed, or cancelled crack jobs can be deleted") {
		t.Fatalf("in-progress warning = level:%q text:%q", model.toastLevel, model.toast)
	}
}

func TestCrackTopDeleteFailureRetainsJobAndEscapeCancels(t *testing.T) {
	wantErr := errors.New("database is busy\ntry later")
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_FAILED,
		singleCrackTopAction(crackTopJobActionDelete, func(context.Context, string) (*clientpb.CrackJob, error) {
			return nil, wantErr
		}),
	)

	_, _ = model.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	_, cancelCommand := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cancelCommand != nil || model.pendingJobAction.jobID != "" || model.runningJobAction.jobID != "" {
		t.Fatalf("escaped delete prompt = command:%v pending=%#v running=%#v", cancelCommand != nil, model.pendingJobAction, model.runningJobAction)
	}

	_, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	_, enterCommand := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if enterCommand != nil || model.pendingJobAction.jobID != crackTopActionTestJobID || model.runningJobAction.jobID != "" {
		t.Fatalf("bare enter confirmed destructive action: command=%v pending=%#v running=%#v", enterCommand != nil, model.pendingJobAction, model.runningJobAction)
	}
	_, deleteCommand := model.Update(tea.KeyPressMsg{Text: "y", Code: 'y'})
	if deleteCommand == nil {
		t.Fatal("y did not confirm deletion")
	}
	message := deleteCommand().(crackTopJobActionCompletedMsg)
	_, toastCommand := model.Update(message)
	if toastCommand == nil || model.refreshing {
		t.Fatalf("failed delete = toast command:%v refreshing:%v", toastCommand != nil, model.refreshing)
	}
	if len(model.snapshot.Jobs) != 1 || model.snapshot.Jobs[0].GetID() != crackTopActionTestJobID {
		t.Fatalf("failed delete removed job: %#v", model.snapshot.Jobs)
	}
	if model.toastLevel != "error" || !strings.Contains(model.toast, "database is busy try later") || strings.Contains(model.toast, "\n") {
		t.Fatalf("failure toast = level:%q text:%q", model.toastLevel, model.toast)
	}
}

func TestCrackTopResizeBelowMinimumCancelsHiddenConfirmation(t *testing.T) {
	deleteCalls := 0
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_COMPLETED,
		singleCrackTopAction(crackTopJobActionDelete, func(context.Context, string) (*clientpb.CrackJob, error) {
			deleteCalls++
			return nil, nil
		}),
	)

	_, _ = model.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	if model.pendingJobAction.jobID == "" {
		t.Fatal("delete confirmation did not open")
	}
	_, resizeCommand := model.Update(tea.WindowSizeMsg{Width: crackTopMinWidth - 1, Height: crackTopMinHeight})
	if resizeCommand != nil || model.pendingJobAction.jobID != "" {
		t.Fatalf("small resize left hidden confirmation actionable: command=%v pending=%#v", resizeCommand != nil, model.pendingJobAction)
	}
	_, confirmCommand := model.Update(tea.KeyPressMsg{Text: "y", Code: 'y'})
	if confirmCommand != nil || model.runningJobAction.jobID != "" || deleteCalls != 0 {
		t.Fatalf("hidden confirmation executed: command=%v running=%#v calls=%d", confirmCommand != nil, model.runningJobAction, deleteCalls)
	}
}

func TestCrackTopBelowMinimumCannotOpenHiddenConfirmation(t *testing.T) {
	deleteCalls := 0
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_COMPLETED,
		singleCrackTopAction(crackTopJobActionDelete, func(context.Context, string) (*clientpb.CrackJob, error) {
			deleteCalls++
			return nil, nil
		}),
	)
	_, _ = model.Update(tea.WindowSizeMsg{Width: crackTopMinWidth, Height: crackTopMinHeight - 1})

	_, openCommand := model.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	_, confirmCommand := model.Update(tea.KeyPressMsg{Text: "y", Code: 'y'})
	if openCommand != nil || confirmCommand != nil || model.pendingJobAction.jobID != "" || model.runningJobAction.jobID != "" || deleteCalls != 0 {
		t.Fatalf("undersized dashboard accepted hidden delete: open=%v confirm=%v pending=%#v running=%#v calls=%d",
			openCommand != nil, confirmCommand != nil, model.pendingJobAction, model.runningJobAction, deleteCalls)
	}
}

func TestCrackTopDeleteAllowsEveryTerminalStatus(t *testing.T) {
	for _, status := range []clientpb.CrackJobStatus{
		clientpb.CrackJobStatus_COMPLETED,
		clientpb.CrackJobStatus_FAILED,
		clientpb.CrackJobStatus_CANCELLED,
	} {
		t.Run(status.String(), func(t *testing.T) {
			model := newCrackTopJobActionTestModel(
				status,
				singleCrackTopAction(crackTopJobActionDelete, func(context.Context, string) (*clientpb.CrackJob, error) { return nil, nil }),
			)
			_, command := model.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
			if command != nil || model.pendingJobAction.jobID != crackTopActionTestJobID {
				t.Fatalf("terminal status %s did not open delete confirmation", status)
			}
		})
	}
}

func TestCrackTopDeleteSuppressesStaleRefreshUntilServerOmitsJob(t *testing.T) {
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_COMPLETED,
		singleCrackTopAction(crackTopJobActionDelete, func(context.Context, string) (*clientpb.CrackJob, error) { return nil, nil }),
	)
	model.removeJob(crackTopActionTestJobID)

	stale := &crackTopSnapshot{
		Jobs:          []*clientpb.CrackJob{{ID: crackTopActionTestJobID, Status: clientpb.CrackJobStatus_COMPLETED, Tasks: []*clientpb.CrackTask{{ID: "stale-task"}}}},
		RefreshedAt:   time.Unix(1_700_000_001, 0),
		taskTelemetry: map[string]crackTopTaskTelemetry{"stale-task": {}},
	}
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: stale})
	if len(model.snapshot.Jobs) != 0 || len(model.dashboard.Jobs) != 0 {
		t.Fatalf("stale refresh restored deleted job: snapshot=%#v dashboard=%#v", model.snapshot.Jobs, model.dashboard.Jobs)
	}
	if _, ok := model.snapshot.taskTelemetry["stale-task"]; ok {
		t.Fatal("stale refresh retained deleted job telemetry")
	}
	if _, ok := model.deletedJobIDs[crackTopActionTestJobID]; !ok {
		t.Fatal("stale refresh cleared the deletion tombstone")
	}

	fresh := &crackTopSnapshot{RefreshedAt: time.Unix(1_700_000_002, 0)}
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: fresh})
	if _, ok := model.deletedJobIDs[crackTopActionTestJobID]; ok {
		t.Fatal("server-confirmed refresh did not clear the deletion tombstone")
	}
}

func TestCrackTopPauseConfirmsAppliesAuthoritativeStatusAndRefreshes(t *testing.T) {
	pausedID := ""
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_IN_PROGRESS,
		singleCrackTopAction(crackTopJobActionPause, func(_ context.Context, jobID string) (*clientpb.CrackJob, error) {
			pausedID = jobID
			return &clientpb.CrackJob{ID: jobID, Status: clientpb.CrackJobStatus_PAUSED, UpdatedAt: 200}, nil
		}),
	)

	_, openCommand := model.Update(tea.KeyPressMsg{Text: "p", Code: 'p'})
	if openCommand != nil || model.pendingJobAction != (crackTopPendingJobAction{action: crackTopJobActionPause, jobID: crackTopActionTestJobID}) {
		t.Fatalf("pause prompt = command:%v prompt:%#v", openCommand != nil, model.pendingJobAction)
	}
	plain := ansi.Strip(model.View().Content)
	for _, expected := range []string{"Pause crack job?", "release its current crackstation work", "may restart", "y pause"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("pause confirmation is missing %q:\n%s", expected, plain)
		}
	}
	_, enterCommand := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if enterCommand != nil || model.runningJobAction.jobID != "" {
		t.Fatal("enter confirmed pause without an explicit y")
	}

	_, pauseCommand := model.Update(tea.KeyPressMsg{Text: "y", Code: 'y'})
	if pauseCommand == nil || model.runningJobAction.action != crackTopJobActionPause {
		t.Fatalf("pause confirmation = command:%v running:%#v", pauseCommand != nil, model.runningJobAction)
	}
	message := pauseCommand().(crackTopJobActionCompletedMsg)
	if pausedID != crackTopActionTestJobID || message.status != clientpb.CrackJobStatus_PAUSED || !message.hasStatus {
		t.Fatalf("pause call/message = called:%q message:%#v", pausedID, message)
	}
	_, refreshCommand := model.Update(message)
	if refreshCommand == nil || !model.refreshing {
		t.Fatalf("successful pause refresh = command:%v refreshing:%v", refreshCommand != nil, model.refreshing)
	}
	if got := model.snapshot.Jobs[0].GetStatus(); got != clientpb.CrackJobStatus_PAUSED {
		t.Fatalf("local pause status = %s, want PAUSED", got)
	}
	if model.dashboard.Jobs[0].RateKnown || model.dashboard.ActiveJobs != 0 {
		t.Fatalf("paused job retained active throughput: row=%#v active=%d", model.dashboard.Jobs[0], model.dashboard.ActiveJobs)
	}
	if model.toastLevel != "success" || !strings.Contains(model.toast, "Paused crack job dddddddd") {
		t.Fatalf("pause toast = level:%q text:%q", model.toastLevel, model.toast)
	}
}

func TestCrackTopCancelConfirmsActiveAndPausedJobs(t *testing.T) {
	for _, status := range []clientpb.CrackJobStatus{clientpb.CrackJobStatus_IN_PROGRESS, clientpb.CrackJobStatus_PAUSED} {
		t.Run(status.String(), func(t *testing.T) {
			model := newCrackTopJobActionTestModel(
				status,
				singleCrackTopAction(crackTopJobActionCancel, func(_ context.Context, jobID string) (*clientpb.CrackJob, error) {
					return &clientpb.CrackJob{ID: jobID, Status: clientpb.CrackJobStatus_CANCELLED, UpdatedAt: 200}, nil
				}),
			)
			_, openCommand := model.Update(tea.KeyPressMsg{Text: "c", Code: 'c'})
			if openCommand != nil || model.pendingJobAction.action != crackTopJobActionCancel {
				t.Fatalf("cancel prompt = command:%v prompt:%#v", openCommand != nil, model.pendingJobAction)
			}
			plain := ansi.Strip(model.View().Content)
			for _, expected := range []string{"Cancel crack job?", "Cancelled jobs cannot be resumed", "y cancel"} {
				if !strings.Contains(plain, expected) {
					t.Fatalf("cancel confirmation is missing %q:\n%s", expected, plain)
				}
			}
			_, cancelCommand := model.Update(tea.KeyPressMsg{Text: "y", Code: 'y'})
			message := cancelCommand().(crackTopJobActionCompletedMsg)
			_, refreshCommand := model.Update(message)
			if refreshCommand == nil || model.snapshot.Jobs[0].GetStatus() != clientpb.CrackJobStatus_CANCELLED {
				t.Fatalf("cancel result = command:%v job:%#v", refreshCommand != nil, model.snapshot.Jobs[0])
			}
		})
	}
}

func TestCrackTopResumePausedJobWithoutDestructiveConfirmation(t *testing.T) {
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_PAUSED,
		singleCrackTopAction(crackTopJobActionResume, func(_ context.Context, jobID string) (*clientpb.CrackJob, error) {
			return &clientpb.CrackJob{ID: jobID, Status: clientpb.CrackJobStatus_IN_PROGRESS, UpdatedAt: 300}, nil
		}),
	)

	_, resumeCommand := model.Update(tea.KeyPressMsg{Text: "u", Code: 'u'})
	if resumeCommand == nil || model.pendingJobAction.jobID != "" || model.runningJobAction.action != crackTopJobActionResume {
		t.Fatalf("resume start = command:%v pending:%#v running:%#v", resumeCommand != nil, model.pendingJobAction, model.runningJobAction)
	}
	plain := strings.ToLower(ansi.Strip(model.View().Content))
	if !strings.Contains(plain, "resuming crack job dddddddd") {
		t.Fatalf("resume progress footer missing:\n%s", plain)
	}
	message := resumeCommand().(crackTopJobActionCompletedMsg)
	_, refreshCommand := model.Update(message)
	if refreshCommand == nil || model.snapshot.Jobs[0].GetStatus() != clientpb.CrackJobStatus_IN_PROGRESS {
		t.Fatalf("resume result = command:%v job:%#v", refreshCommand != nil, model.snapshot.Jobs[0])
	}
	if model.toastLevel != "success" || !strings.Contains(model.toast, "Resumed crack job dddddddd") {
		t.Fatalf("resume toast = level:%q text:%q", model.toastLevel, model.toast)
	}
}

func TestCrackTopLifecycleActionsRejectIneligibleStatuses(t *testing.T) {
	tests := []struct {
		name   string
		key    rune
		action crackTopJobAction
		status clientpb.CrackJobStatus
		want   string
	}{
		{name: "pause paused", key: 'p', action: crackTopJobActionPause, status: clientpb.CrackJobStatus_PAUSED, want: "Only in-progress crack jobs can be paused"},
		{name: "resume active", key: 'u', action: crackTopJobActionResume, status: clientpb.CrackJobStatus_IN_PROGRESS, want: "Only paused crack jobs can be resumed"},
		{name: "cancel complete", key: 'c', action: crackTopJobActionCancel, status: clientpb.CrackJobStatus_COMPLETED, want: "Only in-progress or paused crack jobs can be cancelled"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			model := newCrackTopJobActionTestModel(
				test.status,
				singleCrackTopAction(test.action, func(context.Context, string) (*clientpb.CrackJob, error) {
					calls++
					return nil, nil
				}),
			)
			_, command := model.Update(tea.KeyPressMsg{Text: string(test.key), Code: test.key})
			if command == nil || calls != 0 || model.pendingJobAction.jobID != "" || model.runningJobAction.jobID != "" {
				t.Fatalf("ineligible action = command:%v calls:%d pending:%#v running:%#v", command != nil, calls, model.pendingJobAction, model.runningJobAction)
			}
			if model.toastLevel != "warning" || !strings.Contains(model.toast, test.want) {
				t.Fatalf("ineligible toast = level:%q text:%q", model.toastLevel, model.toast)
			}
		})
	}
}

func TestCrackTopLifecycleRejectsNonAuthoritativeSuccessResponse(t *testing.T) {
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_PAUSED,
		singleCrackTopAction(crackTopJobActionResume, func(context.Context, string) (*clientpb.CrackJob, error) {
			return nil, nil
		}),
	)

	_, resumeCommand := model.Update(tea.KeyPressMsg{Text: "u", Code: 'u'})
	message := resumeCommand().(crackTopJobActionCompletedMsg)
	if message.err == nil || !strings.Contains(message.err.Error(), "no job state") {
		t.Fatalf("empty resume response error = %v", message.err)
	}
	_, toastCommand := model.Update(message)
	if toastCommand == nil || model.refreshing || model.snapshot.Jobs[0].GetStatus() != clientpb.CrackJobStatus_PAUSED {
		t.Fatalf("empty resume response mutated state: command=%v refreshing=%v job=%#v", toastCommand != nil, model.refreshing, model.snapshot.Jobs[0])
	}
	if model.toastLevel != "error" || !strings.Contains(model.toast, "server returned no job state") {
		t.Fatalf("empty resume toast = level:%q text:%q", model.toastLevel, model.toast)
	}
}

func TestCrackTopLifecycleRejectsIDOnlyResumeResponse(t *testing.T) {
	model := newCrackTopJobActionTestModel(
		clientpb.CrackJobStatus_PAUSED,
		singleCrackTopAction(crackTopJobActionResume, func(_ context.Context, jobID string) (*clientpb.CrackJob, error) {
			return &clientpb.CrackJob{ID: jobID}, nil
		}),
	)

	_, resumeCommand := model.Update(tea.KeyPressMsg{Text: "u", Code: 'u'})
	message := resumeCommand().(crackTopJobActionCompletedMsg)
	if message.err == nil || !strings.Contains(message.err.Error(), "without an update timestamp") {
		t.Fatalf("ID-only resume response error = %v", message.err)
	}
	_, toastCommand := model.Update(message)
	if toastCommand == nil || model.refreshing || model.snapshot.Jobs[0].GetStatus() != clientpb.CrackJobStatus_PAUSED {
		t.Fatalf("ID-only resume response mutated state: command=%v refreshing=%v job=%#v", toastCommand != nil, model.refreshing, model.snapshot.Jobs[0])
	}
}

func TestCrackTopLifecycleStatusOverrideSuppressesStaleRefresh(t *testing.T) {
	model := newCrackTopJobActionTestModel(clientpb.CrackJobStatus_IN_PROGRESS, nil)
	model.overrideJobStatus(crackTopActionTestJobID, clientpb.CrackJobStatus_PAUSED, 200)

	stale := &crackTopSnapshot{
		Jobs:          []*clientpb.CrackJob{{ID: crackTopActionTestJobID, Status: clientpb.CrackJobStatus_IN_PROGRESS, UpdatedAt: 100}},
		RefreshedAt:   time.Unix(1_700_000_001, 0),
		taskTelemetry: map[string]crackTopTaskTelemetry{},
	}
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: stale})
	if got := model.snapshot.Jobs[0].GetStatus(); got != clientpb.CrackJobStatus_PAUSED {
		t.Fatalf("stale refresh status = %s, want PAUSED", got)
	}
	if _, ok := model.jobStatusOverrides[crackTopActionTestJobID]; !ok {
		t.Fatal("stale refresh cleared authoritative status override")
	}

	observed := &crackTopSnapshot{
		Jobs:          []*clientpb.CrackJob{{ID: crackTopActionTestJobID, Status: clientpb.CrackJobStatus_PAUSED, UpdatedAt: 200}},
		RefreshedAt:   time.Unix(1_700_000_002, 0),
		taskTelemetry: map[string]crackTopTaskTelemetry{},
	}
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: observed})
	if _, ok := model.jobStatusOverrides[crackTopActionTestJobID]; ok {
		t.Fatal("server-confirmed refresh did not clear status override")
	}
}

func TestCrackTopLifecycleStatusOverrideYieldsToRepeatedSameSecondConflict(t *testing.T) {
	model := newCrackTopJobActionTestModel(clientpb.CrackJobStatus_IN_PROGRESS, nil)
	model.overrideJobStatus(crackTopActionTestJobID, clientpb.CrackJobStatus_PAUSED, 200)

	conflictingSnapshot := func(refreshedAt int64) *crackTopSnapshot {
		return &crackTopSnapshot{
			Jobs: []*clientpb.CrackJob{{
				ID:        crackTopActionTestJobID,
				Status:    clientpb.CrackJobStatus_CANCELLED,
				UpdatedAt: 200,
			}},
			RefreshedAt:   time.Unix(refreshedAt, 0),
			taskTelemetry: map[string]crackTopTaskTelemetry{},
		}
	}

	_, _ = model.Update(crackTopSnapshotMsg{snapshot: conflictingSnapshot(1_700_000_001)})
	if got := model.snapshot.Jobs[0].GetStatus(); got != clientpb.CrackJobStatus_PAUSED {
		t.Fatalf("first same-second conflict status = %s, want PAUSED stale-snapshot mask", got)
	}
	override, ok := model.jobStatusOverrides[crackTopActionTestJobID]
	if !ok || !override.equalVersionDisagreementSeen {
		t.Fatalf("first same-second conflict override = %#v, present=%v", override, ok)
	}

	_, _ = model.Update(crackTopSnapshotMsg{snapshot: conflictingSnapshot(1_700_000_002)})
	if got := model.snapshot.Jobs[0].GetStatus(); got != clientpb.CrackJobStatus_CANCELLED {
		t.Fatalf("repeated same-second conflict status = %s, want CANCELLED", got)
	}
	if _, ok := model.jobStatusOverrides[crackTopActionTestJobID]; ok {
		t.Fatal("repeated same-second conflict did not release status override")
	}
}

func TestCrackTopActiveFilterRetainsPausedJob(t *testing.T) {
	model := newCrackTopJobActionTestModel(clientpb.CrackJobStatus_PAUSED, nil)
	model.filter = crackTopFilterActive
	model.normalizeJobSelection()

	jobs := model.filteredJobs()
	if len(jobs) != 1 || jobs[0].ID != crackTopActionTestJobID || model.selectedJobID != crackTopActionTestJobID {
		t.Fatalf("active filter lost paused job: jobs=%#v selected=%q", jobs, model.selectedJobID)
	}
	if model.dashboard.ActiveJobs != 0 {
		t.Fatalf("paused job counted as active throughput work: %d", model.dashboard.ActiveJobs)
	}
}
