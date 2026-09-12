package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type standaloneKeyspaceTestResult struct {
	keyspace string
	job      *clientpb.CrackJob
	query    *clientpb.CrackQueryResult
	err      error
}

func resetStandaloneCrackKeyspaceTasksForTest(t *testing.T) {
	t.Helper()
	cleanup := func() {
		crackQueueMu.Lock()
		defer crackQueueMu.Unlock()
		for taskID, entry := range standaloneCrackKeyspaceTasks {
			if entry == nil {
				delete(standaloneCrackKeyspaceTasks, taskID)
				continue
			}
			completeStandaloneCrackKeyspaceTaskLocked(taskID, entry, nil,
				status.Error(codes.Canceled, "standalone keyspace test cleanup"))
		}
		standaloneCrackKeyspaceTasks = map[string]*standaloneCrackKeyspaceTask{}
	}
	cleanup()
	t.Cleanup(cleanup)
}

func disableCrackQueueReaperForTest(t *testing.T) {
	t.Helper()
	original := crackQueueReaperStarter
	crackQueueReaperStarter = func() {}
	t.Cleanup(func() { crackQueueReaperStarter = original })
}

func setStandaloneKeyspaceTestStatus(station *core.Crackstation, hostUUID string, state clientpb.States, syncing bool, currentJobID string) {
	station.UpdateStatus(&clientpb.CrackstationStatus{
		HostUUID:          hostUUID,
		State:             state,
		IsSyncing:         syncing,
		CurrentCrackJobID: currentJobID,
	})
}

func receiveStandaloneKeyspaceAssignment(t *testing.T, station *core.Crackstation) crackTaskAssignment {
	return receiveStandaloneAssignment(t, station, consts.CrackKeyspace)
}

func receiveStandaloneAssignment(t *testing.T, station *core.Crackstation, eventType string) crackTaskAssignment {
	t.Helper()
	select {
	case event := <-station.Events:
		if event.EventType != eventType {
			t.Fatalf("event type = %q, want %q", event.EventType, eventType)
		}
		var assignment crackTaskAssignment
		if err := json.Unmarshal(event.Data, &assignment); err != nil {
			t.Fatalf("decode keyspace assignment: %v", err)
		}
		if assignment.TaskID == "" || assignment.HostUUID == "" || assignment.Attempt == 0 || assignment.LeaseToken == "" {
			t.Fatalf("incomplete keyspace assignment: %#v", assignment)
		}
		return assignment
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for keyspace assignment")
		return crackTaskAssignment{}
	}
}

func waitStandaloneKeyspaceResult(t *testing.T, result <-chan standaloneKeyspaceTestResult) standaloneKeyspaceTestResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for standalone keyspace result")
		return standaloneKeyspaceTestResult{}
	}
}

//nolint:gocyclo // The test keeps assignment, worker updates, result delivery, and cleanup in one lifecycle.
func TestStandaloneCrackKeyspaceLifecycleUsesExistingWorkerProtocol(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	disableCrackQueueReaperForTest(t)

	wordlist := &models.CrackFile{
		Name:       "standalone-keyspace.txt",
		Type:       int32(clientpb.CrackFileType_WORDLIST),
		Sha2_256:   strings.Repeat("a", 64),
		IsComplete: true,
	}
	if err := database.Create(wordlist).Error; err != nil {
		t.Fatalf("create managed wordlist: %v", err)
	}
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	setStandaloneKeyspaceTestStatus(station, station.HostUUID, clientpb.States_IDLE, false, "")

	rpcServer := &Server{}
	result := make(chan standaloneKeyspaceTestResult, 1)
	go func() {
		response, err := rpcServer.Crack(t.Context(), &clientpb.CrackCommand{
			AttackMode:          clientpb.CrackAttackMode_STRAIGHT,
			HashType:            clientpb.HashType_INVALID,
			Keyspace:            true,
			PositionalArguments: []string{wordlist.Name},
		})
		got := standaloneKeyspaceTestResult{err: err}
		if response != nil {
			got.keyspace = response.Keyspace
			got.job = response.Job
			got.query = response.Query
		}
		result <- got
	}()

	assignment := receiveStandaloneKeyspaceAssignment(t, station)
	if assignment.HostUUID != station.HostUUID {
		t.Fatalf("assigned host = %q, want %q", assignment.HostUUID, station.HostUUID)
	}
	wordlistURI := managedCrackFileURI(wordlist)
	if !activeStandaloneCrackKeyspaceReferencesManagedFile(wordlistURI) {
		t.Fatal("active standalone task did not protect its managed wordlist")
	}

	stationContext := crackstationTestContext("queue-test")
	fetched, err := rpcServer.CrackTaskByID(stationContext, &clientpb.CrackTask{
		ID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt, LeaseToken: assignment.LeaseToken,
	})
	if err != nil {
		t.Fatalf("fetch standalone task: %v", err)
	}
	if fetched.CrackJobID != "" || fetched.Kind != clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE || fetched.State != clientpb.CrackTaskState_CRACK_TASK_LEASED {
		t.Fatalf("fetched standalone task lifecycle = %#v", fetched)
	}
	if fetched.Command == nil || !fetched.Command.Keyspace || !fetched.Command.Quiet || len(fetched.Command.Hashes) != 0 ||
		len(fetched.Command.PositionalArguments) != 1 || fetched.Command.PositionalArguments[0] != wordlistURI {
		t.Fatalf("fetched standalone command = %#v", fetched.Command)
	}
	if fetched.CreatedAt == 0 || fetched.UpdatedAt == 0 || fetched.LeaseExpiresAt <= fetched.UpdatedAt {
		t.Fatalf("fetched standalone lease timestamps = %#v", fetched)
	}

	running := proto.Clone(fetched).(*clientpb.CrackTask)
	running.Command = nil
	running.State = clientpb.CrackTaskState_CRACK_TASK_RUNNING
	running.StartedAt = time.Now().Unix()
	if _, err := rpcServer.CrackTaskUpdate(stationContext, running); err != nil {
		t.Fatalf("start standalone task: %v", err)
	}
	oversizedStatus := json.RawMessage(`"` + strings.Repeat("x", maxCrackStatusBytes) + `"`)
	oversizedStatusData, err := json.Marshal(crackTaskStatusEvent{
		TaskID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt,
		LeaseToken: assignment.LeaseToken, Status: oversizedStatus,
	})
	if err != nil {
		t.Fatalf("encode oversized standalone heartbeat: %v", err)
	}
	if _, err := rpcServer.CrackstationTrigger(stationContext, &clientpb.Event{EventType: consts.CrackTaskStatus, Data: oversizedStatusData}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("oversized standalone heartbeat error = %v, want InvalidArgument", err)
	}
	statusJSON := json.RawMessage(`{"progress":1}`)
	statusData, err := json.Marshal(crackTaskStatusEvent{
		TaskID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt,
		LeaseToken: assignment.LeaseToken, Status: statusJSON,
	})
	if err != nil {
		t.Fatalf("encode standalone heartbeat: %v", err)
	}
	if _, err := rpcServer.CrackstationTrigger(stationContext, &clientpb.Event{EventType: consts.CrackTaskStatus, Data: statusData}); err != nil {
		t.Fatalf("heartbeat standalone task: %v", err)
	}

	completed := proto.Clone(running).(*clientpb.CrackTask)
	completed.State = clientpb.CrackTaskState_CRACK_TASK_COMPLETED
	completed.CompletedAt = time.Now().Unix()
	completed.Keyspace = "12345"
	completed.Stdout = []byte("12345\n")
	completed.LatestStatusJSON = statusJSON
	if _, err := rpcServer.CrackTaskUpdate(stationContext, completed); err != nil {
		t.Fatalf("complete standalone task: %v", err)
	}
	got := waitStandaloneKeyspaceResult(t, result)
	if got.err != nil || got.keyspace != "12345" {
		t.Fatalf("standalone keyspace result = %q, %v; want 12345", got.keyspace, got.err)
	}
	if got.query == nil || got.query.Mode != clientpb.CrackQueryMode_CRACK_QUERY_KEYSPACE || got.query.Value != "12345" ||
		got.query.CrackstationHostUUID != station.HostUUID || got.query.HashcatVersion != "hashcat-test-v1" {
		t.Fatalf("standalone keyspace query metadata = %#v", got.query)
	}
	if got.job != nil {
		t.Fatalf("standalone keyspace returned a persisted job: %#v", got.job)
	}
	if activeStandaloneCrackKeyspaceReferencesManagedFile(wordlistURI) {
		t.Fatal("completed standalone task still protects its managed wordlist")
	}

	for name, model := range map[string]interface{}{
		"jobs":     &models.CrackJob{},
		"tasks":    &models.CrackTask{},
		"commands": &models.CrackCommand{},
	} {
		var count int64
		if err := database.Model(model).Count(&count).Error; err != nil {
			t.Fatalf("count persisted %s: %v", name, err)
		}
		if count != 0 {
			t.Fatalf("standalone keyspace persisted %d %s", count, name)
		}
	}
}

func TestSelectIdleCrackstationForStandaloneKeyspace(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)

	noStatus := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	_ = noStatus
	cracking := addQueueTestStation(t, database, "22222222-2222-4222-8222-222222222222", nil)
	setStandaloneKeyspaceTestStatus(cracking, cracking.HostUUID, clientpb.States_CRACKING, false, "")
	syncing := addQueueTestStation(t, database, "33333333-3333-4333-8333-333333333333", nil)
	setStandaloneKeyspaceTestStatus(syncing, syncing.HostUUID, clientpb.States_IDLE, true, "")
	currentJob := addQueueTestStation(t, database, "44444444-4444-4444-8444-444444444444", nil)
	setStandaloneKeyspaceTestStatus(currentJob, currentJob.HostUUID, clientpb.States_IDLE, false, models.NewUUID().String())
	durable := addQueueTestStation(t, database, "55555555-5555-4555-8555-555555555555", nil)
	setStandaloneKeyspaceTestStatus(durable, durable.HostUUID, clientpb.States_IDLE, false, "")
	firstIdle := addQueueTestStation(t, database, "66666666-6666-4666-8666-666666666666", nil)
	setStandaloneKeyspaceTestStatus(firstIdle, firstIdle.HostUUID, clientpb.States_IDLE, false, "")
	secondIdle := addQueueTestStation(t, database, "77777777-7777-4777-8777-777777777777", nil)
	setStandaloneKeyspaceTestStatus(secondIdle, secondIdle.HostUUID, clientpb.States_IDLE, false, "")
	mismatchedStatus := addQueueTestStation(t, database, "88888888-8888-4888-8888-888888888888", nil)
	setStandaloneKeyspaceTestStatus(mismatchedStatus, firstIdle.HostUUID, clientpb.States_IDLE, false, "")

	job := createQueueTestJob(t, database, models.CrackCommand{})
	if err := database.Create(&models.CrackTask{
		CrackJobID:      job.ID,
		CrackstationID:  models.ParseUUIDOrNil(durable.HostUUID),
		Kind:            int32(clientpb.CrackTaskKind_CRACK_TASK_BENCHMARK),
		State:           int32(clientpb.CrackTaskState_CRACK_TASK_RUNNING),
		Attempt:         1,
		LeaseToken:      "active-durable-lease",
		LeaseExpiresAt:  time.Now().Add(time.Minute),
		LastHeartbeatAt: time.Now(),
		UpdatedAt:       time.Now(),
	}).Error; err != nil {
		t.Fatalf("create active durable task: %v", err)
	}

	crackQueueMu.Lock()
	selected, err := selectIdleCrackstationLocked(context.Background())
	crackQueueMu.Unlock()
	if err != nil {
		t.Fatalf("select idle crackstation: %v", err)
	}
	if selected == nil || selected.HostUUID != firstIdle.HostUUID {
		t.Fatalf("selected host = %#v, want %q", selected, firstIdle.HostUUID)
	}
}

func TestStandaloneCrackKeyspaceCancellationRetainsReservationUntilWorkerAck(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	disableCrackQueueReaperForTest(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	setStandaloneKeyspaceTestStatus(station, station.HostUUID, clientpb.States_IDLE, false, "")

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan standaloneKeyspaceTestResult, 1)
	go func() {
		keyspace, err := runStandaloneCrackKeyspace(ctx, &models.CrackCommand{
			AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE), Keyspace: true, PositionalArguments: []string{"?d"},
		})
		result <- standaloneKeyspaceTestResult{keyspace: keyspace, err: err}
	}()
	assignment := receiveStandaloneKeyspaceAssignment(t, station)
	cancel()
	got := waitStandaloneKeyspaceResult(t, result)
	if status.Code(got.err) != codes.Canceled {
		t.Fatalf("cancelled keyspace error = %v, want Canceled", got.err)
	}

	crackQueueMu.Lock()
	entry := standaloneCrackKeyspaceTasks[assignment.TaskID]
	reserved, selectErr := selectIdleCrackstationLocked(context.Background())
	crackQueueMu.Unlock()
	if entry == nil || !entry.cancelRequested {
		t.Fatalf("cancelled keyspace reservation = %#v, want tombstone", entry)
	}
	if selectErr != nil || reserved != nil {
		t.Fatalf("selection while cancellation is unacknowledged = %#v, %v; want no station", reserved, selectErr)
	}

	rpcServer := &Server{}
	stationContext := crackstationTestContext("queue-test")
	if _, err := rpcServer.CrackTaskByID(stationContext, &clientpb.CrackTask{
		ID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt, LeaseToken: "wrong-lease-token",
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("wrong-lease cancellation acknowledgement error = %v, want PermissionDenied", err)
	}
	crackQueueMu.Lock()
	entry = standaloneCrackKeyspaceTasks[assignment.TaskID]
	crackQueueMu.Unlock()
	if entry == nil || !entry.cancelRequested {
		t.Fatal("wrong lease released the cancelled keyspace reservation")
	}

	_, err := rpcServer.CrackTaskByID(stationContext, &clientpb.CrackTask{
		ID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt, LeaseToken: assignment.LeaseToken,
	})
	if status.Code(err) != codes.Aborted {
		t.Fatalf("cancel acknowledgement error=%v, want Aborted", err)
	}
	crackQueueMu.Lock()
	_, stillReserved := standaloneCrackKeyspaceTasks[assignment.TaskID]
	available, selectErr := selectIdleCrackstationLocked(context.Background())
	crackQueueMu.Unlock()
	if stillReserved || selectErr != nil || available == nil || available.HostUUID != station.HostUUID {
		t.Fatalf("selection after cancellation acknowledgement reserved=%v station=%#v err=%v", stillReserved, available, selectErr)
	}
}

func TestStandaloneCrackKeyspaceRunningCancellationStaysReserved(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	disableCrackQueueReaperForTest(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	setStandaloneKeyspaceTestStatus(station, station.HostUUID, clientpb.States_IDLE, false, "")

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan standaloneKeyspaceTestResult, 1)
	go func() {
		keyspace, err := runStandaloneCrackKeyspace(ctx, &models.CrackCommand{
			AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE), Keyspace: true, PositionalArguments: []string{"?d"},
		})
		result <- standaloneKeyspaceTestResult{keyspace: keyspace, err: err}
	}()
	assignment := receiveStandaloneKeyspaceAssignment(t, station)
	rpcServer := &Server{}
	stationContext := crackstationTestContext("queue-test")
	fetched, err := rpcServer.CrackTaskByID(stationContext, &clientpb.CrackTask{
		ID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt, LeaseToken: assignment.LeaseToken,
	})
	if err != nil {
		t.Fatalf("fetch running-cancellation task: %v", err)
	}
	running := proto.Clone(fetched).(*clientpb.CrackTask)
	running.Command = nil
	running.State = clientpb.CrackTaskState_CRACK_TASK_RUNNING
	running.StartedAt = time.Now().Unix()
	if _, err := rpcServer.CrackTaskUpdate(stationContext, running); err != nil {
		t.Fatalf("start running-cancellation task: %v", err)
	}

	cancel()
	got := waitStandaloneKeyspaceResult(t, result)
	if status.Code(got.err) != codes.Canceled {
		t.Fatalf("running keyspace cancellation error = %v, want Canceled", got.err)
	}
	heartbeatData, err := json.Marshal(crackTaskStatusEvent{
		TaskID: assignment.TaskID, HostUUID: assignment.HostUUID, Attempt: assignment.Attempt,
		LeaseToken: assignment.LeaseToken, Status: json.RawMessage(`{"progress":1}`),
	})
	if err != nil {
		t.Fatalf("encode cancelled running heartbeat: %v", err)
	}
	if _, err := rpcServer.CrackstationTrigger(stationContext, &clientpb.Event{EventType: consts.CrackTaskStatus, Data: heartbeatData}); status.Code(err) != codes.Aborted {
		t.Fatalf("cancelled running heartbeat error = %v, want Aborted", err)
	}
	crackQueueMu.Lock()
	entry := standaloneCrackKeyspaceTasks[assignment.TaskID]
	crackQueueMu.Unlock()
	if entry == nil || !entry.cancelRequested || entry.task.State != clientpb.CrackTaskState_CRACK_TASK_RUNNING {
		t.Fatalf("cancelled running reservation after heartbeat = %#v", entry)
	}

	terminal := proto.Clone(running).(*clientpb.CrackTask)
	terminal.State = clientpb.CrackTaskState_CRACK_TASK_FAILED
	terminal.CompletedAt = time.Now().Unix()
	terminal.Err = "cancelled locally"
	if _, err := rpcServer.CrackTaskUpdate(stationContext, terminal); status.Code(err) != codes.Aborted {
		t.Fatalf("cancelled terminal acknowledgement error = %v, want Aborted", err)
	}
	crackQueueMu.Lock()
	_, stillReserved := standaloneCrackKeyspaceTasks[assignment.TaskID]
	crackQueueMu.Unlock()
	if stillReserved {
		t.Fatal("terminal acknowledgement did not release cancelled running reservation")
	}
}

func TestStandaloneCrackKeyspaceReservationReleasePaths(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	station := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	setStandaloneKeyspaceTestStatus(station, station.HostUUID, clientpb.States_IDLE, false, "")
	command := &models.CrackCommand{
		AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE), Keyspace: true, PositionalArguments: []string{"?d"},
	}

	disconnected, err := reserveStandaloneCrackKeyspaceTask(context.Background(), models.NewUUID(), command)
	if err != nil {
		t.Fatalf("reserve disconnect task: %v", err)
	}
	receiveStandaloneKeyspaceAssignment(t, station)
	crackQueueMu.Lock()
	available := map[string]*core.Crackstation{station.HostUUID: station}
	excludeStandaloneCrackKeyspaceHostsLocked(available)
	replacement := core.NewCrackstation(&clientpb.Crackstation{HostUUID: station.HostUUID})
	failStandaloneCrackKeyspaceTasksForStationLocked(replacement)
	if standaloneCrackKeyspaceTasks[disconnected.task.ID] == nil {
		crackQueueMu.Unlock()
		t.Fatal("replacement connection failed work assigned to the old connection")
	}
	failStandaloneCrackKeyspaceTasksForStationLocked(station)
	crackQueueMu.Unlock()
	if len(available) != 0 {
		t.Fatalf("standalone reservation remained scheduler-available: %#v", available)
	}
	select {
	case <-disconnected.done:
		if status.Code(disconnected.resultErr) != codes.Unavailable {
			t.Fatalf("disconnect result = %v, want Unavailable", disconnected.resultErr)
		}
	default:
		t.Fatal("disconnect did not release standalone reservation")
	}

	expired, err := reserveStandaloneCrackKeyspaceTask(context.Background(), models.NewUUID(), command)
	if err != nil {
		t.Fatalf("reserve expiry task: %v", err)
	}
	receiveStandaloneKeyspaceAssignment(t, station)
	crackQueueMu.Lock()
	expired.task.LeaseExpiresAt = time.Now().Add(-time.Second).Unix()
	reapStandaloneCrackKeyspaceTasksLocked(time.Now())
	crackQueueMu.Unlock()
	select {
	case <-expired.done:
		if status.Code(expired.resultErr) != codes.DeadlineExceeded {
			t.Fatalf("expiry result = %v, want DeadlineExceeded", expired.resultErr)
		}
	default:
		t.Fatal("lease expiry did not release standalone reservation")
	}

	for index := 0; index < cap(station.Events); index++ {
		station.Events <- &clientpb.Event{EventType: "occupied"}
	}
	if _, err := reserveStandaloneCrackKeyspaceTask(context.Background(), models.NewUUID(), command); !errors.Is(err, errCrackstationEventQueueFull) {
		t.Fatalf("full event queue error = %v, want %v", err, errCrackstationEventQueueFull)
	}
	crackQueueMu.Lock()
	remaining := len(standaloneCrackKeyspaceTasks)
	crackQueueMu.Unlock()
	if remaining != 0 {
		t.Fatalf("failed dispatch retained %d standalone reservations", remaining)
	}
}

func TestStandaloneCrackKeyspaceSkipsIdleStationWithFullEventQueue(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	resetStandaloneCrackKeyspaceTasksForTest(t)
	first := addQueueTestStation(t, database, "11111111-1111-4111-8111-111111111111", nil)
	setStandaloneKeyspaceTestStatus(first, first.HostUUID, clientpb.States_IDLE, false, "")
	second := addQueueTestStation(t, database, "22222222-2222-4222-8222-222222222222", nil)
	setStandaloneKeyspaceTestStatus(second, second.HostUUID, clientpb.States_IDLE, false, "")
	for index := 0; index < cap(first.Events); index++ {
		first.Events <- &clientpb.Event{EventType: "occupied"}
	}

	entry, err := reserveStandaloneCrackKeyspaceTask(context.Background(), models.NewUUID(), &models.CrackCommand{
		AttackMode: int32(clientpb.CrackAttackMode_BRUTEFORCE), Keyspace: true, PositionalArguments: []string{"?d"},
	})
	if err != nil {
		t.Fatalf("reserve keyspace task after first event queue is full: %v", err)
	}
	assignment := receiveStandaloneKeyspaceAssignment(t, second)
	if entry.task.HostUUID != second.HostUUID || assignment.HostUUID != second.HostUUID {
		t.Fatalf("keyspace task selected host %q/%q, want %q", entry.task.HostUUID, assignment.HostUUID, second.HostUUID)
	}
}

func TestStandaloneCrackKeyspaceResultValidation(t *testing.T) {
	tests := []struct {
		name    string
		task    *clientpb.CrackTask
		want    string
		wantErr bool
	}{
		{name: "keyspace field", task: &clientpb.CrackTask{Keyspace: "42"}, want: "42"},
		{name: "stdout fallback", task: &clientpb.CrackTask{Stdout: []byte("  84\n")}, want: "84"},
		{name: "truncated stdout", task: &clientpb.CrackTask{Stdout: []byte("84"), StdoutTruncated: true}, wantErr: true},
		{name: "truncated stderr", task: &clientpb.CrackTask{Stdout: []byte("84"), StderrTruncated: true}, wantErr: true},
		{name: "nonzero exit", task: &clientpb.CrackTask{Stdout: []byte("84"), ExitCode: 1}, wantErr: true},
		{name: "invalid UTF-8", task: &clientpb.CrackTask{Stdout: []byte{0xff}}, wantErr: true},
		{name: "disagreeing fields", task: &clientpb.CrackTask{Keyspace: "42", Stdout: []byte("84")}, wantErr: true},
		{name: "empty", task: &clientpb.CrackTask{}, wantErr: true},
		{name: "overflow", task: &clientpb.CrackTask{Stdout: []byte("18446744073709551616")}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := standaloneCrackKeyspaceResult(test.task)
			if test.wantErr != (err != nil) || got != test.want {
				t.Fatalf("result = %q, %v; want %q, error=%v", got, err, test.want, test.wantErr)
			}
		})
	}

	if _, err := standaloneCrackKeyspaceResult(&clientpb.CrackTask{Keyspace: "01"}); err == nil {
		t.Fatal("noncanonical keyspace update was accepted")
	}
	if _, err := standaloneCrackKeyspaceResult(&clientpb.CrackTask{Keyspace: " 1 "}); err == nil {
		t.Fatal("whitespace-padded keyspace update was accepted")
	}
	if err := validateStandaloneCrackTaskUpdate(&clientpb.CrackTask{RecoveredJSON: []byte("[]")}); err == nil {
		t.Fatal("recovered results were accepted for a keyspace-only task")
	}
}
