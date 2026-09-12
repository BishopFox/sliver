package crack

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type crackTopRPCTestClient struct {
	calls atomic.Int32
	call  func(context.Context, *commonpb.Empty) (*clientpb.CrackTopSnapshot, error)
}

func (client *crackTopRPCTestClient) CrackTop(ctx context.Context, request *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.CrackTopSnapshot, error) {
	client.calls.Add(1)
	if client.call == nil {
		return nil, nil
	}
	return client.call(ctx, request)
}

//nolint:gocyclo // This test intentionally validates every field in one complete lean-snapshot conversion.
func TestLoadCrackTopSnapshotUsesOneRPCAndConvertsLeanSnapshot(t *testing.T) {
	hashMode := uint32(5600)
	wire := &clientpb.CrackTopSnapshot{
		ObservedAt: 1_700_000_123,
		Jobs: []*clientpb.CrackTopJob{
			{
				ID:          "job-one",
				CreatedAt:   "2026-09-12T10:00:00Z",
				CompletedAt: "",
				Status:      clientpb.CrackJobStatus_IN_PROGRESS,
				UpdatedAt:   1_700_000_120,
				ResultCount: 7,
				AttackMode:  clientpb.CrackAttackMode_BRUTEFORCE,
				HashType:    clientpb.HashType_NET_NTLM_V2,
				HashMode:    &hashMode,
				Tasks: []*clientpb.CrackTopTask{
					{
						ID:               "task-live",
						HostUUID:         "worker-one",
						CreatedAt:        11,
						StartedAt:        12,
						CompletedAt:      13,
						Kind:             clientpb.CrackTaskKind_CRACK_TASK_CRACK,
						State:            clientpb.CrackTaskState_CRACK_TASK_RUNNING,
						Attempt:          2,
						LeaseExpiresAt:   1_700_000_500,
						UpdatedAt:        14,
						LastHeartbeatAt:  15,
						ShardSkip:        100,
						ShardLimit:       200,
						ProgressCurrent:  "125",
						ProgressTotal:    "300",
						StatusAvailable:  true,
						StatusParseError: false,
						ProgressExact:    true,
						Devices: []*clientpb.CrackTopDevice{
							{
								ID:                   "gpu-0",
								Speed:                9_876_543,
								SpeedAvailable:       true,
								Temperature:          71.5,
								TemperatureAvailable: true,
								Utilization:          99.25,
								UtilizationAvailable: true,
							},
							nil,
							{ID: "gpu-1", Speed: 123, Temperature: 45, Utilization: 67},
						},
					},
					nil,
					{
						ID:               "task-malformed",
						HostUUID:         "worker-two",
						Kind:             clientpb.CrackTaskKind_CRACK_TASK_CRACK,
						State:            clientpb.CrackTaskState_CRACK_TASK_RUNNING,
						ProgressCurrent:  "sensitive-raw-data-must-not-be-rebuilt",
						StatusAvailable:  true,
						StatusParseError: true,
					},
				},
			},
			nil,
			{ID: "job-two", Status: clientpb.CrackJobStatus_COMPLETED},
		},
		Crackstations: []*clientpb.CrackTopStation{
			{
				ID:                "station-one",
				Name:              "rig-one",
				HostUUID:          "worker-one",
				State:             clientpb.States_CRACKING,
				CurrentCrackJobID: "job-one",
				IsSyncing:         false,
				StatusAvailable:   true,
			},
			nil,
			{ID: "station-two", Name: "rig-two", HostUUID: "worker-two"},
		},
	}
	wireBefore := proto.Clone(wire).(*clientpb.CrackTopSnapshot)
	client := &crackTopRPCTestClient{
		call: func(ctx context.Context, request *commonpb.Empty) (*clientpb.CrackTopSnapshot, error) {
			if ctx == nil {
				t.Fatal("CrackTop received a nil context")
			}
			if request == nil {
				t.Fatal("CrackTop received a nil request")
			}
			return wire, nil
		},
	}

	fallback := time.Unix(1_600_000_000, 987_654_321)
	snapshot, err := loadCrackTopSnapshot(t.Context(), client, time.Second, fallback)
	if err != nil {
		t.Fatalf("loadCrackTopSnapshot: %v", err)
	}
	if client.calls.Load() != 1 {
		t.Fatalf("CrackTop calls = %d, want 1", client.calls.Load())
	}
	if want := time.Unix(wire.GetObservedAt(), 0); !snapshot.RefreshedAt.Equal(want) {
		t.Fatalf("RefreshedAt = %s, want server observation time %s", snapshot.RefreshedAt, want)
	}
	if !proto.Equal(wire, wireBefore) {
		t.Fatal("loader mutated the wire snapshot")
	}

	if len(snapshot.Jobs) != 3 || snapshot.Jobs[1] != nil || snapshot.Jobs[2].GetID() != "job-two" {
		t.Fatalf("jobs did not preserve wire order and nil entries: %#v", snapshot.Jobs)
	}
	job := snapshot.Jobs[0]
	if job.GetID() != "job-one" || job.GetCreatedAt() != wire.Jobs[0].GetCreatedAt() || job.GetCompletedAt() != wire.Jobs[0].GetCompletedAt() ||
		job.GetStatus() != clientpb.CrackJobStatus_IN_PROGRESS || job.GetUpdatedAt() != 1_700_000_120 || job.GetResultCount() != 7 {
		t.Fatalf("converted job metadata = %#v", job)
	}
	if job.GetCommand() == nil || job.GetCommand().GetAttackMode() != clientpb.CrackAttackMode_BRUTEFORCE ||
		job.GetCommand().GetHashType() != clientpb.HashType_NET_NTLM_V2 || job.GetCommand().HashMode == nil || job.GetCommand().GetHashMode() != hashMode {
		t.Fatalf("converted job command = %#v", job.GetCommand())
	}
	if job.GetErr() != "" || job.GetResultFileID() != "" || job.GetKeyspace() != "" || len(job.GetResults()) != 0 ||
		len(job.GetCommand().GetHashes()) != 0 || len(job.GetCommand().GetPositionalArguments()) != 0 || len(job.GetCommand().GetCredentialIDs()) != 0 {
		t.Fatalf("lean job conversion populated target or result fields: %#v", job)
	}
	if len(job.GetTasks()) != 3 || job.GetTasks()[1] != nil {
		t.Fatalf("tasks did not preserve wire order and nil entries: %#v", job.GetTasks())
	}
	task := job.GetTasks()[0]
	if task.GetID() != "task-live" || task.GetHostUUID() != "worker-one" || task.GetCreatedAt() != 11 || task.GetStartedAt() != 12 ||
		task.GetCompletedAt() != 13 || task.GetKind() != clientpb.CrackTaskKind_CRACK_TASK_CRACK ||
		task.GetState() != clientpb.CrackTaskState_CRACK_TASK_RUNNING || task.GetAttempt() != 2 ||
		task.GetLeaseExpiresAt() != 1_700_000_500 || task.GetUpdatedAt() != 14 || task.GetLastHeartbeatAt() != 15 ||
		task.GetShardSkip() != 100 || task.GetShardLimit() != 200 {
		t.Fatalf("converted task metadata = %#v", task)
	}
	if task.GetErr() != "" || len(task.GetStdout()) != 0 || len(task.GetStderr()) != 0 || task.GetCommand() != nil ||
		task.GetLeaseToken() != "" || task.GetKeyspace() != "" || len(task.GetLatestStatusJSON()) != 0 || len(task.GetRecoveredJSON()) != 0 {
		t.Fatalf("lean task conversion populated execution or raw telemetry fields: %#v", task)
	}

	metadata, ok := snapshot.taskTelemetry["task-live"]
	if !ok || metadata.parseErr || metadata.status == nil {
		t.Fatalf("live task telemetry = %#v, present=%v", metadata, ok)
	}
	if !metadata.progressExact {
		t.Fatal("server-declared exact progress was discarded")
	}
	if metadata.status.ProgressCurrent != "125" || metadata.status.ProgressTotal != "300" || len(metadata.status.Devices) != 2 {
		t.Fatalf("converted status = %#v", metadata.status)
	}
	firstDevice := metadata.status.Devices[0]
	if firstDevice.ID != "gpu-0" || firstDevice.Speed != "9876543" || firstDevice.Temperature != "71.5" || firstDevice.Utilization != "99.25" {
		t.Fatalf("converted available device telemetry = %#v", firstDevice)
	}
	secondDevice := metadata.status.Devices[1]
	if secondDevice.ID != "gpu-1" || secondDevice.Speed != "" || secondDevice.Temperature != "" || secondDevice.Utilization != "" {
		t.Fatalf("unavailable device telemetry was treated as available: %#v", secondDevice)
	}
	malformed, ok := snapshot.taskTelemetry["task-malformed"]
	if !ok || !malformed.parseErr || malformed.status != nil {
		t.Fatalf("malformed task telemetry = %#v, present=%v", malformed, ok)
	}

	if len(snapshot.Stations) != 3 || snapshot.Stations[1] != nil {
		t.Fatalf("stations did not preserve wire order and nil entries: %#v", snapshot.Stations)
	}
	station := snapshot.Stations[0]
	if station.GetID() != "station-one" || station.GetName() != "rig-one" || station.GetHostUUID() != "worker-one" || station.GetStatus() == nil ||
		station.GetStatus().GetName() != "rig-one" || station.GetStatus().GetHostUUID() != "worker-one" ||
		station.GetStatus().GetState() != clientpb.States_CRACKING || station.GetStatus().GetCurrentCrackJobID() != "job-one" || station.GetStatus().GetIsSyncing() {
		t.Fatalf("converted station = %#v", station)
	}
	if station.GetOperatorName() != "" || station.GetGOOS() != "" || station.GetGOARCH() != "" || station.GetVersion() != "" ||
		len(station.GetBenchmarks()) != 0 || len(station.GetCUDA()) != 0 || len(station.GetMetal()) != 0 || len(station.GetOpenCL()) != 0 ||
		len(station.GetHIP()) != 0 || len(station.GetCapabilities()) != 0 {
		t.Fatalf("lean station conversion populated private inventory fields: %#v", station)
	}
	if snapshot.Stations[2].GetStatus() != nil {
		t.Fatalf("station without available status got a fabricated status: %#v", snapshot.Stations[2])
	}

	*snapshot.Jobs[0].Command.HashMode = 1000
	snapshot.Jobs[0].Tasks[0].ID = "changed-task"
	snapshot.Stations[0].Status.Name = "changed-station"
	metadata.status.Devices[0].ID = "changed-device"
	if wire.Jobs[0].GetHashMode() != 5600 || wire.Jobs[0].Tasks[0].GetID() != "task-live" ||
		wire.Crackstations[0].GetName() != "rig-one" || wire.Jobs[0].Tasks[0].Devices[0].GetID() != "gpu-0" {
		t.Fatal("converted snapshot aliases mutable wire data")
	}
}

//nolint:gocyclo // This test intentionally covers the full loader-to-dashboard telemetry path.
func TestLoadCrackTopSnapshotFeedsDashboardTelemetry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	client := &crackTopRPCTestClient{
		call: func(context.Context, *commonpb.Empty) (*clientpb.CrackTopSnapshot, error) {
			return &clientpb.CrackTopSnapshot{
				ObservedAt: now.Unix(),
				Jobs: []*clientpb.CrackTopJob{{
					ID:     crackTopTestJobID,
					Status: clientpb.CrackJobStatus_IN_PROGRESS,
					Tasks: []*clientpb.CrackTopTask{{
						ID:              "task-wire",
						HostUUID:        "worker-wire",
						Kind:            clientpb.CrackTaskKind_CRACK_TASK_UNSPECIFIED,
						State:           clientpb.CrackTaskState_CRACK_TASK_RUNNING,
						LeaseExpiresAt:  now.Add(time.Minute).Unix(),
						LastHeartbeatAt: now.Unix(),
						ProgressCurrent: "25",
						ProgressTotal:   "100",
						StatusAvailable: true,
						ProgressExact:   true,
						Devices: []*clientpb.CrackTopDevice{{
							ID:                   "gpu-0",
							Speed:                125_000,
							SpeedAvailable:       true,
							Temperature:          67,
							TemperatureAvailable: true,
							Utilization:          98,
							UtilizationAvailable: true,
						}},
					}},
				}},
				Crackstations: []*clientpb.CrackTopStation{{
					ID:                "station-wire",
					Name:              "Wire Worker",
					HostUUID:          "worker-wire",
					State:             clientpb.States_CRACKING,
					CurrentCrackJobID: crackTopTestJobID,
					StatusAvailable:   true,
				}},
			}, nil
		},
	}

	snapshot, err := loadCrackTopSnapshot(t.Context(), client, time.Second, time.Time{})
	if err != nil {
		t.Fatalf("loadCrackTopSnapshot: %v", err)
	}
	dashboard := buildCrackTopDashboard(snapshot)
	if len(dashboard.Jobs) != 1 || !dashboard.Jobs[0].ProgressKnown || dashboard.Jobs[0].Progress != 0.25 {
		t.Fatalf("job progress from narrow telemetry = %#v", dashboard.Jobs)
	}
	if !dashboard.ClusterRateKnown || !dashboard.ClusterComplete || dashboard.ClusterRate != 125_000 {
		t.Fatalf("cluster telemetry = rate:%d known:%v complete:%v", dashboard.ClusterRate, dashboard.ClusterRateKnown, dashboard.ClusterComplete)
	}
	if len(dashboard.Workers) != 1 {
		t.Fatalf("workers = %#v, want one", dashboard.Workers)
	}
	worker := dashboard.Workers[0]
	if !worker.TelemetryLive || !worker.RateKnown || worker.Rate != 125_000 || worker.Devices != 1 ||
		!worker.HasTemp || worker.Temperature != 67 || !worker.HasUtil || worker.Utilization != 98 {
		t.Fatalf("worker telemetry from narrow snapshot = %#v", worker)
	}
}

func TestLoadCrackTopSnapshotUsesCompletionTimeOnlyWithoutObservedAt(t *testing.T) {
	fallback := time.Unix(1_700_000_000, 123_456_789)
	client := &crackTopRPCTestClient{
		call: func(context.Context, *commonpb.Empty) (*clientpb.CrackTopSnapshot, error) {
			return &clientpb.CrackTopSnapshot{}, nil
		},
	}

	snapshot, err := loadCrackTopSnapshot(t.Context(), client, 0, fallback)
	if err != nil {
		t.Fatalf("loadCrackTopSnapshot: %v", err)
	}
	if !snapshot.RefreshedAt.Equal(fallback) {
		t.Fatalf("RefreshedAt = %s, want deterministic completion time %s", snapshot.RefreshedAt, fallback)
	}
	if client.calls.Load() != 1 {
		t.Fatalf("CrackTop calls = %d, want 1", client.calls.Load())
	}
}

func TestLoadCrackTopSnapshotPropagatesDeadlineAndCancelsCallContext(t *testing.T) {
	var callContext context.Context
	var deadline time.Time
	client := &crackTopRPCTestClient{
		call: func(ctx context.Context, _ *commonpb.Empty) (*clientpb.CrackTopSnapshot, error) {
			callContext = ctx
			var ok bool
			deadline, ok = ctx.Deadline()
			if !ok {
				t.Fatal("CrackTop context has no deadline")
			}
			return &clientpb.CrackTopSnapshot{ObservedAt: 1}, nil
		},
	}

	started := time.Now()
	timeout := time.Second
	_, err := loadCrackTopSnapshot(t.Context(), client, timeout, time.Time{})
	if err != nil {
		t.Fatalf("loadCrackTopSnapshot: %v", err)
	}
	if client.calls.Load() != 1 {
		t.Fatalf("CrackTop calls = %d, want 1", client.calls.Load())
	}
	if deadline.Before(started) || deadline.After(started.Add(timeout+100*time.Millisecond)) {
		t.Fatalf("RPC deadline = %s, want about %s after %s", deadline, timeout, started)
	}
	select {
	case <-callContext.Done():
		if !errors.Is(callContext.Err(), context.Canceled) {
			t.Fatalf("completed call context error = %v, want context canceled", callContext.Err())
		}
	default:
		t.Fatal("completed call context was not canceled")
	}
}

func TestLoadCrackTopSnapshotReturnsNilOnTimeout(t *testing.T) {
	client := &crackTopRPCTestClient{
		call: func(ctx context.Context, _ *commonpb.Empty) (*clientpb.CrackTopSnapshot, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	snapshot, err := loadCrackTopSnapshot(t.Context(), client, 10*time.Millisecond, time.Time{})
	if snapshot != nil {
		t.Fatalf("timed-out load returned snapshot: %#v", snapshot)
	}
	if err == nil || err.Error() != "load crack top snapshot: context deadline exceeded" {
		t.Fatalf("timed-out load error = %v", err)
	}
	if client.calls.Load() != 1 {
		t.Fatalf("CrackTop calls = %d, want 1", client.calls.Load())
	}
}

func TestLoadCrackTopSnapshotPropagatesParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	client := &crackTopRPCTestClient{
		call: func(ctx context.Context, _ *commonpb.Empty) (*clientpb.CrackTopSnapshot, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	snapshot, err := loadCrackTopSnapshot(ctx, client, time.Second, time.Time{})
	if snapshot != nil {
		t.Fatalf("canceled load returned snapshot: %#v", snapshot)
	}
	if err == nil || err.Error() != "load crack top snapshot: context canceled" {
		t.Fatalf("canceled load error = %v", err)
	}
	if client.calls.Load() != 1 {
		t.Fatalf("CrackTop calls = %d, want 1", client.calls.Load())
	}
}

func TestLoadCrackTopSnapshotSanitizesRPCErrorAndRejectsEmptyResponse(t *testing.T) {
	t.Run("rpc error", func(t *testing.T) {
		raw := errors.New("bad\r\n\x1b[31mserver\x1b[0m\tfailure")
		client := &crackTopRPCTestClient{
			call: func(context.Context, *commonpb.Empty) (*clientpb.CrackTopSnapshot, error) {
				return nil, raw
			},
		}

		snapshot, err := loadCrackTopSnapshot(t.Context(), client, time.Second, time.Time{})
		if snapshot != nil {
			t.Fatalf("failed load returned snapshot: %#v", snapshot)
		}
		if err == nil || err.Error() != "load crack top snapshot: bad server failure" {
			t.Fatalf("sanitized error = %v", err)
		}
		if errors.Is(err, raw) {
			t.Fatal("sanitized error retained the raw RPC error in its unwrap chain")
		}
		if strings.ContainsAny(err.Error(), "\r\n\t\x1b") {
			t.Fatalf("error still contains terminal controls: %q", err)
		}
		if client.calls.Load() != 1 {
			t.Fatalf("CrackTop calls = %d, want 1", client.calls.Load())
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &crackTopRPCTestClient{}
		snapshot, err := loadCrackTopSnapshot(t.Context(), client, time.Second, time.Time{})
		if snapshot != nil {
			t.Fatalf("empty load returned snapshot: %#v", snapshot)
		}
		if err == nil || err.Error() != "load crack top snapshot: server returned an empty response" {
			t.Fatalf("empty response error = %v", err)
		}
		if client.calls.Load() != 1 {
			t.Fatalf("CrackTop calls = %d, want 1", client.calls.Load())
		}
	})
}
