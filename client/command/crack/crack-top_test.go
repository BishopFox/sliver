package crack

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/x/ansi"
)

const crackTopTestJobID = "aaaaaaaa-1111-4111-8111-111111111111"

func crackTopTestStatus(current, total string, devices ...string) []byte {
	return []byte(fmt.Sprintf(
		`{"session":"top-test","status":2,"progress":[%s,%s],"devices":[%s]}`,
		current,
		total,
		strings.Join(devices, ","),
	))
}

func crackTopTestStation(id, name, jobID string, state clientpb.States) *clientpb.Crackstation {
	return &clientpb.Crackstation{
		HostUUID: id,
		Name:     name,
		Status: &clientpb.CrackstationStatus{
			HostUUID:          id,
			Name:              name,
			State:             state,
			CurrentCrackJobID: jobID,
		},
	}
}

func crackTopTestTask(id, hostID string, kind clientpb.CrackTaskKind, state clientpb.CrackTaskState, limit uint64, leaseExpiresAt int64, status []byte) *clientpb.CrackTask {
	return &clientpb.CrackTask{
		ID:               id,
		HostUUID:         hostID,
		Kind:             kind,
		State:            state,
		ShardLimit:       limit,
		LeaseExpiresAt:   leaseExpiresAt,
		LastHeartbeatAt:  leaseExpiresAt - 60,
		LatestStatusJSON: status,
	}
}

func crackTopLiveTestSnapshot() *crackTopSnapshot {
	now := time.Unix(1_700_000_000, 0)
	firstStatus := crackTopTestStatus("50", "100",
		`{"device_id":1,"device_name":"GPU 0","device_type":"GPU","speed":1500,"temp":61,"util":98}`,
		`{"device_id":2,"device_name":"GPU 1","device_type":"GPU","speed":500,"temp":65,"util":96}`,
	)
	secondStatus := crackTopTestStatus("75", "300",
		`{"device_id":1,"device_name":"GPU 0","device_type":"GPU","speed":1000000,"temp":68,"util":97}`,
		`{"device_id":2,"device_name":"GPU 1","device_type":"GPU","speed":1500000,"temp":70,"util":99}`,
	)
	job := &clientpb.CrackJob{
		ID:          crackTopTestJobID,
		CreatedAt:   "2026-09-12T00:00:00Z",
		UpdatedAt:   now.Unix(),
		Status:      clientpb.CrackJobStatus_IN_PROGRESS,
		Keyspace:    "400",
		ResultCount: 7,
		Command:     &clientpb.CrackCommand{HashType: clientpb.HashType_MD5},
		Tasks: []*clientpb.CrackTask{
			crackTopTestTask("task-alpha", "worker01", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), firstStatus),
			crackTopTestTask("task-beta", "worker02", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 300, now.Add(time.Minute).Unix(), secondStatus),
		},
	}
	return &crackTopSnapshot{
		Jobs: []*clientpb.CrackJob{job},
		Stations: []*clientpb.Crackstation{
			crackTopTestStation("worker01", "Alpha Rig", job.ID, clientpb.States_CRACKING),
			crackTopTestStation("worker02", "Beta Rig", job.ID, clientpb.States_CRACKING),
		},
		RefreshedAt: now,
	}
}

func crackTopWorkerRowsByID(rows []crackTopWorkerRow) map[string]crackTopWorkerRow {
	result := make(map[string]crackTopWorkerRow, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}
	return result
}

func crackTopJobRowIDs(rows []crackTopJobRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func crackTopWorkerRowIDs(rows []crackTopWorkerRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func TestBuildCrackTopDashboardWeightsUnequalShardsAndAggregatesDeviceRates(t *testing.T) {
	dashboard := buildCrackTopDashboard(crackTopLiveTestSnapshot())
	if len(dashboard.Jobs) != 1 {
		t.Fatalf("jobs = %#v, want one", dashboard.Jobs)
	}
	job := dashboard.Jobs[0]
	// (50% * 100 + 25% * 300) / 400 = 31.25%. A simple mean would
	// incorrectly report 37.5%.
	const wantProgress = 0.3125
	if !job.ProgressKnown || math.Abs(job.Progress-wantProgress) > 1e-12 {
		t.Fatalf("job progress = %.12f known=%v, want %.12f", job.Progress, job.ProgressKnown, wantProgress)
	}
	if !dashboard.ProgressKnown || math.Abs(dashboard.Progress-wantProgress) > 1e-12 {
		t.Fatalf("global progress = %.12f known=%v, want %.12f", dashboard.Progress, dashboard.ProgressKnown, wantProgress)
	}
	if !job.RateKnown || job.Rate != 2_502_000 || job.Reporting != 2 {
		t.Fatalf("job rate/reporting = %d/%v, %d; want 2502000/true, 2", job.Rate, job.RateKnown, job.Reporting)
	}
	if !dashboard.ClusterRateKnown || dashboard.ClusterRate != 2_502_000 {
		t.Fatalf("cluster rate = %d known=%v, want 2502000/true", dashboard.ClusterRate, dashboard.ClusterRateKnown)
	}
	if dashboard.Recovered != 7 || dashboard.ActiveJobs != 1 || dashboard.OnlineWorkers != 2 {
		t.Fatalf("dashboard counters = active:%d online:%d recovered:%d", dashboard.ActiveJobs, dashboard.OnlineWorkers, dashboard.Recovered)
	}

	workers := crackTopWorkerRowsByID(dashboard.Workers)
	alpha, alphaOK := workers["worker01"]
	beta, betaOK := workers["worker02"]
	if !alphaOK || !betaOK {
		t.Fatalf("workers = %#v, want worker01 and worker02", dashboard.Workers)
	}
	if !alpha.RateKnown || alpha.Rate != 2_000 || alpha.Devices != 2 {
		t.Fatalf("alpha telemetry = rate:%d known:%v devices:%d", alpha.Rate, alpha.RateKnown, alpha.Devices)
	}
	if !beta.RateKnown || beta.Rate != 2_500_000 || beta.Devices != 2 {
		t.Fatalf("beta telemetry = rate:%d known:%v devices:%d", beta.Rate, beta.RateKnown, beta.Devices)
	}
}

func TestCrackTopProgressUsesExactServerNormalizedCountersAndClamps(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, testCase := range []struct {
		name         string
		current      string
		total        string
		wantProgress float64
	}{
		{name: "normalized nonzero skip", current: "1", total: "4", wantProgress: 0.25},
		{name: "before shard start", current: "0", total: "1", wantProgress: 0},
		{name: "past shard end", current: "5", total: "4", wantProgress: 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			task := crackTopTestTask(
				"shard",
				"worker",
				clientpb.CrackTaskKind_CRACK_TASK_CRACK,
				clientpb.CrackTaskState_CRACK_TASK_RUNNING,
				400,
				now.Add(time.Minute).Unix(),
				nil,
			)
			task.ShardSkip = 1_000
			job := &clientpb.CrackJob{
				ID:     crackTopTestJobID,
				Status: clientpb.CrackJobStatus_IN_PROGRESS,
				Tasks:  []*clientpb.CrackTask{task},
			}
			dashboard := buildCrackTopDashboard(&crackTopSnapshot{
				Jobs:        []*clientpb.CrackJob{job},
				RefreshedAt: now,
				taskTelemetry: map[string]crackTopTaskTelemetry{
					task.GetID(): {
						status: &crackStatusView{
							ProgressCurrent: testCase.current,
							ProgressTotal:   testCase.total,
						},
						progressExact: true,
					},
				},
			})
			if len(dashboard.Jobs) != 1 || !dashboard.Jobs[0].ProgressKnown {
				t.Fatalf("progress dashboard = %#v, want one known job", dashboard)
			}
			if got := dashboard.Jobs[0].Progress; math.Abs(got-testCase.wantProgress) > 1e-12 {
				t.Fatalf("shard-relative progress = %.12f, want %.12f", got, testCase.wantProgress)
			}
			if !dashboard.ProgressKnown || math.Abs(dashboard.Progress-testCase.wantProgress) > 1e-12 {
				t.Fatalf("global shard-relative progress = %.12f known=%v, want %.12f", dashboard.Progress, dashboard.ProgressKnown, testCase.wantProgress)
			}
		})
	}
}

func TestCrackTopUnknownNonzeroSkipProgressDoesNotHideLiveRate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	task := crackTopTestTask(
		"unknown-progress",
		"worker",
		clientpb.CrackTaskKind_CRACK_TASK_CRACK,
		clientpb.CrackTaskState_CRACK_TASK_RUNNING,
		20_000_000,
		now.Add(time.Minute).Unix(),
		nil,
	)
	task.ShardSkip = 1_000_000
	job := &clientpb.CrackJob{
		ID:     crackTopTestJobID,
		Status: clientpb.CrackJobStatus_IN_PROGRESS,
		Tasks:  []*clientpb.CrackTask{task},
	}
	dashboard := buildCrackTopDashboard(&crackTopSnapshot{
		Jobs:        []*clientpb.CrackJob{job},
		Stations:    []*clientpb.Crackstation{crackTopTestStation("worker", "Worker", job.ID, clientpb.States_CRACKING)},
		RefreshedAt: now,
		taskTelemetry: map[string]crackTopTaskTelemetry{
			task.GetID(): {
				status: &crackStatusView{
					ProgressCurrent: "5898240000",
					ProgressTotal:   "20000000000",
					Devices:         []crackDeviceView{{ID: "0", Speed: "1234"}},
				},
			},
		},
	})
	if len(dashboard.Jobs) != 1 || !dashboard.Jobs[0].ProgressKnown || dashboard.Jobs[0].ProgressComplete || dashboard.Jobs[0].Progress != 0 || dashboard.ProgressKnown {
		t.Fatalf("unknown shard semantics did not produce a zero lower bound: %#v", dashboard)
	}
	if !dashboard.Jobs[0].RateKnown || dashboard.Jobs[0].Rate != 1234 || !dashboard.ClusterRateKnown || dashboard.ClusterRate != 1234 {
		t.Fatalf("unknown progress semantics hid valid live rate: %#v", dashboard)
	}
}

func TestCrackTopPartialProgressIsWeightedLowerBound(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	completed := crackTopTestTask(
		"completed-small",
		"worker-one",
		clientpb.CrackTaskKind_CRACK_TASK_CRACK,
		clientpb.CrackTaskState_CRACK_TASK_COMPLETED,
		100,
		0,
		nil,
	)
	unknown := crackTopTestTask(
		"unknown-large",
		"worker-two",
		clientpb.CrackTaskKind_CRACK_TASK_CRACK,
		clientpb.CrackTaskState_CRACK_TASK_RUNNING,
		900,
		now.Add(time.Minute).Unix(),
		nil,
	)
	job := &clientpb.CrackJob{
		ID:     crackTopTestJobID,
		Status: clientpb.CrackJobStatus_IN_PROGRESS,
		Tasks:  []*clientpb.CrackTask{completed, unknown},
	}
	dashboard := buildCrackTopDashboard(&crackTopSnapshot{Jobs: []*clientpb.CrackJob{job}, RefreshedAt: now})
	if len(dashboard.Jobs) != 1 || !dashboard.Jobs[0].ProgressKnown || dashboard.Jobs[0].ProgressComplete ||
		math.Abs(dashboard.Jobs[0].Progress-0.1) > 1e-12 {
		t.Fatalf("partial weighted progress = %#v, want a 10%% lower bound", dashboard.Jobs)
	}
	if dashboard.ProgressKnown {
		t.Fatalf("partial per-job progress became exact global progress: %#v", dashboard)
	}

	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.dashboard = dashboard
	model.normalizeJobSelection()
	pane := ansi.Strip(model.renderJobsPane(90, 10))
	if !strings.Contains(pane, "≥ 10.0%") {
		t.Fatalf("partial progress was not labeled as a lower bound:\n%s", pane)
	}
}

func TestCrackTopProgressParserRejectsExpensiveOrNonCounterSyntax(t *testing.T) {
	for _, value := range []string{"1e1000000", "1/2", "-1", "1.5", strings.Repeat("9", crackTopMaxProgressTextSize+1)} {
		if _, ok := crackTopRat(value); ok {
			t.Errorf("accepted invalid progress counter %q", value)
		}
	}
	for _, value := range []string{"0", "12345678901234567890"} {
		if _, ok := crackTopRat(value); !ok {
			t.Errorf("rejected valid progress counter %q", value)
		}
	}
}

func TestCrackTopTaskSampleRequiresCurrentLiveStationAssignment(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	task := crackTopTestTask(
		"task",
		"worker",
		clientpb.CrackTaskKind_CRACK_TASK_CRACK,
		clientpb.CrackTaskState_CRACK_TASK_RUNNING,
		100,
		now.Add(time.Minute).Unix(),
		crackTopTestStatus("50", "100", `{"device_id":1,"speed":100}`),
	)
	status, err := crackTaskStatus(task)
	if err != nil {
		t.Fatalf("parse live task status: %v", err)
	}
	sample := crackTopTaskSample{
		jobID:    crackTopTestJobID,
		task:     task,
		status:   status,
		lastSeen: now,
	}
	for _, testCase := range []struct {
		name    string
		station *clientpb.Crackstation
		want    bool
	}{
		{name: "no connected station", station: nil},
		{name: "station status unavailable", station: &clientpb.Crackstation{HostUUID: "worker", Name: "Worker"}},
		{name: "blank current job", station: crackTopTestStation("worker", "Worker", "", clientpb.States_CRACKING)},
		{name: "different current job", station: crackTopTestStation("worker", "Worker", "different-job", clientpb.States_CRACKING)},
		{name: "idle station", station: crackTopTestStation("worker", "Worker", crackTopTestJobID, clientpb.States_IDLE)},
		{name: "exact active assignment", station: crackTopTestStation("worker", "Worker", crackTopTestJobID, clientpb.States_CRACKING), want: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := crackTopTaskSampleIsLive(sample, testCase.station, now); got != testCase.want {
				t.Fatalf("sample live = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestCrackTopHidesStaleAndOfflineDeviceTelemetry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	stale := crackTopTestTask(
		"stale-task",
		"stale",
		clientpb.CrackTaskKind_CRACK_TASK_CRACK,
		clientpb.CrackTaskState_CRACK_TASK_RUNNING,
		100,
		now.Unix(),
		crackTopTestStatus("50", "100", `{"device_id":1,"speed":111,"temp":99,"util":95}`),
	)
	stale.LastHeartbeatAt = now.Add(-time.Minute).Unix()
	offline := crackTopTestTask(
		"offline-task",
		"offline",
		clientpb.CrackTaskKind_CRACK_TASK_CRACK,
		clientpb.CrackTaskState_CRACK_TASK_RUNNING,
		100,
		now.Add(time.Minute).Unix(),
		crackTopTestStatus("50", "100", `{"device_id":1,"speed":222,"temp":88,"util":94}`),
	)
	offline.LastHeartbeatAt = now.Unix()
	job := &clientpb.CrackJob{
		ID:     crackTopTestJobID,
		Status: clientpb.CrackJobStatus_IN_PROGRESS,
		Tasks:  []*clientpb.CrackTask{stale, offline},
	}
	snapshot := &crackTopSnapshot{
		Jobs:        []*clientpb.CrackJob{job},
		Stations:    []*clientpb.Crackstation{crackTopTestStation("stale", "Stale Worker", job.ID, clientpb.States_CRACKING)},
		RefreshedAt: now,
	}
	dashboard := buildCrackTopDashboard(snapshot)
	workers := crackTopWorkerRowsByID(dashboard.Workers)
	for _, id := range []string{"stale", "offline"} {
		worker, ok := workers[id]
		if !ok {
			t.Fatalf("worker %q missing from %#v", id, dashboard.Workers)
		}
		if worker.TelemetryLive || worker.RateKnown || worker.Devices != 0 || worker.HasTemp || worker.HasUtil {
			t.Errorf("worker %q exposed non-live device telemetry: %#v", id, worker)
		}
	}

	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.snapshot = snapshot
	model.dashboard = dashboard
	model.refreshing = false
	model.width = 132
	model.height = 30
	model.normalizeJobSelection()
	plain := ansi.Strip(model.View().Content)
	for _, hidden := range []string{"111 H/s", "222 H/s", "99 C", "88 C", "95% util", "94% util"} {
		if strings.Contains(plain, hidden) {
			t.Errorf("view exposed stale/offline telemetry %q:\n%s", hidden, plain)
		}
	}
}

func TestCrackTopDashboardExcludesUnavailableAndNonCrackTaskRates(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	status := func(rate uint64) []byte {
		return crackTopTestStatus("50", "100", fmt.Sprintf(`{"device_id":1,"speed":%d}`, rate))
	}
	stations := []*clientpb.Crackstation{
		crackTopTestStation("live", "Live", crackTopTestJobID, clientpb.States_CRACKING),
		crackTopTestStation("idle", "Idle", "", clientpb.States_IDLE),
		crackTopTestStation("expired", "Expired", crackTopTestJobID, clientpb.States_CRACKING),
		crackTopTestStation("query", "Query", crackTopTestJobID, clientpb.States_CRACKING),
		crackTopTestStation("syncing", "Syncing", crackTopTestJobID, clientpb.States_CRACKING),
	}
	stations[4].Status.IsSyncing = true
	job := &clientpb.CrackJob{
		ID:     crackTopTestJobID,
		Status: clientpb.CrackJobStatus_IN_PROGRESS,
		Tasks: []*clientpb.CrackTask{
			crackTopTestTask("live", "live", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), status(100)),
			crackTopTestTask("offline", "offline", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), status(200)),
			crackTopTestTask("idle", "idle", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), status(300)),
			crackTopTestTask("expired", "expired", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Unix(), status(400)),
			crackTopTestTask("syncing", "syncing", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), status(500)),
			crackTopTestTask("query", "query", clientpb.CrackTaskKind_CRACK_TASK_QUERY, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 10_000, now.Add(time.Minute).Unix(), status(900_000)),
		},
	}
	dashboard := buildCrackTopDashboard(&crackTopSnapshot{Jobs: []*clientpb.CrackJob{job}, Stations: stations, RefreshedAt: now})
	if dashboard.ClusterRate != 100 || !dashboard.ClusterRateKnown {
		t.Fatalf("cluster rate = %d known=%v, want only live worker rate 100", dashboard.ClusterRate, dashboard.ClusterRateKnown)
	}
	if len(dashboard.Jobs) != 1 || dashboard.Jobs[0].Rate != 100 || dashboard.Jobs[0].Reporting != 1 {
		t.Fatalf("job telemetry = %#v, want only one reporting worker at 100 H/s", dashboard.Jobs)
	}
	if dashboard.Jobs[0].Workers != 5 {
		t.Fatalf("crack-task workers = %d, want 5 (query task excluded)", dashboard.Jobs[0].Workers)
	}
	if !dashboard.Jobs[0].ProgressKnown || math.Abs(dashboard.Jobs[0].Progress-0.5) > 1e-12 {
		t.Fatalf("progress = %.12f known=%v, want 0.5 with query task excluded", dashboard.Jobs[0].Progress, dashboard.Jobs[0].ProgressKnown)
	}
	workers := crackTopWorkerRowsByID(dashboard.Workers)
	if workers["offline"].Online || workers["offline"].RateKnown || dashboard.Unavailable != 1 {
		t.Fatalf("offline telemetry = %#v unavailable=%d", workers["offline"], dashboard.Unavailable)
	}
	for _, id := range []string{"idle", "expired", "syncing", "query"} {
		if workers[id].RateKnown {
			t.Errorf("worker %q unexpectedly contributes rate %#v", id, workers[id])
		}
	}
	if workers["query"].Devices != 0 {
		t.Fatalf("non-crack task populated device telemetry: %#v", workers["query"])
	}
}

func TestCrackTopClusterRateRemainsUnknownWhenConnectedStatusIsIncomplete(t *testing.T) {
	stations := []*clientpb.Crackstation{
		{HostUUID: "missing-status", Name: "Missing Status"},
		crackTopTestStation("missing-job", "Missing Job", "", clientpb.States_CRACKING),
	}
	for _, station := range stations {
		dashboard := buildCrackTopDashboard(&crackTopSnapshot{
			Stations:    []*clientpb.Crackstation{station},
			RefreshedAt: time.Unix(1_700_000_000, 0),
		})
		if dashboard.ClusterRateKnown || dashboard.ClusterComplete || dashboard.ExpectedWorkers != 1 || dashboard.ReportingWorkers != 0 {
			t.Errorf("incomplete connected station produced an exact cluster rate: %#v", dashboard)
		}
	}

	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.dashboard = buildCrackTopDashboard(&crackTopSnapshot{Stations: stations[:1]})
	model.normalizeWorkerSelection()
	pane := ansi.Strip(model.renderWorkersPane(90, 10))
	if !strings.Contains(pane, "status unavailable") || strings.Contains(pane, "idle") {
		t.Fatalf("status-less connected station was mislabeled:\n%s", pane)
	}
}

func TestCrackTopRateAndCountersSaturate(t *testing.T) {
	maxRate := fmt.Sprintf("%d", uint64(math.MaxUint64))
	parsed, err := parseCrackStatusJSON(crackTopTestStatus("1", "2",
		fmt.Sprintf(`{"device_id":1,"speed":%s}`, maxRate),
		`{"device_id":2,"speed":1}`,
	))
	if err != nil {
		t.Fatalf("parse max-rate status: %v", err)
	}
	if rate, known := crackTopStatusRate(parsed); !known || rate != math.MaxUint64 {
		t.Fatalf("saturated device rate = %d known=%v, want MaxUint64", rate, known)
	}

	now := time.Unix(1_700_000_000, 0)
	job := &clientpb.CrackJob{
		ID:          crackTopTestJobID,
		Status:      clientpb.CrackJobStatus_IN_PROGRESS,
		ResultCount: math.MaxUint64,
		Tasks: []*clientpb.CrackTask{
			crackTopTestTask("max", "maxrate", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 2, now.Add(time.Minute).Unix(), crackTopTestStatus("1", "2", fmt.Sprintf(`{"device_id":1,"speed":%s}`, maxRate))),
			crackTopTestTask("extra", "extra", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 2, now.Add(time.Minute).Unix(), crackTopTestStatus("1", "2", `{"device_id":1,"speed":10}`)),
		},
	}
	completed := &clientpb.CrackJob{ID: "bbbbbbbb-2222-4222-8222-222222222222", Status: clientpb.CrackJobStatus_COMPLETED, ResultCount: 1}
	dashboard := buildCrackTopDashboard(&crackTopSnapshot{
		Jobs: []*clientpb.CrackJob{job, completed},
		Stations: []*clientpb.Crackstation{
			crackTopTestStation("maxrate", "Max", job.ID, clientpb.States_CRACKING),
			crackTopTestStation("extra", "Extra", job.ID, clientpb.States_CRACKING),
		},
		RefreshedAt: now,
	})
	if dashboard.ClusterRate != math.MaxUint64 || dashboard.Jobs[0].Rate != math.MaxUint64 {
		t.Fatalf("saturated rates = cluster:%d job:%d", dashboard.ClusterRate, dashboard.Jobs[0].Rate)
	}
	if dashboard.Recovered != math.MaxUint64 {
		t.Fatalf("saturated recovered count = %d, want MaxUint64", dashboard.Recovered)
	}
}

func crackTopOrderingTestSnapshot(reverse bool) *crackTopSnapshot {
	now := time.Unix(1_700_000_000, 0)
	activeID := "dddddddd-4444-4444-8444-444444444444"
	jobs := []*clientpb.CrackJob{
		{ID: "bbbbbbbb-2222-4222-8222-222222222222", Status: clientpb.CrackJobStatus_COMPLETED, UpdatedAt: 200},
		{ID: "cccccccc-3333-4333-8333-333333333333", Status: clientpb.CrackJobStatus_FAILED, UpdatedAt: 300},
		{ID: "aaaaaaaa-1111-4111-8111-111111111111", Status: clientpb.CrackJobStatus_COMPLETED, UpdatedAt: 200},
		{ID: "eeeeeeee-5555-4555-8555-555555555555", Status: clientpb.CrackJobStatus_CANCELLED, UpdatedAt: 50},
		{
			ID:        activeID,
			Status:    clientpb.CrackJobStatus_IN_PROGRESS,
			UpdatedAt: 100,
			Tasks: []*clientpb.CrackTask{
				crackTopTestTask("fast", "fast", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), crackTopTestStatus("1", "2", `{"device_id":1,"speed":300}`)),
				crackTopTestTask("slow", "slow", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), crackTopTestStatus("1", "2", `{"device_id":1,"speed":100}`)),
				crackTopTestTask("offline", "offline", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), crackTopTestStatus("1", "2", `{"device_id":1,"speed":999}`)),
			},
		},
	}
	stations := []*clientpb.Crackstation{
		crackTopTestStation("unknown", "Beta", "", clientpb.States_IDLE),
		crackTopTestStation("slow", "Alpha", activeID, clientpb.States_CRACKING),
		crackTopTestStation("fast", "Zeta", activeID, clientpb.States_CRACKING),
	}
	if reverse {
		for left, right := 0, len(jobs)-1; left < right; left, right = left+1, right-1 {
			jobs[left], jobs[right] = jobs[right], jobs[left]
		}
		for left, right := 0, len(stations)-1; left < right; left, right = left+1, right-1 {
			stations[left], stations[right] = stations[right], stations[left]
		}
	}
	return &crackTopSnapshot{Jobs: jobs, Stations: stations, RefreshedAt: now}
}

func TestCrackTopDashboardOrderingSelectionAndFilteringAreDeterministic(t *testing.T) {
	first := buildCrackTopDashboard(crackTopOrderingTestSnapshot(false))
	second := buildCrackTopDashboard(crackTopOrderingTestSnapshot(true))
	wantJobs := []string{
		"cccccccc-3333-4333-8333-333333333333",
		"aaaaaaaa-1111-4111-8111-111111111111",
		"bbbbbbbb-2222-4222-8222-222222222222",
		"dddddddd-4444-4444-8444-444444444444",
		"eeeeeeee-5555-4555-8555-555555555555",
	}
	wantWorkers := []string{"fast", "slow", "unknown", "offline"}
	if got := crackTopJobRowIDs(first.Jobs); !reflect.DeepEqual(got, wantJobs) {
		t.Fatalf("job order = %#v, want %#v", got, wantJobs)
	}
	if got := crackTopWorkerRowIDs(first.Workers); !reflect.DeepEqual(got, wantWorkers) {
		t.Fatalf("worker order = %#v, want %#v", got, wantWorkers)
	}
	if !reflect.DeepEqual(crackTopJobRowIDs(second.Jobs), wantJobs) || !reflect.DeepEqual(crackTopWorkerRowIDs(second.Workers), wantWorkers) {
		t.Fatalf("reversed inputs changed ordering: jobs=%#v workers=%#v", crackTopJobRowIDs(second.Jobs), crackTopWorkerRowIDs(second.Workers))
	}

	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.dashboard = first
	model.normalizeJobSelection()
	if model.selectedJobID != "dddddddd-4444-4444-8444-444444444444" {
		t.Fatalf("default selection = %q, want active job", model.selectedJobID)
	}
	model.selectedJobID = "bbbbbbbb-2222-4222-8222-222222222222"
	model.dashboard.Jobs = append([]crackTopJobRow(nil), model.dashboard.Jobs...)
	for left, right := 0, len(model.dashboard.Jobs)-1; left < right; left, right = left+1, right-1 {
		model.dashboard.Jobs[left], model.dashboard.Jobs[right] = model.dashboard.Jobs[right], model.dashboard.Jobs[left]
	}
	model.normalizeJobSelection()
	if model.selectedJobID != "bbbbbbbb-2222-4222-8222-222222222222" || model.jobCursor != 2 {
		t.Fatalf("selection after reorder = %q at %d", model.selectedJobID, model.jobCursor)
	}

	model.filter = crackTopFilterActive
	if got := crackTopJobRowIDs(model.filteredJobs()); !reflect.DeepEqual(got, []string{"dddddddd-4444-4444-8444-444444444444"}) {
		t.Fatalf("active filter = %#v", got)
	}
	model.normalizeJobSelection()
	if model.selectedJobID != "dddddddd-4444-4444-8444-444444444444" {
		t.Fatalf("active-filter selection = %q", model.selectedJobID)
	}
	model.filter = crackTopFilterCompleted
	if got := crackTopJobRowIDs(model.filteredJobs()); !reflect.DeepEqual(got, []string{"bbbbbbbb-2222-4222-8222-222222222222", "aaaaaaaa-1111-4111-8111-111111111111"}) {
		t.Fatalf("completed filter = %#v", got)
	}
	model.filter = crackTopFilterFailed
	if got := crackTopJobRowIDs(model.filteredJobs()); !reflect.DeepEqual(got, []string{"eeeeeeee-5555-4555-8555-555555555555", "cccccccc-3333-4333-8333-333333333333"}) {
		t.Fatalf("failed filter = %#v", got)
	}
}

func TestCrackTopViewIsFullScreenBoundedAndShowsLiveTelemetry(t *testing.T) {
	snapshot := crackTopLiveTestSnapshot()
	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.snapshot = snapshot
	model.dashboard = buildCrackTopDashboard(snapshot)
	model.refreshing = false
	model.normalizeJobSelection()

	for _, size := range []struct {
		name          string
		width, height int
	}{
		{name: "wide", width: 132, height: 36},
		{name: "narrow", width: 90, height: 28},
		{name: "minimum", width: crackTopMinWidth, height: crackTopMinHeight},
		{name: "tiny", width: 31, height: 8},
		{name: "one cell", width: 1, height: 1},
	} {
		t.Run(size.name, func(t *testing.T) {
			model.width = size.width
			model.height = size.height
			view := model.View()
			if !view.AltScreen {
				t.Fatal("crack top view did not request the alternate screen")
			}
			if got := lipgloss.Width(view.Content); got > size.width {
				t.Fatalf("view width = %d, want <= %d", got, size.width)
			}
			if got := lipgloss.Height(view.Content); got > size.height {
				t.Fatalf("view height = %d, want <= %d", got, size.height)
			}
			for lineNumber, line := range strings.Split(view.Content, "\n") {
				if got := ansi.StringWidth(line); got > size.width {
					t.Fatalf("line %d width = %d, want <= %d: %q", lineNumber+1, got, size.width, ansi.Strip(line))
				}
			}
		})
	}

	model.width = 132
	model.height = 36
	plain := ansi.Strip(model.View().Content)
	for _, expected := range []string{
		"CRACK TOP",
		"ACTIVE 1   JOBS 1 loaded",
		"WORKERS 2/2 online",
		"CLUSTER 2.50 MH/s",
		"RECOVERED 7",
		"GLOBAL",
		"31.2%",
		"all active + recent history",
		"aaaaaaaa",
		"IN_PROGRESS",
		"CRACKSTATIONS",
		"Alpha Rig [worker01]",
		"Beta Rig [worker02]",
		"2.00 kH/s",
		"2.50 MH/s",
		"2 devices",
	} {
		if !strings.Contains(plain, expected) {
			t.Errorf("wide view does not contain %q:\n%s", expected, plain)
		}
	}
}

func TestCrackTopMalformedStatusIsNonFatalAndVisible(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	job := &clientpb.CrackJob{
		ID:     crackTopTestJobID,
		Status: clientpb.CrackJobStatus_IN_PROGRESS,
		Tasks: []*clientpb.CrackTask{
			crackTopTestTask("broken", "broken", clientpb.CrackTaskKind_CRACK_TASK_CRACK, clientpb.CrackTaskState_CRACK_TASK_RUNNING, 100, now.Add(time.Minute).Unix(), []byte(`{"unterminated":`)),
		},
	}
	snapshot := &crackTopSnapshot{
		Jobs:        []*clientpb.CrackJob{job},
		Stations:    []*clientpb.Crackstation{crackTopTestStation("broken", "Broken Worker", job.ID, clientpb.States_CRACKING)},
		RefreshedAt: now,
	}
	dashboard := buildCrackTopDashboard(snapshot)
	if len(dashboard.Jobs) != 1 || dashboard.Jobs[0].TelemetryErrors != 1 {
		t.Fatalf("malformed telemetry dashboard = %#v", dashboard)
	}
	if dashboard.Jobs[0].RateKnown || dashboard.ClusterRateKnown {
		t.Fatalf("malformed status contributed a live rate: %#v", dashboard)
	}
	if !dashboard.Jobs[0].ProgressKnown || dashboard.Jobs[0].ProgressComplete || dashboard.Jobs[0].Progress != 0 || dashboard.ProgressKnown {
		t.Fatalf("malformed running status did not stay a zero lower bound: %#v", dashboard)
	}
	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.snapshot = snapshot
	model.dashboard = dashboard
	model.refreshing = false
	model.width = 132
	model.height = 30
	model.normalizeJobSelection()
	plain := ansi.Strip(model.View().Content)
	if !strings.Contains(plain, "telemetry !") || !strings.Contains(plain, "Broken Worker") {
		t.Fatalf("malformed telemetry was not represented safely:\n%s", plain)
	}
	if strings.Contains(plain, "NaN") || strings.Contains(plain, "Inf") {
		t.Fatalf("malformed telemetry rendered an invalid number:\n%s", plain)
	}
}

func TestCrackTopJobPaneShowsZeroLiveWorkers(t *testing.T) {
	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.dashboard = crackTopDashboard{Jobs: []crackTopJobRow{{
		ID:      crackTopTestJobID,
		Status:  clientpb.CrackJobStatus_IN_PROGRESS,
		Workers: 2,
	}}}
	model.normalizeJobSelection()

	pane := ansi.Strip(model.renderJobsPane(90, 10))
	if !strings.Contains(pane, "0/2 live") {
		t.Fatalf("job pane did not distinguish assigned from live workers:\n%s", pane)
	}
}

func TestCrackTopAcceptedTimersReleaseTheirCancelContexts(t *testing.T) {
	model := newCrackTopModel(context.Background(), nil, nil, time.Second)

	pollCanceled := false
	model.refreshing = false
	model.pollGeneration = 1
	model.pollCancel = func() { pollCanceled = true }
	_, _ = model.Update(crackTopPollMsg{generation: 1})
	if !pollCanceled {
		t.Fatal("accepted periodic timer did not release its context")
	}

	eventCanceled := false
	model.refreshing = false
	model.eventRefreshPending = true
	model.eventGeneration = 2
	model.eventCancel = func() { eventCanceled = true }
	_, _ = model.Update(crackTopEventRefreshMsg{generation: 2})
	if !eventCanceled {
		t.Fatal("accepted event debounce timer did not release its context")
	}

	toastCanceled := false
	model.toast = "done"
	model.toastGeneration = 3
	model.toastCancel = func() { toastCanceled = true }
	_, _ = model.Update(crackTopToastExpiredMsg{generation: 3})
	if !toastCanceled || model.toast != "" {
		t.Fatalf("accepted toast timer cleanup = canceled:%v toast:%q", toastCanceled, model.toast)
	}
}

func TestCrackTopFooterPrioritizesToastWithoutHidingDegradedState(t *testing.T) {
	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.snapshot = crackTopLiveTestSnapshot()
	model.lastError = "refresh failed"
	model.toast = "credential recovered"
	model.toastLevel = "success"

	footer := ansi.Strip(model.renderFooter(100))
	if !strings.Contains(footer, "DEGRADED") || !strings.Contains(footer, "credential recovered") {
		t.Fatalf("active toast footer = %q", footer)
	}
	if strings.Contains(footer, "refresh failed") {
		t.Fatalf("refresh error displaced active toast: %q", footer)
	}

	model.toast = ""
	footer = ansi.Strip(model.renderFooter(100))
	if !strings.Contains(footer, "refresh failed") || !strings.Contains(footer, "showing last snapshot") {
		t.Fatalf("degraded footer after toast = %q", footer)
	}
}

func crackTopSnapshotMessageFromRefresh(t *testing.T, command tea.Cmd) crackTopSnapshotMsg {
	t.Helper()
	if command == nil {
		t.Fatal("refresh command is nil")
	}
	message := command()
	if snapshot, ok := message.(crackTopSnapshotMsg); ok {
		return snapshot
	}
	batch, ok := message.(tea.BatchMsg)
	if !ok {
		t.Fatalf("refresh command returned %T, want crackTopSnapshotMsg or tea.BatchMsg", message)
	}
	for _, subcommand := range batch {
		if subcommand == nil {
			continue
		}
		if snapshot, ok := subcommand().(crackTopSnapshotMsg); ok {
			return snapshot
		}
	}
	t.Fatal("refresh batch did not contain crackTopSnapshotMsg")
	return crackTopSnapshotMsg{}
}

func TestCrackTopRefreshCoalescesAndRejectsStalePollGeneration(t *testing.T) {
	loads := 0
	snapshot := crackTopLiveTestSnapshot()
	model := newCrackTopModel(context.Background(), func(context.Context) (*crackTopSnapshot, error) {
		loads++
		return snapshot, nil
	}, nil, time.Second)
	model.refreshing = false
	model.pollGeneration = 7

	_, firstRefresh := model.Update(tea.KeyPressMsg{Text: "r", Code: 'r'})
	if firstRefresh == nil || !model.refreshing || model.pollGeneration != 8 {
		t.Fatalf("first refresh state = refreshing:%v generation:%d command:%v", model.refreshing, model.pollGeneration, firstRefresh != nil)
	}
	_, duplicateRefresh := model.Update(tea.KeyPressMsg{Text: "r", Code: 'r'})
	if duplicateRefresh != nil || !model.refreshQueued {
		t.Fatalf("duplicate refresh command=%v queued=%v, want nil/true", duplicateRefresh != nil, model.refreshQueued)
	}

	firstResult := crackTopSnapshotMessageFromRefresh(t, firstRefresh)
	if loads != 1 {
		t.Fatalf("loads after first command = %d, want 1", loads)
	}
	_, queuedRefresh := model.Update(firstResult)
	if queuedRefresh == nil || !model.refreshing || model.refreshQueued || model.pollGeneration != 9 {
		t.Fatalf("queued refresh state = refreshing:%v queued:%v generation:%d command:%v", model.refreshing, model.refreshQueued, model.pollGeneration, queuedRefresh != nil)
	}
	secondResult := crackTopSnapshotMessageFromRefresh(t, queuedRefresh)
	if loads != 2 {
		t.Fatalf("loads after coalesced command = %d, want 2", loads)
	}
	_, pollCommand := model.Update(secondResult)
	if pollCommand == nil || model.refreshing {
		t.Fatalf("completed refresh state = refreshing:%v poll command:%v", model.refreshing, pollCommand != nil)
	}
	currentGeneration := model.pollGeneration
	_, staleCommand := model.Update(crackTopPollMsg{generation: currentGeneration - 1})
	if staleCommand != nil || model.refreshing {
		t.Fatalf("stale poll started refresh: refreshing=%v command=%v", model.refreshing, staleCommand != nil)
	}
	_, currentCommand := model.Update(crackTopPollMsg{generation: currentGeneration})
	if currentCommand == nil || !model.refreshing || model.pollGeneration == currentGeneration {
		t.Fatalf("current poll state = refreshing:%v generation:%d command:%v", model.refreshing, model.pollGeneration, currentCommand != nil)
	}
}

func TestWaitForCrackTopEventRecognizesRefreshEventsAndClosure(t *testing.T) {
	for _, eventType := range []string{
		consts.CrackTaskStatus,
		consts.CrackJobCreated,
		consts.CrackJobUpdated,
		consts.CredentialCrackedEvent,
		consts.CrackstationConnected,
		consts.CrackstationDisconnected,
		consts.CrackStatusEvent,
	} {
		t.Run(eventType, func(t *testing.T) {
			listener := make(chan *clientpb.Event, 3)
			listener <- nil
			listener <- &clientpb.Event{EventType: "unrelated"}
			listener <- &clientpb.Event{EventType: eventType}
			if message := waitForCrackTopEventCmd(listener)(); message != (crackTopEventMsg{}) {
				t.Fatalf("event %q returned %T, want crackTopEventMsg", eventType, message)
			}
		})
	}

	closed := make(chan *clientpb.Event)
	close(closed)
	if _, ok := waitForCrackTopEventCmd(closed)().(crackTopListenerClosedMsg); !ok {
		t.Fatal("closed event listener did not return crackTopListenerClosedMsg")
	}
	if _, ok := waitForCrackTopEventCmd(nil)().(crackTopListenerClosedMsg); !ok {
		t.Fatal("nil event listener did not return crackTopListenerClosedMsg")
	}
}

func TestCrackTopEventRefreshRearmsListenerAndCoalesces(t *testing.T) {
	listener := make(chan *clientpb.Event, 1)
	model := newCrackTopModel(context.Background(), func(context.Context) (*crackTopSnapshot, error) {
		return crackTopLiveTestSnapshot(), nil
	}, listener, time.Second)
	model.refreshing = false

	_, command := model.Update(crackTopEventMsg{})
	if command == nil || model.refreshing || !model.eventRefreshPending {
		t.Fatalf("event debounce command=%v refreshing=%v pending=%v", command != nil, model.refreshing, model.eventRefreshPending)
	}
	firstGeneration := model.eventGeneration
	message := command()
	batch, ok := message.(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("event command = %T len=%d, want two-command batch", message, len(batch))
	}

	_, coalesced := model.Update(crackTopEventMsg{})
	secondGeneration := model.eventGeneration
	if coalesced == nil || !model.eventRefreshPending || secondGeneration <= firstGeneration {
		t.Fatalf("coalesced debounce command=%v pending=%v generation=%d, want newer than %d", coalesced != nil, model.eventRefreshPending, secondGeneration, firstGeneration)
	}
	if staleTimerMessage := batch[1](); staleTimerMessage != nil {
		t.Fatalf("superseded debounce timer returned %T, want cancellation", staleTimerMessage)
	}
	coalescedMessage := coalesced()
	coalescedBatch, ok := coalescedMessage.(tea.BatchMsg)
	if !ok || len(coalescedBatch) != 2 {
		t.Fatalf("coalesced event command = %T len=%d, want listener plus replacement debounce", coalescedMessage, len(coalescedBatch))
	}
	listener <- &clientpb.Event{EventType: consts.CrackJobUpdated}
	if listenerMessage := coalescedBatch[0](); listenerMessage != (crackTopEventMsg{}) {
		t.Fatalf("rearmed listener returned %T, want crackTopEventMsg", listenerMessage)
	}

	_, staleRefresh := model.Update(crackTopEventRefreshMsg{generation: firstGeneration})
	if staleRefresh != nil || model.refreshing || !model.eventRefreshPending {
		t.Fatalf("superseded debounce refreshed: command=%v refreshing=%v pending=%v", staleRefresh != nil, model.refreshing, model.eventRefreshPending)
	}
	_, refresh := model.Update(crackTopEventRefreshMsg{generation: secondGeneration})
	if refresh == nil || !model.refreshing || model.eventRefreshPending {
		t.Fatalf("debounced refresh command=%v refreshing=%v pending=%v", refresh != nil, model.refreshing, model.eventRefreshPending)
	}

	_, duringRefresh := model.Update(crackTopEventMsg{})
	if duringRefresh == nil || !model.eventRefreshPending {
		t.Fatalf("event during refresh command=%v pending=%v", duringRefresh != nil, model.eventRefreshPending)
	}
	queuedGeneration := model.eventGeneration
	_, queued := model.Update(crackTopEventRefreshMsg{generation: queuedGeneration})
	if queued != nil || !model.eventRefreshQueued || model.eventRefreshPending {
		t.Fatalf("event refresh coalescing command=%v queued=%v pending=%v", queued != nil, model.eventRefreshQueued, model.eventRefreshPending)
	}
}

func crackTopRunModelCommands(t *testing.T, model *crackTopModel, command tea.Cmd) *crackTopModel {
	t.Helper()
	commands := []tea.Cmd{command}
	for steps := 0; len(commands) > 0; steps++ {
		if steps > 64 {
			t.Fatal("command chain did not settle")
		}
		current := commands[0]
		commands = commands[1:]
		if current == nil {
			continue
		}
		message := current()
		if batch, ok := message.(tea.BatchMsg); ok {
			commands = append(commands, batch...)
			continue
		}
		if message == nil {
			continue
		}
		updated, next := model.Update(message)
		var ok bool
		model, ok = updated.(*crackTopModel)
		if !ok {
			t.Fatalf("updated model = %T, want *crackTopModel", updated)
		}
		commands = append(commands, next)
	}
	return model
}

func TestCrackTopHuhFilterAppliesAndEscapeCancels(t *testing.T) {
	model := newCrackTopModel(context.Background(), nil, nil, time.Second)
	model.dashboard = crackTopDashboard{Jobs: []crackTopJobRow{
		{ID: "active", Status: clientpb.CrackJobStatus_IN_PROGRESS},
		{ID: "done", Status: clientpb.CrackJobStatus_COMPLETED},
	}}
	model.normalizeJobSelection()

	_, initCommand := model.Update(tea.KeyPressMsg{Text: "f", Code: 'f'})
	if initCommand == nil || model.filterForm == nil {
		t.Fatalf("filter open command=%v form=%v", initCommand != nil, model.filterForm != nil)
	}
	model.width = 90
	model.height = 28
	model.resizeFilterForm()
	modalView := model.View().Content
	modal := ansi.Strip(modalView)
	if !strings.Contains(modal, "Filter crack jobs") || !strings.Contains(modal, "Job scope") || !strings.Contains(modal, "All loaded jobs") {
		t.Fatalf("filter modal is missing Huh content:\n%s", modal)
	}
	for _, background := range []string{"CRACK TOP", "JOBS 2 loaded", "tab pane"} {
		if !strings.Contains(modal, background) {
			t.Fatalf("filter modal replaced background content %q:\n%s", background, modal)
		}
	}
	for _, size := range []struct {
		name          string
		width, height int
	}{
		{name: "normal", width: 90, height: 28},
		{name: "minimum", width: crackTopMinWidth, height: crackTopMinHeight},
		{name: "wide", width: 132, height: 36},
	} {
		t.Run("modal bounds "+size.name, func(t *testing.T) {
			model.width = size.width
			model.height = size.height
			model.resizeFilterForm()
			view := model.View().Content
			if got := lipgloss.Width(view); got > size.width {
				t.Fatalf("filter modal width = %d, want <= %d", got, size.width)
			}
			if got := lipgloss.Height(view); got > size.height {
				t.Fatalf("filter modal height = %d, want <= %d", got, size.height)
			}
			for lineNumber, line := range strings.Split(view, "\n") {
				if got := ansi.StringWidth(line); got > size.width {
					t.Fatalf("filter modal line %d width = %d, want <= %d", lineNumber+1, got, size.width)
				}
			}
		})
	}
	model.width = 90
	model.height = 28
	model.resizeFilterForm()

	_, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if model.filterDraft != crackTopFilterActive {
		t.Fatalf("filter draft after down = %q, want active", model.filterDraft)
	}
	_, submitCommand := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = crackTopRunModelCommands(t, model, submitCommand)
	if model.filterForm != nil || model.filter != crackTopFilterActive {
		t.Fatalf("applied filter = %q form-open=%v, want active/false", model.filter, model.filterForm != nil)
	}
	if jobs := model.filteredJobs(); len(jobs) != 1 || jobs[0].ID != "active" || model.selectedJobID != "active" {
		t.Fatalf("filtered jobs/selection = %#v / %q", jobs, model.selectedJobID)
	}

	_, _ = model.Update(tea.KeyPressMsg{Text: "f", Code: 'f'})
	if model.filterForm == nil {
		t.Fatal("filter did not reopen")
	}
	_, cancelCommand := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cancelCommand != nil || model.filterForm != nil || model.filter != crackTopFilterActive {
		t.Fatalf("escape cancel = command:%v form:%v filter:%q", cancelCommand != nil, model.filterForm != nil, model.filter)
	}
	_, quitCommand := model.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if quitCommand == nil {
		t.Fatal("ctrl+c did not return a quit command")
	}
	if _, ok := quitCommand().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c command returned %T, want tea.QuitMsg", quitCommand())
	}
}
