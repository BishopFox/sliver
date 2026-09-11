package rpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/glebarez/sqlite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type crackstationRegisterTestStream struct {
	ctx     context.Context
	actions chan string
	events  chan *clientpb.Event
	mu      sync.Mutex
	header  metadata.MD
	trailer metadata.MD
}

func newCrackstationRegisterTestStream(ctx context.Context) *crackstationRegisterTestStream {
	return &crackstationRegisterTestStream{
		ctx:     ctx,
		actions: make(chan string, 16),
		events:  make(chan *clientpb.Event, 16),
	}
}

func (stream *crackstationRegisterTestStream) Send(event *clientpb.Event) error {
	select {
	case <-stream.ctx.Done():
		return stream.ctx.Err()
	case stream.events <- event:
	}
	select {
	case <-stream.ctx.Done():
		return stream.ctx.Err()
	case stream.actions <- event.EventType:
		return nil
	}
}

func (stream *crackstationRegisterTestStream) SetHeader(header metadata.MD) error {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	stream.header = metadata.Join(stream.header, header)
	return nil
}

func (stream *crackstationRegisterTestStream) SendHeader(header metadata.MD) error {
	if err := stream.SetHeader(header); err != nil {
		return err
	}
	select {
	case <-stream.ctx.Done():
		return stream.ctx.Err()
	case stream.actions <- "header":
		return nil
	}
}

func (stream *crackstationRegisterTestStream) SetTrailer(trailer metadata.MD) {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	stream.trailer = metadata.Join(stream.trailer, trailer)
}

func (stream *crackstationRegisterTestStream) Context() context.Context { return stream.ctx }

func (stream *crackstationRegisterTestStream) SendMsg(message any) error {
	event, ok := message.(*clientpb.Event)
	if !ok {
		return status.Error(codes.Internal, "unexpected stream message")
	}
	return stream.Send(event)
}

func (stream *crackstationRegisterTestStream) RecvMsg(any) error {
	<-stream.ctx.Done()
	return stream.ctx.Err()
}

var _ rpcpb.SliverRPC_CrackstationRegisterServer = (*crackstationRegisterTestStream)(nil)

func crackstationTestContext(commonName string) context.Context {
	certificate := &x509.Certificate{Subject: pkix.Name{CommonName: commonName}}
	return peer.NewContext(context.Background(), &peer.Peer{AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{certificate}},
	}}})
}

func TestCrackTaskUpdateEnforcesLeaseAndPreservesServerOwnedFields(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)

	station := &models.Crackstation{ID: models.NewUUID(), HashcatVersion: "hashcat-v1"}
	if err := database.Omit("Tasks", "Benchmarks").Create(station).Error; err != nil {
		t.Fatalf("create crackstation: %v", err)
	}
	job := &models.CrackJob{}
	if err := database.Omit("Tasks", "Command").Create(job).Error; err != nil {
		t.Fatalf("create crack job: %v", err)
	}
	task := &models.CrackTask{
		CrackJobID:     job.ID,
		CrackstationID: station.ID,
		Kind:           int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK),
		State:          int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt:        2,
		LeaseToken:     "lease-token",
		LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Omit("Command").Create(task).Error; err != nil {
		t.Fatalf("create crack task: %v", err)
	}
	command := &models.CrackCommand{
		CrackTaskID: task.ID,
		Session:     "server-owned-command",
	}
	if err := database.Create(command).Error; err != nil {
		t.Fatalf("create crack command: %v", err)
	}

	var original models.CrackTask
	if err := database.First(&original, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load original crack task: %v", err)
	}
	workerStartedAt := time.Unix(10, 0)
	workerCompletedAt := time.Unix(20, 0)
	rpcServer := &Server{}
	operatorName := "crackstation-test"
	runtimeStation := core.NewCrackstation(&clientpb.Crackstation{HostUUID: station.ID.String(), OperatorName: operatorName, HashcatVersion: "hashcat-v1"})
	if err := core.AddCrackstation(runtimeStation); err != nil {
		t.Fatalf("add runtime crackstation: %v", err)
	}
	t.Cleanup(func() { core.RemoveCrackstation(station.ID.String()) })
	otherHost := models.NewUUID().String()
	otherStation := core.NewCrackstation(&clientpb.Crackstation{HostUUID: otherHost, OperatorName: operatorName})
	if err := core.AddCrackstation(otherStation); err != nil {
		t.Fatalf("add second runtime crackstation: %v", err)
	}
	t.Cleanup(func() { core.RemoveCrackstation(otherHost) })
	ctx := crackstationTestContext(operatorName)
	if _, err := rpcServer.CrackTaskByID(ctx, &clientpb.CrackTask{
		ID: task.ID.String(), HostUUID: otherHost, Attempt: task.Attempt, LeaseToken: task.LeaseToken,
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("same-certificate cross-host fetch error = %v, want PermissionDenied", err)
	}
	if fetched, err := rpcServer.CrackTaskByID(ctx, &clientpb.CrackTask{
		ID: task.ID.String(), HostUUID: station.ID.String(), Attempt: task.Attempt, LeaseToken: task.LeaseToken,
	}); err != nil || fetched.LeaseToken != task.LeaseToken {
		t.Fatalf("assigned task fetch = (%#v, %v)", fetched, err)
	}
	if _, err := rpcServer.CrackTaskUpdate(ctx, &clientpb.CrackTask{
		ID: task.ID.String(), HostUUID: station.ID.String(), Attempt: task.Attempt, LeaseToken: task.LeaseToken,
		LatestStatusJSON: []byte(`{"unterminated":`),
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid status JSON update error = %v, want InvalidArgument", err)
	}
	var afterInvalid models.CrackTask
	if err := database.First(&afterInvalid, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load task after invalid status: %v", err)
	}
	if len(afterInvalid.LatestStatusJSON) != 0 || afterInvalid.State != int32(clientpb.CrackTaskState_CRACK_TASK_LEASED) {
		t.Fatalf("invalid status JSON was persisted: %#v", afterInvalid)
	}

	beforeStart := time.Now()
	if _, err := rpcServer.CrackTaskUpdate(ctx, &clientpb.CrackTask{
		ID:         task.ID.String(),
		HostUUID:   station.ID.String(),
		Attempt:    task.Attempt,
		LeaseToken: task.LeaseToken,
		StartedAt:  workerStartedAt.Unix(),
		Command:    &clientpb.CrackCommand{Session: "replacement-command"},
	}); err != nil {
		t.Fatalf("start crack task: %v", err)
	}
	afterStart := time.Now()
	var running models.CrackTask
	if err := database.First(&running, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load running crack task: %v", err)
	}
	if running.StartedAt.Before(beforeStart) || running.StartedAt.After(afterStart) || running.StartedAt.Equal(workerStartedAt) {
		t.Fatalf("server-owned started time = %s, call window [%s, %s]", running.StartedAt, beforeStart, afterStart)
	}
	serverStartedAt := running.StartedAt
	beforeFinish := time.Now()
	if _, err := rpcServer.CrackTaskUpdate(ctx, &clientpb.CrackTask{
		ID:               task.ID.String(),
		HostUUID:         station.ID.String(),
		Attempt:          task.Attempt,
		LeaseToken:       task.LeaseToken,
		StartedAt:        workerStartedAt.Add(time.Hour).Unix(),
		CompletedAt:      workerCompletedAt.Unix(),
		Err:              "hashcat exit status 1",
		Stdout:           []byte("stdout"),
		Stderr:           []byte("stderr"),
		ExitCode:         1,
		StdoutTruncated:  true,
		StderrTruncated:  true,
		StdoutTotalBytes: 11,
		StderrTotalBytes: 12,
		Command:          &clientpb.CrackCommand{Session: "replacement-command"},
	}); err != nil {
		t.Fatalf("finish crack task: %v", err)
	}
	afterFinish := time.Now()

	var got models.CrackTask
	if err := database.Preload("Command").First(&got, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load updated crack task: %v", err)
	}
	if got.CrackJobID != job.ID {
		t.Fatalf("CrackJobID = %s, want %s", got.CrackJobID, job.ID)
	}
	if got.CrackstationID != station.ID {
		t.Fatalf("CrackstationID = %s, want %s", got.CrackstationID, station.ID)
	}
	if !got.CreatedAt.Equal(original.CreatedAt) {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt, original.CreatedAt)
	}
	if !got.StartedAt.Equal(serverStartedAt) || got.CompletedAt.Before(beforeFinish) || got.CompletedAt.After(afterFinish) || got.CompletedAt.Equal(workerCompletedAt) {
		t.Fatalf("server-owned execution times = (%s, %s), started=%s finish window=[%s, %s]", got.StartedAt, got.CompletedAt, serverStartedAt, beforeFinish, afterFinish)
	}
	if got.Err != "hashcat exit status 1" || got.ExitCode != 1 {
		t.Fatalf("process result = (%q, %d), want (%q, 1)", got.Err, got.ExitCode, "hashcat exit status 1")
	}
	if !bytes.Equal(got.Stdout, []byte("stdout")) || !bytes.Equal(got.Stderr, []byte("stderr")) {
		t.Fatalf("process output = (%q, %q)", got.Stdout, got.Stderr)
	}
	if !got.StdoutTruncated || !got.StderrTruncated || got.StdoutTotalBytes != 11 || got.StderrTotalBytes != 12 {
		t.Fatalf("process output metadata was not persisted: %#v", got)
	}
	if got.Command.ID != command.ID || got.Command.Session != command.Session {
		t.Fatalf("command was replaced: got (%s, %q), want (%s, %q)", got.Command.ID, got.Command.Session, command.ID, command.Session)
	}
	var commandCount int64
	if err := database.Model(&models.CrackCommand{}).Count(&commandCount).Error; err != nil {
		t.Fatalf("count crack commands: %v", err)
	}
	if commandCount != 1 {
		t.Fatalf("crack command count = %d, want 1", commandCount)
	}
	_, err := rpcServer.CrackTaskUpdate(ctx, &clientpb.CrackTask{
		ID: task.ID.String(), HostUUID: station.ID.String(), Attempt: task.Attempt, LeaseToken: task.LeaseToken,
	})
	if status.Code(err) != codes.Aborted {
		t.Fatalf("terminal task replay error = %v, want Aborted", err)
	}

	_, err = rpcServer.CrackTaskUpdate(ctx, &clientpb.CrackTask{ID: models.NewUUID().String()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("missing crack task error = %v, want NotFound", err)
	}
}

func TestExpiredCrackTaskLeaseCannotFetchUpdateOrRenew(t *testing.T) {
	for _, test := range []struct {
		name      string
		expiresAt func() time.Time
	}{
		{name: "expired", expiresAt: func() time.Time { return time.Now().Add(-time.Second) }},
		{name: "equality boundary", expiresAt: func() time.Time { return time.Now() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := setupCrackstationRPCTestDB(t)
			station := &models.Crackstation{ID: models.NewUUID()}
			if err := database.Omit("Tasks", "Benchmarks").Create(station).Error; err != nil {
				t.Fatalf("create crackstation: %v", err)
			}
			job := &models.CrackJob{}
			if err := database.Omit("Tasks", "Command", "Results").Create(job).Error; err != nil {
				t.Fatalf("create crack job: %v", err)
			}
			task := &models.CrackTask{
				CrackJobID: job.ID, CrackstationID: station.ID,
				Kind: int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK), State: int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
				Attempt: 1, LeaseToken: "expired-lease-token", LeaseExpiresAt: test.expiresAt(),
			}
			if err := database.Create(task).Error; err != nil {
				t.Fatalf("create expired task: %v", err)
			}
			operatorName := "expired-lease-test"
			runtimeStation := core.NewCrackstation(&clientpb.Crackstation{HostUUID: station.ID.String(), OperatorName: operatorName})
			if err := core.AddCrackstation(runtimeStation); err != nil {
				t.Fatalf("add runtime station: %v", err)
			}
			t.Cleanup(func() { core.RemoveCrackstation(station.ID.String()) })
			ctx := crackstationTestContext(operatorName)
			rpcServer := &Server{}
			events := core.EventBroker.Subscribe()
			defer core.EventBroker.Unsubscribe(events)
			request := &clientpb.CrackTask{ID: task.ID.String(), HostUUID: station.ID.String(), Attempt: task.Attempt, LeaseToken: task.LeaseToken}
			if _, err := rpcServer.CrackTaskByID(ctx, request); status.Code(err) != codes.Aborted {
				t.Fatalf("expired task fetch error = %v, want Aborted", err)
			}
			update := proto.Clone(request).(*clientpb.CrackTask)
			update.StartedAt = time.Now().Unix()
			update.Stdout = []byte("must not persist")
			if _, err := rpcServer.CrackTaskUpdate(ctx, update); status.Code(err) != codes.Aborted {
				t.Fatalf("expired task update error = %v, want Aborted", err)
			}
			statusData, err := json.Marshal(crackTaskStatusEvent{
				TaskID: task.ID.String(), HostUUID: station.ID.String(), Attempt: task.Attempt,
				LeaseToken: task.LeaseToken, Status: json.RawMessage(`{"status":3}`),
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := rpcServer.CrackstationTrigger(ctx, &clientpb.Event{EventType: consts.CrackTaskStatus, Data: statusData}); status.Code(err) != codes.Aborted {
				t.Fatalf("expired status update error = %v, want Aborted", err)
			}
			select {
			case event := <-events:
				if event.EventType == consts.CrackTaskStatus || event.EventType == consts.CrackJobUpdated {
					t.Fatalf("expired lease emitted event %#v", event)
				}
			case <-time.After(20 * time.Millisecond):
			}
			var got models.CrackTask
			if err := database.First(&got, "id = ?", task.ID).Error; err != nil {
				t.Fatalf("reload expired task: %v", err)
			}
			if got.State != task.State || got.LeaseToken != task.LeaseToken || !got.LeaseExpiresAt.Equal(task.LeaseExpiresAt) || !got.StartedAt.IsZero() || !got.LastHeartbeatAt.IsZero() || len(got.Stdout) != 0 || len(got.LatestStatusJSON) != 0 {
				t.Fatalf("expired task changed: %#v", got)
			}
		})
	}
}

func TestCrackstationTriggerRejectsOversizedEnvelopeBeforeParsing(t *testing.T) {
	rpcServer := &Server{}
	padding := strings.Repeat("x", maxCrackStatusEventBytes)
	taskStatus, err := json.Marshal(map[string]any{
		"task_id": "ignored", "host_uuid": "ignored", "attempt": 1,
		"lease_token": "ignored", "status": json.RawMessage(`{}`), "padding": padding,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []*clientpb.Event{
		{EventType: consts.CrackTaskStatus, Data: taskStatus},
		{EventType: consts.CrackStatusEvent, Data: bytes.Repeat([]byte{'x'}, maxCrackStatusBytes+1)},
	} {
		if _, err := rpcServer.CrackstationTrigger(t.Context(), event); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("oversized %q event error = %v, want InvalidArgument", event.EventType, err)
		}
	}
}

func TestCrackstationBenchmarkRejectsEmptyAndInvalidResults(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	station := &models.Crackstation{ID: models.NewUUID(), HashcatVersion: "hashcat-v1"}
	if err := database.Omit("Tasks", "Benchmarks").Create(station).Error; err != nil {
		t.Fatalf("create crackstation: %v", err)
	}
	existing := &models.Benchmark{CrackstationID: station.ID, HashType: int32(clientpb.HashType_MD5), PerSecondRate: 10}
	if err := database.Create(existing).Error; err != nil {
		t.Fatalf("create benchmark: %v", err)
	}
	operatorName := "benchmark-test"
	runtimeStation := core.NewCrackstation(&clientpb.Crackstation{HostUUID: station.ID.String(), OperatorName: operatorName, HashcatVersion: "hashcat-v1"})
	if err := core.AddCrackstation(runtimeStation); err != nil {
		t.Fatalf("add runtime crackstation: %v", err)
	}
	t.Cleanup(func() { core.RemoveCrackstation(station.ID.String()) })
	rpcServer := &Server{}
	ctx := crackstationTestContext(operatorName)
	for _, stale := range []*clientpb.CrackBenchmark{
		{HostUUID: station.ID.String(), Benchmarks: map[int32]uint64{int32(clientpb.HashType_MD5): 20}, HashcatVersion: "hashcat-v1"},
		{HostUUID: station.ID.String(), Benchmarks: map[int32]uint64{int32(clientpb.HashType_MD5): 20}, SchemaVersion: crackBenchmarkSchemaVersion, HashcatVersion: "hashcat-v2"},
	} {
		if _, err := rpcServer.CrackstationBenchmark(ctx, stale); status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("stale benchmark marker %#v error = %v, want FailedPrecondition", stale, err)
		}
	}
	invalid := []map[int32]uint64{
		{},
		{int32(clientpb.HashType_MD5): 0},
		{-1: 100},
		{int32(clientpb.HashType_INVALID): 100},
	}
	for _, benchmarks := range invalid {
		if _, err := rpcServer.CrackstationBenchmark(ctx, &clientpb.CrackBenchmark{
			HostUUID: station.ID.String(), Benchmarks: benchmarks,
			SchemaVersion: crackBenchmarkSchemaVersion, HashcatVersion: "hashcat-v1",
		}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("benchmark %#v error = %v, want InvalidArgument", benchmarks, err)
		}
	}
	var benchmarks []models.Benchmark
	if err := database.Where("crackstation_id = ?", station.ID).Find(&benchmarks).Error; err != nil {
		t.Fatalf("load preserved benchmarks: %v", err)
	}
	if len(benchmarks) != 1 || benchmarks[0].PerSecondRate != existing.PerSecondRate {
		t.Fatalf("invalid upload replaced benchmark: %#v", benchmarks)
	}
	if _, err := rpcServer.CrackstationBenchmark(ctx, &clientpb.CrackBenchmark{
		HostUUID: station.ID.String(), Benchmarks: map[int32]uint64{int32(clientpb.HashType_SHA1): 25},
		SchemaVersion: crackBenchmarkSchemaVersion, HashcatVersion: "hashcat-v1",
	}); err != nil {
		t.Fatalf("replace benchmark: %v", err)
	}
	benchmarks = nil
	if err := database.Where("crackstation_id = ?", station.ID).Find(&benchmarks).Error; err != nil {
		t.Fatalf("load replacement benchmarks: %v", err)
	}
	if len(benchmarks) != 1 || benchmarks[0].HashType != int32(clientpb.HashType_SHA1) || benchmarks[0].PerSecondRate != 25 {
		t.Fatalf("replacement benchmarks = %#v", benchmarks)
	}
	var persisted models.Crackstation
	if err := database.First(&persisted, "id = ?", station.ID).Error; err != nil {
		t.Fatalf("load benchmark freshness marker: %v", err)
	}
	if persisted.HashcatVersion != "hashcat-v1" || persisted.BenchmarkHashcatVersion != "hashcat-v1" || persisted.BenchmarkSchemaVersion != crackBenchmarkSchemaVersion {
		t.Fatalf("benchmark freshness marker = %#v", persisted)
	}
}

func TestCrackstationRegisterSignalsReadyThenBenchmarksOnlyUntilRecorded(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	originalReaperStarter := crackQueueReaperStarter
	crackQueueReaperStarter = func() {}
	t.Cleanup(func() { crackQueueReaperStarter = originalReaperStarter })
	rpcServer := &Server{}
	operatorName := "register-benchmark-test"
	hostUUID := models.NewUUID().String()
	request := &clientpb.Crackstation{HostUUID: hostUUID, Name: "gpu-worker", HashcatVersion: "hashcat-v1"}

	runRegistration := func() (*crackstationRegisterTestStream, context.CancelFunc, <-chan error) {
		ctx, cancel := context.WithCancel(crackstationTestContext(operatorName))
		stream := newCrackstationRegisterTestStream(ctx)
		done := make(chan error, 1)
		go func() { done <- rpcServer.CrackstationRegister(request, stream) }()
		return stream, cancel, done
	}
	waitAction := func(stream *crackstationRegisterTestStream) string {
		select {
		case action := <-stream.actions:
			return action
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for registration action")
			return ""
		}
	}
	waitDone := func(cancel context.CancelFunc, done <-chan error) {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("registration returned error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("registration did not stop after cancellation")
		}
	}

	firstStream, firstCancel, firstDone := runRegistration()
	if action := waitAction(firstStream); action != "header" {
		t.Fatalf("first registration action = %q, want header", action)
	}
	if action := waitAction(firstStream); action != "crack-file-updated" {
		t.Fatalf("second registration action = %q, want crack-file-updated", action)
	}
	if action := waitAction(firstStream); action != "crack-benchmark" {
		t.Fatalf("third registration action = %q, want crack-benchmark", action)
	}
	if _, err := rpcServer.CrackstationBenchmark(firstStream.Context(), &clientpb.CrackBenchmark{
		HostUUID: hostUUID, SchemaVersion: crackBenchmarkSchemaVersion, HashcatVersion: "hashcat-v1",
		Benchmarks: map[int32]uint64{
			int32(clientpb.HashType_MD5): 100,
		},
	}); err != nil {
		t.Fatalf("record first-connect benchmark: %v", err)
	}
	waitDone(firstCancel, firstDone)

	persisted, err := db.CrackstationByHostUUID(hostUUID)
	if err != nil {
		t.Fatalf("load registered crackstation: %v", err)
	}
	if persisted.OperatorName != operatorName || len(persisted.Benchmarks) != 1 || persisted.Benchmarks[0].PerSecondRate != 100 {
		t.Fatalf("persisted crackstation = %#v", persisted)
	}
	if persisted.HashcatVersion != "hashcat-v1" || persisted.BenchmarkHashcatVersion != "hashcat-v1" || persisted.BenchmarkSchemaVersion != crackBenchmarkSchemaVersion {
		t.Fatalf("persisted benchmark version marker = %#v", persisted)
	}

	secondStream, secondCancel, secondDone := runRegistration()
	if action := waitAction(secondStream); action != "header" {
		t.Fatalf("reconnect first action = %q, want header", action)
	}
	if action := waitAction(secondStream); action != "crack-file-updated" {
		t.Fatalf("reconnect second action = %q, want crack-file-updated", action)
	}
	select {
	case action := <-secondStream.actions:
		t.Fatalf("reconnect unexpectedly dispatched %q", action)
	case <-time.After(100 * time.Millisecond):
	}
	waitDone(secondCancel, secondDone)

	request.HashcatVersion = "hashcat-v2"
	thirdStream, thirdCancel, thirdDone := runRegistration()
	if action := waitAction(thirdStream); action != "header" {
		t.Fatalf("version-change registration first action = %q, want header", action)
	}
	if action := waitAction(thirdStream); action != "crack-file-updated" {
		t.Fatalf("version-change registration second action = %q, want crack-file-updated", action)
	}
	if action := waitAction(thirdStream); action != "crack-benchmark" {
		t.Fatalf("version-change registration third action = %q, want crack-benchmark", action)
	}
	waitDone(thirdCancel, thirdDone)

	fourthStream, fourthCancel, fourthDone := runRegistration()
	if action := waitAction(fourthStream); action != "header" {
		t.Fatalf("unrecorded refresh first action = %q, want header", action)
	}
	if action := waitAction(fourthStream); action != "crack-file-updated" {
		t.Fatalf("unrecorded refresh second action = %q, want crack-file-updated", action)
	}
	if action := waitAction(fourthStream); action != "crack-benchmark" {
		t.Fatalf("unrecorded refresh third action = %q, want crack-benchmark", action)
	}
	if _, err := rpcServer.CrackstationBenchmark(fourthStream.Context(), &clientpb.CrackBenchmark{
		HostUUID: hostUUID, SchemaVersion: crackBenchmarkSchemaVersion, HashcatVersion: "hashcat-v2",
		Benchmarks: map[int32]uint64{int32(clientpb.HashType_MD5): 200},
	}); err != nil {
		t.Fatalf("record version-change benchmark: %v", err)
	}
	waitDone(fourthCancel, fourthDone)

	fifthStream, fifthCancel, fifthDone := runRegistration()
	if action := waitAction(fifthStream); action != "header" {
		t.Fatalf("fresh reconnect first action = %q, want header", action)
	}
	if action := waitAction(fifthStream); action != "crack-file-updated" {
		t.Fatalf("fresh reconnect second action = %q, want crack-file-updated", action)
	}
	select {
	case action := <-fifthStream.actions:
		t.Fatalf("fresh reconnect unexpectedly dispatched %q", action)
	case <-time.After(100 * time.Millisecond):
	}
	waitDone(fifthCancel, fifthDone)

	var count int64
	if err := database.Model(&models.Benchmark{}).Where("crackstation_id = ?", models.ParseUUIDOrNil(hostUUID)).Count(&count).Error; err != nil {
		t.Fatalf("count benchmarks: %v", err)
	}
	if count != 1 {
		t.Fatalf("benchmark count after reconnect = %d, want 1", count)
	}
	persisted, err = db.CrackstationByHostUUID(hostUUID)
	if err != nil {
		t.Fatalf("reload version-refreshed crackstation: %v", err)
	}
	if persisted.HashcatVersion != "hashcat-v2" || persisted.BenchmarkHashcatVersion != "hashcat-v2" || persisted.BenchmarkSchemaVersion != crackBenchmarkSchemaVersion || persisted.Benchmarks[0].PerSecondRate != 200 {
		t.Fatalf("version-refreshed crackstation = %#v", persisted)
	}
}

func TestCrackstationRegisterRefreshesLegacyNonemptyBenchmarks(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	originalReaperStarter := crackQueueReaperStarter
	crackQueueReaperStarter = func() {}
	t.Cleanup(func() { crackQueueReaperStarter = originalReaperStarter })
	hostID := models.NewUUID()
	operatorName := "legacy-benchmark-test"
	legacy := &models.Crackstation{ID: hostID, OperatorName: operatorName}
	if err := database.Omit("Tasks", "Benchmarks").Create(legacy).Error; err != nil {
		t.Fatalf("create legacy crackstation: %v", err)
	}
	if err := database.Create(&models.Benchmark{CrackstationID: hostID, HashType: int32(clientpb.HashType_MD5), PerSecondRate: 99}).Error; err != nil {
		t.Fatalf("create legacy partial benchmark: %v", err)
	}
	ctx, cancel := context.WithCancel(crackstationTestContext(operatorName))
	stream := newCrackstationRegisterTestStream(ctx)
	done := make(chan error, 1)
	go func() {
		done <- (&Server{}).CrackstationRegister(&clientpb.Crackstation{
			HostUUID: hostID.String(), Name: "legacy-worker", HashcatVersion: "hashcat-v1",
		}, stream)
	}()
	for _, want := range []string{"header", "crack-file-updated", "crack-benchmark"} {
		select {
		case got := <-stream.actions:
			if got != want {
				t.Fatalf("registration action = %q, want %q", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %q", want)
		}
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("legacy registration returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("legacy registration did not stop")
	}
}

func TestCrackstationRegisterRejectsOfflineHostOwnerTakeover(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	hostID := models.NewUUID()
	station := &models.Crackstation{ID: hostID, OperatorName: "station-owner"}
	if err := database.Omit("Tasks", "Benchmarks").Create(station).Error; err != nil {
		t.Fatalf("create owned crackstation: %v", err)
	}
	task := &models.CrackTask{
		CrackstationID: hostID,
		Kind:           int32(clientpb.CrackTaskKind_CRACK_TASK_CRACK),
		State:          int32(clientpb.CrackTaskState_CRACK_TASK_LEASED),
		Attempt:        2,
		LeaseToken:     "owner-only-token",
		LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := database.Create(task).Error; err != nil {
		t.Fatalf("create owned task: %v", err)
	}
	stream := newCrackstationRegisterTestStream(crackstationTestContext("attacker"))
	err := (&Server{}).CrackstationRegister(&clientpb.Crackstation{
		HostUUID: hostID.String(),
		Name:     "impersonated-worker",
	}, stream)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("offline owner takeover error = %v, want PermissionDenied", err)
	}
	if core.GetCrackstation(hostID.String()) != nil {
		t.Fatal("attacker was added to the runtime crackstation registry")
	}
	select {
	case action := <-stream.actions:
		t.Fatalf("attacker received registration action %q", action)
	default:
	}
	var persistedStation models.Crackstation
	if err := database.First(&persistedStation, "id = ?", hostID).Error; err != nil {
		t.Fatalf("load persisted crackstation: %v", err)
	}
	if persistedStation.OperatorName != "station-owner" {
		t.Fatalf("persisted owner = %q, want station-owner", persistedStation.OperatorName)
	}
	var persistedTask models.CrackTask
	if err := database.First(&persistedTask, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("load persisted task: %v", err)
	}
	if persistedTask.State != task.State || persistedTask.Attempt != task.Attempt || persistedTask.LeaseToken != task.LeaseToken || persistedTask.CrackstationID != hostID {
		t.Fatalf("owner task changed during takeover: %#v", persistedTask)
	}
}

func TestRegisterCrackstationRecordAtomicallyBindsLegacyOwner(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	hostID := models.NewUUID()
	if err := database.Omit("Tasks", "Benchmarks").Create(&models.Crackstation{ID: hostID}).Error; err != nil {
		t.Fatalf("create legacy crackstation: %v", err)
	}
	persisted, err := registerCrackstationRecord(hostID, "first-owner", "hashcat-v1")
	if err != nil {
		t.Fatalf("bind legacy crackstation owner: %v", err)
	}
	if persisted.OperatorName != "first-owner" {
		t.Fatalf("legacy owner = %q, want first-owner", persisted.OperatorName)
	}
	if _, err := registerCrackstationRecord(hostID, "different-owner", "hashcat-v1"); !errors.Is(err, errCrackstationOwnerMismatch) {
		t.Fatalf("different owner bind error = %v, want owner mismatch", err)
	}
	var count int64
	if err := database.Model(&models.Crackstation{}).Where("id = ?", hostID).Count(&count).Error; err != nil {
		t.Fatalf("count legacy crackstation: %v", err)
	}
	if count != 1 {
		t.Fatalf("legacy crackstation count = %d, want 1", count)
	}
}

func TestRegisterCrackstationRecordBindsLegacyNullOwner(t *testing.T) {
	database := setupCrackstationRPCTestDB(t)
	hostID := models.NewUUID()
	if err := database.Omit("Tasks", "Benchmarks").Create(&models.Crackstation{ID: hostID}).Error; err != nil {
		t.Fatalf("create legacy crackstation: %v", err)
	}
	if err := database.Model(&models.Crackstation{}).Where("id = ?", hostID).Update("operator_name", nil).Error; err != nil {
		t.Fatalf("set legacy null owner: %v", err)
	}
	var nullOwnerCount int64
	if err := database.Model(&models.Crackstation{}).Where("id = ? AND operator_name IS NULL", hostID).Count(&nullOwnerCount).Error; err != nil {
		t.Fatalf("count legacy null owner: %v", err)
	}
	if nullOwnerCount != 1 {
		t.Fatalf("legacy null owner count = %d, want 1", nullOwnerCount)
	}

	persisted, err := registerCrackstationRecord(hostID, "first-owner", "hashcat-v1")
	if err != nil {
		t.Fatalf("bind legacy null crackstation owner: %v", err)
	}
	if persisted.OperatorName != "first-owner" {
		t.Fatalf("legacy null owner = %q, want first-owner", persisted.OperatorName)
	}
	if _, err := registerCrackstationRecord(hostID, "different-owner", "hashcat-v1"); !errors.Is(err, errCrackstationOwnerMismatch) {
		t.Fatalf("different owner bind error = %v, want owner mismatch", err)
	}
}

func setupCrackstationRPCTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	originalDB := db.Client
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "crackstation-rpc.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	if err := database.AutoMigrate(
		&models.Crackstation{},
		&models.CrackJob{},
		&models.CrackTask{},
		&models.CrackCommand{},
		&models.CrackResult{},
		&models.CrackJobCredential{},
		&models.Credential{},
		&models.Benchmark{},
		&models.CrackFile{},
		&models.CrackFileChunk{},
	); err != nil {
		t.Fatalf("migrate crackstation models: %v", err)
	}
	db.Client = database
	t.Cleanup(func() {
		db.Client = originalDB
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return database
}
