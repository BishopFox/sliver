package top

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/x/ansi"
)

//nolint:gocyclo // This test validates all history fields across success, gap, failure, and recovery snapshots.
func TestCrackTopAcceptedSnapshotsAppendRealOverviewHistory(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	first := crackTopLiveTestSnapshot()

	_, _ = model.Update(crackTopSnapshotMsg{snapshot: first})
	if len(model.history) != 1 {
		t.Fatalf("history length after first accepted snapshot = %d, want 1", len(model.history))
	}
	if sample := model.history[0]; !sample.At.Equal(first.RefreshedAt) ||
		!sample.ClusterRateKnown || sample.ClusterRate != 2_502_000 ||
		!sample.ProgressKnown || sample.Progress != 0.3125 ||
		!sample.UtilizationKnown || sample.Utilization != 98.5 ||
		!sample.TemperatureKnown || sample.Temperature != 70 {
		t.Fatalf("first history sample does not reflect live fixture telemetry: %#v", sample)
	}

	gapTime := first.RefreshedAt.Add(time.Second)
	unknownGap := &crackTopSnapshot{
		Stations: []*clientpb.Crackstation{{
			HostUUID: "worker-without-status",
			Name:     "Awaiting Status",
		}},
		RefreshedAt: gapTime,
	}
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: unknownGap})
	if len(model.history) != 2 {
		t.Fatalf("history length after second accepted snapshot = %d, want 2", len(model.history))
	}
	gap := model.history[1]
	if !gap.At.Equal(gapTime) {
		t.Fatalf("unknown-gap time = %s, want %s", gap.At, gapTime)
	}
	if gap.ClusterRateKnown || gap.ProgressKnown || gap.UtilizationKnown || gap.TemperatureKnown {
		t.Fatalf("unreported telemetry was recorded as known instead of an unknown gap: %#v", gap)
	}

	_, _ = model.Update(crackTopSnapshotMsg{err: errors.New("refresh failed")})
	if len(model.history) != 3 {
		t.Fatalf("failed refresh history length = %d, want an explicit third gap sample", len(model.history))
	}
	failure := model.history[2]
	if failure.ClusterRateKnown || failure.ProgressKnown || failure.UtilizationKnown || failure.TemperatureKnown || failure.CountsKnown {
		t.Fatalf("failed refresh did not produce an unknown chart gap: %#v", failure)
	}

	recovered := crackTopLiveTestSnapshot()
	recovered.RefreshedAt = gapTime.Add(time.Second)
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: recovered})
	if len(model.history) != 4 || !model.history[3].ClusterRateKnown {
		t.Fatalf("recovered refresh did not resume live history after the gap: %#v", model.history)
	}
}

func TestCrackTopWideOverviewRendersRealDataCardsChartsAndJournal(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.width = 150
	model.height = 40
	model.refreshing = false
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: crackTopLiveTestSnapshot()})

	view := model.View()
	if !view.AltScreen {
		t.Fatal("wide crack top overview did not request the alternate screen")
	}
	plain := ansi.Strip(view.Content)
	for _, expected := range []string{
		"CLUSTER / STATE",
		"THROUGHPUT",
		"SAMPLE AVG",
		"DIAGNOSTICS",
		"KEYSPACE / PROGRESS",
		"OBSERVED WORKERS",
		"DEVICE LOAD",
		"rate",
		"progress",
		"worker max util",
		"RECENT JOURNAL",
		"ACTIVE 1",
		"JOBS 1 loaded",
		"2/2 online",
		"2.50 MH/s",
		"RECOVERED 7 loaded",
		"31.2%",
		"98% avg worker max",
		"70°C max",
		"QUEUE 2 run   0 leased   0 waiting",
		"Alpha Rig",
		"Beta Rig",
		"snapshot ready",
	} {
		if !strings.Contains(plain, expected) {
			t.Errorf("wide overview does not contain %q:\n%s", expected, plain)
		}
	}
	if got := ansi.StringWidth(strings.Split(view.Content, "\n")[0]); got > model.width {
		t.Fatalf("wide overview first-line width = %d, want <= %d", got, model.width)
	}
}

func TestCrackTopOverviewDoesNotPresentDefaultsAsLiveBeforeSnapshot(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.width = crackTopOverviewWidth
	model.height = crackTopOverviewHeight
	plain := ansi.Strip(model.View().Content)
	for _, expected := range []string{"WAITING", "first live snapshot", "ACTIVE —", "RECOVERED —", "QUEUE —"} {
		if !strings.Contains(plain, expected) {
			t.Errorf("pre-snapshot overview is missing %q:\n%s", expected, plain)
		}
	}
	for _, unsupported := range []string{"HEALTHY", "OK   LIVE", "ACTIVE 0", "RECOVERED 0", "QUEUE 0"} {
		if strings.Contains(plain, unsupported) {
			t.Errorf("pre-snapshot overview presented %q as observed:\n%s", unsupported, plain)
		}
	}

	_, _ = model.Update(crackTopSnapshotMsg{err: errors.New("rpc unavailable")})
	failed := ansi.Strip(model.View().Content)
	for _, expected := range []string{"NO SNAPSHOT", "refresh failed", "ERROR", "no live snapshot"} {
		if !strings.Contains(failed, expected) {
			t.Errorf("initial-failure overview is missing %q:\n%s", expected, failed)
		}
	}
}

func TestCrackTopCompactViewDoesNotPresentDefaultsAsLiveBeforeSnapshot(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.width = 90
	model.height = 28
	model.refreshing = false

	plain := ansi.Strip(model.View().Content)
	for _, expected := range []string{"waiting for data", "ACTIVE —", "JOBS —", "WORKERS —", "RECOVERED —", "awaiting first snapshot"} {
		if !strings.Contains(plain, expected) {
			t.Errorf("compact pre-snapshot view is missing %q:\n%s", expected, plain)
		}
	}
	for _, unsupported := range []string{"ACTIVE 0", "JOBS 0 loaded", "WORKERS 0/0", "RECOVERED 0", "no active jobs"} {
		if strings.Contains(plain, unsupported) {
			t.Errorf("compact pre-snapshot view presented %q as observed:\n%s", unsupported, plain)
		}
	}

	_, _ = model.Update(crackTopSnapshotMsg{err: errors.New("rpc unavailable")})
	failed := ansi.Strip(model.View().Content)
	for _, expected := range []string{"no snapshot", "ACTIVE —", "current data unavailable"} {
		if !strings.Contains(failed, expected) {
			t.Errorf("compact initial-failure view is missing %q:\n%s", expected, failed)
		}
	}
	for _, unsupported := range []string{"ACTIVE 0", "JOBS 0 loaded", "WORKERS 0/0", "RECOVERED 0", "no active jobs"} {
		if strings.Contains(failed, unsupported) {
			t.Errorf("compact initial-failure view presented %q as observed:\n%s", unsupported, failed)
		}
	}
}

func TestCrackTopOverviewMarksRetainedSnapshotMetricsUnavailableAfterRefreshFailure(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.width = 150
	model.height = 40
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: crackTopLiveTestSnapshot()})
	_, _ = model.Update(crackTopSnapshotMsg{err: errors.New("temporary outage")})

	meters := ansi.Strip(model.renderOverviewMeters(150, crackTopOverviewMetersHeight))
	if count := strings.Count(meters, "current data unavailable"); count != 3 {
		t.Fatalf("failed refresh meters did not expose progress/worker gaps:\n%s", meters)
	}
	if strings.Contains(meters, "no active jobs") || strings.Contains(meters, "no observed workers") {
		t.Fatalf("failed refresh meters converted unknown counts to zero:\n%s", meters)
	}
	throughput := ansi.Strip(model.renderOverviewThroughputCard(55, crackTopOverviewSummaryHeight))
	if !strings.Contains(throughput, "CLUSTER last 2.50 MH/s") || !strings.Contains(throughput, "RECOVERED 7 last snapshot") || !strings.Contains(throughput, "VIEW —") {
		t.Fatalf("retained throughput was not labeled as stale:\n%s", throughput)
	}
	diagnostics := ansi.Strip(model.renderOverviewDiagnosticsCard(55, crackTopOverviewSummaryHeight))
	if !strings.Contains(diagnostics, "QUEUE last 2 run") {
		t.Fatalf("retained queue counts were presented as current:\n%s", diagnostics)
	}
	jobs := ansi.Strip(model.renderJobsPane(90, 8))
	workers := ansi.Strip(model.renderWorkersPane(59, 8))
	if !strings.Contains(jobs, "LAST SNAPSHOT") || !strings.Contains(workers, "LAST SNAPSHOT") {
		t.Fatalf("retained detail panes were not labeled stale:\njobs:\n%s\nworkers:\n%s", jobs, workers)
	}
}

func TestCrackTopOverviewRefreshInFlightIsSilent(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.width = 150
	model.height = 40
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: crackTopLiveTestSnapshot()})
	if command := model.startRefresh(); command == nil || !model.refreshing {
		t.Fatalf("refresh did not start: command=%v refreshing=%v", command != nil, model.refreshing)
	}

	top := ansi.Strip(model.renderOverviewTopBar(model.width))
	diagnostics := ansi.Strip(model.renderOverviewDiagnosticsCard(50, crackTopOverviewSummaryHeight))
	footer := ansi.Strip(model.renderOverviewFooter(model.width))
	header := ansi.Strip(model.renderHeader(model.width))
	for name, rendered := range map[string]string{
		"top bar":     top,
		"diagnostics": diagnostics,
		"footer":      footer,
		"header":      header,
	} {
		lower := strings.ToLower(rendered)
		if strings.Contains(lower, "refreshing") || strings.Contains(lower, "live snapshot in flight") {
			t.Errorf("%s exposed an in-flight refresh:\n%s", name, rendered)
		}
	}
	if !strings.Contains(diagnostics, "OK   LIVE DATA COMPLETE") {
		t.Errorf("silent refresh changed snapshot diagnostics:\n%s", diagnostics)
	}
	if !strings.Contains(footer, "JOBS focus") {
		t.Errorf("silent refresh displaced the normal footer:\n%s", footer)
	}
	if !strings.Contains(header, "updated ") {
		t.Errorf("silent refresh hid the current snapshot timestamp:\n%s", header)
	}
}

func TestCrackTopOverviewLabelsPartialRateHistoryAsLowerBounds(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.width = 150
	model.height = 40
	model.snapshot = &crackTopSnapshot{RefreshedAt: time.Now()}
	model.dashboard = crackTopDashboard{ClusterRate: 2_000, ClusterRateKnown: true, ClusterComplete: false}
	model.history = []crackTopHistorySample{
		{At: time.Now().Add(-time.Second), ClusterRate: 1_000, ClusterRateKnown: true, ClusterComplete: true},
		{At: time.Now(), ClusterRate: 2_000, ClusterRateKnown: true, ClusterComplete: false},
	}

	average, peak, known, complete := crackTopRateStats(model.history)
	if !known || complete || average != 1_500 || peak != 2_000 {
		t.Fatalf("partial rate stats = avg:%d peak:%d known:%v complete:%v", average, peak, known, complete)
	}
	plain := ansi.Strip(model.renderOverviewThroughputCard(55, crackTopOverviewSummaryHeight))
	if !strings.Contains(plain, "CLUSTER ≥ 2.00 kH/s") || !strings.Contains(plain, "AVG ≥ 1.50 kH/s") || !strings.Contains(plain, "PEAK ≥ 2.00 kH/s") {
		t.Fatalf("partial rate card lost lower-bound semantics:\n%s", plain)
	}
	rateChart := ansi.Strip(model.renderOverviewCharts(150, 8))
	if !strings.Contains(rateChart, "rate  ≥ 2.00 kH/s") {
		t.Fatalf("partial rate chart lost current lower-bound semantics:\n%s", rateChart)
	}
}

func TestCrackTopOverviewPausedOnlyWorkloadIsNotIdle(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.snapshot = &crackTopSnapshot{RefreshedAt: time.Now()}
	model.dashboard = crackTopDashboard{Jobs: []crackTopJobRow{{ID: "paused", Status: clientpb.CrackJobStatus_PAUSED}}}
	plain := ansi.Strip(model.renderOverviewStateCard(50, crackTopOverviewSummaryHeight))
	if !strings.Contains(plain, "PAUSED") || strings.Contains(plain, "IDLE") {
		t.Fatalf("paused-only workload state is misleading:\n%s", plain)
	}
}

func TestCrackTopPlotPreservesUnknownGapAndBounds(t *testing.T) {
	const (
		width  = 4
		height = 4
	)
	plot := crackTopRenderPlot([]crackTopChartPoint{
		{value: 100, known: true},
		{value: 100, known: true},
		{known: false},
		{value: 100, known: true},
	}, width, height, 100)

	lines := strings.Split(plot, "\n")
	if len(lines) != height {
		t.Fatalf("plot height = %d, want %d:\n%s", len(lines), height, plot)
	}
	for index, line := range lines {
		if got := ansi.StringWidth(line); got > width {
			t.Fatalf("plot line %d width = %d, want <= %d: %q", index+1, got, width, line)
		}
	}
	top := []rune(lines[0])
	if len(top) != width {
		t.Fatalf("plot top row width = %d runes, want %d: %q", len(top), width, lines[0])
	}
	if top[0] != '╶' || top[1] != '╴' || top[2] != ' ' || top[3] != '●' {
		t.Fatalf("plot did not preserve the unknown sample as a visible gap: %q", lines[0])
	}
}

func TestCrackTopPlotConnectsVerticalTransitions(t *testing.T) {
	testCases := []struct {
		name   string
		points []crackTopChartPoint
		height int
		want   string
	}{
		{
			name: "rising",
			points: []crackTopChartPoint{
				{value: 0, known: true},
				{value: 50, known: true},
				{value: 100, known: true},
			},
			height: 3,
			want:   "  ╷\n·┌┘\n╶┘·",
		},
		{
			name: "falling",
			points: []crackTopChartPoint{
				{value: 100, known: true},
				{value: 50, known: true},
				{value: 0, known: true},
			},
			height: 3,
			want:   "╶┐ \n·└┐\n··╵",
		},
		{
			name: "flat",
			points: []crackTopChartPoint{
				{value: 50, known: true},
				{value: 50, known: true},
				{value: 50, known: true},
			},
			height: 3,
			want:   "   \n╶─╴\n···",
		},
		{
			name: "full range rise",
			points: []crackTopChartPoint{
				{value: 0, known: true},
				{value: 100, known: true},
			},
			height: 4,
			want:   " ╷\n │\n·│\n╶┘",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := crackTopRenderPlot(testCase.points, len(testCase.points), testCase.height, 100)
			if got != testCase.want {
				t.Fatalf("plot =\n%q\nwant\n%q", got, testCase.want)
			}
		})
	}
}

func TestCrackTopChartStatsDoesNotPresentStaleLatestAsCurrent(t *testing.T) {
	latest, average, peak, latestKnown, historyKnown, latestLowerBound, historyLowerBound := crackTopChartStats([]crackTopChartPoint{
		{value: 25, known: true},
		{value: 75, known: true},
		{known: false},
	})
	if latestKnown || !historyKnown {
		t.Fatalf("latest/history known = %v/%v, want false/true", latestKnown, historyKnown)
	}
	if latestLowerBound || historyLowerBound {
		t.Fatalf("unexpected lower-bound flags = latest:%v history:%v", latestLowerBound, historyLowerBound)
	}
	if latest != 0 || average != 50 || peak != 75 {
		t.Fatalf("latest/average/peak = %v/%v/%v, want 0/50/75", latest, average, peak)
	}
}

func TestCrackTopChartStatsUseOnlyVisiblePlotWindow(t *testing.T) {
	points := []crackTopChartPoint{
		{value: 1_000, known: true},
		{value: 10, known: true},
		{value: 20, known: true},
		{value: 30, known: true},
	}
	visible := crackTopVisibleChartPoints(points, 3)
	latest, average, peak, latestKnown, historyKnown, _, _ := crackTopChartStats(visible)
	if len(visible) != 3 || !latestKnown || !historyKnown || latest != 30 || average != 20 || peak != 30 {
		t.Fatalf("visible chart stats = len:%d latest:%v avg:%v peak:%v latest-known:%v history-known:%v", len(visible), latest, average, peak, latestKnown, historyKnown)
	}
}

func TestCrackTopRecoveredDeltaUsesOnlyObservedCountEndpoints(t *testing.T) {
	history := []crackTopHistorySample{
		{Recovered: 0, CountsKnown: false},
		{Recovered: 10, CountsKnown: true},
		{Recovered: 0, CountsKnown: false},
		{Recovered: 13, CountsKnown: true},
	}
	if delta, known := crackTopRecoveredDelta(history); !known || delta != 3 {
		t.Fatalf("recovered delta across boundary gaps = %d known:%v, want 3 known", delta, known)
	}
	if delta, known := crackTopRecoveredDelta([]crackTopHistorySample{{Recovered: 13}, {Recovered: 14}}); known || delta != 0 {
		t.Fatalf("unobserved recovered endpoints = %d known:%v, want unknown", delta, known)
	}
	if delta, known := crackTopRecoveredDelta([]crackTopHistorySample{{Recovered: 13, CountsKnown: true}}); known || delta != 0 {
		t.Fatalf("single recovered endpoint = %d known:%v, want unknown", delta, known)
	}
	if delta, known := crackTopRecoveredDelta([]crackTopHistorySample{{Recovered: 13, CountsKnown: true}, {Recovered: 12, CountsKnown: true}}); known || delta != 0 {
		t.Fatalf("decreasing recovered counter = %d known:%v, want unknown", delta, known)
	}
}

func TestCrackTopChartSpanDisclosesSampleIndexedHistory(t *testing.T) {
	points := []crackTopChartPoint{
		{at: time.Unix(1_700_000_000, 0)},
		{at: time.Unix(1_700_000_001, 0)},
		{at: time.Unix(1_700_000_061, 0)},
	}
	if got := crackTopChartSpan(points); got != "1m1s/3 samples" {
		t.Fatalf("chart span = %q, want elapsed time and sample count", got)
	}
}

func TestCrackTopPlotDistinguishesUnknownFromObservedZero(t *testing.T) {
	plot := crackTopRenderPlot([]crackTopChartPoint{
		{value: 0, known: true},
		{known: false},
		{value: 0, known: true},
	}, 3, 3, 100)
	bottom := strings.Split(plot, "\n")[2]
	if bottom != "●·●" {
		t.Fatalf("zero/unknown/zero baseline = %q, want observed points around a grid gap", bottom)
	}
}

func TestCrackTopAuthoritativeJobActionIsJournaled(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.runningJobAction = crackTopPendingJobAction{action: crackTopJobActionPause, jobID: crackTopTestJobID}
	_, _ = model.Update(crackTopJobActionCompletedMsg{
		action:    crackTopJobActionPause,
		jobID:     crackTopTestJobID,
		status:    clientpb.CrackJobStatus_PAUSED,
		updatedAt: 1_700_000_100,
		hasStatus: true,
	})
	if len(model.journal) != 1 || model.journal[0].label != "JOB" || model.journal[0].message != "aaaaaaaa PAUSED" {
		t.Fatalf("authoritative action journal = %#v", model.journal)
	}
}

//nolint:gocyclo // This test keeps the complete action-to-authoritative-snapshot freshness transition together.
func TestCrackTopActionMarksTaskTelemetryStaleUntilAuthoritativeSnapshot(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.width = 150
	model.height = 40
	initial := crackTopLiveTestSnapshot()
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: initial})
	actionUpdatedAt := initial.RefreshedAt.Add(time.Second).Unix()

	model.runningJobAction = crackTopPendingJobAction{action: crackTopJobActionPause, jobID: crackTopTestJobID}
	_, _ = model.Update(crackTopJobActionCompletedMsg{
		action:    crackTopJobActionPause,
		jobID:     crackTopTestJobID,
		status:    clientpb.CrackJobStatus_PAUSED,
		updatedAt: actionUpdatedAt,
		hasStatus: true,
	})

	if !model.authoritativeSnapshotPending() {
		t.Fatal("successful lifecycle action did not mark task telemetry as awaiting an authoritative snapshot")
	}
	latest := model.history[len(model.history)-1]
	if latest.ClusterRateKnown || latest.ProgressKnown || latest.UtilizationKnown || latest.CountsKnown {
		t.Fatalf("post-action history sample presented stale task telemetry as current: %#v", latest)
	}
	plain := ansi.Strip(strings.Join([]string{
		model.renderOverviewStateCard(50, crackTopOverviewSummaryHeight),
		model.renderOverviewThroughputCard(50, crackTopOverviewSummaryHeight),
		model.renderOverviewDiagnosticsCard(50, crackTopOverviewSummaryHeight),
		model.renderOverviewMeters(150, crackTopOverviewMetersHeight),
		model.renderJobsPane(90, 8),
		model.renderWorkersPane(59, 8),
	}, "\n"))
	for _, expected := range []string{"SYNCING", "TASK DATA LAST", "RECOVERED 7 last snapshot", "task data pending", "QUEUE last 2 run", "last snapshot • syncing", "TASK DATA LAST/SYNCING", "LAST SNAPSHOT/SYNCING"} {
		if !strings.Contains(plain, expected) {
			t.Errorf("post-action overview is missing %q:\n%s", expected, plain)
		}
	}
	if strings.Contains(strings.ToLower(plain), "refreshing") {
		t.Errorf("post-action overview exposed refresh progress:\n%s", plain)
	}

	stale := crackTopLiveTestSnapshot()
	stale.RefreshedAt = stale.RefreshedAt.Add(500 * time.Millisecond)
	stale.Jobs[0].UpdatedAt = actionUpdatedAt
	stale.Stations = stale.Stations[:1]
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: stale})
	if !model.authoritativeSnapshotPending() {
		t.Fatal("equal-version pre-action snapshot prematurely cleared lifecycle synchronization state")
	}
	staleSample := model.history[len(model.history)-1]
	if staleSample.ClusterRateKnown || staleSample.ProgressKnown || staleSample.UtilizationKnown || staleSample.CountsKnown {
		t.Fatalf("client-overridden pre-action snapshot was recorded as authoritative telemetry: %#v", staleSample)
	}
	foundUnrelatedTransition := false
	for _, entry := range model.journal {
		if entry.label == "WORKER" && entry.message == "Beta Rig disconnected" {
			foundUnrelatedTransition = true
			break
		}
	}
	if !foundUnrelatedTransition {
		t.Fatalf("pending-action snapshot discarded an unrelated worker transition: %#v", model.journal)
	}

	authoritative := crackTopLiveTestSnapshot()
	authoritative.RefreshedAt = authoritative.RefreshedAt.Add(2 * time.Second)
	authoritative.Jobs[0].Status = clientpb.CrackJobStatus_PAUSED
	authoritative.Jobs[0].UpdatedAt = actionUpdatedAt
	for _, task := range authoritative.Jobs[0].Tasks {
		task.State = clientpb.CrackTaskState_CRACK_TASK_QUEUED
	}
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: authoritative})
	if model.authoritativeSnapshotPending() {
		t.Fatal("matching authoritative snapshot did not clear lifecycle synchronization state")
	}
	diagnostics := ansi.Strip(model.renderOverviewDiagnosticsCard(50, crackTopOverviewSummaryHeight))
	if strings.Contains(diagnostics, "QUEUE last") || strings.Contains(diagnostics, "SYNC   action") {
		t.Fatalf("authoritative snapshot remained labeled as stale:\n%s", diagnostics)
	}
}

func TestCrackTopOverviewSelectionMarkerFollowsFocusedPane(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	model.dashboard = buildCrackTopDashboard(crackTopLiveTestSnapshot())
	model.normalizeJobSelection()
	model.normalizeWorkerSelection()

	jobs := ansi.Strip(model.renderJobsPane(75, 8))
	workers := ansi.Strip(model.renderWorkersPane(55, 8))
	if !strings.Contains(jobs, "› aaaaaaaa") || strings.Contains(workers, "› ") {
		t.Fatalf("job focus markers are ambiguous:\njobs:\n%s\nworkers:\n%s", jobs, workers)
	}

	_, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	jobs = ansi.Strip(model.renderJobsPane(75, 8))
	workers = ansi.Strip(model.renderWorkersPane(55, 8))
	if strings.Contains(jobs, "› ") || !strings.Contains(workers, "› ") {
		t.Fatalf("worker focus markers are ambiguous:\njobs:\n%s\nworkers:\n%s", jobs, workers)
	}
}

func TestCrackTopJournalRecordsRecoveredJobAndWorkerTransitions(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	first := crackTopLiveTestSnapshot()
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: first})

	second := crackTopLiveTestSnapshot()
	second.RefreshedAt = first.RefreshedAt.Add(time.Second)
	second.Jobs[0].Status = clientpb.CrackJobStatus_COMPLETED
	second.Jobs[0].ResultCount = 10
	second.Stations = second.Stations[:1]
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: second})

	wants := map[string]string{
		"JOB":       "aaaaaaaa COMPLETED",
		"WORKER":    "Beta Rig disconnected",
		"RECOVERED": "+3 across loaded jobs",
	}
	for label, message := range wants {
		found := false
		for _, entry := range model.journal {
			if entry.label == label && entry.message == message {
				found = true
				if !entry.at.Equal(second.RefreshedAt) {
					t.Errorf("%s journal time = %s, want %s", label, entry.at, second.RefreshedAt)
				}
				break
			}
		}
		if !found {
			t.Errorf("journal missing %s transition %q: %#v", label, message, model.journal)
		}
	}
}

func TestCrackTopJournalRecordsActiveJobRemovedByAnotherOperator(t *testing.T) {
	model := newCrackTopModel(t.Context(), nil, nil, time.Second)
	first := crackTopLiveTestSnapshot()
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: first})

	second := &crackTopSnapshot{RefreshedAt: first.RefreshedAt.Add(time.Second)}
	_, _ = model.Update(crackTopSnapshotMsg{snapshot: second})

	for _, entry := range model.journal {
		if entry.label == "JOB" && entry.message == "aaaaaaaa removed from live set" {
			return
		}
	}
	t.Fatalf("journal did not record externally removed active job: %#v", model.journal)
}
